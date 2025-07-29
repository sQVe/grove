package testutils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCase represents a generic test case structure for table-driven tests
type TestCase[T any] struct {
	Name        string
	Input       T
	Expected    interface{}
	ExpectError bool
	ErrorMsg    string
	Setup       func(*testing.T)
	Cleanup     func(*testing.T)
}

// AssertStringContains checks if a string contains all expected substrings
func AssertStringContains(t *testing.T, actual string, expectedSubstrings []string, msgAndArgs ...interface{}) {
	t.Helper()
	for _, expected := range expectedSubstrings {
		assert.Contains(t, actual, expected, msgAndArgs...)
	}
}

// AssertStringNotContains checks if a string does not contain any of the forbidden substrings
func AssertStringNotContains(t *testing.T, actual string, forbiddenSubstrings []string, msgAndArgs ...interface{}) {
	t.Helper()
	for _, forbidden := range forbiddenSubstrings {
		assert.NotContains(t, actual, forbidden, msgAndArgs...)
	}
}

// AssertGroveError checks if an error is of a specific type
func AssertGroveError(t *testing.T, err error, expectedCode string, msgAndArgs ...interface{}) {
	t.Helper()
	require.Error(t, err, msgAndArgs...)

	// Check if it's a Grove error by looking for expected patterns
	errStr := err.Error()
	assert.Contains(t, errStr, expectedCode, msgAndArgs...)
}

// RunTableDrivenTest runs a set of test cases with common setup/teardown
func RunTableDrivenTest[T any](t *testing.T, testCases []TestCase[T], testFunc func(*testing.T, TestCase[T])) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Setup != nil {
				tc.Setup(t)
			}
			if tc.Cleanup != nil {
				defer tc.Cleanup(t)
			}

			testFunc(t, tc)
		})
	}
}

// StringSlicesEqual compares two string slices for equality (helper for tests without testify)
func StringSlicesEqual(a, b []string) bool {
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

// ContainsString checks if a string slice contains a specific string
func ContainsString(slice []string, target string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

// ContainsSubstring checks if any string in a slice contains the target substring
func ContainsSubstring(slice []string, target string) bool {
	for _, item := range slice {
		if strings.Contains(item, target) {
			return true
		}
	}
	return false
}

// MockGitError creates a mock git error for testing error handling
type MockGitError struct {
	Command  string
	Args     []string
	ExitCode int
	Message  string
}

func (e *MockGitError) Error() string {
	if e.Command != "" {
		return e.Command + " failed: " + e.Message
	}
	return e.Message
}

// NewMockGitError creates a new mock git error
func NewMockGitError(command string, exitCode int, message string) *MockGitError {
	return &MockGitError{
		Command:  command,
		ExitCode: exitCode,
		Message:  message,
	}
}

// Common error patterns for testing
var (
	MockNetworkError    = NewMockGitError("fetch", 128, "connection timeout")
	MockAuthError       = NewMockGitError("push", 128, "authentication failed")
	MockPermissionError = NewMockGitError("clone", 128, "permission denied")
	MockDiskSpaceError  = NewMockGitError("worktree", 128, "no space left on device")
)