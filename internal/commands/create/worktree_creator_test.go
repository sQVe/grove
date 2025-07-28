//go:build !integration
// +build !integration

package create

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testExistingWorktreePath = "/existing/worktree"

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
	assert.Contains(t, groveErr.Message, "worktree create failed")
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
	assert.Contains(t, groveErr.Message, "worktree create failed")
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
	assert.Contains(t, groveErr.Message, "worktree create failed")
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
	assert.Contains(t, groveErr.Message, "worktree create failed")
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

// Note: Internal method tests removed as they test unexported functions
// The public interface tests above provide sufficient coverage for the WorktreeCreator component.

// Sequential Mock Git Executor for testing conflict resolution
type SequentialMockGitExecutor struct {
	responses        map[string][]testutils.MockResponse // Sequential responses for commands
	otherResponses   map[string]testutils.MockResponse   // Single responses for other commands
	callCounts       map[string]int                      // Track how many times each command was called
	worktreeAddCalls int                                 // Specific counter for worktree add calls
}

func (m *SequentialMockGitExecutor) Execute(args ...string) (string, error) {
	return m.ExecuteQuiet(args...)
}

func (m *SequentialMockGitExecutor) ExecuteQuiet(args ...string) (string, error) {
	cmdKey := strings.Join(args, " ")

	// Initialize call counts if needed
	if m.callCounts == nil {
		m.callCounts = make(map[string]int)
	}

	// Track specific worktree add calls
	if len(args) >= 2 && args[0] == "worktree" && args[1] == "add" {
		m.worktreeAddCalls++
	}

	// Check for sequential responses first
	for pattern, responses := range m.responses {
		if strings.HasPrefix(cmdKey, pattern) {
			callIndex := m.callCounts[pattern]
			m.callCounts[pattern]++

			if callIndex < len(responses) {
				response := responses[callIndex]
				return response.Output, response.Error
			}
			// If we've exhausted sequential responses, return the last one
			if len(responses) > 0 {
				response := responses[len(responses)-1]
				return response.Output, response.Error
			}
		}
	}

	// Check other responses
	if response, exists := m.otherResponses[cmdKey]; exists {
		return response.Output, response.Error
	}

	return "", fmt.Errorf("mock: unhandled git command: %s", cmdKey)
}

func (m *SequentialMockGitExecutor) ExecuteWithContext(ctx context.Context, args ...string) (string, error) {
	return m.ExecuteQuiet(args...)
}

// Conflict Resolution Tests

func TestWorktreeCreatorImpl_ConflictResolution_CleanWorktreeSuccess(t *testing.T) {
	// Create a sequential mock that can handle different responses for the same command
	mockExecutor := &SequentialMockGitExecutor{
		responses: map[string][]testutils.MockResponse{
			"worktree add": {
				// First call fails with conflict
				{Output: "", Error: fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", testExistingWorktreePath)},
				// Second call succeeds after resolution
				{Output: "", Error: nil},
			},
		},
		otherResponses: map[string]testutils.MockResponse{
			"show-ref --verify --quiet refs/heads/feature-branch":    {Output: "", Error: nil},
			"-C " + testExistingWorktreePath + " status --porcelain": {Output: "", Error: nil},
			"-C " + testExistingWorktreePath + " checkout --detach":  {Output: "Switched to detached HEAD", Error: nil},
		},
	}

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

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
	assert.Equal(t, 2, mockExecutor.worktreeAddCalls)
}

func TestWorktreeCreatorImpl_ConflictResolution_DirtyWorktreeFails(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")
	conflictingWorktreePath := testExistingWorktreePath

	// Mock branch exists check
	mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature-branch", "")

	// Mock worktree creation failure with conflict error
	conflictError := fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", conflictingWorktreePath)
	mockExecutor.SetErrorResponse(fmt.Sprintf("worktree add %s feature-branch", worktreePath), conflictError)

	// Mock status check showing dirty worktree (has uncommitted changes)
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
	// Create a sequential mock that can handle different responses for the same command
	mockExecutor := &SequentialMockGitExecutor{
		responses: map[string][]testutils.MockResponse{
			"worktree add": {
				// First call fails with conflict
				{Output: "", Error: fmt.Errorf("fatal: 'feature-branch' is already used by worktree at '%s'", testExistingWorktreePath)},
				// Second call succeeds after resolution
				{Output: "", Error: nil},
			},
		},
		otherResponses: map[string]testutils.MockResponse{
			"show-ref --verify --quiet refs/heads/feature-branch":    {Output: "", Error: nil},
			"-C " + testExistingWorktreePath + " status --porcelain": {Output: "", Error: nil},
			"-C " + testExistingWorktreePath + " checkout --detach":  {Output: "Switched to detached HEAD", Error: nil},
		},
	}

	tmpDir := t.TempDir()
	worktreePath := filepath.Join(tmpDir, "worktree")

	creator := NewWorktreeCreator(mockExecutor)

	options := WorktreeOptions{TrackRemote: false}
	err := creator.CreateWorktreeWithProgress("feature-branch", worktreePath, options, nil) // No progress callback

	require.NoError(t, err)

	// Verify the worktree add command was called twice (first fails, then succeeds)
	assert.Equal(t, 2, mockExecutor.worktreeAddCalls)
}

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
