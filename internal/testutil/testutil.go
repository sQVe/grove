package testutil

import (
	"path/filepath"
	"testing"
)

// TempDir returns a temp directory with symlinks resolved.
// On macOS, /var symlinks to /private/var which causes path mismatches
// when comparing with git output.
func TempDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	resolved, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("failed to resolve symlinks: %v", err)
	}
	return resolved
}
