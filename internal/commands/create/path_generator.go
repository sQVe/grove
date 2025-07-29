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
	"github.com/sqve/grove/internal/logger"
)

var (
	homeDir     string
	homeDirOnce sync.Once
)

// PathGeneratorConfig defines configurable parameters for path generation
type PathGeneratorConfig struct {
	// MaxCollisionAttempts defines the limit for collision resolution attempts
	MaxCollisionAttempts int

	// MaxPathLength defines the maximum allowed path length
	// 4096 bytes is conservative and works across most filesystems:
	// - Linux ext4: 4096 bytes for path, 255 bytes for filename
	// - macOS HFS+/APFS: ~1024 bytes practical limit
	// - Windows NTFS: 32,767 characters (much higher)
	MaxPathLength int

	// CommonCollisionNumbers defines the small numbers to try first for collision resolution
	// These are most likely to be available and provide good performance for typical use cases
	CommonCollisionNumbers []int
}

// DefaultPathGeneratorConfig returns the default configuration
func DefaultPathGeneratorConfig() PathGeneratorConfig {
	return PathGeneratorConfig{
		MaxCollisionAttempts:   999,
		MaxPathLength:          4096,
		CommonCollisionNumbers: []int{1, 2, 3, 4, 5},
	}
}

// getConfigFromViper loads configuration from Viper with defaults
func getConfigFromViper() PathGeneratorConfig {
	config := DefaultPathGeneratorConfig()

	if viper.IsSet("path_generator.max_collision_attempts") {
		config.MaxCollisionAttempts = viper.GetInt("path_generator.max_collision_attempts")
	}

	if viper.IsSet("path_generator.max_path_length") {
		config.MaxPathLength = viper.GetInt("path_generator.max_path_length")
	}

	if viper.IsSet("path_generator.common_collision_numbers") {
		config.CommonCollisionNumbers = viper.GetIntSlice("path_generator.common_collision_numbers")
	}

	return config
}

type pathGenerator struct {
	config PathGeneratorConfig
}

// getHomeDir returns the cached home directory or retrieves it once
func getHomeDir() (string, error) {
	var err error
	homeDirOnce.Do(func() {
		homeDir, err = os.UserHomeDir()
	})
	return homeDir, err
}

// resetHomeDirCache should only be called from test code to ensure test isolation.
func resetHomeDirCache() {
	homeDir = ""
	homeDirOnce = sync.Once{}
}

func NewPathGenerator() PathGenerator {
	return &pathGenerator{
		config: getConfigFromViper(),
	}
}

// ResolveUserPath resolves a user-provided path against the configured base path.
// For relative paths, it resolves against the bare repository root instead of cwd.
func (pg *pathGenerator) ResolveUserPath(userPath string) (string, error) {
	if userPath == "" {
		return "", fmt.Errorf("user path cannot be empty")
	}

	if filepath.IsAbs(userPath) {
		return userPath, nil
	}

	configuredBase := pg.getConfiguredBasePath()
	return filepath.Join(configuredBase, userPath), nil
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
// Uses atomic operations to prevent race conditions between path checking and creation.
func (pg *pathGenerator) resolveCollisions(basePath string) (string, error) {
	// Try atomic creation of the base path first
	if err := pg.tryAtomicPathCreation(basePath); err == nil {
		return basePath, nil
	} else if !os.IsExist(err) {
		// If error is not "already exists", it's a real failure
		return "", &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "failed to check path availability",
			Cause:   err,
			Context: map[string]interface{}{
				"path": basePath,
			},
			Operation: "atomic_path_check",
		}
	}

	dir := filepath.Dir(basePath)
	name := filepath.Base(basePath)

	// Base path exists, find alternative using atomic operations
	result, err := pg.findNextAvailablePathAtomic(dir, name)
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
			Message: fmt.Sprintf("unable to find unique path after %d attempts", pg.config.MaxCollisionAttempts),
			Context: map[string]interface{}{
				"base_path": basePath,
				"attempts":  pg.config.MaxCollisionAttempts,
			},
			Operation: "collision_resolution",
		}
	}

	return result, nil
}

// tryAtomicPathCreation attempts to atomically create a directory, returning error if it exists
func (pg *pathGenerator) tryAtomicPathCreation(path string) error {
	parentDir := filepath.Dir(path)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return err
	}

	return os.Mkdir(path, 0o755)
}

// findNextAvailablePathAtomic uses atomic operations to find and reserve the next available path
func (pg *pathGenerator) findNextAvailablePathAtomic(dir, name string) (string, error) {
	// First, try common small numbers (most likely to be available)
	for _, num := range pg.config.CommonCollisionNumbers {
		candidateName := fmt.Sprintf("%s-%d", name, num)
		candidatePath := filepath.Join(dir, candidateName)

		if err := pg.tryAtomicPathCreation(candidatePath); err == nil {
			return candidatePath, nil
		} else if !os.IsExist(err) {
			// Real error, not just "already exists"
			return "", err
		}
	}

	// If common numbers are taken, search sequentially in the remaining range
	nextNumber := 1 // Default starting point
	if len(pg.config.CommonCollisionNumbers) > 0 {
		nextNumber = pg.config.CommonCollisionNumbers[len(pg.config.CommonCollisionNumbers)-1] + 1
	}
	return pg.findAvailablePathInRangeAtomic(dir, name, nextNumber, pg.config.MaxCollisionAttempts)
}

// findAvailablePathInRangeAtomic searches for available paths within a numeric range using atomic operations
func (pg *pathGenerator) findAvailablePathInRangeAtomic(dir, name string, start, end int) (string, error) {
	for i := start; i <= end; i++ {
		candidateName := fmt.Sprintf("%s-%d", name, i)
		candidatePath := filepath.Join(dir, candidateName)

		if err := pg.tryAtomicPathCreation(candidatePath); err == nil {
			return candidatePath, nil
		} else if !os.IsExist(err) {
			// Real error, not just "already exists"
			return "", err
		}
	}

	return "", nil // No available path found in range
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
	if len(path) > pg.config.MaxPathLength {
		return &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "path exceeds maximum length",
			Context: map[string]interface{}{
				"path":       path,
				"length":     len(path),
				"max_length": pg.config.MaxPathLength,
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

	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logger.Debug("failed to close temp file", "file", tempFile, "error", closeErr)
		}
		if removeErr := os.Remove(tempFile); removeErr != nil {
			logger.Debug("failed to remove temp file", "file", tempFile, "error", removeErr)
		}
	}()

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
