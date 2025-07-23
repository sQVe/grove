//go:build integration
// +build integration

package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListCommandWithRealGitRepo(t *testing.T) {
	// Create temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "grove-list-integration-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save original directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize Grove repository structure
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a normal git repository first, then convert to Grove structure
	_, err = git.ExecuteGit("init")
	require.NoError(t, err)

	// Create initial commit to have something to work with
	readmeFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Now convert to Grove structure by moving .git to .bare and creating worktrees
	err = os.Rename(filepath.Join(tempDir, ".git"), filepath.Join(tempDir, ".bare"))
	require.NoError(t, err)

	// Create .git file pointing to bare repo
	bareDir := filepath.Join(tempDir, ".bare")
	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	// Create main worktree with a new branch name to avoid conflicts
	mainWorktreeDir := filepath.Join(tempDir, "main")
	_, err = git.ExecuteGit("worktree", "add", "-b", "main-worktree", mainWorktreeDir)
	require.NoError(t, err)

	// Test that list command works with single worktree
	options := &ListOptions{Sort: SortByActivity}
	err = runListCommand(options)
	require.NoError(t, err)
}

func TestListCommandWithMultipleWorktrees(t *testing.T) {
	// Create temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "grove-list-multi-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save original directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize Grove repository structure
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a normal git repository first, then convert to Grove structure
	_, err = git.ExecuteGit("init")
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Now convert to Grove structure by moving .git to .bare
	err = os.Rename(filepath.Join(tempDir, ".git"), filepath.Join(tempDir, ".bare"))
	require.NoError(t, err)

	// Create .git file pointing to bare repo
	bareDir := filepath.Join(tempDir, ".bare")
	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	// Create main worktree with a new branch name to avoid conflicts
	mainWorktreeDir := filepath.Join(tempDir, "main")
	_, err = git.ExecuteGit("worktree", "add", "-b", "main-worktree", mainWorktreeDir)
	require.NoError(t, err)

	// Create feature worktree with some files
	featureWorktreeDir := filepath.Join(tempDir, "feature-auth")
	_, err = git.ExecuteGit("worktree", "add", "-b", "feature/auth", featureWorktreeDir)
	require.NoError(t, err)

	// Add some files to feature worktree
	err = os.Chdir(featureWorktreeDir)
	require.NoError(t, err)

	authFile := filepath.Join(featureWorktreeDir, "auth.js")
	err = os.WriteFile(authFile, []byte("// Authentication module\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "auth.js")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Add authentication module")
	require.NoError(t, err)

	// Create some uncommitted changes in feature worktree
	testFile := filepath.Join(featureWorktreeDir, "test.js")
	err = os.WriteFile(testFile, []byte("// Test file\n"), 0o644)
	require.NoError(t, err)

	// Create untracked file
	untrackedFile := filepath.Join(featureWorktreeDir, "untracked.tmp")
	err = os.WriteFile(untrackedFile, []byte("temporary"), 0o644)
	require.NoError(t, err)

	// Create another worktree with older activity
	oldWorktreeDir := filepath.Join(tempDir, "old-feature")
	_, err = git.ExecuteGit("worktree", "add", "-b", "old/feature", oldWorktreeDir)
	require.NoError(t, err)

	// Wait a moment to ensure different timestamps
	time.Sleep(100 * time.Millisecond)

	// Modify file timestamps to simulate older activity
	oldTime := time.Now().Add(-7 * 24 * time.Hour) // 7 days ago
	err = os.Chtimes(oldWorktreeDir, oldTime, oldTime)
	require.NoError(t, err)

	// Return to main repo directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	t.Run("list all worktrees", func(t *testing.T) {
		options := &ListOptions{Sort: SortByActivity}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("list dirty worktrees only", func(t *testing.T) {
		options := &ListOptions{
			Sort:      SortByActivity,
			DirtyOnly: true,
		}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("list clean worktrees only", func(t *testing.T) {
		options := &ListOptions{
			Sort:      SortByActivity,
			CleanOnly: true,
		}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("list stale worktrees", func(t *testing.T) {
		options := &ListOptions{
			Sort:      SortByActivity,
			StaleOnly: true,
			StaleDays: 3, // Should catch the old worktree
		}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("sort by name", func(t *testing.T) {
		options := &ListOptions{Sort: SortByName}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("sort by status", func(t *testing.T) {
		options := &ListOptions{Sort: SortByStatus}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("verbose output", func(t *testing.T) {
		options := &ListOptions{
			Sort:    SortByActivity,
			Verbose: true,
		}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("porcelain output", func(t *testing.T) {
		options := &ListOptions{
			Sort:      SortByActivity,
			Porcelain: true,
		}
		err = runListCommand(options)
		require.NoError(t, err)
	})
}

func TestListCommandWithComplexBranchNames(t *testing.T) {
	// Create temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "grove-list-complex-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save original directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize Grove repository structure
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a normal git repository first, then convert to Grove structure
	_, err = git.ExecuteGit("init")
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Now convert to Grove structure by moving .git to .bare
	err = os.Rename(filepath.Join(tempDir, ".git"), filepath.Join(tempDir, ".bare"))
	require.NoError(t, err)

	// Create .git file pointing to bare repo
	bareDir := filepath.Join(tempDir, ".bare")
	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	// Test various complex branch name scenarios
	complexBranches := []struct {
		branchName string
		dirName    string
	}{
		{"feature/user-auth", "feature-user-auth"},
		{"bugfix/fix-123", "bugfix-fix-123"},
		{"release/v1.0.0", "release-v1.0.0"},
		{"hotfix/critical-security", "hotfix-critical-security"},
	}

	for _, branch := range complexBranches {
		worktreeDir := filepath.Join(tempDir, branch.dirName)
		_, err = git.ExecuteGit("worktree", "add", "-b", branch.branchName, worktreeDir)
		require.NoError(t, err, "Failed to create worktree for branch: %s", branch.branchName)

		// Add some activity to each worktree
		err = os.Chdir(worktreeDir)
		require.NoError(t, err)

		testFile := filepath.Join(worktreeDir, "test.txt")
		err = os.WriteFile(testFile, []byte("Test content for "+branch.branchName), 0o644)
		require.NoError(t, err)

		_, err = git.ExecuteGit("add", "test.txt")
		require.NoError(t, err)

		_, err = git.ExecuteGit("commit", "-m", "Add test file for "+branch.branchName)
		require.NoError(t, err)

		// Add some waiting time to differentiate activities
		time.Sleep(50 * time.Millisecond)
	}

	// Return to main repo directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Test listing with complex branch names
	options := &ListOptions{Sort: SortByActivity}
	err = runListCommand(options)
	require.NoError(t, err)
}

func TestListCommandWithRemoteTracking(t *testing.T) {
	// Create temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "grove-list-remote-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create a fake remote repository
	remoteDir := filepath.Join(tempDir, "remote.git")
	err = git.InitBare(remoteDir)
	require.NoError(t, err)

	// Save original directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize Grove repository structure
	workDir := filepath.Join(tempDir, "work")
	err = os.MkdirAll(workDir, 0o755)
	require.NoError(t, err)

	err = os.Chdir(workDir)
	require.NoError(t, err)

	// Create a normal git repository first, then convert to Grove structure
	_, err = git.ExecuteGit("init")
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(workDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Add remote origin
	_, err = git.ExecuteGit("remote", "add", "origin", remoteDir)
	require.NoError(t, err)

	// Push to remote to establish tracking
	_, err = git.ExecuteGit("push", "-u", "origin", "main")
	require.NoError(t, err)

	// Now convert to Grove structure by moving .git to .bare
	err = os.Rename(filepath.Join(workDir, ".git"), filepath.Join(workDir, ".bare"))
	require.NoError(t, err)

	// Create .git file pointing to bare repo
	bareDir := filepath.Join(workDir, ".bare")
	err = git.CreateGitFile(workDir, bareDir)
	require.NoError(t, err)

	// Create main worktree with a new branch name to avoid conflicts
	mainWorktreeDir := filepath.Join(workDir, "main")
	_, err = git.ExecuteGit("worktree", "add", "-b", "main-worktree", mainWorktreeDir)
	require.NoError(t, err)

	// Set up remote tracking for the main worktree
	err = os.Chdir(mainWorktreeDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("push", "-u", "origin", "main")
	require.NoError(t, err)

	// Create some commits to be ahead
	for i := 0; i < 2; i++ {
		testFile := filepath.Join(mainWorktreeDir, "file"+string(rune('a'+i))+".txt")
		err = os.WriteFile(testFile, []byte("Content"), 0o644)
		require.NoError(t, err)

		_, err = git.ExecuteGit("add", ".")
		require.NoError(t, err)

		_, err = git.ExecuteGit("commit", "-m", "Add file "+string(rune('a'+i)))
		require.NoError(t, err)
	}

	// Return to main repo directory
	err = os.Chdir(workDir)
	require.NoError(t, err)

	// Test listing worktrees with remote tracking information
	options := &ListOptions{Sort: SortByActivity}
	err = runListCommand(options)
	require.NoError(t, err)
}

func TestListCommandCurrentWorktreeDetection(t *testing.T) {
	// Create temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "grove-list-current-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save original directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize Grove repository structure
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a normal git repository first, then convert to Grove structure
	_, err = git.ExecuteGit("init")
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Now convert to Grove structure by moving .git to .bare
	err = os.Rename(filepath.Join(tempDir, ".git"), filepath.Join(tempDir, ".bare"))
	require.NoError(t, err)

	// Create .git file pointing to bare repo
	bareDir := filepath.Join(tempDir, ".bare")
	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	// Create multiple worktrees
	mainWorktreeDir := filepath.Join(tempDir, "main")
	_, err = git.ExecuteGit("worktree", "add", "-b", "main-worktree", mainWorktreeDir)
	require.NoError(t, err)

	featureWorktreeDir := filepath.Join(tempDir, "feature")
	_, err = git.ExecuteGit("worktree", "add", "-b", "feature", featureWorktreeDir)
	require.NoError(t, err)

	// Test from main worktree - should mark main as current
	err = os.Chdir(mainWorktreeDir)
	require.NoError(t, err)

	t.Run("current worktree detection from main", func(t *testing.T) {
		options := &ListOptions{Sort: SortByActivity}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	// Test from feature worktree - should mark feature as current
	err = os.Chdir(featureWorktreeDir)
	require.NoError(t, err)

	t.Run("current worktree detection from feature", func(t *testing.T) {
		options := &ListOptions{Sort: SortByActivity}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	// Test from bare repository directory - should not mark any as current
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	t.Run("current worktree detection from bare repo", func(t *testing.T) {
		options := &ListOptions{Sort: SortByActivity}
		err = runListCommand(options)
		require.NoError(t, err)
	})
}

func TestListCommandEmptyRepository(t *testing.T) {
	// Create temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "grove-list-empty-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save original directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize Grove repository structure but with no worktrees
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create bare repository
	bareDir := filepath.Join(tempDir, ".bare")
	err = git.InitBare(bareDir)
	require.NoError(t, err)

	// Create .git file pointing to bare repo
	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	// Test listing when no worktrees exist
	options := &ListOptions{Sort: SortByActivity}
	err = runListCommand(options)
	require.NoError(t, err)
}

func TestListCommandValidationErrors(t *testing.T) {
	// Create temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "grove-list-validation-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save original directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize Grove repository structure
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a normal git repository first, then convert to Grove structure
	_, err = git.ExecuteGit("init")
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Now convert to Grove structure by moving .git to .bare
	err = os.Rename(filepath.Join(tempDir, ".git"), filepath.Join(tempDir, ".bare"))
	require.NoError(t, err)

	// Create .git file pointing to bare repo
	bareDir := filepath.Join(tempDir, ".bare")
	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	t.Run("multiple filter options", func(t *testing.T) {
		options := &ListOptions{
			Sort:      SortByActivity,
			DirtyOnly: true,
			CleanOnly: true,
		}
		err = runListCommand(options)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Cannot specify multiple filters")
	})

	t.Run("invalid sort option", func(t *testing.T) {
		options := &ListOptions{
			Sort: "invalid",
		}
		err = runListCommand(options)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "Invalid sort option")
	})
}

func TestListCommandCornerCases(t *testing.T) {
	// Create temporary directory for the test repository
	tempDir, err := os.MkdirTemp("", "grove-list-corner-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Save original directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	// Initialize Grove repository structure
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a normal git repository first, then convert to Grove structure
	_, err = git.ExecuteGit("init")
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(tempDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Now convert to Grove structure by moving .git to .bare
	err = os.Rename(filepath.Join(tempDir, ".git"), filepath.Join(tempDir, ".bare"))
	require.NoError(t, err)

	// Create .git file pointing to bare repo
	bareDir := filepath.Join(tempDir, ".bare")
	err = git.CreateGitFile(tempDir, bareDir)
	require.NoError(t, err)

	// Create a worktree with detached HEAD
	detachedWorktreeDir := filepath.Join(tempDir, "detached")

	// Get the current commit hash
	output, err := git.ExecuteGit("rev-parse", "HEAD")
	require.NoError(t, err)
	commitHash := strings.TrimSpace(output)

	_, err = git.ExecuteGit("worktree", "add", "--detach", detachedWorktreeDir, commitHash)
	require.NoError(t, err)

	// Create a worktree with all types of file changes
	mixedWorktreeDir := filepath.Join(tempDir, "mixed")
	_, err = git.ExecuteGit("worktree", "add", "-b", "mixed", mixedWorktreeDir)
	require.NoError(t, err)

	err = os.Chdir(mixedWorktreeDir)
	require.NoError(t, err)

	// Create modified file
	err = os.WriteFile(filepath.Join(mixedWorktreeDir, "README.md"), []byte("# Modified\n"), 0o644)
	require.NoError(t, err)

	// Create staged file
	stagedFile := filepath.Join(mixedWorktreeDir, "staged.txt")
	err = os.WriteFile(stagedFile, []byte("staged content"), 0o644)
	require.NoError(t, err)
	_, err = git.ExecuteGit("add", "staged.txt")
	require.NoError(t, err)

	// Create untracked file
	untrackedFile := filepath.Join(mixedWorktreeDir, "untracked.tmp")
	err = os.WriteFile(untrackedFile, []byte("untracked"), 0o644)
	require.NoError(t, err)

	// Return to main repo directory
	err = os.Chdir(tempDir)
	require.NoError(t, err)

	t.Run("list with detached HEAD", func(t *testing.T) {
		options := &ListOptions{Sort: SortByActivity}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("list with mixed file states", func(t *testing.T) {
		options := &ListOptions{Sort: SortByStatus}
		err = runListCommand(options)
		require.NoError(t, err)
	})

	t.Run("filter only dirty with mixed states", func(t *testing.T) {
		options := &ListOptions{
			Sort:      SortByStatus,
			DirtyOnly: true,
		}
		err = runListCommand(options)
		require.NoError(t, err)
	})
}

func TestListCommandNewListCommand(t *testing.T) {
	// Test the command integration
	cmd := NewListCmd()

	// Verify the command can be created and has the right properties
	assert.Equal(t, "list", cmd.Use)
	assert.NotNil(t, cmd)

	// Verify the cobra command has the expected structure
	assert.Equal(t, "list", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Test that flags are properly configured
	flags := cmd.Flags()
	assert.NotNil(t, flags.Lookup("sort"))
	assert.NotNil(t, flags.Lookup("verbose"))
	assert.NotNil(t, flags.Lookup("porcelain"))
	assert.NotNil(t, flags.Lookup("dirty"))
	assert.NotNil(t, flags.Lookup("stale"))
	assert.NotNil(t, flags.Lookup("clean"))
	assert.NotNil(t, flags.Lookup("days"))
}
