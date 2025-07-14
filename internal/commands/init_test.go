package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-init-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	// Test init in temp directory
	testDir := filepath.Join(tempDir, "test-repo")
	cmd := NewInitCmd()
	cmd.SetArgs([]string{testDir})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify bare repository was created
	assert.DirExists(t, testDir)

	// Change to test directory to verify it's a valid bare repo
	err = os.Chdir(testDir)
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Verify it's a git repository
	isRepo, err := git.IsGitRepository()
	require.NoError(t, err)
	assert.True(t, isRepo)

	// Verify it's a bare repository (should have config file in root)
	configPath := filepath.Join(testDir, "config")
	assert.FileExists(t, configPath)

	// Verify HEAD file exists
	headPath := filepath.Join(testDir, "HEAD")
	assert.FileExists(t, headPath)
}

func TestInitCommandCurrentDirectory(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-init-current-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test init in current directory (no args)
	cmd := NewInitCmd()
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify bare repository was created in current directory
	isRepo, err := git.IsGitRepository()
	require.NoError(t, err)
	assert.True(t, isRepo)

	// Verify it's a bare repository
	configPath := filepath.Join(tempDir, "config")
	assert.FileExists(t, configPath)
}

func TestInitCommandExistingRepo(t *testing.T) {
	// This test runs in the grove project directory which is already a git repo
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"."})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already a git repository")
}
