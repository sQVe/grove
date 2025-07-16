//go:build integration
// +build integration

package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/testutils"
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
	testDir := testutils.NewTestDirectory(t, "grove-git-test-*")
	defer testDir.Cleanup()

	bareDir := filepath.Join(testDir.Path, "test-bare.git")

	// Initialize bare repository
	err := InitBare(bareDir)
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

	err = os.MkdirAll(mainDir, 0o750)
	require.NoError(t, err)

	err = os.MkdirAll(bareDir, 0o750)
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

	err = os.MkdirAll(mainDir, 0o750)
	require.NoError(t, err)

	err = os.MkdirAll(bareDir, 0o750)
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

func TestIsTraditionalRepo(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-traditional-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test empty directory - should be false
	assert.False(t, IsTraditionalRepo(tempDir))

	// Create .git directory
	gitDir := filepath.Join(tempDir, ".git")
	err = os.Mkdir(gitDir, 0o750)
	require.NoError(t, err)

	// Should now be true
	assert.True(t, IsTraditionalRepo(tempDir))

	// Test with .git file (Grove repo) - should be false
	err = os.Remove(gitDir)
	require.NoError(t, err)

	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
	require.NoError(t, err)

	assert.False(t, IsTraditionalRepo(tempDir))
}

func TestIsGroveRepo(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-grove-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test empty directory - should be false
	assert.False(t, IsGroveRepo(tempDir))

	// Create only .git file - should be false (no .bare)
	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
	require.NoError(t, err)

	assert.False(t, IsGroveRepo(tempDir))

	// Create .bare directory
	bareDir := filepath.Join(tempDir, ".bare")
	err = os.Mkdir(bareDir, 0o750)
	require.NoError(t, err)

	// Should now be true
	assert.True(t, IsGroveRepo(tempDir))

	// Test with .git directory instead of file - should be false
	err = os.Remove(gitFile)
	require.NoError(t, err)

	err = os.Mkdir(gitFile, 0o750)
	require.NoError(t, err)

	assert.False(t, IsGroveRepo(tempDir))
}

func TestValidateGroveStructure(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-validate-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Test invalid structure - no .git file
	err = ValidateGroveStructure(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".git file does not exist")

	// Create .git directory instead of file
	gitDir := filepath.Join(tempDir, ".git")
	err = os.Mkdir(gitDir, 0o750)
	require.NoError(t, err)

	err = ValidateGroveStructure(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".git should be a file, not a directory")

	// Remove .git directory and create .git file
	err = os.Remove(gitDir)
	require.NoError(t, err)

	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
	require.NoError(t, err)

	// Test without .bare directory
	err = ValidateGroveStructure(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".bare directory does not exist")

	// Create .bare directory but not a real git repo
	bareDir := filepath.Join(tempDir, ".bare")
	err = os.Mkdir(bareDir, 0o750)
	require.NoError(t, err)

	// Should fail git status test
	err = ValidateGroveStructure(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git status failed")
}

func TestConvertToGroveStructureSuccess(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-convert-success-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Initialize a real git repository
	_, err = ExecuteGit("init", tempDir)
	require.NoError(t, err)

	// Create a dummy file and commit it to have a clean status
	dummyFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(dummyFile, []byte("# Test\n"), 0o644)
	require.NoError(t, err)

	// Change to temp directory to run git commands
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	_, err = ExecuteGit("add", ".")
	require.NoError(t, err)

	_, err = ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Verify it's a traditional repo
	assert.True(t, IsTraditionalRepo(tempDir))
	assert.False(t, IsGroveRepo(tempDir))

	// Use mock executor to bypass safety checks for this test
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSafeRepositoryState()

	// Convert to Grove structure using mock executor
	err = ConvertToGroveStructureWithExecutor(mockExecutor, tempDir)
	require.NoError(t, err)

	// Verify conversion
	assert.False(t, IsTraditionalRepo(tempDir))
	assert.True(t, IsGroveRepo(tempDir))

	// Verify files exist
	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(tempDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify .git file content
	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(content))

	// Verify git operations still work
	_, err = ExecuteGit("status")
	require.NoError(t, err)
}

func TestConvertToGroveStructureFailures(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grove-convert-fail1-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		err = ConvertToGroveStructure(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not contain a traditional Git repository")
	})

	t.Run("already has .bare directory", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grove-convert-fail2-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Create .git directory
		gitDir := filepath.Join(tempDir, ".git")
		err = os.Mkdir(gitDir, 0o750)
		require.NoError(t, err)

		// Create .bare directory
		bareDir := filepath.Join(tempDir, ".bare")
		err = os.Mkdir(bareDir, 0o750)
		require.NoError(t, err)

		err = ConvertToGroveStructure(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ".bare directory already exists")
	})

	t.Run("already Grove repo", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grove-convert-fail3-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Create Grove structure
		gitFile := filepath.Join(tempDir, ".git")
		err = os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
		require.NoError(t, err)

		bareDir := filepath.Join(tempDir, ".bare")
		err = os.Mkdir(bareDir, 0o750)
		require.NoError(t, err)

		err = ConvertToGroveStructure(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not contain a traditional Git repository")
	})
}

func TestCheckRepositorySafetyForConversion(t *testing.T) {
	t.Run("unsafe repository with multiple issues", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grove-safety-unsafe-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Initialize repository
		_, err = ExecuteGit("init", tempDir)
		require.NoError(t, err)

		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer func() { _ = os.Chdir(originalDir) }()

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Create uncommitted changes
		testFile := filepath.Join(tempDir, "test.txt")
		err = os.WriteFile(testFile, []byte("test content"), 0o644)
		require.NoError(t, err)

		// Create untracked files
		untrackedFile := filepath.Join(tempDir, "untracked.txt")
		err = os.WriteFile(untrackedFile, []byte("untracked"), 0o644)
		require.NoError(t, err)

		// Create a stash (need to commit first to have something to stash)
		_, err = ExecuteGit("add", "test.txt")
		require.NoError(t, err)

		_, err = ExecuteGit("commit", "-m", "Initial commit")
		require.NoError(t, err)

		// Modify file and stash
		err = os.WriteFile(testFile, []byte("modified content"), 0o644)
		require.NoError(t, err)

		_, err = ExecuteGit("stash")
		require.NoError(t, err)

		// Create another uncommitted change
		err = os.WriteFile(testFile, []byte("another change"), 0o644)
		require.NoError(t, err)

		// Check safety - should have multiple issues
		issues, err := checkRepositorySafetyForConversion(DefaultExecutor, tempDir)
		require.NoError(t, err)
		assert.NotEmpty(t, issues)

		// Should have issues for uncommitted changes, stashed changes, and untracked files
		issueTypes := make(map[string]bool)
		for _, issue := range issues {
			issueTypes[issue.Type] = true
		}

		assert.True(t, issueTypes["uncommitted_changes"], "Should detect uncommitted changes")
		assert.True(t, issueTypes["stashed_changes"], "Should detect stashed changes")
		assert.True(t, issueTypes["untracked_files"], "Should detect untracked files")
	})
}
