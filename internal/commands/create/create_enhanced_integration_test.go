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

func setupTestRepositoryForCreate(t *testing.T) (string, func()) {
	t.Helper()

	tempDir := testutils.NewTestDirectory(t, "grove-create-enhanced-test")
	
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

func TestCreateCommand_BaseBranch_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryForCreate(t)
	defer cleanup()

	// Create a development branch with some commits
	devBranch := generateUniqueBranchNameEnhanced("develop")
	_, err := git.ExecuteGit("checkout", "-b", devBranch)
	require.NoError(t, err)

	// Add a file to dev branch
	devFile := filepath.Join(repoDir, "dev-feature.txt")
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
}

func TestCreateCommand_FileCopying_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryForCreate(t)
	defer cleanup()

	// Set up base branch with various files
	_, err := git.ExecuteGit("checkout", "main")
	require.NoError(t, err)

	// Create files that should be copied
	files := map[string]string{
		".env":                    "NODE_ENV=development",
		".env.local":             "LOCAL_VAR=local_value",
		".env.example":           "NODE_ENV=production",
		"docker-compose.override.yml": "version: '3'\nservices:\n  app:\n    volumes:\n      - .:/app",
		".vscode/settings.json":  `{"editor.tabSize": 2}`,
		".idea/workspace.xml":    `<workspace></workspace>`,
		"regular-file.txt":       "regular content",
		".gitignore.local":       "*.local",
	}

	for filename, content := range files {
		filePath := filepath.Join(repoDir, filename)
		err = os.MkdirAll(filepath.Dir(filePath), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)
	}

	// Commit the files
	_, err = git.ExecuteGit("add", ".")
	require.NoError(t, err)
	_, err = git.ExecuteGit("commit", "-m", "Add test files")
	require.NoError(t, err)

	tests := []struct {
		name         string
		args         []string
		expectFiles  []string
		expectNotFiles []string
	}{
		{
			name: "copy-env flag",
			args: []string{generateUniqueBranchNameEnhanced("test-copy-env"), "--copy-env"},
			expectFiles: []string{
				".env",
				".env.local", 
				".env.example",
				"docker-compose.override.yml",
			},
			expectNotFiles: []string{
				".vscode/settings.json",
				".idea/workspace.xml",
				"regular-file.txt",
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
				".env.example",
			},
			expectNotFiles: []string{
				"docker-compose.override.yml",
				"regular-file.txt",
			},
		},
		{
			name: "no-copy flag",
			args: []string{generateUniqueBranchNameEnhanced("test-no-copy"), "--no-copy"},
			expectFiles: []string{
				"README.md", // This should always exist from git
			},
			expectNotFiles: []string{
				".env",
				".env.local",
				".vscode/settings.json",
				"docker-compose.override.yml",
			},
		},
		{
			name: "default behavior (no copying)",
			args: []string{generateUniqueBranchNameEnhanced("test-default")},
			expectFiles: []string{
				"README.md", // This should always exist from git
			},
			expectNotFiles: []string{
				".env",
				".env.local",
				".vscode/settings.json",
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

			tempDir := testutils.NewTestDirectory(t, "grove-create-url-test")
			defer tempDir.Cleanup()

			originalDir, err := os.Getwd()
			require.NoError(t, err)
			defer os.Chdir(originalDir)

			err = os.Chdir(tempDir.Path)
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
	}
}

func TestCreateCommand_RemoteBranches_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryForCreate(t)
	defer cleanup()

	// Set up a fake remote for testing
	remoteDir := filepath.Join(repoDir, "..", "remote")
	err := os.MkdirAll(remoteDir, 0o755)
	require.NoError(t, err)

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer os.Chdir(originalDir)

	// Initialize remote repository
	err = os.Chdir(remoteDir)
	require.NoError(t, err)

	_, err = git.ExecuteGit("init", "--bare")
	require.NoError(t, err)

	// Go back to main repo
	err = os.Chdir(repoDir)
	require.NoError(t, err)

	// Add remote
	_, err = git.ExecuteGit("remote", "add", "origin", remoteDir)
	require.NoError(t, err)

	// Create and push a branch to remote
	remoteBranch := generateUniqueBranchNameEnhanced("remote-feature")
	_, err = git.ExecuteGit("checkout", "-b", remoteBranch)
	require.NoError(t, err)

	remoteFile := filepath.Join(repoDir, "remote-file.txt")
	err = os.WriteFile(remoteFile, []byte("remote content"), 0o644)
	require.NoError(t, err)

	_, err = git.ExecuteGit("add", "remote-file.txt")
	require.NoError(t, err)

	_, err = git.ExecuteGit("commit", "-m", "Add remote file")
	require.NoError(t, err)

	_, err = git.ExecuteGit("push", "origin", remoteBranch)
	require.NoError(t, err)

	// Now test creating worktree from remote branch
	t.Run("create from remote branch", func(t *testing.T) {
		// Go back to main branch
		_, err = git.ExecuteGit("checkout", "main")
		require.NoError(t, err)

		// Create worktree from remote branch
		cmd := NewCreateCmd()
		remoteBranchRef := "origin/" + remoteBranch
		cmd.SetArgs([]string{remoteBranchRef})

		err = cmd.Execute()
		require.NoError(t, err)

		// Verify worktree was created
		worktreePath := filepath.Join(repoDir, remoteBranch) // Should use branch name without origin/
		assert.DirExists(t, worktreePath)

		// Verify remote file exists
		remoteFileInWorktree := filepath.Join(worktreePath, "remote-file.txt")
		assert.FileExists(t, remoteFileInWorktree)
	})
}

func TestCreateCommand_ConflictResolution_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryForCreate(t)
	defer cleanup()

	branchName := generateUniqueBranchNameEnhanced("conflict-test")

	// Create the branch first
	_, err := git.ExecuteGit("checkout", "-b", branchName)
	require.NoError(t, err)

	// Go back to main
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
}

func TestCreateCommand_CustomPath_Integration(t *testing.T) {
	repoDir, cleanup := setupTestRepositoryForCreate(t)
	defer cleanup()

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
			cmd := NewCreateCmd()
			cmd.SetArgs([]string{tt.branchName, tt.customPath})

			err := cmd.Execute()
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
			errorText:   "cannot use both --no-copy and --copy-env",
		},
		{
			name:        "no-copy with copy patterns",
			args:        []string{"test-branch", "--no-copy", "--copy", ".env*"},
			expectError: true,
			errorText:   "cannot use both --no-copy and --copy",
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
			repoDir, cleanup := setupTestRepositoryForCreate(t)
			defer cleanup()

			cmd := NewCreateCmd()
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if tt.expectError {
				assert.Error(t, err)
				if tt.errorText != "" {
					assert.Contains(t, err.Error(), tt.errorText)
				}
			} else {
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
	repoDir, cleanup := setupTestRepositoryForCreate(t)
	defer cleanup()

	// Test that progress callback works without panicking
	branchName := generateUniqueBranchNameEnhanced("progress-test")

	cmd := NewCreateCmd()
	cmd.SetArgs([]string{branchName})

	// Execute should work with progress indicators
	err := cmd.Execute()
	require.NoError(t, err)

	worktreePath := filepath.Join(repoDir, branchName)
	assert.DirExists(t, worktreePath)

	// Clean up
	_, err = git.ExecuteGit("worktree", "remove", worktreePath)
	require.NoError(t, err)
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
			errorText:   "requires at least 1 arg",
		},
		{
			name:        "too many arguments",
			args:        []string{"branch", "path", "extra"},
			expectError: true,
			errorText:   "accepts at most 2 arg",
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
				repoDir, cleanup := setupTestRepositoryForCreate(t)
				defer cleanup()

				err := cmd.Execute()
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