package git

import (
	"testing"
)

func TestBranchToDirectoryName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple branch name",
			input:    "main",
			expected: "main",
		},
		{
			name:     "branch with forward slash",
			input:    "fix/123",
			expected: "fix-123",
		},
		{
			name:     "branch with multiple slashes",
			input:    "feature/user/auth",
			expected: "feature-user-auth",
		},
		{
			name:     "branch with backslash",
			input:    "fix\\windows",
			expected: "fix-windows",
		},
		{
			name:     "branch with special characters",
			input:    "bugfix/issue#456",
			expected: "bugfix-issue-456",
		},
		{
			name:     "branch with spaces and tabs",
			input:    "fix bug\twith spaces",
			expected: "fix-bug-with-spaces",
		},
		{
			name:     "branch with multiple problematic characters",
			input:    "fix/issue:*?\"<>|#123",
			expected: "fix-issue-123",
		},
		{
			name:     "branch with multiple consecutive hyphens",
			input:    "fix//issue",
			expected: "fix-issue",
		},
		{
			name:     "branch with leading/trailing hyphens",
			input:    "/fix/issue/",
			expected: "fix-issue",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only problematic characters",
			input:    "///",
			expected: "worktree",
		},
		{
			name:     "version branch",
			input:    "release/v1.2.3",
			expected: "release-v1.2.3",
		},
		{
			name:     "branch with underscores (should be preserved)",
			input:    "feature_branch",
			expected: "feature_branch",
		},
		{
			name:     "branch with dots (should be preserved)",
			input:    "hotfix/v1.2.3",
			expected: "hotfix-v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BranchToDirectoryName(tt.input)
			if result != tt.expected {
				t.Errorf("BranchToDirectoryName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsValidDirectoryName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid simple name",
			input:    "main",
			expected: true,
		},
		{
			name:     "valid name with hyphens",
			input:    "fix-123",
			expected: true,
		},
		{
			name:     "valid name with underscores",
			input:    "feature_branch",
			expected: true,
		},
		{
			name:     "invalid - forward slash",
			input:    "fix/123",
			expected: false,
		},
		{
			name:     "invalid - backslash",
			input:    "fix\\123",
			expected: false,
		},
		{
			name:     "invalid - colon",
			input:    "fix:123",
			expected: false,
		},
		{
			name:     "invalid - asterisk",
			input:    "fix*123",
			expected: false,
		},
		{
			name:     "invalid - question mark",
			input:    "fix?123",
			expected: false,
		},
		{
			name:     "invalid - double quote",
			input:    "fix\"123",
			expected: false,
		},
		{
			name:     "invalid - less than",
			input:    "fix<123",
			expected: false,
		},
		{
			name:     "invalid - greater than",
			input:    "fix>123",
			expected: false,
		},
		{
			name:     "invalid - pipe",
			input:    "fix|123",
			expected: false,
		},
		{
			name:     "invalid - hash",
			input:    "fix#123",
			expected: false,
		},
		{
			name:     "invalid - leading space",
			input:    " fix123",
			expected: false,
		},
		{
			name:     "invalid - trailing space",
			input:    "fix123 ",
			expected: false,
		},
		{
			name:     "invalid - leading dot",
			input:    ".fix123",
			expected: false,
		},
		{
			name:     "invalid - trailing dot",
			input:    "fix123.",
			expected: false,
		},
		{
			name:     "invalid - empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "invalid - Windows reserved name CON",
			input:    "CON",
			expected: false,
		},
		{
			name:     "invalid - Windows reserved name PRN",
			input:    "PRN",
			expected: false,
		},
		{
			name:     "invalid - Windows reserved name AUX",
			input:    "AUX",
			expected: false,
		},
		{
			name:     "invalid - Windows reserved name NUL",
			input:    "NUL",
			expected: false,
		},
		{
			name:     "invalid - Windows reserved name COM1",
			input:    "COM1",
			expected: false,
		},
		{
			name:     "invalid - Windows reserved name LPT1",
			input:    "LPT1",
			expected: false,
		},
		{
			name:     "valid - similar to reserved but different",
			input:    "CONSOLE",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidDirectoryName(tt.input)
			if result != tt.expected {
				t.Errorf("IsValidDirectoryName(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "normal branch name",
			input:    "main",
			expected: "main",
		},
		{
			name:     "branch name with slash",
			input:    "fix/123",
			expected: "fix/123",
		},
		{
			name:     "branch name starting with hyphen",
			input:    "-fix",
			expected: "branch-fix",
		},
		{
			name:     "branch name ending with dot",
			input:    "fix.",
			expected: "fix",
		},
		{
			name:     "branch name with both issues",
			input:    "-fix.",
			expected: "branch-fix",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "valid branch name with special characters",
			input:    "feature/user_auth",
			expected: "feature/user_auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeBranchName(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeBranchName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
