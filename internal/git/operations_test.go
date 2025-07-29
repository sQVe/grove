//go:build !integration
// +build !integration

package git

import (
	"fmt"
	"os"
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupMockGitExecutor() *testutils.MockGitExecutor {
	return testutils.NewMockGitExecutor()
}

func assertSafetyIssue(t *testing.T, issue SafetyIssue, expectedType, expectedDescContains string) {
	t.Helper()
	assert.Equal(t, expectedType, issue.Type)
	assert.Contains(t, issue.Description, expectedDescContains)
	assert.NotEmpty(t, issue.Solution)
}

func TestGitOperations_SafetyChecks(t *testing.T) {
	t.Run("checkGitStatus - clean repo", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("status --porcelain=v1", "")
		mock.SetSuccessResponse("status", "On branch main\nnothing to commit, working tree clean\n")

		issues, err := checkGitStatus(mock)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	t.Run("checkGitStatus - uncommitted changes", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("status --porcelain=v1", " M file1.txt\nA  file2.txt\n")
		mock.SetSuccessResponse("status", "On branch main\nChanges to be committed:\n  new file:   file2.txt\n\nChanges not staged for commit:\n  modified:   file1.txt\n")

		issues, err := checkGitStatus(mock)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "uncommitted_changes", issues[0].Type)
		assert.Contains(t, issues[0].Description, "modified")
		assert.Contains(t, issues[0].Description, "added")
		assert.Contains(t, issues[0].Solution, "git")
	})

	t.Run("checkGitStatus - rebase in progress", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("status --porcelain=v1", "")
		statusOutput := "On branch feature\nrebase in progress; onto abc123\nYou are currently rebasing branch 'feature' on 'abc123'.\n"
		mock.SetSuccessResponse("status", statusOutput)

		issues, err := checkGitStatus(mock)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "ongoing_rebase", issues[0].Type)
		assert.Contains(t, issues[0].Description, "rebase")
		assert.Contains(t, issues[0].Solution, "git rebase")
	})

	t.Run("checkStashedChanges - no stashes", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("stash list", "")

		issues, err := checkStashedChanges(mock)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	t.Run("checkStashedChanges - has stashes", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("stash list", "stash@{0}: WIP on main: abc123 Last commit\nstash@{1}: On feature: def456 Another stash\n")

		issues, err := checkStashedChanges(mock)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "stashed_changes", issues[0].Type)
		assert.Contains(t, issues[0].Description, "2 stashed")
		assert.Contains(t, issues[0].Solution, "git stash")
	})

	t.Run("checkUntrackedFiles - no untracked", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("ls-files --others --exclude-standard", "")

		issues, err := checkUntrackedFiles(mock)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	t.Run("checkUntrackedFiles - has untracked", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("ls-files --others --exclude-standard", "newfile.txt\ntemp.log\n")

		issues, err := checkUntrackedFiles(mock)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "untracked_files", issues[0].Type)
		assert.Contains(t, issues[0].Description, "2 untracked")
		assert.Contains(t, issues[0].Solution, "git add")
	})

	t.Run("checkExistingWorktrees - no worktrees", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("worktree list", "/path/to/repo  abc123 [main]\n")

		issues, err := checkExistingWorktrees(mock)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	t.Run("checkExistingWorktrees - has worktrees", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("worktree list", "/path/to/repo        abc123 [main]\n/path/to/feature     def456 [feature]\n/path/to/bugfix      789ghi [bugfix]\n")

		issues, err := checkExistingWorktrees(mock)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "existing_worktrees", issues[0].Type)
		assert.Contains(t, issues[0].Description, "existing worktree")
		assert.Contains(t, issues[0].Solution, "git worktree remove")
	})

	t.Run("checkUnpushedCommits - no unpushed", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("for-each-ref --format=%(refname:short) %(upstream:short) %(upstream:track) refs/heads", "main origin/main [up to date]\n")

		issues, err := checkUnpushedCommits(mock)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	t.Run("checkUnpushedCommits - has unpushed", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("for-each-ref --format=%(refname:short) %(upstream:short) %(upstream:track) refs/heads", "main origin/main [ahead 2]\nfeature origin/feature [ahead 1]\n")

		issues, err := checkUnpushedCommits(mock)
		require.NoError(t, err)
		require.Len(t, issues, 2)
		assert.Equal(t, "unpushed_commits", issues[0].Type)
		assert.Contains(t, issues[0].Description, "main")
		assert.Contains(t, issues[0].Description, "ahead 2")
		assert.Contains(t, issues[0].Solution, "git push origin main")
	})

	t.Run("checkLocalOnlyBranches - no local-only", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("for-each-ref --format=%(refname:short) %(upstream) refs/heads", "main origin/main\nfeature origin/feature\n")

		issues, err := checkLocalOnlyBranches(mock)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})

	t.Run("checkLocalOnlyBranches - has local-only", func(t *testing.T) {
		mock := testutils.NewMockGitExecutor()
		mock.SetSuccessResponse("for-each-ref --format=%(refname:short) %(upstream) refs/heads", "main origin/main\nexperiment\ntemp\n")

		issues, err := checkLocalOnlyBranches(mock)
		require.NoError(t, err)
		require.Len(t, issues, 1)
		assert.Equal(t, "local_only_branches", issues[0].Type)
		assert.Contains(t, issues[0].Description, "experiment, temp")
		assert.Contains(t, issues[0].Solution, "git push -u origin")
	})
}

// These individual test functions are redundant with TestSafetyChecks which provides comprehensive coverage.
// Removed: TestCheckGitStatus, TestCheckStashedChanges, TestCheckUntrackedFiles,
// TestCheckExistingWorktrees, TestCheckUnpushedCommits, TestCheckLocalOnlyBranches.

func TestGitOperations_GitChangeCounts(t *testing.T) {
	t.Run("HasChanges", func(t *testing.T) {
		tests := []struct {
			name     string
			counts   GitChangeCounts
			expected bool
		}{
			{
				name:     "no changes",
				counts:   GitChangeCounts{},
				expected: false,
			},
			{
				name:     "only untracked",
				counts:   GitChangeCounts{Untracked: 2},
				expected: false,
			},
			{
				name:     "has modified",
				counts:   GitChangeCounts{Modified: 1},
				expected: true,
			},
			{
				name:     "has added",
				counts:   GitChangeCounts{Added: 1},
				expected: true,
			},
			{
				name:     "has deleted",
				counts:   GitChangeCounts{Deleted: 1},
				expected: true,
			},
			{
				name:     "has renamed",
				counts:   GitChangeCounts{Renamed: 1},
				expected: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.counts.HasChanges())
			})
		}
	})

	t.Run("BuildDescription", func(t *testing.T) {
		tests := []struct {
			name     string
			counts   GitChangeCounts
			expected string
		}{
			{
				name:     "no changes",
				counts:   GitChangeCounts{},
				expected: "",
			},
			{
				name:     "single type",
				counts:   GitChangeCounts{Modified: 2},
				expected: "Uncommitted changes (2 modified)",
			},
			{
				name:     "multiple types",
				counts:   GitChangeCounts{Modified: 1, Added: 2, Deleted: 1},
				expected: "Uncommitted changes (1 modified, 2 added, 1 deleted)",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.counts.BuildDescription())
			})
		}
	})

	t.Run("BuildSolution", func(t *testing.T) {
		tests := []struct {
			name     string
			counts   GitChangeCounts
			expected string
		}{
			{
				name:     "no changes",
				counts:   GitChangeCounts{},
				expected: "",
			},
			{
				name:     "has changes",
				counts:   GitChangeCounts{Modified: 1},
				expected: "git add <files> && git commit",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				assert.Equal(t, tt.expected, tt.counts.BuildSolution())
			})
		}
	})
}

func TestCountGitChanges(t *testing.T) {
	tests := []struct {
		name     string
		lines    []string
		expected GitChangeCounts
	}{
		{
			name:     "empty lines",
			lines:    []string{""},
			expected: GitChangeCounts{},
		},
		{
			name:     "modified files",
			lines:    []string{" M file1.txt", "M  file2.txt"},
			expected: GitChangeCounts{Modified: 2},
		},
		{
			name:     "added files",
			lines:    []string{"A  file1.txt", "AM file2.txt"},
			expected: GitChangeCounts{Added: 2, Modified: 1},
		},
		{
			name:     "deleted files",
			lines:    []string{" D file1.txt", "D  file2.txt"},
			expected: GitChangeCounts{Deleted: 2},
		},
		{
			name:     "renamed files",
			lines:    []string{"R  old.txt -> new.txt", "C  copy.txt -> new_copy.txt"},
			expected: GitChangeCounts{Renamed: 2},
		},
		{
			name:     "untracked files",
			lines:    []string{"?? file1.txt", "?? file2.txt"},
			expected: GitChangeCounts{Untracked: 2},
		},
		{
			name:     "mixed changes",
			lines:    []string{" M file1.txt", "A  file2.txt", "?? file3.txt"},
			expected: GitChangeCounts{Modified: 1, Added: 1, Untracked: 1},
		},
		{
			name:     "short lines ignored",
			lines:    []string{"X", ""},
			expected: GitChangeCounts{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countGitChanges(tt.lines)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckOngoingGitOperations(t *testing.T) {
	tests := []struct {
		name           string
		statusOutput   string
		expectError    bool
		expectedIssues int
		expectedType   string
	}{
		{
			name:           "no ongoing operations",
			statusOutput:   "On branch main\nnothing to commit, working tree clean\n",
			expectedIssues: 0,
		},
		{
			name:           "rebase in progress",
			statusOutput:   "rebase in progress; onto abc123\n",
			expectedIssues: 1,
			expectedType:   "ongoing_rebase",
		},
		{
			name:           "merge in progress",
			statusOutput:   "merge in progress\n",
			expectedIssues: 1,
			expectedType:   "ongoing_merge",
		},
		{
			name:           "cherry-pick in progress",
			statusOutput:   "cherry-pick in progress\n",
			expectedIssues: 1,
			expectedType:   "ongoing_cherry_pick",
		},
		{
			name:           "bisect in progress",
			statusOutput:   "bisect in progress\n",
			expectedIssues: 1,
			expectedType:   "ongoing_bisect",
		},
		{
			name:           "multiple ongoing operations",
			statusOutput:   "rebase in progress; merge in progress\n",
			expectedIssues: 2,
			expectedType:   "ongoing_rebase",
		},
		{
			name:           "git command fails",
			statusOutput:   "",
			expectError:    true,
			expectedIssues: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := setupMockGitExecutor()
			if tt.expectError {
				mock.SetErrorResponse("status", fmt.Errorf("git command failed"))
			} else {
				mock.SetSuccessResponse("status", tt.statusOutput)
			}

			issues, err := checkOngoingGitOperations(mock)
			require.NoError(t, err) // Function should not return errors.
			assert.Len(t, issues, tt.expectedIssues)

			if tt.expectedIssues > 0 {
				assertSafetyIssue(t, issues[0], tt.expectedType, "progress")
			}
		})
	}
}

func TestParseGitStatusLine(t *testing.T) {
	tests := []struct {
		name             string
		line             string
		expectedStaged   rune
		expectedUnstaged rune
	}{
		{
			name:             "empty line",
			line:             "",
			expectedStaged:   ' ',
			expectedUnstaged: ' ',
		},
		{
			name:             "single char line",
			line:             "X",
			expectedStaged:   ' ',
			expectedUnstaged: ' ',
		},
		{
			name:             "modified unstaged",
			line:             " M",
			expectedStaged:   ' ',
			expectedUnstaged: 'M',
		},
		{
			name:             "added staged",
			line:             "A ",
			expectedStaged:   'A',
			expectedUnstaged: ' ',
		},
		{
			name:             "untracked",
			line:             "??",
			expectedStaged:   '?',
			expectedUnstaged: '?',
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			staged, unstaged := parseGitStatusLine(tt.line)
			assert.Equal(t, tt.expectedStaged, staged)
			assert.Equal(t, tt.expectedUnstaged, unstaged)
		})
	}
}

func TestCheckRepositorySafetyForConversionWithMock(t *testing.T) {
	t.Run("safe repository", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "grove-safety-mock-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		// Use mock executor to simulate a safe repository state.
		mockExecutor := testutils.NewMockGitExecutor()
		mockExecutor.SetSafeRepositoryState()

		issues, err := checkRepositorySafetyForConversion(mockExecutor, tempDir)
		require.NoError(t, err)
		assert.Empty(t, issues)
	})
}

func TestFormatSafetyIssuesError(t *testing.T) {
	issues := []SafetyIssue{
		{
			Type:        "uncommitted_changes",
			Description: "Uncommitted changes (2 modified)",
			Solution:    "git add <files> && git commit",
		},
		{
			Type:        "stashed_changes",
			Description: "1 stashed change(s)",
			Solution:    "Apply with 'git stash pop' or remove with 'git stash drop'",
		},
		{
			Type:        "untracked_files",
			Description: "3 untracked file(s)",
			Solution:    "Add to git with 'git add <files>' or add to .gitignore",
		},
	}

	err := formatSafetyIssuesError(issues)
	require.Error(t, err)

	errMsg := err.Error()
	assert.Contains(t, errMsg, "Repository is not ready for conversion:")
	assert.Contains(t, errMsg, "✗ Uncommitted changes")
	assert.Contains(t, errMsg, "✗ 1 stashed change(s)")
	assert.Contains(t, errMsg, "✗ 3 untracked file(s)")
	assert.Contains(t, errMsg, "git add <files> && git commit")
	assert.Contains(t, errMsg, "git stash pop")
	assert.Contains(t, errMsg, "add to .gitignore")
	assert.Contains(t, errMsg, "Please resolve these issues before converting")
}

func TestValidatePaths(t *testing.T) {
	t.Run("valid paths", func(t *testing.T) {
		tempDir := t.TempDir()
		bareDir := tempDir + "/.bare"

		err := validatePaths(tempDir, bareDir)
		assert.NoError(t, err)
	})

	t.Run("path with directory traversal", func(t *testing.T) {
		tempDir := t.TempDir()
		maliciousPath := tempDir + "/../../../etc/passwd"

		err := validatePaths(tempDir, maliciousPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "paths contain directory traversal sequences")
	})

	t.Run("bare directory with directory traversal", func(t *testing.T) {
		tempDir := t.TempDir()
		maliciousPath := "../../../etc/passwd"

		err := validatePaths(tempDir, maliciousPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "paths contain directory traversal sequences")
	})
}

func TestCreateGitFileWithSecurity(t *testing.T) {
	t.Run("valid paths", func(t *testing.T) {
		tempDir := t.TempDir()
		bareDir := tempDir + "/.bare"

		err := CreateGitFile(tempDir, bareDir)
		assert.NoError(t, err)
	})

	t.Run("path with directory traversal in bare directory", func(t *testing.T) {
		tempDir := t.TempDir()
		maliciousPath := tempDir + "/../../../etc/passwd"

		err := CreateGitFile(tempDir, maliciousPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid paths")
	})

	t.Run("relative path with directory traversal", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create a path that would result in ../ in the relative path.
		parentDir := tempDir + "/.."
		bareDir := parentDir + "/malicious"

		err := CreateGitFile(tempDir, bareDir)
		assert.Error(t, err)
		// The error should come from validatePaths first (since path contains ..).
		assert.Contains(t, err.Error(), "invalid paths")
	})
}
