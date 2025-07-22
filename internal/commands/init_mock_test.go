//go:build !integration
// +build !integration

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sqve/grove/internal/testutils"
)

func TestRunInitFromRemoteWithExecutor_Success(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-remote-mock-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()
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

	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.NoError(t, err)

	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(tempDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify .git file content.
	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(content))

	assert.GreaterOrEqual(t, len(mock.Commands), 4)
}

func TestRunInitFromRemoteWithExecutor_CloneFailure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-remote-fail-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()
	mock.SetResponse("clone", "", fmt.Errorf("authentication failed"))

	err = RunInitRemoteWithExecutor(mock, "https://private.com/repo.git", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone repository")
}

func TestRunInitFromRemoteWithExecutor_ConfigFailure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-config-fail-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", fmt.Errorf("config write failed"))

	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to configure remote tracking")
}

func TestRunInitFromRemoteWithExecutor_FetchFailure(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-fetch-fail-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", nil)
	mock.SetResponse("fetch", "", fmt.Errorf("network timeout"))

	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to configure remote tracking")
}

func TestRunInitFromRemoteWithExecutor_UpstreamWarning(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-upstream-warn-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", nil)
	mock.SetResponse("fetch", "", nil)
	mock.SetResponse("for-each-ref", "", fmt.Errorf("no refs found"))
	// Add default branch detection responses
	mock.SetResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main", nil)
	// Add worktree creation responses
	mock.SetResponse("config --bool core.bare true", "", nil)
	mock.SetResponse("worktree add", "", nil)

	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.NoError(t, err) // Should not fail even if upstream setup fails.
}

func TestRunInitFromRemoteWithExecutor_NonEmptyDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-nonempty-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	testFile := filepath.Join(tempDir, "existing.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o600)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()

	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not empty")

	assert.Equal(t, 0, mock.CallCount)
}

func TestRunInitFromRemoteWithExecutor_HiddenFilesAllowed(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-hidden-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	hiddenFile := filepath.Join(tempDir, ".gitignore")
	err = os.WriteFile(hiddenFile, []byte("*.log"), 0o600)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()
	mock.SetResponse("clone", "", nil)
	mock.SetResponse("config", "", nil)
	mock.SetResponse("fetch", "", nil)
	mock.SetResponse("for-each-ref", "", nil)
	// Add default branch detection responses
	mock.SetResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main", nil)
	// Add worktree creation responses
	mock.SetResponse("config --bool core.bare true", "", nil)
	mock.SetResponse("worktree add", "", nil)

	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "")
	require.NoError(t, err)

	assert.Positive(t, mock.CallCount)
}

func TestRunInitRemoteWithBranches(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-branches-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()
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

	err = RunInitRemoteWithExecutor(mock, "https://github.com/user/repo.git", "main,develop")
	require.NoError(t, err)

	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

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
	tempDir, err := os.MkdirTemp("", "grove-worktrees-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mock := testutils.NewMockGitExecutor()
	mock.SetResponse("branch", "  origin/main\n  origin/develop\n  origin/feature", nil)
	mock.SetResponse("worktree add", "", nil)

	branches := []string{"main", "develop", "nonexistent"}
	err = CreateAdditionalWorktrees(mock, tempDir, branches)
	require.NoError(t, err)

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
