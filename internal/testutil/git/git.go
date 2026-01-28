package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// WriteFile writes content to a file in the repository, creating parent dirs.
func (r *TestRepo) WriteFile(name, content string) {
	r.t.Helper()
	path := filepath.Join(r.Path, name)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, fs.DirGit); err != nil {
		r.t.Fatalf("Failed to create directory %s: %v", dir, err)
	}
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

// Run executes a git command and returns combined stdout/stderr and error.
func (r *TestRepo) Run(args ...string) (string, error) {
	r.t.Helper()
	cmd := exec.Command("git", args...) // nolint:gosec
	cmd.Dir = r.Path
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunOutput executes a git command, fails on error, returns combined output.
func (r *TestRepo) RunOutput(args ...string) string {
	r.t.Helper()
	out, err := r.Run(args...)
	if err != nil {
		r.t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
	return out
}

// MustFail executes a git command and fails the test if it succeeds.
func (r *TestRepo) MustFail(args ...string) {
	r.t.Helper()
	_, err := r.Run(args...)
	if err == nil {
		r.t.Fatalf("expected git %v to fail, but it succeeded", args)
	}
}

// AssertBranchExists fails if the branch doesn't exist.
func (r *TestRepo) AssertBranchExists(name string) {
	r.t.Helper()
	_, err := r.Run("rev-parse", "--verify", "refs/heads/"+name)
	if err != nil {
		r.t.Fatalf("expected branch %q to exist, but it doesn't", name)
	}
}

// AssertOnBranch fails if HEAD is not on the given branch.
func (r *TestRepo) AssertOnBranch(name string) {
	r.t.Helper()
	out := strings.TrimSpace(r.RunOutput("rev-parse", "--abbrev-ref", "HEAD"))
	if out != name {
		r.t.Fatalf("expected to be on branch %q, but on %q", name, out)
	}
}

// AssertClean fails if the working tree has uncommitted changes.
func (r *TestRepo) AssertClean() {
	r.t.Helper()
	out := r.RunOutput("status", "--porcelain")
	if out != "" {
		r.t.Fatalf("expected clean working tree, but got:\n%s", out)
	}
}

// BareTestRepo provides a bare test git repository.
type BareTestRepo struct {
	t    *testing.T
	Dir  string
	Path string
}

// NewBareTestRepo creates a bare test repository.
func NewBareTestRepo(t *testing.T) *BareTestRepo {
	t.Helper()

	dir := testutil.TempDir(t)
	bareDir := filepath.Join(dir, "repo.git")

	cmd := exec.Command("git", "init", "--bare", "-b", "main") // nolint:gosec
	if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
		t.Fatalf("Failed to create bare repo dir: %v", err)
	}
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	return &BareTestRepo{
		t:    t,
		Dir:  dir,
		Path: bareDir,
	}
}

// Run executes a git command and returns combined stdout/stderr and error.
func (r *BareTestRepo) Run(args ...string) (string, error) {
	r.t.Helper()
	cmd := exec.Command("git", args...) // nolint:gosec
	cmd.Dir = r.Path
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunOutput executes a git command, fails on error, returns combined output.
func (r *BareTestRepo) RunOutput(args ...string) string {
	r.t.Helper()
	out, err := r.Run(args...)
	if err != nil {
		r.t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
	return out
}

// MustFail executes a git command and fails the test if it succeeds.
func (r *BareTestRepo) MustFail(args ...string) {
	r.t.Helper()
	_, err := r.Run(args...)
	if err == nil {
		r.t.Fatalf("expected git %v to fail, but it succeeded", args)
	}
}

// GroveWorkspace represents a complete Grove workspace with bare repo and worktrees.
type GroveWorkspace struct {
	t         *testing.T
	Dir       string
	BareDir   string
	Worktrees map[string]string
}

// NewGroveWorkspace creates a Grove workspace with specified worktrees.
// First branch becomes the main worktree. Git config is set for commits.
func NewGroveWorkspace(t *testing.T, branches ...string) *GroveWorkspace {
	t.Helper()

	if len(branches) == 0 {
		branches = []string{"main"}
	}

	dir := testutil.TempDir(t)
	bareDir := filepath.Join(dir, ".bare")

	if err := os.MkdirAll(bareDir, fs.DirGit); err != nil {
		t.Fatalf("Failed to create bare dir: %v", err)
	}

	cmd := exec.Command("git", "init", "--bare", "-b", branches[0]) // nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init bare repo: %v", err)
	}

	configs := [][]string{
		{"commit.gpgsign", "false"},
		{"user.email", "test@example.com"},
		{"user.name", "Test User"},
	}

	for _, cfg := range configs {
		cmd := exec.Command("git", "config", cfg[0], cfg[1]) // nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to set git config %s: %v", cfg[0], err)
		}
	}

	w := &GroveWorkspace{
		t:         t,
		Dir:       dir,
		BareDir:   bareDir,
		Worktrees: make(map[string]string),
	}

	// Create first worktree and add initial commit
	firstPath := w.CreateWorktree(branches[0])
	testFile := filepath.Join(firstPath, ".gitkeep")
	if err := os.WriteFile(testFile, []byte(""), fs.FileGit); err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = firstPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage initial file: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "initial")
	cmd.Dir = firstPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create initial commit: %v", err)
	}

	// Create remaining worktrees
	for _, branch := range branches[1:] {
		w.CreateWorktree(branch)
	}

	return w
}

// CreateWorktree adds a worktree for the given branch.
// If the branch doesn't exist, creates it.
func (w *GroveWorkspace) CreateWorktree(branch string) string {
	w.t.Helper()

	worktreePath := filepath.Join(w.Dir, branch)

	var cmd *exec.Cmd
	if len(w.Worktrees) == 0 {
		// First worktree needs --orphan since bare repo has no commits
		cmd = exec.Command("git", "worktree", "add", "--orphan", "-b", branch, worktreePath) // nolint:gosec
	} else {
		cmd = exec.Command("git", "worktree", "add", "-b", branch, worktreePath) // nolint:gosec
	}
	cmd.Dir = w.BareDir
	if err := cmd.Run(); err != nil {
		w.t.Fatalf("Failed to create worktree for %s: %v", branch, err)
	}

	CleanupWorktree(w.t, w.BareDir, worktreePath)
	w.Worktrees[branch] = worktreePath
	return worktreePath
}

// WorktreePath returns the path for a worktree by branch name.
func (w *GroveWorkspace) WorktreePath(branch string) string {
	path, ok := w.Worktrees[branch]
	if !ok {
		w.t.Fatalf("worktree for branch %q not found", branch)
	}
	return path
}

// Run executes a git command in the bare repo and returns combined stdout/stderr and error.
func (w *GroveWorkspace) Run(args ...string) (string, error) {
	w.t.Helper()
	cmd := exec.Command("git", args...) // nolint:gosec
	cmd.Dir = w.BareDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// RunOutput executes a git command in the bare repo, fails on error, returns combined output.
func (w *GroveWorkspace) RunOutput(args ...string) string {
	w.t.Helper()
	out, err := w.Run(args...)
	if err != nil {
		w.t.Fatalf("git %v failed: %v\nOutput: %s", args, err, out)
	}
	return out
}

// MustFail executes a git command in the bare repo and fails the test if it succeeds.
func (w *GroveWorkspace) MustFail(args ...string) {
	w.t.Helper()
	_, err := w.Run(args...)
	if err == nil {
		w.t.Fatalf("expected git %v to fail, but it succeeded", args)
	}
}
