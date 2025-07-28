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

func TestWorktreeCreatorImpl_CreateWorktree_ExistingBranchSuccess(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return success (branch exists).
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")
	// Mock successful worktree creation for existing branch.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), "")

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: false,
		Force:       false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.NoError(t, err)
	assert.True(t, mockExecutor.HasCommand("worktree", "add", worktreePath, "feature-branch"))
}

func TestWorktreeCreatorImpl_CreateWorktree_NewBranchSuccess(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return error (branch doesn't exist).
	mockExecutor.SetErrorResponse("show-ref --verify --quiet refs/heads/feature-branch", fmt.Errorf("ref not found"))
	// Mock successful worktree creation with new branch.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("worktree add -b feature-branch %s", worktreePath), "")

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: false,
		Force:       false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.NoError(t, err)
	assert.True(t, mockExecutor.HasCommand("worktree", "add", "-b", "feature-branch", worktreePath))
}

func TestWorktreeCreatorImpl_CreateWorktree_TrackRemoteSuccess(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return error (branch doesn't exist).
	mockExecutor.SetErrorResponse("show-ref --verify --quiet refs/heads/feature-branch", fmt.Errorf("ref not found"))
	// Mock successful worktree creation with new branch.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("worktree add -b feature-branch %s", worktreePath), "")
	// Mock getting default remote name.
	mockExecutor.SetSuccessResponse("config --get clone.defaultRemoteName", "origin")
	// Mock setting up remote tracking.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("-C %s branch --set-upstream-to=origin/feature-branch feature-branch", worktreePath), "")

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: true,
		Force:       false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.NoError(t, err)
	assert.True(t, mockExecutor.HasCommand("worktree", "add", "-b", "feature-branch", worktreePath))
	assert.True(t, mockExecutor.HasCommand("-C", worktreePath, "branch", "--set-upstream-to=origin/feature-branch", "feature-branch"))
}

func TestWorktreeCreatorImpl_CreateWorktree_ForceSuccess(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return success (branch exists).
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")
	// Mock successful worktree creation with existing branch.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), "")

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: false,
		Force:       true,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.NoError(t, err)
	assert.True(t, mockExecutor.HasCommand("worktree", "add", worktreePath, "feature-branch"))
}

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
		Force:       false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitWorktree, groveErr.Code)
	assert.Contains(t, groveErr.Message, "worktree add failed")
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
		Force:       false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitWorktree, groveErr.Code)
	assert.Contains(t, groveErr.Message, "worktree add failed")
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
		Force:       false,
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitWorktree, groveErr.Code)
	assert.Contains(t, groveErr.Message, "worktree add failed")
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
		Force:       false,
	}

	err := creator.CreateWorktree("nonexistent-branch", worktreePath, options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitWorktree, groveErr.Code)
	assert.Contains(t, groveErr.Message, "worktree add failed")
}

func TestWorktreeCreatorImpl_CreateWorktree_ComplexOptionsSuccess(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return error (branch doesn't exist).
	mockExecutor.SetErrorResponse("show-ref --verify --quiet refs/heads/local-branch", fmt.Errorf("ref not found"))
	// Mock successful worktree creation with new branch.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("worktree add -b local-branch %s", worktreePath), "")
	// Mock getting default remote name.
	mockExecutor.SetSuccessResponse("config --get clone.defaultRemoteName", "origin")
	// Mock setting up remote tracking.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("-C %s branch --set-upstream-to=origin/local-branch local-branch", worktreePath), "")

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: true,
		Force:       true,
	}

	err := creator.CreateWorktree("local-branch", worktreePath, options)

	require.NoError(t, err)
	assert.True(t, mockExecutor.HasCommand("worktree", "add", "-b", "local-branch", worktreePath))
	assert.True(t, mockExecutor.HasCommand("-C", worktreePath, "branch", "--set-upstream-to=origin/local-branch", "local-branch"))
}

// Note: Internal method tests removed as they test unexported functions
// The public interface tests above provide sufficient coverage for the WorktreeCreator component.

func TestWorktreeCreatorImpl_CreateWorktree_EmptyBranch(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{}

	err := creator.CreateWorktree("", "/path/to/worktree", options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)

	// Should not make any git calls for invalid input.
	assert.Equal(t, 0, mockExecutor.CallCount)
}

func TestWorktreeCreatorImpl_CreateWorktree_EmptyPath(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{}

	err := creator.CreateWorktree("feature-branch", "", options)

	require.Error(t, err)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeWorktreeCreation, groveErr.Code)

	// Should not make any git calls for invalid input.
	assert.Equal(t, 0, mockExecutor.CallCount)
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
