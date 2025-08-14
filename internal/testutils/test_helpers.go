package testutils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var (
	// binaryCache holds the cached binary path and its hash.
	binaryCache struct {
		sync.RWMutex
		path      string
		hash      string
		buildTime time.Time
	}

	// buildMutex ensures only one binary build happens at a time.
	buildMutex sync.Mutex
)

// filesystemHelper provides common filesystem operations for test helpers.
type filesystemHelper struct {
	t               *testing.T
	tempDir         string
	cleanFilesystem bool
}

// CreateTempFile creates a temporary file with the given content.
func (h *filesystemHelper) CreateTempFile(name, content string) string {
	h.t.Helper()

	dir := h.tempDir
	if dir == "" {
		dir = h.t.TempDir()
	}

	filePath := filepath.Join(dir, name)

	err := os.MkdirAll(filepath.Dir(filePath), 0o755)
	require.NoError(h.t, err, "failed to create parent directory")

	err = os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(h.t, err, "failed to create temp file")

	return filePath
}

// CreateTempDir creates a temporary directory with the given path pattern.
func (h *filesystemHelper) CreateTempDir(pattern string) string {
	h.t.Helper()

	dir := h.tempDir
	if dir == "" {
		dir = h.t.TempDir()
	}

	fullPath := filepath.Join(dir, pattern)

	err := os.MkdirAll(fullPath, 0o755)
	require.NoError(h.t, err, "failed to create temp directory")

	return fullPath
}

// GetTempPath returns a path within the test's temporary directory without creating it.
func (h *filesystemHelper) GetTempPath(pattern string) string {
	h.t.Helper()

	dir := h.tempDir
	if dir == "" && h.cleanFilesystem {
		dir = h.t.TempDir()
		h.tempDir = dir
	} else if dir == "" {
		dir = h.t.TempDir()
	}

	return filepath.Join(dir, pattern)
}

// IntegrationTestHelper provides utilities for integration tests that execute the grove binary.
// It includes binary caching for massive performance improvements.
type IntegrationTestHelper struct {
	filesystemHelper
	runner *TestRunner
}

// NewIntegrationTestHelper creates a new integration test helper.
func NewIntegrationTestHelper(t *testing.T) *IntegrationTestHelper {
	return &IntegrationTestHelper{
		filesystemHelper: filesystemHelper{t: t},
		runner:           NewTestRunner(t),
	}
}

// WithCleanFilesystem configures the helper to use a clean filesystem.
func (h *IntegrationTestHelper) WithCleanFilesystem() *IntegrationTestHelper {
	h.cleanFilesystem = true
	h.tempDir = h.t.TempDir()
	return h
}

// WithCleanEnvironment configures the helper to use a clean environment.
func (h *IntegrationTestHelper) WithCleanEnvironment() *IntegrationTestHelper {
	h.runner = h.runner.WithCleanEnvironment()
	return h
}

// WithIsolatedWorkingDir configures the helper to use an isolated working directory.
func (h *IntegrationTestHelper) WithIsolatedWorkingDir() *IntegrationTestHelper {
	h.runner = h.runner.WithIsolatedWorkingDir()
	return h
}

// ExecGrove executes the grove binary with the given arguments.
// This method uses binary caching for massive performance improvements.
func (h *IntegrationTestHelper) ExecGrove(args ...string) (stdout, stderr string, err error) {
	h.t.Helper()

	// Get or build the cached binary.
	binaryPath, err := h.getCachedBinary()
	if err != nil {
		return "", "", fmt.Errorf("failed to get grove binary: %w", err)
	}

	// Execute the binary with arguments.
	cmd := exec.Command(binaryPath, args...)

	// Set working directory if using clean filesystem.
	if h.cleanFilesystem && h.tempDir != "" {
		cmd.Dir = h.tempDir
	}

	// Capture output.
	var stdoutBuf, stderrBuf []byte
	stdoutBuf, err = cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderrBuf = exitErr.Stderr
		}
		// Wrap error with context about the command that failed.
		return string(stdoutBuf), string(stderrBuf), fmt.Errorf("grove command failed (args: %v): %w", args, err)
	}

	return string(stdoutBuf), string(stderrBuf), nil
}

// getCachedBinary returns the path to the cached grove binary, building it if necessary.
func (h *IntegrationTestHelper) getCachedBinary() (string, error) {
	h.t.Helper()

	// Calculate source hash.
	sourceHash, err := calculateSourceHash()
	if err != nil {
		return "", fmt.Errorf("failed to calculate source hash: %w", err)
	}

	// Check if we have a valid cached binary.
	binaryCache.RLock()
	if binaryCache.path != "" && binaryCache.hash == sourceHash {
		// Verify the binary still exists.
		if _, err := os.Stat(binaryCache.path); err == nil {
			path := binaryCache.path
			binaryCache.RUnlock()
			return path, nil
		}
	}
	binaryCache.RUnlock()

	// Need to build the binary.
	buildMutex.Lock()
	defer buildMutex.Unlock()

	// Double-check after acquiring lock.
	binaryCache.RLock()
	if binaryCache.path != "" && binaryCache.hash == sourceHash {
		if _, err := os.Stat(binaryCache.path); err == nil {
			path := binaryCache.path
			binaryCache.RUnlock()
			return path, nil
		}
	}
	binaryCache.RUnlock()

	// Build the binary.
	binaryPath, err := buildGroveBinary(h.t)
	if err != nil {
		return "", fmt.Errorf("failed to build grove binary: %w", err)
	}

	// Update cache.
	binaryCache.Lock()
	binaryCache.path = binaryPath
	binaryCache.hash = sourceHash
	binaryCache.buildTime = time.Now()
	binaryCache.Unlock()

	return binaryPath, nil
}

// UnitTestHelper provides utilities for unit tests with filesystem operations.
type UnitTestHelper struct {
	filesystemHelper
	runner *TestRunner
}

// NewUnitTestHelper creates a new unit test helper.
func NewUnitTestHelper(t *testing.T) *UnitTestHelper {
	return &UnitTestHelper{
		filesystemHelper: filesystemHelper{t: t},
		runner:           NewTestRunner(t),
	}
}

// WithCleanFilesystem configures the helper to use a clean filesystem.
func (h *UnitTestHelper) WithCleanFilesystem() *UnitTestHelper {
	h.cleanFilesystem = true
	h.tempDir = h.t.TempDir()
	return h
}

// WithCleanEnvironment configures the helper to use a clean environment.
func (h *UnitTestHelper) WithCleanEnvironment() *UnitTestHelper {
	h.runner = h.runner.WithCleanEnvironment()
	return h
}

// WithIsolatedWorkingDir configures the helper to use an isolated working directory.
func (h *UnitTestHelper) WithIsolatedWorkingDir() *UnitTestHelper {
	h.runner = h.runner.WithIsolatedWorkingDir()
	return h
}

// Run executes the test function with the configured isolation settings.
func (h *UnitTestHelper) Run(testFunc func()) {
	h.t.Helper()
	h.runner.Run(testFunc)
}

// calculateSourceHash calculates a hash of the source code to detect changes.
func calculateSourceHash() (string, error) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}

	hasher := sha256.New()

	sourceDirs := []string{"cmd", "internal", "pkg"}
	for _, dir := range sourceDirs {
		dirPath := filepath.Join(projectRoot, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return fmt.Errorf("error walking path %s: %w", path, err)
			}

			if !info.IsDir() && filepath.Ext(path) == ".go" {
				file, err := os.Open(path)
				if err != nil {
					return fmt.Errorf("failed to open file %s: %w", path, err)
				}
				defer func() {
					_ = file.Close()
				}()

				if _, err := io.Copy(hasher, file); err != nil {
					return fmt.Errorf("failed to hash file %s: %w", path, err)
				}
			}

			return nil
		})
		if err != nil {
			return "", fmt.Errorf("failed to walk directory %s: %w", dir, err)
		}
	}

	for _, fileName := range []string{"go.mod", "go.sum"} {
		filePath := filepath.Join(projectRoot, fileName)
		if _, err := os.Stat(filePath); err == nil {
			if err := func() error {
				file, err := os.Open(filePath)
				if err != nil {
					return fmt.Errorf("failed to open %s: %w", fileName, err)
				}
				defer func() {
					_ = file.Close()
				}()

				if _, err := io.Copy(hasher, file); err != nil {
					return fmt.Errorf("failed to hash %s: %w", fileName, err)
				}
				return nil
			}(); err != nil {
				return "", fmt.Errorf("failed to process %s: %w", filePath, err)
			}
		}
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// findProjectRoot finds the project root directory by looking for go.mod.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("could not find project root (no go.mod found)")
}

// buildGroveBinary builds the grove binary and returns its path.
func buildGroveBinary(t *testing.T) (string, error) {
	t.Helper()

	projectRoot, err := findProjectRoot()
	if err != nil {
		return "", err
	}

	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "grove")
	if IsWindows() {
		binaryPath += ".exe"
	}

	cmd := exec.Command("go", "build", "-o", binaryPath, "./cmd/grove")
	cmd.Dir = projectRoot

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to build binary: %w\nOutput: %s", err, output)
	}

	return binaryPath, nil
}
