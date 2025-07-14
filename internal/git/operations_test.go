package git

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGitAvailable(t *testing.T) {
	// Git should be available in the test environment
	assert.True(t, IsGitAvailable(), "git should be available in PATH")
}

func TestIsGitRepository(t *testing.T) {
	// Since we're in the grove project which is a git repo
	isRepo, err := IsGitRepository()
	require.NoError(t, err)
	assert.True(t, isRepo, "should detect we're in a git repository")
}

func TestGetRepositoryRoot(t *testing.T) {
	root, err := GetRepositoryRoot()
	require.NoError(t, err)
	assert.Contains(t, root, "grove", "repository root should contain 'grove'")

	// Verify the root directory exists
	_, err = os.Stat(root)
	assert.NoError(t, err, "repository root should be a valid directory")
}

func TestValidateRepository(t *testing.T) {
	err := ValidateRepository()
	assert.NoError(t, err, "grove project should be a valid git repository with commits")
}

func TestExecuteGit(t *testing.T) {
	// Test successful command
	output, err := ExecuteGit("--version")
	require.NoError(t, err)
	assert.Contains(t, output, "git version", "should return git version")

	// Test command that should fail
	_, err = ExecuteGit("invalid-command")
	require.Error(t, err)

	gitErr, ok := err.(*GitError)
	require.True(t, ok, "error should be a GitError")
	assert.Equal(t, "git", gitErr.Command)
	assert.Contains(t, gitErr.Args, "invalid-command")
	assert.NotEqual(t, 0, gitErr.ExitCode)
}
