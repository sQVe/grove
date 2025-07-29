package testutils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// UnitTestHelper provides utilities for robust unit testing
type UnitTestHelper struct {
	t           *testing.T
	tempDir     string
	originalDir string
	cleanupFns  []func()
}

// NewUnitTestHelper creates a new unit test helper
func NewUnitTestHelper(t *testing.T) *UnitTestHelper {
	t.Helper()

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	helper := &UnitTestHelper{
		t:           t,
		tempDir:     t.TempDir(),
		originalDir: originalDir,
		cleanupFns:  make([]func(), 0),
	}

	// Register cleanup
	t.Cleanup(func() {
		helper.cleanup()
	})

	return helper
}

// GetTempDir returns a unique temporary directory for this test
func (h *UnitTestHelper) GetTempDir() string {
	return h.tempDir
}

// CreateTempFile creates a temporary file with given content
func (h *UnitTestHelper) CreateTempFile(name, content string) string {
	h.t.Helper()

	filePath := filepath.Join(h.tempDir, name)

	// Ensure parent directory exists
	parentDir := filepath.Dir(filePath)
	err := os.MkdirAll(parentDir, 0o755)
	require.NoError(h.t, err, "Failed to create parent directory")

	err = os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(h.t, err, "Failed to create temp file")

	return filePath
}

// CreateTempDir creates a temporary directory structure
func (h *UnitTestHelper) CreateTempDir(path string) string {
	h.t.Helper()

	fullPath := filepath.Join(h.tempDir, path)
	err := os.MkdirAll(fullPath, 0o755)
	require.NoError(h.t, err, "Failed to create temp directory")

	return fullPath
}

// WithCleanFilesystem removes potential leftover files that could interfere with tests
func (h *UnitTestHelper) WithCleanFilesystem(patterns ...string) *UnitTestHelper {
	h.t.Helper()

	// Common patterns that might interfere with tests
	defaultPatterns := []string{
		"/tmp/grove-*",
		"/tmp/test-*",
		"/tmp/path-gen-*",
		"/tmp/create-*",
		"/tmp/grove_write_test_*",
	}

	allPatterns := append(defaultPatterns, patterns...)

	for _, pattern := range allPatterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, match := range matches {
			// Skip if it's our own temp directory
			if strings.Contains(match, h.tempDir) {
				continue
			}
			_ = os.RemoveAll(match) // Best effort cleanup
		}
	}

	return h
}

// WithIsolatedPath ensures tests don't interfere with each other via path validation
func (h *UnitTestHelper) WithIsolatedPath() *UnitTestHelper {
	h.t.Helper()

	// Create a unique test path that won't conflict with other tests
	testID := time.Now().UnixNano()
	testPath := filepath.Join("/tmp", "grove-unit-test", h.t.Name(), string(rune(testID)))

	// Clean up the test path if it exists
	_ = os.RemoveAll(testPath)

	// Register cleanup
	h.addCleanup(func() {
		_ = os.RemoveAll(testPath)
	})

	return h
}

// GetUniqueTestPath returns a unique path for testing that won't conflict with other tests
func (h *UnitTestHelper) GetUniqueTestPath(suffix string) string {
	h.t.Helper()

	// Create a unique path using test name and timestamp
	testID := time.Now().UnixNano()
	safeName := strings.ReplaceAll(h.t.Name(), "/", "_")

	return filepath.Join("/tmp", "grove-unit-test", safeName, string(rune(testID)), suffix)
}

// AssertNoFileExists asserts that a file or directory does not exist
func (h *UnitTestHelper) AssertNoFileExists(path string) {
	h.t.Helper()

	_, err := os.Stat(path)
	require.True(h.t, os.IsNotExist(err), "File should not exist: %s", path)
}

// AssertFileExists asserts that a file or directory exists
func (h *UnitTestHelper) AssertFileExists(path string) {
	h.t.Helper()

	_, err := os.Stat(path)
	require.NoError(h.t, err, "File should exist: %s", path)
}

// addCleanup adds a cleanup function
func (h *UnitTestHelper) addCleanup(fn func()) {
	h.cleanupFns = append(h.cleanupFns, fn)
}

// cleanup performs all cleanup operations
func (h *UnitTestHelper) cleanup() {
	// Execute cleanup functions in reverse order
	for i := len(h.cleanupFns) - 1; i >= 0; i-- {
		h.cleanupFns[i]()
	}
}
