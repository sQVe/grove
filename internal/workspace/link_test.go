package workspace

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/testutil"
)

func TestLinkDirectoriesToWorktree(t *testing.T) {
	t.Parallel()

	t.Run("links parents before children regardless of pattern order", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct {
			name     string
			patterns []string
		}{
			{"child first", []string{"apps/*/node_modules", "apps/*"}},
			{"parent first", []string{"apps/*", "apps/*/node_modules"}},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				sourceDir := testutil.TempDir(t)
				destDir := testutil.TempDir(t)
				parent := filepath.Join("apps", "a")
				if err := os.MkdirAll(filepath.Join(sourceDir, parent, "node_modules"), fs.DirStrict); err != nil {
					t.Fatal(err)
				}

				result, err := LinkDirectoriesToWorktree(sourceDir, destDir, tc.patterns)
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Linked) != 1 || result.Linked[0] != parent || len(result.Skipped) != 0 || len(result.Conflicts) != 0 {
					t.Errorf("Expected only parent %q linked, got %+v", parent, result)
				}
			})
		}
	})

	t.Run("deduplicates overlapping patterns in match order", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)
		names := []string{filepath.Join("apps", "b", "node_modules"), filepath.Join("apps", "a", "node_modules")}
		for _, name := range names {
			if err := os.MkdirAll(filepath.Join(sourceDir, name), fs.DirStrict); err != nil {
				t.Fatal(err)
			}
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{"apps/b/node_modules", "apps/*/node_modules", "apps/a/node_modules"})
		if err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(result.Linked, names) {
			t.Errorf("Expected %v in Linked, got %v", names, result.Linked)
		}
		if len(result.Skipped) != 0 || len(result.Conflicts) != 0 {
			t.Errorf("Expected no skips or conflicts, got %+v", result)
		}
	})

	t.Run("ignores nested paths beneath linked parents", func(t *testing.T) {
		t.Parallel()
		for _, parent := range []string{"apps", "apps/a"} {
			t.Run(parent, func(t *testing.T) {
				t.Parallel()
				sourceDir := testutil.TempDir(t)
				destDir := testutil.TempDir(t)
				name := filepath.Join("apps", "a", "node_modules")
				if err := os.MkdirAll(filepath.Join(sourceDir, name), fs.DirStrict); err != nil {
					t.Fatal(err)
				}

				result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{parent, "apps/*/node_modules"})
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Linked) != 1 || result.Linked[0] != filepath.FromSlash(parent) || len(result.Skipped) != 0 || len(result.Conflicts) != 0 {
					t.Errorf("Expected only parent %q linked, got %+v", parent, result)
				}
				info, err := os.Stat(filepath.Join(destDir, name))
				if err != nil || !info.IsDir() {
					t.Fatalf("Nested directory is not reachable: %v", err)
				}
			})
		}
	})

	t.Run("does not create directories through existing parent symlinks", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)
		targetDir := testutil.TempDir(t)
		if err := os.MkdirAll(filepath.Join(sourceDir, "apps", "a", "node_modules"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(targetDir, filepath.Join(destDir, "apps")); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{"apps/*/node_modules"})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Linked) != 0 || len(result.Skipped) != 0 || len(result.Conflicts) != 0 {
			t.Errorf("Expected empty result, got %+v", result)
		}
		entries, err := os.ReadDir(targetDir)
		if err != nil {
			t.Fatal(err)
		}
		if len(entries) != 0 {
			t.Errorf("Expected symlink target untouched, got %v", entries)
		}
	})

	t.Run("reports a file at a dest parent as a conflict and keeps linking", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			nested   string
			pattern  string
			conflict string
		}{
			{"one parent component", filepath.Join("apps", "node_modules"), "apps/node_modules", filepath.Join("apps", "node_modules")},
			{"two parent components", filepath.Join("apps", "a", "node_modules"), "apps/*/node_modules", filepath.Join("apps", "a", "node_modules")},
		}
		for _, tc := range tests {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				sourceDir := testutil.TempDir(t)
				destDir := testutil.TempDir(t)
				if err := os.MkdirAll(filepath.Join(sourceDir, tc.nested), fs.DirStrict); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(sourceDir, "vendor"), fs.DirStrict); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(destDir, "apps"), []byte("not a dir"), fs.FileStrict); err != nil {
					t.Fatal(err)
				}

				result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{tc.pattern, "vendor"})
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Linked) != 1 || result.Linked[0] != "vendor" {
					t.Errorf("Expected [vendor] in Linked, got %v", result.Linked)
				}
				if len(result.Conflicts) != 1 || result.Conflicts[0] != tc.conflict {
					t.Errorf("Expected [%s] in Conflicts, got %v", tc.conflict, result.Conflicts)
				}
			})
		}
	})

	t.Run("preserves existing nested destinations and ignores source files", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)
		conflict := filepath.Join("apps", "a", "node_modules")
		skipped := filepath.Join("apps", "b", "node_modules")
		for _, name := range []string{conflict, skipped} {
			if err := os.MkdirAll(filepath.Join(sourceDir, name), fs.DirStrict); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Dir(filepath.Join(destDir, name)), fs.DirStrict); err != nil {
				t.Fatal(err)
			}
		}
		if err := os.Mkdir(filepath.Join(destDir, conflict), fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("missing", filepath.Join(destDir, skipped)); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(sourceDir, "apps", "c"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(sourceDir, "apps", "c", "node_modules"), []byte("file"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{"apps/*/node_modules"})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Linked) != 0 {
			t.Errorf("Expected nothing linked, got %v", result.Linked)
		}
		if len(result.Conflicts) != 1 || result.Conflicts[0] != conflict {
			t.Errorf("Expected [%s] in Conflicts, got %v", conflict, result.Conflicts)
		}
		if len(result.Skipped) != 1 || result.Skipped[0] != skipped {
			t.Errorf("Expected [%s] in Skipped, got %v", skipped, result.Skipped)
		}
		info, err := os.Lstat(filepath.Join(destDir, conflict))
		if err != nil || !info.IsDir() {
			t.Fatalf("Existing directory changed: %v", err)
		}
		target, err := os.Readlink(filepath.Join(destDir, skipped))
		if err != nil || target != "missing" {
			t.Fatalf("Existing symlink changed: target %q, error %v", target, err)
		}
	})

	t.Run("rejects absolute and parent traversal patterns", func(t *testing.T) {
		t.Parallel()
		for _, tc := range []struct{ name, pattern string }{
			{"rejects absolute paths", "/apps/*/node_modules"},
			{"rejects parent prefixes", "../source/apps/*/node_modules"},
			{"rejects escaping parent segments", "apps/../../source/apps/*/node_modules"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				root := testutil.TempDir(t)
				sourceDir := filepath.Join(root, "source")
				destDir := filepath.Join(root, "dest")
				if err := os.MkdirAll(filepath.Join(sourceDir, "apps", "a", "node_modules"), fs.DirStrict); err != nil {
					t.Fatal(err)
				}
				if err := os.MkdirAll(destDir, fs.DirStrict); err != nil {
					t.Fatal(err)
				}
				pattern := tc.pattern
				if pattern == "/apps/*/node_modules" {
					pattern = filepath.VolumeName(sourceDir) + pattern
				}
				result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{pattern})
				if err != nil {
					t.Fatal(err)
				}
				if len(result.Linked) != 0 || len(result.Skipped) != 0 || len(result.Conflicts) != 0 {
					t.Errorf("Expected invalid pattern %q to be ignored, got %+v", pattern, result)
				}
			})
		}
	})

	t.Run("accepts parent segments that clean to a safe path", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)
		if err := os.Mkdir(filepath.Join(sourceDir, "b"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{"a/../b"})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Linked) != 1 || result.Linked[0] != "b" {
			t.Errorf("Expected [b] in Linked, got %v", result.Linked)
		}
	})

	t.Run("links nested directories with resolving relative targets", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)
		names := []string{filepath.Join("apps", "a", "node_modules"), filepath.Join("apps", "b", "node_modules")}
		for _, name := range names {
			if err := os.MkdirAll(filepath.Join(sourceDir, name), fs.DirStrict); err != nil {
				t.Fatal(err)
			}
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{"apps/*/node_modules"})
		if err != nil {
			t.Fatal(err)
		}
		if len(result.Linked) != len(names) {
			t.Fatalf("Expected %v in Linked, got %v", names, result.Linked)
		}
		for i, name := range names {
			if result.Linked[i] != name {
				t.Errorf("Expected %q in Linked, got %q", name, result.Linked[i])
			}
			destPath := filepath.Join(destDir, name)
			sourcePath := filepath.Join(sourceDir, name)
			target, err := os.Readlink(destPath)
			if err != nil {
				t.Fatal(err)
			}
			expected, err := filepath.Rel(filepath.Dir(destPath), sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			if target != expected || filepath.IsAbs(target) {
				t.Errorf("Expected relative target %q, got %q", expected, target)
			}
			sourceInfo, err := os.Stat(sourcePath)
			if err != nil {
				t.Fatal(err)
			}
			destInfo, err := os.Stat(destPath)
			if err != nil {
				t.Fatalf("Symlink does not resolve: %v", err)
			}
			if !destInfo.IsDir() || !os.SameFile(sourceInfo, destInfo) {
				t.Errorf("Symlink %q does not reach source directory %q", destPath, sourcePath)
			}
		}
	})

	t.Run("creates symlinks for matching directories", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(sourceDir, ".claude"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".claude"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}

		if len(result.Linked) != 1 || result.Linked[0] != ".claude" {
			t.Errorf("Expected [.claude] in Linked, got %v", result.Linked)
		}

		destPath := filepath.Join(destDir, ".claude")
		info, err := os.Lstat(destPath)
		if err != nil {
			t.Fatalf("Symlink not created: %v", err)
		}
		if info.Mode()&os.ModeSymlink == 0 {
			t.Error("Expected symlink, got regular file/dir")
		}
	})

	t.Run("creates relative symlinks", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(sourceDir, ".claude"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		_, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".claude"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}

		target, err := os.Readlink(filepath.Join(destDir, ".claude"))
		if err != nil {
			t.Fatalf("Readlink failed: %v", err)
		}
		if filepath.IsAbs(target) {
			t.Errorf("Expected relative symlink target, got %q", target)
		}

		resolved, err := filepath.EvalSymlinks(filepath.Join(destDir, ".claude"))
		if err != nil {
			t.Fatalf("EvalSymlinks failed: %v", err)
		}
		expected, err := filepath.EvalSymlinks(filepath.Join(sourceDir, ".claude"))
		if err != nil {
			t.Fatal(err)
		}
		if resolved != expected {
			t.Errorf("Symlink resolves to %q, want %q", resolved, expected)
		}
	})

	t.Run("skips when dest is already a symlink", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(sourceDir, ".claude"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../source/.claude", filepath.Join(destDir, ".claude")); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".claude"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}

		if len(result.Linked) != 0 {
			t.Errorf("Expected nothing linked, got %v", result.Linked)
		}
		if len(result.Skipped) != 1 || result.Skipped[0] != ".claude" {
			t.Errorf("Expected [.claude] in Skipped, got %v", result.Skipped)
		}
		if len(result.Conflicts) != 0 {
			t.Errorf("Expected no conflicts for existing symlink, got %v", result.Conflicts)
		}
	})

	t.Run("reports conflict when dest exists as real directory", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(sourceDir, ".claude"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(destDir, ".claude"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".claude"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}

		if len(result.Conflicts) != 1 || result.Conflicts[0] != ".claude" {
			t.Errorf("Expected [.claude] in Conflicts, got %v", result.Conflicts)
		}
		if len(result.Skipped) != 0 {
			t.Errorf("Expected Skipped empty when dest is real dir, got %v", result.Skipped)
		}
	})

	t.Run("returns empty result for empty patterns", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}
		if len(result.Linked) != 0 || len(result.Skipped) != 0 {
			t.Errorf("Expected empty result, got %+v", result)
		}
	})

	t.Run("returns empty result for nil patterns", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, nil)
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}
		if len(result.Linked) != 0 || len(result.Skipped) != 0 {
			t.Errorf("Expected empty result, got %+v", result)
		}
	})

	t.Run("only links directories, not files", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.WriteFile(filepath.Join(sourceDir, ".claude"), []byte("file"), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".claude"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}
		if len(result.Linked) != 0 {
			t.Errorf("Expected nothing linked for file, got %v", result.Linked)
		}
	})

	t.Run("links multiple matching directories", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		for _, name := range []string{".claude", ".cursor"} {
			if err := os.MkdirAll(filepath.Join(sourceDir, name), fs.DirStrict); err != nil {
				t.Fatal(err)
			}
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".claude", ".cursor"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}
		if len(result.Linked) != 2 {
			t.Errorf("Expected 2 linked, got %v", result.Linked)
		}
	})

	t.Run("follows symlinks to directories in source", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)
		targetDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(targetDir, "data"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.Join(targetDir, "data"), filepath.Join(sourceDir, ".claude")); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".claude"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}
		if len(result.Linked) != 1 || result.Linked[0] != ".claude" {
			t.Errorf("Expected [.claude] in Linked, got %v", result.Linked)
		}
	})

	t.Run("matches wildcard patterns", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(sourceDir, ".claude-data"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".claude*"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}
		if len(result.Linked) != 1 || result.Linked[0] != ".claude-data" {
			t.Errorf("Expected [.claude-data] in Linked, got %v", result.Linked)
		}
	})
}
