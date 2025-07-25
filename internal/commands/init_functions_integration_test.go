//go:build integration
// +build integration

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRunInitLocal tests the internal runInitLocal function directly
func TestRunInitLocal(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	tempDir, err := os.MkdirTemp("", "grove-init-local-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = runInitLocal("")
	require.NoError(t, err)

	// Verify .bare directory was created
	bareDir := filepath.Join(tempDir, ".bare")
	_, err = os.Stat(bareDir)
	require.NoError(t, err)

	// Verify .git file was created
	gitFile := filepath.Join(tempDir, ".git")
	_, err = os.Stat(gitFile)
	require.NoError(t, err)

	// Test with existing .git (should fail)
	err = runInitLocal("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository already exists at")
}

// TestRunInitLocalWithTargetDir tests the runInitLocal function with a target directory
func TestRunInitLocalWithTargetDir(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-local-target-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	targetDir := filepath.Join(tempDir, "new-repo")

	err = runInitLocal(targetDir)
	require.NoError(t, err)

	// Verify target directory was created
	_, err = os.Stat(targetDir)
	require.NoError(t, err)

	// Verify .bare directory was created
	bareDir := filepath.Join(targetDir, ".bare")
	_, err = os.Stat(bareDir)
	require.NoError(t, err)

	// Verify .git file was created
	gitFile := filepath.Join(targetDir, ".git")
	_, err = os.Stat(gitFile)
	require.NoError(t, err)
}

// TestRunInitConvertWithExecutor tests the runInitConvertWithExecutor function with a mock
func TestRunInitConvertWithExecutor(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-convert-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	mockExecutor := testutils.NewMockGitExecutor()

	// Test conversion of non-traditional repo (should fail)
	err = runInitConvertWithExecutor(mockExecutor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository not found at")
}

// TestRunInitConvertAlreadyGrove tests conversion attempt on already converted repository
func TestRunInitConvertAlreadyGrove(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-convert-grove-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a mock .git file and .bare directory to simulate a grove repo
	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: .bare"), 0o644)
	require.NoError(t, err)

	bareDir := filepath.Join(tempDir, ".bare")
	err = os.MkdirAll(bareDir, 0o755)
	require.NoError(t, err)

	mockExecutor := testutils.NewMockGitExecutor()

	// Test conversion of already grove repo (should fail)
	err = runInitConvertWithExecutor(mockExecutor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository already exists at")
}

// TestRunInitRouting tests the command argument validation logic
func TestRunInitRouting(t *testing.T) {
	cmd := NewInitCmd()

	// Test convert flag validation
	cmd.SetArgs([]string{"--convert", "arg"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify arguments when using --convert flag")

	// Test too many args validation - need to create a fresh command instance
	cmd2 := NewInitCmd()
	cmd2.SetArgs([]string{"arg1", "arg2", "arg3"})
	err = cmd2.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many arguments")
}