//go:build !integration
// +build !integration

package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockGitExecutor_BasicFunctionality(t *testing.T) {
	mock := NewMockGitExecutor()

	_, err := mock.Execute("unknown", "command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unhandled git command")

	assert.Equal(t, 1, mock.CallCount)
	assert.Len(t, mock.Commands, 1)
	assert.Equal(t, []string{"unknown", "command"}, mock.Commands[0])
}

func TestMockGitExecutor_SetResponse(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetResponse("status", "clean", nil)
	output, err := mock.Execute("status")
	require.NoError(t, err)
	assert.Equal(t, "clean", output)

	mock.SetResponse("fail", "output", fmt.Errorf("test error"))
	output, err = mock.Execute("fail")
	require.Error(t, err)
	assert.Equal(t, "output", output)
	assert.Contains(t, err.Error(), "test error")
}

func TestMockGitExecutor_SetSuccessResponse(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetSuccessResponse("status", "clean working tree")
	output, err := mock.Execute("status")
	require.NoError(t, err)
	assert.Equal(t, "clean working tree", output)
}

func TestMockGitExecutor_SetErrorResponse(t *testing.T) {
	mock := NewMockGitExecutor()

	testError := fmt.Errorf("repository not found")
	mock.SetErrorResponse("clone", testError)
	output, err := mock.Execute("clone")
	require.Error(t, err)
	assert.Equal(t, "", output)
	assert.Equal(t, testError, err)
}

func TestMockGitExecutor_SetErrorResponseWithMessage(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetErrorResponseWithMessage("clone", "repository not found")
	output, err := mock.Execute("clone")
	require.Error(t, err)
	assert.Equal(t, "", output)
	assert.Contains(t, err.Error(), "repository not found")
}

func TestMockGitExecutor_SetResponseSlice(t *testing.T) {
	mock := NewMockGitExecutor()

	// Test slice-based response setting (for utils mock compatibility).
	mock.SetResponseSlice([]string{"rev-parse", "--git-dir"}, ".git", nil)
	output, err := mock.Execute("rev-parse", "--git-dir")
	require.NoError(t, err)
	assert.Equal(t, ".git", output)

	mock.SetResponseSlice([]string{"rev-parse", "HEAD"}, "", fmt.Errorf("not a git repository"))
	output, err = mock.Execute("rev-parse", "HEAD")
	require.Error(t, err)
	assert.Equal(t, "", output)
	assert.Contains(t, err.Error(), "not a git repository")
}

func TestMockGitExecutor_SetDelay(t *testing.T) {
	mock := NewMockGitExecutor()

	delay := 10 * time.Millisecond
	mock.SetDelay("clone", delay)
	mock.SetSuccessResponse("clone", "cloned")

	start := time.Now()
	output, err := mock.Execute("clone")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, "cloned", output)
	assert.GreaterOrEqual(t, duration, delay)
}

func TestMockGitExecutor_LastCommand(t *testing.T) {
	mock := NewMockGitExecutor()

	assert.Nil(t, mock.LastCommand())

	mock.SetSuccessResponse("status", "clean")
	mock.SetSuccessResponse("log", "commit history")

	_, _ = mock.Execute("status")
	_, _ = mock.Execute("log", "--oneline")

	lastCmd := mock.LastCommand()
	assert.Equal(t, []string{"log", "--oneline"}, lastCmd)
}

func TestMockGitExecutor_HasCommand(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetSuccessResponse("status", "clean")
	mock.SetSuccessResponse("log", "history")

	_, _ = mock.Execute("status")
	_, _ = mock.Execute("log", "--oneline", "--graph")

	assert.True(t, mock.HasCommand("status"))
	assert.True(t, mock.HasCommand("log", "--oneline", "--graph"))
	assert.False(t, mock.HasCommand("nonexistent"))
	assert.False(t, mock.HasCommand("status", "extra-arg"))
}

func TestMockGitExecutor_ExecuteWithContext(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetSuccessResponse("status", "clean")
	ctx := context.Background()
	output, err := mock.ExecuteWithContext(ctx, "status")
	require.NoError(t, err)
	assert.Equal(t, "clean", output)

	cancelledCtx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = mock.ExecuteWithContext(cancelledCtx, "status")
	require.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}

func TestMockGitExecutor_CloneDirectoryCreation(t *testing.T) {
	mock := NewMockGitExecutor()

	tempDir, err := os.MkdirTemp("", "mock-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	targetDir := filepath.Join(tempDir, "test-repo")
	mock.SetSuccessResponse("clone --bare", "")

	// Test that clone command creates directory.
	_, err = mock.Execute("clone", "--bare", "https://example.com/repo.git", targetDir)
	require.NoError(t, err)

	assert.DirExists(t, targetDir)
}

func TestMockGitExecutor_PatternMatching(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetResponse("branch --set-upstream", "upstream set", nil)

	output, err := mock.Execute("branch", "--set-upstream-to=origin/main", "main")
	require.NoError(t, err)
	assert.Equal(t, "upstream set", output)

	// Test exact match takes precedence.
	mock.SetResponse("branch --set-upstream-to=origin/main main", "exact match", nil)

	output, err = mock.Execute("branch", "--set-upstream-to=origin/main", "main")
	require.NoError(t, err)
	assert.Equal(t, "exact match", output)
}

func TestMockGitExecutor_Reset(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetSuccessResponse("status", "clean")
	mock.SetDelay("clone", 10*time.Millisecond)
	_, _ = mock.Execute("status")
	_, _ = mock.Execute("log")

	assert.Equal(t, 2, mock.CallCount)
	assert.Len(t, mock.Commands, 2)

	mock.Reset()
	assert.Equal(t, 0, mock.CallCount)
	assert.Len(t, mock.Commands, 0)
	assert.Nil(t, mock.LastCommand())

	_, err := mock.Execute("status")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unhandled git command")
}

func TestMockGitExecutor_SetSafeRepositoryState(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetSafeRepositoryState()

	output, err := mock.Execute("status", "--porcelain=v1")
	require.NoError(t, err)
	assert.Equal(t, "", output)

	output, err = mock.Execute("stash", "list")
	require.NoError(t, err)
	assert.Equal(t, "", output)

	output, err = mock.Execute("worktree", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "/path/to/repo")
}

func TestMockGitExecutor_SetUnsafeRepositoryState(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetUnsafeRepositoryState()

	output, err := mock.Execute("status", "--porcelain=v1")
	require.NoError(t, err)
	assert.Contains(t, output, "M file1.txt")

	output, err = mock.Execute("stash", "list")
	require.NoError(t, err)
	assert.Contains(t, output, "stash@{0}")

	output, err = mock.Execute("ls-files", "--others", "--exclude-standard")
	require.NoError(t, err)
	assert.Contains(t, output, "newfile.txt")
}

func TestMockGitExecutor_SetConversionState(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetConversionState()

	output, err := mock.Execute("rev-parse", "--is-bare-repository")
	require.NoError(t, err)
	assert.Equal(t, "false", output)

	output, err = mock.Execute("config", "--get", "core.bare")
	require.NoError(t, err)
	assert.Equal(t, "false", output)

	output, err = mock.Execute("symbolic-ref", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "refs/heads/main", output)
}

func TestMockGitExecutor_MultipleResponseFormats(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetResponse("status", "clean", nil)                            // string-based.
	mock.SetResponseSlice([]string{"rev-parse", "HEAD"}, "abc123", nil) // slice-based.
	mock.SetSuccessResponse("log", "commit history")                    // convenience method.

	output, err := mock.Execute("status")
	require.NoError(t, err)
	assert.Equal(t, "clean", output)

	output, err = mock.Execute("rev-parse", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "abc123", output)

	output, err = mock.Execute("log")
	require.NoError(t, err)
	assert.Equal(t, "commit history", output)
}

func TestMockGitExecutor_CommandTracking(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetSuccessResponse("status", "clean")
	mock.SetSuccessResponse("log", "history")

	_, _ = mock.Execute("status")
	_, _ = mock.Execute("log", "--oneline")
	_, _ = mock.Execute("status")

	assert.Equal(t, 3, mock.CallCount)
	assert.Len(t, mock.Commands, 3)

	assert.Equal(t, []string{"status"}, mock.Commands[0])
	assert.Equal(t, []string{"log", "--oneline"}, mock.Commands[1])
	assert.Equal(t, []string{"status"}, mock.Commands[2])

	assert.True(t, mock.HasCommand("status"))
	assert.True(t, mock.HasCommand("log", "--oneline"))
}

func TestMockGitExecutor_SetResponsePattern(t *testing.T) {
	mock := NewMockGitExecutor()

	pattern := regexp.MustCompile(`^branch --set-upstream-to=origin/.+ .+$`)
	mock.SetResponsePattern(pattern, "upstream configured", nil)

	output, err := mock.Execute("branch", "--set-upstream-to=origin/main", "main")
	require.NoError(t, err)
	assert.Equal(t, "upstream configured", output)

	output, err = mock.Execute("branch", "--set-upstream-to=origin/develop", "develop")
	require.NoError(t, err)
	assert.Equal(t, "upstream configured", output)

	errorPattern := regexp.MustCompile(`^push.*--force`)
	mock.SetResponsePattern(errorPattern, "", fmt.Errorf("force push rejected"))

	_, err = mock.Execute("push", "--force", "origin", "main")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "force push rejected")
}

func TestMockGitExecutor_SetSuccessResponsePattern(t *testing.T) {
	mock := NewMockGitExecutor()

	pattern := regexp.MustCompile(`^log --oneline.*`)
	mock.SetSuccessResponsePattern(pattern, "commit1\ncommit2\ncommit3")

	output, err := mock.Execute("log", "--oneline")
	require.NoError(t, err)
	assert.Equal(t, "commit1\ncommit2\ncommit3", output)

	output, err = mock.Execute("log", "--oneline", "--graph")
	require.NoError(t, err)
	assert.Equal(t, "commit1\ncommit2\ncommit3", output)

	output, err = mock.Execute("log", "--oneline", "-n", "5")
	require.NoError(t, err)
	assert.Equal(t, "commit1\ncommit2\ncommit3", output)
}

func TestMockGitExecutor_SetErrorResponsePattern(t *testing.T) {
	mock := NewMockGitExecutor()

	pattern := regexp.MustCompile(`^clone.*private.*`)
	testError := fmt.Errorf("authentication failed")
	mock.SetErrorResponsePattern(pattern, testError)

	_, err := mock.Execute("clone", "https://private.com/repo.git")
	require.Error(t, err)
	assert.Equal(t, testError, err)
}

func TestMockGitExecutor_SetErrorResponsePatternWithMessage(t *testing.T) {
	mock := NewMockGitExecutor()

	pattern := regexp.MustCompile(`^rebase.*interactive`)
	mock.SetErrorResponsePatternWithMessage(pattern, "interactive rebase not supported")

	_, err := mock.Execute("rebase", "--interactive", "HEAD~3")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "interactive rebase not supported")
}

func TestMockGitExecutor_RegexPatternPrecedence(t *testing.T) {
	mock := NewMockGitExecutor()

	mock.SetResponse("status", "string response", nil)
	pattern := regexp.MustCompile(`^status.*`)
	mock.SetSuccessResponsePattern(pattern, "regex response")

	output, err := mock.Execute("status")
	require.NoError(t, err)
	assert.Equal(t, "string response", output)

	// Use a different command that matches the regex but not string patterns.
	mock.Reset()
	pattern = regexp.MustCompile(`^(log|show).*`)
	mock.SetSuccessResponsePattern(pattern, "regex response")
	mock.SetResponse("log", "specific log response", nil)

	output, err = mock.Execute("log")
	require.NoError(t, err)
	assert.Equal(t, "specific log response", output)

	output, err = mock.Execute("show", "HEAD")
	require.NoError(t, err)
	assert.Equal(t, "regex response", output)
}

func TestMockGitExecutor_ComplexRegexPatterns(t *testing.T) {
	mock := NewMockGitExecutor()

	tests := []struct {
		name           string
		pattern        string
		expectedOutput string
		commands       [][]string
	}{
		{
			name:           "commit with message",
			pattern:        `^commit -m .*$`,
			expectedOutput: "commit created",
			commands: [][]string{
				{"commit", "-m", "initial commit"},
				{"commit", "-m", "fix: resolve bug"},
				{"commit", "-m", "feat: add new feature"},
			},
		},
		{
			name:           "branch operations",
			pattern:        `^branch (-d|--delete) .+$`,
			expectedOutput: "branch deleted",
			commands: [][]string{
				{"branch", "-d", "feature-branch"},
				{"branch", "--delete", "old-branch"},
			},
		},
		{
			name:           "remote operations",
			pattern:        `^remote (add|remove) \S+.*$`,
			expectedOutput: "remote updated",
			commands: [][]string{
				{"remote", "add", "origin", "https://github.com/user/repo.git"},
				{"remote", "remove", "upstream"},
			},
		},
		{
			name:           "tag operations",
			pattern:        `^tag -[a-z] v\d+\.\d+\.\d+`,
			expectedOutput: "tag created",
			commands: [][]string{
				{"tag", "-a", "v1.0.0", "-m", "version 1.0.0"},
				{"tag", "-s", "v2.1.3", "-m", "signed version"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := regexp.MustCompile(tt.pattern)
			mock.SetSuccessResponsePattern(pattern, tt.expectedOutput)

			for _, cmd := range tt.commands {
				output, err := mock.Execute(cmd...)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedOutput, output)
			}
		})
	}
}

func TestMockGitExecutor_RegexPatternReset(t *testing.T) {
	mock := NewMockGitExecutor()

	pattern1 := regexp.MustCompile(`^status.*`)
	pattern2 := regexp.MustCompile(`^log.*`)
	mock.SetSuccessResponsePattern(pattern1, "status output")
	mock.SetSuccessResponsePattern(pattern2, "log output")

	output, err := mock.Execute("status")
	require.NoError(t, err)
	assert.Equal(t, "status output", output)

	output, err = mock.Execute("log", "--oneline")
	require.NoError(t, err)
	assert.Equal(t, "log output", output)

	mock.Reset()
	_, err = mock.Execute("status")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unhandled git command")

	_, err = mock.Execute("log", "--oneline")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unhandled git command")
}

func TestMockGitExecutor_RegexPatternMatchingOrder(t *testing.T) {
	mock := NewMockGitExecutor()

	// Test that regex patterns are matched in order (first match wins).
	pattern1 := regexp.MustCompile(`^branch.*`)
	pattern2 := regexp.MustCompile(`^branch --set-upstream.*`)

	mock.SetSuccessResponsePattern(pattern1, "general branch response")
	mock.SetSuccessResponsePattern(pattern2, "specific upstream response")

	output, err := mock.Execute("branch", "--set-upstream-to=origin/main", "main")
	require.NoError(t, err)
	assert.Equal(t, "general branch response", output)

	mock.Reset()
	mock.SetSuccessResponsePattern(pattern2, "specific upstream response")
	mock.SetSuccessResponsePattern(pattern1, "general branch response")

	output, err = mock.Execute("branch", "--set-upstream-to=origin/main", "main")
	require.NoError(t, err)
	assert.Equal(t, "specific upstream response", output)
}

func TestMockGitExecutor_RegexPatternWithSpecialCharacters(t *testing.T) {
	mock := NewMockGitExecutor()

	pattern := regexp.MustCompile(`^config --get user\.email$`)
	mock.SetSuccessResponsePattern(pattern, "user@example.com")

	output, err := mock.Execute("config", "--get", "user.email")
	require.NoError(t, err)
	assert.Equal(t, "user@example.com", output)

	// Test that it doesn't match similar but different commands.
	_, err = mock.Execute("config", "--get", "user.name")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unhandled git command")
}

func TestMockGitExecutor_RegexPatternCommandTracking(t *testing.T) {
	mock := NewMockGitExecutor()

	pattern := regexp.MustCompile(`^push.*`)
	mock.SetSuccessResponsePattern(pattern, "pushed successfully")

	_, _ = mock.Execute("push", "origin", "main")
	_, _ = mock.Execute("push", "--force", "origin", "feature")
	_, _ = mock.Execute("push", "--tags")

	assert.Equal(t, 3, mock.CallCount)
	assert.Len(t, mock.Commands, 3)
	assert.Equal(t, []string{"push", "origin", "main"}, mock.Commands[0])
	assert.Equal(t, []string{"push", "--force", "origin", "feature"}, mock.Commands[1])
	assert.Equal(t, []string{"push", "--tags"}, mock.Commands[2])

	assert.True(t, mock.HasCommand("push", "origin", "main"))
	assert.True(t, mock.HasCommand("push", "--tags"))
	assert.False(t, mock.HasCommand("pull", "origin", "main"))
}
