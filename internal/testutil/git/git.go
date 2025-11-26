package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

// TestRepo provides a test git repository with proper configuration
type TestRepo struct {
	t    *testing.T
	Dir  string
	Path string
}

// NewTestRepo creates a new test repository with git config set up
// By default creates an initial commit. Pass true to skip.
func NewTestRepo(t *testing.T, skipInitialCommit ...bool) *TestRepo {
	t.Helper()

	dir := t.TempDir()
	repoPath := filepath.Join(dir, "repo")

	if err := os.MkdirAll(repoPath, fs.DirGit); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init repo: %v", err)
	}

	configs := [][]string{
		{"commit.gpgsign", "false"},
		{"user.email", "test@example.com"},
		{"user.name", "Test User"},
	}

	for _, cfg := range configs {
		cmd := exec.Command("git", "config", cfg[0], cfg[1]) // nolint:gosec // Test helper with controlled input
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config %s: %v", cfg[0], err)
		}
	}

	skip := len(skipInitialCommit) > 0 && skipInitialCommit[0]
	if !skip {
		testFile := filepath.Join(repoPath, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), fs.FileGit); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to create initial commit: %v", err)
		}
	}

	t.Cleanup(func() {
		// Cleanup is automatic with t.TempDir() but we can add additional
		// cleanup if needed
	})

	return &TestRepo{
		t:    t,
		Dir:  dir,
		Path: repoPath,
	}
}
