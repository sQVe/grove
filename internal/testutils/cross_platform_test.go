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

// TestCrossPlatformBinaryBuilding tests binary building works on all supported platforms
func TestCrossPlatformBinaryBuilding(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Build binary should work on current platform
	binaryPath := helper.GetBinary()

	// Binary should exist
	assert.FileExists(t, binaryPath, "Binary should exist after build")

	// Binary should have correct platform-specific extension
	switch runtime.GOOS {
	case windowsOS:
		assert.True(t, strings.HasSuffix(binaryPath, ".exe"),
			"Windows binary should have .exe extension")
	case "linux", "darwin":
		assert.False(t, strings.HasSuffix(binaryPath, ".exe"),
			"Unix binary should not have .exe extension")
	default:
		t.Logf("Testing on platform: %s", runtime.GOOS)
	}

	// Binary should be executable (on Unix systems)
	if runtime.GOOS != "windows" {
		fileInfo, err := os.Stat(binaryPath)
		require.NoError(t, err)

		mode := fileInfo.Mode()
		assert.True(t, mode&0o111 != 0, "Binary should be executable on Unix systems")
	}

	// Binary should actually run
	stdout, stderr, err := helper.ExecGrove("--version")
	require.NoError(t, err, "Binary should execute successfully")
	assert.NotEmpty(t, stdout, "Version command should produce output")
	assert.Empty(t, stderr, "Version command should not produce errors")
}

// TestCrossPlatformPathHandling tests that path operations work correctly across platforms
func TestCrossPlatformPathHandling(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Test unique path generation with platform-specific separators
	path1 := helper.GetUniqueTestPath("test-file")
	path2 := helper.GetUniqueTestPath("nested/dir/file")

	// Paths should use the correct separator for the platform
	expectedSeparator := string(filepath.Separator)
	assert.Contains(t, path1, expectedSeparator, "Path should use platform separator")
	assert.Contains(t, path2, expectedSeparator, "Nested path should use platform separator")

	// Paths should be valid for the current platform
	assert.True(t, filepath.IsAbs(path1), "Generated path should be absolute")
	assert.True(t, filepath.IsAbs(path2), "Generated nested path should be absolute")

	// Test creating files with platform-specific paths
	testContent := "cross-platform test content"
	filePath := helper.CreateTempFile("cross-platform-test.txt", testContent)

	// File should exist and be readable
	assert.FileExists(t, filePath)
	actualContent, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, testContent, string(actualContent))

	// Test creating nested directories
	nestedDir := helper.CreateTempDir("cross/platform/nested")
	assert.DirExists(t, nestedDir)

	// Should be able to create files in nested directories
	nestedFile := filepath.Join(nestedDir, "nested-file.txt")
	err = os.WriteFile(nestedFile, []byte("nested content"), 0o644)
	require.NoError(t, err)
	assert.FileExists(t, nestedFile)
}

// TestCrossPlatformEnvironmentHandling tests environment variable handling across platforms
func TestCrossPlatformEnvironmentHandling(t *testing.T) {
	runner := NewTestRunner(t)

	// Test platform-specific environment variables
	var platformTestVar string
	var expectedValue string

	switch runtime.GOOS {
	case windowsOS:
		platformTestVar = "USERPROFILE"
		expectedValue = os.Getenv("USERPROFILE")
	case "linux", "darwin":
		platformTestVar = "HOME"
		expectedValue = os.Getenv("HOME")
	default:
		t.Skip("Skipping platform-specific environment test for", runtime.GOOS)
	}

	// Set up test environment
	testVar := "CROSS_PLATFORM_TEST_VAR"
	originalValue := os.Getenv(testVar)
	require.NoError(t, os.Setenv(testVar, "test_value"))
	defer func() {
		if originalValue == "" {
			if err := os.Unsetenv(testVar); err != nil {
				t.Logf("Warning: Failed to unset environment variable: %v", err)
			}
		} else {
			if err := os.Setenv(testVar, originalValue); err != nil {
				t.Logf("Warning: Failed to restore environment variable: %v", err)
			}
		}
	}()

	runner.WithCleanEnvironment().Run(func() {
		// Platform-specific variables should be preserved
		actualValue := os.Getenv(platformTestVar)
		assert.Equal(t, expectedValue, actualValue,
			"Platform-specific environment variable should be preserved")

		// Custom test variable should be cleared
		testValue := os.Getenv(testVar)
		assert.Empty(t, testValue, "Custom environment variable should be cleared")

		// PATH should always be preserved
		pathValue := os.Getenv("PATH")
		assert.NotEmpty(t, pathValue, "PATH should always be preserved")
	})
}

// TestCrossPlatformTempDirectories tests temporary directory handling across platforms
func TestCrossPlatformTempDirectories(t *testing.T) {
	helper := NewUnitTestHelper(t)

	tempDir := helper.GetTempDir()

	// Temp directory should exist
	assert.DirExists(t, tempDir, "Temp directory should exist")

	// Temp directory should be in platform-appropriate location
	platformTempDir := os.TempDir()
	assert.Contains(t, tempDir, platformTempDir,
		"Temp directory should be in platform temp location")

	// Should be able to create files in temp directory
	testFile := filepath.Join(tempDir, "temp-test.txt")
	err := os.WriteFile(testFile, []byte("temp content"), 0o644)
	require.NoError(t, err)
	assert.FileExists(t, testFile)

	// Should handle platform-specific file permissions
	fileInfo, err := os.Stat(testFile)
	require.NoError(t, err)

	mode := fileInfo.Mode()
	if runtime.GOOS != "windows" {
		// Unix systems should respect file permissions
		assert.Equal(t, os.FileMode(0o644), mode&0o777,
			"File should have correct permissions on Unix")
	}
}

// TestCrossPlatformFilesystemCleanup tests filesystem cleanup works across platforms
func TestCrossPlatformFilesystemCleanup(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Create test files in platform-appropriate temp location
	tempDir := os.TempDir()
	testFiles := []string{
		filepath.Join(tempDir, "cross-platform-cleanup-1.txt"),
		filepath.Join(tempDir, "cross-platform-cleanup-2.txt"),
		filepath.Join(tempDir, "cross-platform-cleanup-3.txt"),
	}

	// Create the test files
	for _, filePath := range testFiles {
		err := os.WriteFile(filePath, []byte("cleanup test"), 0o644)
		require.NoError(t, err)
		assert.FileExists(t, filePath)
	}

	// Apply cleanup with platform-appropriate glob pattern
	cleanupPattern := filepath.Join(tempDir, "cross-platform-cleanup-*.txt")
	helper.WithCleanFilesystem(cleanupPattern)

	// Files should be cleaned up
	for _, filePath := range testFiles {
		assert.NoFileExists(t, filePath, "File should be cleaned up: %s", filePath)
	}
}

// TestCrossPlatformProjectDiscovery tests project root discovery across platforms
func TestCrossPlatformProjectDiscovery(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Find project root
	projectRoot, err := helper.findProjectRoot()
	require.NoError(t, err, "Should find project root on %s", runtime.GOOS)

	// Project root should be absolute path
	assert.True(t, filepath.IsAbs(projectRoot),
		"Project root should be absolute path")

	// Project root should contain go.mod
	goModPath := filepath.Join(projectRoot, "go.mod")
	assert.FileExists(t, goModPath, "Project root should contain go.mod")

	// Project root should use platform-appropriate separators
	expectedSeparator := string(filepath.Separator)
	assert.Contains(t, projectRoot, expectedSeparator,
		"Project root should use platform separator")

	// Test from different working directories
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Logf("Warning: Failed to restore working directory: %v", err)
		}
	}()

	// Create and change to a subdirectory
	testDir := filepath.Join(helper.GetTempDir(), "subdir", "nested")
	err = os.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	err = os.Chdir(testDir)
	require.NoError(t, err)

	// Should still find the same project root
	projectRoot2, err := helper.findProjectRoot()
	require.NoError(t, err, "Should find project root from subdirectory")
	assert.Equal(t, projectRoot, projectRoot2,
		"Should find same project root from different directories")

	// Restore original directory
	err = os.Chdir(originalDir)
	require.NoError(t, err)
}

// TestCrossPlatformCommandExecution tests command execution across platforms
func TestCrossPlatformCommandExecution(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Test basic command execution
	stdout, stderr, err := helper.ExecGrove("--help")
	require.NoError(t, err, "Help command should work on %s", runtime.GOOS)
	assert.NotEmpty(t, stdout, "Help should produce output")
	assert.Empty(t, stderr, "Help should not produce errors")

	// Test command execution in different directory
	testDir := helper.GetTempDir()
	stdout2, stderr2, err := helper.ExecGroveInDir(testDir, "--version")
	require.NoError(t, err, "Version command should work in different directory")
	assert.NotEmpty(t, stdout2, "Version should produce output")
	assert.Empty(t, stderr2, "Version should not produce errors")

	// Test error handling
	_, stderr3, err := helper.ExecGrove("--nonexistent-flag")
	assert.Error(t, err, "Invalid command should fail")
	assert.NotEmpty(t, stderr3, "Invalid command should produce error output")
}

// TestCrossPlatformFileOperations tests file operations work consistently across platforms
func TestCrossPlatformFileOperations(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Test file creation with various content types
	testCases := []struct {
		name    string
		content string
	}{
		{"simple", "simple content"},
		{"multiline", "line 1\nline 2\nline 3"},
		{"unicode", "unicode: 你好世界 🌍"},
		{"mixed_endings", "windows\r\nlinux\nunix\r"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			filePath := helper.CreateTempFile(tc.name+".txt", tc.content)

			// File should exist
			assert.FileExists(t, filePath)

			// Content should be preserved
			actualContent, err := os.ReadFile(filePath)
			require.NoError(t, err)
			assert.Equal(t, tc.content, string(actualContent),
				"Content should be preserved for %s", tc.name)
		})
	}

	// Test directory creation with various path structures
	dirTests := []string{
		"simple",
		"nested/dir",
		"deeply/nested/directory/structure",
	}

	for _, dirPath := range dirTests {
		t.Run("dir_"+strings.ReplaceAll(dirPath, "/", "_"), func(t *testing.T) {
			fullPath := helper.CreateTempDir(dirPath)

			// Directory should exist
			assert.DirExists(t, fullPath)

			// Should be able to create files in directory
			testFile := filepath.Join(fullPath, "test.txt")
			err := os.WriteFile(testFile, []byte("test"), 0o644)
			require.NoError(t, err)
			assert.FileExists(t, testFile)
		})
	}
}

// TestCrossPlatformGlobPatterns tests that glob patterns work consistently across platforms
func TestCrossPlatformGlobPatterns(t *testing.T) {
	// Create test files with different patterns
	tempDir := os.TempDir()
	testFiles := []string{
		filepath.Join(tempDir, "glob-test-1.txt"),
		filepath.Join(tempDir, "glob-test-2.log"),
		filepath.Join(tempDir, "glob-other-1.txt"),
		filepath.Join(tempDir, "not-matching.dat"),
	}

	// Create the files
	for _, filePath := range testFiles {
		err := os.WriteFile(filePath, []byte("glob test"), 0o644)
		require.NoError(t, err)
	}

	// Test different glob patterns
	patterns := []struct {
		name     string
		pattern  string
		expected int // number of files that should match
	}{
		{"wildcard", filepath.Join(tempDir, "glob-test-*.txt"), 1},
		{"extension", filepath.Join(tempDir, "glob-*.log"), 1},
		{"prefix", filepath.Join(tempDir, "glob-*"), 3},
		{"nomatch", filepath.Join(tempDir, "nonexistent-*"), 0},
	}

	for _, pt := range patterns {
		t.Run(pt.name, func(t *testing.T) {
			matches, err := filepath.Glob(pt.pattern)
			require.NoError(t, err, "Glob should work on %s", runtime.GOOS)
			assert.Len(t, matches, pt.expected,
				"Pattern %s should match %d files", pt.pattern, pt.expected)
		})
	}

	// Clean up test files
	for _, filePath := range testFiles {
		if err := os.Remove(filePath); err != nil {
			t.Logf("Warning: failed to remove test file %s: %v", filePath, err)
		}
	}
}

// BenchmarkCrossPlatformBinaryBuilding benchmarks binary building performance across platforms
func BenchmarkCrossPlatformBinaryBuilding(b *testing.B) {
	// Only run this benchmark on CI or when explicitly requested
	if os.Getenv("CI") == "" && os.Getenv("BENCH_CROSS_PLATFORM") == "" {
		b.Skip("Skipping cross-platform benchmark (set BENCH_CROSS_PLATFORM=1 to run)")
	}

	// Create helper once outside the benchmark loop
	helper := NewIntegrationTestHelper(&testing.T{})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Time only the binary access (should be cached after first call)
		binaryPath := helper.GetBinary()

		// Verify binary exists
		if _, err := os.Stat(binaryPath); err != nil {
			b.Fatalf("Binary build failed: %v", err)
		}
	}
}
