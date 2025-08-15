package init

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
)

type MockGitExecutor struct {
	mock.Mock
}

func (m *MockGitExecutor) Execute(args ...string) (string, error) {
	mockArgs := m.Called(args)
	return mockArgs.String(0), mockArgs.Error(1)
}

func (m *MockGitExecutor) ExecuteQuiet(args ...string) (string, error) {
	mockArgs := m.Called(args)
	return mockArgs.String(0), mockArgs.Error(1)
}

func (m *MockGitExecutor) ExecuteWithContext(ctx context.Context, args ...string) (string, error) {
	mockArgs := m.Called(ctx, args)
	return mockArgs.String(0), mockArgs.Error(1)
}

func TestExtractRepositoryName(t *testing.T) {
	tests := []struct {
		name     string
		repoURL  string
		expected string
	}{
		{
			name:     "HTTPS URL with .git",
			repoURL:  "https://github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "HTTPS URL without .git",
			repoURL:  "https://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "SSH URL with .git",
			repoURL:  "git@github.com:user/repo.git",
			expected: "repo",
		},
		{
			name:     "SSH URL without .git",
			repoURL:  "git@github.com:user/repo",
			expected: "repo",
		},
		{
			name:     "complex path",
			repoURL:  "https://gitlab.com/group/subgroup/project.git",
			expected: "project",
		},
		{
			name:     "single name",
			repoURL:  "myrepo",
			expected: "myrepo",
		},
		{
			name:     "empty string",
			repoURL:  "",
			expected: "",
		},
		{
			name:     "just .git",
			repoURL:  ".git",
			expected: "",
		},
		{
			name:     "platform url",
			repoURL:  "git@github.com:hups-co/platform.git",
			expected: "platform",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractRepositoryName(tt.repoURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseBranches(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single branch",
			input:    "main",
			expected: []string{"main"},
		},
		{
			name:     "multiple branches",
			input:    "main,develop,feature/auth",
			expected: []string{"main", "develop", "feature/auth"},
		},
		{
			name:     "with spaces",
			input:    "main, develop, feature/auth ",
			expected: []string{"main", "develop", "feature/auth"},
		},
		{
			name:     "with empty segments",
			input:    "main,,develop,",
			expected: []string{"main", "develop"},
		},
		{
			name:     "invalid branch names filtered",
			input:    "main,invalid~branch,develop",
			expected: []string{"main", "develop"},
		},
		{
			name:     "all invalid branches",
			input:    "invalid~branch,another^bad",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseBranches(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidBranchName(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected bool
	}{
		// Valid names
		{"valid simple", "main", true},
		{"valid with slash", "feature/auth", true},
		{"valid with dash", "feature-auth", true},
		{"valid with underscore", "feature_auth", true},
		{"valid alphanumeric", "feature123", true},

		// Invalid names
		{"empty", "", false},
		{"dash only", "-", false},
		{"starts with dash", "-main", false},
		{"ends with .lock", "main.lock", false},
		{"starts with slash", "/main", false},
		{"ends with slash", "main/", false},
		{"consecutive dots", "feature..branch", false},
		{"contains tilde", "feature~branch", false},
		{"contains caret", "feature^branch", false},
		{"contains colon", "feature:branch", false},
		{"contains question mark", "feature?branch", false},
		{"contains asterisk", "feature*branch", false},
		{"contains square bracket", "feature[branch", false},
		{"contains backslash", "feature\\branch", false},
		{"contains space", "feature branch", false},
		{"control character", "feature\x00branch", false},
		{"DEL character", "feature\x7fbranch", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBranchName(tt.branch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateAndPrepareDirectory(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	t.Run("empty directory", func(t *testing.T) {
		tempDir := helper.CreateTempDir("empty")

		// Change to temp directory
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		result, err := validateAndPrepareDirectory()
		assert.NoError(t, err)
		assert.Equal(t, tempDir, result)
	})

	t.Run("directory with hidden files only", func(t *testing.T) {
		tempDir := helper.CreateTempDir("hidden-only")

		// Create hidden files
		hiddenFile := filepath.Join(tempDir, ".hidden")
		err := os.WriteFile(hiddenFile, []byte("content"), 0o644)
		require.NoError(t, err)

		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		result, err := validateAndPrepareDirectory()
		assert.NoError(t, err)
		assert.Equal(t, tempDir, result)
	})

	t.Run("directory with visible files", func(t *testing.T) {
		tempDir := helper.CreateTempDir("with-files")

		// Create visible file
		visibleFile := filepath.Join(tempDir, "README.md")
		err := os.WriteFile(visibleFile, []byte("content"), 0o644)
		require.NoError(t, err)

		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		result, err := validateAndPrepareDirectory()
		assert.Error(t, err)
		assert.Empty(t, result)

		var groveErr *groveErrors.GroveError
		require.True(t, errors.As(err, &groveErr))
		assert.Equal(t, groveErrors.ErrCodeRepoInvalid, groveErr.Code)
		assert.Contains(t, err.Error(), "directory is not empty")
	})
}

func TestCreateAdditionalWorktrees(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	t.Run("empty branches list", func(t *testing.T) {
		mockExecutor := &MockGitExecutor{}
		tempDir := helper.CreateTempDir("test-repo")

		err := CreateAdditionalWorktrees(mockExecutor, tempDir, []string{})
		assert.NoError(t, err)

		mockExecutor.AssertNotCalled(t, "Execute")
	})

	t.Run("successful worktree creation", func(t *testing.T) {
		mockExecutor := &MockGitExecutor{}
		tempDir := helper.CreateTempDir("test-repo")
		branches := []string{"develop", "feature/auth"}

		// Mock branch listing
		remoteBranches := "origin/develop\norigin/feature/auth\norigin/HEAD -> origin/main"
		mockExecutor.On("Execute", []string{"branch", "-r"}).Return(remoteBranches, nil)

		// Mock worktree creation calls
		mockExecutor.On("Execute", mock.MatchedBy(func(args []string) bool {
			return len(args) >= 3 && args[0] == "worktree" && args[1] == "add"
		})).Return("", nil).Times(2)

		// Change to temp directory
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(oldWd) }()

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		err = CreateAdditionalWorktrees(mockExecutor, tempDir, branches)
		assert.NoError(t, err)

		mockExecutor.AssertExpectations(t)
	})
}

func TestRunInitLocal_ExistingGitFile(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tempDir := helper.CreateTempDir("existing-git")

	// Create existing .git file
	gitFile := filepath.Join(tempDir, ".git")
	err := os.WriteFile(gitFile, []byte("gitdir: .bare"), 0o644)
	require.NoError(t, err)

	err = runInitLocal(tempDir)
	assert.Error(t, err)

	var groveErr *groveErrors.GroveError
	require.True(t, errors.As(err, &groveErr))
	assert.Equal(t, groveErrors.ErrCodeRepoExists, groveErr.Code)
}
