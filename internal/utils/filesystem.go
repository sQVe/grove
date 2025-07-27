package utils

import (
	"fmt"
	"os"

	"github.com/sqve/grove/internal/logger"
)

// WithDirectoryChange executes a function within a specific directory and restores the original directory afterwards.
// This utility eliminates the need for repetitive directory change patterns throughout the codebase.
func WithDirectoryChange(targetDir string, fn func() error) error {
	log := logger.WithComponent("filesystem")

	originalDir, err := os.Getwd()
	if err != nil {
		log.ErrorOperation("failed to get current directory", err)
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(targetDir); err != nil {
		log.ErrorOperation("failed to change directory", err, "target_dir", targetDir)
		return fmt.Errorf("failed to change to directory %s: %w", targetDir, err)
	}

	// Ensure we restore the original directory, even if the operation panics.
	defer func() {
		if restoreErr := os.Chdir(originalDir); restoreErr != nil {
			log.ErrorOperation("failed to restore original directory", restoreErr,
				"original_dir", originalDir, "target_dir", targetDir)
			// Note: We log the error but don't panic here, as the original operation might have succeeded.
		}
	}()

	return fn()
}
