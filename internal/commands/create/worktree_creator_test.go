//go:build !integration
// +build !integration

package create

import (
	"fmt"
	"path/filepath"
	"testing"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Note: Basic worktree creation tests moved to worktree_creator_basic_test.go.
// Error handling tests moved to worktree_creator_errors_test.go.
// Validation tests moved to worktree_creator_validation_test.go.

func getTestExistingWorktreePath(helper *testutils.UnitTestHelper) string {
	return helper.GetUniqueTestPath("existing-worktree")
}

// Conflict Resolution Tests
//
// These tests verify automatic worktree conflict resolution when a branch
// is already checked out in another worktree. The resolution process:
// 1. Detects the conflict during worktree creation
// 2. Locates the conflicting worktree
// 3. Checks if it's safe to resolve (no uncommitted changes)
// 4. Switches the conflicting worktree to detached HEAD
// 5. Retries the original worktree creation.

func TestWorktreeCreatorImpl_ConflictResolution_CleanWorktreeSuccess(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	tmpDir := helper.CreateTempDir("conflict-resolution-test")
	worktreePath := filepath.Join(tmpDir, "worktree")

	testExistingWorktreePath := getTestExistingWorktreePath(helper)

	// TESTING STRATEGY: Simulate automatic conflict resolution for branch checkout conflicts.
	// This test verifies the complete workflow when a branch is already checked out elsewhere:
	// 1. Initial worktree creation fails with "branch already used" error
	// 2. System detects conflict and locates the conflicting worktree 
	// 3. System checks if conflicting worktree is clean (no uncommitted changes)
	// 4. System detaches the conflicting worktree from the branch (checkout --detach)
	// 5. System retries the original worktree creation and succeeds
	// 
	// Mock Setup: Uses SequentialMockGitExecutor to simulate the retry behavior
	// where the same command returns different results on subsequent calls.
	mockExecutor := testutils.NewSequentialMockGitExecutor()

	mockExecutor.SetSequentialResponse(
		fmt.Sprintf("worktree add %s feature-branch", worktreePath),
		[]testutils.MockResponse{
			{Output: "", Error: fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", testExistingWorktreePath)},
			{Output: "", Error: nil},
		},
	)

	mockExecutor.SetSingleResponse("show-ref --verify --quiet refs/heads/feature-branch", testutils.MockResponse{Output: "", Error: nil})                                                                                                                                   // Branch exists check
	mockExecutor.SetSingleResponse("worktree list --porcelain", testutils.MockResponse{Output: "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree " + testExistingWorktreePath + "\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n", Error: nil}) // List existing worktrees
	mockExecutor.SetSingleResponse("-C "+testExistingWorktreePath+" status --porcelain", testutils.MockResponse{Output: "", Error: nil})                                                                                                                                    // Check if conflicting worktree is clean
	mockExecutor.SetSingleResponse("-C "+testExistingWorktreePath+" checkout --detach", testutils.MockResponse{Output: "Switched to detached HEAD", Error: nil})                                                                                                            // Detach conflicting worktree

	creator := NewWorktreeCreator(mockExecutor)

	var progressMessages []string
	progressCallback := func(message string) {
		progressMessages = append(progressMessages, message)
	}

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, progressCallback)

	require.NoError(t, err)

	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Resolved conflict: switched previous worktree to detached HEAD")

	assert.Equal(t, 2, mockExecutor.GetSpecificCounter("worktree_add"))
}

func TestWorktreeCreatorImpl_ConflictResolution_DirtyWorktreeFails(t *testing.T) {
	// TESTING STRATEGY: Verify conflict resolution safety mechanisms.
	// This test ensures automatic conflict resolution FAILS when the conflicting 
	// worktree has uncommitted changes, preventing potential data loss.
	//
	// Safety Behavior: System detects uncommitted changes via 'git status --porcelain'
	// and aborts the automatic resolution, returning a clear error message to the user.
	// This prevents accidentally losing work-in-progress in the conflicting worktree.
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := helper.CreateTempDir("conflict-resolution-test")
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := getTestExistingWorktreePath(helper)

	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), conflictError)

	mockExecutor.SetSuccessResponse("worktree list --porcelain", "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree "+conflictingWorktreePath+"\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n")

	// Mock status check showing dirty worktree prevents automatic resolution.
	// Format: " M" = modified, "??" = untracked (git status --porcelain format).
	mockExecutor.SetSuccessResponse(fmt.Sprintf("-C %s status --porcelain", conflictingWorktreePath), " M modified_file.txt\n?? untracked_file.txt")

	creator := NewWorktreeCreator(mockExecutor)

	var progressMessages []string
	progressCallback := func(message string) {
		progressMessages = append(progressMessages, message)
	}

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, progressCallback)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)

	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Cannot resolve automatically: conflicting worktree has uncommitted changes")

	assert.True(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "status", "--porcelain"))
	assert.False(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "checkout", "--detach"))
}

func TestWorktreeCreatorImpl_ConflictResolution_StatusCheckFails(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := helper.CreateTempDir("conflict-resolution-test")
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := getTestExistingWorktreePath(helper)

	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), conflictError)

	mockExecutor.SetSuccessResponse("worktree list --porcelain", "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree "+conflictingWorktreePath+"\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n")

	// Mock status check failure (worktree no longer exists).
	mockExecutor.SetErrorResponse(fmt.Sprintf("-C %s status --porcelain", conflictingWorktreePath), fmt.Errorf("fatal: not a git repository"))

	creator := NewWorktreeCreator(mockExecutor)

	var progressMessages []string
	progressCallback := func(message string) {
		progressMessages = append(progressMessages, message)
	}

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, progressCallback)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)

	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Automatic conflict resolution failed")

	assert.True(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "status", "--porcelain"))
	assert.False(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "checkout", "--detach"))
}

func TestWorktreeCreatorImpl_ConflictResolution_CheckoutFails(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := helper.CreateTempDir("conflict-resolution-test")
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := getTestExistingWorktreePath(helper)

	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), conflictError)

	mockExecutor.SetSuccessResponse("worktree list --porcelain", "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree "+conflictingWorktreePath+"\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n")

	mockExecutor.SetSuccessResponse(fmt.Sprintf("-C %s status --porcelain", conflictingWorktreePath), "")

	mockExecutor.SetErrorResponse(fmt.Sprintf("-C %s checkout --detach", conflictingWorktreePath), fmt.Errorf("error: pathspec 'HEAD' did not match any file(s) known to git"))

	creator := NewWorktreeCreator(mockExecutor)

	var progressMessages []string
	progressCallback := func(message string) {
		progressMessages = append(progressMessages, message)
	}

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, progressCallback)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)

	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Automatic conflict resolution failed")

	assert.True(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "status", "--porcelain"))
	assert.True(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "checkout", "--detach"))
}

func TestWorktreeCreatorImpl_ConflictResolution_BasicConflictError(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := helper.CreateTempDir("conflict-resolution-test")
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := getTestExistingWorktreePath(helper)

	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse("worktree add "+worktreePath+" feature-branch", conflictError)

	// Don't mock the status check to cause resolution failure.

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, nil)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)
}

func TestWorktreeCreatorImpl_ConflictResolution_NoProgressCallback(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	tmpDir := helper.CreateTempDir("conflict-resolution-test")
	worktreePath := filepath.Join(tmpDir, "worktree")

	testExistingWorktreePath := getTestExistingWorktreePath(helper)

	mockExecutor := testutils.NewSequentialMockGitExecutor()

	mockExecutor.SetSequentialResponse(
		fmt.Sprintf("worktree add %s feature-branch", worktreePath),
		[]testutils.MockResponse{
			{Output: "", Error: fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", testExistingWorktreePath)},
			{Output: "", Error: nil},
		},
	)

	mockExecutor.SetSingleResponse("show-ref --verify --quiet refs/heads/feature-branch", testutils.MockResponse{Output: "", Error: nil})
	mockExecutor.SetSingleResponse("worktree list --porcelain", testutils.MockResponse{Output: "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree " + testExistingWorktreePath + "\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n", Error: nil})
	mockExecutor.SetSingleResponse("-C "+testExistingWorktreePath+" status --porcelain", testutils.MockResponse{Output: "", Error: nil})
	mockExecutor.SetSingleResponse("-C "+testExistingWorktreePath+" checkout --detach", testutils.MockResponse{Output: "Switched to detached HEAD", Error: nil})

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, nil) // No progress callback

	require.NoError(t, err)

	assert.Equal(t, 2, mockExecutor.GetSpecificCounter("worktree_add"))
}
