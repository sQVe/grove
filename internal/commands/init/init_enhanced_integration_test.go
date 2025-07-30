//go:build integration
// +build integration

package init

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutils"
	"github.com/sqve/grove/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommand_Convert_Integration(t *testing.T) {
	// Test the --convert flag with various repository states

	t.Run("convert traditional repo", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			testDir := helper.CreateTempDir("grove-init-convert-traditional")
			err := os.Chdir(testDir)
			require.NoError(t, err)

			_, err = git.ExecuteGit("init")
			require.NoError(t, err)

			readmeFile := filepath.Join(testDir, "README.md")
			err = os.WriteFile(readmeFile, []byte("# Test Repository\n"), 0o644)
			require.NoError(t, err)

			_, err = git.ExecuteGit("add", "README.md")
			require.NoError(t, err)

			_, err = git.ExecuteGit("commit", "-m", "Initial commit")
			require.NoError(t, err)

			// Set up a fake remote to satisfy safety checks
			fakeRemoteDir := helper.CreateTempDir("fake-remote")
			_, err = git.ExecuteGit("init", "--bare", fakeRemoteDir)
			require.NoError(t, err)

			_, err = git.ExecuteGit("remote", "add", "origin", fakeRemoteDir)
			require.NoError(t, err)

			_, err = git.ExecuteGit("push", "-u", "origin", "main")
			require.NoError(t, err)

			gitDir := filepath.Join(testDir, ".git")
			stat, err := os.Stat(gitDir)
			require.NoError(t, err)
			assert.True(t, stat.IsDir()) // Should be a directory, not a file

			cmd := NewInitCmd()
			cmd.SetArgs([]string{"--convert"})

			err = cmd.Execute()
			require.NoError(t, err)

			bareDir := filepath.Join(testDir, ".bare")
			assert.DirExists(t, bareDir)

			gitFile := filepath.Join(testDir, ".git")
			assert.FileExists(t, gitFile)

			content, err := os.ReadFile(gitFile)
			require.NoError(t, err)
			assert.Equal(t, "gitdir: .bare\n", string(content))

			isRepo, err := utils.IsGitRepository(git.DefaultExecutor)
			require.NoError(t, err)
			assert.True(t, isRepo)
		})
	})

	t.Run("convert already grove repo", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			testDir := helper.CreateTempDir("grove-init-convert-already-grove")
			err := os.Chdir(testDir)
			require.NoError(t, err)

			bareDir := filepath.Join(testDir, ".bare")
			err = git.InitBare(bareDir)
			require.NoError(t, err)

			err = git.CreateGitFile(testDir, bareDir)
			require.NoError(t, err)

			cmd := NewInitCmd()
			cmd.SetArgs([]string{"--convert"})

			err = cmd.Execute()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "repository already exists at")
		})
	})

	t.Run("convert non-git directory", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			tempDir := helper.CreateTempDir("grove-init-convert-non-git")
			err := os.Chdir(tempDir)
			require.NoError(t, err)

			cmd := NewInitCmd()
			cmd.SetArgs([]string{"--convert"})

			err = cmd.Execute()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "repository not found at")
		})
	})

	t.Run("convert with arguments should fail", func(t *testing.T) {
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert", "some-arg"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify arguments when using --convert")
	})

	t.Run("convert with branches should fail", func(t *testing.T) {
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--convert", "--branches=main,dev"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot use --branches flag with --convert")
	})
}

func TestInitCommand_Branches_Integration(t *testing.T) {

	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	t.Run("branches without URL should fail", func(t *testing.T) {
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"--branches=main,dev"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--branches flag requires a remote URL")
	})

	t.Run("branches with local directory should succeed", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		tempDir := helper.CreateTempDir("grove-init-branches-local")

		cmd := NewInitCmd()
		cmd.SetArgs([]string{tempDir, "--branches=main,dev"})

		err := cmd.Execute()
		// The current implementation allows branches with local directories
		// It just ignores the branches flag for local initialization
		assert.NoError(t, err)

		bareDir := filepath.Join(tempDir, ".bare")
		assert.DirExists(t, bareDir)
	})

	// Note: Testing with real remote URLs would require network access
	// Consider adding tests with mock HTTP servers for full coverage
}

func TestInitCommand_URLParsing_Integration(t *testing.T) {
	// Test smart URL parsing for different platforms

	tests := []struct {
		name        string
		url         string
		expectError bool
		skipReason  string
	}{
		{
			name:       "GitHub standard URL",
			url:        "https://github.com/git/git.git",
			skipReason: "Requires network access",
		},
		{
			name:       "GitHub URL with branch",
			url:        "https://github.com/git/git/tree/main",
			skipReason: "Requires network access",
		},
		{
			name:       "GitHub PR URL",
			url:        "https://github.com/git/git/pull/123",
			skipReason: "Requires network access",
		},
		{
			name:       "GitLab URL",
			url:        "https://gitlab.com/gitlab-org/gitlab.git",
			skipReason: "Requires network access",
		},
		{
			name:        "Invalid URL treated as local path",
			url:         "not-a-valid-url",
			expectError: false, // Current implementation treats this as a local directory name
		},
		{
			name:       "SSH URL",
			url:        "git@github.com:git/git.git",
			skipReason: "Requires SSH access",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipReason != "" && !testing.Short() {
				t.Skip(tt.skipReason)
			}

			helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
			runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
			runner.Run(func() {
				tempDir := helper.CreateTempDir("grove-init-url-parsing")
				err := os.Chdir(tempDir)
				require.NoError(t, err)

				cmd := NewInitCmd()
				cmd.SetArgs([]string{tt.url})

				err = cmd.Execute()
				if tt.expectError {
					assert.Error(t, err)
				} else if tt.skipReason == "" {
					assert.NoError(t, err)
				}
			})
		})
	}
}

func TestInitCommand_ErrorScenarios_Integration(t *testing.T) {
	t.Run("existing .git file", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		testDir := helper.CreateTempDir("grove-init-existing-git-file")

		gitFile := filepath.Join(testDir, ".git")
		err := os.WriteFile(gitFile, []byte("gitdir: somewhere"), 0o644)
		require.NoError(t, err)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{testDir})

		err = cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("existing .git directory", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		testDir := helper.CreateTempDir("grove-init-existing-git-dir")

		gitDir := filepath.Join(testDir, ".git")
		err := os.MkdirAll(gitDir, 0o755)
		require.NoError(t, err)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{testDir})

		err = cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("existing .bare directory", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		testDir := helper.CreateTempDir("grove-init-existing-bare")

		bareDir := filepath.Join(testDir, ".bare")
		err := os.MkdirAll(bareDir, 0o755)
		require.NoError(t, err)

		cmd := NewInitCmd()
		cmd.SetArgs([]string{testDir})

		err = cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("permission denied", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Cannot test permission denied as root")
		}

		// Try to create in /root (should fail for non-root users)
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"/root/grove-test"})

		err := cmd.Execute()
		assert.Error(t, err)
	})

	t.Run("too many arguments", func(t *testing.T) {
		cmd := NewInitCmd()
		cmd.SetArgs([]string{"arg1", "arg2", "arg3"})

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "too many arguments")
	})

	t.Run("git not available", func(t *testing.T) {
		// This test would require mocking the git availability check
		t.Skip("Requires mocking git availability")
	})
}

func TestInitCommand_RemoteScenarios_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent tests in short mode")
	}

	t.Run("non-empty directory for remote", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			tempDir := helper.CreateTempDir("grove-init-remote-non-empty")

			testFile := filepath.Join(tempDir, "existing.txt")
			err := os.WriteFile(testFile, []byte("existing content"), 0o644)
			require.NoError(t, err)

			err = os.Chdir(tempDir)
			require.NoError(t, err)

			cmd := NewInitCmd()
			// Using a fake URL to avoid network dependency
			cmd.SetArgs([]string{"https://example.com/fake/repo.git"})

			err = cmd.Execute()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not empty")
		})
	})

	t.Run("hidden files allowed for remote", func(t *testing.T) {
		// Skip this test as it requires network access and causes timeouts
		// The directory validation logic is tested elsewhere
		t.Skip("Test causes network timeouts with fake URLs")
	})
}

func TestInitCommand_BranchParsing_Integration(t *testing.T) {
	tests := []struct {
		name             string
		branchesStr      string
		expectedCount    int
		expectedBranches []string
	}{
		{
			name:             "single branch",
			branchesStr:      "main",
			expectedCount:    1,
			expectedBranches: []string{"main"},
		},
		{
			name:             "multiple branches",
			branchesStr:      "main,develop,feature/auth",
			expectedCount:    3,
			expectedBranches: []string{"main", "develop", "feature/auth"},
		},
		{
			name:             "branches with spaces",
			branchesStr:      "main, develop , feature/auth",
			expectedCount:    3,
			expectedBranches: []string{"main", "develop", "feature/auth"},
		},
		{
			name:             "empty string",
			branchesStr:      "",
			expectedCount:    0,
			expectedBranches: nil,
		},
		{
			name:             "invalid branch names filtered",
			branchesStr:      "main,invalid..branch,valid-branch,-invalid",
			expectedCount:    2,
			expectedBranches: []string{"main", "valid-branch"},
		},
		{
			name:             "branch with slashes",
			branchesStr:      "feature/user-auth,bugfix/critical-fix",
			expectedCount:    2,
			expectedBranches: []string{"feature/user-auth", "bugfix/critical-fix"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseBranches(tt.branchesStr)
			assert.Equal(t, tt.expectedCount, len(result))
			assert.Equal(t, tt.expectedBranches, result)
		})
	}
}

func TestInitCommand_BranchValidation_Integration(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected bool
	}{
		{"valid simple", "main", true},
		{"valid with slash", "feature/auth", true},
		{"valid with dash", "bug-fix", true},
		{"valid with numbers", "v1.2.3", true},
		{"empty", "", false},
		{"just dash", "-", false},
		{"starts with dash", "-invalid", false},
		{"ends with .lock", "branch.lock", false},
		{"starts with slash", "/invalid", false},
		{"ends with slash", "invalid/", false},
		{"contains double dot", "invalid..branch", false},
		{"contains space", "invalid branch", false},
		{"contains tilde", "invalid~branch", false},
		{"contains caret", "invalid^branch", false},
		{"contains colon", "invalid:branch", false},
		{"contains question", "invalid?branch", false},
		{"contains asterisk", "invalid*branch", false},
		{"contains bracket", "invalid[branch", false},
		{"contains backslash", "invalid\\branch", false},
		{"valid complex", "feature/user-auth-v2", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidBranchName(tt.branch)
			assert.Equal(t, tt.expected, result, "Branch name: %q", tt.branch)
		})
	}
}

func TestInitCommand_LocalPathHandling_Integration(t *testing.T) {
	t.Run("current directory", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			tempDir := helper.CreateTempDir("grove-init-current-dir")

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
		})
	})

	t.Run("relative path", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		runner := testutils.NewTestRunner(t).WithIsolatedWorkingDir()
		runner.Run(func() {
			tempDir := helper.CreateTempDir("grove-init-relative-path")

			err := os.Chdir(tempDir)
			require.NoError(t, err)

			targetPath := "my-repo"
			cmd := NewInitCmd()
			cmd.SetArgs([]string{targetPath})

			err = cmd.Execute()
			require.NoError(t, err)

			fullTargetPath := filepath.Join(tempDir, targetPath)
			bareDir := filepath.Join(fullTargetPath, ".bare")
			assert.DirExists(t, bareDir)

			gitFile := filepath.Join(fullTargetPath, ".git")
			assert.FileExists(t, gitFile)
		})
	})

	t.Run("absolute path", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		tempDir := helper.CreateTempDir("grove-init-absolute-path")

		targetPath := filepath.Join(tempDir, "absolute-repo")
		cmd := NewInitCmd()
		cmd.SetArgs([]string{targetPath})

		err := cmd.Execute()
		require.NoError(t, err)

		bareDir := filepath.Join(targetPath, ".bare")
		assert.DirExists(t, bareDir)

		gitFile := filepath.Join(targetPath, ".git")
		assert.FileExists(t, gitFile)
	})

	t.Run("nested path creation", func(t *testing.T) {
		helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
		tempDir := helper.CreateTempDir("grove-init-nested-path")

		targetPath := filepath.Join(tempDir, "level1", "level2", "repo")
		cmd := NewInitCmd()
		cmd.SetArgs([]string{targetPath})

		err := cmd.Execute()
		require.NoError(t, err)

		assert.DirExists(t, targetPath)

		bareDir := filepath.Join(targetPath, ".bare")
		assert.DirExists(t, bareDir)

		gitFile := filepath.Join(targetPath, ".git")
		assert.FileExists(t, gitFile)
	})
}
