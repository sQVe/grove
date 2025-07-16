package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestDirectory represents a temporary directory for testing with cleanup.
type TestDirectory struct {
	Path string
	t    *testing.T
}

// NewTestDirectory creates a temporary directory for testing.
func NewTestDirectory(t *testing.T, prefix string) *TestDirectory {
	t.Helper()

	tempDir, err := os.MkdirTemp("", prefix)
	require.NoError(t, err)

	return &TestDirectory{
		Path: tempDir,
		t:    t,
	}
}

// Cleanup removes the temporary directory and all its contents.
func (td *TestDirectory) Cleanup() {
	td.t.Helper()
	_ = os.RemoveAll(td.Path)
}

// WithWorkingDirectory executes a function with the working directory set to the repository.
func WithWorkingDirectory(t *testing.T, dir string, callback func()) {
	t.Helper()

	originalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(dir)
	require.NoError(t, err)

	defer func() {
		err := os.Chdir(originalDir)
		require.NoError(t, err)
	}()

	callback()
}
