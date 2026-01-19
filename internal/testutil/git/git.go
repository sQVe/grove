package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/testutil"
)

// TestRepo provides a test git repository with proper configuration
type TestRepo struct {
	t    *testing.T
	Dir  string
	Path string
}

// NewTestRepo creates a new test repository with git config set up.
// Pass an optional branch name (default "main").
func NewTestRepo(t *testing.T, branchName ...string) *TestRepo {
	t.Helper()

	dir := testutil.TempDir(t)
	repoPath := filepath.Join(dir, "repo")

	if err := os.MkdirAll(repoPath, fs.DirGit); err != nil {
		t.Fatalf("Failed to create repo dir: %v", err)
	}

	branch := "main"
	if len(branchName) > 0 && branchName[0] != "" {
		branch = branchName[0]
	}

	cmd := exec.Command("git", "init", "-b", branch) // nolint:gosec
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

	// Always create initial commit
	{
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

// AddRemote adds a remote to the repository
func (r *TestRepo) AddRemote(name, url string) {
	r.t.Helper()
	cmd := exec.Command("git", "remote", "add", name, url) // nolint:gosec
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to add remote: %v", err)
	}
}

// SetSymbolicRef sets a symbolic reference
func (r *TestRepo) SetSymbolicRef(name, target string) {
	r.t.Helper()
	cmd := exec.Command("git", "symbolic-ref", name, target) // nolint:gosec
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to set symbolic ref: %v", err)
	}
}

// CreateBranch creates a new branch at the current HEAD
func (r *TestRepo) CreateBranch(name string) {
	r.t.Helper()
	cmd := exec.Command("git", "branch", name) // nolint:gosec
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to create branch: %v", err)
	}
}

// Checkout switches to a branch
func (r *TestRepo) Checkout(name string) {
	r.t.Helper()
	cmd := exec.Command("git", "checkout", name) // nolint:gosec
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to checkout: %v", err)
	}
}

// WriteFile writes content to a file in the repository
func (r *TestRepo) WriteFile(name, content string) {
	r.t.Helper()
	path := filepath.Join(r.Path, name)
	if err := os.WriteFile(path, []byte(content), fs.FileGit); err != nil {
		r.t.Fatalf("Failed to write file: %v", err)
	}
}

// Add stages a file
func (r *TestRepo) Add(name string) {
	r.t.Helper()
	cmd := exec.Command("git", "add", name) // nolint:gosec
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to add file: %v", err)
	}
}

// Commit creates a commit with the given message
func (r *TestRepo) Commit(message string) {
	r.t.Helper()
	cmd := exec.Command("git", "commit", "-m", message) // nolint:gosec
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to commit: %v", err)
	}
}

// Merge merges a branch into the current branch
func (r *TestRepo) Merge(branch string) {
	r.t.Helper()
	cmd := exec.Command("git", "merge", branch, "--no-edit") // nolint:gosec
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to merge: %v", err)
	}
}

// SquashMerge performs a squash merge of a branch into the current branch
func (r *TestRepo) SquashMerge(branch string) {
	r.t.Helper()
	cmd := exec.Command("git", "merge", "--squash", branch) // nolint:gosec
	cmd.Dir = r.Path
	if err := cmd.Run(); err != nil {
		r.t.Fatalf("Failed to squash merge: %v", err)
	}
	// Squash merge requires a separate commit
	commitCmd := exec.Command("git", "commit", "-m", "Squash merge "+branch) // nolint:gosec
	commitCmd.Dir = r.Path
	if err := commitCmd.Run(); err != nil {
		r.t.Fatalf("Failed to commit squash merge: %v", err)
	}
}

// CleanupWorktree registers cleanup for a worktree path to release Windows file locks.
// Call this after creating a worktree with git worktree add.
func CleanupWorktree(t *testing.T, bareDir, worktreePath string) {
	t.Helper()
	t.Cleanup(func() {
		cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath) // nolint:gosec
		cmd.Dir = bareDir
		_ = cmd.Run()
	})
}
