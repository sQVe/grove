package workspace

import (
	"errors"
	"fmt"
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

// PreserveDirectoriesToWorktree recursively copies named directories from source to dest.
// Skips directories that don't exist in source. Skips individual files that already exist in dest.
// Rejects directory names with path traversal (absolute paths or ".." components).
func PreserveDirectoriesToWorktree(sourceDir, destDir string, directories []string) (*PreserveResult, error) {
	result := &PreserveResult{}

	if len(directories) == 0 {
		return result, nil
	}

	for _, dir := range directories {
		cleaned := filepath.Clean(dir)
		if filepath.IsAbs(cleaned) || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
			logger.Debug("Skipping invalid preserve directory (path traversal): %s", dir)
			continue
		}

		srcPath := filepath.Join(sourceDir, cleaned)
		srcInfo, err := os.Lstat(srcPath)
		if err != nil || !srcInfo.IsDir() {
			logger.Debug("Preserve directory does not exist or is not a directory: %s", srcPath)
			continue
		}

		err = filepath.WalkDir(srcPath, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if d.Type()&os.ModeSymlink != 0 {
				return nil
			}

			relPath, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}

			destPath := filepath.Join(destDir, relPath)

			if d.IsDir() {
				return os.MkdirAll(destPath, fs.DirGit)
			}

			if err := os.MkdirAll(filepath.Dir(destPath), fs.DirGit); err != nil {
				return err
			}

			if err := fs.CopyFileExclusive(path, destPath, fs.FileGit); err != nil {
				if errors.Is(err, os.ErrExist) {
					result.Skipped = append(result.Skipped, relPath)
					return nil
				}
				return err
			}

			result.Copied = append(result.Copied, relPath)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("failed to preserve directory %s: %w", dir, err)
		}
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
