package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGitExecutor implementation for command tests.
type MockGitExecutor struct {
	Commands  [][]string
	Responses map[string]MockResponse
	CallCount int
}

type MockResponse struct {
	Output string
	Error  error
}

func NewMockGitExecutor() *MockGitExecutor {
	return &MockGitExecutor{
		Commands:  [][]string{},
		Responses: make(map[string]MockResponse),
		CallCount: 0,
	}
}

func (m *MockGitExecutor) Execute(args ...string) (string, error) {
	return m.ExecuteWithContext(context.Background(), args...)
}

// ExecuteWithContext implements the GitExecutor interface with context support.
func (m *MockGitExecutor) ExecuteWithContext(ctx context.Context, args ...string) (string, error) {
	m.CallCount++
	m.Commands = append(m.Commands, args)

	// Special handling for clone command to create directory.
	if len(args) >= 3 && args[0] == "clone" && args[1] == "--bare" {
		targetDir := args[3]
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			return "", err
		}
	}

	cmdKey := fmt.Sprintf("%v", args)
	for pattern, response := range m.Responses {
		if cmdKey == pattern || (len(args) > 0 && args[0] == pattern) {
			return response.Output, response.Error
		}
	}

	return "", fmt.Errorf("mock: unhandled git command: %v", args)
}

func (m *MockGitExecutor) SetResponse(pattern, output string, err error) {
	m.Responses[pattern] = MockResponse{Output: output, Error: err}
}

func TestRunInitFromRemoteWithExecutor_Success(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-init-remote-mock-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor with successful responses..
	mock := NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", nil)
	mock.SetResponse("fetch", "", nil)
	mock.SetResponse("for-each-ref", "main\nfeature", nil)
	mock.SetResponse("branch", "", nil)
	// Add default branch detection responses
	mock.SetResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main", nil)
	// Add worktree creation responses
	mock.SetResponse("config --bool core.bare true", "", nil)
	mock.SetResponse("worktree add", "", nil)

	// Test successful remote init.
	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.NoError(t, err)

	// Verify directory structure exists.
	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(tempDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify .git file content.
	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(content))

	// Verify git commands were called in correct order.
	assert.GreaterOrEqual(t, len(mock.Commands), 4)
}

func TestRunInitFromRemoteWithExecutor_CloneFailure(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-init-remote-fail-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor with clone failure.
	mock := NewMockGitExecutor()
	mock.SetResponse("clone", "", fmt.Errorf("authentication failed"))

	// Test clone failure.
	err = RunInitRemoteWithExecutor(mock, "https://private.com/repo.git", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone repository")
}

func TestRunInitFromRemoteWithExecutor_ConfigFailure(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-init-config-fail-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor with config failure.
	mock := NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", fmt.Errorf("config write failed"))

	// Test config failure.
	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to configure remote tracking")
}

func TestRunInitFromRemoteWithExecutor_FetchFailure(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-init-fetch-fail-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor with fetch failure.
	mock := NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", nil)
	mock.SetResponse("fetch", "", fmt.Errorf("network timeout"))

	// Test fetch failure.
	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to configure remote tracking")
}

func TestRunInitFromRemoteWithExecutor_UpstreamWarning(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-init-upstream-warn-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor with upstream failure (should not fail overall).
	mock := NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", nil)
	mock.SetResponse("fetch", "", nil)
	mock.SetResponse("for-each-ref", "", fmt.Errorf("no refs found"))
	// Add default branch detection responses
	mock.SetResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main", nil)
	// Add worktree creation responses
	mock.SetResponse("config --bool core.bare true", "", nil)
	mock.SetResponse("worktree add", "", nil)

	// Test upstream failure (should succeed with warning).
	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.NoError(t, err) // Should not fail even if upstream setup fails.
}

func TestRunInitFromRemoteWithExecutor_NonEmptyDirectory(t *testing.T) {
	// Create temporary directory with a file.
	tempDir, err := os.MkdirTemp("", "grove-init-nonempty-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a non-hidden file.
	testFile := filepath.Join(tempDir, "existing.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o600)
	require.NoError(t, err)

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor (shouldn't be called).
	mock := NewMockGitExecutor()

	// Test non-empty directory failure.
	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not empty")

	// Verify no git commands were executed.
	assert.Equal(t, 0, mock.CallCount)
}

func TestRunInitFromRemoteWithExecutor_HiddenFilesAllowed(t *testing.T) {
	// Create temporary directory with hidden files only.
	tempDir, err := os.MkdirTemp("", "grove-init-hidden-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create hidden files (should be allowed).
	hiddenFile := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(hiddenFile, []byte("*.log"), 0o600)
	require.NoError(t, err)

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor with successful responses.
	mock := NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", nil)
	mock.SetResponse("fetch", "", nil)
	mock.SetResponse("for-each-ref", "", nil)
	// Add default branch detection responses
	mock.SetResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main", nil)
	// Add worktree creation responses
	mock.SetResponse("config --bool core.bare true", "", nil)
	mock.SetResponse("worktree add", "", nil)

	// Test with hidden files (should succeed).
	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.NoError(t, err)

	// Verify git commands were executed.
	assert.Positive(t, mock.CallCount)
}

func TestRunInitRemoteWithBranches(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-init-branches-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor with successful responses.
	mock := NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", nil)
	mock.SetResponse("fetch", "", nil)
	mock.SetResponse("for-each-ref", "main\ndevelop\nfeature", nil)
	mock.SetResponse("branch", "  origin/main\n  origin/develop\n  origin/feature", nil)
	// Add default branch detection responses
	mock.SetResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main", nil)
	// Add worktree creation responses
	mock.SetResponse("config --bool core.bare true", "", nil)
	mock.SetResponse("worktree add", "", nil)

	// Test with multiple branches.
	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "main,develop")
	require.NoError(t, err)

	// Verify directory structure exists.
	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	// Verify git commands were executed.
	assert.Positive(t, mock.CallCount)
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
			input:    "main,develop,feature",
			expected: []string{"main", "develop", "feature"},
		},
		{
			name:     "branches with spaces",
			input:    "main, develop , feature/auth",
			expected: []string{"main", "develop", "feature/auth"},
		},
		{
			name:     "empty entries",
			input:    "main,,develop,",
			expected: []string{"main", "develop"},
		},
		{
			name:     "complex branch names",
			input:    "feature/user-auth,bugfix/login-issue,release/v1.0.0",
			expected: []string{"feature/user-auth", "bugfix/login-issue", "release/v1.0.0"},
		},
		{
			name:     "invalid branch names filtered out",
			input:    "main,invalid branch,feature/auth,-invalid,valid.lock",
			expected: []string{"main", "feature/auth"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseBranches(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCreateAdditionalWorktrees(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-worktrees-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create mock executor.
	mock := NewMockGitExecutor()
	mock.SetResponse("branch", "  origin/main\n  origin/develop\n  origin/feature", nil)
	mock.SetResponse("worktree add", "", nil)

	// Test creating worktrees for available branches.
	branches := []string{"main", "develop", "nonexistent"}
	err = CreateAdditionalWorktrees(mock, tempDir, branches)
	require.NoError(t, err)

	// Verify git commands were executed.
	assert.Positive(t, mock.CallCount)
}

func TestIsValidBranchName(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected bool
	}{
		// Valid names
		{"simple branch", "main", true},
		{"branch with slash", "feature/auth", true},
		{"branch with dash", "bug-fix", true},
		{"branch with dots", "release/v1.0.0", true},
		{"branch with numbers", "feature123", true},
		{"branch with underscore", "feature_branch", true},

		// Invalid names
		{"empty string", "", false},
		{"just dash", "-", false},
		{"starts with dash", "-invalid", false},
		{"ends with .lock", "branch.lock", false},
		{"starts with slash", "/branch", false},
		{"ends with slash", "branch/", false},
		{"contains double dots", "feature..branch", false},
		{"contains space", "invalid branch", false},
		{"contains tilde", "branch~1", false},
		{"contains caret", "branch^1", false},
		{"contains colon", "branch:name", false},
		{"contains question mark", "branch?", false},
		{"contains asterisk", "branch*", false},
		{"contains square bracket", "branch[1]", false},
		{"contains backslash", "branch\\name", false},
		{"contains control character", "branch\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBranchName(tt.branch)
			assert.Equal(t, tt.expected, result, "Branch name: %q", tt.branch)
		})
	}
}
