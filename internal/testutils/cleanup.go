package testutils

import (
	"os"
	"testing"
)

type CleanupFunc func()

type Cleanup struct {
	t        *testing.T
	cleanups []CleanupFunc
	hasRun   bool
}

func NewCleanup(t *testing.T) *Cleanup {
	t.Helper()
	return &Cleanup{
		t:        t,
		cleanups: make([]CleanupFunc, 0),
		hasRun:   false,
	}
}

func (c *Cleanup) Add(fn CleanupFunc) {
	c.cleanups = append(c.cleanups, fn)
}

func (c *Cleanup) AddDir(dir string) {
	c.Add(func() {
		_ = os.RemoveAll(dir)
	})
}

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

	for i := len(c.cleanups) - 1; i >= 0; i-- {
		c.cleanups[i]()
	}
}

func (c *Cleanup) Defer() {
	c.t.Cleanup(c.Run)
}

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

func SetupTestEnvironment(t *testing.T, prefix string) (*TestDirectory, *Cleanup) {
	t.Helper()

	cleanup := NewCleanup(t)
	cleanup.Defer()

	testDir := NewTestDirectory(t, prefix)
	cleanup.AddDir(testDir.Path)

	cleanup.Add(RestoreWorkingDirectory(t))

	return testDir, cleanup
}
