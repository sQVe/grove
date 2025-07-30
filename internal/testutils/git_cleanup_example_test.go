package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Example showing OLD problematic pattern vs NEW robust pattern

func TestGitCleanup_OldPattern_Problematic(t *testing.T) {
	// ❌ OLD PATTERN - PROBLEMATIC
	// This is what we DON'T want to do:

	tempDir := t.TempDir()

	// Create repository
	bareDir := filepath.Join(tempDir, ".bare")
	err := git.InitBare(bareDir)
	require.NoError(t, err)

	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	// Create worktrees without tracking for cleanup
	// Use unique names to avoid conflicts in demo (but this proves the point!)
	uniqueID := fmt.Sprintf("%d", time.Now().UnixNano())
	worktree1 := filepath.Join(tempDir, "feature-branch")
	_, err = git.ExecuteGit("worktree", "add", "-b", "feature-branch-"+uniqueID, worktree1)
	require.NoError(t, err)

	worktree2 := filepath.Join(tempDir, "another-branch")
	_, err = git.ExecuteGit("worktree", "add", "-b", "another-branch-"+uniqueID, worktree2)
	require.NoError(t, err)

	// ❌ PROBLEM: Manual cleanup is error-prone and often forgotten
	// ❌ PROBLEM: If test fails before cleanup, worktrees are left behind
	// ❌ PROBLEM: Branch names might conflict with other tests
	// ❌ PROBLEM: No automatic recovery from cleanup failures

	// Some test logic here...
	assert.DirExists(t, worktree1)
	assert.DirExists(t, worktree2)

	// Manual cleanup (often fails or is forgotten)
	_, err = git.ExecuteGit("worktree", "remove", worktree1)
	if err != nil {
		// If this fails, worktree is left behind!
		t.Logf("Failed to remove worktree: %v", err)
	}

	_, err = git.ExecuteGit("worktree", "remove", worktree2)
	if err != nil {
		// If this fails, worktree is left behind!
		t.Logf("Failed to remove worktree: %v", err)
	}
}

func TestGitCleanup_NewPattern_Robust(t *testing.T) {
	// ✅ NEW PATTERN - ROBUST
	// This is the recommended approach:

	gitHelper := NewGitTestHelper(t)

	// Create repository with automatic cleanup
	repoDir := gitHelper.CreateTestRepository()
	gitHelper.SetupInitialCommit()

	// Create worktrees with automatic tracking and cleanup
	worktree1 := gitHelper.CreateWorktree("feature-branch")
	worktree2 := gitHelper.CreateWorktree("another-branch")

	// ✅ BENEFITS:
	// ✅ Automatic cleanup happens via t.Cleanup() - GUARANTEED
	// ✅ Unique branch names prevent conflicts between tests
	// ✅ Force cleanup handles edge cases and failures
	// ✅ Comprehensive cleanup includes branches AND worktrees
	// ✅ No manual cleanup code needed

	// Test logic (cleanup happens automatically even if test fails)
	assert.DirExists(t, worktree1)
	assert.DirExists(t, worktree2)
	assert.DirExists(t, repoDir)

	// Create more worktrees dynamically
	dynamicWorktree := gitHelper.CreateWorktreeFromBranch("hotfix", "main")
	assert.DirExists(t, dynamicWorktree)

	// All cleanup happens automatically via t.Cleanup()!
}

func TestGitCleanup_IntegrationHelper_WithGitSupport(t *testing.T) {
	// ✅ ENHANCED INTEGRATION HELPER
	// For existing integration tests, add git cleanup support:

	helper := NewIntegrationTestHelper(t).
		WithCleanFilesystem().
		WithGitCleanup() // ← Add this line to existing tests!

	tempDir := helper.CreateTempDir("git-test")

	// Create repository
	bareDir := filepath.Join(tempDir, ".bare")
	err := git.InitBare(bareDir)
	require.NoError(t, err)

	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	// Create worktrees - they'll be cleaned up automatically
	// Use unique branch name to avoid conflicts
	uniqueBranch := fmt.Sprintf("test-branch-%d", time.Now().UnixNano())
	worktreePath := filepath.Join(tempDir, "test-worktree")
	_, err = git.ExecuteGit("worktree", "add", "-b", uniqueBranch, worktreePath)
	require.NoError(t, err)

	// Test logic...
	assert.DirExists(t, worktreePath)

	// WithGitCleanup() ensures automatic cleanup of ALL git artifacts
}

func TestGitCleanup_MigrationPattern(t *testing.T) {
	// 🔄 MIGRATION PATTERN
	// How to migrate existing problematic tests:

	// STEP 1: Replace manual repo setup with GitTestHelper
	gitHelper := NewGitTestHelper(t)
	_ = gitHelper.CreateTestRepository()
	gitHelper.SetupInitialCommit()

	// STEP 2: Replace manual worktree creation with helper methods
	mainWorktree := gitHelper.CreateWorktree("main")
	featureWorktree := gitHelper.CreateWorktreeFromBranch("feature", "main")

	// STEP 3: Remove all manual cleanup code - it's handled automatically

	// Your existing test logic remains the same
	err := os.Chdir(mainWorktree)
	require.NoError(t, err)

	// Create test files
	testFile := filepath.Join(mainWorktree, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "test.txt")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Add test file")
	require.NoError(t, err)

	// Verify the feature worktree exists and is separate
	assert.DirExists(t, featureWorktree)
	assert.NotEqual(t, mainWorktree, featureWorktree)

	// All cleanup (worktrees, branches, temp dirs) happens automatically!
}

// Example of generating unique branch names for parallel test safety
func TestUniqueNaming(t *testing.T) {
	gitHelper := NewGitTestHelper(t)

	// Each test gets unique prefixes - no conflicts!
	branch1 := gitHelper.GenerateUniqueBranchName("feature")
	branch2 := gitHelper.GenerateUniqueBranchName("bugfix")
	branch3 := gitHelper.GenerateUniqueBranchName("hotfix/critical")

	// Names will be like: test-a1b2c3d4-1642123456-feature
	t.Logf("Unique branches: %s, %s, %s", branch1, branch2, branch3)

	// These are guaranteed unique even if tests run in parallel
	assert.NotEqual(t, branch1, branch2)
	assert.NotEqual(t, branch2, branch3)
	assert.Contains(t, branch1, gitHelper.GetTestID())
}
