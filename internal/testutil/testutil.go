package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

// TempDir returns a temp directory with symlinks resolved.
// On macOS, /var symlinks to /private/var which causes path mismatches
// when comparing with git output.
//
// Use this instead of t.TempDir() when tests compare filesystem paths with
// git command output (e.g., git worktree list, git rev-parse). For tests
// that don't involve git path comparisons, t.TempDir() is sufficient.
func TempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}
	return resolved
}

// WriteFile writes content to path, creating parent dirs. Fails test on error.
func WriteFile(t *testing.T, path, content string) {
	t.Helper()
	WriteFileMode(t, path, content, 0o644)
}

// WriteFileMode writes content with specific permissions.
func WriteFileMode(t *testing.T, path, content string, perm os.FileMode) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, fs.DirGit); err != nil {
		t.Fatalf("failed to create directory %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}

// MustExec runs a command in dir, fails on error, returns stdout.
func MustExec(t *testing.T, dir, name string, args ...string) string {
	t.Helper()
	cmd := exec.Command(name, args...) // nolint:gosec
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%s %v failed: %v", name, args, err)
	}
	return string(out)
}

// InTempDir executes fn in a temp directory, restoring cwd afterward.
// WARNING: Not safe for use with t.Parallel() as it changes process cwd.
func InTempDir(t *testing.T, fn func(dir string)) {
	t.Helper()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("warning: failed to restore cwd: %v", err)
		}
	}()

	dir := TempDir(t)
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to change to temp directory: %v", err)
	}
	fn(dir)
}
