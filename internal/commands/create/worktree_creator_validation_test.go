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

// Validation tests for worktree creation input validation

func TestWorktreeCreatorImpl_CreateWorktree_ValidationErrors(t *testing.T) {
	tests := []struct {
		name           string
		branchName     string
		path           string
		expectGitCalls bool
	}{
		{
			name:           "empty branch name",
			branchName:     "",
			path:           "/path/to/worktree",
			expectGitCalls: false,
		},
		{
			name:           "empty path",
			branchName:     "feature-branch",
			path:           "",
			expectGitCalls: false,
		},
		{
			name:           "whitespace only branch name",
			branchName:     "   ",
			path:           "/path/to/worktree",
			expectGitCalls: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			creator := NewWorktreeCreator(mockExecutor)
			options := WorktreeOptions{}

			err := creator.CreateWorktree(tt.branchName, tt.path, options)

			require.Error(t, err)
			assert.IsType(t, &groveErrors.GroveError{}, err)
			groveErr := err.(*groveErrors.GroveError)
			assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)

			if !tt.expectGitCalls {
				assert.Equal(t, 0, mockExecutor.CallCount, "Should not make git calls for invalid input")
			}
		})
	}
}

func TestWorktreeCreatorImpl_CreateWorktree_ValidationSuccess(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "valid")

	// Mock branch exists check to return success (branch exists).
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/valid-branch", "")
	// Mock successful worktree creation.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("worktree add %s valid-branch", worktreePath), "")

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{}

	err := creator.CreateWorktree("valid-branch", worktreePath, options)

	require.NoError(t, err)
	assert.Equal(t, 2, mockExecutor.CallCount) // Branch check + worktree add.
	assert.True(t, mockExecutor.HasCommand("worktree", "add", worktreePath, "valid-branch"))
}
