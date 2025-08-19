package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/validation"
)

// Initialize creates a new grove workspace in the specified directory
func Initialize(path string) error {
	if git.IsInsideGitRepo(path) {
		return fmt.Errorf("cannot initialize grove inside existing git repository")
	}

	if validation.DirectoryExists(path) {
		logger.Debug("Directory %s exists, checking if empty", path)

		isEmpty, err := validation.IsEmptyDir(path)
		if err != nil {
			return fmt.Errorf("failed to check if directory is empty: %w", err)
		}
		if !isEmpty {
			return fmt.Errorf("directory %s is not empty", path)
		}
	} else {
		logger.Debug("Creating new directory: %s", path)
		if err := os.MkdirAll(path, fs.DirGit); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	bareDir := filepath.Join(path, ".bare")
	if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
		return fmt.Errorf("failed to create .bare directory: %w", err)
	}
	logger.Debug("Created .bare directory at %s", bareDir)

	if err := git.InitBare(bareDir); err != nil {
		_ = os.RemoveAll(bareDir)
		return fmt.Errorf("failed to initialize bare git repository: %w", err)
	}
	logger.Debug("Git bare repository initialized successfully")

	gitFile := filepath.Join(path, ".git")
	gitContent := "gitdir: .bare"
	if err := os.WriteFile(gitFile, []byte(gitContent), fs.FileGit); err != nil {
		_ = os.RemoveAll(bareDir)
		return fmt.Errorf("failed to create .git file: %w", err)
	}
	logger.Debug("Created .git file pointing to .bare")

	return nil
}
