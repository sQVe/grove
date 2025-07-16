package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecuteGit(t *testing.T) {
	// Test successful command
	output, err := ExecuteGit("--version")
	require.NoError(t, err)
	assert.Contains(t, output, "git version", "should return git version")

	// Test command that should fail
	_, err = ExecuteGit("invalid-command")
	require.Error(t, err)

	var gitErr *GitError
	require.ErrorAs(t, err, &gitErr, "error should be a GitError")
	assert.Equal(t, "git", gitErr.Command)
	assert.Contains(t, gitErr.Args, "invalid-command")
	assert.NotEqual(t, 0, gitErr.ExitCode)
}

func TestInitBare(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-git-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	bareDir := filepath.Join(tempDir, "test-bare.git")

	// Initialize bare repository
	err = InitBare(bareDir)
	require.NoError(t, err)

	// Verify bare repository was created
	assert.DirExists(t, bareDir)

	// Check for key bare repository files
	configPath := filepath.Join(bareDir, "config")
	assert.FileExists(t, configPath)

	headPath := filepath.Join(bareDir, "HEAD")
	assert.FileExists(t, headPath)

	objectsDir := filepath.Join(bareDir, "objects")
	assert.DirExists(t, objectsDir)

	refsDir := filepath.Join(bareDir, "refs")
	assert.DirExists(t, refsDir)
}

func TestCreateGitFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-gitfile-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	mainDir := filepath.Join(tempDir, "main")
	bareDir := filepath.Join(tempDir, "main", ".bare")

	err = os.MkdirAll(mainDir, 0750)
	require.NoError(t, err)

	err = os.MkdirAll(bareDir, 0750)
	require.NoError(t, err)

	// Create .git file
	err = CreateGitFile(mainDir, bareDir)
	require.NoError(t, err)

	// Verify .git file was created
	gitFile := filepath.Join(mainDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify content
	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(content))
}

func TestCreateGitFileAbsolutePath(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-gitfile-abs-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	mainDir := filepath.Join(tempDir, "main")
	bareDir := filepath.Join(tempDir, "separate", "bare")

	err = os.MkdirAll(mainDir, 0750)
	require.NoError(t, err)

	err = os.MkdirAll(bareDir, 0750)
	require.NoError(t, err)

	// Create .git file with absolute path
	err = CreateGitFile(mainDir, bareDir)
	require.NoError(t, err)

	// Verify .git file was created
	gitFile := filepath.Join(mainDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify content contains the path to bare dir
	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "gitdir:")
}
