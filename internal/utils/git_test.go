package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type MockGitExecutor struct {
	mock.Mock
}

func (m *MockGitExecutor) Execute(args ...string) (string, error) {
	mockArgs := m.Called(args)
	return mockArgs.String(0), mockArgs.Error(1)
}

func TestIsGitRepository_Success(t *testing.T) {
	mockExecutor := &MockGitExecutor{}

	mockExecutor.On("Execute", []string{"rev-parse", "--git-dir"}).
		Return(".git", nil)

	isRepo, err := IsGitRepository(mockExecutor)

	assert.NoError(t, err)
	assert.True(t, isRepo)
	mockExecutor.AssertExpectations(t)
}

func TestIsGitRepository_NotARepo(t *testing.T) {
	mockExecutor := &MockGitExecutor{}

	mockExecutor.On("Execute", []string{"rev-parse", "--git-dir"}).
		Return("", errors.New("fatal: not a git repository (or any of the parent directories): .git: exit 128"))

	isRepo, err := IsGitRepository(mockExecutor)

	assert.NoError(t, err)
	assert.False(t, isRepo)
	mockExecutor.AssertExpectations(t)
}

func TestIsGitRepository_UnexpectedError(t *testing.T) {
	mockExecutor := &MockGitExecutor{}

	expectedErr := errors.New("unexpected git error")
	mockExecutor.On("Execute", []string{"rev-parse", "--git-dir"}).
		Return("", expectedErr)

	isRepo, err := IsGitRepository(mockExecutor)

	assert.Error(t, err)
	assert.False(t, isRepo)
	assert.Equal(t, expectedErr, err)
	mockExecutor.AssertExpectations(t)
}

func TestGetRepositoryRoot_Success(t *testing.T) {
	mockExecutor := &MockGitExecutor{}
	expectedRoot := "/path/to/repo"

	mockExecutor.On("Execute", []string{"rev-parse", "--show-toplevel"}).
		Return(expectedRoot, nil)

	root, err := GetRepositoryRoot(mockExecutor)

	assert.NoError(t, err)
	assert.Equal(t, expectedRoot, root)
	mockExecutor.AssertExpectations(t)
}

func TestGetRepositoryRoot_WithWhitespace(t *testing.T) {
	mockExecutor := &MockGitExecutor{}
	expectedRoot := "/path/to/repo"

	mockExecutor.On("Execute", []string{"rev-parse", "--show-toplevel"}).
		Return("  "+expectedRoot+"\n", nil)

	root, err := GetRepositoryRoot(mockExecutor)

	assert.NoError(t, err)
	assert.Equal(t, expectedRoot, root)
	mockExecutor.AssertExpectations(t)
}

func TestGetRepositoryRoot_Error(t *testing.T) {
	mockExecutor := &MockGitExecutor{}
	expectedErr := errors.New("not a git repository")

	mockExecutor.On("Execute", []string{"rev-parse", "--show-toplevel"}).
		Return("", expectedErr)

	root, err := GetRepositoryRoot(mockExecutor)

	assert.Error(t, err)
	assert.Equal(t, "", root)
	assert.Equal(t, expectedErr, err)
	mockExecutor.AssertExpectations(t)
}

func TestValidateRepository_Success(t *testing.T) {
	mockExecutor := &MockGitExecutor{}

	// Mock IsGitRepository call
	mockExecutor.On("Execute", []string{"rev-parse", "--git-dir"}).
		Return(".git", nil)

	// Mock commit check call
	mockExecutor.On("Execute", []string{"rev-parse", "HEAD"}).
		Return("abc123", nil)

	err := ValidateRepository(mockExecutor)

	assert.NoError(t, err)
	mockExecutor.AssertExpectations(t)
}

func TestValidateRepository_NotARepo(t *testing.T) {
	mockExecutor := &MockGitExecutor{}

	mockExecutor.On("Execute", []string{"rev-parse", "--git-dir"}).
		Return("", errors.New("fatal: not a git repository: exit 128"))

	err := ValidateRepository(mockExecutor)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
	mockExecutor.AssertExpectations(t)
}

func TestParseGitPlatformURL_GitHub(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedRepo   string
		expectedBranch string
		expectError    bool
	}{
		{
			name:           "basic GitHub repo",
			url:            "https://github.com/user/repo",
			expectedRepo:   "https://github.com/user/repo.git",
			expectedBranch: "",
			expectError:    false,
		},
		{
			name:           "GitHub repo with .git",
			url:            "https://github.com/user/repo.git",
			expectedRepo:   "https://github.com/user/repo.git",
			expectedBranch: "",
			expectError:    false,
		},
		{
			name:           "GitHub with branch",
			url:            "https://github.com/user/repo/tree/feature-branch",
			expectedRepo:   "https://github.com/user/repo.git",
			expectedBranch: "feature-branch",
			expectError:    false,
		},
		{
			name:        "invalid URL",
			url:         "not-a-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParseGitPlatformURL(tt.url)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.expectedRepo, result.RepoURL)
				assert.Equal(t, tt.expectedBranch, result.BranchName)
				assert.Equal(t, "github", result.Platform)
			}
		})
	}
}

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected bool
	}{
		{"HTTPS git URL", "https://github.com/user/repo.git", true},
		{"HTTP git URL", "http://example.com/repo.git", true},
		{"SSH git URL", "git@github.com:user/repo.git", true},
		{"GitHub HTTPS", "https://github.com/user/repo", true},
		{"GitLab HTTPS", "https://gitlab.com/user/repo", true},
		{"SSH full", "ssh://git@example.com/user/repo.git", true},
		{"not a URL", "main", false},
		{"FTP URL", "ftp://example.com/file", false},
		{"empty string", "", false},
		{"local path", "/path/to/repo", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsGitURL(tt.url)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsGitAvailable(t *testing.T) {
	// This test just ensures the function doesn't panic and returns a boolean
	result := IsGitAvailable()
	assert.IsType(t, true, result) // Should return a boolean
}

func TestIsHidden(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"hidden file", ".hidden", true},
		{"hidden dir", ".git", true},
		{"visible file", "README.md", false},
		{"visible dir", "src", false},
		{"empty string", "", false},
		{"just dot", ".", true},    // Current directory (starts with dot)
		{"double dot", "..", true}, // Parent directory (starts with dot)
		{"hidden with extension", ".gitignore", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsHidden(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}
