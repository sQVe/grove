//go:build !integration
// +build !integration

package create

import (
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Error handling tests for worktree creation failures

func TestWorktreeCreatorImpl_CreateWorktree_GitCommandFailure(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return success (branch exists).
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")
	// Mock git command failure.
	expectedError := errors.New("fatal: '/path/to/worktree' already exists")
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), expectedError)

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)
	assert.Contains(t, groveErr.Message, "worktree existing-branch failed")
}

func TestWorktreeCreatorImpl_CreateWorktree_BranchAlreadyCheckedOut(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return success (branch exists).
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")
	// Mock specific error for branch already checked out.
	expectedError := errors.New("fatal: 'feature-branch' is already checked out at '/other/path'")
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), expectedError)

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)
	assert.Contains(t, groveErr.Message, "worktree existing-branch failed")
}

func TestWorktreeCreatorImpl_CreateWorktree_PathAlreadyExists(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return success (branch exists).
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")
	// Mock specific error for path already exists.
	expectedError := errors.New("fatal: '/path/to/worktree' already exists")
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), expectedError)

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)
	assert.Contains(t, groveErr.Message, "worktree existing-branch failed")
}

func TestWorktreeCreatorImpl_CreateWorktree_InvalidBranch(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return success (branch exists).
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/nonexistent-branch", "")
	// Mock specific error for invalid branch.
	expectedError := errors.New("fatal: invalid reference: nonexistent-branch")
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s nonexistent-branch", worktreePath), expectedError)

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: false,
	}

	err := creator.CreateWorktree("nonexistent-branch", worktreePath, options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)
	assert.Contains(t, groveErr.Message, "worktree existing-branch failed")
}
