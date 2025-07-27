//go:build !integration
// +build !integration

package git

import (
	"errors"
	"testing"

	"github.com/sqve/grove/internal/testutils"
)

func TestCreateWorktreeWithSafeNaming(t *testing.T) {
	tests := []struct {
		name          string
		branchName    string
		basePath      string
		expectedPath  string
		expectedError string
		simulateError bool
	}{
		{
			name:         "simple branch name",
			branchName:   "main",
			basePath:     "/repo/worktrees",
			expectedPath: "/repo/worktrees/main",
		},
		{
			name:         "branch with forward slash",
			branchName:   "fix/123",
			basePath:     "/repo/worktrees",
			expectedPath: "/repo/worktrees/fix-123",
		},
		{
			name:         "branch with multiple slashes",
			branchName:   "feature/user/auth",
			basePath:     "/repo/worktrees",
			expectedPath: "/repo/worktrees/feature-user-auth",
		},
		{
			name:         "branch with special characters",
			branchName:   "bugfix/issue#456",
			basePath:     "/repo/worktrees",
			expectedPath: "/repo/worktrees/bugfix-issue-456",
		},
		{
			name:          "empty branch name",
			branchName:    "",
			basePath:      "/repo/worktrees",
			expectedError: "branch name cannot be empty",
		},
		{
			name:          "empty base path",
			branchName:    "main",
			basePath:      "",
			expectedError: "base path cannot be empty",
		},
		{
			name:          "git command fails",
			branchName:    "main",
			basePath:      "/repo/worktrees",
			simulateError: true,
			expectedError: "failed to create worktree for branch main at /repo/worktrees/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := testutils.NewMockGitExecutor()

			if tt.simulateError {
				expectedPath := "/repo/worktrees/main"
				executor.SetResponseSlice([]string{"worktree", "add", "-b", tt.branchName, expectedPath}, "", errors.New("git error"))
			} else if tt.expectedError == "" {
				executor.SetResponseSlice([]string{"worktree", "add", "-b", tt.branchName, tt.expectedPath}, "", nil)
			}

			path, err := CreateWorktreeWithSafeNaming(executor, tt.branchName, tt.basePath)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("CreateWorktreeWithSafeNaming() error = nil, want error containing %q", tt.expectedError)
					return
				}
				if err.Error() != tt.expectedError && !contains(err.Error(), tt.expectedError) {
					t.Errorf("CreateWorktreeWithSafeNaming() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return
			}

			if err != nil {
				t.Errorf("CreateWorktreeWithSafeNaming() error = %v, want nil", err)
				return
			}

			if path != tt.expectedPath {
				t.Errorf("CreateWorktreeWithSafeNaming() path = %q, want %q", path, tt.expectedPath)
			}

			if !tt.simulateError && tt.expectedError == "" {
				commands := executor.Commands
				if len(commands) != 1 {
					t.Errorf("Expected 1 git command, got %d", len(commands))
					return
				}

				expectedCmd := []string{"worktree", "add", "-b", tt.branchName, tt.expectedPath}
				if !slicesEqual(commands[0], expectedCmd) {
					t.Errorf("Expected git command %v, got %v", expectedCmd, commands[0])
				}
			}
		})
	}
}

func TestCreateWorktreeFromExistingBranch(t *testing.T) {
	tests := []struct {
		name          string
		branchName    string
		basePath      string
		expectedPath  string
		expectedError string
		simulateError bool
	}{
		{
			name:         "simple branch name",
			branchName:   "main",
			basePath:     "/repo/worktrees",
			expectedPath: "/repo/worktrees/main",
		},
		{
			name:         "branch with forward slash",
			branchName:   "fix/123",
			basePath:     "/repo/worktrees",
			expectedPath: "/repo/worktrees/fix-123",
		},
		{
			name:          "empty branch name",
			branchName:    "",
			basePath:      "/repo/worktrees",
			expectedError: "branch name cannot be empty",
		},
		{
			name:          "empty base path",
			branchName:    "main",
			basePath:      "",
			expectedError: "base path cannot be empty",
		},
		{
			name:          "git command fails",
			branchName:    "main",
			basePath:      "/repo/worktrees",
			simulateError: true,
			expectedError: "failed to create worktree from existing branch main at /repo/worktrees/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := testutils.NewMockGitExecutor()

			if tt.simulateError {
				expectedPath := "/repo/worktrees/main"
				executor.SetResponseSlice([]string{"worktree", "add", expectedPath, tt.branchName}, "", errors.New("git error"))
			} else if tt.expectedError == "" {
				executor.SetResponseSlice([]string{"worktree", "add", tt.expectedPath, tt.branchName}, "", nil)
			}

			path, err := CreateWorktreeFromExistingBranch(executor, tt.branchName, tt.basePath)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("CreateWorktreeFromExistingBranch() error = nil, want error containing %q", tt.expectedError)
					return
				}
				if !contains(err.Error(), tt.expectedError) {
					t.Errorf("CreateWorktreeFromExistingBranch() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return
			}

			if err != nil {
				t.Errorf("CreateWorktreeFromExistingBranch() error = %v, want nil", err)
				return
			}

			if path != tt.expectedPath {
				t.Errorf("CreateWorktreeFromExistingBranch() path = %q, want %q", path, tt.expectedPath)
			}

			if !tt.simulateError && tt.expectedError == "" {
				commands := executor.Commands
				if len(commands) != 1 {
					t.Errorf("Expected 1 git command, got %d", len(commands))
					return
				}

				expectedCmd := []string{"worktree", "add", tt.expectedPath, tt.branchName}
				if !slicesEqual(commands[0], expectedCmd) {
					t.Errorf("Expected git command %v, got %v", expectedCmd, commands[0])
				}
			}
		})
	}
}

func TestRemoveWorktree(t *testing.T) {
	tests := []struct {
		name          string
		worktreePath  string
		expectedError string
		simulateError bool
	}{
		{
			name:         "valid worktree path",
			worktreePath: "/repo/worktrees/feature-branch",
		},
		{
			name:          "empty worktree path",
			worktreePath:  "",
			expectedError: "worktree path cannot be empty",
		},
		{
			name:          "git command fails",
			worktreePath:  "/repo/worktrees/feature-branch",
			simulateError: true,
			expectedError: "failed to remove worktree at /repo/worktrees/feature-branch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := testutils.NewMockGitExecutor()

			if tt.simulateError {
				executor.SetResponseSlice([]string{"worktree", "remove", tt.worktreePath}, "", errors.New("git error"))
			} else if tt.expectedError == "" {
				executor.SetResponseSlice([]string{"worktree", "remove", tt.worktreePath}, "", nil)
			}

			err := RemoveWorktree(executor, tt.worktreePath)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("RemoveWorktree() error = nil, want error containing %q", tt.expectedError)
					return
				}
				if !contains(err.Error(), tt.expectedError) {
					t.Errorf("RemoveWorktree() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return
			}

			if err != nil {
				t.Errorf("RemoveWorktree() error = %v, want nil", err)
				return
			}

			if !tt.simulateError {
				commands := executor.Commands
				if len(commands) != 1 {
					t.Errorf("Expected 1 git command, got %d", len(commands))
					return
				}

				expectedCmd := []string{"worktree", "remove", tt.worktreePath}
				if !slicesEqual(commands[0], expectedCmd) {
					t.Errorf("Expected git command %v, got %v", expectedCmd, commands[0])
				}
			}
		})
	}
}

func TestListWorktrees(t *testing.T) {
	tests := []struct {
		name          string
		gitOutput     string
		expectedPaths []string
		expectedError string
		simulateError bool
	}{
		{
			name: "single worktree",
			gitOutput: `worktree /repo
HEAD abc123
branch refs/heads/main

`,
			expectedPaths: []string{"/repo"},
		},
		{
			name: "multiple worktrees",
			gitOutput: `worktree /repo
HEAD abc123
branch refs/heads/main

worktree /repo/worktrees/feature-branch
HEAD def456
branch refs/heads/feature-branch

worktree /repo/worktrees/fix-123
HEAD ghi789
branch refs/heads/fix-123

`,
			expectedPaths: []string{"/repo", "/repo/worktrees/feature-branch", "/repo/worktrees/fix-123"},
		},
		{
			name:          "empty output",
			gitOutput:     "",
			expectedPaths: nil,
		},
		{
			name:          "git command fails",
			simulateError: true,
			expectedError: "failed to list worktrees",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := testutils.NewMockGitExecutor()

			if tt.simulateError {
				executor.SetResponseSlice([]string{"worktree", "list", "--porcelain"}, "", errors.New("git error"))
			} else {
				executor.SetResponseSlice([]string{"worktree", "list", "--porcelain"}, tt.gitOutput, nil)
			}

			paths, err := ListWorktreesPaths(executor)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("ListWorktrees() error = nil, want error containing %q", tt.expectedError)
					return
				}
				if !contains(err.Error(), tt.expectedError) {
					t.Errorf("ListWorktrees() error = %q, want error containing %q", err.Error(), tt.expectedError)
				}
				return
			}

			if err != nil {
				t.Errorf("ListWorktrees() error = %v, want nil", err)
				return
			}

			if len(paths) != len(tt.expectedPaths) {
				t.Errorf("ListWorktrees() returned %d paths, want %d", len(paths), len(tt.expectedPaths))
				return
			}

			for i, path := range paths {
				if path != tt.expectedPaths[i] {
					t.Errorf("ListWorktrees() path[%d] = %q, want %q", i, path, tt.expectedPaths[i])
				}
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single line",
			input:    "line1",
			expected: []string{"line1"},
		},
		{
			name:     "multiple lines with \\n",
			input:    "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "lines with \\r\\n",
			input:    "line1\r\nline2\r\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single newline",
			input:    "\n",
			expected: []string{""},
		},
		{
			name:     "trailing newline",
			input:    "line1\nline2\n",
			expected: []string{"line1", "line2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitLines(%q) returned %d lines, want %d", tt.input, len(result), len(tt.expected))
				return
			}
			for i, line := range result {
				if line != tt.expected[i] {
					t.Errorf("splitLines(%q) line[%d] = %q, want %q", tt.input, i, line, tt.expected[i])
				}
			}
		})
	}
}

func TestCleanBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "refs/heads/ prefix",
			input:    "refs/heads/main",
			expected: "main",
		},
		{
			name:     "refs/heads/ prefix with feature branch",
			input:    "refs/heads/feature/user-auth",
			expected: "feature/user-auth",
		},
		{
			name:     "refs/remotes/origin/ prefix",
			input:    "refs/remotes/origin/main",
			expected: "main",
		},
		{
			name:     "refs/remotes/origin/ prefix with feature branch",
			input:    "refs/remotes/origin/feature/fix-bug",
			expected: "feature/fix-bug",
		},
		{
			name:     "refs/remotes/upstream/ prefix",
			input:    "refs/remotes/upstream/develop",
			expected: "develop",
		},
		{
			name:     "refs/remotes/custom-remote/ prefix",
			input:    "refs/remotes/custom-remote/feature/test",
			expected: "feature/test",
		},
		{
			name:     "regular branch name unchanged",
			input:    "main",
			expected: "main",
		},
		{
			name:     "feature branch name unchanged",
			input:    "feature/user-login",
			expected: "feature/user-login",
		},
		{
			name:     "bugfix branch name unchanged",
			input:    "bugfix/issue-123",
			expected: "bugfix/issue-123",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "refs/heads/ only (edge case)",
			input:    "refs/heads/",
			expected: "",
		},
		{
			name:     "refs/remotes/origin/ only (edge case)",
			input:    "refs/remotes/origin/",
			expected: "",
		},
		{
			name:     "incomplete ref path",
			input:    "refs/heads",
			expected: "refs/heads",
		},
		{
			name:     "malformed ref with only refs/remotes/",
			input:    "refs/remotes/",
			expected: "refs/remotes/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("CleanBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			indexOf(s, substr) >= 0)))
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
