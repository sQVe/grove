//go:build !integration
// +build !integration

package create

import (
	"testing"

	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testWorktreePath = "/repo/worktrees/feature-branch"
	testSourcePath   = "/repo/main"
)

// Mock implementations for testing.
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

type MockPathGenerator struct {
	mock.Mock
}

func (m *MockPathGenerator) GeneratePath(branchName, basePath string) (string, error) {
	args := m.Called(branchName, basePath)
	return args.String(0), args.Error(1)
}

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

func (m *MockFileManager) GetCurrentWorktreePath() (string, error) {
	args := m.Called()
	return args.String(0), args.Error(1)
}

func (m *MockFileManager) ResolveConflicts(conflicts []FileConflict, strategy ConflictStrategy) error {
	args := m.Called(conflicts, strategy)
	return args.Error(0)
}

func (m *MockFileManager) FindWorktreeByBranch(branchName string) (string, error) {
	args := m.Called(branchName)
	return args.String(0), args.Error(1)
}

func TestCreateServiceImpl_Create_Success(t *testing.T) {
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	// Set up test data.
	options := &CreateOptions{
		BranchName:   "feature-branch",
		WorktreePath: "",
		CopyFiles:    true,
		CopyPatterns: []string{".env*", "*.local"}, // Provide patterns so CopyFiles is called.
	}

	branchInfo := &BranchInfo{
		Name:   "feature-branch",
		Exists: true,
	}

	generatedPath := testWorktreePath
	sourceWorktree := testSourcePath

	// Set up mock expectations.
	mockBranchResolver.On("ResolveBranch", "feature-branch", "", false).Return(branchInfo, nil)
	mockPathGenerator.On("GeneratePath", "feature-branch", "").Return(generatedPath, nil)
	mockWorktreeCreator.On("CreateWorktree", "feature-branch", generatedPath, WorktreeOptions{}).Return(nil)
	// When CopyFiles is true, the service calls handleFileCopying which calls DiscoverSourceWorktree and CopyFiles.
	mockFileManager.On("DiscoverSourceWorktree").Return(sourceWorktree, nil)
	mockFileManager.On("CopyFiles", sourceWorktree, generatedPath, []string{".env*", "*.local"}, CopyOptions{
		ConflictStrategy: ConflictPrompt,
		DryRun:           false,
	}).Return(nil)

	// Execute the test.
	result, err := service.Create(options)

	// Verify results.
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, generatedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)
	assert.False(t, result.WasCreated)

	// Verify all mock expectations were met.
	mockBranchResolver.AssertExpectations(t)
	mockPathGenerator.AssertExpectations(t)
	mockWorktreeCreator.AssertExpectations(t)
	mockFileManager.AssertExpectations(t)
}

func TestCreateServiceImpl_Create_BranchResolutionFailure(t *testing.T) {
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	options := &CreateOptions{
		BranchName: "nonexistent-branch",
	}

	expectedError := &errors.GroveError{
		Code:    errors.ErrCodeBranchNotFound,
		Message: "branch not found",
	}

	// Set up mock expectations.
	mockBranchResolver.On("ResolveBranch", "nonexistent-branch", "", false).Return(nil, expectedError)

	// Execute the test.
	result, err := service.Create(options)

	// Verify results.
	assert.Nil(t, result)
	require.Error(t, err)
	assert.IsType(t, &errors.GroveError{}, err)

	// Verify mock expectations.
	mockBranchResolver.AssertExpectations(t)
	mockPathGenerator.AssertNotCalled(t, "GeneratePath")
	mockWorktreeCreator.AssertNotCalled(t, "CreateWorktree")
}

func TestCreateServiceImpl_Create_PathGenerationFailure(t *testing.T) {
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	options := &CreateOptions{
		BranchName: "feature-branch",
	}

	branchInfo := &BranchInfo{
		Name:   "feature-branch",
		Exists: true,
	}

	pathError := &errors.GroveError{
		Code:    errors.ErrCodeFileSystem,
		Message: "failed to generate path",
	}

	// Set up mock expectations.
	mockBranchResolver.On("ResolveBranch", "feature-branch", "", false).Return(branchInfo, nil)
	mockPathGenerator.On("GeneratePath", "feature-branch", "").Return("", pathError)

	// Execute the test.
	result, err := service.Create(options)

	// Verify results.
	assert.Nil(t, result)
	require.Error(t, err)
	assert.IsType(t, &errors.GroveError{}, err)

	// Verify mock expectations.
	mockBranchResolver.AssertExpectations(t)
	mockPathGenerator.AssertExpectations(t)
	mockWorktreeCreator.AssertNotCalled(t, "CreateWorktree")
}

func TestCreateServiceImpl_Create_WorktreeCreationFailure(t *testing.T) {
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	options := &CreateOptions{
		BranchName: "feature-branch",
	}

	branchInfo := &BranchInfo{
		Name:   "feature-branch",
		Exists: true,
	}

	generatedPath := testWorktreePath
	worktreeError := &errors.GroveError{
		Code:    errors.ErrCodeWorktreeCreation,
		Message: "failed to create worktree",
	}

	// Set up mock expectations.
	mockBranchResolver.On("ResolveBranch", "feature-branch", "", false).Return(branchInfo, nil)
	mockPathGenerator.On("GeneratePath", "feature-branch", "").Return(generatedPath, nil)
	mockWorktreeCreator.On("CreateWorktree", "feature-branch", generatedPath, WorktreeOptions{}).Return(worktreeError)

	// Execute the test.
	result, err := service.Create(options)

	// Verify results.
	assert.Nil(t, result)
	require.Error(t, err)
	assert.IsType(t, &errors.GroveError{}, err)

	// Verify mock expectations.
	mockBranchResolver.AssertExpectations(t)
	mockPathGenerator.AssertExpectations(t)
	mockWorktreeCreator.AssertExpectations(t)
	mockFileManager.AssertNotCalled(t, "CopyFiles")
}

func TestCreateServiceImpl_Create_FileCopyingSuccess(t *testing.T) {
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	options := &CreateOptions{
		BranchName:   "feature-branch",
		CopyFiles:    true,
		CopyPatterns: []string{".env*", ".vscode/"},
	}

	branchInfo := &BranchInfo{
		Name:   "feature-branch",
		Exists: true,
	}

	generatedPath := testWorktreePath
	sourceWorktree := testSourcePath

	// Set up mock expectations.
	mockBranchResolver.On("ResolveBranch", "feature-branch", "", false).Return(branchInfo, nil)
	mockPathGenerator.On("GeneratePath", "feature-branch", "").Return(generatedPath, nil)
	mockWorktreeCreator.On("CreateWorktree", "feature-branch", generatedPath, WorktreeOptions{}).Return(nil)
	mockFileManager.On("DiscoverSourceWorktree").Return(sourceWorktree, nil)
	mockFileManager.On("CopyFiles", sourceWorktree, generatedPath, []string{".env*", ".vscode/"}, CopyOptions{
		ConflictStrategy: ConflictPrompt,
		DryRun:           false,
	}).Return(nil)

	// Execute the test.
	result, err := service.Create(options)

	// Verify results.
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, generatedPath, result.WorktreePath)

	// Verify all mock expectations were met.
	mockBranchResolver.AssertExpectations(t)
	mockPathGenerator.AssertExpectations(t)
	mockWorktreeCreator.AssertExpectations(t)
	mockFileManager.AssertExpectations(t)
}

func TestCreateServiceImpl_Create_FileCopyingFailure(t *testing.T) {
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	options := &CreateOptions{
		BranchName:   "feature-branch",
		CopyFiles:    true,
		CopyPatterns: []string{".env*", "*.local"}, // Provide patterns so CopyFiles is called.
	}

	branchInfo := &BranchInfo{
		Name:   "feature-branch",
		Exists: true,
	}

	generatedPath := testWorktreePath
	sourceWorktree := testSourcePath
	copyError := &errors.GroveError{
		Code:    errors.ErrCodeFileCopyFailed,
		Message: "failed to copy files",
	}

	// Set up mock expectations.
	mockBranchResolver.On("ResolveBranch", "feature-branch", "", false).Return(branchInfo, nil)
	mockPathGenerator.On("GeneratePath", "feature-branch", "").Return(generatedPath, nil)
	mockWorktreeCreator.On("CreateWorktree", "feature-branch", generatedPath, WorktreeOptions{}).Return(nil)
	mockFileManager.On("DiscoverSourceWorktree").Return(sourceWorktree, nil)
	mockFileManager.On("CopyFiles", sourceWorktree, generatedPath, []string{".env*", "*.local"}, CopyOptions{
		ConflictStrategy: ConflictPrompt,
		DryRun:           false,
	}).Return(copyError)

	// Execute the test.
	result, err := service.Create(options)

	// Verify results - service should still succeed even if file copying fails.
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, generatedPath, result.WorktreePath)

	// Verify all mock expectations were met.
	mockBranchResolver.AssertExpectations(t)
	mockPathGenerator.AssertExpectations(t)
	mockWorktreeCreator.AssertExpectations(t)
	mockFileManager.AssertExpectations(t)
}

func TestCreateServiceImpl_Create_URLInput(t *testing.T) {
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	options := &CreateOptions{
		BranchName: "https://github.com/owner/repo/pull/123",
	}

	urlInfo := &URLBranchInfo{
		BranchName: "feature-branch",
		Platform:   "github",
	}

	branchInfo := &BranchInfo{
		Name:   "feature-branch",
		Exists: false,
	}

	generatedPath := testWorktreePath

	// Set up mock expectations for URL resolution.
	mockBranchResolver.On("ResolveURL", "https://github.com/owner/repo/pull/123").Return(urlInfo, nil)
	mockBranchResolver.On("ResolveBranch", "feature-branch", "", false).Return(branchInfo, nil)
	mockPathGenerator.On("GeneratePath", "feature-branch", "").Return(generatedPath, nil)
	mockWorktreeCreator.On("CreateWorktree", "feature-branch", generatedPath, WorktreeOptions{}).Return(nil)

	// Execute the test.
	result, err := service.Create(options)

	// Verify results.
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, generatedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)

	// Verify all mock expectations were met.
	mockBranchResolver.AssertExpectations(t)
	mockPathGenerator.AssertExpectations(t)
	mockWorktreeCreator.AssertExpectations(t)
}

func TestCreateServiceImpl_Create_RemoteBranchInput(t *testing.T) {
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	options := &CreateOptions{
		BranchName: "origin/feature-branch",
	}

	branchInfo := &BranchInfo{
		Name:           "feature-branch",
		Exists:         true,
		IsRemote:       true,
		RemoteName:     "origin",
		TrackingBranch: "origin/feature-branch",
	}

	generatedPath := testWorktreePath

	// Set up mock expectations for remote branch resolution.
	mockBranchResolver.On("ResolveRemoteBranch", "origin/feature-branch").Return(branchInfo, nil)
	mockPathGenerator.On("GeneratePath", "feature-branch", "").Return(generatedPath, nil)
	mockWorktreeCreator.On("CreateWorktree", "feature-branch", generatedPath, WorktreeOptions{TrackRemote: true}).Return(nil)

	// Execute the test.
	result, err := service.Create(options)

	// Verify results.
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, generatedPath, result.WorktreePath)
	assert.Equal(t, "feature-branch", result.BranchName)

	// Verify all mock expectations were met.
	mockBranchResolver.AssertExpectations(t)
	mockPathGenerator.AssertExpectations(t)
	mockWorktreeCreator.AssertExpectations(t)
}

func TestCreateServiceImpl_validateOptions(t *testing.T) {
	service := &CreateServiceImpl{
		logger: logger.WithComponent("test"),
	}

	tests := []struct {
		name        string
		options     *CreateOptions
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid options",
			options: &CreateOptions{
				BranchName: "feature-branch",
			},
			expectError: false,
		},
		{
			name: "empty branch name",
			options: &CreateOptions{
				BranchName: "",
			},
			expectError: true,
			errorMsg:    "branch name cannot be empty",
		},
		{
			name: "whitespace only branch name",
			options: &CreateOptions{
				BranchName: "   ",
			},
			expectError: true,
			errorMsg:    "branch name cannot be empty",
		},
		{
			name:        "nil options",
			options:     nil,
			expectError: true,
			errorMsg:    "options cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateOptions(tt.options)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCreateServiceImpl_classifyInput(t *testing.T) {
	// Mock dependencies for the service.
	mockBranchResolver := new(MockBranchResolver)
	mockPathGenerator := new(MockPathGenerator)
	mockWorktreeCreator := new(MockWorktreeCreator)
	mockFileManager := new(MockFileManager)

	service := NewCreateService(mockBranchResolver, mockPathGenerator, mockWorktreeCreator, mockFileManager)

	tests := []struct {
		name         string
		input        string
		expectedType InputType
		setupMocks   func()
		expectError  bool
	}{
		{
			name:         "regular branch name",
			input:        "feature-branch",
			expectedType: InputTypeBranch,
			setupMocks:   func() {}, // No mocks needed for simple branch names.
			expectError:  false,
		},
		{
			name:         "GitHub PR URL",
			input:        "https://github.com/owner/repo/pull/123",
			expectedType: InputTypeURL,
			setupMocks: func() {
				urlInfo := &URLBranchInfo{
					BranchName: "pr-123",
					Platform:   "github",
				}
				mockBranchResolver.On("ResolveURL", "https://github.com/owner/repo/pull/123").Return(urlInfo, nil)
			},
			expectError: false,
		},
		{
			name:         "remote branch reference",
			input:        "origin/feature-branch",
			expectedType: InputTypeRemoteBranch,
			setupMocks:   func() {}, // No mocks needed for simple remote branch parsing.
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset mocks.
			mockBranchResolver.ExpectedCalls = nil

			// Setup test-specific mocks.
			tt.setupMocks()

			result, err := service.classifyInput(tt.input)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, result)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedType, result.Type)
				assert.Equal(t, tt.input, result.OriginalName)
			}

			// Verify mock expectations.
			mockBranchResolver.AssertExpectations(t)
		})
	}
}
