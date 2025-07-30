//go:build integration
// +build integration

package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationTestHelper_RealWorldScenario tests the integration helper in a realistic scenario
func TestIntegrationTestHelper_RealWorldScenario(t *testing.T) {
	helper := NewIntegrationTestHelper(t).WithCleanFilesystem()

	// Test multiple grove executions in sequence
	testCases := []struct {
		name string
		args []string
	}{
		{"version", []string{"--version"}},
		{"help", []string{"--help"}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stdout, stderr, err := helper.ExecGrove(tc.args...)

			// Commands should succeed
			require.NoError(t, err, "Command %v should succeed", tc.args)
			assert.NotEmpty(t, stdout, "Should have output for %v", tc.args)

			// Stderr should be empty for these commands
			assert.Empty(t, stderr, "Stderr should be empty for %v", tc.args)
		})
	}
}

// TestIntegrationTestHelper_ConcurrentBinaryBuilding tests that binary building is thread-safe
func TestIntegrationTestHelper_ConcurrentBinaryBuilding(t *testing.T) {
	const numGoroutines = 10

	helper := NewIntegrationTestHelper(t)

	var wg sync.WaitGroup
	binaryPaths := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	// Launch multiple goroutines that try to get the binary
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			// Add small delay to increase chance of concurrent access
			time.Sleep(time.Duration(index) * time.Millisecond)

			defer func() {
				if r := recover(); r != nil {
					errors[index] = r.(error)
				}
			}()

			binaryPaths[index] = helper.GetBinary()
		}(i)
	}

	wg.Wait()

	// Check that no errors occurred
	for i, err := range errors {
		assert.NoError(t, err, "Goroutine %d should not error", i)
	}

	// All paths should be the same (cached result)
	for i := 1; i < numGoroutines; i++ {
		assert.Equal(t, binaryPaths[0], binaryPaths[i],
			"All goroutines should get same binary path (cached)")
	}

	// Binary should exist and be valid
	assert.FileExists(t, binaryPaths[0])
}

// TestIntegrationTestHelper_ProjectDiscoveryFromDifferentLocations tests project root discovery
func TestIntegrationTestHelper_ProjectDiscoveryFromDifferentLocations(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Test from current location
	projectRoot1, err := helper.findProjectRoot()
	require.NoError(t, err, "Should find project root from current location")

	// Change to a subdirectory and test again
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Create and change to a deeper directory
	testDir := filepath.Join(helper.GetTempDir(), "deep", "nested", "dir")
	err = os.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Should still find the same project root
	projectRoot2, err := helper.findProjectRoot()
	require.NoError(t, err, "Should find project root from nested directory")

	assert.Equal(t, projectRoot1, projectRoot2,
		"Project root should be same regardless of current directory")

	// Verify it's actually the correct project root
	goModPath := filepath.Join(projectRoot2, "go.mod")
	assert.FileExists(t, goModPath, "Project root should contain go.mod")
}

// TestIntegrationTestHelper_ErrorHandling tests error scenarios
func TestIntegrationTestHelper_ErrorHandling(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Test with invalid command that should fail
	stdout, stderr, err := helper.ExecGrove("--invalid-flag-that-does-not-exist")

	// Command should fail
	assert.Error(t, err, "Invalid command should fail")

	// Should capture error output
	assert.NotEmpty(t, stderr, "Should capture error output")

	// Stdout might be empty for error cases
	t.Logf("stdout: %s", stdout)
	t.Logf("stderr: %s", stderr)
}

// TestUnitTestHelper_ParallelPathGeneration tests that path generation is safe for parallel tests
func TestUnitTestHelper_ParallelPathGeneration(t *testing.T) {
	const numGoroutines = 20

	var wg sync.WaitGroup
	paths := make([]string, numGoroutines)

	// Generate paths concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			helper := NewUnitTestHelper(t)
			paths[index] = helper.GetUniqueTestPath("parallel-test")
		}(i)
	}

	wg.Wait()

	// All paths should be unique
	pathSet := make(map[string]bool)
	for i, path := range paths {
		assert.NotEmpty(t, path, "Path %d should not be empty", i)
		assert.False(t, pathSet[path], "Path should be unique: %s", path)
		pathSet[path] = true
	}

	assert.Len(t, pathSet, numGoroutines, "Should have generated unique paths")
}

// TestTestRunner_CompleteIsolation tests full environment and filesystem isolation
func TestTestRunner_CompleteIsolation(t *testing.T) {
	// Set up some initial state
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	testVar := "TEST_RUNNER_ISOLATION_VAR"
	originalValue := os.Getenv(testVar)
	os.Setenv(testVar, "initial_value")
	defer func() {
		if originalValue == "" {
			os.Unsetenv(testVar)
		} else {
			os.Setenv(testVar, originalValue)
		}
	}()

	// Create a subtest that will have its own cleanup cycle
	t.Run("isolation_test", func(t *testing.T) {
		runner := NewTestRunner(t)

		var isolatedDir string
		var isolatedEnvVar string

		runner.
			WithIsolatedWorkingDir().
			WithCleanEnvironment().
			WithCleanFilesystem().
			Run(func() {
				// Capture state inside isolation
				isolatedDir, _ = os.Getwd()
				isolatedEnvVar = os.Getenv(testVar)

				// Create a test file in isolated environment
				testFile := "isolated-test-file.txt"
				err := os.WriteFile(testFile, []byte("isolated content"), 0o644)
				require.NoError(t, err)
				assert.FileExists(t, testFile)
			})

		// Verify isolation worked
		assert.NotEqual(t, originalDir, isolatedDir, "Should be in isolated directory")
		assert.Empty(t, isolatedEnvVar, "Environment variable should be cleared in isolation")
	})

	// After subtest completes (with cleanup), verify restoration
	currentDir, err := os.Getwd()
	require.NoError(t, err)
	assert.Equal(t, originalDir, currentDir, "Should restore original directory after subtest")

	currentEnvVar := os.Getenv(testVar)
	assert.Equal(t, "initial_value", currentEnvVar, "Should restore original environment after subtest")
}

// TestFilesystemCleanup_Performance tests filesystem cleanup performance
func TestFilesystemCleanup_Performance(t *testing.T) {
	helper := NewUnitTestHelper(t)

	// Create many test files to clean up
	numFiles := 100
	testFiles := make([]string, numFiles)

	for i := 0; i < numFiles; i++ {
		filename := filepath.Join("/tmp", fmt.Sprintf("cleanup-perf-test-%d", i))
		err := os.WriteFile(filename, []byte("test"), 0o644)
		require.NoError(t, err)
		testFiles[i] = filename
	}

	// Measure cleanup time
	start := time.Now()
	helper.WithCleanFilesystem("/tmp/cleanup-perf-test-*")
	cleanupTime := time.Since(start)

	// Cleanup should be reasonably fast (less than 1 second for 100 files)
	assert.Less(t, cleanupTime, time.Second, "Cleanup should be fast")

	// Verify files were cleaned up
	for _, file := range testFiles {
		assert.NoFileExists(t, file, "File should be cleaned up: %s", file)
	}
}

// TestCrossPlatformBinaryNames tests that binary names are correct on different platforms
func TestCrossPlatformBinaryNames(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	binaryPath := helper.GetBinary()

	// Check platform-specific binary naming
	switch runtime.GOOS {
	case "windows":
		assert.True(t, strings.HasSuffix(binaryPath, ".exe"),
			"Windows binary should have .exe extension")
	default:
		assert.False(t, strings.HasSuffix(binaryPath, ".exe"),
			"Non-Windows binary should not have .exe extension")
	}

	// Binary should be named 'grove' (plus extension)
	baseName := filepath.Base(binaryPath)
	expectedName := "grove"
	if runtime.GOOS == "windows" {
		expectedName += ".exe"
	}

	assert.Equal(t, expectedName, baseName,
		"Binary should have correct name")
}

// TestEnvironmentIsolation_GoVariables tests that Go-specific variables are preserved
func TestEnvironmentIsolation_GoVariables(t *testing.T) {
	// Set up some Go environment variables
	originalGopath := os.Getenv("GOPATH")
	originalGoroot := os.Getenv("GOROOT")

	// Set test values if not already set
	if originalGopath == "" {
		os.Setenv("GOPATH", "/test/gopath")
		defer os.Unsetenv("GOPATH")
	}
	if originalGoroot == "" {
		os.Setenv("GOROOT", "/test/goroot")
		defer os.Unsetenv("GOROOT")
	}

	runner := NewTestRunner(t)

	var isolatedGopath, isolatedGoroot string

	runner.WithCleanEnvironment().Run(func() {
		isolatedGopath = os.Getenv("GOPATH")
		isolatedGoroot = os.Getenv("GOROOT")
	})

	// Go variables should be preserved in clean environment
	if originalGopath != "" {
		assert.Equal(t, originalGopath, isolatedGopath, "GOPATH should be preserved")
	}
	if originalGoroot != "" {
		assert.Equal(t, originalGoroot, isolatedGoroot, "GOROOT should be preserved")
	}
}
