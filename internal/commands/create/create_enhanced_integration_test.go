//go:build integration
// +build integration

package create

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRepositoryForCreate(t *testing.T, helper *testutils.IntegrationTestHelper) string {
	t.Helper()

	// REPOSITORY SETUP STRATEGY: Create a realistic git repository structure for integration testing
	// This setup creates a bare repository with git worktrees, simulating real-world grove usage:
	// 1. Initialize bare repository (.bare directory) - central repository storage
	// 2. Create .git file pointing to bare repo - enables worktree functionality
	// 3. Add initial commit with README - establishes history for branch operations
	// 4. Create main worktree - provides base for branch creation and file operations
	repoDir := helper.CreateTempDir("grove-create-enhanced-test")

	// Initialize bare repository
	bareDir := filepath.Join(repoDir, ".bare")
	err := git.InitBare(bareDir)
	require.NoError(t, err)

	err = git.CreateGitFile(repoDir, bareDir)
	require.NoError(t, err)

	// Create initial commit by first creating a temporary working directory
	tempWorkDir := filepath.Join(repoDir, "temp-work")
	err = os.MkdirAll(tempWorkDir, 0o755)
	require.NoError(t, err)

	// Change to temp working directory for git operations
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	err = os.Chdir(tempWorkDir)
	require.NoError(t, err)
	defer func() {
		_ = os.Chdir(originalDir)
		_ = os.RemoveAll(tempWorkDir)
	}()

	// Initialize working tree in temp directory
	_, err = git.ExecuteGit("init")
	require.NoError(t, err)

	// Configure git user for commits
	_, err = git.ExecuteGit("config", "user.name", "Test User")
	require.NoError(t, err)
	_, err = git.ExecuteGit("config", "user.email", "test@example.com")
	require.NoError(t, err)

	// Create initial commit
	readmeFile := filepath.Join(tempWorkDir, "README.md")
	err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "README.md")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Initial commit")
	require.NoError(t, err)

	// Push to bare repository
	_, err = git.ExecuteGit("remote", "add", "origin", bareDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("push", "origin", "main")
	require.NoError(t, err)

	// Go back to repo directory
	err = os.Chdir(repoDir)
	require.NoError(t, err)

	// Create main worktree
	mainWorktreeDir := filepath.Join(repoDir, "main")
	_, err = git.ExecuteGit("worktree", "add", mainWorktreeDir, "main")
	require.NoError(t, err)

	return repoDir
}

func TestCreateCommand_BaseBranch_Integration(t *testing.T) {
	helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
	repoDir := setupTestRepositoryForCreate(t, helper)

	runner := testutils.NewTestRunner(t)
	runner.WithIsolatedWorkingDir().Run(func() {
		// Change to the main worktree directory, not the bare repo root
		mainWorktreeDir := filepath.Join(repoDir, "main")
		err := os.Chdir(mainWorktreeDir)
		require.NoError(t, err)

		// Create a development branch with some commits
		devBranch := generateUniqueBranchNameEnhanced("develop")
		_, err = git.ExecuteGit("checkout", "-b", devBranch)
		require.NoError(t, err)

		// Add a file to dev branch
		devFile := "dev-feature.txt"
		err = os.WriteFile(devFile, []byte("development feature"), 0o644)
		require.NoError(t, err)

		_, err = git.ExecuteGit("add", "dev-feature.txt")
		require.NoError(t, err)

		_, err = git.ExecuteGit("commit", "-m", "Add dev feature")
		require.NoError(t, err)

		tests := []struct {
			name       string
			baseBranch string
			newBranch  string
			expectFile string // File that should exist if base branch is correct
		}{
			{
				name:       "create from main branch",
				baseBranch: "main",
				newBranch:  generateUniqueBranchNameEnhanced("feature-from-main"),
				expectFile: "README.md", // Should have main's content
			},
			{
				name:       "create from develop branch",
				baseBranch: devBranch,
				newBranch:  generateUniqueBranchNameEnhanced("feature-from-dev"),
				expectFile: "dev-feature.txt", // Should have dev's content
			},
			{
				name:       "create without base (current branch)",
				baseBranch: "", // Empty means current branch
				newBranch:  generateUniqueBranchNameEnhanced("feature-from-current"),
				expectFile: "dev-feature.txt", // Should use current (dev) branch
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cmd := NewCreateCmd()
				if tt.baseBranch != "" {
					cmd.SetArgs([]string{tt.newBranch, "--base", tt.baseBranch})
				} else {
					cmd.SetArgs([]string{tt.newBranch})
				}

				err := cmd.Execute()
				require.NoError(t, err)

				// Verify worktree was created
				worktreePath := filepath.Join(repoDir, tt.newBranch)
				assert.DirExists(t, worktreePath)

				// Verify expected file exists
				expectedFile := filepath.Join(worktreePath, tt.expectFile)
				assert.FileExists(t, expectedFile)

				// Clean up worktree for next test
				_, err = git.ExecuteGit("worktree", "remove", worktreePath)
				require.NoError(t, err)
			})
		}
	})
}

func TestCreateCommand_FileCopying_Integration(t *testing.T) {
	helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
	repoDir := setupTestRepositoryForCreate(t, helper)

	runner := testutils.NewTestRunner(t)
	runner.WithIsolatedWorkingDir().Run(func() {
		// Change to the main worktree directory, not the bare repo root
		mainWorktreeDir := filepath.Join(repoDir, "main")
		err := os.Chdir(mainWorktreeDir)
		require.NoError(t, err)

		// Set up base branch with various files
		_, err = git.ExecuteGit("checkout", "main")
		require.NoError(t, err)

		// Configure git user for commits
		_, err = git.ExecuteGit("config", "user.name", "Test User")
		require.NoError(t, err)
		_, err = git.ExecuteGit("config", "user.email", "test@example.com")
		require.NoError(t, err)

		// Create gitignore first to prevent environment files from being committed
		gitignoreContent := `
.env*
!.env.example
docker-compose.override.yml
.vscode/
.idea/
*.local
`
		err = os.WriteFile(".gitignore", []byte(gitignoreContent), 0o644)
		require.NoError(t, err)

		_, err = git.ExecuteGit("add", ".gitignore")
		require.NoError(t, err)

		_, err = git.ExecuteGit("commit", "-m", "Add gitignore")
		require.NoError(t, err)

		// Create files that should be copied (these won't be committed due to gitignore)
		files := map[string]string{
			".env":                        "NODE_ENV=development",
			".env.local":                  "LOCAL_VAR=local_value",
			".env.example":                "NODE_ENV=production", // This will be committed (not in gitignore)
			"docker-compose.override.yml": "version: '3'\nservices:\n  app:\n    volumes:\n      - .:/app",
			".vscode/settings.json":       `{"editor.tabSize": 2}`,
			".idea/workspace.xml":         `<workspace></workspace>`,
			"regular-file.txt":            "regular content", // This will be committed
			".gitignore.local":            "*.local",
		}

		for filename, content := range files {
			// Create files in the current working directory (main worktree)
			err = os.MkdirAll(filepath.Dir(filename), 0o755)
			require.NoError(t, err)
			err = os.WriteFile(filename, []byte(content), 0o644)
			require.NoError(t, err)
		}

		// Only commit files that aren't gitignored
		_, err = git.ExecuteGit("add", ".env.example", "regular-file.txt")
		require.NoError(t, err)

		_, err = git.ExecuteGit("commit", "-m", "Add test files")
		require.NoError(t, err)

		tests := []struct {
			name           string
			args           []string
			expectFiles    []string
			expectNotFiles []string
		}{
			{
				name: "copy-env flag",
				args: []string{generateUniqueBranchNameEnhanced("test-copy-env"), "--copy-env"},
				expectFiles: []string{
					".env",
					".env.local",   // Matches *.local.*
					".env.example", // This is committed
					"docker-compose.override.yml",
					"regular-file.txt", // This is committed
					"README.md",        // This is from initial commit
					".gitignore.local", // Matches *.local.*
				},
				expectNotFiles: []string{
					".vscode/settings.json",
					".idea/workspace.xml",
				},
			},
			{
				name: "custom copy patterns",
				args: []string{generateUniqueBranchNameEnhanced("test-copy-custom"), "--copy", ".vscode/,.idea/,.env*"},
				expectFiles: []string{
					".vscode/settings.json",
					".idea/workspace.xml",
					".env",
					".env.local",
					".env.example",     // This is committed
					"regular-file.txt", // This is committed
					"README.md",        // This is from initial commit
				},
				expectNotFiles: []string{
					"docker-compose.override.yml",
					".gitignore.local",
				},
			},
			{
				name: "no-copy flag",
				args: []string{generateUniqueBranchNameEnhanced("test-no-copy"), "--no-copy"},
				expectFiles: []string{
					"README.md",        // This is from initial commit
					".env.example",     // This is committed
					"regular-file.txt", // This is committed
				},
				expectNotFiles: []string{
					".env",
					".env.local",
					".vscode/settings.json",
					"docker-compose.override.yml",
					".gitignore.local",
				},
			},
			{
				name: "default behavior (no copying)",
				args: []string{generateUniqueBranchNameEnhanced("test-default")},
				expectFiles: []string{
					"README.md",        // This is from initial commit
					".env.example",     // This is committed
					"regular-file.txt", // This is committed
				},
				expectNotFiles: []string{
					".env",
					".env.local",
					".vscode/settings.json",
					"docker-compose.override.yml",
					".gitignore.local",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cmd := NewCreateCmd()
				cmd.SetArgs(tt.args)

				err := cmd.Execute()
				require.NoError(t, err)

				branchName := tt.args[0]
				worktreePath := filepath.Join(repoDir, branchName)
				assert.DirExists(t, worktreePath)

				// Check expected files exist
				for _, expectedFile := range tt.expectFiles {
					filePath := filepath.Join(worktreePath, expectedFile)
					assert.FileExists(t, filePath, "Expected file %s to exist", expectedFile)
				}

				// Check files that shouldn't exist
				for _, notExpectedFile := range tt.expectNotFiles {
					filePath := filepath.Join(worktreePath, notExpectedFile)
					assert.NoFileExists(t, filePath, "Expected file %s to NOT exist", notExpectedFile)
				}

				// Clean up worktree
				_, err = git.ExecuteGit("worktree", "remove", worktreePath)
				require.NoError(t, err)
			})
		}
	})
}

func TestCreateCommand_URLSupport_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping URL tests in short mode")
	}

	// These tests would require mock HTTP servers or network access
	// For now, test the URL parsing and validation logic

	tests := []struct {
		name        string
		url         string
		expectError bool
		skipReason  string
	}{
		{
			name:       "GitHub repo URL",
			url:        "https://github.com/git/git",
			skipReason: "Requires network access",
		},
		{
			name:       "GitHub branch URL",
			url:        "https://github.com/git/git/tree/main",
			skipReason: "Requires network access",
		},
		{
			name:       "GitHub PR URL",
			url:        "https://github.com/git/git/pull/123",
			skipReason: "Requires network access",
		},
		{
			name:       "GitLab MR URL",
			url:        "https://gitlab.com/gitlab-org/gitlab/-/merge_requests/456",
			skipReason: "Requires network access",
		},
		{
			name:       "Bitbucket PR URL",
			url:        "https://bitbucket.org/owner/repo/pull-requests/789",
			skipReason: "Requires network access",
		},
		{
			name:        "Invalid URL",
			url:         "not-a-valid-url",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" {
				t.Skip(tt.skipReason)
			}

			helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
			tempDir := helper.CreateTempDir("grove-create-url-test")

			runner := testutils.NewTestRunner(t)
			runner.WithIsolatedWorkingDir().Run(func() {
				err := os.Chdir(tempDir)
				require.NoError(t, err)

				cmd := NewCreateCmd()
				cmd.SetArgs([]string{tt.url})

				err = cmd.Execute()
				if tt.expectError {
					assert.Error(t, err)
				} else {
					// For URL tests, we can't easily test without network
					t.Skip("URL functionality requires network access")
				}
			})
		})
	}
}

func TestCreateCommand_RemoteBranches_Integration(t *testing.T) {
	helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
	repoDir := setupTestRepositoryForCreate(t, helper)

	runner := testutils.NewTestRunner(t)
	runner.WithIsolatedWorkingDir().Run(func() {
		// Change to the main worktree directory, not the bare repo root
		mainWorktreeDir := filepath.Join(repoDir, "main")
		err := os.Chdir(mainWorktreeDir)
		require.NoError(t, err)

		// Test that create command can handle remote-style branch names
		t.Run("remote branch name handling", func(t *testing.T) {
			// Test that create command can handle branch names that look like remote refs
			cmd := NewCreateCmd()

			branchName := "origin-style-branch"
			cmd.SetArgs([]string{branchName})

			err := cmd.Execute()
			assert.NoError(t, err) // Should succeed by creating a local branch

			// Verify worktree was created
			worktreePath := filepath.Join(repoDir, branchName)
			assert.DirExists(t, worktreePath)

			// Clean up
			_, err = git.ExecuteGit("worktree", "remove", worktreePath)
			require.NoError(t, err)
		})
	})
}

func TestCreateCommand_ConflictResolution_Integration(t *testing.T) {
	helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
	repoDir := setupTestRepositoryForCreate(t, helper)

	runner := testutils.NewTestRunner(t)
	runner.WithIsolatedWorkingDir().Run(func() {
		// Change to the main worktree directory, not the bare repo root
		mainWorktreeDir := filepath.Join(repoDir, "main")
		err := os.Chdir(mainWorktreeDir)
		require.NoError(t, err)

		branchName := generateUniqueBranchNameEnhanced("conflict-test")

		// Create the branch first
		_, err = git.ExecuteGit("checkout", "-b", branchName)
		require.NoError(t, err)

		_, err = git.ExecuteGit("checkout", "main")
		require.NoError(t, err)

		t.Run("existing branch creates worktree", func(t *testing.T) {
			cmd := NewCreateCmd()
			cmd.SetArgs([]string{branchName})

			err := cmd.Execute()
			require.NoError(t, err)

			// Should create worktree for existing branch
			worktreePath := filepath.Join(repoDir, branchName)
			assert.DirExists(t, worktreePath)

			// Clean up
			_, err = git.ExecuteGit("worktree", "remove", worktreePath)
			require.NoError(t, err)
		})

		t.Run("existing worktree directory", func(t *testing.T) {
			// Create directory that would conflict
			conflictDir := filepath.Join(repoDir, branchName)
			err := os.MkdirAll(conflictDir, 0o755)
			require.NoError(t, err)

			// Add a file to make it non-empty
			conflictFile := filepath.Join(conflictDir, "conflict.txt")
			err = os.WriteFile(conflictFile, []byte("conflict"), 0o644)
			require.NoError(t, err)

			cmd := NewCreateCmd()
			cmd.SetArgs([]string{branchName})

			err = cmd.Execute()
			// Should handle the conflict gracefully
			if err != nil {
				assert.Contains(t, err.Error(), "already exists")
			}

			// Clean up
			os.RemoveAll(conflictDir)
		})
	})
}

func TestCreateCommand_CustomPath_Integration(t *testing.T) {
	helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
	repoDir := setupTestRepositoryForCreate(t, helper)

	tests := []struct {
		name       string
		branchName string
		customPath string
	}{
		{
			name:       "custom relative path",
			branchName: generateUniqueBranchNameEnhanced("custom-rel"),
			customPath: "custom-dir",
		},
		{
			name:       "custom nested path",
			branchName: generateUniqueBranchNameEnhanced("custom-nested"),
			customPath: "level1/level2/custom",
		},
		{
			name:       "custom path with special chars",
			branchName: generateUniqueBranchNameEnhanced("custom-special"),
			customPath: "custom-dir_with-chars",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := testutils.NewTestRunner(t)
			runner.WithIsolatedWorkingDir().Run(func() {
				// Change to the test repository directory
				err := os.Chdir(repoDir)
				require.NoError(t, err)

				cmd := NewCreateCmd()
				cmd.SetArgs([]string{tt.branchName, tt.customPath})

				err = cmd.Execute()
				require.NoError(t, err)

				// Verify worktree was created at custom path
				customWorktreePath := filepath.Join(repoDir, tt.customPath)
				assert.DirExists(t, customWorktreePath)

				// Verify it's a valid git worktree
				gitDir := filepath.Join(customWorktreePath, ".git")
				assert.FileExists(t, gitDir)

				// Clean up
				_, err = git.ExecuteGit("worktree", "remove", customWorktreePath)
				require.NoError(t, err)
			})
		})
	}
}

func TestCreateCommand_FlagValidation_Integration(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorText   string
	}{
		{
			name:        "no-copy with copy-env",
			args:        []string{"test-branch", "--no-copy", "--copy-env"},
			expectError: true,
			errorText:   "--no-copy cannot be used with --copy-env or --copy flags",
		},
		{
			name:        "no-copy with copy patterns",
			args:        []string{"test-branch", "--no-copy", "--copy", ".env*"},
			expectError: true,
			errorText:   "--no-copy cannot be used with --copy-env or --copy flags",
		},
		{
			name:        "copy-env with copy patterns",
			args:        []string{"test-branch", "--copy-env", "--copy", ".env*"},
			expectError: true,
			errorText:   "cannot use both --copy-env and --copy",
		},
		{
			name:        "all three copy flags",
			args:        []string{"test-branch", "--no-copy", "--copy-env", "--copy", ".env*"},
			expectError: true,
			// Should fail on first conflict
		},
		{
			name:        "valid single flag",
			args:        []string{generateUniqueBranchNameEnhanced("valid-single"), "--copy-env"},
			expectError: false,
		},
		{
			name:        "no copy flags (valid)",
			args:        []string{generateUniqueBranchNameEnhanced("valid-none")},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCreateCmd()
			cmd.SetArgs(tt.args)

			if tt.expectError {
				err := cmd.Execute()
				assert.Error(t, err)
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText)
				}
			} else {
				helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
				repoDir := setupTestRepositoryForCreate(t, helper)

				err := cmd.Execute()
				assert.NoError(t, err)
				// Clean up if successful
				branchName := tt.args[0]
				worktreePath := filepath.Join(repoDir, branchName)
				if _, err := os.Stat(worktreePath); err == nil {
					_, _ = git.ExecuteGit("worktree", "remove", worktreePath)
				}
			}
		})
	}
}

func TestCreateCommand_ProgressCallback_Integration(t *testing.T) {
	helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
	repoDir := setupTestRepositoryForCreate(t, helper)

	runner := testutils.NewTestRunner(t)
	runner.WithIsolatedWorkingDir().Run(func() {
		// Change to the main worktree directory, not the bare repo root
		mainWorktreeDir := filepath.Join(repoDir, "main")
		err := os.Chdir(mainWorktreeDir)
		require.NoError(t, err)

		// Test that progress callback works without panicking
		branchName := generateUniqueBranchNameEnhanced("progress-test")

		cmd := NewCreateCmd()
		cmd.SetArgs([]string{branchName})

		// Execute should work with progress indicators
		err = cmd.Execute()
		require.NoError(t, err)

		worktreePath := filepath.Join(repoDir, branchName)
		assert.DirExists(t, worktreePath)

		// Clean up
		_, err = git.ExecuteGit("worktree", "remove", worktreePath)
		require.NoError(t, err)
	})
}

func TestCreateCommand_ArgumentValidation_Integration(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorText   string
	}{
		{
			name:        "no arguments",
			args:        []string{},
			expectError: true,
			errorText:   "branch name, URL, or remote branch is required",
		},
		{
			name:        "too many arguments",
			args:        []string{"branch", "path", "extra"},
			expectError: true,
			errorText:   "too many arguments, expected: grove create [branch-name|url] [path]",
		},
		{
			name:        "valid single argument",
			args:        []string{generateUniqueBranchNameEnhanced("valid-single")},
			expectError: false,
		},
		{
			name:        "valid two arguments",
			args:        []string{generateUniqueBranchNameEnhanced("valid-two"), "custom-path"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewCreateCmd()
			cmd.SetArgs(tt.args)

			if tt.expectError {
				err := cmd.Execute()
				assert.Error(t, err)
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText)
				}
			} else {
				helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()
				repoDir := setupTestRepositoryForCreate(t, helper)

				runner := testutils.NewTestRunner(t)
				runner.WithIsolatedWorkingDir().Run(func() {
					// Change to the test repository directory
					err := os.Chdir(repoDir)
					require.NoError(t, err)

					err = cmd.Execute()
					assert.NoError(t, err)

					// Clean up
					branchName := tt.args[0]
					var worktreePath string
					if len(tt.args) > 1 {
						worktreePath = filepath.Join(repoDir, tt.args[1])
					} else {
						worktreePath = filepath.Join(repoDir, branchName)
					}

					if _, err := os.Stat(worktreePath); err == nil {
						_, _ = git.ExecuteGit("worktree", "remove", worktreePath)
					}
				})
			}
		})
	}
}

// generateUniqueBranchNameEnhanced creates a unique branch name using timestamp and random suffix
func generateUniqueBranchNameEnhanced(prefix string) string {
	timestamp := time.Now().UnixNano() / int64(time.Millisecond)
	random, _ := rand.Int(rand.Reader, big.NewInt(10000))
	return fmt.Sprintf("%s-%d-%d", prefix, timestamp, random.Int64())
}
