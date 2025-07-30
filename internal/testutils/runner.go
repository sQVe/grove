package testutils

import (
	"os"
	"os/exec"
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

func NewTestRunner(t *testing.T) *TestRunner {
	t.Helper()

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	runner := &TestRunner{
		t:           t,
		originalDir: originalDir,
		cleanupFns:  make([]func(), 0),
	}

	t.Cleanup(func() {
		runner.cleanup()
	})

	return runner
}

// WithIsolatedWorkingDir creates an isolated working directory for the test.
func (r *TestRunner) WithIsolatedWorkingDir() *TestRunner {
	r.t.Helper()

	tempDir := r.t.TempDir()

	err := os.Chdir(tempDir)
	require.NoError(r.t, err, "Failed to change to temp directory")

	// Register cleanup to restore original directory.
	r.addCleanup(func() {
		_ = os.Chdir(r.originalDir)
	})

	return r
}

// WithCleanEnvironment ensures a clean environment for testing.
func (r *TestRunner) WithCleanEnvironment() *TestRunner {
	r.t.Helper()

	// Store original environment.
	originalEnv := os.Environ()

	// Set minimal environment.
	cleanEnv := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"TMPDIR=" + r.t.TempDir(),
	}

	// Add Go-specific variables, but ensure GOROOT is always valid.
	gorootSet := false
	for _, env := range originalEnv {
		if strings.HasPrefix(env, "GO") {
			// Skip fake GOROOT values that might have been set by tests
			if strings.HasPrefix(env, "GOROOT=") && strings.Contains(env, "/test/") {
				continue
			}
			cleanEnv = append(cleanEnv, env)
			if strings.HasPrefix(env, "GOROOT=") {
				gorootSet = true
			}
		}
	}

	// Ensure GOROOT is always set to a valid value
	if !gorootSet {
		if cmd := exec.Command("go", "env", "GOROOT"); cmd != nil {
			if output, err := cmd.Output(); err == nil {
				if goroot := strings.TrimSpace(string(output)); goroot != "" {
					cleanEnv = append(cleanEnv, "GOROOT="+goroot)
				}
			}
		}
	}

	// Clear and set clean environment.
	os.Clearenv()
	for _, env := range cleanEnv {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			if err := os.Setenv(parts[0], parts[1]); err != nil {
				r.t.Fatalf("Failed to set environment variable %s: %v", parts[0], err)
			}
		}
	}

	// Register cleanup to restore original environment.
	r.addCleanup(func() {
		os.Clearenv()
		for _, env := range originalEnv {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				if err := os.Setenv(parts[0], parts[1]); err != nil {
					r.t.Logf("Warning: Failed to restore environment variable %s: %v", parts[0], err)
				}
			}
		}
	})

	return r
}

// WithCleanFilesystem removes leftover files from previous test runs.
func (r *TestRunner) WithCleanFilesystem(patterns ...string) *TestRunner {
	r.t.Helper()

	// Default patterns for common test artifacts.
	defaultPatterns := []string{
		"/tmp/grove-*",
		"/tmp/create-cmd-*",
		"/tmp/grove-list-*",
		"/tmp/grove-test*",
		"/tmp/*-grove-*",
	}

	defaultPatterns = append(defaultPatterns, patterns...)
	allPatterns := defaultPatterns

	for _, pattern := range allPatterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, match := range matches {
			_ = os.RemoveAll(match) // Best effort cleanup.
		}
	}

	return r
}

func (r *TestRunner) Run(testFn func()) {
	r.t.Helper()
	testFn()
}

func (r *TestRunner) addCleanup(fn func()) {
	r.cleanupFns = append(r.cleanupFns, fn)
}

func (r *TestRunner) cleanup() {
	// Execute cleanup functions in reverse order.
	for i := len(r.cleanupFns) - 1; i >= 0; i-- {
		r.cleanupFns[i]()
	}
}
