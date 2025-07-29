package testutils

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// IntegrationTestHelper provides robust integration testing utilities
type IntegrationTestHelper struct {
	binaryPath  string
	buildOnce   sync.Once
	buildErr    error
	originalDir string
	tempDir     string
	t           *testing.T
}

// NewIntegrationTestHelper creates a new integration test helper
func NewIntegrationTestHelper(t *testing.T) *IntegrationTestHelper {
	t.Helper()

	originalDir, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	helper := &IntegrationTestHelper{
		t:           t,
		originalDir: originalDir,
		tempDir:     t.TempDir(),
	}

	// Register cleanup
	t.Cleanup(func() {
		helper.cleanup()
	})

	return helper
}

// GetBinary builds the grove binary once and returns the path
func (h *IntegrationTestHelper) GetBinary() string {
	h.t.Helper()

	h.buildOnce.Do(func() {
		h.binaryPath, h.buildErr = h.buildBinary()
	})

	require.NoError(h.t, h.buildErr, "Failed to build grove binary")
	return h.binaryPath
}

// ExecGrove executes the grove binary with given arguments in an isolated environment
func (h *IntegrationTestHelper) ExecGrove(args ...string) (string, string, error) {
	h.t.Helper()

	binaryPath := h.GetBinary()

	cmd := exec.Command(binaryPath, args...)

	// Set clean environment
	cmd.Env = h.getCleanEnvironment()

	// Use temporary directory as working directory
	cmd.Dir = h.tempDir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// ExecGroveInDir executes grove in a specific directory
func (h *IntegrationTestHelper) ExecGroveInDir(dir string, args ...string) (string, string, error) {
	h.t.Helper()

	binaryPath := h.GetBinary()

	cmd := exec.Command(binaryPath, args...)
	cmd.Env = h.getCleanEnvironment()
	cmd.Dir = dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// GetTempDir returns a clean temporary directory for the test
func (h *IntegrationTestHelper) GetTempDir() string {
	return h.tempDir
}

// buildBinary builds the grove binary in a location-independent way
func (h *IntegrationTestHelper) buildBinary() (string, error) {
	// Find the project root by looking for go.mod
	projectRoot, err := h.findProjectRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}

	binaryPath := filepath.Join(h.tempDir, "grove")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}

	// Build from the cmd/grove directory
	cmdDir := filepath.Join(projectRoot, "cmd", "grove")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = cmdDir

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("build failed: %w\nStderr: %s", err, stderr.String())
	}

	return binaryPath, nil
}

// findProjectRoot finds the project root by looking for go.mod
func (h *IntegrationTestHelper) findProjectRoot() (string, error) {
	// Start from the current test file location
	_, filename, _, ok := runtime.Caller(1)
	if !ok {
		return "", fmt.Errorf("could not determine caller location")
	}

	dir := filepath.Dir(filename)

	// Walk up the directory tree looking for go.mod
	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found in any parent directory")
}

// getCleanEnvironment returns a clean environment for test execution
func (h *IntegrationTestHelper) getCleanEnvironment() []string {
	// Start with minimal environment
	env := []string{
		"PATH=" + os.Getenv("PATH"),
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"TMPDIR=" + h.tempDir,
	}

	// Add Go-specific variables if present
	if gopath := os.Getenv("GOPATH"); gopath != "" {
		env = append(env, "GOPATH="+gopath)
	}
	if goroot := os.Getenv("GOROOT"); goroot != "" {
		env = append(env, "GOROOT="+goroot)
	}

	return env
}

// cleanup performs cleanup operations
func (h *IntegrationTestHelper) cleanup() {
	// Restore original working directory
	if h.originalDir != "" {
		_ = os.Chdir(h.originalDir)
	}

	// Additional cleanup can be added here
}

// WithCleanFilesystem ensures no leftover files from previous test runs
func (h *IntegrationTestHelper) WithCleanFilesystem(patterns ...string) *IntegrationTestHelper {
	h.t.Helper()

	// Clean up common test artifacts
	defaultPatterns := []string{
		"/tmp/grove-*",
		"/tmp/create-cmd-*",
		"/tmp/grove-list-*",
		"/tmp/grove-test*",
	}

	allPatterns := append(defaultPatterns, patterns...)

	for _, pattern := range allPatterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue // Skip invalid patterns
		}

		for _, match := range matches {
			_ = os.RemoveAll(match) // Best effort cleanup
		}
	}

	return h
}
