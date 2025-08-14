package testutils

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// NewTestRepo creates a new temporary Git repository for testing.
// The repository is initialized and ready for Git operations.
// Cleanup is handled automatically when the test finishes.
func NewTestRepo(t *testing.T) (repoPath string) {
	t.Helper()

	repoPath = t.TempDir()

	// Initialize Git repository
	err := runGitCommand(repoPath, "init")
	require.NoError(t, err, "failed to initialize Git repository")

	// Configure test user for commits
	err = runGitCommand(repoPath, "config", "user.name", "Test User")
	require.NoError(t, err, "failed to configure Git user name")

	err = runGitCommand(repoPath, "config", "user.email", "test@example.com")
	require.NoError(t, err, "failed to configure Git user email")

	return repoPath
}

// NewTestRepoWithCommit creates a new Git repository with an initial commit.
// Returns the repository path and the hash of the initial commit.
// Useful for tests that require a repository with existing commit history.
func NewTestRepoWithCommit(t *testing.T) (repoPath, commitHash string) {
	t.Helper()

	repoPath = NewTestRepo(t)

	// Create initial file and commit
	testFile := filepath.Join(repoPath, "README.md")
	err := os.WriteFile(testFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err, "failed to create test file")

	err = runGitCommand(repoPath, "add", "README.md")
	require.NoError(t, err, "failed to stage test file")

	err = runGitCommand(repoPath, "commit", "-m", "Initial commit")
	require.NoError(t, err, "failed to create initial commit")

	// Get commit hash
	commitHash = getCommitHash(t, repoPath, "HEAD")

	return repoPath, commitHash
}

// GitState represents the expected state of a Git repository for testing.
type GitState struct {
	// CurrentBranch is the expected current branch name
	CurrentBranch string

	// Branches is the list of expected branch names
	Branches []string

	// Commits is the expected number of commits in the current branch
	Commits int

	// WorkingTreeClean indicates if the working tree should be clean
	WorkingTreeClean bool

	// Files is the list of files that should exist in the working directory
	Files []string
}

// runGitCommand executes a Git command in the specified directory.
// This is a simple helper for test repository setup.
func runGitCommand(workDir string, args ...string) error {
	// This is a simplified implementation for test setup
	// In production, this would use the GitCommander interface
	cmd := exec.Command("git", args...)
	cmd.Dir = workDir

	// Suppress output for clean test runs
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

// getCommitHash retrieves the commit hash for the specified ref.
func getCommitHash(t *testing.T, repoPath, ref string) string {
	t.Helper()

	cmd := exec.Command("git", "rev-parse", ref)
	cmd.Dir = repoPath

	output, err := cmd.Output()
	require.NoError(t, err, "failed to get commit hash")

	return strings.TrimSpace(string(output))
}
