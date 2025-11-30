package workspace

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

type PreserveResult struct {
	Copied  []string
	Skipped []string // Already exist in destination
}

// PreserveFilesToWorktree copies matching files, skips existing (never overwrites).
func PreserveFilesToWorktree(sourceDir, destDir string, patterns, ignoredFiles []string) (*PreserveResult, error) {
	result := &PreserveResult{}

	if len(ignoredFiles) == 0 || len(patterns) == 0 {
		return result, nil
	}

	var filesToCopy []string
	for _, file := range ignoredFiles {
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
