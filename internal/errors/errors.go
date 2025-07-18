package errors

import (
	"errors"
	"fmt"
)

// Error codes for programmatic handling
const (
	// System errors
	ErrCodeGitNotFound     = "GIT_NOT_FOUND"
	ErrCodeDirectoryAccess = "DIRECTORY_ACCESS"
	ErrCodeFileSystem      = "FILE_SYSTEM"
	ErrCodePermission      = "PERMISSION"

	// Repository errors
	ErrCodeRepoExists     = "REPO_EXISTS"
	ErrCodeRepoNotFound   = "REPO_NOT_FOUND"
	ErrCodeRepoInvalid    = "REPO_INVALID"
	ErrCodeRepoConversion = "REPO_CONVERSION"

	// Git operation errors
	ErrCodeGitOperation = "GIT_OPERATION"
	ErrCodeGitClone     = "GIT_CLONE"
	ErrCodeGitInit      = "GIT_INIT"
	ErrCodeGitWorktree  = "GIT_WORKTREE"

	// URL and parsing errors
	ErrCodeInvalidURL     = "INVALID_URL"
	ErrCodeURLParsing     = "URL_PARSING"
	ErrCodeUnsupportedURL = "UNSUPPORTED_URL"

	// Security errors
	ErrCodePathTraversal     = "PATH_TRAVERSAL"
	ErrCodeSecurityViolation = "SECURITY_VIOLATION"

	// Configuration errors
	ErrCodeConfigInvalid = "CONFIG_INVALID"
	ErrCodeConfigMissing = "CONFIG_MISSING"

	// Network errors
	ErrCodeNetworkTimeout     = "NETWORK_TIMEOUT"
	ErrCodeNetworkUnavailable = "NETWORK_UNAVAILABLE"

	// Authentication errors
	ErrCodeAuthenticationFailed = "AUTHENTICATION_FAILED"
)

// GroveError represents a standardized error with code and context.
//
// GroveError provides structured error handling for Grove operations with:
//   - Code: standardized error code for programmatic handling
//   - Message: human-readable error description
//   - Cause: underlying error that caused this error (optional)
//   - Context: additional contextual information as key-value pairs
//   - Operation: the operation that failed (optional)
//
// Example usage:
//
//	err := ErrGitNotFound(nil).WithContext("path", "/usr/bin")
//	if IsGroveError(err, ErrCodeGitNotFound) {
//	  // Handle git not found error
//	}
type GroveError struct {
	Code      string                 // Standardized error code (see ErrCode* constants)
	Message   string                 // Human-readable error message
	Cause     error                  // Underlying error that caused this error
	Context   map[string]interface{} // Additional contextual information
	Operation string                 // The operation that failed
}

// Error implements the error interface
func (e *GroveError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error
func (e *GroveError) Unwrap() error {
	return e.Cause
}

// Is checks if the error matches the target error code
func (e *GroveError) Is(target error) bool {
	if t, ok := target.(*GroveError); ok {
		return e.Code == t.Code
	}
	return false
}

// WithContext adds context information to the error
func (e *GroveError) WithContext(key string, value interface{}) *GroveError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}
	e.Context[key] = value
	return e
}

// IsRetryable determines if this error represents a retryable condition
func (e *GroveError) IsRetryable() bool {
	switch e.Code {
	case ErrCodeNetworkTimeout,
		ErrCodeNetworkUnavailable:
		return true
	case ErrCodeAuthenticationFailed,
		ErrCodeInvalidURL,
		ErrCodeGitClone,
		ErrCodePermission,
		ErrCodeSecurityViolation:
		return false
	case ErrCodeGitOperation:
		// Git operations might be retryable depending on the underlying error
		// For now, we'll make them retryable and let the retry logic decide
		return true
	default:
		// Conservative approach: unknown errors are not retryable
		return false
	}
}

// NewGroveError creates a new standardized error
func NewGroveError(code, message string, cause error) *GroveError {
	return &GroveError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// NewGroveErrorf creates a new standardized error with formatted message
func NewGroveErrorf(code string, cause error, format string, args ...interface{}) *GroveError {
	return &GroveError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

// Error factory functions for common error types

// System errors
func ErrGitNotFound(cause error) *GroveError {
	return NewGroveError(ErrCodeGitNotFound, "git is not available in PATH", cause)
}

func ErrDirectoryAccess(path string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeDirectoryAccess, cause, "failed to access directory: %s", path).
		WithContext("path", path)
}

func ErrFileSystem(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeFileSystem, cause, "file system operation failed: %s", operation).
		WithContext("operation", operation)
}

// Repository errors
func ErrRepoExists(path string) *GroveError {
	return NewGroveErrorf(ErrCodeRepoExists, nil, "repository already exists at: %s", path).
		WithContext("path", path)
}

func ErrRepoNotFound(path string) *GroveError {
	return NewGroveErrorf(ErrCodeRepoNotFound, nil, "repository not found at: %s", path).
		WithContext("path", path)
}

func ErrRepoInvalid(path, reason string) *GroveError {
	return NewGroveErrorf(ErrCodeRepoInvalid, nil, "invalid repository at %s: %s", path, reason).
		WithContext("path", path).
		WithContext("reason", reason)
}

func ErrRepoConversion(path string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeRepoConversion, cause, "failed to convert repository at: %s", path).
		WithContext("path", path)
}

// Git operation errors
func ErrGitOperation(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeGitOperation, cause, "git %s failed", operation).
		WithContext("operation", operation)
}

func ErrGitClone(url string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeGitClone, cause, "failed to clone repository: %s", url).
		WithContext("url", url)
}

func ErrGitInit(path string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeGitInit, cause, "failed to initialize repository at: %s", path).
		WithContext("path", path)
}

func ErrGitWorktree(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeGitWorktree, cause, "worktree %s failed", operation).
		WithContext("operation", operation)
}

// URL and parsing errors
func ErrInvalidURL(url, reason string) *GroveError {
	return NewGroveErrorf(ErrCodeInvalidURL, nil, "invalid URL %s: %s", url, reason).
		WithContext("url", url).
		WithContext("reason", reason)
}

func ErrURLParsing(url string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeURLParsing, cause, "failed to parse URL: %s", url).
		WithContext("url", url)
}

func ErrUnsupportedURL(url string) *GroveError {
	return NewGroveErrorf(ErrCodeUnsupportedURL, nil, "unsupported URL format: %s", url).
		WithContext("url", url)
}

// Security errors
func ErrPathTraversal(path string) *GroveError {
	return NewGroveErrorf(ErrCodePathTraversal, nil, "path contains directory traversal: %s", path).
		WithContext("path", path)
}

func ErrSecurityViolation(operation, reason string) *GroveError {
	return NewGroveErrorf(ErrCodeSecurityViolation, nil, "security violation in %s: %s", operation, reason).
		WithContext("operation", operation).
		WithContext("reason", reason)
}

// Network errors
func ErrNetworkTimeout(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeNetworkTimeout, cause, "network timeout during %s", operation).
		WithContext("operation", operation)
}

func ErrNetworkUnavailable(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeNetworkUnavailable, cause, "network unavailable during %s", operation).
		WithContext("operation", operation)
}

// Authentication errors
func ErrAuthenticationFailed(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeAuthenticationFailed, cause, "authentication failed during %s", operation).
		WithContext("operation", operation)
}

// Helper function to check if an error is a specific Grove error
func IsGroveError(err error, code string) bool {
	var groveErr *GroveError
	if errors.As(err, &groveErr) {
		return groveErr.Code == code
	}
	return false
}

// Helper function to get Grove error code from any error
func GetErrorCode(err error) string {
	var groveErr *GroveError
	if errors.As(err, &groveErr) {
		return groveErr.Code
	}
	return ""
}

// Helper function to get error context
func GetErrorContext(err error) map[string]interface{} {
	var groveErr *GroveError
	if errors.As(err, &groveErr) {
		return groveErr.Context
	}
	return nil
}
