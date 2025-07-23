//go:build integration
// +build integration

package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommandLocal(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-local-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	testDir := filepath.Join(tempDir, "test-repo")
	cmd := NewInitCmd()
	cmd.SetArgs([]string{testDir})

	err = cmd.Execute()
	require.NoError(t, err)

	assert.DirExists(t, testDir)

	bareDir := filepath.Join(testDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(testDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify .git file content.
	content, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(content))

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
	tempDir, err := os.MkdirTemp("", "grove-init-current-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

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

	gitFile := filepath.Join(tempDir, ".git")
	err = os.WriteFile(gitFile, []byte("existing"), 0o600)
	require.NoError(t, err)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{tempDir})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository already exists at")
}

func TestInitCommandExistingBareDir(t *testing.T) {
	// Create temporary directory.
	tempDir, err := os.MkdirTemp("", "grove-init-bare-exists-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	bareDir := filepath.Join(tempDir, ".bare")
	err = os.MkdirAll(bareDir, 0o750)
	require.NoError(t, err)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{tempDir})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository already exists at")
}

// Note: Testing the remote clone functionality would require actual network access
// or complex mocking. These tests focus on the local functionality and URL detection.
// Integration tests with real repositories should be done separately.

func TestInitFromRemoteNonEmptyDirectory(t *testing.T) {
	// Create temporary directory with a file.
	tempDir, err := os.MkdirTemp("", "grove-init-remote-nonempty-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	testFile := filepath.Join(tempDir, "existing.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o600)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	cmd := NewInitCmd()
	cmd.SetArgs([]string{"https://github.com/user/repo.git"})

	err = cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "is not empty")
}

func TestInitCommandConvertSuccess(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-convert-success-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Initialize a real git repository
	_, err = git.ExecuteGit("init", tempDir)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create multiple tracked files with different content types
	files := map[string]string{
		"README.md":    "# Test Repository\n",
		"main.js":      "console.log('Hello World');\n",
		"package.json": `{"name": "test", "version": "1.0.0"}`,
		"src/app.js":   "// Main application code\n",
	}

	// Create directories and files
	for filePath, content := range files {
		dir := filepath.Dir(filePath)
		if dir != "." {
			err = os.MkdirAll(filepath.Join(tempDir, dir), 0o755)
			require.NoError(t, err)
		}
		err = os.WriteFile(filepath.Join(tempDir, filePath), []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create .gitignore file
	gitignoreContent := "*.log\n.env\nnode_modules/\n.DS_Store\n"
	err = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignoreContent), 0o644)
	require.NoError(t, err)

	// Create some gitignored files that should be preserved
	gitignoredFiles := map[string]string{
		".env":             "DATABASE_URL=localhost\nSECRET_KEY=secret\n",
		"debug.log":        "Debug log content\n",
		".DS_Store":        "Mac OS metadata\n",
		"node_modules/pkg": "Package content\n",
	}

	for filePath, content := range gitignoredFiles {
		dir := filepath.Dir(filePath)
		if dir != "." {
			err = os.MkdirAll(filepath.Join(tempDir, dir), 0o755)
			require.NoError(t, err)
		}
		err = os.WriteFile(filepath.Join(tempDir, filePath), []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Commit the tracked files
	_, err = git.ExecuteGit("add", ".")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Set up fake remote to pass safety checks
	fakeRemoteDir := filepath.Join(tempDir, "..", "fake-remote.git")
	_, err = git.ExecuteGit("init", "--bare", fakeRemoteDir)
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(fakeRemoteDir) }()

	_, err = git.ExecuteGit("remote", "add", "origin", fakeRemoteDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("push", "-u", "origin", "main")
	require.NoError(t, err)

	// Verify it's a traditional repo before conversion
	assert.True(t, git.IsTraditionalRepo(tempDir))
	assert.False(t, git.IsGroveRepo(tempDir))

	// Get list of all files before conversion
	beforeFiles := make(map[string]string)
	err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(tempDir, path)
		if err != nil {
			return err
		}
		// Skip .git directory contents
		if strings.HasPrefix(relPath, ".git/") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		beforeFiles[relPath] = string(content)
		return nil
	})
	require.NoError(t, err)

	// Execute successful conversion
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--convert"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify conversion was successful
	assert.False(t, git.IsTraditionalRepo(tempDir))
	assert.True(t, git.IsGroveRepo(tempDir))

	// Verify Grove structure exists
	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(tempDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify .git file content
	gitContent, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(gitContent))

	// Verify worktree was created
	worktreeDir := filepath.Join(tempDir, "main")
	assert.DirExists(t, worktreeDir)

	// Verify worktree .git file
	worktreeGitFile := filepath.Join(worktreeDir, ".git")
	assert.FileExists(t, worktreeGitFile)

	worktreeGitContent, err := os.ReadFile(worktreeGitFile)
	require.NoError(t, err)
	assert.Contains(t, string(worktreeGitContent), "gitdir:")
	assert.Contains(t, string(worktreeGitContent), ".bare/worktrees/main")

	// Test git operations work in worktree
	worktreeOriginalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(worktreeDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("status")
	require.NoError(t, err)

	_, err = git.ExecuteGit("log", "--oneline")
	require.NoError(t, err)

	// Verify all tracked files are present in worktree
	for filePath, expectedContent := range files {
		if filePath == "src/app.js" {
			assert.DirExists(t, filepath.Join(worktreeDir, "src"))
		}
		actualContent, err := os.ReadFile(filepath.Join(worktreeDir, filePath))
		require.NoError(t, err, "File %s should exist in worktree", filePath)
		assert.Equal(t, expectedContent, string(actualContent), "File %s content should match", filePath)
	}

	// Verify .gitignore is present
	gitignoreActual, err := os.ReadFile(filepath.Join(worktreeDir, ".gitignore"))
	require.NoError(t, err)
	assert.Equal(t, gitignoreContent, string(gitignoreActual))

	// Test creating new files and committing
	testFile := filepath.Join(worktreeDir, "test.txt")
	err = os.WriteFile(testFile, []byte("Test content"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "test.txt")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Add test file")
	require.NoError(t, err)

	// Verify the commit was successful
	output, err := git.ExecuteGit("log", "--oneline")
	require.NoError(t, err)
	assert.Contains(t, output, "Add test file")

	// Return to original directory
	err = os.Chdir(worktreeOriginalDir)
	require.NoError(t, err)

	// CRITICAL: Verify gitignored files are preserved
	// This tests the file handling during conversion
	for filePath, expectedContent := range gitignoredFiles {
		// Check if file exists in worktree (it should)
		actualContent, err := os.ReadFile(filepath.Join(worktreeDir, filePath))
		if err != nil {
			t.Errorf("Gitignored file %s should be preserved during conversion but was not found", filePath)
			continue
		}
		assert.Equal(t, expectedContent, string(actualContent), "Gitignored file %s content should be preserved", filePath)
	}
}

func TestInitCommandConvertNonMainBranch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "grove-init-convert-nonmain-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Initialize a real git repository
	_, err = git.ExecuteGit("init", tempDir)
	require.NoError(t, err)

	// Save current directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Change to temp directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create initial commit on main branch
	files := map[string]string{
		"README.md":    "# Test Repository\n",
		"main.js":      "console.log('Hello World');\n",
		"package.json": `{"name": "test", "version": "1.0.0"}`,
	}

	for filePath, content := range files {
		err = os.WriteFile(filepath.Join(tempDir, filePath), []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create .gitignore file
	gitignoreContent := "*.log\n.env\nnode_modules/\n.DS_Store\n"
	err = os.WriteFile(filepath.Join(tempDir, ".gitignore"), []byte(gitignoreContent), 0o644)
	require.NoError(t, err)

	// Commit initial files
	_, err = git.ExecuteGit("add", ".")
	require.NoError(t, err)
	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Create and checkout feature branch
	branchName := "feat/user-auth"
	_, err = git.ExecuteGit("checkout", "-b", branchName)
	require.NoError(t, err)

	// Add feature-specific files
	featureFiles := map[string]string{
		"src/auth.js":        "// Authentication module\nmodule.exports = {};\n",
		"src/user.js":        "// User model\nclass User {}\nmodule.exports = User;\n",
		"tests/auth.test.js": "// Auth tests\ndescribe('Auth', () => {});\n",
	}

	for filePath, content := range featureFiles {
		dir := filepath.Dir(filePath)
		if dir != "." {
			err = os.MkdirAll(filepath.Join(tempDir, dir), 0o755)
			require.NoError(t, err)
		}
		err = os.WriteFile(filepath.Join(tempDir, filePath), []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Create gitignored files specific to this branch
	gitignoredFiles := map[string]string{
		".env":                 "AUTH_SECRET=secret123\nDB_URL=localhost\n",
		"debug.log":            "Feature branch debug log\n",
		"node_modules/express": "Express package content\n",
		".DS_Store":            "Mac OS metadata\n",
	}

	for filePath, content := range gitignoredFiles {
		dir := filepath.Dir(filePath)
		if dir != "." {
			err = os.MkdirAll(filepath.Join(tempDir, dir), 0o755)
			require.NoError(t, err)
		}
		err = os.WriteFile(filepath.Join(tempDir, filePath), []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Commit feature branch changes
	_, err = git.ExecuteGit("add", "src/", "tests/")
	require.NoError(t, err)
	_, err = git.ExecuteGit("commit", "-m", "Add user authentication feature")
	require.NoError(t, err)

	// Set up fake remote to pass safety checks
	fakeRemoteDir := filepath.Join(tempDir, "..", "fake-remote.git")
	_, err = git.ExecuteGit("init", "--bare", fakeRemoteDir)
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(fakeRemoteDir) }()

	_, err = git.ExecuteGit("remote", "add", "origin", fakeRemoteDir)
	require.NoError(t, err)

	// Push both branches to remote
	_, err = git.ExecuteGit("push", "-u", "origin", "main")
	require.NoError(t, err)
	_, err = git.ExecuteGit("push", "-u", "origin", branchName)
	require.NoError(t, err)

	// Verify current branch is the feature branch
	currentBranch, err := git.ExecuteGit("branch", "--show-current")
	require.NoError(t, err)
	assert.Equal(t, branchName, strings.TrimSpace(currentBranch))

	// Verify it's a traditional repo before conversion
	assert.True(t, git.IsTraditionalRepo(tempDir))
	assert.False(t, git.IsGroveRepo(tempDir))

	// Execute conversion
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--convert"})

	err = cmd.Execute()
	require.NoError(t, err)

	// Verify conversion was successful
	assert.False(t, git.IsTraditionalRepo(tempDir))
	assert.True(t, git.IsGroveRepo(tempDir))

	// Verify Grove structure exists
	bareDir := filepath.Join(tempDir, ".bare")
	assert.DirExists(t, bareDir)

	gitFile := filepath.Join(tempDir, ".git")
	assert.FileExists(t, gitFile)

	// Verify .git file content
	gitContent, err := os.ReadFile(gitFile)
	require.NoError(t, err)
	assert.Equal(t, "gitdir: .bare\n", string(gitContent))

	// CRITICAL: Verify worktree was created with feature branch name, not "main"
	// Git converts slashes to hyphens for directory names
	sanitizedBranchName := strings.ReplaceAll(branchName, "/", "-")
	worktreeDir := filepath.Join(tempDir, sanitizedBranchName)
	assert.DirExists(t, worktreeDir, "Worktree should be created with feature branch name")

	// Verify no main worktree was created
	mainWorktreeDir := filepath.Join(tempDir, "main")
	assert.NoDirExists(t, mainWorktreeDir, "Main worktree should not be created automatically")

	// Verify worktree .git file
	worktreeGitFile := filepath.Join(worktreeDir, ".git")
	assert.FileExists(t, worktreeGitFile)

	worktreeGitContent, err := os.ReadFile(worktreeGitFile)
	require.NoError(t, err)
	assert.Contains(t, string(worktreeGitContent), "gitdir:")

	// Git worktree names sanitize branch names by replacing slashes with dashes
	// "feat/user-auth" becomes "feat-user-auth" in the worktree directory name
	expectedWorktreeName := strings.ReplaceAll(branchName, "/", "-")
	assert.Contains(t, string(worktreeGitContent), fmt.Sprintf(".bare/worktrees/%s", expectedWorktreeName))

	// Verify all tracked files are preserved in feature branch worktree
	allFiles := make(map[string]string)
	for k, v := range files {
		allFiles[k] = v
	}
	for k, v := range featureFiles {
		allFiles[k] = v
	}
	allFiles[".gitignore"] = gitignoreContent

	for filePath, expectedContent := range allFiles {
		actualContent, err := os.ReadFile(filepath.Join(worktreeDir, filePath))
		require.NoError(t, err, "File %s should exist in feature branch worktree", filePath)
		assert.Equal(t, expectedContent, string(actualContent), "File %s content should be preserved", filePath)
	}

	// CRITICAL: Verify gitignored files are preserved in feature branch worktree
	for filePath, expectedContent := range gitignoredFiles {
		actualContent, err := os.ReadFile(filepath.Join(worktreeDir, filePath))
		if err != nil {
			t.Errorf("Gitignored file %s should be preserved during conversion but was not found", filePath)
			continue
		}
		assert.Equal(t, expectedContent, string(actualContent), "Gitignored file %s content should be preserved", filePath)
	}

	// Verify git functionality works in the new worktree
	originalDir2, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir2) }()

	err = os.Chdir(worktreeDir)
	require.NoError(t, err)

	// Test git status works
	_, err = git.ExecuteGit("status")
	require.NoError(t, err)

	// Test current branch is correct
	currentBranchAfter, err := git.ExecuteGit("branch", "--show-current")
	require.NoError(t, err)
	assert.Equal(t, branchName, strings.TrimSpace(currentBranchAfter))

	// Test that we can create new commits
	err = os.WriteFile(filepath.Join(worktreeDir, "new-file.txt"), []byte("test content"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "new-file.txt")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Test commit after conversion")
	require.NoError(t, err)
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
		assert.Contains(t, err.Error(), "repository not found at")
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
		assert.Contains(t, err.Error(), "repository already exists at")
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
