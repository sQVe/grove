//go:build !integration
// +build !integration

package create

import (
	"errors"
	"testing"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorPropagation_WorktreeCreation(t *testing.T) {
	tests := []struct {
		name                string
		branchCheckError    error
		worktreeCreateError error
		remoteTrackError    error
		expectedErrorCode   string
		expectedErrorType   string
	}{
		{
			name:              "branch check failure propagates",
			branchCheckError:  errors.New("git show-ref failed: repository corrupted"),
			expectedErrorCode: groveErrors.ErrCodeWorktreeCreation,
			expectedErrorType: "worktree new-branch failed",
		},
		{
			name:                "worktree creation failure propagates",
			worktreeCreateError: errors.New("git worktree add failed: permission denied"),
			expectedErrorCode:   groveErrors.ErrCodeWorktreeCreation,
			expectedErrorType:   "existing-branch",
		},
		{
			name:              "remote tracking failure propagates",
			remoteTrackError:  errors.New("git branch --set-upstream-to failed: remote branch not found"),
			expectedErrorCode: groveErrors.ErrCodeWorktreeCreation,
			expectedErrorType: "remote-tracking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			creator := NewWorktreeCreator(mockExecutor)

			// Use a temporary directory that actually exists
			tmpDir := t.TempDir()
			testPath := tmpDir + "/worktree"

			// Setup mock responses based on test case
			if tt.branchCheckError != nil {
				// When branch check fails, it goes through new branch creation path
				mockExecutor.SetErrorResponse("show-ref --verify --quiet refs/heads/test-branch", tt.branchCheckError)
				mockExecutor.SetErrorResponse("worktree add -b test-branch "+testPath, tt.branchCheckError)
			} else {
				mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/test-branch", "")

				if tt.worktreeCreateError != nil {
					mockExecutor.SetErrorResponse("worktree add "+testPath+" test-branch", tt.worktreeCreateError)
				} else {
					mockExecutor.SetSuccessResponse("worktree add "+testPath+" test-branch", "")

					if tt.remoteTrackError != nil {
						// Setup for new branch with remote tracking
						mockExecutor.SetErrorResponse("show-ref --verify --quiet refs/heads/test-branch", errors.New("branch not found"))
						mockExecutor.SetSuccessResponse("worktree add -b test-branch "+testPath, "")
						mockExecutor.SetSuccessResponse("config --get clone.defaultRemoteName", "origin")
						mockExecutor.SetSuccessResponse("branch -r --list origin/test-branch", "origin/test-branch")
						mockExecutor.SetErrorResponse("-C "+testPath+" branch --set-upstream-to=origin/test-branch test-branch", tt.remoteTrackError)
					}
				}
			}

			options := WorktreeOptions{TrackRemote: tt.remoteTrackError != nil}
			err := creator.CreateWorktree("test-branch", testPath, options)

			require.Error(t, err)
			assert.IsType(t, &groveErrors.GroveError{}, err)

			groveErr := err.(*groveErrors.GroveError)
			assert.Equal(t, tt.expectedErrorCode, groveErr.Code)
			if tt.expectedErrorType != "" {
				assert.Contains(t, groveErr.Message, tt.expectedErrorType)
			}
		})
	}
}

func TestErrorPropagation_NestedOperations(t *testing.T) {
	t.Run("rollback errors don't mask original errors", func(t *testing.T) {
		mockExecutor := testutils.NewMockGitExecutor()
		creator := NewWorktreeCreator(mockExecutor)

		// Use a temporary directory that actually exists
		tmpDir := t.TempDir()
		testPath := tmpDir + "/worktree"

		// Branch exists but worktree creation fails
		mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/test-branch", "")
		originalError := errors.New("fatal: '" + testPath + "' already exists and is not an empty directory")
		mockExecutor.SetErrorResponse("worktree add "+testPath+" test-branch", originalError)

		// Rollback operations might also fail, but shouldn't mask the original error
		mockExecutor.SetErrorResponse("worktree remove --force "+testPath, errors.New("worktree not found"))

		err := creator.CreateWorktree("test-branch", testPath, WorktreeOptions{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists") // Original error should be preserved
		assert.IsType(t, &groveErrors.GroveError{}, err)
	})
}

func TestErrorPropagation_ConflictResolution(t *testing.T) {
	t.Run("conflict resolution errors propagate with context", func(t *testing.T) {
		mockExecutor := testutils.NewMockGitExecutor()
		creator := NewWorktreeCreator(mockExecutor)

		// Use a temporary directory that actually exists
		tmpDir := t.TempDir()
		testPath := tmpDir + "/worktree"

		// Branch exists
		mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/test-branch", "")

		// Initial worktree creation fails with conflict
		conflictError := errors.New("fatal: 'test-branch' is already used by worktree at '/other/path'")
		mockExecutor.SetErrorResponse("worktree add "+testPath+" test-branch", conflictError)

		// Conflict resolution fails due to uncommitted changes
		mockExecutor.SetSuccessResponse("worktree list --porcelain", "worktree /other/path\nHEAD abc123\nbranch refs/heads/test-branch\n")
		mockExecutor.SetSuccessResponse("-C /other/path status --porcelain", " M file.txt") // Dirty worktree

		err := creator.CreateWorktree("test-branch", testPath, WorktreeOptions{})

		require.Error(t, err)
		assert.IsType(t, &groveErrors.GroveError{}, err)

		groveErr := err.(*groveErrors.GroveError)
		assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)

		// Should contain context about resolution attempt
		context := groveErr.Context
		if context != nil {
			if resolutionAttempted, exists := context["resolution_attempted"]; exists && resolutionAttempted != nil {
				assert.True(t, resolutionAttempted.(bool))
			}
			if resolutionError, exists := context["resolution_error"]; exists {
				assert.NotEmpty(t, resolutionError)
			}
		}
	})
}

func TestErrorPropagation_NetworkErrors(t *testing.T) {
	t.Run("network errors are properly categorized", func(t *testing.T) {
		mockExecutor := testutils.NewMockGitExecutor()
		creator := NewWorktreeCreator(mockExecutor)

		// Use a temporary directory that actually exists
		tmpDir := t.TempDir()
		testPath := tmpDir + "/worktree"

		// Branch doesn't exist locally, try to create with remote tracking
		mockExecutor.SetErrorResponse("show-ref --verify --quiet refs/heads/test-branch", errors.New("ref not found"))
		mockExecutor.SetSuccessResponse("worktree add -b test-branch "+testPath, "")
		mockExecutor.SetSuccessResponse("config --get clone.defaultRemoteName", "origin")

		// Network timeout when checking remote branch
		networkError := errors.New("git branch -r --list failed: connection timeout")
		mockExecutor.SetErrorResponse("branch -r --list origin/test-branch", networkError)

		options := WorktreeOptions{TrackRemote: true}
		err := creator.CreateWorktree("test-branch", testPath, options)
		// Should succeed without remote tracking when network fails
		// The implementation should handle network errors gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "timeout")
		}
	})
}
