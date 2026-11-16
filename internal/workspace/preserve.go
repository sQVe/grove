package workspace

import (
	"errors"
	"fmt"
	"io"
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

func copyFileIntoRoot(root *os.Root, sourcePath, relativePath string) error {
	input, err := os.Open(sourcePath) //nolint:gosec // G304: source path is selected by preserve configuration; source containment is not checked here.
	if err != nil {
		return err
	}
	defer func() { _ = input.Close() }()

	output, err := root.OpenFile(relativePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fs.FileGit)
	if err != nil {
		return err
	}

	if _, err := io.Copy(output, input); err != nil {
		_ = output.Close()
		_ = root.Remove(relativePath)
		return err
	}
	return output.Close()
}

func copyPreservedFile(root *os.Root, sourcePath, relativePath string, result *PreserveResult) error {
	if err := root.MkdirAll(filepath.Dir(relativePath), fs.DirGit); err != nil {
		logger.Warning("Skipping preserved file %s: cannot create its directory in the worktree: %v", relativePath, err)
		return nil
	}

	err := copyFileIntoRoot(root, sourcePath, relativePath)
	if errors.Is(err, os.ErrExist) {
		result.Skipped = append(result.Skipped, relativePath)
		return nil
	}
	if err != nil {
		return err
	}

	result.Copied = append(result.Copied, relativePath)
	return nil
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

	root, err := os.OpenRoot(destDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	logger.Debug("Preserving files from %s to %s: %v", sourceDir, destDir, filesToCopy)

	for _, file := range filesToCopy {
		sourcePath := filepath.Join(sourceDir, file)

		if _, err := os.Stat(sourcePath); err != nil {
			logger.Debug("Source file does not exist, skipping: %s", sourcePath)
			continue
		}

		if err := copyPreservedFile(root, sourcePath, file, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

// PreserveDirectoriesToWorktree recursively copies named directories from source to dest.
// Skips directories that don't exist in source. Skips individual files that already exist in dest.
// Rejects absolute paths and paths that resolve outside the source directory.
func PreserveDirectoriesToWorktree(sourceDir, destDir string, directories []string) (*PreserveResult, error) {
	result := &PreserveResult{}

	if len(directories) == 0 {
		return result, nil
	}

	root, err := os.OpenRoot(destDir)
	if err != nil {
		return nil, err
	}
	defer func() { _ = root.Close() }()

	for _, dir := range directories {
		cleaned := filepath.Clean(dir)
		if isPathTraversal(dir) {
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

			if d.IsDir() {
				// Visit files even when directory creation fails so each gets a warning.
				_ = root.MkdirAll(relPath, fs.DirGit)
				return nil
			}

			return copyPreservedFile(root, path, relPath, result)
		})
		if err != nil {
			return nil, fmt.Errorf("failed to preserve directory %s: %w", dir, err)
		}
	}

	return result, nil
}

func isPathTraversal(path string) bool {
	cleaned := filepath.Clean(path)
	return filepath.IsAbs(cleaned) || cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator))
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
