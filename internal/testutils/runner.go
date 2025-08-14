package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestRunner provides environment isolation capabilities for tests.
// It ensures tests don't interfere with each other or the system environment.
type TestRunner struct {
	t                *testing.T
	originalEnv      map[string]string
	originalWorkDir  string
	cleanEnvironment bool
	isolatedWorkDir  bool
}

// NewTestRunner creates a new TestRunner instance for environment isolation.
func NewTestRunner(t *testing.T) *TestRunner {
	return &TestRunner{
		t:           t,
		originalEnv: make(map[string]string),
	}
}

// WithCleanEnvironment configures the runner to use a clean environment.
// This removes all environment variables except essential ones.
func (r *TestRunner) WithCleanEnvironment() *TestRunner {
	r.cleanEnvironment = true
	return r
}

// WithIsolatedWorkingDir configures the runner to use an isolated working directory.
// This prevents tests from affecting each other through working directory changes.
func (r *TestRunner) WithIsolatedWorkingDir() *TestRunner {
	r.isolatedWorkDir = true
	return r
}

// Run executes the test function with the configured isolation settings.
// Environment and working directory are automatically restored even if the test panics.
func (r *TestRunner) Run(testFunc func()) {
	r.t.Helper()

	// Save original environment if clean environment is requested.
	if r.cleanEnvironment {
		r.saveEnvironment()
		defer r.restoreEnvironment() // Ensures cleanup even on panic
		r.setupCleanEnvironment()
	}

	// Save and isolate working directory if requested.
	if r.isolatedWorkDir {
		r.saveWorkingDir()
		defer r.restoreWorkingDir() // Ensures cleanup even on panic
		r.setupIsolatedWorkDir()
	}

	// Execute the test function.
	testFunc()
}

// saveEnvironment saves the current environment variables.
func (r *TestRunner) saveEnvironment() {
	r.t.Helper()

	for _, env := range os.Environ() {
		if idx := indexOf(env, '='); idx != -1 {
			key := env[:idx]
			value := env[idx+1:]
			r.originalEnv[key] = value
		}
	}
}

// restoreEnvironment restores the original environment variables.
func (r *TestRunner) restoreEnvironment() {
	r.t.Helper()

	// Clear all current environment variables.
	os.Clearenv()

	// Restore original environment.
	for key, value := range r.originalEnv {
		_ = os.Setenv(key, value)
	}
}

// setupCleanEnvironment sets up a minimal clean environment.
func (r *TestRunner) setupCleanEnvironment() {
	r.t.Helper()

	// Clear all environment variables.
	os.Clearenv()

	// Set only essential environment variables.
	essentialVars := map[string]string{
		"PATH":        r.originalEnv["PATH"], // Keep PATH for finding executables.
		"HOME":        r.t.TempDir(),         // Use temporary home directory.
		"TMPDIR":      r.t.TempDir(),         // Use temporary directory for temp files.
		"TEMP":        r.t.TempDir(),         // Windows temp directory.
		"TMP":         r.t.TempDir(),         // Alternative temp directory.
		"USERPROFILE": r.t.TempDir(),         // Windows home directory.
	}

	// Apply essential variables.
	for key, value := range essentialVars {
		if value != "" {
			_ = os.Setenv(key, value)
		}
	}

	// Set test-specific environment variables.
	_ = os.Setenv("GROVE_TEST_MODE", "1")
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	_ = os.Setenv("GIT_CONFIG_GLOBAL", "/dev/null")
}

// saveWorkingDir saves the current working directory.
func (r *TestRunner) saveWorkingDir() {
	r.t.Helper()

	var err error
	r.originalWorkDir, err = os.Getwd()
	require.NoError(r.t, err, "failed to get current working directory")
}

// restoreWorkingDir restores the original working directory.
func (r *TestRunner) restoreWorkingDir() {
	r.t.Helper()

	if r.originalWorkDir != "" {
		err := os.Chdir(r.originalWorkDir)
		// Best effort - don't fail the test if we can't restore.
		if err != nil {
			r.t.Logf("Warning: failed to restore working directory: %v", err)
		}
	}
}

// setupIsolatedWorkDir sets up an isolated working directory.
func (r *TestRunner) setupIsolatedWorkDir() {
	r.t.Helper()

	// Create a temporary directory for the test.
	tempDir := r.t.TempDir()

	// Change to the temporary directory.
	err := os.Chdir(tempDir)
	require.NoError(r.t, err, "failed to change to temporary directory")
}

// indexOf finds the index of the first occurrence of sep in s.
func indexOf(s string, sep rune) int {
	for i, c := range s {
		if c == sep {
			return i
		}
	}
	return -1
}
