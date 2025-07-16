package testutils

import (
	"os"
	"testing"
)

// CleanupFunc represents a function that cleans up test resources.
type CleanupFunc func()

// Cleanup manages test resource cleanup with automatic defer support.
type Cleanup struct {
	t        *testing.T
	cleanups []CleanupFunc
	hasRun   bool
}

// NewCleanup creates a new cleanup manager.
func NewCleanup(t *testing.T) *Cleanup {
	t.Helper()
	return &Cleanup{
		t:        t,
		cleanups: make([]CleanupFunc, 0),
		hasRun:   false,
	}
}

// Add adds a cleanup function to be executed later.
func (c *Cleanup) Add(fn CleanupFunc) {
	c.cleanups = append(c.cleanups, fn)
}

// AddDir adds a directory to be removed during cleanup.
func (c *Cleanup) AddDir(dir string) {
	c.Add(func() {
		_ = os.RemoveAll(dir)
	})
}

// AddFile adds a file to be removed during cleanup.
func (c *Cleanup) AddFile(file string) {
	c.Add(func() {
		_ = os.Remove(file)
	})
}

// Run executes all cleanup functions in reverse order (LIFO).
func (c *Cleanup) Run() {
	if c.hasRun {
		return
	}

	c.hasRun = true

	// Execute cleanups in reverse order (LIFO)
	for i := len(c.cleanups) - 1; i >= 0; i-- {
		c.cleanups[i]()
	}
}

// Defer registers the cleanup to run automatically when the test ends.
func (c *Cleanup) Defer() {
	c.t.Cleanup(c.Run)
}

// RestoreWorkingDirectory creates a cleanup function that restores the working directory.
func RestoreWorkingDirectory(t *testing.T) CleanupFunc {
	t.Helper()

	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}

	return func() {
		err := os.Chdir(originalDir)
		if err != nil {
			t.Logf("Failed to restore working directory: %v", err)
		}
	}
}

// RestoreEnvVar creates a cleanup function that restores an environment variable.
func RestoreEnvVar(t *testing.T, key string) CleanupFunc {
	t.Helper()

	originalValue, existed := os.LookupEnv(key)

	return func() {
		if existed {
			err := os.Setenv(key, originalValue)
			if err != nil {
				t.Logf("Failed to restore environment variable %s: %v", key, err)
			}
		} else {
			err := os.Unsetenv(key)
			if err != nil {
				t.Logf("Failed to unset environment variable %s: %v", key, err)
			}
		}
	}
}

// SetupTestEnvironment creates a clean test environment with automatic cleanup.
func SetupTestEnvironment(t *testing.T, prefix string) (*TestDirectory, *Cleanup) {
	t.Helper()

	cleanup := NewCleanup(t)
	cleanup.Defer()

	// Create test directory
	testDir := NewTestDirectory(t, prefix)
	cleanup.AddDir(testDir.Path)

	// Save and restore working directory
	cleanup.Add(RestoreWorkingDirectory(t))

	return testDir, cleanup
}
