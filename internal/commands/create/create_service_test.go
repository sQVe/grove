package create

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
)

// MockBranchResolver implements BranchResolver interface for testing
// Test constants to avoid linting issues
var (
	testMainBranch  = "main"
	testFeaturePath = testutils.NormalizePath("/path/to/worktree/feature-branch")
	testRepoPath    = testutils.NormalizePath("/path/to/worktree/repo")
)

type MockBranchResolver struct {
	mock.Mock
}

func (m *MockBranchResolver) ResolveBranch(name, base string, createIfMissing bool) (*BranchInfo, error) {
	args := m.Called(name, base, createIfMissing)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BranchInfo), args.Error(1)
}

func (m *MockBranchResolver) ResolveURL(url string) (*URLBranchInfo, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*URLBranchInfo), args.Error(1)
}

func (m *MockBranchResolver) ResolveRemoteBranch(remoteBranch string) (*BranchInfo, error) {
	args := m.Called(remoteBranch)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*BranchInfo), args.Error(1)
}

func (m *MockBranchResolver) RemoteExists(remoteName string) bool {
	args := m.Called(remoteName)
	return args.Bool(0)
}

// MockPathGenerator implements PathGenerator interface for testing
type MockPathGenerator struct {
	mock.Mock
}

func (m *MockPathGenerator) GeneratePath(branchName, basePath string) (string, error) {
	args := m.Called(branchName, basePath)
	return args.String(0), args.Error(1)
}

func (m *MockPathGenerator) ResolveUserPath(userPath string) (string, error) {
	args := m.Called(userPath)
	return args.String(0), args.Error(1)
}

// MockWorktreeCreator implements WorktreeCreator interface for testing
type MockWorktreeCreator struct {
	mock.Mock
}

func (m *MockWorktreeCreator) CreateWorktree(branchName, path string, options WorktreeOptions) error {
	args := m.Called(branchName, path, options)
	return args.Error(0)
}

func (m *MockWorktreeCreator) CreateWorktreeWithProgress(branchName, path string, options WorktreeOptions, progressCallback ProgressCallback) error {
	args := m.Called(branchName, path, options, progressCallback)
	return args.Error(0)
}

// MockFileManager implements FileManager interface for testing
type MockFileManager struct {
	mock.Mock
}

func (m *MockFileManager) CopyFiles(sourceWorktree, targetWorktree string, patterns []string, options CopyOptions) error {
	args := m.Called(sourceWorktree, targetWorktree, patterns, options)
	return args.Error(0)
}

func (m *MockFileManager) DiscoverSourceWorktree() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockFileManager) FindWorktreeByBranch(branchName string) (string, error) {
	args := m.Called(branchName)
	return args.String(0), args.Error(1)
}

func (m *MockFileManager) GetCurrentWorktreePath() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockFileManager) ResolveConflicts(conflicts []FileConflict, strategy ConflictStrategy) error {
	args := m.Called(conflicts, strategy)
	return args.Error(0)
}

// Test helper functions
func assertGroveError(t *testing.T, err error, expectedCode string) {
	t.Helper()
	require.Error(t, err)
	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr, "Expected GroveError but got %T", err)
	assert.Equal(t, expectedCode, groveErr.Code)
}

func mockRemoteExistsForSlashBranch(deps *TestDependencies, branchName string) {
	if strings.Contains(branchName, "/") {
		parts := strings.SplitN(branchName, "/", 2)
		deps.BranchResolver.On("RemoteExists", parts[0]).Return(false).Once()
	}
}

// TestDependencies holds all mocked dependencies for create service tests
type TestDependencies struct {
	Service         *CreateServiceImpl
	GitCommander    *testutils.MockGitCommander
	BranchResolver  *MockBranchResolver
	PathGenerator   *MockPathGenerator
	WorktreeCreator *MockWorktreeCreator
	FileManager     *MockFileManager
}

// Test fixtures
func setupCreateServiceTest(t *testing.T) *TestDependencies {
	// Use the centralized mock creation for consistency
	gitCommander := testutils.CreateMockGitCommander()
	branchResolver := &MockBranchResolver{}
	pathGenerator := &MockPathGenerator{}
	worktreeCreator := &MockWorktreeCreator{}
	fileManager := &MockFileManager{}

	service := NewCreateService(
		gitCommander,
		branchResolver,
		pathGenerator,
		worktreeCreator,
		fileManager,
	)

	return &TestDependencies{
		Service:         service,
		GitCommander:    gitCommander,
		BranchResolver:  branchResolver,
		PathGenerator:   pathGenerator,
		WorktreeCreator: worktreeCreator,
		FileManager:     fileManager,
	}
}

func TestCreateService_Create_Success(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName:   "feature-branch",
		WorktreePath: "",
		BaseBranch:   testMainBranch,
		CopyFiles:    false,
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   false,
		IsRemote: false,
	}

	expectedPath := testFeaturePath

	deps.BranchResolver.On("ResolveBranch", "feature-branch", testMainBranch, true).Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "feature-branch", "").Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "feature-branch", expectedPath, mock.MatchedBy(func(opts WorktreeOptions) bool {
		return opts.BaseBranch == testMainBranch
	}), mock.AnythingOfType("ProgressCallback")).Return(nil)

	result, err := deps.Service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.True(t, result.WasCreated)
	assert.Equal(t, testMainBranch, result.BaseBranch)
	assert.Equal(t, 0, result.CopiedFiles)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_Create_WithUserPath(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName:   "feature-branch",
		WorktreePath: "./custom-path",
		BaseBranch:   testMainBranch,
		CopyFiles:    false,
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   true,
		IsRemote: false,
	}

	expectedPath := testutils.NormalizePath("/path/to/custom-path")

	deps.BranchResolver.On("ResolveBranch", "feature-branch", testMainBranch, true).Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("ResolveUserPath", "./custom-path").Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "feature-branch", expectedPath, mock.MatchedBy(func(opts WorktreeOptions) bool {
		return opts.BaseBranch == testMainBranch
	}), mock.AnythingOfType("ProgressCallback")).Return(nil)

	result, err := deps.Service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.False(t, result.WasCreated) // Branch already existed
	assert.Equal(t, testMainBranch, result.BaseBranch)
	assert.Equal(t, 0, result.CopiedFiles)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_Create_WithFileCopying(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName:   "feature-branch",
		WorktreePath: "",
		BaseBranch:   testMainBranch,
		CopyFiles:    true,
		CopyPatterns: []string{".env*", ".vscode/"},
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   false,
		IsRemote: false,
	}

	expectedPath := testFeaturePath
	sourceWorktree := testutils.NormalizePath("/path/to/main/worktree")

	deps.BranchResolver.On("ResolveBranch", "feature-branch", testMainBranch, true).Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "feature-branch", "").Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "feature-branch", expectedPath, mock.MatchedBy(func(opts WorktreeOptions) bool {
		return opts.BaseBranch == testMainBranch
	}), mock.AnythingOfType("ProgressCallback")).Return(nil)

	// File copying mocks
	deps.FileManager.On("FindWorktreeByBranch", testMainBranch).Return(sourceWorktree, nil)
	deps.FileManager.On("CopyFiles", sourceWorktree, expectedPath, []string{".env*", ".vscode/"}, mock.MatchedBy(func(opts CopyOptions) bool {
		return !opts.DryRun
	})).Return(nil)

	result, err := deps.Service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.True(t, result.WasCreated)
	assert.Equal(t, testMainBranch, result.BaseBranch)
	assert.Equal(t, 2, result.CopiedFiles) // Estimated from patterns count

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
	deps.FileManager.AssertExpectations(t)
}

func TestCreateService_Create_InvalidOptions(t *testing.T) {
	deps := setupCreateServiceTest(t)

	testCases := []struct {
		name    string
		options *CreateOptions
		wantErr string
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: "options cannot be nil",
		},
		{
			name: "empty branch name",
			options: &CreateOptions{
				BranchName: "",
			},
			wantErr: "branch name cannot be empty",
		},
		{
			name: "path traversal attempt",
			options: &CreateOptions{
				BranchName:   "feature-branch",
				WorktreePath: "../../../etc/passwd",
			},
			wantErr: "path cannot contain '..' components",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := deps.Service.Create(tc.options)

			assert.Nil(t, result)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.ErrorAs(t, err, &groveErr)
			assert.Equal(t, errors.ErrCodeConfigInvalid, groveErr.Code)
			assert.Contains(t, groveErr.Cause.Error(), tc.wantErr)
		})
	}
}

func TestCreateService_Create_BranchResolutionFailure(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "invalid-branch",
		BaseBranch: testMainBranch,
		CopyFiles:  false,
	}

	expectedError := fmt.Errorf("branch does not exist and cannot be created")
	deps.BranchResolver.On("ResolveBranch", "invalid-branch", testMainBranch, true).Return(nil, expectedError)

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeGitOperation, groveErr.Code)
	assert.Equal(t, "failed to resolve branch information", groveErr.Message)
	assert.Equal(t, expectedError, groveErr.Cause)

	deps.BranchResolver.AssertExpectations(t)
}

func TestCreateService_Create_PathGenerationFailure(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName:   "feature-branch",
		WorktreePath: "",
		BaseBranch:   testMainBranch,
		CopyFiles:    false,
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   false,
		IsRemote: false,
	}

	expectedError := fmt.Errorf("cannot generate path: workspace not configured")

	deps.BranchResolver.On("ResolveBranch", "feature-branch", testMainBranch, true).Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "feature-branch", "").Return("", expectedError)

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeFileSystem, groveErr.Code)
	assert.Equal(t, "failed to generate worktree path", groveErr.Message)
	assert.Equal(t, expectedError, groveErr.Cause)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
}

func TestCreateService_Create_WorktreeCreationFailure(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName:   "feature-branch",
		WorktreePath: "",
		BaseBranch:   testMainBranch,
		CopyFiles:    false,
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   false,
		IsRemote: false,
	}

	expectedPath := testFeaturePath
	expectedError := fmt.Errorf("failed to create worktree: permission denied")

	deps.BranchResolver.On("ResolveBranch", "feature-branch", testMainBranch, true).Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "feature-branch", "").Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "feature-branch", expectedPath, mock.MatchedBy(func(opts WorktreeOptions) bool {
		return opts.BaseBranch == testMainBranch
	}), mock.AnythingOfType("ProgressCallback")).Return(expectedError)

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeGitOperation, groveErr.Code)
	assert.Equal(t, "failed to create worktree", groveErr.Message)
	assert.Equal(t, expectedError, groveErr.Cause)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_Create_FileCopyingFailure_NotCritical(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName:   "feature-branch",
		WorktreePath: "",
		BaseBranch:   testMainBranch,
		CopyFiles:    true,
		CopyPatterns: []string{".env*"},
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   false,
		IsRemote: false,
	}

	expectedPath := testFeaturePath
	sourceWorktree := testutils.NormalizePath("/path/to/main/worktree")
	copyError := fmt.Errorf("file not found")

	deps.BranchResolver.On("ResolveBranch", "feature-branch", testMainBranch, true).Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "feature-branch", "").Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "feature-branch", expectedPath, mock.MatchedBy(func(opts WorktreeOptions) bool {
		return opts.BaseBranch == testMainBranch
	}), mock.AnythingOfType("ProgressCallback")).Return(nil)

	// File copying fails but is not critical
	deps.FileManager.On("FindWorktreeByBranch", testMainBranch).Return(sourceWorktree, nil)
	deps.FileManager.On("CopyFiles", sourceWorktree, expectedPath, []string{".env*"}, mock.MatchedBy(func(opts CopyOptions) bool {
		return !opts.DryRun
	})).Return(copyError)

	result, err := deps.Service.Create(options)

	// Assert - should succeed despite file copying failure
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.True(t, result.WasCreated)
	assert.Equal(t, testMainBranch, result.BaseBranch)
	assert.Equal(t, 0, result.CopiedFiles) // No files copied due to error

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
	deps.FileManager.AssertExpectations(t)
}

func TestCreateService_Create_URLInput(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName:   "https://github.com/owner/repo/tree/feature-branch",
		WorktreePath: "",
		BaseBranch:   testMainBranch,
		CopyFiles:    false,
	}

	expectedURLInfo := &URLBranchInfo{
		BranchName: "feature-branch",
		RepoURL:    "https://github.com/owner/repo",
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   false,
		IsRemote: true,
	}

	expectedPath := testFeaturePath

	deps.BranchResolver.On("ResolveURL", "https://github.com/owner/repo/tree/feature-branch").Return(expectedURLInfo, nil)
	deps.BranchResolver.On("ResolveBranch", "feature-branch", testMainBranch, true).Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "feature-branch", "").Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "feature-branch", expectedPath, mock.MatchedBy(func(opts WorktreeOptions) bool {
		return opts.BaseBranch == testMainBranch && opts.TrackRemote
	}), mock.AnythingOfType("ProgressCallback")).Return(nil)

	result, err := deps.Service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.True(t, result.WasCreated)
	assert.Equal(t, testMainBranch, result.BaseBranch)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_Create_RemoteBranchInput(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName:   "origin/feature-branch",
		WorktreePath: "",
		BaseBranch:   testMainBranch,
		CopyFiles:    false,
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   true,
		IsRemote: true,
	}

	expectedPath := testFeaturePath

	deps.BranchResolver.On("RemoteExists", "origin").Return(true)
	deps.BranchResolver.On("ResolveRemoteBranch", "origin/feature-branch").Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "feature-branch", "").Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "feature-branch", expectedPath, mock.MatchedBy(func(opts WorktreeOptions) bool {
		return opts.BaseBranch == testMainBranch && opts.TrackRemote
	}), mock.AnythingOfType("ProgressCallback")).Return(nil)

	result, err := deps.Service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.False(t, result.WasCreated) // Branch already existed
	assert.Equal(t, testMainBranch, result.BaseBranch)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

// ============================================================================
// Branch Creation Tests
// ============================================================================

func TestCreateService_CreateBranch_ExistingBranch(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "existing-branch",
		BaseBranch: testMainBranch,
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "existing-branch",
		Exists:   true,
		IsRemote: false,
	}

	expectedPath := testutils.NormalizePath("/path/to/worktree/existing-branch")

	deps.BranchResolver.On("ResolveBranch", "existing-branch", testMainBranch, true).Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "existing-branch", "").Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "existing-branch", expectedPath, mock.MatchedBy(func(opts WorktreeOptions) bool {
		return opts.BaseBranch == testMainBranch
	}), mock.AnythingOfType("ProgressCallback")).Return(nil)

	result, err := deps.Service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "existing-branch", result.BranchName)
	assert.False(t, result.WasCreated) // Branch already existed

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_CreateBranch_InvalidName(t *testing.T) {
	deps := setupCreateServiceTest(t)

	testCases := []struct {
		name       string
		branchName string
		wantErr    string
	}{
		{
			name:       "contains spaces",
			branchName: "feature branch",
			wantErr:    "invalid branch name",
		},
		{
			name:       "starts with hyphen",
			branchName: "-feature",
			wantErr:    "invalid branch name",
		},
		{
			name:       "contains double dots",
			branchName: "feature..branch",
			wantErr:    "invalid branch name",
		},
		{
			name:       "ends with slash",
			branchName: "feature/",
			wantErr:    "invalid branch name",
		},
		{
			name:       "contains backslash",
			branchName: "feature\\branch",
			wantErr:    "invalid branch name",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			options := &CreateOptions{
				BranchName: tc.branchName,
				BaseBranch: testMainBranch,
			}

			deps.BranchResolver.On("ResolveBranch", tc.branchName, testMainBranch, true).
				Return(nil, fmt.Errorf("%s", tc.wantErr)).Once()

			result, err := deps.Service.Create(options)

			assert.Nil(t, result)
			assertGroveError(t, err, errors.ErrCodeGitOperation)
		})
	}

	deps.BranchResolver.AssertExpectations(t)
}

func TestCreateService_CreateBranch_ReservedNames(t *testing.T) {
	deps := setupCreateServiceTest(t)

	reservedNames := []string{
		"HEAD",
		"FETCH_HEAD",
		"ORIG_HEAD",
		"MERGE_HEAD",
		"CHERRY_PICK_HEAD",
	}

	for _, name := range reservedNames {
		t.Run(name, func(t *testing.T) {
			options := &CreateOptions{
				BranchName: name,
				BaseBranch: testMainBranch,
			}

			deps.BranchResolver.On("ResolveBranch", name, testMainBranch, true).
				Return(nil, fmt.Errorf("reserved name")).Once()

			result, err := deps.Service.Create(options)

			assert.Nil(t, result)
			require.Error(t, err)
		})
	}

	deps.BranchResolver.AssertExpectations(t)
}

func TestCreateService_CreateBranch_UnicodeNames(t *testing.T) {
	deps := setupCreateServiceTest(t)

	unicodeNames := []string{
		"feature-日本語",
		"функция-branch",
		"功能-分支",
		"특징-브랜치",
	}

	for _, name := range unicodeNames {
		t.Run(name, func(t *testing.T) {
			options := &CreateOptions{
				BranchName: name,
				BaseBranch: testMainBranch,
			}

			expectedBranchInfo := &BranchInfo{
				Name:     name,
				Exists:   false,
				IsRemote: false,
			}

			expectedPath := testutils.NormalizePath(fmt.Sprintf("/path/to/worktree/%s", name))

			deps.BranchResolver.On("ResolveBranch", name, testMainBranch, true).
				Return(expectedBranchInfo, nil).Once()
			deps.PathGenerator.On("GeneratePath", name, "").
				Return(expectedPath, nil).Once()
			deps.WorktreeCreator.On("CreateWorktreeWithProgress", name, expectedPath,
				mock.Anything, mock.AnythingOfType("ProgressCallback")).
				Return(nil).Once()

			result, err := deps.Service.Create(options)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, name, result.BranchName)
		})
	}

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_CreateBranch_LongNames(t *testing.T) {
	deps := setupCreateServiceTest(t)

	// Test branch name length limits
	const (
		maxBranchNameLength = 255
		branchPrefix        = "feature/"
	)

	testCases := []struct {
		name       string
		branchName string
		shouldFail bool
	}{
		{
			name:       "exactly 255 chars",
			branchName: branchPrefix + strings.Repeat("a", maxBranchNameLength-len(branchPrefix)),
			shouldFail: false,
		},
		{
			name:       "over 255 chars",
			branchName: branchPrefix + strings.Repeat("a", maxBranchNameLength-len(branchPrefix)+1),
			shouldFail: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			options := &CreateOptions{
				BranchName: tc.branchName,
				BaseBranch: testMainBranch,
			}

			if tc.shouldFail {
				// Mock RemoteExists check for branch names with slashes
				mockRemoteExistsForSlashBranch(deps, tc.branchName)

				deps.BranchResolver.On("ResolveBranch", tc.branchName, testMainBranch, true).
					Return(nil, fmt.Errorf("name too long")).Once()

				result, err := deps.Service.Create(options)

				assert.Nil(t, result)
				require.Error(t, err)
			} else {
				expectedBranchInfo := &BranchInfo{
					Name:     tc.branchName,
					Exists:   false,
					IsRemote: false,
				}

				expectedPath := testutils.NormalizePath("/path/to/worktree/longname")

				// Mock RemoteExists check for branch names with slashes
				mockRemoteExistsForSlashBranch(deps, tc.branchName)

				deps.BranchResolver.On("ResolveBranch", tc.branchName, testMainBranch, true).
					Return(expectedBranchInfo, nil).Once()
				deps.PathGenerator.On("GeneratePath", tc.branchName, "").
					Return(expectedPath, nil).Once()
				deps.WorktreeCreator.On("CreateWorktreeWithProgress", tc.branchName, expectedPath,
					mock.Anything, mock.AnythingOfType("ProgressCallback")).
					Return(nil).Once()

				result, err := deps.Service.Create(options)

				require.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

// ============================================================================
// Remote Repository Tests
// ============================================================================

func TestCreateService_CloneRepository_Success(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "https://github.com/owner/repo",
		BaseBranch: testMainBranch,
	}

	expectedURLInfo := &URLBranchInfo{
		BranchName: testMainBranch,
		RepoURL:    "https://github.com/owner/repo",
	}

	expectedBranchInfo := &BranchInfo{
		Name:     testMainBranch,
		Exists:   false,
		IsRemote: true,
	}

	expectedPath := testRepoPath

	deps.BranchResolver.On("ResolveURL", "https://github.com/owner/repo").
		Return(expectedURLInfo, nil)
	deps.BranchResolver.On("ResolveBranch", testMainBranch, testMainBranch, true).
		Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", testMainBranch, "").
		Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", testMainBranch, expectedPath,
		mock.MatchedBy(func(opts WorktreeOptions) bool {
			// Note: CloneURL would need to be added to WorktreeOptions for full clone support
			return opts.TrackRemote
		}), mock.AnythingOfType("ProgressCallback")).
		Return(nil)

	result, err := deps.Service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, expectedPath, result.WorktreePath)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_CloneRepository_InvalidURL_NotAURL(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "not-a-url",
		BaseBranch: testMainBranch,
	}

	// Since it doesn't look like a URL, it should be treated as a branch name
	deps.BranchResolver.On("ResolveBranch", "not-a-url", testMainBranch, true).
		Return(nil, fmt.Errorf("invalid branch name")).Once()

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	assertGroveError(t, err, errors.ErrCodeGitOperation)

	deps.BranchResolver.AssertExpectations(t)
}

func TestCreateService_CloneRepository_InvalidURL_UnsupportedScheme(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "ftp://invalid.com/repo",
		BaseBranch: testMainBranch,
	}

	// Mock the URL resolution to fail with unsupported scheme
	deps.BranchResolver.On("ResolveURL", "ftp://invalid.com/repo").
		Return(nil, fmt.Errorf("unsupported scheme")).Once()

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	assertGroveError(t, err, errors.ErrCodeConfigInvalid)

	deps.BranchResolver.AssertExpectations(t)
}

func TestCreateService_CloneRepository_InvalidURL_MalformedURL(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "://missing-scheme.com/repo",
		BaseBranch: testMainBranch,
	}

	// Mock the URL resolution to fail with malformed URL
	deps.BranchResolver.On("ResolveURL", "://missing-scheme.com/repo").
		Return(nil, fmt.Errorf("malformed URL")).Once()

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	assertGroveError(t, err, errors.ErrCodeConfigInvalid)

	deps.BranchResolver.AssertExpectations(t)
}

func TestCreateService_CloneRepository_AuthenticationFailed(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "https://github.com/private/repo",
		BaseBranch: testMainBranch,
	}

	expectedURLInfo := &URLBranchInfo{
		BranchName: testMainBranch,
		RepoURL:    "https://github.com/private/repo",
	}

	expectedBranchInfo := &BranchInfo{
		Name:     testMainBranch,
		Exists:   false,
		IsRemote: true,
	}

	expectedPath := testRepoPath

	deps.BranchResolver.On("ResolveURL", "https://github.com/private/repo").
		Return(expectedURLInfo, nil)
	deps.BranchResolver.On("ResolveBranch", testMainBranch, testMainBranch, true).
		Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", testMainBranch, "").
		Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", testMainBranch, expectedPath,
		mock.Anything, mock.AnythingOfType("ProgressCallback")).
		Return(fmt.Errorf("authentication required"))

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	assertGroveError(t, err, errors.ErrCodeGitOperation)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_CloneRepository_NetworkError(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "https://github.com/owner/repo",
		BaseBranch: testMainBranch,
	}

	expectedURLInfo := &URLBranchInfo{
		BranchName: testMainBranch,
		RepoURL:    "https://github.com/owner/repo",
	}

	expectedBranchInfo := &BranchInfo{
		Name:     testMainBranch,
		Exists:   false,
		IsRemote: true,
	}

	expectedPath := testRepoPath

	deps.BranchResolver.On("ResolveURL", "https://github.com/owner/repo").
		Return(expectedURLInfo, nil)
	deps.BranchResolver.On("ResolveBranch", testMainBranch, testMainBranch, true).
		Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", testMainBranch, "").
		Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", testMainBranch, expectedPath,
		mock.Anything, mock.AnythingOfType("ProgressCallback")).
		Return(fmt.Errorf("network unreachable"))

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create worktree")

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

func TestCreateService_CloneRepository_LargeRepository(t *testing.T) {
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "https://github.com/torvalds/linux",
		BaseBranch: testMainBranch,
		// Note: Shallow clone option would need to be added to CreateOptions for optimization
	}

	expectedURLInfo := &URLBranchInfo{
		BranchName: "master",
		RepoURL:    "https://github.com/torvalds/linux",
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "master",
		Exists:   false,
		IsRemote: true,
	}

	expectedPath := testutils.NormalizePath("/path/to/worktree/linux")

	deps.BranchResolver.On("ResolveURL", "https://github.com/torvalds/linux").
		Return(expectedURLInfo, nil)
	deps.BranchResolver.On("ResolveBranch", "master", testMainBranch, true).
		Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "master", "").
		Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "master", expectedPath,
		mock.MatchedBy(func(opts WorktreeOptions) bool {
			// Would check for shallow clone option if it existed
			return opts.TrackRemote
		}), mock.AnythingOfType("ProgressCallback")).
		Return(nil)

	result, err := deps.Service.Create(options)

	require.NoError(t, err)
	assert.NotNil(t, result)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
}

// ============================================================================
// Non-Critical Failure Tests
// ============================================================================

func TestCreateService_Create_FileCopyFailureIsNonCritical(t *testing.T) {
	// Test verifies that worktree creation succeeds even when optional file copying fails
	deps := setupCreateServiceTest(t)

	options := &CreateOptions{
		BranchName: "feature-branch",
		BaseBranch: testMainBranch,
		CopyFiles:  true,
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature-branch",
		Exists:   false,
		IsRemote: false,
	}

	expectedPath := testFeaturePath

	deps.BranchResolver.On("ResolveBranch", "feature-branch", testMainBranch, true).
		Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("GeneratePath", "feature-branch", "").
		Return(expectedPath, nil)
	deps.WorktreeCreator.On("CreateWorktreeWithProgress", "feature-branch", expectedPath,
		mock.Anything, mock.AnythingOfType("ProgressCallback")).
		Return(nil)

	// File copying fails but this is non-critical
	// First try to find by base branch
	deps.FileManager.On("FindWorktreeByBranch", testMainBranch).
		Return("", fmt.Errorf("base branch worktree not found")).Once()
	// Then try to discover current source worktree
	deps.FileManager.On("DiscoverSourceWorktree").
		Return("", fmt.Errorf("source worktree not found")).Once()

	result, err := deps.Service.Create(options)

	// Should succeed even if file copying fails (non-critical)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.CopiedFiles)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.Equal(t, expectedPath, result.WorktreePath)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
	deps.WorktreeCreator.AssertExpectations(t)
	deps.FileManager.AssertExpectations(t)
}

// ============================================================================
// Validation Tests
// ============================================================================

func TestCreateService_ValidateInput_RequiredFields(t *testing.T) {
	deps := setupCreateServiceTest(t)

	testCases := []struct {
		name    string
		options *CreateOptions
		wantErr string
	}{
		{
			name:    "nil options",
			options: nil,
			wantErr: "options cannot be nil",
		},
		{
			name: "empty branch name",
			options: &CreateOptions{
				BranchName: "",
				BaseBranch: testMainBranch,
			},
			wantErr: "branch name cannot be empty",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := deps.Service.Create(tc.options)

			assert.Nil(t, result)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func TestCreateService_ValidateInput_PathTraversal(t *testing.T) {
	deps := setupCreateServiceTest(t)

	pathTraversalAttempts := []string{
		"../../../etc/passwd",
		"..\\..\\windows\\system32",
		"./../../sensitive",
		"path/../../../etc",
	}

	for _, path := range pathTraversalAttempts {
		t.Run(path, func(t *testing.T) {
			options := &CreateOptions{
				BranchName:   "feature",
				WorktreePath: path,
				BaseBranch:   testMainBranch,
			}

			result, err := deps.Service.Create(options)

			assert.Nil(t, result)
			assertGroveError(t, err, errors.ErrCodeConfigInvalid)
			assert.Contains(t, err.Error(), "path cannot contain '..' components")
		})
	}
}

func TestCreateService_ValidateInput_SpecialCharacters(t *testing.T) {
	deps := setupCreateServiceTest(t)

	specialCharBranches := []string{
		"feature*branch",
		"feature?branch",
		"feature[branch]",
		"feature~branch",
		"feature^branch",
		"feature:branch",
	}

	for _, name := range specialCharBranches {
		t.Run(name, func(t *testing.T) {
			options := &CreateOptions{
				BranchName: name,
				BaseBranch: testMainBranch,
			}

			deps.BranchResolver.On("ResolveBranch", name, testMainBranch, true).
				Return(nil, fmt.Errorf("invalid characters")).Once()

			result, err := deps.Service.Create(options)

			assert.Nil(t, result)
			require.Error(t, err)
		})
	}

	deps.BranchResolver.AssertExpectations(t)
}

func TestCreateService_ValidateInput_MaxLengths(t *testing.T) {
	deps := setupCreateServiceTest(t)

	// Test maximum path length
	const typicalPathMax = 4096
	veryLongPath := strings.Repeat("a", typicalPathMax+1) // Over typical PATH_MAX

	options := &CreateOptions{
		BranchName:   "feature",
		WorktreePath: veryLongPath,
		BaseBranch:   testMainBranch,
	}

	expectedBranchInfo := &BranchInfo{
		Name:     "feature",
		Exists:   false,
		IsRemote: false,
	}

	deps.BranchResolver.On("ResolveBranch", "feature", testMainBranch, true).
		Return(expectedBranchInfo, nil)
	deps.PathGenerator.On("ResolveUserPath", veryLongPath).
		Return("", fmt.Errorf("path too long"))

	result, err := deps.Service.Create(options)

	assert.Nil(t, result)
	require.Error(t, err)

	deps.BranchResolver.AssertExpectations(t)
	deps.PathGenerator.AssertExpectations(t)
}
