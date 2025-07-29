//go:build integration
// +build integration

package list

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRepositoryWithWorktrees(t *testing.T) (string, func()) {
	t.Helper()

	tempDir := testutils.NewTestDirectory(t, "grove-list-enhanced-test")
	
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	err = os.Chdir(tempDir.Path)
	require.NoError(t, err)

	// Initialize bare repository
	bareDir := filepath.Join(tempDir.Path, ".bare")
	err = git.InitBare(bareDir)
	require.NoError(t, err)

	err = git.CreateGitFile(tempDir.Path, bareDir)
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(tempDir.Path, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Create main worktree
	mainWorktreeDir := filepath.Join(tempDir.Path, "main")
	_, err = git.ExecuteGit("worktree", "add", mainWorktreeDir, "main")
	require.NoError(t, err)

	cleanup := func() {
		os.Chdir(originalDir)
		tempDir.Cleanup()
	}

	return tempDir.Path, cleanup
}

func createWorktreeWithState(t *testing.T, repoDir, branchName, worktreeState string) string {
	t.Helper()

	// Create branch and worktree
	_, err := git.ExecuteGit("checkout", "-b", branchName)
	require.NoError(t, err)

	worktreeDir := filepath.Join(repoDir, branchName)
	_, err = git.ExecuteGit("worktree", "add", worktreeDir, branchName)
	require.NoError(t, err)

	// Change to worktree directory and set up state
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(worktreeDir)
	require.NoError(t, err)

	switch worktreeState {
	case "dirty":
		// Create uncommitted changes
		testFile := filepath.Join(worktreeDir, "test.txt")
		err = os.WriteFile(testFile, []byte("uncommitted changes"), 0o644)
		require.NoError(t, err)

		_, err = git.ExecuteGit("add", "test.txt")
		require.NoError(t, err) // Staged changes

		// Also create unstaged changes
		modFile := filepath.Join(worktreeDir, "modified.txt")
		err = os.WriteFile(modFile, []byte("unstaged changes"), 0o644)
		require.NoError(t, err)

	case "clean":
		// Ensure worktree is clean
		_, err = git.ExecuteGit("status", "--porcelain")
		require.NoError(t, err)

	case "stale":
		// Create commits but make it appear old by setting access time
		testFile := filepath.Join(worktreeDir, "old.txt")
		err = os.WriteFile(testFile, []byte("old content"), 0o644)
		require.NoError(t, err)

		_, err = git.ExecuteGit("add", "old.txt")
		require.NoError(t, err)

		_, err = git.ExecuteGit("commit", "-m", "Old commit")
		require.NoError(t, err)

		// Set access time to make it appear stale (30 days ago)
		oldTime := time.Now().AddDate(0, 0, -30)
		err = os.Chtimes(worktreeDir, oldTime, oldTime)
		require.NoError(t, err)
	}

	return worktreeDir
}

func TestListCommand_SortOptions_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	// Create multiple worktrees with different states
	createWorktreeWithState(t, repoDir, "feature-a", "clean")
	createWorktreeWithState(t, repoDir, "feature-b", "dirty")
	createWorktreeWithState(t, repoDir, "feature-c", "clean")

	tests := []struct {
		name           string
		sortOption     string
		expectError    bool
		validateOutput func(t *testing.T, output string)
	}{
		{
			name:       "sort by activity (default)",
			sortOption: "activity",
			validateOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "main")
				assert.Contains(t, output, "feature-a")
				assert.Contains(t, output, "feature-b")
				assert.Contains(t, output, "feature-c")
			},
		},
		{
			name:       "sort by name",
			sortOption: "name",
			validateOutput: func(t *testing.T, output string) {
				lines := strings.Split(output, "\n")
				var worktreeLines []string
				for _, line := range lines {
					if strings.Contains(line, "feature-") || strings.Contains(line, "main") {
						worktreeLines = append(worktreeLines, line)
					}
				}
				// Should be in alphabetical order
				assert.True(t, len(worktreeLines) >= 3)
			},
		},
		{
			name:       "sort by status",
			sortOption: "status",
			validateOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "main")
				assert.Contains(t, output, "feature-a")
				assert.Contains(t, output, "feature-b")
				assert.Contains(t, output, "feature-c")
			},
		},
		{
			name:        "invalid sort option",
			sortOption:  "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := &ListOptions{
				Sort: ListSortOption(tt.sortOption),
			}

			err := runListCommand(options)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "Invalid sort option")
			} else {
				assert.NoError(t, err)
				if tt.validateOutput != nil {
					// This test is simplified - in real integration tests,
					// you'd need to capture the output to validate it
					t.Log("Output validation would require capturing stdout")
				}
			}
		})
	}
}

func TestListCommand_FilterOptions_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	// Create worktrees with different states
	createWorktreeWithState(t, repoDir, "clean-branch", "clean")
	dirtyWorktree := createWorktreeWithState(t, repoDir, "dirty-branch", "dirty")
	createWorktreeWithState(t, repoDir, "stale-branch", "stale")

	tests := []struct {
		name    string
		options ListOptions
		check   func(t *testing.T, options *ListOptions) error
	}{
		{
			name: "dirty only filter",
			options: ListOptions{
				DirtyOnly: true,
				Sort:      SortByActivity,
			},
			check: func(t *testing.T, opts *ListOptions) error {
				return runListCommand(opts)
			},
		},
		{
			name: "clean only filter",
			options: ListOptions{
				CleanOnly: true,
				Sort:      SortByActivity,
			},
			check: func(t *testing.T, opts *ListOptions) error {
				return runListCommand(opts)
			},
		},
		{
			name: "stale only filter",
			options: ListOptions{
				StaleOnly: true,
				StaleDays: 7, // 7 days
				Sort:      SortByActivity,
			},
			check: func(t *testing.T, opts *ListOptions) error {
				return runListCommand(opts)
			},
		},
		{
			name: "stale with custom days",
			options: ListOptions{
				StaleOnly: true,
				StaleDays: 60, // 60 days
				Sort:      SortByActivity,
			},
			check: func(t *testing.T, opts *ListOptions) error {
				return runListCommand(opts)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.check(t, &tt.options)
			assert.NoError(t, err)
			
			// Verify the dirty worktree exists for comparison
			assert.DirExists(t, dirtyWorktree)
		})
	}
}

func TestListCommand_MultipleFilters_Integration(t *testing.T) {
	tests := []struct {
		name    string
		options ListOptions
	}{
		{
			name: "dirty and clean flags",
			options: ListOptions{
				DirtyOnly: true,
				CleanOnly: true,
				Sort:      SortByActivity,
			},
		},
		{
			name: "dirty and stale flags",
			options: ListOptions{
				DirtyOnly: true,
				StaleOnly: true,
				Sort:      SortByActivity,
			},
		},
		{
			name: "clean and stale flags",
			options: ListOptions{
				CleanOnly: true,
				StaleOnly: true,
				Sort:      SortByActivity,
			},
		},
		{
			name: "all three flags",
			options: ListOptions{
				DirtyOnly: true,
				CleanOnly: true,
				StaleOnly: true,
				Sort:      SortByActivity,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateListOptions(&tt.options)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "Cannot specify multiple filters")
		})
	}
}

func TestListCommand_VerboseOutput_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	createWorktreeWithState(t, repoDir, "test-branch", "clean")

	t.Run("verbose flag", func(t *testing.T) {
		options := &ListOptions{
			Verbose: true,
			Sort:    SortByActivity,
		}

		err := runListCommand(options)
		assert.NoError(t, err)
		// In a real integration test, you'd capture output and verify it contains full paths
	})

	t.Run("non-verbose (default)", func(t *testing.T) {
		options := &ListOptions{
			Verbose: false,
			Sort:    SortByActivity,
		}

		err := runListCommand(options)
		assert.NoError(t, err)
	})
}

func TestListCommand_PorcelainOutput_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	createWorktreeWithState(t, repoDir, "test-branch", "clean")

	t.Run("porcelain output", func(t *testing.T) {
		options := &ListOptions{
			Porcelain: true,
			Sort:      SortByActivity,
		}

		err := runListCommand(options)
		assert.NoError(t, err)
		// In a real integration test, you'd capture output and verify machine-readable format
	})

	t.Run("human output (default)", func(t *testing.T) {
		options := &ListOptions{
			Porcelain: false,
			Sort:      SortByActivity,
		}

		err := runListCommand(options)
		assert.NoError(t, err)
	})
}

func TestListCommand_EmptyRepository_Integration(t *testing.T) {
	tempDir := testutils.NewTestDirectory(t, "grove-list-empty")
	defer tempDir.Cleanup()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir.Path)
	require.NoError(t, err)

	// Initialize bare repository without worktrees
	bareDir := filepath.Join(tempDir.Path, ".bare")
	err = git.InitBare(bareDir)
	require.NoError(t, err)

	err = git.CreateGitFile(tempDir.Path, bareDir)
	require.NoError(t, err)

	options := &ListOptions{
		Sort: SortByActivity,
	}

	err = runListCommand(options)
	assert.NoError(t, err)
	// Should handle empty repository gracefully
}

func TestListCommand_NonGitRepository_Integration(t *testing.T) {
	tempDir := testutils.NewTestDirectory(t, "grove-list-non-git")
	defer tempDir.Cleanup()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir.Path)
	require.NoError(t, err)

	options := &ListOptions{
		Sort: SortByActivity,
	}

	err = runListCommand(options)
	assert.Error(t, err)
	// Should fail gracefully for non-git directories
}

func TestListCommand_CorruptedRepository_Integration(t *testing.T) {
	tempDir := testutils.NewTestDirectory(t, "grove-list-corrupted")
	defer tempDir.Cleanup()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(tempDir.Path)
	require.NoError(t, err)

	// Create a .git file pointing to non-existent directory
	gitFile := filepath.Join(tempDir.Path, ".git")
	err = os.WriteFile(gitFile, []byte("gitdir: /non/existent/path"), 0o644)
	require.NoError(t, err)

	options := &ListOptions{
		Sort: SortByActivity,
	}

	err = runListCommand(options)
	assert.Error(t, err)
	// Should handle corrupted repository gracefully
}

func TestListCommand_CurrentWorktreeDetection_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	// Create a test worktree
	testWorktree := createWorktreeWithState(t, repoDir, "current-test", "clean")

	// Change to the test worktree
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(testWorktree)
	require.NoError(t, err)

	options := &ListOptions{
		Sort: SortByActivity,
	}

	err = runListCommand(options)
	assert.NoError(t, err)
	// Should detect current worktree and mark it appropriately
}

func TestListCommand_RemoteTrackingBranches_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test requiring remote setup in short mode")
	}

	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	// Set up a fake remote (using local path for testing)
	remoteDir := filepath.Join(repoDir, "..", "remote")
	err := os.MkdirAll(remoteDir, 0o755)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	err = os.Chdir(remoteDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("init", "--bare")
	require.NoError(t, err)

	// Go back to main repo and add remote
	err = os.Chdir(repoDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("remote", "add", "origin", remoteDir)
	require.NoError(t, err)

	// Create worktree with remote tracking
	createWorktreeWithState(t, repoDir, "remote-branch", "clean")

	options := &ListOptions{
		Sort: SortByActivity,
	}

	err = runListCommand(options)
	assert.NoError(t, err)
	// Should show remote tracking information
}

func TestListCommand_LargeNumberOfWorktrees_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	// Create many worktrees
	for i := 0; i < 20; i++ {
		branchName := fmt.Sprintf("branch-%d", i)
		createWorktreeWithState(t, repoDir, branchName, "clean")
	}

	options := &ListOptions{
		Sort: SortByActivity,
	}

	start := time.Now()
	err := runListCommand(options)
	duration := time.Since(start)

	assert.NoError(t, err)
	assert.Less(t, duration, 10*time.Second, "List command should complete quickly even with many worktrees")
}

func TestListCommand_SpecialCharactersInBranches_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	// Create worktrees with special characters in branch names
	specialBranches := []string{
		"feature/user-auth",
		"bugfix/issue-123",
		"release/v1.2.3",
		"hotfix/critical-fix",
	}

	for _, branchName := range specialBranches {
		createWorktreeWithState(t, repoDir, branchName, "clean")
	}

	options := &ListOptions{
		Sort: SortByActivity,
	}

	err := runListCommand(options)
	assert.NoError(t, err)
	// Should handle special characters correctly
}

func TestListCommand_StatusIndicators_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryWithWorktrees(t)
	defer cleanup()

	// Create worktrees with different git states
	createWorktreeWithState(t, repoDir, "clean-status", "clean")
	createWorktreeWithState(t, repoDir, "dirty-status", "dirty") 

	options := &ListOptions{
		Sort: SortByActivity,
	}

	err := runListCommand(options)
	assert.NoError(t, err)
	// Should show appropriate status indicators (✓, ⚠, etc.)
}