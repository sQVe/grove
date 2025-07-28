package validation

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/errors"
)

const (
	// testFilePrefix is used for permission testing to avoid conflicts with user files.
	testFilePrefix = ".grove_write_test"
)

func generateTestFileName() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%x", testFilePrefix, b), nil
}

func ValidatePath(path string) error {
	if path == "" {
		return nil // Empty path is valid, will use default.
	}

	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return errors.ErrPathTraversal(path)
	}

	// Prevent accidental overwrites of existing directories.
	if _, err := os.Stat(cleanPath); err == nil {
		return errors.ErrPathExists(cleanPath).
			WithContext("suggestion", "Use --force to overwrite or choose a different path")
	}

	// Ensure we can create the worktree directory by testing parent permissions.
	parentDir := filepath.Dir(cleanPath)
	if parentDir != "." {
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			// Parent doesn't exist, that's okay, we'll create it.
			return nil
		}
		// Test directory creation permissions with a unique test file.
		testFileName, err := generateTestFileName()
		if err != nil {
			return errors.ErrDirectoryAccess(parentDir, fmt.Errorf("failed to generate test filename: %v", err))
		}
		testFile := filepath.Join(parentDir, testFileName)

		f, err := os.Create(testFile)
		if err != nil {
			return errors.ErrDirectoryAccess(parentDir, err)
		}
		_ = f.Close()
		_ = os.Remove(testFile)
	}

	return nil
}

func ValidateFilePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			return errors.ErrInvalidPattern(pattern, "pattern cannot be empty")
		}

		// Path traversal attempts pose security risks.
		if strings.Contains(pattern, "..") {
			return errors.ErrInvalidPattern(pattern, "path traversal not allowed")
		}

		// Malformed patterns will cause runtime errors.
		if _, err := filepath.Match(pattern, "test"); err != nil {
			return errors.ErrInvalidPattern(pattern, fmt.Sprintf("invalid glob pattern: %v", err))
		}
	}
	return nil
}
