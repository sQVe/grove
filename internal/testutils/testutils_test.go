package testutils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockGitExecutor(t *testing.T) {
	mock := NewMockGitExecutor()

	// Test unhandled command
	_, err := mock.Execute("unknown", "command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mock: unhandled git command")

	// Test setting responses
	mock.SetSuccessResponse("status", "clean")
	output, err := mock.Execute("status")
	require.NoError(t, err)
	assert.Equal(t, "clean", output)

	// Test error response
	mock.SetErrorResponse("fail", assert.AnError)
	_, err = mock.Execute("fail")
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)

	// Test safe repository state
	mock.SetSafeRepositoryState()
	output, err = mock.Execute("status", "--porcelain=v1")
	require.NoError(t, err)
	assert.Empty(t, output)
}

func TestTestDirectory(t *testing.T) {
	testDir := NewTestDirectory(t, "testutils-test-*")
	defer testDir.Cleanup()

	// Verify directory exists
	assert.DirExists(t, testDir.Path)

	// Test creating file in directory
	testFile := filepath.Join(testDir.Path, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Verify file exists
	assert.FileExists(t, testFile)
}

func TestAssertDirectoryEmpty(t *testing.T) {
	testDir := NewTestDirectory(t, "testutils-empty-*")
	defer testDir.Cleanup()

	// Should pass for empty directory
	AssertDirectoryEmpty(t, testDir.Path)

	// Create a file and test again
	testFile := filepath.Join(testDir.Path, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0o644)
	require.NoError(t, err)

	// Should pass for directory with hidden files only
	hiddenFile := filepath.Join(testDir.Path, ".hidden")
	err = os.WriteFile(hiddenFile, []byte("hidden"), 0o644)
	require.NoError(t, err)

	// Should fail for directory with visible files
	AssertDirectoryNotEmpty(t, testDir.Path)
}

func TestCleanup(t *testing.T) {
	cleanup := NewCleanup(t)

	// Test adding cleanup functions
	called := false

	cleanup.Add(func() {
		called = true
	})

	// Run cleanup
	cleanup.Run()
	assert.True(t, called)

	// Test multiple runs don't execute again
	called = false
	cleanup.Run()
	assert.False(t, called)
}

func TestWithWorkingDirectory(t *testing.T) {
	testDir := NewTestDirectory(t, "testutils-wd-*")
	defer testDir.Cleanup()

	originalDir, err := os.Getwd()
	require.NoError(t, err)

	var currentDir string

	WithWorkingDirectory(t, testDir.Path, func() {
		currentDir, err = os.Getwd()
		require.NoError(t, err)
	})

	// Verify we were in the test directory
	assert.Equal(t, testDir.Path, currentDir)

	// Verify we're back to original directory
	finalDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, originalDir, finalDir)
}

func TestAssertFileContent(t *testing.T) {
	testDir := NewTestDirectory(t, "testutils-content-*")
	defer testDir.Cleanup()

	testFile := filepath.Join(testDir.Path, "test.txt")
	content := "expected content"
	err := os.WriteFile(testFile, []byte(content), 0o644)
	require.NoError(t, err)

	AssertFileContent(t, testFile, content)
}

func TestAssertErrorContains(t *testing.T) {
	err := assert.AnError
	AssertErrorContains(t, err, "assert.AnError")
}
