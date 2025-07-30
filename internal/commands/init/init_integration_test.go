//go:build integration
// +build integration

package init

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutils"
	"github.com/sqve/grove/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommandLocal(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir := helper.CreateTempDir("grove-init-local")
		testDir := filepath.Join(tempDir, "test-repo")

		cmd := NewInitCmd()
		cmd.SetArgs([]string{testDir})

		err := cmd.Execute()
		require.NoError(t, err)

		assert.DirExists(t, testDir)

		bareDir := filepath.Join(testDir, ".bare")
		assert.DirExists(t, bareDir)

		gitFile := filepath.Join(testDir, ".git")
		assert.FileExists(t, gitFile)

		content, err := os.ReadFile(gitFile)
		require.NoError(t, err)
		assert.Equal(t, "gitdir: .bare\n", string(content))

		err = os.Chdir(testDir)
		require.NoError(t, err)

		isRepo, err := utils.IsGitRepository(git.DefaultExecutor)
		require.NoError(t, err)
		assert.True(t, isRepo)

		configPath := filepath.Join(bareDir, "config")
		assert.FileExists(t, configPath)

		headPath := filepath.Join(bareDir, "HEAD")
		assert.FileExists(t, headPath)
	})
}

func TestInitCommandCurrentDirectory(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir := helper.CreateTempDir("grove-init-current")

		err := os.Chdir(tempDir)
		require.NoError(t, err)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{})

		err = cmd.Execute()
		require.NoError(t, err)

		bareDir := filepath.Join(tempDir, ".bare")
		assert.DirExists(t, bareDir)

		gitFile := filepath.Join(tempDir, ".git")
		assert.FileExists(t, gitFile)

		isRepo, err := utils.IsGitRepository(git.DefaultExecutor)
		require.NoError(t, err)
		assert.True(t, isRepo)
	})
}

func TestInitCommandExistingGitFile(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tempDir := helper.CreateTempDir("grove-init-existing")
	gitFile := filepath.Join(tempDir, ".git")
	err := os.WriteFile(gitFile, []byte("existing"), 0o600)
	require.NoError(t, err)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{tempDir})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository already exists at")
}

func TestInitCommandExistingBareDir(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tempDir := helper.CreateTempDir("grove-init-bare-exists")
	bareDir := filepath.Join(tempDir, ".bare")
	err := os.Mkdir(bareDir, 0o750)
	require.NoError(t, err)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{tempDir})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository already exists at")
}

// Note: Testing the remote clone functionality would require actual network access.
// or complex mocking. These tests focus on the local functionality and URL detection.
// Integration tests with real repositories should be done separately.

func TestInitFromRemoteNonEmptyDirectory(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir := helper.CreateTempDir("grove-init-remote-nonempty")
		testFile := filepath.Join(tempDir, "existing.txt")
		err := os.WriteFile(testFile, []byte("content"), 0o600)
		require.NoError(t, err)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"https://github.com/user/repo.git"})

		err = cmd.Execute()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is not empty")
	})
}

// createTestFiles creates test files in the specified directory with given content.
// It handles nested directory creation automatically.
func createTestFiles(t *testing.T, baseDir string, files map[string]string) {
	for filePath, content := range files {
		fullPath := filepath.Join(baseDir, filePath)
		dir := filepath.Dir(fullPath)
		if dir != baseDir {
			err := os.MkdirAll(dir, 0o755)
			require.NoError(t, err)
		}
		err := os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err)
	}
}

// setupTraditionalRepoForConvert creates a traditional git repository with test data for conversion testing.
// Returns the repository directory, tracked files map, and gitignored files map.
func setupTraditionalRepoForConvert(t *testing.T, helper *testutils.UnitTestHelper) (string, map[string]string, map[string]string) {
	tempDir := helper.CreateTempDir("grove-init-convert")

	_, err := git.ExecuteGit("init", tempDir)
	require.NoError(t, err)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	files := map[string]string{
		"README.md":    "# Test Repository\n",
		"main.js":      "console.log('Hello World');\n",
		"package.json": `{"name": "test", "version": "1.0.0"}`,
		"src/app.js":   "// Main application code\n",
	}

	gitignoredFiles := map[string]string{
		".env":             "DATABASE_URL=localhost\nSECRET_KEY=secret\n",
		"debug.log":        "Debug log content\n",
		".DS_Store":        "Mac OS metadata\n",
		"node_modules/pkg": "Package content\n",
	}

	createTestFiles(t, tempDir, files)

	gitignoreContent := "*.log\n.env\nnode_modules/\n.DS_Store\n"
	err = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignoreContent), 0o644)
	require.NoError(t, err)

	createTestFiles(t, tempDir, gitignoredFiles)

	_, err = git.ExecuteGit("add", ".")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Set up fake remote to pass safety checks
	fakeRemoteDir := helper.CreateTempDir("fake-remote")
	_, err = git.ExecuteGit("init", "--bare", fakeRemoteDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("remote", "add", "origin", fakeRemoteDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("push", "-u", "origin", "main")
	require.NoError(t, err)

	return tempDir, files, gitignoredFiles
}

func TestInitCommandConvert_repositoryStructure(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir, _, _ := setupTraditionalRepoForConvert(t, helper)

		assert.True(t, git.IsTraditionalRepo(tempDir))
		assert.False(t, git.IsGroveRepo(tempDir))

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})

		err := cmd.Execute()
		require.NoError(t, err)

		assert.False(t, git.IsTraditionalRepo(tempDir))
		assert.True(t, git.IsGroveRepo(tempDir))

		bareDir := filepath.Join(tempDir, ".bare")
		assert.DirExists(t, bareDir)

		gitFile := filepath.Join(tempDir, ".git")
		assert.FileExists(t, gitFile)

		gitContent, err := os.ReadFile(gitFile)
		require.NoError(t, err)
		assert.Equal(t, "gitdir: .bare\n", string(gitContent))

		worktreeDir := filepath.Join(tempDir, "main")
		assert.DirExists(t, worktreeDir)

		worktreeGitFile := filepath.Join(worktreeDir, ".git")
		assert.FileExists(t, worktreeGitFile)

		worktreeGitContent, err := os.ReadFile(worktreeGitFile)
		require.NoError(t, err)
		assert.Contains(t, string(worktreeGitContent), "gitdir:")
		assert.Contains(t, string(worktreeGitContent), ".bare/worktrees/main")
	})
}

func TestInitCommandConvert_filePreservation(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir, files, gitignoredFiles := setupTraditionalRepoForConvert(t, helper)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})

		err := cmd.Execute()
		require.NoError(t, err)

		worktreeDir := filepath.Join(tempDir, "main")

		for filePath, expectedContent := range files {
			if filePath == "src/app.js" {
				assert.DirExists(t, filepath.Join(worktreeDir, "src"))
			}
			actualContent, err := os.ReadFile(filepath.Join(worktreeDir, filePath))
			require.NoError(t, err, "File %s should exist in worktree", filePath)
			assert.Equal(t, expectedContent, string(actualContent), "File %s content should match", filePath)
		}

		gitignoreContent := "*.log\n.env\nnode_modules/\n.DS_Store\n"
		gitignoreActual, err := os.ReadFile(filepath.Join(worktreeDir, ".gitignore"))
		require.NoError(t, err)
		assert.Equal(t, gitignoreContent, string(gitignoreActual))

		for filePath, expectedContent := range gitignoredFiles {
			actualContent, err := os.ReadFile(filepath.Join(worktreeDir, filePath))
			if err != nil {
				t.Errorf("Gitignored file %s should be preserved during conversion but was not found", filePath)
				continue
			}
			assert.Equal(t, expectedContent, string(actualContent), "Gitignored file %s content should be preserved", filePath)
		}
	})
}

func TestInitCommandConvert_gitOperations(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
	runner.Run(func() {
		tempDir, _, _ := setupTraditionalRepoForConvert(t, helper)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})

		err := cmd.Execute()
		require.NoError(t, err)

		worktreeDir := filepath.Join(tempDir, "main")
		worktreeOriginalDir, err := os.Getwd()
		require.NoError(t, err)

		err = os.Chdir(worktreeDir)
		require.NoError(t, err)

		_, err = git.ExecuteGit("status")
		require.NoError(t, err)

		_, err = git.ExecuteGit("log", "--oneline")
		require.NoError(t, err)

		testFile := filepath.Join(worktreeDir, "test.txt")
		err = os.WriteFile(testFile, []byte("Test content"), 0o644)
		require.NoError(t, err)

		_, err = git.ExecuteGit("add", "test.txt")
		require.NoError(t, err)

		_, err = git.ExecuteGit("commit", "-m", "Add test file")
		require.NoError(t, err)

		output, err := git.ExecuteGit("log", "--oneline")
		require.NoError(t, err)
		assert.Contains(t, output, "Add test file")

		err = os.Chdir(worktreeOriginalDir)
		require.NoError(t, err)
	})
}

// TestInitCommandConvertNonMainBranch tests converting repositories when checked out to non-main branches
func TestInitCommandConvertNonMainBranch(t *testing.T) {
	t.Run("setup and convert", func(t *testing.T) {
		testConvertNonMainBranchSetup(t)
	})

	t.Run("file preservation", func(t *testing.T) {
		testConvertNonMainBranchFilePreservation(t)
	})

	t.Run("git operations after conversion", func(t *testing.T) {
		testConvertNonMainBranchGitOperations(t)
	})
}

// testConvertNonMainBranchSetup tests the basic setup and conversion of a non-main branch repository
func testConvertNonMainBranchSetup(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()

	runner.Run(func() {
		tempDir, branchName := setupFeatureBranchRepo(t, helper)

		assert.True(t, git.IsTraditionalRepo(tempDir))
		assert.False(t, git.IsGroveRepo(tempDir))

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})
		err := cmd.Execute()
		require.NoError(t, err)

		assert.False(t, git.IsTraditionalRepo(tempDir))
		assert.True(t, git.IsGroveRepo(tempDir))

		verifyConvertedRepoStructure(t, tempDir, branchName)
	})
}

// testConvertNonMainBranchFilePreservation tests that all files are preserved during conversion
func testConvertNonMainBranchFilePreservation(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()

	runner.Run(func() {
		tempDir, branchName := setupFeatureBranchRepo(t, helper)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})
		err := cmd.Execute()
		require.NoError(t, err)

		verifyFeatureBranchFilesPreserved(t, tempDir, branchName)
	})
}

// testConvertNonMainBranchGitOperations tests git operations work after conversion
func testConvertNonMainBranchGitOperations(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()

	runner.Run(func() {
		tempDir, branchName := setupFeatureBranchRepo(t, helper)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert"})
		err := cmd.Execute()
		require.NoError(t, err)

		sanitizedBranchName := strings.ReplaceAll(branchName, "/", "-")
		worktreeDir := filepath.Join(tempDir, sanitizedBranchName)

		err = os.Chdir(worktreeDir)
		require.NoError(t, err)

		_, err = git.ExecuteGit("status")
		require.NoError(t, err)

		currentBranch, err := git.ExecuteGit("branch", "--show-current")
		require.NoError(t, err)
		assert.Equal(t, branchName, strings.TrimSpace(currentBranch))

		err = os.WriteFile(filepath.Join(worktreeDir, "new-file.txt"), []byte("test content"), 0o644)
		require.NoError(t, err)

		_, err = git.ExecuteGit("add", "new-file.txt")
		require.NoError(t, err)

		_, err = git.ExecuteGit("commit", "-m", "Test commit after conversion")
		require.NoError(t, err)
	})
}

// setupFeatureBranchRepo creates a test repository with a feature branch and returns the directory and branch name.
// This helper consolidates the common setup logic for feature branch conversion tests.
func setupFeatureBranchRepo(t *testing.T, helper *testutils.UnitTestHelper) (string, string) {
	t.Helper()

	tempDir := helper.CreateTempDir("grove-init-convert-nonmain")

	_, err := git.ExecuteGit("init", tempDir)
	require.NoError(t, err)

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	files := map[string]string{
		"README.md":    "# Test Repository\n",
		"main.js":      "console.log('Hello World');\n",
		"package.json": `{"name": "test", "version": "1.0.0"}`,
	}

	createTestFiles(t, tempDir, files)

	gitignoreContent := "*.log\n.env\nnode_modules/\n.DS_Store\n"
	err = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignoreContent), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", ".")
	require.NoError(t, err)
	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	branchName := "feat/user-auth"
	_, err = git.ExecuteGit("checkout", "-b", branchName)
	require.NoError(t, err)

	featureFiles := map[string]string{
		"src/auth.js":        "// Authentication module\nmodule.exports = {};\n",
		"src/user.js":        "// User model\nclass User {}\nmodule.exports = User;\n",
		"tests/auth.test.js": "// Auth tests\ndescribe('Auth', () => {});\n",
	}

	createTestFiles(t, tempDir, featureFiles)
	gitignoredFiles := map[string]string{
		".env":                 "AUTH_SECRET=secret123\nDB_URL=localhost\n",
		"debug.log":            "Feature branch debug log\n",
		"node_modules/express": "Express package content\n",
		".DS_Store":            "Mac OS metadata\n",
	}

	createTestFiles(t, tempDir, gitignoredFiles)

	_, err = git.ExecuteGit("add", "src/", "tests/")
	require.NoError(t, err)
	_, err = git.ExecuteGit("commit", "-m", "Add user authentication feature")
	require.NoError(t, err)

	fakeRemoteDir := helper.CreateTempDir("fake-remote")
	_, err = git.ExecuteGit("init", "--bare", fakeRemoteDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("remote", "add", "origin", fakeRemoteDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("push", "-u", "origin", "main")
	require.NoError(t, err)
	_, err = git.ExecuteGit("push", "-u", "origin", branchName)
	require.NoError(t, err)

	return tempDir, branchName
}

// verifyConvertedRepoStructure verifies the basic structure after conversion.
func verifyConvertedRepoStructure(t *testing.T, tempDir, branchName string) {
	t.Helper()

	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(tempDir, ".git")
	assert.FileExists(t, gitFile)

	gitContent, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(gitContent))

	// Verify worktree was created with feature branch name, not "main"
	sanitizedBranchName := strings.ReplaceAll(branchName, "/", "-")
	worktreeDir := filepath.Join(tempDir, sanitizedBranchName)
	assert.DirExists(t, worktreeDir, "Worktree should be created with feature branch name")

	mainWorktreeDir := filepath.Join(tempDir, "main")
	assert.NoDirExists(t, mainWorktreeDir, "Main worktree should not be created automatically")

	worktreeGitFile := filepath.Join(worktreeDir, ".git")
	assert.FileExists(t, worktreeGitFile)

	worktreeGitContent, err := os.ReadFile(worktreeGitFile)
	require.NoError(t, err)
	assert.Contains(t, string(worktreeGitContent), "gitdir:")

	expectedWorktreeName := strings.ReplaceAll(branchName, "/", "-")
	assert.Contains(t, string(worktreeGitContent), fmt.Sprintf(".bare/worktrees/%s", expectedWorktreeName))
}

// verifyFeatureBranchFilesPreserved verifies all files are preserved during conversion.
func verifyFeatureBranchFilesPreserved(t *testing.T, tempDir, branchName string) {
	t.Helper()

	sanitizedBranchName := strings.ReplaceAll(branchName, "/", "-")
	worktreeDir := filepath.Join(tempDir, sanitizedBranchName)

	allFiles := map[string]string{
		"README.md":          "# Test Repository\n",
		"main.js":            "console.log('Hello World');\n",
		"package.json":       `{"name": "test", "version": "1.0.0"}`,
		"src/auth.js":        "// Authentication module\nmodule.exports = {};\n",
		"src/user.js":        "// User model\nclass User {}\nmodule.exports = User;\n",
		"tests/auth.test.js": "// Auth tests\ndescribe('Auth', () => {});\n",
		".gitignore":         "*.log\n.env\nnode_modules/\n.DS_Store\n",
	}

	gitignoredFiles := map[string]string{
		".env":                 "AUTH_SECRET=secret123\nDB_URL=localhost\n",
		"debug.log":            "Feature branch debug log\n",
		"node_modules/express": "Express package content\n",
		".DS_Store":            "Mac OS metadata\n",
	}

	for filePath, expectedContent := range allFiles {
		actualContent, err := os.ReadFile(filepath.Join(worktreeDir, filePath))
		require.NoError(t, err, "File %s should exist in feature branch worktree", filePath)
		assert.Equal(t, expectedContent, string(actualContent), "File %s content should be preserved", filePath)
	}

	for filePath, expectedContent := range gitignoredFiles {
		actualContent, err := os.ReadFile(filepath.Join(worktreeDir, filePath))
		if err != nil {
			t.Errorf("Gitignored file %s should be preserved during conversion but was not found", filePath)
			continue
		}
		assert.Equal(t, expectedContent, string(actualContent), "Gitignored file %s content should be preserved", filePath)
	}
}

func TestInitCommandConvertFailures(t *testing.T) {
	t.Run("not a git repo", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			tempDir := helper.CreateTempDir("grove-init-convert-fail1")

			err := os.Chdir(tempDir)
			require.NoError(t, err)

			cmd := NewInitCmd()
			cmd.SetArgs([]string{"--convert"})

			err = cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "repository not found at")
		})
	})

	t.Run("already Grove repo", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			tempDir := helper.CreateTempDir("grove-init-convert-fail2")

			gitFile := filepath.Join(tempDir, ".git")
			err := os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o600)
			require.NoError(t, err)

			bareDir := filepath.Join(tempDir, ".bare")
			err = os.Mkdir(bareDir, 0o750)
			require.NoError(t, err)

			err = os.Chdir(tempDir)
			require.NoError(t, err)

			cmd := NewInitCmd()
			cmd.SetArgs([]string{"--convert"})

			err = cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "repository already exists at")
		})
	})

	t.Run("repository with safety issues", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			tempDir := helper.CreateTempDir("grove-init-convert-safety")

			_, err := git.ExecuteGit("init", tempDir)
			require.NoError(t, err)

			testFile := filepath.Join(tempDir, "test.txt")
			err = os.WriteFile(testFile, []byte("test content"), 0o644)
			require.NoError(t, err)

			untrackedFile := filepath.Join(tempDir, "untracked.txt")
			err = os.WriteFile(untrackedFile, []byte("untracked"), 0o644)
			require.NoError(t, err)

			err = os.Chdir(tempDir)
			require.NoError(t, err)

			cmd := NewInitCmd()
			cmd.SetArgs([]string{"--convert"})

			err = cmd.Execute()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "Repository is not ready for conversion:")
			assert.Contains(t, err.Error(), "untracked file(s)")
			assert.Contains(t, err.Error(), "Please resolve these issues before converting")
		})
	})
}
