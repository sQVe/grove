package workspace

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

// LinkResult holds the outcome of a directory linking operation.
type LinkResult struct {
	Linked    []string
	Skipped   []string
	Conflicts []string
}

// LinkDirectoriesToWorktree creates relative symlinks in destDir for directories
// in sourceDir whose names or relative paths match any of the given patterns.
// Existing paths in destDir are skipped, never overwritten.
func LinkDirectoriesToWorktree(sourceDir, destDir string, patterns []string) (*LinkResult, error) {
	result := &LinkResult{}

	if len(patterns) == 0 {
		return result, nil
	}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return result, fmt.Errorf("reading source dir %s: %w", sourceDir, err)
	}

	var names []string
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
		names = append(names, name)
	}

	for _, pattern := range patterns {
		if !strings.Contains(pattern, "/") {
			continue
		}
		if isPathTraversal(filepath.FromSlash(pattern)) {
			logger.Debug("Skipping invalid link pattern (path traversal): %s", pattern)
			continue
		}
		matches, err := filepath.Glob(filepath.Join(sourceDir, filepath.FromSlash(pattern)))
		if err != nil {
			continue
		}
		for _, sourcePath := range matches {
			info, err := os.Stat(sourcePath)
			if err != nil || !info.IsDir() {
				continue
			}
			name, err := filepath.Rel(sourceDir, sourcePath)
			if err != nil {
				return result, err
			}
			names = append(names, name)
		}
	}

	seen := make(map[string]bool, len(names))
	names = slices.DeleteFunc(names, func(name string) bool {
		duplicate := seen[name]
		seen[name] = true
		return duplicate
	})

	// Link parents before children so creating a child's directory cannot cause a conflict.
	slices.SortStableFunc(names, func(a, b string) int {
		return strings.Count(a, string(filepath.Separator)) - strings.Count(b, string(filepath.Separator))
	})

linkLoop:
	for _, name := range names {
		parent := destDir
		parts := strings.Split(name, string(filepath.Separator))
		for _, part := range parts[:len(parts)-1] {
			parent = filepath.Join(parent, part)
			if info, err := os.Lstat(parent); err == nil {
				if info.Mode()&os.ModeSymlink != 0 {
					continue linkLoop
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				// A non-directory on the way to destPath blocks this link but not the rest.
				result.Conflicts = append(result.Conflicts, name)
				continue linkLoop
			}
		}

		destPath := filepath.Join(destDir, name)
		if info, err := os.Lstat(destPath); err == nil {
			if info.Mode()&os.ModeSymlink != 0 {
				result.Skipped = append(result.Skipped, name)
			} else {
				result.Conflicts = append(result.Conflicts, name)
			}
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			result.Conflicts = append(result.Conflicts, name)
			continue
		}

		relTarget, err := filepath.Rel(filepath.Dir(destPath), filepath.Join(sourceDir, name))
		if err != nil {
			return result, err
		}

		// Windows reports a file in the parent chain as not-exist rather than ENOTDIR,
		// so the Lstat checks above miss it and this is where that case surfaces.
		if err := os.MkdirAll(filepath.Dir(destPath), fs.DirGit); err != nil {
			result.Conflicts = append(result.Conflicts, name)
			continue
		}

		if err := os.Symlink(relTarget, destPath); err != nil {
			if errors.Is(err, os.ErrExist) {
				result.Skipped = append(result.Skipped, name)
				continue
			}
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
