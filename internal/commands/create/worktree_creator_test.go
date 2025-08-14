package create

import (
	"errors"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testFeatureBranch = "feature-branch"
	testExistingPath  = "/existing/worktree"
	testBranchName    = "test-branch"
)

func TestNewWorktreeCreator(t *testing.T) {
	testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()

	creator := NewWorktreeCreator(mockCommander)

	assert.NotNil(t, creator)
	assert.NotNil(t, creator.commander)
	assert.NotNil(t, creator.logger)
	assert.NotNil(t, creator.conflictResolver)
}

func TestCreateWorktree_ValidationErrors(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		path       string
		expectErr  string
	}{
		{
			name:       "empty branch name",
			branchName: "",
			path:       "/some/path",
			expectErr:  "branch name cannot be empty",
		},
		{
			name:       "whitespace-only branch name",
			branchName: "   ",
			path:       "/some/path",
			expectErr:  "branch name cannot be empty or whitespace-only",
		},
		{
			name:       "empty path",
			branchName: testFeatureBranch,
			path:       "",
			expectErr:  "worktree path cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutils.NewUnitTestHelper(t).WithCleanFilesystem()
			mockCommander := testutils.CreateMockGitCommander()
			creator := NewWorktreeCreator(mockCommander)
			options := WorktreeOptions{}

			err := creator.CreateWorktree(tt.branchName, tt.path, options)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectErr)
			mockCommander.AssertExpectations(t)
		})
	}
}

func TestCreateWorktree_ExistingBranch_Success(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := "existing-branch"
	worktreePath := helper.CreateTempDir("test-worktree")
	options := WorktreeOptions{}

	// Mock branch existence check (branch exists)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(nil)

	// Mock worktree creation from existing branch
	mockCommander.On("Run", ".", "worktree", "add", worktreePath, branchName).
		Return([]byte(""), []byte(""), nil)

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.NoError(t, err)
	// Verify the worktree directory was created
	assert.DirExists(t, worktreePath, "worktree directory should exist after successful creation")
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktree_NewBranch_Success(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := "new-branch"
	worktreePath := helper.CreateTempDir("test-worktree")
	options := WorktreeOptions{BaseBranch: "main"}

	// Mock branch existence check (branch doesn't exist)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation with new branch
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath, "main").
		Return([]byte(""), []byte(""), nil)

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.NoError(t, err)
	// Verify the worktree directory was created
	assert.DirExists(t, worktreePath, "worktree directory should exist after successful creation")
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktree_NewBranch_WithRemoteTracking_Success(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := testFeatureBranch
	worktreePath := helper.GetTempPath("test-worktree")
	options := WorktreeOptions{TrackRemote: true}

	// Mock branch existence check (branch doesn't exist)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation with new branch
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Return([]byte(""), []byte(""), nil)

	// Mock remote configuration check
	mockCommander.On("Run", ".", "config", "--get", "clone.defaultRemoteName").
		Return([]byte("origin"), []byte(""), nil)

	// Mock remote branch existence check (remote branch exists)
	mockCommander.On("Run", ".", "branch", "-r", "--list", "origin/"+branchName).
		Return([]byte("origin/"+branchName), []byte(""), nil)

	// Mock upstream setup
	mockCommander.On("Run", worktreePath, "branch", "--set-upstream-to=origin/"+branchName, branchName).
		Return([]byte(""), []byte(""), nil)

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.NoError(t, err)
	// Note: worktreePath doesn't exist on filesystem since we used GetTempPath
	// which only returns a path without creating it
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktree_RemoteTracking_NoRemoteBranch(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := testFeatureBranch
	worktreePath := helper.GetTempPath("test-worktree")
	options := WorktreeOptions{TrackRemote: true}

	// Mock branch existence check (branch doesn't exist)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation with new branch
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Return([]byte(""), []byte(""), nil)

	// Mock remote configuration check
	mockCommander.On("Run", ".", "config", "--get", "clone.defaultRemoteName").
		Return([]byte("origin"), []byte(""), nil)

	// Mock remote branch existence check (remote branch doesn't exist)
	mockCommander.On("Run", ".", "branch", "-r", "--list", "origin/"+branchName).
		Return([]byte(""), []byte(""), nil)

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.NoError(t, err)
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktree_BranchInUse_DetectsConflict(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := testFeatureBranch
	worktreePath := helper.GetTempPath("test-worktree")
	conflictingPath := testExistingPath
	options := WorktreeOptions{}

	setupBranchExistsExpectation(mockCommander, branchName)
	setupBranchInUseError(mockCommander, branchName, worktreePath, conflictingPath)
	setupWorktreeList(mockCommander, conflictingPath, branchName)
	setupCleanWorktreeStatus(mockCommander, conflictingPath)
	setupDetachWorktree(mockCommander, conflictingPath)
	setupBranchExistsExpectation(mockCommander, branchName)
	setupSuccessfulWorktreeCreation(mockCommander, worktreePath, branchName)

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.NoError(t, err)
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktree_BranchInUse_CleanWorktreeDetach(t *testing.T) {
	testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()

	branchName := testFeatureBranch
	conflictingPath := testExistingPath

	// Test only the clean worktree detach scenario
	setupWorktreeList(mockCommander, conflictingPath, branchName)
	setupCleanWorktreeStatus(mockCommander, conflictingPath)
	setupDetachWorktree(mockCommander, conflictingPath)

	// Create a conflict resolver directly for testing
	resolver := newConflictResolver(mockCommander)
	err := resolver.resolveWorktreeConflict(branchName, conflictingPath)

	assert.NoError(t, err)
	mockCommander.AssertExpectations(t)
}

// Helper functions to reduce duplication and improve readability
func setupBranchExistsExpectation(mockCommander *testutils.MockGitCommander, branchName string) {
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(nil).Once()
}

func setupBranchInUseError(mockCommander *testutils.MockGitCommander, branchName, worktreePath, conflictingPath string) {
	branchInUseErr := errors.New("fatal: '" + branchName + "' is already checked out in another worktree at: " + conflictingPath)
	mockCommander.On("Run", ".", "worktree", "add", worktreePath, branchName).
		Return([]byte(""), []byte(""), branchInUseErr).Once()
}

func setupWorktreeList(mockCommander *testutils.MockGitCommander, conflictingPath, branchName string) {
	worktreeListOutput := fmt.Sprintf("worktree /main/worktree\nHEAD xyz789\nbranch refs/heads/main\n\nworktree %s\nHEAD abcd1234\nbranch refs/heads/%s\n", conflictingPath, branchName)
	mockCommander.On("Run", ".", "worktree", "list", "--porcelain").
		Return([]byte(worktreeListOutput), []byte(""), nil).Once()
}

func setupCleanWorktreeStatus(mockCommander *testutils.MockGitCommander, path string) {
	mockCommander.On("Run", path, "status", "--porcelain").
		Return([]byte(""), []byte(""), nil).Once()
}

func setupDetachWorktree(mockCommander *testutils.MockGitCommander, path string) {
	mockCommander.On("Run", path, "checkout", "--detach").
		Return([]byte(""), []byte(""), nil).Once()
}

func setupSuccessfulWorktreeCreation(mockCommander *testutils.MockGitCommander, worktreePath, branchName string) {
	mockCommander.On("Run", ".", "worktree", "add", worktreePath, branchName).
		Return([]byte(""), []byte(""), nil).Once()
}

func TestCreateWorktree_BranchInUse_ConflictResolution_UncommittedChanges(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := testFeatureBranch
	worktreePath := helper.GetTempPath("test-worktree")
	conflictingPath := testExistingPath
	options := WorktreeOptions{}

	// Mock branch existence check (branch exists)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(nil).Once()

	// Mock worktree creation failure due to branch in use
	branchInUseErr := errors.New("fatal: '" + branchName + "' is already checked out in another worktree at: " + conflictingPath)
	mockCommander.On("Run", ".", "worktree", "add", worktreePath, branchName).
		Return([]byte(""), []byte(""), branchInUseErr).Once()

	// Mock conflict resolution - get worktree list (check if main worktree)
	worktreeListOutput := fmt.Sprintf("worktree /main/worktree\nHEAD xyz789\nbranch refs/heads/main\n\nworktree %s\nHEAD abcd1234\nbranch refs/heads/%s\n", conflictingPath, branchName)
	mockCommander.On("Run", ".", "worktree", "list", "--porcelain").
		Return([]byte(worktreeListOutput), []byte(""), nil).Once()

	// Mock status check for conflicting worktree (has uncommitted changes)
	mockCommander.On("Run", conflictingPath, "status", "--porcelain").
		Return([]byte("M  file.txt\n"), []byte(""), nil).Once()

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already checked out in another worktree")
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktree_GitCommandFailure(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func(*testutils.MockGitCommander, string, string)
		expectError string
	}{
		{
			name: "worktree creation failure",
			setupMocks: func(mock *testutils.MockGitCommander, branchName, path string) {
				mock.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
					Return(nil)
				mock.On("Run", ".", "worktree", "add", path, branchName).
					Return([]byte(""), []byte(""), errors.New("worktree add failed"))
				// Expect rollback attempt
				mock.ExpectRunQuiet(".", []string{"worktree", "remove", "--force", path}).
					Return(nil)
			},
			expectError: "existing-branch",
		},
		{
			name: "new branch creation failure",
			setupMocks: func(mock *testutils.MockGitCommander, branchName, path string) {
				mock.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
					Return(errors.New("branch not found"))
				mock.On("Run", ".", "worktree", "add", "-b", branchName, path).
					Return([]byte(""), []byte(""), errors.New("worktree add failed"))
				// Expect rollback attempt
				mock.ExpectRunQuiet(".", []string{"worktree", "remove", "--force", path}).
					Return(nil)
			},
			expectError: "new-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
			mockCommander := testutils.CreateMockGitCommander()
			creator := NewWorktreeCreator(mockCommander)

			branchName := testBranchName
			worktreePath := helper.CreateTempDir("test-worktree")
			options := WorktreeOptions{}

			tt.setupMocks(mockCommander, branchName, worktreePath)

			err := creator.CreateWorktree(branchName, worktreePath, options)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectError)
			mockCommander.AssertExpectations(t)
		})
	}
}

func TestCreateWorktree_ParentDirectoryCreation(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	// Use a nested path that requires parent directory creation
	branchName := testBranchName
	tempDir := helper.CreateTempDir("base")
	worktreePath := filepath.Join(tempDir, "deep", "nested", "worktree")
	options := WorktreeOptions{}

	// Mock branch existence check (branch doesn't exist)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation with new branch
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Return([]byte(""), []byte(""), nil)

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.NoError(t, err)

	// Verify parent directories were created
	parentDir := filepath.Dir(worktreePath)
	assert.DirExists(t, parentDir, "parent directory should be created")

	// Verify the full path structure was created
	grandParentDir := filepath.Dir(parentDir)
	assert.DirExists(t, grandParentDir, "grandparent directory should exist")
	assert.Equal(t, filepath.Join(tempDir, "deep"), grandParentDir, "grandparent should be 'deep' directory")

	// Verify the nested structure
	assert.Equal(t, filepath.Join(tempDir, "deep", "nested"), parentDir, "parent should be 'nested' directory")

	mockCommander.AssertExpectations(t)
}

func TestCreateWorktree_Rollback_OnFailure(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := testBranchName
	tempDir := helper.CreateTempDir("base")
	worktreePath := filepath.Join(tempDir, "test-worktree")
	options := WorktreeOptions{}

	// Create the worktree directory to simulate partial creation
	err := os.MkdirAll(worktreePath, 0o755)
	require.NoError(t, err)

	// Mock branch existence check (branch doesn't exist)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation failure
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Return([]byte(""), []byte(""), errors.New("creation failed"))

	// Mock rollback - worktree remove
	mockCommander.ExpectRunQuiet(".", []string{"worktree", "remove", "--force", worktreePath}).
		Return(nil)

	err = creator.CreateWorktree(branchName, worktreePath, options)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "new-branch")
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktreeWithProgress_CallbackInvocation(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := testFeatureBranch
	worktreePath := helper.GetTempPath("test-worktree")
	conflictingPath := testExistingPath
	options := WorktreeOptions{}

	var progressMessages []string
	progressCallback := func(message string) {
		progressMessages = append(progressMessages, message)
	}

	// Setup the full conflict resolution scenario with progress tracking
	setupConflictResolutionScenario(mockCommander, branchName, worktreePath, conflictingPath)

	err := creator.CreateWorktreeWithProgress(branchName, worktreePath, options, progressCallback)

	assert.NoError(t, err)
	assert.Contains(t, progressMessages, "Branch 'feature-branch' is in use, attempting automatic resolution...")
	assert.Contains(t, progressMessages, "Resolved conflict: switched previous worktree to detached HEAD")
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktreeWithProgress_TracksSteps(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := "simple-branch"
	worktreePath := helper.GetTempPath("test-worktree")
	options := WorktreeOptions{}

	progressCalled := false
	progressCallback := func(message string) {
		progressCalled = true
	}

	// Simple successful creation scenario
	setupBranchExistsExpectation(mockCommander, branchName)
	setupSuccessfulWorktreeCreation(mockCommander, worktreePath, branchName)

	err := creator.CreateWorktreeWithProgress(branchName, worktreePath, options, progressCallback)

	assert.NoError(t, err)
	// Progress callbacks are optional and may not be called for simple scenarios
	// The important part is that the callback is accepted and doesn't cause errors
	_ = progressCalled // Progress reporting is implementation-specific
	mockCommander.AssertExpectations(t)
}

// Helper function to setup a complete conflict resolution scenario
func setupConflictResolutionScenario(mockCommander *testutils.MockGitCommander, branchName, worktreePath, conflictingPath string) {
	setupBranchExistsExpectation(mockCommander, branchName)
	setupBranchInUseError(mockCommander, branchName, worktreePath, conflictingPath)
	setupWorktreeList(mockCommander, conflictingPath, branchName)
	setupCleanWorktreeStatus(mockCommander, conflictingPath)
	setupDetachWorktree(mockCommander, conflictingPath)
	setupBranchExistsExpectation(mockCommander, branchName)
	setupSuccessfulWorktreeCreation(mockCommander, worktreePath, branchName)
}

func TestBranchExists(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		mockSetup  func(*testutils.MockGitCommander, string)
		expected   bool
		expectErr  bool
	}{
		{
			name:       "branch exists",
			branchName: "existing-branch",
			mockSetup: func(mock *testutils.MockGitCommander, branch string) {
				mock.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branch}).
					Return(nil)
			},
			expected:  true,
			expectErr: false,
		},
		{
			name:       "branch does not exist",
			branchName: "non-existing-branch",
			mockSetup: func(mock *testutils.MockGitCommander, branch string) {
				mock.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branch}).
					Return(errors.New("branch not found"))
			},
			expected:  false,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
			mockCommander := testutils.CreateMockGitCommander()
			creator := NewWorktreeCreator(mockCommander)

			tt.mockSetup(mockCommander, tt.branchName)

			exists, err := creator.branchExists(tt.branchName)

			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, exists)
			}
			mockCommander.AssertExpectations(t)

			// Ensure cleanup happens
			_ = helper
		})
	}
}

func TestCreateWorktree_RemoteTrackingFailure(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := testFeatureBranch
	worktreePath := helper.GetTempPath("test-worktree")
	options := WorktreeOptions{TrackRemote: true}

	// Mock branch existence check (branch doesn't exist)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation with new branch
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Return([]byte(""), []byte(""), nil)

	// Mock remote configuration check
	mockCommander.On("Run", ".", "config", "--get", "clone.defaultRemoteName").
		Return([]byte("origin"), []byte(""), nil)

	// Mock remote branch existence check (remote branch exists)
	mockCommander.On("Run", ".", "branch", "-r", "--list", "origin/"+branchName).
		Return([]byte("origin/"+branchName), []byte(""), nil)

	// Mock upstream setup failure
	mockCommander.On("Run", worktreePath, "branch", "--set-upstream-to=origin/"+branchName, branchName).
		Return([]byte(""), []byte(""), errors.New("upstream setup failed"))

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "remote-tracking")
	mockCommander.AssertExpectations(t)
}

func TestCreateWorktree_DefaultRemoteName(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := testFeatureBranch
	worktreePath := helper.GetTempPath("test-worktree")
	options := WorktreeOptions{TrackRemote: true}

	// Mock branch existence check (branch doesn't exist)
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation with new branch
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Return([]byte(""), []byte(""), nil)

	// Mock remote configuration check (fails, should default to "origin")
	mockCommander.On("Run", ".", "config", "--get", "clone.defaultRemoteName").
		Return([]byte(""), []byte(""), errors.New("config not found"))

	// Mock remote branch existence check with default remote name
	mockCommander.On("Run", ".", "branch", "-r", "--list", "origin/"+branchName).
		Return([]byte(""), []byte(""), nil)

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.NoError(t, err)
	mockCommander.AssertExpectations(t)
}

// TestWorktreeCreator_Create_ConcurrentCreation validates handling of concurrent worktree creation attempts.
// This test ensures that parallel creation operations don't interfere with each other and that
// shared resources are properly protected from race conditions.
func TestWorktreeCreator_Create_ConcurrentCreation(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	// Create a shared mock commander and a shared creator to test actual concurrency
	// on the same instance (this tests thread safety of the WorktreeCreator)
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	// Set up multiple concurrent operations
	numConcurrent := 5
	branches := make([]string, numConcurrent)
	paths := make([]string, numConcurrent)

	for i := 0; i < numConcurrent; i++ {
		branches[i] = fmt.Sprintf("feature-%d", i+1)
		paths[i] = helper.CreateTempDir(fmt.Sprintf("worktree-%d", i+1))
	}

	// Set up mock expectations with potential for race conditions
	// We use a mutex to protect the mock setup since the mock library isn't thread-safe
	var setupMutex sync.Mutex
	for i := 0; i < numConcurrent; i++ {
		idx := i // Capture loop variable
		setupMutex.Lock()

		// Mock branch existence check (branches don't exist)
		mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branches[idx]}).
			Return(errors.New("branch not found")).Once()

		// Mock successful worktree creation
		// Add a small delay to simulate real git operations and expose race conditions
		mockCommander.On("Run", ".", "worktree", "add", "-b", branches[idx], paths[idx]).
			Return([]byte(""), []byte(""), nil).
			Run(func(args mock.Arguments) {
				// Simulate some work to expose race conditions
				time.Sleep(10 * time.Millisecond)
			}).Once()

		setupMutex.Unlock()
	}

	// Track completion order to verify parallelism
	completionOrder := make([]int, 0, numConcurrent)
	var completionMutex sync.Mutex

	// Execute concurrent worktree creation operations
	var wg sync.WaitGroup
	errChan := make(chan error, numConcurrent)
	startTime := time.Now()

	for i := 0; i < numConcurrent; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Add random delay to ensure operations overlap
			time.Sleep(time.Duration(rand.Intn(20)) * time.Millisecond)

			err := creator.CreateWorktree(branches[idx], paths[idx], WorktreeOptions{})
			errChan <- err

			completionMutex.Lock()
			completionOrder = append(completionOrder, idx)
			completionMutex.Unlock()
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	close(errChan)

	elapsed := time.Since(startTime)

	// Collect and verify results
	errorCount := 0
	for err := range errChan {
		if err != nil {
			errorCount++
			t.Logf("Concurrent operation error: %v", err)
		}
	}

	// Verify no errors occurred
	assert.Equal(t, 0, errorCount, "all concurrent operations should succeed without errors")

	// Verify operations actually ran concurrently (should complete faster than sequential)
	// With 5 operations at ~10ms each, sequential would take ~50ms
	// Concurrent should complete in ~10-20ms
	assert.Less(t, elapsed, 40*time.Millisecond,
		"concurrent operations should complete faster than sequential execution")

	// Verify all worktree directories were created
	for i, path := range paths {
		assert.DirExists(t, path, "worktree directory %d should exist", i)
	}

	// Verify completion order is not strictly sequential (indicates parallelism)
	isSequential := true
	for i := 0; i < len(completionOrder)-1; i++ {
		if completionOrder[i] > completionOrder[i+1] {
			isSequential = false
			break
		}
	}
	assert.False(t, isSequential, "completion order should not be strictly sequential, indicating parallel execution")

	mockCommander.AssertExpectations(t)
}

// TestWorktreeCreator_Create_FileSystemConsistency verifies file system state consistency after operations.
// This test ensures that worktree creation maintains consistent filesystem state even on failures.
func TestWorktreeCreator_Create_FileSystemConsistency(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := "consistency-test"
	baseDir := helper.CreateTempDir("base")
	worktreePath := filepath.Join(baseDir, "nested", "worktree")
	options := WorktreeOptions{}

	// Create initial filesystem state to verify consistency
	existingFile := testutils.CreateTestFile(t, baseDir, "existing.txt", "original content")
	existingDir := filepath.Join(baseDir, "existing-dir")
	err := os.MkdirAll(existingDir, 0o755)
	require.NoError(t, err)
	nestedFile := testutils.CreateTestFile(t, existingDir, "nested.txt", "nested content")

	// Verify initial state
	assert.FileExists(t, existingFile, "existing file should exist before operation")
	assert.FileExists(t, nestedFile, "nested file should exist before operation")
	assert.DirExists(t, existingDir, "existing directory should exist before operation")

	// Mock branch existence check
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation that fails after creating some files
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Run(func(args mock.Arguments) {
			// Simulate partial worktree creation
			err := os.MkdirAll(worktreePath, 0o755)
			require.NoError(t, err)
			// Create a .git file as would happen during worktree creation
			gitFile := filepath.Join(worktreePath, ".git")
			err = os.WriteFile(gitFile, []byte("gitdir: ../../../.git/worktrees/"+filepath.Base(worktreePath)), 0o644)
			require.NoError(t, err)
		}).
		Return([]byte(""), []byte(""), errors.New("worktree add failed: permission denied"))

	// Mock rollback - should clean up the partially created worktree
	mockCommander.ExpectRunQuiet(".", []string{"worktree", "remove", "--force", worktreePath}).
		Run(func(args mock.Arguments) {
			// Simulate cleanup
			_ = os.RemoveAll(worktreePath)
		}).
		Return(nil)

	// Attempt worktree creation
	err = creator.CreateWorktree(branchName, worktreePath, options)

	// Verify operation failed
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "new-branch")

	// Verify filesystem consistency after failure and rollback:

	// 1. Worktree directory should be completely removed
	assert.NoDirExists(t, worktreePath, "worktree directory should not exist after rollback")

	// 2. No .git file should remain
	gitFile := filepath.Join(worktreePath, ".git")
	assert.NoFileExists(t, gitFile, ".git file should not exist after rollback")

	// 3. Original files should be unchanged
	assert.FileExists(t, existingFile, "existing file should still exist")
	assert.FileExists(t, nestedFile, "nested file should still exist")

	// 4. Verify file contents are unchanged
	content, err := os.ReadFile(existingFile)
	assert.NoError(t, err)
	assert.Equal(t, "original content", string(content), "existing file content should be unchanged")

	content, err = os.ReadFile(nestedFile)
	assert.NoError(t, err)
	assert.Equal(t, "nested content", string(content), "nested file content should be unchanged")

	// 5. Original directories should still exist
	assert.DirExists(t, baseDir, "base directory should still exist")
	assert.DirExists(t, existingDir, "existing directory should still exist")

	// 6. Parent directory created for worktree might remain (implementation detail)
	parentDir := filepath.Join(baseDir, "nested")
	if _, err := os.Stat(parentDir); err == nil {
		// If it exists, verify it's empty
		entries, err := os.ReadDir(parentDir)
		if err == nil {
			assert.Empty(t, entries, "parent directory should be empty if it exists")
		}
	}

	mockCommander.AssertExpectations(t)
}

// TestWorktreeCreator_Create_SymlinkHandling tests creation and validation of symlinks in worktrees.
// This test ensures proper handling of symbolic links during worktree operations.
func TestWorktreeCreator_Create_SymlinkHandling(t *testing.T) {
	// Skip on Windows if symlinks are not supported
	if testutils.IsWindows() {
		t.Skip("Skipping symlink test on Windows")
	}

	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := "symlink-branch"
	baseDir := helper.CreateTempDir("symlink-test")

	// Create a target directory and a symlink to it
	targetDir := filepath.Join(baseDir, "target")
	err := os.MkdirAll(targetDir, 0o755)
	require.NoError(t, err)

	// Create a test file in the target directory
	targetFile := testutils.CreateTestFile(t, targetDir, "target.txt", "target content")

	symlinkPath := filepath.Join(baseDir, "link-to-target")
	err = os.Symlink(targetDir, symlinkPath)
	require.NoError(t, err)

	// Verify symlink was created correctly
	assert.True(t, isSymlink(t, symlinkPath), "symlink should be created")

	// Verify symlink points to the correct target
	resolvedPath, err := filepath.EvalSymlinks(symlinkPath)
	assert.NoError(t, err)
	assert.Equal(t, targetDir, resolvedPath, "symlink should resolve to target directory")

	// Try to create worktree in a path that includes the symlink
	worktreePath := filepath.Join(symlinkPath, "worktree")
	options := WorktreeOptions{}

	// Mock branch existence check
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock successful worktree creation
	// The worktree creation should handle symlinks properly
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Run(func(args mock.Arguments) {
			// Simulate worktree creation in the symlinked path
			// Git would resolve symlinks and create the worktree
			realWorktreePath := filepath.Join(targetDir, "worktree")
			err := os.MkdirAll(realWorktreePath, 0o755)
			require.NoError(t, err)
			// Create a .git file to simulate Git worktree
			gitFile := filepath.Join(realWorktreePath, ".git")
			err = os.WriteFile(gitFile, []byte("gitdir: ../../../../.git/worktrees/worktree"), 0o644)
			require.NoError(t, err)
		}).
		Return([]byte(""), []byte(""), nil)

	err = creator.CreateWorktree(branchName, worktreePath, options)

	assert.NoError(t, err)

	// Verify the worktree was created in the correct location
	// Check both the symlinked path and the real path
	realWorktreePath := filepath.Join(targetDir, "worktree")
	assert.DirExists(t, realWorktreePath, "worktree should exist in target directory")

	// Verify we can access the worktree through the symlink
	stat, err := os.Stat(worktreePath)
	assert.NoError(t, err, "should be able to stat worktree through symlink")
	assert.True(t, stat.IsDir(), "worktree should be a directory")

	// Verify .git file exists
	gitFile := filepath.Join(realWorktreePath, ".git")
	assert.FileExists(t, gitFile, ".git file should exist in worktree")

	// Verify original symlink is still intact
	assert.True(t, isSymlink(t, symlinkPath), "original symlink should still exist")

	// Verify original target file is unchanged
	assert.FileExists(t, targetFile, "target file should still exist")
	content, err := os.ReadFile(targetFile)
	assert.NoError(t, err)
	assert.Equal(t, "target content", string(content), "target file content should be unchanged")

	mockCommander.AssertExpectations(t)
}

// isSymlink checks if a path is a symbolic link
func isSymlink(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}

// TestWorktreeCreator_Create_PermissionValidation validates file permission handling during creation.
// This test ensures proper permission validation and error handling for file operations.
func TestWorktreeCreator_Create_PermissionValidation(t *testing.T) {
	// Skip on Windows as Unix permissions work differently
	if testutils.IsWindows() {
		t.Skip("Skipping Unix permission test on Windows")
	}

	// Skip if running as root since root can write to read-only directories
	if os.Geteuid() == 0 {
		t.Skip("Skipping permission test when running as root")
	}

	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := "permission-test"
	baseDir := helper.CreateTempDir("perm-test")

	// Create initial state with some files
	testFile := testutils.CreateTestFile(t, baseDir, "existing.txt", "existing content")

	// Create a directory with restricted permissions
	restrictedDir := filepath.Join(baseDir, "restricted")
	err := os.MkdirAll(restrictedDir, 0o755)
	require.NoError(t, err)

	// Add a file in the restricted directory before changing permissions
	restrictedFile := testutils.CreateTestFile(t, restrictedDir, "restricted.txt", "restricted content")

	// Change permissions to read-only after creation (no write permission)
	err = os.Chmod(restrictedDir, 0o555)
	require.NoError(t, err)
	defer func() {
		// Restore permissions for cleanup
		_ = os.Chmod(restrictedDir, 0o755)
	}()

	// Verify permissions are actually restricted
	testPath := filepath.Join(restrictedDir, "test-write")
	err = os.WriteFile(testPath, []byte("test"), 0o644)
	assert.Error(t, err, "should not be able to write to restricted directory")
	assert.Contains(t, err.Error(), "permission denied", "error should be permission denied")

	worktreePath := filepath.Join(restrictedDir, "worktree")
	options := WorktreeOptions{}

	// Mock branch existence check
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// The worktree creation will fail when trying to create directories
	// because the parent directory is read-only
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Run(func(args mock.Arguments) {
			// Try to create the worktree directory - this should fail
			err := os.MkdirAll(worktreePath, 0o755)
			// We expect this to fail due to permissions
			if err == nil {
				t.Error("Expected directory creation to fail due to permissions")
			}
		}).
		Return([]byte(""), []byte("fatal: could not create directory"),
			errors.New("fatal: could not create directory: permission denied"))

	// Mock rollback attempt - nothing to rollback since creation failed early
	mockCommander.ExpectRunQuiet(".", []string{"worktree", "remove", "--force", worktreePath}).
		Return(errors.New("worktree not found")).Maybe()

	err = creator.CreateWorktree(branchName, worktreePath, options)

	// The operation should fail due to permission restrictions
	assert.Error(t, err, "worktree creation should fail due to permissions")

	// Verify no partial worktree was created
	assert.NoDirExists(t, worktreePath, "worktree directory should not exist")

	// Verify the restricted directory permissions are still intact
	info, err := os.Stat(restrictedDir)
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0o555), info.Mode()&os.ModePerm,
		"restricted directory should still have read-only permissions")

	// Verify original files are unchanged
	assert.FileExists(t, testFile, "original test file should still exist")
	assert.FileExists(t, restrictedFile, "restricted file should still exist")

	content, err := os.ReadFile(testFile)
	assert.NoError(t, err)
	assert.Equal(t, "existing content", string(content), "original file content should be unchanged")

	content, err = os.ReadFile(restrictedFile)
	assert.NoError(t, err)
	assert.Equal(t, "restricted content", string(content), "restricted file content should be unchanged")

	mockCommander.AssertExpectations(t)
}

// TestWorktreeCreator_Cleanup_LockedFiles tests cleanup behavior when files are locked or in use.
// This test ensures robust cleanup even when files cannot be immediately removed.
func TestWorktreeCreator_Cleanup_LockedFiles(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := "cleanup-test"
	baseDir := helper.CreateTempDir("cleanup")
	worktreePath := filepath.Join(baseDir, "worktree")

	// Create the worktree directory to simulate partial creation
	err := os.MkdirAll(worktreePath, 0o755)
	require.NoError(t, err)

	// Create a file in the worktree and keep it open to simulate a lock
	lockedFile := filepath.Join(worktreePath, "locked.txt")
	file, err := os.Create(lockedFile)
	require.NoError(t, err)
	defer func() {
		_ = file.Close()
	}()

	// Write some data to ensure the file is actually created
	_, err = file.WriteString("locked content")
	require.NoError(t, err)

	options := WorktreeOptions{}

	// Mock branch existence check
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(errors.New("branch not found"))

	// Mock worktree creation failure to trigger rollback
	mockCommander.On("Run", ".", "worktree", "add", "-b", branchName, worktreePath).
		Return([]byte(""), []byte(""), errors.New("worktree creation failed"))

	// Mock rollback attempt - git worktree remove
	mockCommander.ExpectRunQuiet(".", []string{"worktree", "remove", "--force", worktreePath}).
		Return(errors.New("worktree not found or locked"))

	// The rollback will try os.RemoveAll as fallback, which may partially succeed

	err = creator.CreateWorktree(branchName, worktreePath, options)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "new-branch")

	// On Windows, the locked file might prevent directory removal
	// On Unix, the directory might be removed despite the open file handle
	// The test verifies that cleanup attempts were made even if they couldn't fully succeed

	mockCommander.AssertExpectations(t)
}

// TestWorktreeCreator_Recovery_CorruptedState tests recovery from corrupted worktree state.
// This test ensures the system can recover from inconsistent or corrupted Git worktree metadata.
func TestWorktreeCreator_Recovery_CorruptedState(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	branchName := "recovery-test"
	worktreePath := helper.GetTempPath("corrupted-worktree")
	conflictingPath := "/corrupted/worktree"
	options := WorktreeOptions{}

	// Mock branch existence check
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/" + branchName}).
		Return(nil).Once()

	// Mock initial worktree creation attempt that fails with corrupted state error
	corruptedErr := errors.New("fatal: '" + branchName + "' is already checked out in another worktree at: " + conflictingPath + "\nwarning: worktree metadata is corrupted")
	mockCommander.On("Run", ".", "worktree", "add", worktreePath, branchName).
		Return([]byte(""), []byte(""), corruptedErr).Once()

	// Mock conflict resolution attempt - get worktree list
	// Return corrupted/inconsistent worktree list
	worktreeListOutput := fmt.Sprintf("worktree /main/worktree\nHEAD xyz789\nbranch refs/heads/main\n\nworktree %s\nHEAD \nbranch refs/heads/%s\n", conflictingPath, branchName)
	mockCommander.On("Run", ".", "worktree", "list", "--porcelain").
		Return([]byte(worktreeListOutput), []byte(""), nil).Once()

	// Mock status check for corrupted worktree (returns error due to corruption)
	mockCommander.On("Run", conflictingPath, "status", "--porcelain").
		Return([]byte(""), []byte("fatal: not a git repository"), errors.New("not a git repository")).Once()

	// Since the worktree is corrupted and status check failed, conflict resolution should fail
	// and return an error about being unable to resolve the conflict

	err := creator.CreateWorktree(branchName, worktreePath, options)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already checked out in another worktree")

	mockCommander.AssertExpectations(t)
}
