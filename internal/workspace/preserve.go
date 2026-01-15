package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

type PreserveResult struct {
	Copied  []string
	Skipped []string // Already exist in destination
}

// PreserveFilesToWorktree copies matching files, skips existing (never overwrites).
// excludePatterns contains path segments to exclude (e.g., "node_modules").
func PreserveFilesToWorktree(sourceDir, destDir string, patterns, ignoredFiles, excludePatterns []string) (*PreserveResult, error) {
	result := &PreserveResult{}

	if len(ignoredFiles) == 0 || len(patterns) == 0 {
		return result, nil
	}

	var filesToCopy []string
	for _, file := range ignoredFiles {
		// Skip files in excluded paths
		if isExcludedPath(file, excludePatterns) {
			continue
		}

		for _, pattern := range patterns {
			if matchesPattern(file, pattern) {
				filesToCopy = append(filesToCopy, file)
				break
			}
		}
	}

	if len(filesToCopy) == 0 {
		return result, nil
	}

	logger.Debug("Preserving files from %s to %s: %v", sourceDir, destDir, filesToCopy)

	for _, file := range filesToCopy {
		sourcePath := filepath.Join(sourceDir, file)
		destPath := filepath.Join(destDir, file)

		if _, err := os.Stat(sourcePath); err != nil {
			logger.Debug("Source file does not exist, skipping: %s", sourcePath)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), fs.DirGit); err != nil {
			return nil, err
		}

		// Use exclusive copy to atomically skip existing files (avoids TOCTOU race)
		if err := fs.CopyFileExclusive(sourcePath, destPath, fs.FileGit); err != nil {
			if errors.Is(err, os.ErrExist) {
				result.Skipped = append(result.Skipped, file)
				continue
			}
			return nil, err
		}

		result.Copied = append(result.Copied, file)
	}

	return result, nil
}

func FindIgnoredFilesInWorktree(worktreeDir string) ([]string, error) {
	return findIgnoredFiles(worktreeDir)
}

// isExcludedPath checks if a file path contains any of the excluded path segments.
// Git always returns paths with forward slashes, even on Windows.
func isExcludedPath(filePath string, excludePatterns []string) bool {
	if len(excludePatterns) == 0 {
		return false
	}

	// Git returns paths with forward slashes on all platforms
	parts := strings.Split(filePath, "/")
	for _, part := range parts {
		for _, exclude := range excludePatterns {
			if part == exclude {
				return true
			}
		}
	}

	return false
}
