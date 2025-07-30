package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// UnitTestHelper provides utilities for robust unit testing
type UnitTestHelper struct {
	tb          testing.TB
	tempDir     string
	originalDir string
	cleanupFns  []func()
}

func NewUnitTestHelper(tb testing.TB) *UnitTestHelper {
	tb.Helper()

	originalDir, err := os.Getwd()
	require.NoError(tb, err, "Failed to get current working directory for test: %s", tb.Name())

	helper := &UnitTestHelper{
		tb:          tb,
		tempDir:     tb.TempDir(),
		originalDir: originalDir,
		cleanupFns:  make([]func(), 0),
	}

	tb.Cleanup(func() {
		helper.cleanup()
	})

	return helper
}

func (h *UnitTestHelper) GetTempDir() string {
	return h.tempDir
}

func (h *UnitTestHelper) CreateTempFile(name, content string) string {
	h.tb.Helper()

	filePath := filepath.Join(h.tempDir, name)

	parentDir := filepath.Dir(filePath)
	err := os.MkdirAll(parentDir, 0o755)
	require.NoError(h.tb, err, "Failed to create parent directory %s for test %s", parentDir, h.tb.Name())

	err = os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(h.tb, err, "Failed to create temp file %s for test %s", filePath, h.tb.Name())

	return filePath
}

func (h *UnitTestHelper) CreateTempDir(path string) string {
	h.tb.Helper()

	fullPath := filepath.Join(h.tempDir, path)
	err := os.MkdirAll(fullPath, 0o755)
	require.NoError(h.tb, err, "Failed to create temp directory %s for test %s", fullPath, h.tb.Name())

	return fullPath
}

// WithCleanFilesystem removes potential leftover files that could interfere with tests.
func (h *UnitTestHelper) WithCleanFilesystem(patterns ...string) *UnitTestHelper {
	h.tb.Helper()

	// Common patterns that might interfere with tests.
	defaultPatterns := []string{
		"/tmp/grove-*",
		"/tmp/test-*",
		"/tmp/path-gen-*",
		"/tmp/create-*",
		"/tmp/grove_write_test_*",
	}

	defaultPatterns = append(defaultPatterns, patterns...)
	allPatterns := defaultPatterns

	for _, pattern := range allPatterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, match := range matches {
			// Skip if it's our own temp directory.
			if strings.Contains(match, h.tempDir) {
				continue
			}
			_ = os.RemoveAll(match) // Best effort cleanup.
		}
	}

	return h
}

// WithIsolatedPath ensures tests don't interfere with each other via path validation.
func (h *UnitTestHelper) WithIsolatedPath() *UnitTestHelper {
	h.tb.Helper()

	// Create a unique test path that won't conflict with other tests.
	testID := time.Now().UnixNano()
	testPath := filepath.Join(os.TempDir(), "grove-unit-test", h.tb.Name(), fmt.Sprintf("%d", testID))

	// Clean up the test path if it exists.
	_ = os.RemoveAll(testPath)

	h.addCleanup(func() {
		_ = os.RemoveAll(testPath)
	})

	return h
}

// GetUniqueTestPath returns a unique path for testing that won't conflict with other tests.
func (h *UnitTestHelper) GetUniqueTestPath(suffix string) string {
	h.tb.Helper()

	// Create a unique path using test name and timestamp.
	testID := time.Now().UnixNano()
	safeName := strings.ReplaceAll(h.tb.Name(), "/", "_")

	return filepath.Join(os.TempDir(), "grove-unit-test", safeName, fmt.Sprintf("%d", testID), suffix)
}

func (h *UnitTestHelper) AssertNoFileExists(path string) {
	h.tb.Helper()

	_, err := os.Stat(path)
	require.True(h.tb, os.IsNotExist(err), "File should not exist: %s (test: %s)", path, h.tb.Name())
}

func (h *UnitTestHelper) AssertFileExists(path string) {
	h.tb.Helper()

	_, err := os.Stat(path)
	require.NoError(h.tb, err, "File should exist: %s (test: %s)", path, h.tb.Name())
}

func (h *UnitTestHelper) addCleanup(fn func()) {
	h.cleanupFns = append(h.cleanupFns, fn)
}

func (h *UnitTestHelper) cleanup() {
	// Execute cleanup functions in reverse order.
	for i := len(h.cleanupFns) - 1; i >= 0; i-- {
		h.cleanupFns[i]()
	}
}
