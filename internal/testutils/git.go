package testutils

import (
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	// Common file permissions used in testing
	dirPermissions  = 0o755
	filePermissions = 0o644
)

// GitTestHelper provides robust git testing with automatic cleanup.
// It creates isolated test repositories and manages worktrees with unique naming
// to prevent test interference and ensure proper cleanup.
type GitTestHelper struct {
	t           *testing.T
	testID      string
	repoDir     string
	worktrees   []string
	branches    []string
	tempDirs    []string
	originalDir string
}

// NewGitTestHelper creates a new git test helper with automatic cleanup.
// The helper generates unique test identifiers to prevent interference between parallel tests.
func NewGitTestHelper(t *testing.T) *GitTestHelper {
	t.Helper()

	testID := generateUniqueTestID(t)

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	helper := &GitTestHelper{
		t:           t,
		testID:      testID,
		originalDir: originalDir,
		worktrees:   make([]string, 0),
		branches:    make([]string, 0),
		tempDirs:    make([]string, 0),
	}

	// Automatic cleanup on test completion
	t.Cleanup(func() {
		helper.cleanupAll()
	})

	return helper
}

// CreateTestRepository creates an isolated test repository with unique naming.
// It initializes a bare repository and creates the necessary git file structure.
func (g *GitTestHelper) CreateTestRepository() string {
	g.t.Helper()

	repoDir := filepath.Join(os.TempDir(), g.testID+"-repo")
	err := os.MkdirAll(repoDir, dirPermissions)
	require.NoError(g.t, err, "Failed to create test repository directory")

	// Create bare repository
	bareDir := filepath.Join(repoDir, ".bare")
	err = g.initBare(bareDir)
	require.NoError(g.t, err, "Failed to initialize bare repository")

	// Create .git file pointing to bare repo
	err = g.createGitFile(repoDir, bareDir)
	require.NoError(g.t, err, "Failed to create git file")

	g.repoDir = repoDir
	g.tempDirs = append(g.tempDirs, repoDir)

	return repoDir
}

// CreateWorktree creates a worktree with automatic cleanup tracking.
// It generates a unique branch name to prevent conflicts between parallel tests.
func (g *GitTestHelper) CreateWorktree(branchName string) string {
	g.t.Helper()

	if g.repoDir == "" {
		require.Fail(g.t, "Must call CreateTestRepository first")
	}

	uniqueBranch := g.testID + "-" + strings.ReplaceAll(branchName, "/", "-")
	worktreePath := filepath.Join(g.repoDir, uniqueBranch)

	_, err := g.executeGit("worktree", "add", "-b", uniqueBranch, worktreePath)
	require.NoError(g.t, err, "Failed to create worktree %s", worktreePath)

	// Track for cleanup
	g.worktrees = append(g.worktrees, worktreePath)
	g.branches = append(g.branches, uniqueBranch)

	return worktreePath
}

// CreateWorktreeFromBranch creates a worktree from an existing branch.
// This is useful for creating feature branches based on a main branch.
func (g *GitTestHelper) CreateWorktreeFromBranch(branchName, baseBranch string) string {
	g.t.Helper()

	if g.repoDir == "" {
		require.Fail(g.t, "Must call CreateTestRepository first")
	}

	uniqueBranch := g.testID + "-" + strings.ReplaceAll(branchName, "/", "-")
	worktreePath := filepath.Join(g.repoDir, uniqueBranch)

	// Create worktree from base branch
	_, err := g.executeGit("worktree", "add", "-b", uniqueBranch, worktreePath, baseBranch)
	require.NoError(g.t, err, "Failed to create worktree %s from %s", worktreePath, baseBranch)

	// Track for cleanup
	g.worktrees = append(g.worktrees, worktreePath)
	g.branches = append(g.branches, uniqueBranch)

	return worktreePath
}

// GetRepoDir returns the test repository directory
func (g *GitTestHelper) GetRepoDir() string {
	return g.repoDir
}

// GetTestID returns the unique test identifier
func (g *GitTestHelper) GetTestID() string {
	return g.testID
}

// GenerateUniqueBranchName creates a unique branch name for the test
func (g *GitTestHelper) GenerateUniqueBranchName(baseName string) string {
	return g.testID + "-" + strings.ReplaceAll(baseName, "/", "-")
}

// SetupInitialCommit creates an initial commit in the repository.
// This establishes a main branch with basic content for testing purposes.
func (g *GitTestHelper) SetupInitialCommit() {
	g.t.Helper()

	if g.repoDir == "" {
		require.Fail(g.t, "Must call CreateTestRepository first")
	}

	// Create a temporary working directory for the initial commit
	tempWorkDir := filepath.Join(g.repoDir, "temp-work")
	err := os.MkdirAll(tempWorkDir, dirPermissions)
	require.NoError(g.t, err, "Failed to create temp working directory")

	// Change to temp working directory
	err = os.Chdir(tempWorkDir)
	require.NoError(g.t, err, "Failed to change to temp working directory")

	// Initialize working tree
	_, err = g.executeGit("init")
	require.NoError(g.t, err, "Failed to initialize working tree")

	// Configure git user for commits
	_, err = g.executeGit("config", "user.name", "Test User")
	require.NoError(g.t, err, "Failed to configure git user name")
	_, err = g.executeGit("config", "user.email", "test@example.com")
	require.NoError(g.t, err, "Failed to configure git user email")

	// Create initial commit
	readmeFile := filepath.Join(tempWorkDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), filePermissions)
	require.NoError(g.t, err, "Failed to create README file")

	_, err = g.executeGit("add", "README.md")
	require.NoError(g.t, err, "Failed to add README to git")

	_, err = g.executeGit("commit", "-m", "Initial commit")
	require.NoError(g.t, err, "Failed to create initial commit")

	// Push to bare repository
	bareDir := filepath.Join(g.repoDir, ".bare")
	_, err = g.executeGit("remote", "add", "origin", bareDir)
	require.NoError(g.t, err, "Failed to add remote origin")

	_, err = g.executeGit("push", "origin", "main")
	require.NoError(g.t, err, "Failed to push to origin")

	// Return to original directory
	err = os.Chdir(g.originalDir)
	require.NoError(g.t, err, "Failed to return to original directory")

	// Clean up temp working directory
	err = os.RemoveAll(tempWorkDir)
	require.NoError(g.t, err, "Failed to remove temp working directory")
}

// cleanupAll performs comprehensive cleanup of all git artifacts
func (g *GitTestHelper) cleanupAll() {
	// Return to original directory first
	if g.originalDir != "" {
		_ = os.Chdir(g.originalDir)
	}

	// Force remove all worktrees (ignore errors - they might already be gone)
	for _, worktree := range g.worktrees {
		// Try to remove worktree gracefully first
		_, _ = g.executeGit("worktree", "remove", worktree)
		// If that fails, force remove
		_, _ = g.executeGit("worktree", "remove", "--force", worktree)
		// As last resort, remove the directory
		_ = os.RemoveAll(worktree)
	}

	// Delete all branches (ignore errors)
	for _, branch := range g.branches {
		_, _ = g.executeGit("branch", "-D", branch)
	}

	// Remove all temp directories
	for _, dir := range g.tempDirs {
		_ = os.RemoveAll(dir)
	}

	// Clean up any remaining artifacts with our test ID
	g.cleanupByTestID()
}

// cleanupByTestID removes any remaining artifacts with our test ID
func (g *GitTestHelper) cleanupByTestID() {
	// Clean up temp directories that might have our test ID
	patterns := []string{
		"/tmp/" + g.testID + "*",
		"/tmp/*" + g.testID + "*",
		os.TempDir() + "/" + g.testID + "*",
		os.TempDir() + "/*" + g.testID + "*",
	}

	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, match := range matches {
			_ = os.RemoveAll(match) // Best effort cleanup
		}
	}
}

// executeGit executes a git command and returns the output
func (g *GitTestHelper) executeGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

// initBare initializes a bare git repository
func (g *GitTestHelper) initBare(bareDir string) error {
	cmd := exec.Command("git", "init", "--bare", bareDir)
	return cmd.Run()
}

// createGitFile creates a .git file pointing to the bare repository
func (g *GitTestHelper) createGitFile(repoDir, bareDir string) error {
	gitFile := filepath.Join(repoDir, ".git")
	content := fmt.Sprintf("gitdir: %s\n", filepath.Base(bareDir))
	return os.WriteFile(gitFile, []byte(content), filePermissions)
}

// generateUniqueTestID creates a unique identifier for the test
func generateUniqueTestID(t *testing.T) string {
	// Create a hash of the test name for uniqueness
	hash := sha256.Sum256([]byte(t.Name()))
	hashStr := fmt.Sprintf("%x", hash)[:8]

	// Add timestamp for additional uniqueness
	timestamp := time.Now().Unix()

	return fmt.Sprintf("test-%s-%d", hashStr, timestamp)
}
