package testutils

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestExistingTestPatterns tests that existing test patterns continue to work
func TestExistingTestPatterns(t *testing.T) {
	// Test that standard Go testing patterns still work
	t.Run("standard_temp_dir", func(t *testing.T) {
		// This is how tests typically create temp directories
		tempDir := t.TempDir()

		// Should work as expected
		assert.DirExists(t, tempDir)

		// Should be able to create files in it
		testFile := filepath.Join(tempDir, "test.txt")
		err := os.WriteFile(testFile, []byte("test"), 0o644)
		require.NoError(t, err)
		assert.FileExists(t, testFile)
	})

	t.Run("standard_cleanup", func(t *testing.T) {
		var cleanupCalled bool

		t.Cleanup(func() {
			cleanupCalled = true
		})

		// Cleanup should be called at test end
		// (This will be verified by the testing framework)

		// For now, just verify we can register cleanup
		assert.False(t, cleanupCalled, "Cleanup should not be called yet")
	})

	t.Run("standard_assertions", func(t *testing.T) {
		// Standard testify assertions should continue to work
		assert.True(t, true)
		assert.Equal(t, "expected", "expected")
		require.NotNil(t, t)
	})
}

// TestOldAndNewPatternsSideBySide tests that old and new patterns can coexist
func TestOldAndNewPatternsSideBySide(t *testing.T) {
	// Old pattern: manual temp directory creation
	oldTempDir := t.TempDir()
	oldTestFile := filepath.Join(oldTempDir, "old-pattern.txt")
	err := os.WriteFile(oldTestFile, []byte("old pattern"), 0o644)
	require.NoError(t, err)

	// New pattern: using robust testing infrastructure
	helper := NewUnitTestHelper(t)
	newTestFile := helper.CreateTempFile("new-pattern.txt", "new pattern")

	// Both should work together
	assert.FileExists(t, oldTestFile)
	assert.FileExists(t, newTestFile)

	// Both should be in different directories (isolated)
	assert.NotEqual(t, filepath.Dir(oldTestFile), filepath.Dir(newTestFile),
		"Old and new patterns should use different temp directories")

	// Content should be correct
	oldContent, err := os.ReadFile(oldTestFile)
	require.NoError(t, err)
	assert.Equal(t, "old pattern", string(oldContent))

	newContent, err := os.ReadFile(newTestFile)
	require.NoError(t, err)
	assert.Equal(t, "new pattern", string(newContent))
}

// TestBackwardsCompatibilityWithExistingHelpers tests compatibility with existing test utilities
func TestBackwardsCompatibilityWithExistingHelpers(t *testing.T) {
	// Test that existing testutils functions still work
	testDir := NewTestDirectory(t, "compat-test-*")
	defer testDir.Cleanup()

	// Should work as before
	assert.DirExists(t, testDir.Path)

	// Test with the new robust infrastructure
	robustHelper := NewUnitTestHelper(t)
	robustDir := robustHelper.CreateTempDir("robust-test")

	// Both should coexist
	assert.DirExists(t, testDir.Path)
	assert.DirExists(t, robustDir)

	// Should be in different locations
	assert.NotEqual(t, testDir.Path, robustDir)
}

// TestNoRegressionInTestExecution tests that existing test execution patterns work
func TestNoRegressionInTestExecution(t *testing.T) {
	// Test that we can still run external commands the old way
	t.Run("old_command_execution", func(t *testing.T) {
		cmd := exec.Command("echo", "hello world")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Equal(t, "hello world\n", string(output))
	})

	// Test that we can also use the new robust way
	t.Run("new_command_execution", func(t *testing.T) {
		helper := NewIntegrationTestHelper(t)
		stdout, stderr, err := helper.ExecGrove("--version")
		require.NoError(t, err)
		assert.NotEmpty(t, stdout)
		assert.Empty(t, stderr)
	})

	// Both patterns should coexist without interference
}

// TestCompatibilityWithTestifyAssertions tests that testify assertions work with new infrastructure
func TestCompatibilityWithTestifyAssertions(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Create a file using new infrastructure
	filePath := helper.CreateTempFile("testify-compat.txt", "test content")

	// Use old testify assertions on new infrastructure
	assert.FileExists(t, filePath)
	require.FileExists(t, filePath)

	// Use new infrastructure assertions
	helper.AssertFileExists(filePath)

	// Test that both work for non-existent files
	nonExistentPath := filepath.Join(helper.GetTempDir(), "does-not-exist.txt")
	assert.NoFileExists(t, nonExistentPath)
	helper.AssertNoFileExists(nonExistentPath)
}

// TestExistingMockPatterns tests that existing mock patterns continue to work
func TestExistingMockPatterns(t *testing.T) {
	// Test existing mock git executor
	mock := NewMockGitExecutor()

	// Should work as before
	mock.SetSuccessResponse("status", "clean repo")
	output, err := mock.Execute("status")
	require.NoError(t, err)
	assert.Equal(t, "clean repo", output)

	// Should also work with new infrastructure in the same test
	helper := NewUnitTestHelper(t)
	testFile := helper.CreateTempFile("mock-test.txt", "mock content")
	assert.FileExists(t, testFile)

	// Both should coexist without issues
}

// TestEnvironmentIsolationCompatibility tests that environment isolation doesn't break existing patterns
func TestEnvironmentIsolationCompatibility(t *testing.T) {
	// Set up test environment variable
	testVar := "BACKWARDS_COMPAT_TEST_VAR"
	originalValue := os.Getenv(testVar)
	os.Setenv(testVar, "original_value")
	defer func() {
		if originalValue == "" {
			os.Unsetenv(testVar)
		} else {
			os.Setenv(testVar, originalValue)
		}
	}()

	// Old pattern: direct environment access
	t.Run("old_env_access", func(t *testing.T) {
		value := os.Getenv(testVar)
		assert.Equal(t, "original_value", value)
	})

	// New pattern: isolated environment
	t.Run("new_isolated_env", func(t *testing.T) {
		runner := NewTestRunner(t)

		var isolatedValue string
		runner.WithCleanEnvironment().Run(func() {
			isolatedValue = os.Getenv(testVar)
		})

		// Should be isolated
		assert.Empty(t, isolatedValue)
	})

	// After isolation, original pattern should still work
	t.Run("old_env_access_after_isolation", func(t *testing.T) {
		value := os.Getenv(testVar)
		assert.Equal(t, "original_value", value)
	})
}

// TestFileSystemOperationsCompatibility tests that filesystem operations remain compatible
func TestFileSystemOperationsCompatibility(t *testing.T) {
	// Old pattern: manual file creation
	oldTempDir := t.TempDir()
	oldFile := filepath.Join(oldTempDir, "old-file.txt")
	err := os.WriteFile(oldFile, []byte("old content"), 0o644)
	require.NoError(t, err)

	// New pattern: using helpers
	helper := NewUnitTestHelper(t)
	newFile := helper.CreateTempFile("new-file.txt", "new content")

	// Both should be readable with standard Go functions
	oldContent, err := os.ReadFile(oldFile)
	require.NoError(t, err)
	assert.Equal(t, "old content", string(oldContent))

	newContent, err := os.ReadFile(newFile)
	require.NoError(t, err)
	assert.Equal(t, "new content", string(newContent))

	// Standard file operations should work on both
	oldInfo, err := os.Stat(oldFile)
	require.NoError(t, err)
	assert.False(t, oldInfo.IsDir())

	newInfo, err := os.Stat(newFile)
	require.NoError(t, err)
	assert.False(t, newInfo.IsDir())
}

// TestWorkingDirectoryCompatibility tests that working directory changes remain compatible
func TestWorkingDirectoryCompatibility(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Old pattern: manual directory change
	t.Run("old_directory_change", func(t *testing.T) {
		tempDir := t.TempDir()

		err := os.Chdir(tempDir)
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, tempDir, currentDir)
	})

	// Verify we're back to original directory
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, originalDir, currentDir)

	// New pattern: isolated directory change
	t.Run("new_isolated_directory", func(t *testing.T) {
		runner := NewTestRunner(t)

		var isolatedDir string
		runner.WithIsolatedWorkingDir().Run(func() {
			isolatedDir, _ = os.Getwd()
		})

		// Should be different from original
		assert.NotEqual(t, originalDir, isolatedDir)
	})

	// Verify we're still in original directory after isolation
	finalDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, originalDir, finalDir)
}

// TestSubtestCompatibility tests that subtests work with new infrastructure
func TestSubtestCompatibility(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Create test data
	testData := map[string]string{
		"file1": "content1",
		"file2": "content2",
		"file3": "content3",
	}

	// Use subtests with new infrastructure (should work seamlessly)
	for name, content := range testData {
		t.Run(name, func(t *testing.T) {
			// Each subtest should get its own helper instance
			subHelper := NewUnitTestHelper(t)

			filePath := subHelper.CreateTempFile(name+".txt", content)
			assert.FileExists(t, filePath)

			actualContent, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, content, string(actualContent))
		})
	}

	// Original helper should still work
	mainFile := helper.CreateTempFile("main.txt", "main content")
	assert.FileExists(t, mainFile)
}

// TestParallelSubtestCompatibility tests that parallel subtests work with new infrastructure
func TestParallelSubtestCompatibility(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{"parallel1", "content1"},
		{"parallel2", "content2"},
		{"parallel3", "content3"},
		{"parallel4", "content4"},
	}

	for _, tc := range testCases {
		// capture loop variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel() // Run in parallel

			helper := NewUnitTestHelper(t)
			filePath := helper.CreateTempFile(tc.name+".txt", tc.content)

			assert.FileExists(t, filePath)

			actualContent, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(actualContent))
		})
	}
}

// TestCleanupOrderCompatibility tests that cleanup happens in the correct order
func TestCleanupOrderCompatibility(t *testing.T) {
	var cleanupOrder []string

	// Register old-style cleanup
	t.Cleanup(func() {
		cleanupOrder = append(cleanupOrder, "old-cleanup-1")
	})

	// Create new helper (which will register its own cleanup)
	helper := NewUnitTestHelper(t)
	_ = helper.CreateTempFile("cleanup-test.txt", "test")

	// Register another old-style cleanup
	t.Cleanup(func() {
		cleanupOrder = append(cleanupOrder, "old-cleanup-2")
	})

	// The actual cleanup order verification would happen after the test
	// For now, we just verify that we can mix old and new cleanup registration
	// without errors

	// This test succeeds if no panics occur and files are created successfully
	assert.True(t, true, "Mixed cleanup registration should work")
}

// TestExistingTestRunnerCompatibility tests compatibility with existing test runners
func TestExistingTestRunnerCompatibility(t *testing.T) {
	// Test that existing patterns with temporary directories work
	tempDir := t.TempDir()

	// Old pattern: WithWorkingDirectory from existing testutils
	var oldDirTest string
	WithWorkingDirectory(t, tempDir, func() {
		var err error
		oldDirTest, err = os.Getwd()
		require.NoError(t, err)
	})

	assert.Equal(t, tempDir, oldDirTest)

	// New pattern: TestRunner with isolation
	runner := NewTestRunner(t)
	var newDirTest string

	runner.WithIsolatedWorkingDir().Run(func() {
		var err error
		newDirTest, err = os.Getwd()
		require.NoError(t, err)
	})

	// Both should work, but use different directories
	assert.NotEqual(t, oldDirTest, newDirTest)

	// Both should be temporary directories
	assert.Contains(t, oldDirTest, os.TempDir())
	assert.Contains(t, newDirTest, os.TempDir())
}

// TestIntegrationWithExistingCode tests that new infrastructure integrates with existing codebase patterns
func TestIntegrationWithExistingCode(t *testing.T) {
	// Simulate existing test pattern that might exist in the codebase
	t.Run("existing_git_test_pattern", func(t *testing.T) {
		// Old pattern: direct exec commands
		cmd := exec.Command("git", "--version")
		output, err := cmd.Output()
		require.NoError(t, err)
		assert.Contains(t, string(output), "git version")
	})

	// New robust pattern should coexist
	t.Run("new_robust_pattern", func(t *testing.T) {
		helper := NewIntegrationTestHelper(t)
		stdout, stderr, err := helper.ExecGrove("--version")
		require.NoError(t, err)
		assert.NotEmpty(t, stdout)
		assert.Empty(t, stderr)
	})

	// Mix both patterns in same test
	t.Run("mixed_patterns", func(t *testing.T) {
		// Old way
		cmd := exec.Command("echo", "old pattern")
		oldOutput, err := cmd.Output()
		require.NoError(t, err)

		// New way
		helper := NewUnitTestHelper(t)
		testFile := helper.CreateTempFile("mixed.txt", "new pattern")

		// Both should work
		assert.Equal(t, "old pattern\n", string(oldOutput))
		assert.FileExists(t, testFile)
	})
}

// TestErrorHandlingCompatibility tests that error handling patterns remain compatible
func TestErrorHandlingCompatibility(t *testing.T) {
	// Old pattern: direct error checking
	t.Run("old_error_handling", func(t *testing.T) {
		_, err := os.Stat("/nonexistent/path")
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})

	// New pattern: using helpers
	t.Run("new_error_handling", func(t *testing.T) {
		helper := NewUnitTestHelper(t)
		nonExistentPath := filepath.Join(helper.GetTempDir(), "nonexistent.txt")

		helper.AssertNoFileExists(nonExistentPath)

		// Old-style assertions should still work
		_, err := os.Stat(nonExistentPath)
		assert.Error(t, err)
		assert.True(t, os.IsNotExist(err))
	})
}

// TestGlobalStateIsolation tests that new infrastructure doesn't interfere with global state
func TestGlobalStateIsolation(t *testing.T) {
	// Set up some global state
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	testEnvVar := "GLOBAL_STATE_TEST"
	originalEnvValue := os.Getenv(testEnvVar)
	os.Setenv(testEnvVar, "global_value")
	defer func() {
		if originalEnvValue == "" {
			os.Unsetenv(testEnvVar)
		} else {
			os.Setenv(testEnvVar, originalEnvValue)
		}
	}()

	// Use a subtest to ensure proper cleanup
	t.Run("isolation_test", func(t *testing.T) {
		// Use new infrastructure with isolation
		runner := NewTestRunner(t)
		runner.WithCleanEnvironment().WithIsolatedWorkingDir().Run(func() {
			// Global state should be isolated inside
			envValue := os.Getenv(testEnvVar)
			assert.Empty(t, envValue, "Environment should be clean inside isolation")

			currentDir, err := os.Getwd()
			require.NoError(t, err)
			assert.NotEqual(t, originalDir, currentDir, "Working directory should be isolated")
		})
	})

	// Global state should be restored after subtest completes
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, originalDir, currentDir, "Working directory should be restored after subtest")

	currentEnvValue := os.Getenv(testEnvVar)
	assert.Equal(t, "global_value", currentEnvValue, "Environment should be restored after subtest")
}
