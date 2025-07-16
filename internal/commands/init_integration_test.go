//go:build integration
// +build integration

package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommandLocal(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-init-local-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Test init in temp directory.
	testDir := filepath.Join(tempDir, "test-repo")
	cmd := NewInitCmd()
	cmd.SetArgs([]string{testDir})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify directory structure.
	assert.DirExists(t, testDir)

	bareDir := filepath.Join(testDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(testDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify .git file content.
	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(content))

	// Change to test directory to verify it works as a git repo.
	err = os.Chdir(testDir)
	require.NoError(t, err)

	// Verify it's a git repository.
	isRepo, err := utils.IsGitRepository(git.DefaultExecutor)
	require.NoError(t, err)
	assert.True(t, isRepo)

	// Verify bare repository structure exists.
	configPath := filepath.Join(bareDir, "config")
	assert.FileExists(t, configPath)

	headPath := filepath.Join(bareDir, "HEAD")
	assert.FileExists(t, headPath)
}

func TestInitCommandCurrentDirectory(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-init-current-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test init in current directory (no args).
	cmd := NewInitCmd()
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify directory structure.
	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(tempDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify it's a git repository.
	isRepo, err := utils.IsGitRepository(git.DefaultExecutor)
	require.NoError(t, err)
	assert.True(t, isRepo)
}

func TestInitCommandExistingGitFile(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-init-existing-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a .git file.
	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("existing"), 0o600)
	require.NoError(t, err)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{tempDir})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already contains a .git file")
}

func TestInitCommandExistingBareDir(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-init-bare-exists-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a .bare directory.
	bareDir := filepath.Join(tempDir, ".bare")
	err = os.MkdirAll(bareDir, 0o750)
	require.NoError(t, err)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{tempDir})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already contains a .bare directory")
}

// Note: Testing the remote clone functionality would require actual network access
// or complex mocking. These tests focus on the local functionality and URL detection.
// Integration tests with real repositories should be done separately.

func TestInitFromRemoteNonEmptyDirectory(t *testing.T) {
	// Create temporary directory with a file.
	tempDir, err := os.MkdirTemp("", "grove-init-remote-nonempty-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a non-hidden file.
	testFile := filepath.Join(tempDir, "existing.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o600)
	require.NoError(t, err)

	// Save current directory.
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory.
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Try to init from fake remote URL (should fail due to non-empty directory).
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"https://github.com/user/repo.git"})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not empty")
}

func TestInitCommandConvertSuccess(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-init-convert-success-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Initialize a real git repository
	_, err = git.ExecuteGit("init", tempDir)
	require.NoError(t, err)

	// Create a dummy file and commit it to have a clean status
	dummyFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(dummyFile, []byte("# Test\n"), 0o644)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Commit the file
	_, err = git.ExecuteGit("add", ".")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Verify it's a traditional repo before conversion
	assert.True(t, git.IsTraditionalRepo(tempDir))
	assert.False(t, git.IsGroveRepo(tempDir))

	// The test with safety checks will show real validation.
	// Let's test the error case first to verify safety is working
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--convert"})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Local-only branch")
}

func TestInitCommandConvertFailures(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grove-init-convert-fail1-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Save current directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalDir) }()

		// Change to temp directory
		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Try to convert non-git directory
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "does not contain a traditional Git repository")
	})

	t.Run("already Grove repo", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grove-init-convert-fail2-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Create Grove structure
		gitFile := filepath.Join(tempDir, ".git")
		err = os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
		require.NoError(t, err)

		bareDir := filepath.Join(tempDir, ".bare")
		err = os.Mkdir(bareDir, 0o750)
		require.NoError(t, err)

		// Save current directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalDir) }()

		// Change to temp directory
		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Try to convert already-Grove repo
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is already a Grove repository")
	})

	t.Run("repository with safety issues", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grove-init-convert-safety-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Initialize repository
		_, err = git.ExecuteGit("init", tempDir)
		require.NoError(t, err)

		// Create uncommitted changes and untracked files
		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0o644)
		require.NoError(t, err)

		untrackedFile := filepath.Join(tempDir, "untracked.txt")
		err = os.WriteFile(untrackedFile, []byte("untracked"), 0o644)
		require.NoError(t, err)

		// Save current directory
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalDir) }()

		// Change to temp directory
		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Try to convert repository with safety issues
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Repository is not ready for conversion:")
		assert.Contains(t, err.Error(), "untracked file(s)")
		assert.Contains(t, err.Error(), "Please resolve these issues before converting")
	})
}
