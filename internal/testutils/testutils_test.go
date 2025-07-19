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

	_, err := mock.Execute("unknown", "command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mock: unhandled git command")

	mock.SetSuccessResponse("status", "clean")
	output, err := mock.Execute("status")
	require.NoError(t, err)
	assert.Equal(t, "clean", output)

	mock.SetErrorResponse("fail", assert.AnError)
	_, err = mock.Execute("fail")
	require.Error(t, err)
	assert.Equal(t, assert.AnError, err)

	mock.SetSafeRepositoryState()
	output, err = mock.Execute("status", "--porcelain=v1")
	require.NoError(t, err)
	assert.Empty(t, output)
}

func TestTestDirectory(t *testing.T) {
	testDir := NewTestDirectory(t, "testutils-test-*")
	defer testDir.Cleanup()

	assert.DirExists(t, testDir.Path)

	testFile := filepath.Join(testDir.Path, "test.txt")
	err := os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	assert.FileExists(t, testFile)
}

func TestAssertDirectoryEmpty(t *testing.T) {
	testDir := NewTestDirectory(t, "testutils-empty-*")
	defer testDir.Cleanup()

	AssertDirectoryEmpty(t, testDir.Path)

	testFile := filepath.Join(testDir.Path, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0o644)
	require.NoError(t, err)

	hiddenFile := filepath.Join(testDir.Path, ".hidden")
	err = os.WriteFile(hiddenFile, []byte("hidden"), 0o644)
	require.NoError(t, err)

	AssertDirectoryNotEmpty(t, testDir.Path)
}

func TestCleanup(t *testing.T) {
	cleanup := NewCleanup(t)

	called := false

	cleanup.Add(func() {
		called = true
	})

	cleanup.Run()
	assert.True(t, called)

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

	assert.Equal(t, testDir.Path, currentDir)

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
