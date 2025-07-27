//go:build !integration
// +build !integration

package completion

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestWorktreeCompletion(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		mockOutput string
		wantError  bool
	}{
		{
			name:       "empty input",
			toComplete: "",
			mockOutput: "",
			wantError:  false,
		},
		{
			name:       "with worktrees",
			toComplete: "ma",
			mockOutput: "worktree /repo/main\nHEAD abc123\nbranch refs/heads/main\n\nworktree /repo/feature\nHEAD def456\nbranch refs/heads/feature\n\n",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalCache.Clear()

			mockExecutor := testutils.NewMockGitExecutor()
			mockExecutor.SetResponse("worktree list --porcelain", tt.mockOutput, nil)

			ctx := NewCompletionContext(mockExecutor)
			cmd := &cobra.Command{}

			_, directive := WorktreeCompletion(ctx, cmd, []string{}, tt.toComplete)

			if tt.wantError {
				assert.Equal(t, cobra.ShellCompDirectiveError, directive)
			} else {
				assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
			}
		})
	}
}

func TestGetWorktreeNames(t *testing.T) {
	tests := []struct {
		name        string
		mockOutput  string
		mockError   error
		expected    []string
		expectError bool
	}{
		{
			name:        "no worktrees",
			mockOutput:  "",
			mockError:   nil,
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "with worktrees",
			mockOutput:  "worktree /repo/main\nHEAD abc123\nbranch refs/heads/main\n\nworktree /repo/feature\nHEAD def456\nbranch refs/heads/feature\n\n",
			mockError:   nil,
			expected:    []string{"main", "feature"},
			expectError: false,
		},
		{
			name:        "git command error",
			mockOutput:  "",
			mockError:   assert.AnError,
			expected:    []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalCache.Clear()

			mockExecutor := testutils.NewMockGitExecutor()
			if tt.mockError != nil {
				mockExecutor.SetErrorResponse("worktree list --porcelain", tt.mockError)
			} else {
				mockExecutor.SetResponse("worktree list --porcelain", tt.mockOutput, nil)
			}

			ctx := NewCompletionContext(mockExecutor)

			results, err := getWorktreeNames(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expected, results)
			}
		})
	}
}

func TestWorktreePathCompletion(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		mockOutput string
		wantError  bool
	}{
		{
			name:       "empty input",
			toComplete: "",
			mockOutput: "",
			wantError:  false,
		},
		{
			name:       "with paths",
			toComplete: "/repo",
			mockOutput: "worktree /repo/main\nHEAD abc123\nbranch refs/heads/main\n\n",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalCache.Clear()

			mockExecutor := testutils.NewMockGitExecutor()
			mockExecutor.SetResponse("worktree list --porcelain", tt.mockOutput, nil)

			ctx := NewCompletionContext(mockExecutor)

			cmd := &cobra.Command{}
			_, directive := WorktreePathCompletion(ctx, cmd, []string{}, tt.toComplete)

			if tt.wantError {
				assert.Equal(t, cobra.ShellCompDirectiveError, directive)
			} else {
				assert.Equal(t, cobra.ShellCompDirectiveDefault, directive)
			}
		})
	}
}

func TestGetWorktreePaths(t *testing.T) {
	tests := []struct {
		name        string
		mockOutput  string
		mockError   error
		expected    []string
		expectError bool
	}{
		{
			name:        "no worktrees",
			mockOutput:  "",
			mockError:   nil,
			expected:    []string{},
			expectError: false,
		},
		{
			name:        "with paths",
			mockOutput:  "worktree /repo/main\nHEAD abc123\nbranch refs/heads/main\n\n",
			mockError:   nil,
			expected:    []string{"/repo/main"},
			expectError: false,
		},
		{
			name:        "git command error",
			mockOutput:  "",
			mockError:   assert.AnError,
			expected:    []string{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			if tt.mockError != nil {
				mockExecutor.SetErrorResponse("worktree list --porcelain", tt.mockError)
			} else {
				mockExecutor.SetResponse("worktree list --porcelain", tt.mockOutput, nil)
			}

			ctx := NewCompletionContext(mockExecutor)

			results, err := getWorktreePaths(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expected, results)
			}
		})
	}
}

func TestBranchToWorktreeName(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected string
	}{
		{
			name:     "simple branch",
			branch:   "main",
			expected: "main",
		},
		{
			name:     "feature branch",
			branch:   "feature/auth",
			expected: "feature-auth",
		},
		{
			name:     "complex branch",
			branch:   "bugfix/user-login/fix-validation",
			expected: "bugfix-user-login-fix-validation",
		},
		{
			name:     "empty branch",
			branch:   "",
			expected: "",
		},
		{
			name:     "branch with refs",
			branch:   "refs/heads/main",
			expected: "refs-heads-main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BranchToWorktreeName(tt.branch)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorktreeNameToBranch(t *testing.T) {
	tests := []struct {
		name         string
		worktreeName string
		expected     string
	}{
		{
			name:         "simple name",
			worktreeName: "main",
			expected:     "main",
		},
		{
			name:         "dashed name",
			worktreeName: "feature-auth",
			expected:     "feature/auth",
		},
		{
			name:         "complex name",
			worktreeName: "bugfix-user-login-fix-validation",
			expected:     "bugfix/user/login/fix/validation", // All dashes become slashes.
		},
		{
			name:         "empty name",
			worktreeName: "",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WorktreeNameToBranch(tt.worktreeName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSuggestWorktreeNamesForBranch(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected []string
	}{
		{
			name:     "simple branch",
			branch:   "main",
			expected: []string{"main", "main"},
		},
		{
			name:     "feature branch",
			branch:   "feature/auth",
			expected: []string{"feature-auth", "auth"},
		},
		{
			name:     "complex branch",
			branch:   "bugfix/user-login/fix-validation",
			expected: []string{"bugfix-user-login-fix-validation"},
		},
		{
			name:     "empty branch",
			branch:   "",
			expected: []string{""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SuggestWorktreeNamesForBranch(tt.branch)
			assert.ElementsMatch(t, tt.expected, results)
		})
	}
}

func TestGetWorktreeInfo(t *testing.T) {
	tests := []struct {
		name        string
		mockOutput  string
		mockError   error
		expected    []WorktreeInfo
		expectError bool
	}{
		{
			name:        "no worktrees",
			mockOutput:  "",
			mockError:   nil,
			expected:    []WorktreeInfo{},
			expectError: false,
		},
		{
			name:        "single worktree",
			mockOutput:  "worktree /repo/main\nHEAD abc123\nbranch refs/heads/main\n\n",
			mockError:   nil,
			expected:    []WorktreeInfo{{Path: "/repo/main", Head: "abc123", Branch: "refs/heads/main"}},
			expectError: false,
		},
		{
			name:        "git command error",
			mockOutput:  "",
			mockError:   assert.AnError,
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			if tt.mockError != nil {
				mockExecutor.SetErrorResponse("worktree list --porcelain", tt.mockError)
			} else {
				mockExecutor.SetResponse("worktree list --porcelain", tt.mockOutput, nil)
			}

			ctx := NewCompletionContext(mockExecutor)

			results, err := GetWorktreeInfo(ctx)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.ElementsMatch(t, tt.expected, results)
			}
		})
	}
}

func TestWorktreeInfoMethods(t *testing.T) {
	worktree := WorktreeInfo{
		Path:       "/repo/main",
		Head:       "abc123",
		Branch:     "refs/heads/main",
		IsBare:     false,
		IsDetached: false,
	}

	t.Run("Name", func(t *testing.T) {
		assert.Equal(t, "main", worktree.Name())
	})

	t.Run("IsMainWorktree - not main", func(t *testing.T) {
		assert.False(t, worktree.IsMainWorktree())
	})

	t.Run("IsMainWorktree - bare", func(t *testing.T) {
		bareWorktree := WorktreeInfo{Path: "/repo", IsBare: true}
		assert.True(t, bareWorktree.IsMainWorktree())
	})

	t.Run("IsMainWorktree - dot name", func(t *testing.T) {
		dotWorktree := WorktreeInfo{Path: "/repo/."}
		assert.True(t, dotWorktree.IsMainWorktree())
	})
}
