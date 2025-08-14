package testutils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// CreateMockGitCommander creates a new MockGitCommander instance with common setup.
// This factory function provides a centralized way to create mocks with consistent configuration.
func CreateMockGitCommander() *MockGitCommander {
	return NewMockGitCommander()
}

// NormalizePath converts file paths to platform-appropriate format.
// This utility handles cross-platform path differences in tests.
func NormalizePath(path string) string {
	// Convert forward slashes to platform-specific separators.
	normalized := filepath.FromSlash(path)

	// Clean the path to remove redundant separators and resolve . and ..
	normalized = filepath.Clean(normalized)

	return normalized
}

// JoinPath joins path components using platform-appropriate separators.
// This utility ensures consistent path handling across Windows, Linux, and macOS.
func JoinPath(components ...string) string {
	if len(components) == 0 {
		return ""
	}

	// Use filepath.Join for platform-appropriate path construction.
	joined := filepath.Join(components...)

	return joined
}

// IsWindows returns true if running on Windows platform.
// This utility helps with platform-specific test logic.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS platform.
// This utility helps with platform-specific test logic.
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux platform.
// This utility helps with platform-specific test logic.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// AssertGitState validates that a Git repository matches the expected state.
// This function provides comprehensive repository state validation for tests.
func AssertGitState(t *testing.T, repoPath string, expected *GitState) {
	t.Helper()

	// Validate current branch.
	if expected.CurrentBranch != "" {
		currentBranch := getCurrentBranch(t, repoPath)
		assert.Equal(t, expected.CurrentBranch, currentBranch, "current branch mismatch")
	}

	// Validate branch list.
	if len(expected.Branches) > 0 {
		branches := getBranches(t, repoPath)
		assert.ElementsMatch(t, expected.Branches, branches, "branch list mismatch")
	}

	// Validate commit count.
	if expected.Commits > 0 {
		commitCount := getCommitCount(t, repoPath)
		assert.Equal(t, expected.Commits, commitCount, "commit count mismatch")
	}

	// Validate working tree status.
	if expected.WorkingTreeClean {
		isClean := isWorkingTreeClean(t, repoPath)
		assert.True(t, isClean, "working tree should be clean but has uncommitted changes")
	}

	// Validate file existence.
	if len(expected.Files) > 0 {
		for _, file := range expected.Files {
			filePath := filepath.Join(repoPath, file)
			assert.FileExists(t, filePath, "expected file %s should exist", file)
		}
	}
}

// CreateTestFile creates a test file with the specified content.
// This utility provides consistent file creation for test scenarios.
func CreateTestFile(t *testing.T, dir, filename, content string) string {
	t.Helper()

	filePath := filepath.Join(dir, filename)

	// Ensure the directory exists.
	err := os.MkdirAll(filepath.Dir(filePath), 0o755)
	require.NoError(t, err, "failed to create directory for test file")

	err = os.WriteFile(filePath, []byte(content), 0o644)
	require.NoError(t, err, "failed to create test file")

	return filePath
}

// getCurrentBranch returns the current branch name for the repository.
// Note: We use exec.Command here because these are test helpers that need
// to run actual git commands for validation. This is different from
// production code which should use the Commander interface.
func getCurrentBranch(t *testing.T, repoPath string) string {
	t.Helper()

	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	require.NoError(t, err, "failed to get current branch")

	return strings.TrimSpace(string(output))
}

// getBranches returns all branch names for the repository.
func getBranches(t *testing.T, repoPath string) []string {
	t.Helper()

	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	require.NoError(t, err, "failed to get branches")

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(branches) == 1 && branches[0] == "" {
		return []string{}
	}

	return branches
}

// getCommitCount returns the number of commits in the current branch.
func getCommitCount(t *testing.T, repoPath string) int {
	t.Helper()

	cmd := exec.Command("git", "rev-list", "--count", "HEAD")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	require.NoError(t, err, "failed to get commit count")

	countStr := strings.TrimSpace(string(output))
	count := 0
	_, err = fmt.Sscanf(countStr, "%d", &count)
	require.NoError(t, err, "failed to parse commit count")

	return count
}

// isWorkingTreeClean returns true if the working tree has no uncommitted changes.
func isWorkingTreeClean(t *testing.T, repoPath string) bool {
	t.Helper()

	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath

	output, err := cmd.Output()
	require.NoError(t, err, "failed to check working tree status")

	return strings.TrimSpace(string(output)) == ""
}
