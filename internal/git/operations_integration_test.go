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
	output, err := ExecuteGit("--version")
	require.NoError(t, err)
	assert.Contains(t, output, "git version", "should return git version")

	_, err = ExecuteGit("invalid-command")
	require.Error(t, err)

	var gitErr *GitError
	require.ErrorAs(t, err, &gitErr, "error should be a GitError")
	assert.Equal(t, "git", gitErr.Command)
	assert.Contains(t, gitErr.Args, "invalid-command")
	assert.NotEqual(t, 0, gitErr.ExitCode)
}

func TestInitBare(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	bareDir := helper.CreateTempDir("test-bare.git")

	err := InitBare(bareDir)
	require.NoError(t, err)

	assert.DirExists(t, bareDir)

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
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	mainDir := helper.CreateTempDir("main")
	bareDir := helper.CreateTempDir("main/.bare")

	err := CreateGitFile(mainDir, bareDir)
	require.NoError(t, err)

	gitFile := filepath.Join(mainDir, ".git")
	assert.FileExists(t, gitFile)

	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(content))
}

func TestCreateGitFileAbsolutePath(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	mainDir := helper.CreateTempDir("main")
	bareDir := helper.CreateTempDir("separate/bare")

	err := CreateGitFile(mainDir, bareDir)
	require.NoError(t, err)

	gitFile := filepath.Join(mainDir, ".git")
	assert.FileExists(t, gitFile)

	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Contains(t, string(content), "gitdir:")
}

func TestIsTraditionalRepo(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tempDir := helper.CreateTempDir("traditional-test")

	assert.False(t, IsTraditionalRepo(tempDir))

	gitDir := filepath.Join(tempDir, ".git")
	err := os.Mkdir(gitDir, 0o750)
	require.NoError(t, err)

	assert.True(t, IsTraditionalRepo(tempDir))

	// Test with .git file (Grove repo) - should be false.
	err = os.Remove(gitDir)
	require.NoError(t, err)

	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
	require.NoError(t, err)

	assert.False(t, IsTraditionalRepo(tempDir))
}

func TestIsGroveRepo(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tempDir := helper.CreateTempDir("grove-test")

	assert.False(t, IsGroveRepo(tempDir))

	// Create only .git file - should be false (no .bare).
	gitFile := filepath.Join(tempDir, ".git")
	err := os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
	require.NoError(t, err)

	assert.False(t, IsGroveRepo(tempDir))

	bareDir := filepath.Join(tempDir, ".bare")
	err = os.Mkdir(bareDir, 0o750)
	require.NoError(t, err)

	assert.True(t, IsGroveRepo(tempDir))

	// Test with .git directory instead of file - should be false.
	err = os.Remove(gitFile)
	require.NoError(t, err)

	err = os.Mkdir(gitFile, 0o750)
	require.NoError(t, err)

	assert.False(t, IsGroveRepo(tempDir))
}

func TestValidateGroveStructure(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tempDir := helper.CreateTempDir("validate-test")

	err := ValidateGroveStructure(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".git file does not exist")

	gitDir := filepath.Join(tempDir, ".git")
	err = os.Mkdir(gitDir, 0o750)
	require.NoError(t, err)

	err = ValidateGroveStructure(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".git should be a file, not a directory")

	err = os.Remove(gitDir)
	require.NoError(t, err)

	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
	require.NoError(t, err)

	err = ValidateGroveStructure(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), ".bare directory does not exist")

	bareDir := filepath.Join(tempDir, ".bare")
	err = os.Mkdir(bareDir, 0o750)
	require.NoError(t, err)

	err = ValidateGroveStructure(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git status failed")
}

func TestConvertToGroveStructureSuccess(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	runner := testutils.NewTestRunner(t)

	tempDir := helper.CreateTempDir("convert-success")

	_, err := ExecuteGit("init", tempDir)
	require.NoError(t, err)

	dummyFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(dummyFile, []byte("# Test\n"), 0o644)
	require.NoError(t, err)

	runner.WithIsolatedWorkingDir().Run(func() {
		err := os.Chdir(tempDir)
		require.NoError(t, err)

		_, err = ExecuteGit("add", ".")
		require.NoError(t, err)

		_, err = ExecuteGit("commit", "-m", "Initial commit")
		require.NoError(t, err)
	})

	assert.True(t, IsTraditionalRepo(tempDir))
	assert.False(t, IsGroveRepo(tempDir))

	// Use mock executor to bypass safety checks for this test.
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSafeRepositoryState()

	err = ConvertToGroveStructureWithExecutor(mockExecutor, tempDir)
	require.NoError(t, err)

	assert.False(t, IsTraditionalRepo(tempDir))
	assert.True(t, IsGroveRepo(tempDir))

	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(tempDir, ".git")
	assert.FileExists(t, gitFile)

	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(content))

	_, err = ExecuteGit("status")
	require.NoError(t, err)
}

func TestConvertToGroveStructureFailures(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		tempDir := helper.CreateTempDir("convert-fail1")

		err := ConvertToGroveStructure(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not contain a traditional Git repository")
	})

	t.Run("already has .bare directory", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		tempDir := helper.CreateTempDir("convert-fail2")

		gitDir := filepath.Join(tempDir, ".git")
		err := os.Mkdir(gitDir, 0o750)
		require.NoError(t, err)

		bareDir := filepath.Join(tempDir, ".bare")
		err = os.Mkdir(bareDir, 0o750)
		require.NoError(t, err)

		err = ConvertToGroveStructure(tempDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), ".bare directory already exists")
	})

	t.Run("already Grove repo", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		tempDir := helper.CreateTempDir("convert-fail3")

		gitFile := filepath.Join(tempDir, ".git")
		err := os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
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
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		runner := testutils.NewTestRunner(t)

		tempDir := helper.CreateTempDir("safety-unsafe")

		_, err := ExecuteGit("init", tempDir)
		require.NoError(t, err)

		runner.WithIsolatedWorkingDir().Run(func() {
			err := os.Chdir(tempDir)
			require.NoError(t, err)

			testFile := filepath.Join(tempDir, "test.txt")
			err = os.WriteFile(testFile, []byte("test content"), 0o644)
			require.NoError(t, err)

			untrackedFile := filepath.Join(tempDir, "untracked.txt")
			err = os.WriteFile(untrackedFile, []byte("untracked"), 0o644)
			require.NoError(t, err)

			// Create a stash (need to commit first to have something to stash).
			_, err = ExecuteGit("add", "test.txt")
			require.NoError(t, err)

			_, err = ExecuteGit("commit", "-m", "Initial commit")
			require.NoError(t, err)

			err = os.WriteFile(testFile, []byte("modified content"), 0o644)
			require.NoError(t, err)

			_, err = ExecuteGit("stash")
			require.NoError(t, err)

			err = os.WriteFile(testFile, []byte("another change"), 0o644)
			require.NoError(t, err)
		})

		issues, err := checkRepositorySafetyForConversion(DefaultExecutor, tempDir)
		require.NoError(t, err)
		assert.NotEmpty(t, issues)

		// Should have issues for uncommitted changes, stashed changes, and untracked files.
		issueTypes := make(map[string]bool)
		for _, issue := range issues {
			issueTypes[issue.Type] = true
		}

		assert.True(t, issueTypes["uncommitted_changes"], "Should detect uncommitted changes")
		assert.True(t, issueTypes["stashed_changes"], "Should detect stashed changes")
		assert.True(t, issueTypes["untracked_files"], "Should detect untracked files")
	})
}
