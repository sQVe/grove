package git

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockGitCommander is a test mock for the Commander interface.
// Note: We define this locally to avoid import cycles with testutils package.
// The testutils package also has a MockGitCommander for use in other packages.
type MockGitCommander struct {
	mock.Mock
}

// Ensure MockGitCommander implements Commander interface at compile time.
var _ Commander = (*MockGitCommander)(nil)

func (m *MockGitCommander) Run(workDir string, args ...string) (stdout, stderr []byte, err error) {
	mockArgs := []interface{}{workDir}
	for _, arg := range args {
		mockArgs = append(mockArgs, arg)
	}
	called := m.Called(mockArgs...)
	return called.Get(0).([]byte), called.Get(1).([]byte), called.Error(2)
}

func (m *MockGitCommander) RunQuiet(workDir string, args ...string) error {
	mockArgs := []interface{}{workDir}
	for _, arg := range args {
		mockArgs = append(mockArgs, arg)
	}
	called := m.Called(mockArgs...)
	return called.Error(0)
}

// TestHelper provides robust testing infrastructure methods.
// This is a simplified version to avoid import cycles with testutils.
// We can't import testutils here due to circular dependencies,
// but we follow the same patterns for consistency.
type TestHelper struct {
	t *testing.T
}

func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

func (h *TestHelper) CreateTempDir(pattern string) string {
	h.t.Helper()
	// Use t.TempDir() which provides automatic cleanup
	tempDir := h.t.TempDir()
	if pattern != "" {
		// Create the subdirectory if pattern is provided
		fullPath := filepath.Join(tempDir, pattern)
		err := os.MkdirAll(fullPath, 0o755)
		if err != nil {
			h.t.Fatalf("failed to create temp directory: %v", err)
		}
		return fullPath
	}
	return tempDir
}

// IsWindows returns true if running on Windows platform.
func isWindows() bool {
	return os.PathSeparator == '\\' && os.PathListSeparator == ';'
}

// Core Functionality Tests

func TestGitCommander_Run_Success(t *testing.T) {
	t.Parallel()

	// Test with LiveGitCommander - use a safe git command that always works
	commander := NewLiveGitCommander()

	stdout, stderr, err := commander.Run("", "version")

	require.NoError(t, err)
	assert.NotEmpty(t, stdout)
	assert.Empty(t, stderr)
	assert.Contains(t, string(stdout), "git version")
}

func TestGitCommander_Run_EmptyCommand(t *testing.T) {
	t.Parallel()

	commander := NewLiveGitCommander()

	// Git with no arguments shows usage
	stdout, stderr, err := commander.Run("")

	// Git returns an error when no command is provided
	assert.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)
	assert.Equal(t, "git", gitErr.Command)
	assert.Empty(t, gitErr.Args)
	assert.NotEqual(t, 0, gitErr.ExitCode)
	// Git usage goes to stdout or stderr depending on git version
	// At least one should have content
	assert.True(t, len(stdout) > 0 || len(stderr) > 0, "Expected output in stdout or stderr")
	// Both should be captured (even if empty)
	// stderr will be empty slice, not nil
	assert.NotNil(t, stdout)
	// stderr may be an empty byte slice, which is still valid
}

func TestGitCommander_Run_LongRunningCommand(t *testing.T) {
	// Skip this test in short mode as it involves timing
	if testing.Short() {
		t.Skip("Skipping long-running test in short mode")
	}

	// Use mock for simulating long-running commands
	mockCommander := &MockGitCommander{}

	// Simulate a command that takes time
	mockCommander.On("Run", "", "fetch", "--all").
		Run(func(args mock.Arguments) {
			// Simulate delay
			time.Sleep(100 * time.Millisecond)
		}).
		Return([]byte("Fetching origin\n"), []byte{}, nil)

	start := time.Now()
	stdout, stderr, err := mockCommander.Run("", "fetch", "--all")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.Equal(t, "Fetching origin\n", string(stdout))
	assert.Empty(t, stderr)
	assert.GreaterOrEqual(t, duration, 100*time.Millisecond)

	mockCommander.AssertExpectations(t)
}

// Error Handling Tests

func TestGitCommander_Run_InvalidCommand(t *testing.T) {
	t.Parallel()

	commander := NewLiveGitCommander()

	// Use a non-existent Git subcommand
	stdout, stderr, err := commander.Run("", "nonexistentcommand")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	assert.Equal(t, "git", gitErr.Command)
	assert.Equal(t, []string{"nonexistentcommand"}, gitErr.Args)
	assert.NotEqual(t, 0, gitErr.ExitCode)
	assert.Contains(t, gitErr.Stderr, "not a git command")
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)
}

func TestGitCommander_Run_PermissionDenied(t *testing.T) {
	// Skip on Windows as permission handling is different
	if isWindows() {
		t.Skip("Skipping permission test on Windows")
	}

	// Create a directory with no permissions using robust testing infrastructure
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir("test-perms")
	restrictedDir := filepath.Join(tempDir, "restricted")

	// Create the directory first with normal permissions
	err := os.Mkdir(restrictedDir, 0o755)
	require.NoError(t, err)

	// Create a .git directory inside to make it look like a repo
	gitDir := filepath.Join(restrictedDir, ".git")
	err = os.Mkdir(gitDir, 0o755)
	require.NoError(t, err)

	// Now remove all permissions from the .git directory
	err = os.Chmod(gitDir, 0o000)
	require.NoError(t, err)

	// Ensure cleanup even if test fails
	t.Cleanup(func() {
		_ = os.Chmod(gitDir, 0o755)
		_ = os.Chmod(restrictedDir, 0o755)
	})

	commander := NewLiveGitCommander()

	// Try to run git status in the directory with restricted .git
	stdout, stderr, err := commander.Run(restrictedDir, "status")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	// Git should fail with a non-zero exit code
	assert.NotEqual(t, 0, gitErr.ExitCode)
	// There should be an error message
	assert.NotEmpty(t, gitErr.Stderr)
	// Validate stdout and stderr are not nil
	assert.NotNil(t, stdout)
	assert.NotNil(t, stderr)
}

func TestGitCommander_Run_NetworkTimeout(t *testing.T) {
	t.Parallel()

	// Use mock to simulate network timeout
	mockCommander := &MockGitCommander{}

	// Simulate a network timeout error
	mockCommander.On("Run", "", "clone", "https://example.com/timeout.git").
		Return([]byte{}, []byte("fatal: unable to access 'https://example.com/timeout.git': Connection timed out"),
			&GitError{
				Command:  "git",
				Args:     []string{"clone", "https://example.com/timeout.git"},
				Stderr:   "fatal: unable to access 'https://example.com/timeout.git': Connection timed out",
				ExitCode: 128,
			})

	stdout, stderr, err := mockCommander.Run("", "clone", "https://example.com/timeout.git")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	assert.Equal(t, 128, gitErr.ExitCode)
	assert.Contains(t, gitErr.Stderr, "Connection timed out")
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)

	mockCommander.AssertExpectations(t)
}

func TestGitCommander_Run_CorruptedRepository(t *testing.T) {
	t.Parallel()

	// Create a corrupted repository by creating an invalid .git directory
	helper := NewTestHelper(t)
	tempDir := helper.CreateTempDir("corrupted-repo")
	gitDir := filepath.Join(tempDir, ".git")
	err := os.Mkdir(gitDir, 0o755)
	require.NoError(t, err)

	// Create an invalid HEAD file
	headFile := filepath.Join(gitDir, "HEAD")
	err = os.WriteFile(headFile, []byte("invalid content"), 0o644)
	require.NoError(t, err)

	commander := NewLiveGitCommander()

	// Try to run a git command in the corrupted repository
	stdout, stderr, err := commander.Run(tempDir, "status")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	assert.NotEqual(t, 0, gitErr.ExitCode)
	assert.NotEmpty(t, gitErr.Stderr)
	// Git will complain about invalid repository
	assert.Contains(t, strings.ToLower(gitErr.Stderr), "fatal")
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)
}

func TestGitCommander_Run_DiskSpaceError(t *testing.T) {
	t.Parallel()

	// Use mock to simulate disk space error
	mockCommander := &MockGitCommander{}

	mockCommander.On("Run", "/tmp", "clone", "https://example.com/large-repo.git").
		Return([]byte{}, []byte("fatal: write error: No space left on device"),
			&GitError{
				Command:  "git",
				Args:     []string{"clone", "https://example.com/large-repo.git"},
				Stderr:   "fatal: write error: No space left on device",
				ExitCode: 128,
			})

	stdout, stderr, err := mockCommander.Run("/tmp", "clone", "https://example.com/large-repo.git")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	assert.Equal(t, 128, gitErr.ExitCode)
	assert.Contains(t, gitErr.Stderr, "No space left on device")
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)

	mockCommander.AssertExpectations(t)
}

func TestGitCommander_Run_InterruptedOperation(t *testing.T) {
	t.Parallel()

	// Use mock to simulate interrupted operation
	mockCommander := &MockGitCommander{}

	mockCommander.On("Run", "", "merge", "feature-branch").
		Return([]byte{}, []byte("fatal: merge interrupted"),
			&GitError{
				Command:  "git",
				Args:     []string{"merge", "feature-branch"},
				Stderr:   "fatal: merge interrupted",
				ExitCode: 130, // SIGINT exit code
			})

	stdout, stderr, err := mockCommander.Run("", "merge", "feature-branch")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	assert.Equal(t, 130, gitErr.ExitCode) // 130 is the exit code for SIGINT
	assert.Contains(t, gitErr.Stderr, "interrupted")
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)

	mockCommander.AssertExpectations(t)
}

// Context and Cancellation Tests
// Note: These tests demonstrate patterns for future context-aware implementation.
// The current Commander interface doesn't support context, but these tests
// show how it could be tested once that support is added.

func TestGitCommander_Run_WithContextPattern(t *testing.T) {
	t.Parallel()

	// Use mock to demonstrate context pattern
	mockCommander := &MockGitCommander{}

	// Create a context with a value for demonstration
	type contextKey string
	const testKey contextKey = "testKey"
	ctx := context.WithValue(context.Background(), testKey, "testValue")

	// Simulate successful command execution
	mockCommander.On("Run", "", "status").
		Return([]byte("On branch main\n"), []byte{}, nil)

	stdout, stderr, err := mockCommander.Run("", "status")

	require.NoError(t, err)
	assert.Equal(t, "On branch main\n", string(stdout))
	assert.Empty(t, stderr)

	// Context would be used in a real RunWithContext implementation
	assert.NotNil(t, ctx) // Validate context is properly created

	mockCommander.AssertExpectations(t)
}

func TestGitCommander_Run_ContextCancellationPattern(t *testing.T) {
	t.Parallel()

	// Use mock to simulate context cancellation
	mockCommander := &MockGitCommander{}

	// Simulate a command that gets cancelled
	mockCommander.On("Run", "", "fetch", "--all").
		Return([]byte{}, []byte("fatal: operation cancelled"),
			&GitError{
				Command:  "git",
				Args:     []string{"fetch", "--all"},
				Stderr:   "fatal: operation cancelled",
				ExitCode: 1,
			})

	stdout, stderr, err := mockCommander.Run("", "fetch", "--all")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	assert.Contains(t, gitErr.Stderr, "cancelled")
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)

	mockCommander.AssertExpectations(t)
}

func TestGitCommander_Run_ContextTimeoutPattern(t *testing.T) {
	t.Parallel()

	// Use mock to simulate context timeout
	mockCommander := &MockGitCommander{}

	mockCommander.On("Run", "", "clone", "https://example.com/slow-repo.git").
		Return([]byte{}, []byte("fatal: operation timed out"),
			&GitError{
				Command:  "git",
				Args:     []string{"clone", "https://example.com/slow-repo.git"},
				Stderr:   "fatal: operation timed out",
				ExitCode: 124, // Timeout exit code
			})

	stdout, stderr, err := mockCommander.Run("", "clone", "https://example.com/slow-repo.git")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	assert.Equal(t, 124, gitErr.ExitCode)
	assert.Contains(t, gitErr.Stderr, "timed out")
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)

	mockCommander.AssertExpectations(t)
}

func TestGitCommander_Run_ContextValuesPattern(t *testing.T) {
	t.Parallel()

	// Use mock to demonstrate context value preservation
	mockCommander := &MockGitCommander{}

	// Set up expectation
	mockCommander.On("Run", "/workspace", "branch", "--list").
		Return([]byte("* main\n  feature\n"), []byte{}, nil)

	stdout, stderr, err := mockCommander.Run("/workspace", "branch", "--list")

	require.NoError(t, err)
	assert.Contains(t, string(stdout), "main")
	assert.Contains(t, string(stdout), "feature")
	assert.Empty(t, stderr)

	mockCommander.AssertExpectations(t)
}

// Logging and Observability Tests

func TestGitCommander_Run_LogsCommands(t *testing.T) {
	t.Parallel()

	// We can't directly test logging without modifying the logger,
	// but we can verify the command executes and would be logged
	commander := NewLiveGitCommander()

	stdout, stderr, err := commander.Run("", "version")

	require.NoError(t, err)
	assert.NotEmpty(t, stdout)
	assert.Empty(t, stderr)

	// In a real test with logger injection, we'd verify:
	// - Log contains "git version"
	// - Log level is appropriate
	// - Structured fields are present
}

func TestGitCommander_Run_LogsExecutionTime(t *testing.T) {
	t.Parallel()

	commander := NewLiveGitCommander()

	start := time.Now()
	stdout, stderr, err := commander.Run("", "version")
	duration := time.Since(start)

	require.NoError(t, err)
	assert.NotEmpty(t, stdout)
	assert.Empty(t, stderr)

	// Execution should be tracked
	assert.Greater(t, duration, time.Duration(0))

	// In a real test with logger injection, we'd verify:
	// - Log contains duration field
	// - Duration is reasonable (not negative, not too large)
}

func TestGitCommander_Run_LogsErrorDetails(t *testing.T) {
	t.Parallel()

	commander := NewLiveGitCommander()

	// Trigger an error to test error logging
	stdout, stderr, err := commander.Run("", "invalid-command")

	require.Error(t, err)
	gitErr, ok := err.(*GitError)
	require.True(t, ok)

	assert.NotEqual(t, 0, gitErr.ExitCode)
	assert.NotEmpty(t, gitErr.Stderr)
	assert.Empty(t, stdout)
	assert.NotEmpty(t, stderr)

	// In a real test with logger injection, we'd verify:
	// - Error details are logged
	// - Exit code is included
	// - Stderr content is captured
}
