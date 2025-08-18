package workspace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/validation"
)

// Initialize creates a new grove workspace in the specified directory
func Initialize(path string) error {
	if validation.DirectoryExists(path) {
		isEmpty, err := validation.IsEmptyDir(path)
		if err != nil {
			return fmt.Errorf("failed to check if directory is empty: %w", err)
		}
		if !isEmpty {
			return fmt.Errorf("directory %s is not empty", path)
		}
	} else {
		if err := os.MkdirAll(path, fs.DirGit); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	bareDir := filepath.Join(path, ".bare")
	if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
		return fmt.Errorf("failed to create .bare directory: %w", err)
	}

	if err := git.InitBare(bareDir); err != nil {
		return fmt.Errorf("failed to initialize bare git repository: %w", err)
	}

	gitFile := filepath.Join(path, ".git")
	gitContent := "gitdir: .bare"
	if err := os.WriteFile(gitFile, []byte(gitContent), fs.FileGit); err != nil {
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	return nil
}
