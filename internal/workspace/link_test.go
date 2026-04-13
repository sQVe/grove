package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/testutil"
)

func TestLinkDirectoriesToWorktree(t *testing.T) {
	t.Parallel()

	t.Run("creates symlinks for matching directories", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(sourceDir, ".beads"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".beads"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}

		if len(result.Linked) != 1 || result.Linked[0] != ".beads" {
			t.Errorf("Expected [.beads] in Linked, got %v", result.Linked)
		}

		destPath := filepath.Join(destDir, ".beads")
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

		if err := os.MkdirAll(filepath.Join(sourceDir, ".beads"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		_, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".beads"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}

		target, err := os.Readlink(filepath.Join(destDir, ".beads"))
		if err != nil {
			t.Fatalf("Readlink failed: %v", err)
		}
		if filepath.IsAbs(target) {
			t.Errorf("Expected relative symlink target, got %q", target)
		}

		resolved, err := filepath.EvalSymlinks(filepath.Join(destDir, ".beads"))
		if err != nil {
			t.Fatalf("EvalSymlinks failed: %v", err)
		}
		expected, err := filepath.EvalSymlinks(filepath.Join(sourceDir, ".beads"))
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

		if err := os.MkdirAll(filepath.Join(sourceDir, ".beads"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink("../source/.beads", filepath.Join(destDir, ".beads")); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".beads"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}

		if len(result.Linked) != 0 {
			t.Errorf("Expected nothing linked, got %v", result.Linked)
		}
		if len(result.Skipped) != 1 || result.Skipped[0] != ".beads" {
			t.Errorf("Expected [.beads] in Skipped, got %v", result.Skipped)
		}
		if len(result.Conflicts) != 0 {
			t.Errorf("Expected no conflicts for existing symlink, got %v", result.Conflicts)
		}
	})

	t.Run("reports conflict when dest exists as real directory", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(sourceDir, ".beads"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Join(destDir, ".beads"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".beads"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}

		if len(result.Conflicts) != 1 || result.Conflicts[0] != ".beads" {
			t.Errorf("Expected [.beads] in Conflicts, got %v", result.Conflicts)
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

		if err := os.WriteFile(filepath.Join(sourceDir, ".beads"), []byte("file"), 0o644); err != nil { //nolint:gosec
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".beads"})
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

		for _, name := range []string{".beads", ".cursor"} {
			if err := os.MkdirAll(filepath.Join(sourceDir, name), fs.DirStrict); err != nil {
				t.Fatal(err)
			}
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".beads", ".cursor"})
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
		if err := os.Symlink(filepath.Join(targetDir, "data"), filepath.Join(sourceDir, ".beads")); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".beads"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}
		if len(result.Linked) != 1 || result.Linked[0] != ".beads" {
			t.Errorf("Expected [.beads] in Linked, got %v", result.Linked)
		}
	})

	t.Run("matches wildcard patterns", func(t *testing.T) {
		t.Parallel()
		sourceDir := testutil.TempDir(t)
		destDir := testutil.TempDir(t)

		if err := os.MkdirAll(filepath.Join(sourceDir, ".beads-data"), fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result, err := LinkDirectoriesToWorktree(sourceDir, destDir, []string{".beads*"})
		if err != nil {
			t.Fatalf("LinkDirectoriesToWorktree failed: %v", err)
		}
		if len(result.Linked) != 1 || result.Linked[0] != ".beads-data" {
			t.Errorf("Expected [.beads-data] in Linked, got %v", result.Linked)
		}
	})
}
