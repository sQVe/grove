package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
)

type pathGenerator struct{}

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

	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}

	return "."
}

// Finds an available path by appending numeric suffixes if the original exists.
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

	const maxCollisionAttempts = 999

	for i := 1; i <= maxCollisionAttempts; i++ {
		candidateName := fmt.Sprintf("%s-%d", name, i)
		candidatePath := filepath.Join(dir, candidateName)

		if _, err := os.Stat(candidatePath); os.IsNotExist(err) {
			return candidatePath, nil
		} else if err != nil {
			return "", &errors.GroveError{
				Code:    errors.ErrCodeFileSystem,
				Message: "failed to check candidate path",
				Cause:   err,
				Context: map[string]interface{}{
					"candidate_path": candidatePath,
				},
				Operation: "candidate_path_check",
			}
		}
	}

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

	cleanPath := filepath.Clean(path)

	if strings.Contains(path, "..") || len(cleanPath) < len(path)/2 {
		return &errors.GroveError{
			Code:    errors.ErrCodePathTraversal,
			Message: "path contains traversal elements",
			Context: map[string]interface{}{
				"path": path,
			},
			Operation: "path_validation",
		}
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

	parentDir := filepath.Dir(path)
	if parentInfo, err := os.Stat(parentDir); err != nil {
		if os.IsNotExist(err) {
			if parentDir == "/" || parentDir == "" {
				return &errors.GroveError{
					Code:    errors.ErrCodeFileSystem,
					Message: "invalid parent directory",
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
	} else if !parentInfo.IsDir() {
		return &errors.GroveError{
			Code:    errors.ErrCodeFileSystem,
			Message: "parent path is not a directory",
			Context: map[string]interface{}{
				"parent_dir": parentDir,
			},
			Operation: "parent_directory_validation",
		}
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
