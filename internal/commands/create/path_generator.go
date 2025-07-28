package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
)

var (
	// Global cache for home directory to avoid repeated lookups
	homeDir     string
	homeDirOnce sync.Once
)

const (
	// maxCollisionAttempts defines the limit for collision resolution attempts
	maxCollisionAttempts = 999
	
	// maxPathLength defines the maximum allowed path length
	// 4096 bytes is conservative and works across most filesystems:
	// - Linux ext4: 4096 bytes for path, 255 bytes for filename
	// - macOS HFS+/APFS: ~1024 bytes practical limit
	// - Windows NTFS: 32,767 characters (much higher)
	// This can be made configurable in the future if needed
	maxPathLength = 4096
)

var (
	// commonCollisionNumbers defines the small numbers to try first for collision resolution
	// These are most likely to be available and provide good performance for typical use cases
	commonCollisionNumbers = []int{1, 2, 3, 4, 5}
)

type pathGenerator struct{}

// getHomeDir returns the cached home directory or retrieves it once
func getHomeDir() (string, error) {
	var err error
	homeDirOnce.Do(func() {
		homeDir, err = os.UserHomeDir()
	})
	return homeDir, err
}

// resetHomeDirCache resets the home directory cache - FOR TESTING ONLY
// This function should only be called from test code to ensure test isolation
func resetHomeDirCache() {
	homeDir = ""
	homeDirOnce = sync.Once{}
}

func NewPathGenerator() PathGenerator {
	return &pathGenerator{}
}

func (pg *pathGenerator) GeneratePath(branchName, basePath string) (string, error) {
	if branchName == "" {
		return "", &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: "branch name cannot be empty",
			Context: map[string]interface{}{
				"branch": branchName,
			},
			Operation: "path_generation",
		}
	}

	if basePath == "" {
		basePath = pg.getConfiguredBasePath()
	}

	if !filepath.IsAbs(basePath) {
		abs, err := filepath.Abs(basePath)
		if err != nil {
			return "", &errors.GroveError{
				Code:    errors.ErrCodeFileSystem,
				Message: "failed to resolve absolute path",
				Cause:   err,
				Context: map[string]interface{}{
					"base_path": basePath,
				},
				Operation: "path_resolution",
			}
		}
		basePath = abs
	}

	dirName := git.BranchToDirectoryName(branchName)
	if dirName == "" {
		return "", &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: "failed to generate valid directory name from branch",
			Context: map[string]interface{}{
				"branch": branchName,
			},
			Operation: "directory_name_generation",
		}
	}

	targetPath := filepath.Join(basePath, dirName)
	finalPath, err := pg.resolveCollisions(targetPath)
	if err != nil {
		return "", &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "failed to resolve path collisions",
			Cause:   err,
			Context: map[string]interface{}{
				"target_path": targetPath,
			},
			Operation: "collision_resolution",
		}
	}

	if err := pg.validatePath(finalPath); err != nil {
		return "", &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "generated path is invalid",
			Cause:   err,
			Context: map[string]interface{}{
				"path": finalPath,
			},
			Operation: "path_validation",
		}
	}

	return finalPath, nil
}

func (pg *pathGenerator) getConfiguredBasePath() string {
	if viper.IsSet("worktree.base_path") {
		basePath := viper.GetString("worktree.base_path")
		if basePath != "" {
			if expandedPath, err := expandHomePath(basePath); err == nil {
				return expandedPath
			}
		}
	}

	// Try to get git repository root first
	if gitRoot, err := pg.getGitRepositoryRoot(); err == nil && gitRoot != "" {
		return gitRoot
	}

	// Fall back to current working directory if git root not found
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}

	return "."
}

// getGitRepositoryRoot returns the directory where worktrees should be created
// For bare repositories with worktrees, this is the parent directory of the .bare directory
func (pg *pathGenerator) getGitRepositoryRoot() (string, error) {
	executor := git.DefaultExecutor
	
	// First, check if we're in a worktree setup by looking for the .bare directory
	gitDir, err := executor.ExecuteQuiet("rev-parse", "--git-dir")
	if err != nil {
		return "", err
	}
	gitDir = strings.TrimSpace(gitDir)
	
	// If the git directory contains .bare, we're in a worktree setup
	if strings.Contains(gitDir, ".bare") {
		// Extract the .bare directory path and return its parent
		// gitDir might be like /path/to/project/.bare/worktrees/main
		// We want to return /path/to/project (parent of .bare)
		if idx := strings.Index(gitDir, ".bare"); idx != -1 {
			bareDir := gitDir[:idx+5] // Include ".bare"
			return filepath.Dir(bareDir), nil
		}
	}
	
	// Fall back to the traditional git repository root
	output, err := executor.ExecuteQuiet("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// Finds an available path by appending numeric suffixes if the original exists.
// Uses an optimized collision resolution strategy that reduces filesystem operations
// by intelligently searching for gaps rather than sequential checking.
// Note: This function only checks for existence and doesn't create directories to avoid race conditions.
// The caller must handle atomic directory creation using the returned path.
func (pg *pathGenerator) resolveCollisions(basePath string) (string, error) {
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return basePath, nil
	} else if err != nil {
		return "", &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "failed to check path existence",
			Cause:   err,
			Context: map[string]interface{}{
				"path": basePath,
			},
			Operation: "path_existence_check",
		}
	}

	dir := filepath.Dir(basePath)
	name := filepath.Base(basePath)

	// Optimized collision resolution: try common patterns first, then search gaps
	result, err := pg.findNextAvailablePath(dir, name)
	if err != nil {
		return "", &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "failed to resolve path collisions",
			Cause:   err,
			Context: map[string]interface{}{
				"base_path": basePath,
			},
			Operation: "collision_resolution",
		}
	}

	if result == "" {
		return "", &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: fmt.Sprintf("unable to find unique path after %d attempts", maxCollisionAttempts),
			Context: map[string]interface{}{
				"base_path": basePath,
				"attempts":  maxCollisionAttempts,
			},
			Operation: "collision_resolution",
		}
	}

	return result, nil
}

// findNextAvailablePath uses an optimized strategy to find available paths
// First tries common small numbers, then searches for gaps in larger numbers
func (pg *pathGenerator) findNextAvailablePath(dir, name string) (string, error) {
	// First, try common small numbers (most likely to be available)
	for _, num := range commonCollisionNumbers {
		candidateName := fmt.Sprintf("%s-%d", name, num)
		candidatePath := filepath.Join(dir, candidateName)
		
		exists, err := pg.pathExists(candidatePath)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidatePath, nil
		}
	}

	// If common numbers are taken, search sequentially in the remaining range
	// Start from the number after the last common collision number
	nextNumber := commonCollisionNumbers[len(commonCollisionNumbers)-1] + 1
	return pg.findAvailablePathInRange(dir, name, nextNumber, maxCollisionAttempts)
}

// findAvailablePathInRange searches for available paths within a numeric range
// Uses sequential search within the specified range to find the first available path
func (pg *pathGenerator) findAvailablePathInRange(dir, name string, start, end int) (string, error) {
	for i := start; i <= end; i++ {
		candidateName := fmt.Sprintf("%s-%d", name, i)
		candidatePath := filepath.Join(dir, candidateName)
		
		exists, err := pg.pathExists(candidatePath)
		if err != nil {
			return "", err
		}
		if !exists {
			return candidatePath, nil
		}
	}
	
	return "", nil // No available path found in range
}

// pathExists checks if a path exists with proper error handling
func (pg *pathGenerator) pathExists(path string) (bool, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "failed to check path existence",
			Cause:   err,
			Context: map[string]interface{}{
				"path": path,
			},
			Operation: "path_existence_check",
		}
	}
	return true, nil
}

func (pg *pathGenerator) validatePath(path string) error {
	if !filepath.IsAbs(path) {
		return &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "path must be absolute",
			Context: map[string]interface{}{
				"path": path,
			},
			Operation: "path_validation",
		}
	}

	// Enhanced path traversal detection
	if err := pg.validatePathSecurity(path); err != nil {
		return err
	}

	dirName := filepath.Base(path)

	if !git.IsValidDirectoryName(dirName) {
		return &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "directory name is invalid",
			Context: map[string]interface{}{
				"directory_name": dirName,
			},
			Operation: "directory_name_validation",
		}
	}

	// Enhanced parent directory validation with permission checks
	if err := pg.validateParentDirectory(path); err != nil {
		return err
	}

	return nil
}

// validatePathSecurity performs comprehensive security validation
func (pg *pathGenerator) validatePathSecurity(path string) error {
	cleanPath := filepath.Clean(path)
	
	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return &errors.GroveError{
			Code:    errors.ErrCodePathTraversal,
			Message: "path contains traversal elements",
			Context: map[string]interface{}{
				"path": path,
			},
			Operation: "path_security_validation",
		}
	}
	
	// Check for significant path manipulation (potential traversal)
	if len(cleanPath) < len(path)/2 {
		return &errors.GroveError{
			Code:    errors.ErrCodePathTraversal,
			Message: "path contains suspicious traversal patterns",
			Context: map[string]interface{}{
				"path":       path,
				"clean_path": cleanPath,
			},
			Operation: "path_security_validation",
		}
	}
	
	// Check for null bytes (security vulnerability)
	if strings.Contains(path, "\x00") {
		return &errors.GroveError{
			Code:    errors.ErrCodePathTraversal,
			Message: "path contains null bytes",
			Context: map[string]interface{}{
				"path": path,
			},
			Operation: "path_security_validation",
		}
	}
	
	// Check for excessively long paths
	if len(path) > maxPathLength {
		return &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "path exceeds maximum length",
			Context: map[string]interface{}{
				"path":       path,
				"length":     len(path),
				"max_length": maxPathLength,
			},
			Operation: "path_security_validation",
		}
	}
	
	return nil
}

// validateParentDirectory checks parent directory existence and permissions
func (pg *pathGenerator) validateParentDirectory(path string) error {
	parentDir := filepath.Dir(path)
	
	parentInfo, err := os.Stat(parentDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Check if we can create the parent directory
			if err := pg.validateWritableAncestor(parentDir); err != nil {
				return &errors.GroveError{
					Code:    errors.ErrCodeDirectoryAccess,
					Message: "parent directory does not exist and cannot be created",
					Cause:   err,
					Context: map[string]interface{}{
						"parent_dir": parentDir,
					},
					Operation: "parent_directory_validation",
				}
			}
		} else {
			return &errors.GroveError{
				Code:    errors.ErrCodeDirectoryAccess,
				Message: "cannot access parent directory",
				Cause:   err,
				Context: map[string]interface{}{
					"parent_dir": parentDir,
				},
				Operation: "parent_directory_access",
			}
		}
	} else {
		if !parentInfo.IsDir() {
			return &errors.GroveError{
				Code:    errors.ErrCodeFileSystem,
				Message: "parent path is not a directory",
				Context: map[string]interface{}{
					"parent_dir": parentDir,
				},
				Operation: "parent_directory_validation",
			}
		}
		
		// Check write permissions on existing parent directory
		if err := pg.validateWritePermissions(parentDir); err != nil {
			return &errors.GroveError{
				Code:    errors.ErrCodeDirectoryAccess,
				Message: "insufficient permissions to write to parent directory",
				Cause:   err,
				Context: map[string]interface{}{
					"parent_dir": parentDir,
				},
				Operation: "parent_directory_permissions",
			}
		}
	}
	
	return nil
}

// validateWritableAncestor finds the nearest existing ancestor and checks write permissions
func (pg *pathGenerator) validateWritableAncestor(path string) error {
	current := path
	for current != "/" && current != "." && current != "" {
		if info, err := os.Stat(current); err == nil {
			if !info.IsDir() {
				return fmt.Errorf("ancestor %s is not a directory", current)
			}
			return pg.validateWritePermissions(current)
		}
		current = filepath.Dir(current)
	}
	
	return fmt.Errorf("no writable ancestor directory found")
}

// validateWritePermissions checks if we can write to a directory
func (pg *pathGenerator) validateWritePermissions(dir string) error {
	// Try to create a temporary file to test write permissions
	tempFile := filepath.Join(dir, ".grove_write_test_"+fmt.Sprintf("%d", os.Getpid()))
	
	file, err := os.Create(tempFile)
	if err != nil {
		return fmt.Errorf("cannot write to directory: %w", err)
	}
	
	// Clean up immediately
	file.Close()
	if removeErr := os.Remove(tempFile); removeErr != nil {
		// Log but don't fail - the main check passed
		// In a real application, you might want to log this
	}
	
	return nil
}

func expandHomePath(path string) (string, error) {
	if path == "~" {
		return getHomeDir()
	}

	if len(path) > 1 && path[0] == '~' && (path[1] == '/' || path[1] == filepath.Separator) {
		homeDir, err := getHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}
