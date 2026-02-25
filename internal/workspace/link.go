package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// LinkResult holds the outcome of a directory linking operation.
type LinkResult struct {
	Linked  []string
	Skipped []string
}

// LinkDirectoriesToWorktree creates relative symlinks in destDir for directories
// in sourceDir whose names match any of the given patterns.
// Existing paths in destDir are skipped, never overwritten.
func LinkDirectoriesToWorktree(sourceDir, destDir string, patterns []string) (*LinkResult, error) {
	result := &LinkResult{}

	if len(patterns) == 0 {
		return result, nil
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return result, err
	}

	for _, entry := range entries {
		isDir := entry.IsDir()
		if !isDir && entry.Type()&os.ModeSymlink != 0 {
			target, err := os.Stat(filepath.Join(sourceDir, entry.Name()))
			if err == nil && target.IsDir() {
				isDir = true
			}
		}
		if !isDir {
			continue
		}

		name := entry.Name()
		if !matchesAnyLinkPattern(name, patterns) {
			continue
		}

		destPath := filepath.Join(destDir, name)
		if _, err := os.Lstat(destPath); err == nil {
			result.Skipped = append(result.Skipped, name)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			return result, fmt.Errorf("checking dest path %s: %w", destPath, err)
		}

		relTarget, err := filepath.Rel(destDir, filepath.Join(sourceDir, name))
		if err != nil {
			return result, err
		}

		if err := os.Symlink(relTarget, destPath); err != nil {
			return result, fmt.Errorf("symlink %s: %w", name, err)
		}

		result.Linked = append(result.Linked, name)
	}

	return result, nil
}

func matchesAnyLinkPattern(name string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, name)
		if err == nil && matched {
			return true
		}
	}
	return false
}
