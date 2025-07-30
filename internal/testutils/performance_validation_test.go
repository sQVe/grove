package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBinaryBuildCachingEffectiveness tests that caching actually improves performance
func TestBinaryBuildCachingEffectiveness(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	// Measure first call (should build binary)
	start1 := time.Now()
	binary1 := helper.GetBinary()
	firstCallDuration := time.Since(start1)

	// Measure second call (should use cache)
	start2 := time.Now()
	binary2 := helper.GetBinary()
	secondCallDuration := time.Since(start2)

	// Verify same binary returned
	assert.Equal(t, binary1, binary2, "Should return same binary path")

	// Second call should be significantly faster (at least 10x faster)
	assert.Less(t, secondCallDuration, firstCallDuration/10,
		"Cached call should be much faster: first=%v, second=%v",
		firstCallDuration, secondCallDuration)

	t.Logf("First call (build): %v", firstCallDuration)
	t.Logf("Second call (cache): %v", secondCallDuration)
	t.Logf("Speedup: %.2fx", float64(firstCallDuration)/float64(secondCallDuration))
}

// TestParallelExecutionSafety tests that parallel test execution is safe
func TestParallelExecutionSafety(t *testing.T) {
	const numGoroutines = 50
	const numOperationsPerGoroutine = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numOperationsPerGoroutine)

	// Launch multiple goroutines performing various operations
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numOperationsPerGoroutine; j++ {
				// Test unique path generation
				helper := NewUnitTestHelper(t)
				path := helper.GetUniqueTestPath(fmt.Sprintf("parallel-%d-%d", goroutineID, j))

				if path == "" {
					errors <- fmt.Errorf("goroutine %d, operation %d: empty path", goroutineID, j)
					return
				}

				// Test file creation
				filePath := helper.CreateTempFile(fmt.Sprintf("parallel-%d-%d.txt", goroutineID, j),
					fmt.Sprintf("content from goroutine %d operation %d", goroutineID, j))

				if _, err := os.Stat(filePath); err != nil {
					errors <- fmt.Errorf("goroutine %d, operation %d: file not created: %v", goroutineID, j, err)
					return
				}

				// Test directory creation
				dirPath := helper.CreateTempDir(fmt.Sprintf("parallel-dir-%d-%d", goroutineID, j))
				if _, err := os.Stat(dirPath); err != nil {
					errors <- fmt.Errorf("goroutine %d, operation %d: directory not created: %v", goroutineID, j, err)
					return
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	var allErrors []error
	for err := range errors {
		allErrors = append(allErrors, err)
	}

	if len(allErrors) > 0 {
		t.Fatalf("Parallel execution had %d errors: %v", len(allErrors), allErrors[0])
	}

	t.Logf("Successfully completed %d parallel operations across %d goroutines",
		numGoroutines*numOperationsPerGoroutine, numGoroutines)
}

// TestParallelBinaryBuilding tests that parallel binary building is safe
func TestParallelBinaryBuilding(t *testing.T) {
	const numGoroutines = 20

	var wg sync.WaitGroup
	binaryPaths := make([]string, numGoroutines)
	errors := make([]error, numGoroutines)

	// All goroutines share the same helper to test concurrent access
	helper := NewIntegrationTestHelper(t)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			defer func() {
				if r := recover(); r != nil {
					errors[index] = fmt.Errorf("panic: %v", r)
				}
			}()

			// All should get the same binary (cached)
			binaryPaths[index] = helper.GetBinary()
		}(i)
	}

	wg.Wait()

	// Check for errors
	for i, err := range errors {
		require.NoError(t, err, "Goroutine %d should not error", i)
	}

	// All should have same binary path
	for i := 1; i < numGoroutines; i++ {
		assert.Equal(t, binaryPaths[0], binaryPaths[i],
			"All goroutines should get same binary path")
	}

	// Binary should exist and be valid
	assert.FileExists(t, binaryPaths[0])

	// Test that binary actually works
	stdout, stderr, err := helper.ExecGrove("--version")
	require.NoError(t, err)
	assert.NotEmpty(t, stdout)
	assert.Empty(t, stderr)
}

// TestMemoryUsage tests that the testing infrastructure doesn't leak memory
func TestMemoryUsage(t *testing.T) {
	var m1, m2 runtime.MemStats

	// Force garbage collection and get baseline
	runtime.GC()
	runtime.ReadMemStats(&m1)

	// Perform many operations
	const numOperations = 100
	for i := 0; i < numOperations; i++ {
		helper := NewUnitTestHelper(t)

		// Create files and directories
		_ = helper.CreateTempFile(fmt.Sprintf("memory-test-%d.txt", i), "test content")
		_ = helper.CreateTempDir(fmt.Sprintf("memory-test-dir-%d", i))
		_ = helper.GetUniqueTestPath(fmt.Sprintf("memory-test-path-%d", i))

		// Force cleanup
		helper.cleanup()
	}

	// Force garbage collection and get final memory
	runtime.GC()
	runtime.ReadMemStats(&m2)

	// Calculate memory increase
	allocIncrease := m2.Alloc - m1.Alloc
	totalAllocIncrease := m2.TotalAlloc - m1.TotalAlloc

	t.Logf("Memory stats after %d operations:", numOperations)
	t.Logf("  Current allocation increase: %d bytes", allocIncrease)
	t.Logf("  Total allocation increase: %d bytes", totalAllocIncrease)
	t.Logf("  GC cycles: %d", m2.NumGC-m1.NumGC)

	// Current allocation should not grow significantly (< 1MB per operation on average)
	avgAllocPerOp := allocIncrease / numOperations
	assert.Less(t, avgAllocPerOp, uint64(1024*1024),
		"Average allocation per operation should be reasonable")
}

// TestCleanupEfficiency tests that cleanup operations are efficient
func TestCleanupEfficiency(t *testing.T) {
	// Create a large number of test files
	const numFiles = 1000
	testFiles := make([]string, numFiles)

	// Create files
	for i := 0; i < numFiles; i++ {
		filename := filepath.Join(os.TempDir(), fmt.Sprintf("cleanup-efficiency-test-%d", i))
		err := os.WriteFile(filename, []byte("test"), 0o644)
		require.NoError(t, err)
		testFiles[i] = filename
	}

	helper := NewUnitTestHelper(t)

	// Measure cleanup time
	start := time.Now()
	helper.WithCleanFilesystem("/tmp/cleanup-efficiency-test-*")
	cleanupDuration := time.Since(start)

	// Cleanup should be reasonably fast (less than 1 second for 1000 files)
	assert.Less(t, cleanupDuration, time.Second,
		"Cleanup of %d files should be fast: %v", numFiles, cleanupDuration)

	// Verify all files were cleaned up
	cleanedCount := 0
	for _, file := range testFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			cleanedCount++
		}
	}

	assert.Equal(t, numFiles, cleanedCount, "All files should be cleaned up")

	t.Logf("Cleaned up %d files in %v (%.2f files/ms)",
		numFiles, cleanupDuration, float64(numFiles)/float64(cleanupDuration.Milliseconds()))
}

// TestResourceCleanupUnderLoad tests that resource cleanup works properly under load
func TestResourceCleanupUnderLoad(t *testing.T) {
	const numHelpers = 50
	const filesPerHelper = 20

	var wg sync.WaitGroup

	// Create many helpers simultaneously
	for i := 0; i < numHelpers; i++ {
		wg.Add(1)
		go func(helperID int) {
			defer wg.Done()

			helper := NewUnitTestHelper(t)

			// Create many files with each helper
			for j := 0; j < filesPerHelper; j++ {
				_ = helper.CreateTempFile(
					fmt.Sprintf("load-test-%d-%d.txt", helperID, j),
					fmt.Sprintf("content from helper %d file %d", helperID, j))
			}

			// Apply filesystem cleanup
			helper.WithCleanFilesystem("/tmp/load-test-*")
		}(i)
	}

	wg.Wait()

	// Check that cleanup was effective
	matches, err := filepath.Glob("/tmp/load-test-*")
	require.NoError(t, err)

	// Most files should be cleaned up (allow for some timing issues)
	assert.Less(t, len(matches), numHelpers*filesPerHelper/10,
		"Most files should be cleaned up, found %d remaining files", len(matches))

	// Clean up any remaining files
	for _, file := range matches {
		if err := os.Remove(file); err != nil {
			t.Logf("Warning: failed to remove file %s: %v", file, err)
		}
	}

	t.Logf("Successfully handled %d helpers creating %d files each",
		numHelpers, filesPerHelper)
}

// TestProjectDiscoveryPerformance tests that project discovery is reasonably fast
func TestProjectDiscoveryPerformance(t *testing.T) {
	helper := NewIntegrationTestHelper(t)

	const numDiscoveries = 100
	totalDuration := time.Duration(0)

	for i := 0; i < numDiscoveries; i++ {
		start := time.Now()
		projectRoot, err := helper.findProjectRoot()
		duration := time.Since(start)
		totalDuration += duration

		require.NoError(t, err, "Project discovery should succeed")
		assert.NotEmpty(t, projectRoot, "Project root should not be empty")
	}

	avgDuration := totalDuration / numDiscoveries

	// Project discovery should be fast (< 10ms on average)
	assert.Less(t, avgDuration, 10*time.Millisecond,
		"Project discovery should be fast: avg=%v", avgDuration)

	t.Logf("Project discovery: %d calls, total=%v, avg=%v",
		numDiscoveries, totalDuration, avgDuration)
}
