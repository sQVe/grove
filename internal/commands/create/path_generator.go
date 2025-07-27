package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/git"
)

type pathGenerator struct{}

func NewPathGenerator() PathGenerator {
	return &pathGenerator{}
}

func (pg *pathGenerator) GeneratePath(branchName, basePath string) (string, error) {
	if branchName == "" {
		return "", fmt.Errorf("branch name cannot be empty")
	}

	if basePath == "" {
		basePath = pg.getConfiguredBasePath()
	}

	if !filepath.IsAbs(basePath) {
		abs, err := filepath.Abs(basePath)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path for %s: %w", basePath, err)
		}
		basePath = abs
	}

	// Generate safe directory name from branch name.
	dirName := git.BranchToDirectoryName(branchName)
	if dirName == "" {
		return "", fmt.Errorf("failed to generate valid directory name from branch %s", branchName)
	}

	targetPath := filepath.Join(basePath, dirName)
	finalPath, err := pg.resolveCollisions(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve path collisions for %s: %w", targetPath, err)
	}

	// Validate the final path.
	if err := pg.validatePath(finalPath); err != nil {
		return "", fmt.Errorf("generated path is invalid: %w", err)
	}

	return finalPath, nil
}

func (pg *pathGenerator) getConfiguredBasePath() string {
	// Try to get from worktree configuration.
	if viper.IsSet("worktree.base_path") {
		basePath := viper.GetString("worktree.base_path")
		if basePath != "" {
			// Expand home directory if present.
			if expandedPath, err := expandHomePath(basePath); err == nil {
				return expandedPath
			}
		}
	}

	// Fallback to current directory.
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}

	return "."
}

func (pg *pathGenerator) resolveCollisions(basePath string) (string, error) {
	// Check if path already exists.
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		return basePath, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to check path existence: %w", err)
	}

	// Path exists, try with suffixes.
	dir := filepath.Dir(basePath)
	name := filepath.Base(basePath)

	for i := 1; i <= 999; i++ { // Prevents infinite loops when finding unique paths.
		candidateName := fmt.Sprintf("%s-%d", name, i)
		candidatePath := filepath.Join(dir, candidateName)

		if _, err := os.Stat(candidatePath); os.IsNotExist(err) {
			return candidatePath, nil
		} else if err != nil {
			return "", fmt.Errorf("failed to check candidate path %s: %w", candidatePath, err)
		}
	}

	return "", fmt.Errorf("unable to find unique path after 999 attempts for %s", basePath)
}

func (pg *pathGenerator) validatePath(path string) error {
	// Ensure path is absolute.
	if !filepath.IsAbs(path) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	cleanPath := filepath.Clean(path)

	if strings.Contains(path, "..") || len(cleanPath) < len(path)/2 {
		return fmt.Errorf("path contains traversal elements: %s", path)
	}

	dirName := filepath.Base(path)

	if !git.IsValidDirectoryName(dirName) {
		return fmt.Errorf("directory name is invalid: %s", dirName)
	}

	parentDir := filepath.Dir(path)
	if parentInfo, err := os.Stat(parentDir); err != nil {
		if os.IsNotExist(err) {
			// Only fail for clearly invalid parent directories like "/" or "".
			if parentDir == "/" || parentDir == "" {
				return fmt.Errorf("invalid parent directory: %s", parentDir)
			}
			// Allow non-existent parents since they may be created later.
		} else {
			return fmt.Errorf("cannot access parent directory %s: %w", parentDir, err)
		}
	} else if !parentInfo.IsDir() {
		return fmt.Errorf("parent path is not a directory: %s", parentDir)
	}

	return nil
}

func expandHomePath(path string) (string, error) {
	if path == "~" {
		return os.UserHomeDir()
	}

	if len(path) > 1 && path[0] == '~' && (path[1] == '/' || path[1] == filepath.Separator) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, path[2:]), nil
	}

	return path, nil
}
