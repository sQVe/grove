package errors

import (
	"errors"
	"fmt"
)

const (
	// Maximum number of context entries to prevent memory leaks.
	maxContextEntries = 10

	// System errors.
	ErrCodeGitNotFound     = "GIT_NOT_FOUND"
	ErrCodeDirectoryAccess = "DIRECTORY_ACCESS"
	ErrCodeFileSystem      = "FILE_SYSTEM"
	ErrCodePermission      = "PERMISSION"

	// Repository errors.
	ErrCodeRepoExists     = "REPO_EXISTS"
	ErrCodeRepoNotFound   = "REPO_NOT_FOUND"
	ErrCodeRepoInvalid    = "REPO_INVALID"
	ErrCodeRepoConversion = "REPO_CONVERSION"

	// Git operation errors.
	ErrCodeGitOperation = "GIT_OPERATION"
	ErrCodeGitClone     = "GIT_CLONE"
	ErrCodeGitInit      = "GIT_INIT"
	ErrCodeGitWorktree  = "GIT_WORKTREE"

	// URL and parsing errors.
	ErrCodeInvalidURL     = "INVALID_URL"
	ErrCodeURLParsing     = "URL_PARSING"
	ErrCodeUnsupportedURL = "UNSUPPORTED_URL"

	// Security errors.
	ErrCodePathTraversal     = "PATH_TRAVERSAL"
	ErrCodeSecurityViolation = "SECURITY_VIOLATION"

	// Configuration errors.
	ErrCodeConfigInvalid = "CONFIG_INVALID"
	ErrCodeConfigMissing = "CONFIG_MISSING"

	// Network errors.
	ErrCodeNetworkTimeout     = "NETWORK_TIMEOUT"
	ErrCodeNetworkUnavailable = "NETWORK_UNAVAILABLE"

	// Authentication errors.
	ErrCodeAuthenticationFailed = "AUTHENTICATION_FAILED"

	// Create command specific errors.
	ErrCodeBranchNotFound         = "BRANCH_NOT_FOUND"
	ErrCodeBranchExists           = "BRANCH_EXISTS"
	ErrCodeInvalidBranchName      = "INVALID_BRANCH_NAME"
	ErrCodeWorktreeCreation       = "WORKTREE_CREATION"
	ErrCodeRemoteNotFound         = "REMOTE_NOT_FOUND"
	ErrCodeFileCopyFailed         = "FILE_COPY_FAILED"
	ErrCodeSourceWorktreeNotFound = "SOURCE_WORKTREE_NOT_FOUND"
	ErrCodePathExists             = "PATH_EXISTS"
	ErrCodeInvalidPattern         = "INVALID_PATTERN"
)

//   - Code: standardized error code for programmatic handling.
//   - Message: human-readable error description.
//   - Cause: underlying error that caused this error (optional).
//   - Context: additional contextual information as key-value pairs.
//   - Operation: the operation that failed (optional).
//
// Example usage:.
//
//	err := ErrGitNotFound(nil).WithContext("path", "/usr/bin")
//	if IsGroveError(err, ErrCodeGitNotFound) {.
//	  // Handle git not found error.
//	}.
type GroveError struct {
	Code      string                 // Standardized error code (see ErrCode* constants).
	Message   string                 // Human-readable error message.
	Cause     error                  // Underlying error that caused this error.
	Context   map[string]interface{} // Additional contextual information.
	Operation string                 // The operation that failed.
}

func (e *GroveError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *GroveError) Unwrap() error {
	return e.Cause
}

func (e *GroveError) Is(target error) bool {
	if t, ok := target.(*GroveError); ok {
		return e.Code == t.Code
	}
	return false
}

func (e *GroveError) WithContext(key string, value interface{}) *GroveError {
	if e.Context == nil {
		e.Context = make(map[string]interface{})
	}

	// Prevent memory leaks by limiting context size.
	if len(e.Context) >= maxContextEntries {
		// Remove oldest entry to make room (simple FIFO approach).
		oldestKey := ""
		for k := range e.Context {
			oldestKey = k
			break
		}
		delete(e.Context, oldestKey)
	}

	e.Context[key] = value
	return e
}

func (e *GroveError) WithOperation(operation string) *GroveError {
	e.Operation = operation
	return e
}

// retryableErrors maps error codes to their retryability status.
var retryableErrors = map[string]bool{
	// Network errors are typically retryable.
	ErrCodeNetworkTimeout:     true,
	ErrCodeNetworkUnavailable: true,

	// Git and file operations may be retryable depending on context.
	ErrCodeGitOperation:     true,
	ErrCodeWorktreeCreation: true,
	ErrCodeFileCopyFailed:   true,

	// These errors are generally not retryable.
	ErrCodeAuthenticationFailed:   false,
	ErrCodeInvalidURL:             false,
	ErrCodeGitClone:               false,
	ErrCodePermission:             false,
	ErrCodeSecurityViolation:      false,
	ErrCodeBranchNotFound:         false,
	ErrCodeBranchExists:           false,
	ErrCodeInvalidBranchName:      false,
	ErrCodeRemoteNotFound:         false,
	ErrCodeSourceWorktreeNotFound: false,
	ErrCodePathExists:             false,
	ErrCodeInvalidPattern:         false,
}

func (e *GroveError) IsRetryable() bool {
	if retryable, exists := retryableErrors[e.Code]; exists {
		return retryable
	}
	// Conservative approach: unknown errors are not retryable.
	return false
}

func NewGroveError(code, message string, cause error) *GroveError {
	return &GroveError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

func NewGroveErrorf(code string, cause error, format string, args ...interface{}) *GroveError {
	return &GroveError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

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

func ErrPathTraversal(path string) *GroveError {
	return NewGroveErrorf(ErrCodePathTraversal, nil, "path contains directory traversal: %s", path).
		WithContext("path", path)
}

func ErrSecurityViolation(operation, reason string) *GroveError {
	return NewGroveErrorf(ErrCodeSecurityViolation, nil, "security violation in %s: %s", operation, reason).
		WithContext("operation", operation).
		WithContext("reason", reason)
}

func ErrNetworkTimeout(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeNetworkTimeout, cause, "network timeout during %s", operation).
		WithContext("operation", operation)
}

func ErrNetworkUnavailable(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeNetworkUnavailable, cause, "network unavailable during %s", operation).
		WithContext("operation", operation)
}

func ErrAuthenticationFailed(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeAuthenticationFailed, cause, "authentication failed during %s", operation).
		WithContext("operation", operation)
}

func ErrBranchNotFound(branchName string) *GroveError {
	return NewGroveErrorf(ErrCodeBranchNotFound, nil, "branch '%s' not found", branchName).
		WithContext("branch", branchName)
}

func ErrBranchExists(branchName string) *GroveError {
	return NewGroveErrorf(ErrCodeBranchExists, nil, "branch '%s' already exists", branchName).
		WithContext("branch", branchName)
}

func ErrInvalidBranchName(branchName, reason string) *GroveError {
	return NewGroveErrorf(ErrCodeInvalidBranchName, nil, "invalid branch name '%s': %s", branchName, reason).
		WithContext("branch", branchName).
		WithContext("reason", reason)
}

func ErrWorktreeCreation(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeWorktreeCreation, cause, "worktree %s failed", operation).
		WithContext("operation", operation)
}

func ErrBranchInUseByWorktree(branchName, worktreePath string) *GroveError {
	message := fmt.Sprintf("branch '%s' is already checked out in another worktree", branchName)
	if worktreePath != "" {
		message += fmt.Sprintf(" at: %s", worktreePath)
	}
	message += "\nTip: Use a different branch name or switch to that worktree"
	
	return NewGroveErrorf(ErrCodeWorktreeCreation, nil, message).
		WithContext("branch", branchName).
		WithContext("worktree_path", worktreePath)
}

func ErrRemoteNotFound(remoteName string) *GroveError {
	return NewGroveErrorf(ErrCodeRemoteNotFound, nil, "remote '%s' not found", remoteName).
		WithContext("remote", remoteName)
}

func ErrFileCopyFailed(operation string, cause error) *GroveError {
	return NewGroveErrorf(ErrCodeFileCopyFailed, cause, "file copy operation failed: %s", operation).
		WithContext("operation", operation)
}

func ErrSourceWorktreeNotFound(path string) *GroveError {
	return NewGroveErrorf(ErrCodeSourceWorktreeNotFound, nil, "source worktree not found at: %s", path).
		WithContext("path", path)
}

func ErrPathExists(path string) *GroveError {
	return NewGroveErrorf(ErrCodePathExists, nil, "path already exists: %s", path).
		WithContext("path", path)
}

func ErrInvalidPattern(pattern, reason string) *GroveError {
	return NewGroveErrorf(ErrCodeInvalidPattern, nil, "invalid pattern '%s': %s", pattern, reason).
		WithContext("pattern", pattern).
		WithContext("reason", reason)
}

func IsGroveError(err error, code string) bool {
	var groveErr *GroveError
	if errors.As(err, &groveErr) {
		return groveErr.Code == code
	}
	return false
}

func GetErrorCode(err error) string {
	var groveErr *GroveError
	if errors.As(err, &groveErr) {
		return groveErr.Code
	}
	return ""
}

func GetErrorContext(err error) map[string]interface{} {
	var groveErr *GroveError
	if errors.As(err, &groveErr) {
		return groveErr.Context
	}
	return nil
}
