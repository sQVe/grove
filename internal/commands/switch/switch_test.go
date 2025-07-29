package switchcmd

import (
	"strings"
	"testing"

	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSwitchCmd(t *testing.T) {
	cmd := NewSwitchCmd()

	assert.NotNil(t, cmd, "NewSwitchCmd should return a command")
	assert.Equal(t, "switch", cmd.Use[:6], "Command should be named 'switch'")
	assert.Equal(t, "Switch to an existing worktree", cmd.Short, "Command should have correct short description")
}

func TestNewSwitchService(t *testing.T) {
	executor := &testutils.MockGitExecutor{}
	service := NewSwitchService(executor)

	assert.NotNil(t, service, "NewSwitchService should return a service")
}

// Security Tests

func TestValidateWorktreeName_SecurityVulnerabilities(t *testing.T) {
	tests := []struct {
		name        string
		worktreeName string
		shouldError bool
		expectedCode string
		description string
	}{
		// Shell injection attempts
		{
			name:         "shell injection with semicolon",
			worktreeName: "branch; rm -rf /",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject shell metacharacters",
		},
		{
			name:         "shell injection with backticks",
			worktreeName: "branch`rm -rf /`",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject command substitution",
		},
		{
			name:         "shell injection with dollar",
			worktreeName: "branch$(rm -rf /)",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject command substitution with $",
		},
		{
			name:         "shell injection with pipe",
			worktreeName: "branch | cat /etc/passwd",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject pipe characters",
		},
		{
			name:         "shell injection with ampersand",
			worktreeName: "branch & malicious-command",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject background execution",
		},
		// Path traversal attempts
		{
			name:         "path traversal with dots",
			worktreeName: "../../../etc/passwd",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject directory traversal",
		},
		{
			name:         "path traversal in middle",
			worktreeName: "branch/../../../etc/passwd",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject directory traversal in middle",
		},
		{
			name:         "windows path traversal",
			worktreeName: "branch\\..\\..\\windows\\system32",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject Windows directory traversal",
		},
		// Control characters and non-printable
		{
			name:         "null byte injection",
			worktreeName: "branch\x00malicious",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject null bytes",
		},
		{
			name:         "newline injection",
			worktreeName: "branch\nmalicious",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject newline characters",
		},
		{
			name:         "tab character",
			worktreeName: "branch\tmalicious",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject tab characters",
		},
		// Reserved names
		{
			name:         "dot directory",
			worktreeName: ".",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject current directory",
		},
		{
			name:         "dot dot directory",
			worktreeName: "..",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject parent directory",
		},
		{
			name:         "windows reserved name",
			worktreeName: "CON",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject Windows reserved names",
		},
		{
			name:         "windows reserved name with extension",
			worktreeName: "NUL.txt",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject Windows reserved names with extension",
		},
		// Edge cases
		{
			name:         "empty string",
			worktreeName: "",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject empty strings",
		},
		{
			name:         "whitespace only",
			worktreeName: "   \t\n  ",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject whitespace-only strings",
		},
		{
			name:         "very long name",
			worktreeName: strings.Repeat("a", 300),
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject overly long names",
		},
		{
			name:         "starts with hyphen",
			worktreeName: "-malicious-flag",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject names starting with hyphens",
		},
		{
			name:         "ends with dot",
			worktreeName: "branch.",
			shouldError:  true,
			expectedCode: errors.ErrCodeConfigInvalid,
			description:  "should reject names ending with dots",
		},
		// Valid cases
		{
			name:         "normal branch name",
			worktreeName: "feature-branch",
			shouldError:  false,
			description:  "should accept normal branch names",
		},
		{
			name:         "alphanumeric with underscores",
			worktreeName: "feature_branch_123",
			shouldError:  false,
			description:  "should accept alphanumeric with underscores",
		},
		{
			name:         "slash separated",
			worktreeName: "feature/new-feature",
			shouldError:  false,
			description:  "should accept slash-separated names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateWorktreeName(tt.worktreeName)
			
			if tt.shouldError {
				require.Error(t, err, tt.description)
				
				if tt.expectedCode != "" {
					var groveErr *errors.GroveError
					require.ErrorAs(t, err, &groveErr, "error should be a GroveError")
					assert.Equal(t, tt.expectedCode, groveErr.Code, "error code should match")
				}
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

func TestShellIntegration_NoInjectionVulnerabilities(t *testing.T) {
	tests := []struct {
		name      string
		shellType ShellType
	}{
		{"bash", ShellBash},
		{"zsh", ShellZsh},
		{"fish", ShellFish},
		{"powershell", ShellPowerShell},
	}

	for _, tt := range tests {
		t.Run(string(tt.name), func(t *testing.T) {
			code, err := GenerateShellIntegration(tt.shellType)
			require.NoError(t, err)
			require.NotEmpty(t, code)

			// Check that the generated code contains proper escaping/quoting
			switch tt.shellType {
			case ShellBash, ShellZsh:
				assert.Contains(t, code, "printf -v escaped_name '%q'", "bash/zsh should use printf for escaping")
				assert.Contains(t, code, "$escaped_name", "bash/zsh should use escaped variable")
			case ShellFish:
				assert.Contains(t, code, "string escape", "fish should use string escape")
				assert.Contains(t, code, "$escaped_name", "fish should use escaped variable")
			case ShellPowerShell:
				assert.Contains(t, code, "-replace \"'\"", "powershell should escape single quotes")
				assert.Contains(t, code, "'$escapedName'", "powershell should quote escaped variable")
			}

			// Ensure no unsafe direct variable substitution in command execution
			if tt.shellType != ShellPowerShell {
				// Should not use $2 directly in command substitution without escaping
				assert.NotContains(t, code, "grove switch --get-path $2", "should not use raw $2 in command")
				assert.NotContains(t, code, "grove switch --get-path \"$2\"", "should not use raw $2 in quoted command")
			}
		})
	}
}

func TestPathTraversalValidation(t *testing.T) {
	service := &switchService{
		executor: &testutils.MockGitExecutor{},
	}

	maliciousPaths := []string{
		"../../../etc/passwd",
		"/var/worktrees/../../../etc/passwd",
		"normal/../../../etc/passwd",
		"..\\..\\..\\windows\\system32",
	}

	for _, path := range maliciousPaths {
		t.Run("path_"+path, func(t *testing.T) {
			result, err := service.validateAndCleanPath(path)
			assert.Error(t, err, "should reject path traversal attempt")
			assert.Empty(t, result, "should not return path for traversal attempt")
			
			var groveErr *errors.GroveError
			if assert.ErrorAs(t, err, &groveErr, "error should be a GroveError") {
				// Should be either path traversal or directory access error
				assert.Contains(t, []string{errors.ErrCodePathTraversal, errors.ErrCodeDirectoryAccess}, 
					groveErr.Code, "should be path traversal or directory access error")
			}
		})
	}
}
