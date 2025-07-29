//go:build !integration
// +build !integration

package create

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Basic worktree creation tests for standard scenarios

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
	mockExecutor.SetResponseSlice([]string{"worktree", "add", "-b", "feature-branch", worktreePath}, "", nil)
	// Mock getting default remote name.
	mockExecutor.SetSuccessResponse("config --get clone.defaultRemoteName", "origin")
	// Mock checking if remote branch exists
	mockExecutor.SetSuccessResponse("branch -r --list origin/feature-branch", "origin/feature-branch")
	// Mock setting up remote tracking.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("-C %s branch --set-upstream-to=origin/feature-branch feature-branch", worktreePath), "")
	mockExecutor.SetResponseSlice([]string{"-C", worktreePath, "branch", "--set-upstream-to=origin/feature-branch", "feature-branch"}, "", nil)

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: true,
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
	}

	err := creator.CreateWorktree("feature-branch", worktreePath, options)

	require.NoError(t, err)
	assert.True(t, mockExecutor.HasCommand("worktree", "add", worktreePath, "feature-branch"))
}

func TestWorktreeCreatorImpl_CreateWorktree_ComplexOptionsSuccess(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	// Mock branch exists check to return error (branch doesn't exist).
	mockExecutor.SetErrorResponse("show-ref --verify --quiet refs/heads/local-branch", fmt.Errorf("ref not found"))
	// Mock successful worktree creation with new branch.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("worktree add -b local-branch %s", worktreePath), "")
	mockExecutor.SetResponseSlice([]string{"worktree", "add", "-b", "local-branch", worktreePath}, "", nil)
	// Mock getting default remote name.
	mockExecutor.SetSuccessResponse("config --get clone.defaultRemoteName", "origin")
	// Mock checking if remote branch exists
	mockExecutor.SetSuccessResponse("branch -r --list origin/local-branch", "origin/local-branch")
	// Mock setting up remote tracking.
	mockExecutor.SetSuccessResponse(fmt.Sprintf("-C %s branch --set-upstream-to=origin/local-branch local-branch", worktreePath), "")
	mockExecutor.SetResponseSlice([]string{"-C", worktreePath, "branch", "--set-upstream-to=origin/local-branch", "local-branch"}, "", nil)

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{
		TrackRemote: true,
	}

	err := creator.CreateWorktree("local-branch", worktreePath, options)

	require.NoError(t, err)
	assert.True(t, mockExecutor.HasCommand("worktree", "add", "-b", "local-branch", worktreePath))
	assert.True(t, mockExecutor.HasCommand("-C", worktreePath, "branch", "--set-upstream-to=origin/local-branch", "local-branch"))
}
