package testutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunner provides consistent test execution environment
type TestRunner struct {
	t           *testing.T
	originalDir string
	cleanupFns  []func()
}

// NewTestRunner creates a new test runner with proper isolation
func NewTestRunner(t *testing.T) *TestRunner {
	t.Helper()

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	runner := &TestRunner{
		t:           t,
		originalDir: originalDir,
		cleanupFns:  make([]func(), 0),
	}

	// Register cleanup
	t.Cleanup(func() {
		runner.cleanup()
	})

	return runner
}

// WithIsolatedWorkingDir creates an isolated working directory for the test
func (r *TestRunner) WithIsolatedWorkingDir() *TestRunner {
	r.t.Helper()

	tempDir := r.t.TempDir()

	err := os.Chdir(tempDir)
	require.NoError(r.t, err, "Failed to change to temp directory")

	// Register cleanup to restore original directory
	r.addCleanup(func() {
		_ = os.Chdir(r.originalDir)
	})

	return r
}

// WithCleanEnvironment ensures a clean environment for testing
func (r *TestRunner) WithCleanEnvironment() *TestRunner {
	r.t.Helper()

	// Store original environment
	originalEnv := os.Environ()

	// Set minimal environment
	cleanEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"TMPDIR=" + r.t.TempDir(),
	}

	// Add Go-specific variables
	for _, env := range originalEnv {
		if strings.HasPrefix(env, "GO") {
			cleanEnv = append(cleanEnv, env)
		}
	}

	// Clear and set clean environment
	os.Clearenv()
	for _, env := range cleanEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			os.Setenv(parts[0], parts[1])
		}
	}

	// Register cleanup to restore original environment
	r.addCleanup(func() {
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				os.Setenv(parts[0], parts[1])
			}
		}
	})

	return r
}

// WithCleanFilesystem removes leftover files from previous test runs
func (r *TestRunner) WithCleanFilesystem(patterns ...string) *TestRunner {
	r.t.Helper()

	// Default patterns for common test artifacts
	defaultPatterns := []string{
		"/tmp/grove-*",
		"/tmp/create-cmd-*",
		"/tmp/grove-list-*",
		"/tmp/grove-test*",
		"/tmp/*-grove-*",
	}

	allPatterns := append(defaultPatterns, patterns...)

	for _, pattern := range allPatterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, match := range matches {
			_ = os.RemoveAll(match) // Best effort cleanup
		}
	}

	return r
}

// Run executes the test function with all configured isolation
func (r *TestRunner) Run(testFn func()) {
	r.t.Helper()
	testFn()
}

// addCleanup adds a cleanup function to be called at test end
func (r *TestRunner) addCleanup(fn func()) {
	r.cleanupFns = append(r.cleanupFns, fn)
}

// cleanup performs all registered cleanup operations
func (r *TestRunner) cleanup() {
	// Execute cleanup functions in reverse order
	for i := len(r.cleanupFns) - 1; i >= 0; i-- {
		r.cleanupFns[i]()
	}
}
