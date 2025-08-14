package errors

import (
	"errors"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestNewGroveError(t *testing.T) {
	code := ErrCodeBranchNotFound
	message := "branch 'feature' not found"
	cause := errors.New("git command failed")

	err := NewGroveError(code, message, cause)

	assert.Equal(t, code, err.Code)
	assert.Equal(t, message, err.Message)
	assert.Equal(t, cause, err.Cause)
	assert.NotNil(t, err.Context) // Context is initialized as empty map
	assert.Empty(t, err.Context)
}

func TestGroveError_Error(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		message  string
		cause    error
		expected string
	}{
		{
			name:     "error without cause",
			code:     ErrCodeBranchNotFound,
			message:  "branch not found",
			cause:    nil,
			expected: "branch not found",
		},
		{
			name:     "error with cause",
			code:     ErrCodeGitOperation,
			message:  "git operation failed",
			cause:    errors.New("exit status 1"),
			expected: "git operation failed: exit status 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewGroveError(tt.code, tt.message, tt.cause)
			assert.Equal(t, tt.expected, err.Error())
		})
	}
}

func TestGroveError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := NewGroveError(ErrCodeGitOperation, "operation failed", cause)

	unwrapped := err.Unwrap()
	assert.Equal(t, cause, unwrapped)
}

func TestGroveError_Unwrap_NilCause(t *testing.T) {
	err := NewGroveError(ErrCodeBranchNotFound, "branch not found", nil)

	unwrapped := err.Unwrap()
	assert.Nil(t, unwrapped)
}

func TestGroveError_WithContext(t *testing.T) {
	err := NewGroveError(ErrCodeBranchNotFound, "branch not found", nil)

	updated := err.WithContext("branch_name", "feature").
		WithContext("repository", "/path/to/repo")

	// Should return same instance
	assert.Equal(t, err, updated)

	// Should have context entries
	assert.Len(t, err.Context, 2)

	// Check context values
	assert.Equal(t, "feature", err.Context["branch_name"])
	assert.Equal(t, "/path/to/repo", err.Context["repository"])
}

func TestGroveError_WithOperation(t *testing.T) {
	err := NewGroveError(ErrCodeGitOperation, "operation failed", nil)

	updated := err.WithOperation("create_worktree")

	// Should return same instance
	assert.Equal(t, err, updated)

	// Should have operation set
	assert.Equal(t, "create_worktree", err.Operation)
}

func TestGroveError_ContextLimit(t *testing.T) {
	// Set a low limit for testing
	viper.Set("errors.max_context_entries", 2)
	defer viper.Set("errors.max_context_entries", defaultMaxContextEntries)

	err := NewGroveError(ErrCodeBranchNotFound, "test error", nil)

	// Add more context entries than the limit
	_ = err.WithContext("key1", "value1").
		WithContext("key2", "value2").
		WithContext("key3", "value3")

	// Should only keep the most recent entries
	assert.Len(t, err.Context, 2)
	assert.Equal(t, "value2", err.Context["key2"])
	assert.Equal(t, "value3", err.Context["key3"])
	assert.Nil(t, err.Context["key1"]) // Should be evicted
}

func TestErrorFunctions(t *testing.T) {
	tests := []struct {
		name     string
		function func(string) *GroveError
		code     string
	}{
		{"ErrRepoExists", ErrRepoExists, ErrCodeRepoExists},
		{"ErrRepoNotFound", ErrRepoNotFound, ErrCodeRepoNotFound},
		{"ErrBranchNotFound", ErrBranchNotFound, ErrCodeBranchNotFound},
		{"ErrBranchExists", ErrBranchExists, ErrCodeBranchExists},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.function("test-value")

			assert.Equal(t, tt.code, err.Code)
			assert.NotEmpty(t, err.Message)
			assert.Contains(t, err.Message, "test-value")
		})
	}
}

func TestErrGitNotFound(t *testing.T) {
	cause := errors.New("command not found")
	err := ErrGitNotFound(cause)

	assert.Equal(t, ErrCodeGitNotFound, err.Code)
	assert.Contains(t, err.Message, "git is not available in PATH")
	assert.Equal(t, cause, err.Cause)
}

func TestErrorWrappingWithStandardLibrary(t *testing.T) {
	originalErr := errors.New("original error")
	groveErr := NewGroveError(ErrCodeGitOperation, "git failed", originalErr)

	// Test with errors.Is
	assert.True(t, errors.Is(groveErr, originalErr))

	// Test with errors.As
	var target *GroveError
	assert.True(t, errors.As(groveErr, &target))
	assert.Equal(t, groveErr, target)
}

func TestDefaultMaxContextEntries(t *testing.T) {
	assert.Equal(t, 10, defaultMaxContextEntries)
}

func TestErrorCodes(t *testing.T) {
	// Verify all error codes are defined
	codes := []string{
		ErrCodeGitNotFound,
		ErrCodeDirectoryAccess,
		ErrCodeFileSystem,
		ErrCodePermission,
		ErrCodeRepoExists,
		ErrCodeRepoNotFound,
		ErrCodeRepoInvalid,
		ErrCodeRepoConversion,
		ErrCodeGitOperation,
		ErrCodeGitClone,
		ErrCodeGitInit,
		ErrCodeGitWorktree,
		ErrCodeInvalidURL,
		ErrCodeURLParsing,
		ErrCodeUnsupportedURL,
		ErrCodePathTraversal,
		ErrCodeSecurityViolation,
		ErrCodeConfigInvalid,
		ErrCodeConfigMissing,
		ErrCodeNetworkTimeout,
		ErrCodeNetworkUnavailable,
		ErrCodeAuthenticationFailed,
		ErrCodeBranchNotFound,
		ErrCodeBranchExists,
		ErrCodeInvalidBranchName,
		ErrCodeWorktreeCreation,
		ErrCodeRemoteNotFound,
		ErrCodeFileCopyFailed,
		ErrCodeSourceWorktreeNotFound,
		ErrCodePathExists,
		ErrCodeInvalidPattern,
	}

	for _, code := range codes {
		assert.NotEmpty(t, code, "Error code should not be empty")
		assert.IsType(t, "", code, "Error code should be a string")
	}
}
