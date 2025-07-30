//go:build !integration
// +build !integration

package testutils

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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

// Test IntegrationTestHelper infrastructure

func TestIntegrationTestHelper_findProjectRoot(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	projectRoot, err := helper.findProjectRoot()
	require.NoError(t, err, "Should find project root")

	// Verify we found a directory with go.mod
	goModPath := filepath.Join(projectRoot, "go.mod")
	assert.FileExists(t, goModPath, "Project root should contain go.mod")

	// Verify the project root looks correct
	assert.Contains(t, projectRoot, "robust-testing-infrastructure", "Should find correct project")
}

func TestIntegrationTestHelper_GetBinary(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	binaryPath := helper.GetBinary()

	// Verify binary was built successfully
	assert.FileExists(t, binaryPath, "Binary should exist after build")

	// Verify binary is executable
	fileInfo, err := os.Stat(binaryPath)
	require.NoError(t, err)

	// On Unix systems, check executable bit
	if runtime.GOOS != windowsOS {
		mode := fileInfo.Mode()
		assert.True(t, mode&0o111 != 0, "Binary should be executable")
	}

	// Verify binary name includes platform-specific extension
	if runtime.GOOS == windowsOS {
		assert.True(t, strings.HasSuffix(binaryPath, ".exe"), "Windows binary should have .exe extension")
	} else {
		assert.False(t, strings.HasSuffix(binaryPath, ".exe"), "Non-Windows binary should not have .exe extension")
	}
}

func TestIntegrationTestHelper_GetBinary_Caching(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Call GetBinary multiple times
	binary1 := helper.GetBinary()
	binary2 := helper.GetBinary()
	binary3 := helper.GetBinary()

	// Should return the same path every time (cached)
	assert.Equal(t, binary1, binary2, "Binary path should be cached")
	assert.Equal(t, binary2, binary3, "Binary path should be cached")

	// Verify only one binary exists in temp directory
	tempDir := helper.GetTempDir()
	entries, err := os.ReadDir(tempDir)
	require.NoError(t, err)

	binaryCount := 0
	for _, entry := range entries {
		if strings.Contains(entry.Name(), "grove") {
			binaryCount++
		}
	}

	assert.Equal(t, 1, binaryCount, "Should only build binary once")
}

func TestIntegrationTestHelper_ExecGrove(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Test basic execution (version command should work)
	stdout, stderr, err := helper.ExecGrove("--version")

	// Command should succeed and return version info
	require.NoError(t, err, "Grove --version should succeed")
	assert.NotEmpty(t, stdout, "Version output should not be empty")
	assert.Contains(t, stdout, "grove", "Output should contain 'grove'")
	assert.Empty(t, stderr, "Stderr should be empty for version command")
}

func TestIntegrationTestHelper_ExecGroveInDir(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Create a test directory
	testDir := filepath.Join(helper.GetTempDir(), "test-exec-dir")
	err := os.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	// Execute in the specific directory
	stdout, stderr, err := helper.ExecGroveInDir(testDir, "--version")

	// Should work the same as regular execution
	require.NoError(t, err, "Grove --version in dir should succeed")
	assert.NotEmpty(t, stdout, "Version output should not be empty")
	assert.Contains(t, stdout, "grove", "Output should contain 'grove'")
	assert.Empty(t, stderr, "Stderr should be empty")
}

func TestIntegrationTestHelper_CleanEnvironment(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Get clean environment
	cleanEnv := helper.getCleanEnvironment()

	// Should contain essential variables
	hasPath := false
	hasHome := false
	hasTmpdir := false

	for _, env := range cleanEnv {
		if strings.HasPrefix(env, "PATH=") {
			hasPath = true
		}
		if strings.HasPrefix(env, "HOME=") {
			hasHome = true
		}
		if strings.HasPrefix(env, "TMPDIR=") {
			hasTmpdir = true
		}
	}

	assert.True(t, hasPath, "Clean environment should include PATH")
	assert.True(t, hasHome, "Clean environment should include HOME")
	assert.True(t, hasTmpdir, "Clean environment should include TMPDIR")

	// Should be minimal (not too many variables)
	assert.Less(t, len(cleanEnv), 20, "Clean environment should be minimal")
}

// Test UnitTestHelper infrastructure

func TestUnitTestHelper_GetUniqueTestPath(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Generate multiple unique paths
	path1 := helper.GetUniqueTestPath("test1")
	path2 := helper.GetUniqueTestPath("test2")
	path3 := helper.GetUniqueTestPath("test1") // Same suffix, should still be unique

	// All paths should be different
	assert.NotEqual(t, path1, path2, "Paths with different suffixes should be unique")
	assert.NotEqual(t, path1, path3, "Paths with same suffix should still be unique")
	assert.NotEqual(t, path2, path3, "All paths should be unique")

	// All paths should contain the test name
	for _, path := range []string{path1, path2, path3} {
		assert.Contains(t, path, "TestUnitTestHelper_GetUniqueTestPath", "Path should contain test name")
		assert.Contains(t, path, "/tmp/grove-unit-test/", "Path should be in expected location")
	}
}

func TestUnitTestHelper_CreateTempFile(t *testing.T) {
	helper := NewUnitTestHelper(t)

	content := "test file content\nwith multiple lines"
	filePath := helper.CreateTempFile("test.txt", content)

	// File should exist
	assert.FileExists(t, filePath, "Temp file should exist")

	// File should contain expected content
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, content, string(actualContent), "File should contain expected content")

	// File should be in helper's temp directory
	assert.Contains(t, filePath, helper.GetTempDir(), "File should be in temp directory")
}

func TestUnitTestHelper_CreateTempDir(t *testing.T) {
	helper := NewUnitTestHelper(t)

	dirPath := helper.CreateTempDir("nested/test/directory")

	// Directory should exist
	assert.DirExists(t, dirPath, "Temp directory should exist")

	// Should be able to create files in it
	testFile := filepath.Join(dirPath, "test.txt")
	err := os.WriteFile(testFile, []byte("test"), 0o644)
	require.NoError(t, err)
	assert.FileExists(t, testFile)

	// Directory should be in helper's temp directory
	assert.Contains(t, dirPath, helper.GetTempDir(), "Directory should be in temp directory")
}

func TestUnitTestHelper_FileAssertions(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Test file that exists
	existingFile := helper.CreateTempFile("exists.txt", "content")
	helper.AssertFileExists(existingFile) // Should not panic/fail

	// Test file that doesn't exist
	nonExistentFile := filepath.Join(helper.GetTempDir(), "does-not-exist.txt")
	helper.AssertNoFileExists(nonExistentFile) // Should not panic/fail
}

func TestUnitTestHelper_WithCleanFilesystem(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Create some test artifacts that should be cleaned
	testPattern := "/tmp/test-cleanup-*"
	testFile1 := "/tmp/test-cleanup-1"
	testFile2 := "/tmp/test-cleanup-2"

	// Create test files
	err := os.WriteFile(testFile1, []byte("test1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(testFile2, []byte("test2"), 0o644)
	require.NoError(t, err)

	// Files should exist before cleanup
	assert.FileExists(t, testFile1)
	assert.FileExists(t, testFile2)

	// Apply cleanup with custom pattern
	helper.WithCleanFilesystem(testPattern)

	// Files should be cleaned up
	assert.NoFileExists(t, testFile1)
	assert.NoFileExists(t, testFile2)
}

// Test TestRunner infrastructure

func TestTestRunner_WithIsolatedWorkingDir(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	runner := NewTestRunner(t)
	runner.WithIsolatedWorkingDir()

	// Should be in a different directory now
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	assert.NotEqual(t, originalDir, currentDir, "Should be in isolated directory")

	// Directory should be a temporary directory
	assert.Contains(t, currentDir, os.TempDir(), "Should be in temp directory")

	// After test cleanup, should restore original directory
	// Note: This will be tested by the testing framework's cleanup
}

func TestTestRunner_WithCleanEnvironment(t *testing.T) {
	// Set a test environment variable
	originalValue := os.Getenv("TEST_RUNNER_TEST_VAR")
	require.NoError(t, os.Setenv("TEST_RUNNER_TEST_VAR", "original_value"))
	defer func() {
		if originalValue == "" {
			if err := os.Unsetenv("TEST_RUNNER_TEST_VAR"); err != nil {
				t.Logf("Warning: Failed to unset environment variable: %v", err)
			}
		} else {
			if err := os.Setenv("TEST_RUNNER_TEST_VAR", originalValue); err != nil {
				t.Logf("Warning: Failed to restore environment variable: %v", err)
			}
		}
	}()

	runner := NewTestRunner(t)

	// Apply clean environment
	runner.WithCleanEnvironment()

	// Test variable should be cleared in clean environment
	cleanValue := os.Getenv("TEST_RUNNER_TEST_VAR")
	assert.Empty(t, cleanValue, "Custom environment variable should be cleared")

	// Essential variables should still be present
	assert.NotEmpty(t, os.Getenv("PATH"), "PATH should be preserved")
	assert.NotEmpty(t, os.Getenv("HOME"), "HOME should be preserved")

	// After test cleanup, environment should be restored
	// Note: This will be tested by the testing framework's cleanup
}

func TestTestRunner_Run(t *testing.T) {
	runner := NewTestRunner(t)

	executed := false

	runner.Run(func() {
		executed = true
	})

	assert.True(t, executed, "Test function should be executed")
}
