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

const testExistingWorktreePath = "/existing/worktree"

// Note: Basic worktree creation tests moved to worktree_creator_basic_test.go
// Error handling tests moved to worktree_creator_errors_test.go
// Validation tests moved to worktree_creator_validation_test.go

// Conflict Resolution Tests
//
// These tests verify automatic worktree conflict resolution when a branch
// is already checked out in another worktree. The resolution process:
// 1. Detects the conflict during worktree creation
// 2. Locates the conflicting worktree
// 3. Checks if it's safe to resolve (no uncommitted changes)
// 4. Switches the conflicting worktree to detached HEAD
// 5. Retries the original worktree creation

func TestWorktreeCreatorImpl_ConflictResolution_CleanWorktreeSuccess(t *testing.T) {
	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Create a sequential mock that simulates the conflict resolution workflow:
	// 1st call: fails with "branch already used" error
	// 2nd call: succeeds after conflict resolution
	mockExecutor := testutils.NewSequentialMockGitExecutor()

	// Set up sequential responses for worktree add command
	mockExecutor.SetSequentialResponse(
		fmt.Sprintf("worktree add %s feature-branch", worktreePath),
		[]testutils.MockResponse{
			// First call fails with conflict - simulates the initial error
			{Output: "", Error: fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", testExistingWorktreePath)},
			// Second call succeeds after resolution - simulates successful retry
			{Output: "", Error: nil},
		},
	)

	// Set up responses for conflict resolution workflow
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

	// Verify progress messages were provided
	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Resolved conflict: switched previous worktree to detached HEAD")

	// Verify the worktree add command was called twice (first fails, then succeeds)
	assert.Equal(t, 2, mockExecutor.GetSpecificCounter("worktree_add"))
}

func TestWorktreeCreatorImpl_ConflictResolution_DirtyWorktreeFails(t *testing.T) {
	// This test verifies that automatic conflict resolution fails when the
	// conflicting worktree has uncommitted changes, preventing data loss
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := testExistingWorktreePath

	// Mock branch exists check
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	// Mock worktree creation failure with conflict error
	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), conflictError)

	// Mock worktree list to show conflicting worktree is not main
	mockExecutor.SetSuccessResponse("worktree list --porcelain", "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree "+conflictingWorktreePath+"\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n")

	// Mock status check showing dirty worktree - this prevents automatic resolution
	// Format: " M" = modified, "??" = untracked (git status --porcelain format)
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

	// Verify progress messages were provided
	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Cannot resolve automatically: conflicting worktree has uncommitted changes")

	// Verify status check was performed but checkout was not attempted
	assert.True(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "status", "--porcelain"))
	assert.False(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "checkout", "--detach"))
}

func TestWorktreeCreatorImpl_ConflictResolution_StatusCheckFails(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := testExistingWorktreePath

	// Mock branch exists check
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	// Mock worktree creation failure with conflict error
	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), conflictError)

	// Mock worktree list to show conflicting worktree is not main
	mockExecutor.SetSuccessResponse("worktree list --porcelain", "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree "+conflictingWorktreePath+"\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n")

	// Mock status check failure (e.g., worktree doesn't exist anymore)
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

	// Verify progress messages were provided
	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Automatic conflict resolution failed")

	// Verify status check was performed but checkout was not attempted
	assert.True(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "status", "--porcelain"))
	assert.False(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "checkout", "--detach"))
}

func TestWorktreeCreatorImpl_ConflictResolution_CheckoutFails(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := testExistingWorktreePath

	// Mock branch exists check
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	// Mock worktree creation failure with conflict error
	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), conflictError)

	// Mock worktree list to show conflicting worktree is not main
	mockExecutor.SetSuccessResponse("worktree list --porcelain", "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree "+conflictingWorktreePath+"\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n")

	// Mock status check showing clean worktree
	mockExecutor.SetSuccessResponse(fmt.Sprintf("-C %s status --porcelain", conflictingWorktreePath), "")

	// Mock checkout --detach failure
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

	// Verify progress messages were provided
	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Automatic conflict resolution failed")

	// Verify both status check and checkout were attempted
	assert.True(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "status", "--porcelain"))
	assert.True(t, mockExecutor.HasCommand("-C", conflictingWorktreePath, "checkout", "--detach"))
}

func TestWorktreeCreatorImpl_ConflictResolution_BasicConflictError(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := testExistingWorktreePath

	// Mock branch exists check
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	// Mock worktree creation failure with conflict error - without resolution
	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse("worktree add "+worktreePath+" feature-branch", conflictError)

	// Don't mock the status check - this should cause the resolution to fail

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, nil)

	// Should return an error since we didn't mock the status check
	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)
}

func TestWorktreeCreatorImpl_ConflictResolution_NoProgressCallback(t *testing.T) {
	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Create a sequential mock that can handle different responses for the same command
	mockExecutor := testutils.NewSequentialMockGitExecutor()

	// Set up sequential responses for worktree add command
	mockExecutor.SetSequentialResponse(
		fmt.Sprintf("worktree add %s feature-branch", worktreePath),
		[]testutils.MockResponse{
			// First call fails with conflict
			{Output: "", Error: fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", testExistingWorktreePath)},
			// Second call succeeds after resolution
			{Output: "", Error: nil},
		},
	)

	// Set up single responses for other commands
	mockExecutor.SetSingleResponse("show-ref --verify --quiet refs/heads/feature-branch", testutils.MockResponse{Output: "", Error: nil})
	mockExecutor.SetSingleResponse("worktree list --porcelain", testutils.MockResponse{Output: "worktree /main/repo\nHEAD abcd1234\nbranch refs/heads/main\n\nworktree " + testExistingWorktreePath + "\nHEAD efgh5678\nbranch refs/heads/feature-branch\n\n", Error: nil})
	mockExecutor.SetSingleResponse("-C "+testExistingWorktreePath+" status --porcelain", testutils.MockResponse{Output: "", Error: nil})
	mockExecutor.SetSingleResponse("-C "+testExistingWorktreePath+" checkout --detach", testutils.MockResponse{Output: "Switched to detached HEAD", Error: nil})

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, nil) // No progress callback

	require.NoError(t, err)

	// Verify the worktree add command was called twice (first fails, then succeeds)
	assert.Equal(t, 2, mockExecutor.GetSpecificCounter("worktree_add"))
}
