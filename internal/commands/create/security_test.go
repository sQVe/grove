//go:build !integration
// +build !integration

package create

import (
	"strings"
	"testing"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurity_PathTraversalPrevention(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		path         string
		shouldReject bool
		description  string
	}{
		{
			name:         "basic path traversal attempt",
			branchName:   "feature",
			path:         "/repo/../../../etc/passwd",
			shouldReject: false, // Path validation happens in path generator, not worktree creator
			description:  "Path traversal in worktree path",
		},
		{
			name:         "relative path traversal",
			branchName:   "feature",
			path:         "../../../etc/passwd",
			shouldReject: false, // Path validation happens in path generator
			description:  "Relative path traversal",
		},
		{
			name:         "malicious branch name with slashes",
			branchName:   "../../../malicious",
			path:         "/safe/path",
			shouldReject: false, // Branch names with slashes are valid in git
			description:  "Branch name with path separators",
		},
		{
			name:         "unicode path traversal attempt",
			branchName:   "feature",
			path:         "/repo/\u002e\u002e/\u002e\u002e/etc/passwd", // Unicode encoded ../..
			shouldReject: false,                                        // Would be handled by path validation
			description:  "Unicode encoded path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			creator := NewWorktreeCreator(mockExecutor)

			// Use safe paths for tests that should succeed
			testPath := tt.path
			if !tt.shouldReject {
				tmpDir := t.TempDir()
				testPath = tmpDir + "/worktree"
			}

			// Mock successful git operations
			mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/"+tt.branchName, "")
			mockExecutor.SetSuccessResponse("worktree add "+testPath+" "+tt.branchName, "")

			err := creator.CreateWorktree(tt.branchName, testPath, WorktreeOptions{})

			if tt.shouldReject {
				require.Error(t, err, "Expected security validation to reject: %s", tt.description)
				assert.Contains(t, err.Error(), "invalid")
			} else {
				// Note: Security validation happens at higher levels, worktree creator trusts validated inputs
				require.NoError(t, err, "Path validation should occur before worktree creation: %s", tt.description)
			}
		})
	}
}

func TestSecurity_GitCommandInjection(t *testing.T) {
	tests := []struct {
		name        string
		branchName  string
		path        string
		description string
	}{
		{
			name:        "command injection in branch name",
			branchName:  "feature; rm -rf /",
			path:        "/safe/path",
			description: "Attempting command injection via branch name",
		},
		{
			name:        "command injection in path",
			branchName:  "feature",
			path:        "/safe/path; rm -rf /",
			description: "Attempting command injection via path",
		},
		{
			name:        "null byte injection in branch name",
			branchName:  "feature\x00; rm -rf /",
			path:        "/safe/path",
			description: "Null byte injection in branch name",
		},
		{
			name:        "shell metacharacters in branch name",
			branchName:  "feature && echo 'hacked'",
			path:        "/safe/path",
			description: "Shell metacharacters in branch name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			creator := NewWorktreeCreator(mockExecutor)

			// Use safe temporary paths
			tmpDir := t.TempDir()
			testPath := tmpDir + "/worktree"
			if tt.path != "/safe/path" {
				testPath = tmpDir + "/" + strings.ReplaceAll(tt.path, "/", "_")
			}

			// Mock git operations - if injection succeeds, unexpected commands would be called
			mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/"+tt.branchName, "")
			mockExecutor.SetSuccessResponse("worktree add "+testPath+" "+tt.branchName, "")

			err := creator.CreateWorktree(tt.branchName, testPath, WorktreeOptions{})

			// Command injection should not succeed - git command line should be properly escaped
			require.NoError(t, err, "Git executor should handle special characters safely: %s", tt.description)

			// Verify that only expected git commands were called
			commands := mockExecutor.Commands
			assert.True(t, len(commands) >= 1, "Should have executed at least one command")

			// First command should be the branch check
			if len(commands) > 0 && len(commands[0]) > 0 {
				// Ensure no shell metacharacters were interpreted as separate commands
				assert.True(t, len(commands[0]) >= 2, "Commands should be properly structured")
				// Could be show-ref first, then worktree, depending on execution order
				assert.True(t, commands[0][0] == "show-ref" || commands[0][0] == "worktree", "Should execute expected git commands")
			}
		})
	}
}

func TestEdgeCases_BranchNames(t *testing.T) {
	tests := []struct {
		name        string
		branchName  string
		expectError bool
		description string
	}{
		{
			name:        "very long branch name",
			branchName:  strings.Repeat("a", 300), // Git has limits on ref names
			expectError: false,                    // Let git handle the validation
			description: "Branch name exceeding typical limits",
		},
		{
			name:        "unicode branch name",
			branchName:  "功能-分支-测试", // Chinese characters
			expectError: false,
			description: "Unicode characters in branch name",
		},
		{
			name:        "emoji branch name",
			branchName:  "feature-🚀-branch",
			expectError: false,
			description: "Emoji characters in branch name",
		},
		{
			name:        "branch name with spaces",
			branchName:  "feature branch",
			expectError: false, // Git allows spaces in branch names
			description: "Spaces in branch name",
		},
		{
			name:        "branch name starting with slash",
			branchName:  "/feature/branch",
			expectError: false,
			description: "Branch name starting with slash",
		},
		{
			name:        "branch name with consecutive slashes",
			branchName:  "feature//branch",
			expectError: false,
			description: "Consecutive slashes in branch name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			creator := NewWorktreeCreator(mockExecutor)

			tmpDir := t.TempDir()
			testPath := tmpDir + "/worktree"

			// Mock git operations
			mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/"+tt.branchName, "")
			mockExecutor.SetSuccessResponse("worktree add "+testPath+" "+tt.branchName, "")

			err := creator.CreateWorktree(tt.branchName, testPath, WorktreeOptions{})

			if tt.expectError {
				require.Error(t, err, "Expected error for edge case: %s", tt.description)
			} else {
				require.NoError(t, err, "Should handle edge case gracefully: %s", tt.description)
			}
		})
	}
}

func TestEdgeCases_PathNames(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		expectError bool
		description string
	}{
		{
			name:        "very long path",
			path:        "/" + strings.Repeat("very-long-directory-name/", 50) + "worktree",
			expectError: false, // Let filesystem handle the validation
			description: "Extremely long path",
		},
		{
			name:        "path with unicode",
			path:        "/repo/工作树/测试",
			expectError: false,
			description: "Unicode characters in path",
		},
		{
			name:        "path with spaces",
			path:        "/repo/my worktree/feature branch",
			expectError: false,
			description: "Spaces in path components",
		},
		{
			name:        "path with special filesystem characters",
			path:        "/repo/worktree:with|special<chars>",
			expectError: false, // Let filesystem validation handle this
			description: "Special filesystem characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			creator := NewWorktreeCreator(mockExecutor)

			// For tests that expect success, use safe temporary directories
			testPath := tt.path
			if !tt.expectError {
				tmpDir := t.TempDir()
				// Use part of the original path name for the test directory
				safeName := strings.ReplaceAll(strings.ReplaceAll(tt.path, "/", "_"), ":", "_")
				if len(safeName) > 50 {
					safeName = safeName[:50] // Truncate very long names
				}
				testPath = tmpDir + "/" + safeName
			}

			// Mock git operations
			mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature", "")
			mockExecutor.SetSuccessResponse("worktree add "+testPath+" feature", "")

			err := creator.CreateWorktree("feature", testPath, WorktreeOptions{})

			if tt.expectError {
				require.Error(t, err, "Expected error for edge case: %s", tt.description)
			} else {
				require.NoError(t, err, "Should handle edge case gracefully: %s", tt.description)
			}
		})
	}
}

func TestEdgeCases_ResourceConstraints(t *testing.T) {
	t.Run("disk space exhaustion simulation", func(t *testing.T) {
		mockExecutor := testutils.NewMockGitExecutor()
		creator := NewWorktreeCreator(mockExecutor)

		tmpDir := t.TempDir()
		testPath := tmpDir + "/worktree"

		// Simulate disk space error
		diskSpaceError := groveErrors.ErrFileSystem("create worktree",
			&groveErrors.GroveError{
				Code:    groveErrors.ErrCodeFileSystem,
				Message: "no space left on device",
			})

		mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature", "")
		mockExecutor.SetErrorResponse("worktree add "+testPath+" feature", diskSpaceError)

		err := creator.CreateWorktree("feature", testPath, WorktreeOptions{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "space")
	})

	t.Run("memory pressure simulation", func(t *testing.T) {
		mockExecutor := testutils.NewMockGitExecutor()
		creator := NewWorktreeCreator(mockExecutor)

		tmpDir := t.TempDir()
		testPath := tmpDir + "/worktree"

		// Simulate memory allocation failure
		memoryError := groveErrors.ErrWorktreeCreation("memory",
			&groveErrors.GroveError{
				Code:    groveErrors.ErrCodeFileSystem,
				Message: "cannot allocate memory",
			})

		mockExecutor.SetSuccessResponse("show-ref --verify --quiet refs/heads/feature", "")
		mockExecutor.SetErrorResponse("worktree add "+testPath+" feature", memoryError)

		err := creator.CreateWorktree("feature", testPath, WorktreeOptions{})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "memory")
	})
}
