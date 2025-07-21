package commands

import (
	"regexp"
	"testing"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// mockGitExecutor sets up a mock git executor and returns a cleanup function.
// This encapsulates the common pattern of saving/restoring git.DefaultExecutor.
func mockGitExecutor() (mockExecutor *testutils.MockGitExecutor, cleanup func()) {
	originalExecutor := git.DefaultExecutor
	mockExecutor = testutils.NewMockGitExecutor()
	git.DefaultExecutor = mockExecutor

	cleanup = func() {
		git.DefaultExecutor = originalExecutor
	}

	return mockExecutor, cleanup
}

// setupWorktreeListMock sets up a regex-based mock for worktree list commands
// that works regardless of the actual .bare directory path.
func setupWorktreeListMock(mockExecutor *testutils.MockGitExecutor, output string, err error) {
	// Pattern matches: -C <any_path>/.bare worktree list --porcelain
	pattern := regexp.MustCompile(`^-C .+/\.bare worktree list --porcelain$`)
	mockExecutor.SetResponsePattern(pattern, output, err)
}

// withMockedWorktreeList sets up a complete mock git executor with worktree list mock
// and returns a cleanup function. This is the most convenient helper for list command tests.
func withMockedWorktreeList(output string, err error) func() {
	mockExecutor, cleanup := mockGitExecutor()
	setupWorktreeListMock(mockExecutor, output, err)
	return cleanup
}

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()

	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Use)
	assert.Contains(t, cmd.Short, "List all worktrees")
	assert.Contains(t, cmd.Long, "List all Git worktrees")

	// Test flags are present
	assert.NotNil(t, cmd.Flags().Lookup("sort"))
	assert.NotNil(t, cmd.Flags().Lookup("verbose"))
	assert.NotNil(t, cmd.Flags().Lookup("porcelain"))
	assert.NotNil(t, cmd.Flags().Lookup("dirty"))
	assert.NotNil(t, cmd.Flags().Lookup("stale"))
	assert.NotNil(t, cmd.Flags().Lookup("clean"))
	assert.NotNil(t, cmd.Flags().Lookup("days"))
}

func TestListSortOptions(t *testing.T) {
	tests := []struct {
		name     string
		sortBy   ListSortOption
		expected string
	}{
		{"sort by activity", SortByActivity, "activity"},
		{"sort by name", SortByName, "name"},
		{"sort by status", SortByStatus, "status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.sortBy))
		})
	}
}

func TestApplyFilters(t *testing.T) {
	now := time.Now()

	worktrees := []git.WorktreeInfo{
		{
			Path:         "/repo/main",
			Branch:       "main",
			LastActivity: now.AddDate(0, 0, -1), // 1 day ago
			Status:       git.WorktreeStatus{IsClean: true},
		},
		{
			Path:         "/repo/feature",
			Branch:       "feature/test",
			LastActivity: now.AddDate(0, 0, -5), // 5 days ago
			Status:       git.WorktreeStatus{Modified: 2, IsClean: false},
		},
		{
			Path:         "/repo/old",
			Branch:       "old/branch",
			LastActivity: now.AddDate(0, 0, -35), // 35 days ago (stale)
			Status:       git.WorktreeStatus{IsClean: true},
		},
	}

	tests := []struct {
		name     string
		options  *ListOptions
		expected int
	}{
		{
			name:     "no filter",
			options:  &ListOptions{},
			expected: 3,
		},
		{
			name:     "dirty only",
			options:  &ListOptions{DirtyOnly: true},
			expected: 1,
		},
		{
			name:     "clean only",
			options:  &ListOptions{CleanOnly: true},
			expected: 2,
		},
		{
			name:     "stale only (30 days)",
			options:  &ListOptions{StaleOnly: true, StaleDays: 30},
			expected: 1,
		},
		{
			name:     "stale only (40 days)",
			options:  &ListOptions{StaleOnly: true, StaleDays: 40},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ListService{}
			filtered := service.applyFilters(worktrees, tt.options)
			assert.Len(t, filtered, tt.expected)
		})
	}
}

func TestSortWorktrees(t *testing.T) {
	now := time.Now()

	worktrees := []git.WorktreeInfo{
		{
			Path:         "/repo/zebra",
			Branch:       "zebra",
			LastActivity: now.AddDate(0, 0, -5), // 5 days ago
			Status:       git.WorktreeStatus{IsClean: true},
			IsCurrent:    false,
		},
		{
			Path:         "/repo/alpha",
			Branch:       "alpha",
			LastActivity: now.AddDate(0, 0, -1), // 1 day ago (most recent)
			Status:       git.WorktreeStatus{Modified: 1, IsClean: false},
			IsCurrent:    false,
		},
		{
			Path:         "/repo/main",
			Branch:       "main",
			LastActivity: now.AddDate(0, 0, -3), // 3 days ago
			Status:       git.WorktreeStatus{IsClean: true},
			IsCurrent:    true, // Current worktree
		},
	}

	t.Run("sort by activity", func(t *testing.T) {
		sorted := make([]git.WorktreeInfo, len(worktrees))
		copy(sorted, worktrees)

		service := &ListService{}
		service.sortWorktrees(sorted, SortByActivity)

		// Sort by most recent activity (alpha is 1 day ago, main is 3 days ago, zebra is 5 days ago)
		assert.Equal(t, "/repo/alpha", sorted[0].Path)
		assert.Equal(t, "/repo/main", sorted[1].Path)
		assert.Equal(t, "/repo/zebra", sorted[2].Path)
	})

	t.Run("sort by name", func(t *testing.T) {
		sorted := make([]git.WorktreeInfo, len(worktrees))
		copy(sorted, worktrees)

		service := &ListService{}
		service.sortWorktrees(sorted, SortByName)

		// Sort alphabetically by path
		assert.Equal(t, "/repo/alpha", sorted[0].Path)
		assert.Equal(t, "/repo/main", sorted[1].Path)
		assert.Equal(t, "/repo/zebra", sorted[2].Path)
	})

	t.Run("sort by status", func(t *testing.T) {
		sorted := make([]git.WorktreeInfo, len(worktrees))
		copy(sorted, worktrees)

		service := &ListService{}
		service.sortWorktrees(sorted, SortByStatus)

		// Sort by status (dirty first, then clean), then by activity within same status
		assert.Equal(t, "/repo/alpha", sorted[0].Path) // dirty, most recent
		assert.Equal(t, "/repo/main", sorted[1].Path)  // clean, more recent than zebra
		assert.Equal(t, "/repo/zebra", sorted[2].Path) // clean, older
	})
}

func TestValidateListOptions(t *testing.T) {
	tests := []struct {
		name        string
		options     *ListOptions
		expectError bool
	}{
		{
			name: "valid options",
			options: &ListOptions{
				Sort: SortByActivity,
			},
			expectError: false,
		},
		{
			name: "multiple filters",
			options: &ListOptions{
				Sort:      SortByActivity,
				DirtyOnly: true,
				CleanOnly: true,
			},
			expectError: true,
		},
		{
			name: "invalid sort option",
			options: &ListOptions{
				Sort: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateListOptions(tt.options)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestFormatStatus(t *testing.T) {
	// This functionality is now in WorktreeFormatter and tested in worktree_formatter_test.go
	t.Skip("Test moved to worktree_formatter_test.go")
}

func TestFormatActivity(t *testing.T) {
	// This functionality is now in WorktreeFormatter and tested in worktree_formatter_test.go
	t.Skip("Test moved to worktree_formatter_test.go")
}

func TestRunListCommand_ValidationErrors(t *testing.T) {
	tests := []struct {
		name    string
		options *ListOptions
		wantErr bool
		errMsg  string
	}{
		{
			name: "multiple filters",
			options: &ListOptions{
				DirtyOnly: true,
				CleanOnly: true,
			},
			wantErr: true,
			errMsg:  "Cannot specify multiple filters",
		},
		{
			name: "invalid sort option",
			options: &ListOptions{
				Sort: "invalid",
			},
			wantErr: true,
			errMsg:  "Invalid sort option",
		},
		{
			name: "valid options",
			options: &ListOptions{
				Sort:      SortByActivity,
				DirtyOnly: true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock the git executor to avoid actual git calls
			mockExecutor, cleanup := mockGitExecutor()
			defer cleanup()
			setupWorktreeListMock(mockExecutor, "", nil)

			err := runListCommand(tt.options)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDisplayPorcelainOutput(t *testing.T) {
	worktrees := []git.WorktreeInfo{
		{
			Path:         "/repo/main",
			Branch:       "main",
			Head:         "abc123",
			IsCurrent:    true,
			LastActivity: time.Unix(1609459200, 0), // Jan 1, 2021
			Status:       git.WorktreeStatus{IsClean: true},
			Remote:       git.RemoteStatus{HasRemote: true, Ahead: 1},
		},
		{
			Path:      "/repo/feature",
			Branch:    "feature/test",
			Head:      "def456",
			IsCurrent: false,
			Status:    git.WorktreeStatus{Modified: 2, Staged: 1, IsClean: false},
		},
	}

	// Capture output (in a real test environment, you might use a buffer)
	presenter := NewListPresenter()
	err := presenter.DisplayPorcelain(worktrees)
	assert.NoError(t, err)

	// For a more thorough test, you could capture stdout and verify the exact output format
}

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Name())
	assert.False(t, cmd.RequiresConfig())
	assert.NotNil(t, cmd.Command())
}

func TestListCommand_Integration(t *testing.T) {
	// Test the command integration with the registry
	cmd := NewListCommand()

	// Verify the command can be created and has the right properties
	assert.Equal(t, "list", cmd.Name())
	assert.NotNil(t, cmd.Command())

	// Verify the cobra command has the expected structure
	cobraCmd := cmd.Command()
	assert.Equal(t, "list", cobraCmd.Use)
	assert.NotEmpty(t, cobraCmd.Short)
	assert.NotEmpty(t, cobraCmd.Long)

	// Test that flags are properly configured
	flags := cobraCmd.Flags()
	assert.NotNil(t, flags.Lookup("sort"))
	assert.NotNil(t, flags.Lookup("verbose"))
	assert.NotNil(t, flags.Lookup("porcelain"))
	assert.NotNil(t, flags.Lookup("dirty"))
	assert.NotNil(t, flags.Lookup("stale"))
	assert.NotNil(t, flags.Lookup("clean"))
	assert.NotNil(t, flags.Lookup("days"))
}

// Mock test to verify the command handles empty worktree list.
func TestListCommand_EmptyWorktrees(t *testing.T) {
	// Setup mock executor
	defer withMockedWorktreeList("", nil)()

	options := &ListOptions{Sort: SortByActivity}
	err := runListCommand(options)

	assert.NoError(t, err)
	// In a real test, you might capture stdout to verify "No worktrees found" is printed
}

func TestDisplayHumanOutput(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		worktrees []git.WorktreeInfo
		verbose   bool
		wantErr   bool
	}{
		{
			name:      "empty worktrees list",
			worktrees: []git.WorktreeInfo{},
			verbose:   false,
			wantErr:   false,
		},
		{
			name: "single worktree - minimal",
			worktrees: []git.WorktreeInfo{
				{
					Path:         "/repo/main",
					Branch:       "main",
					IsCurrent:    true,
					LastActivity: now.Add(-1 * time.Hour),
					Status:       git.WorktreeStatus{IsClean: true},
				},
			},
			verbose: false,
			wantErr: false,
		},
		{
			name: "multiple worktrees with various status",
			worktrees: []git.WorktreeInfo{
				{
					Path:         "/repo/main",
					Branch:       "main",
					IsCurrent:    true,
					LastActivity: now.Add(-1 * time.Hour),
					Status:       git.WorktreeStatus{IsClean: true},
					Remote:       git.RemoteStatus{HasRemote: true, Ahead: 2},
				},
				{
					Path:         "/repo/feature-very-long-name-that-affects-column-width",
					Branch:       "feature/very-long-branch-name-for-testing-column-width",
					IsCurrent:    false,
					LastActivity: now.Add(-2 * time.Hour),
					Status:       git.WorktreeStatus{Modified: 3, Staged: 1, Untracked: 2, IsClean: false},
					Remote:       git.RemoteStatus{HasRemote: true, Behind: 1},
				},
				{
					Path:         "/repo/old",
					Branch:       "old",
					IsCurrent:    false,
					LastActivity: now.Add(-30 * 24 * time.Hour), // 30 days ago
					Status:       git.WorktreeStatus{IsClean: true},
					Remote:       git.RemoteStatus{IsMerged: true},
				},
			},
			verbose: false,
			wantErr: false,
		},
		{
			name: "verbose output with complex status",
			worktrees: []git.WorktreeInfo{
				{
					Path:         "/repo/complex",
					Branch:       "feature/complex-status",
					IsCurrent:    false,
					LastActivity: now.Add(-5 * time.Minute),
					Status: git.WorktreeStatus{
						Modified:  5,
						Staged:    3,
						Untracked: 2,
						IsClean:   false,
					},
					Remote: git.RemoteStatus{
						HasRemote: true,
						Ahead:     3,
						Behind:    2,
					},
				},
			},
			verbose: true,
			wantErr: false,
		},
		{
			name: "worktree with special characters in path",
			worktrees: []git.WorktreeInfo{
				{
					Path:         "/repo/feature-with-special-chars",
					Branch:       "feature/test-123_special",
					IsCurrent:    false,
					LastActivity: now,
					Status:       git.WorktreeStatus{IsClean: true},
				},
			},
			verbose: false,
			wantErr: false,
		},
		{
			name: "worktree with zero activity time",
			worktrees: []git.WorktreeInfo{
				{
					Path:         "/repo/unknown",
					Branch:       "unknown",
					IsCurrent:    false,
					LastActivity: time.Time{}, // Zero time
					Status:       git.WorktreeStatus{IsClean: true},
				},
			},
			verbose: false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			presenter := NewListPresenter()
			err := presenter.DisplayHuman(tt.worktrees, tt.verbose)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDisplayHumanOutput_ColumnWidthCalculation(t *testing.T) {
	// Test that column widths are calculated correctly for various input sizes
	worktrees := []git.WorktreeInfo{
		{
			Path:         "/repo/short",
			Branch:       "a",
			IsCurrent:    true,
			LastActivity: time.Now(),
			Status:       git.WorktreeStatus{IsClean: true},
		},
		{
			Path:         "/repo/this-is-a-very-long-worktree-name-that-should-expand-column-width",
			Branch:       "this-is-also-a-very-long-branch-name-that-should-expand-column-width-significantly",
			IsCurrent:    false,
			LastActivity: time.Now().Add(-1 * time.Hour),
			Status:       git.WorktreeStatus{IsClean: true},
		},
	}

	// This test mainly verifies that the function handles extreme column widths without errors
	presenter := NewListPresenter()
	err := presenter.DisplayHuman(worktrees, false)
	assert.NoError(t, err)

	// Test verbose mode as well
	err = presenter.DisplayHuman(worktrees, true)
	assert.NoError(t, err)
}

func TestDisplayHumanOutput_EmptyWorktreesMessage(t *testing.T) {
	// Test that empty worktrees shows appropriate message
	presenter := NewListPresenter()
	err := presenter.DisplayHuman([]git.WorktreeInfo{}, false)
	assert.NoError(t, err)

	err = presenter.DisplayHuman([]git.WorktreeInfo{}, true)
	assert.NoError(t, err)
}

func TestRunListCommand_EdgeCases(t *testing.T) {
	// Save original executor
	originalExecutor := git.DefaultExecutor
	defer func() { git.DefaultExecutor = originalExecutor }()

	tests := []struct {
		name        string
		options     *ListOptions
		gitResponse string
		gitError    error
		wantErr     bool
		expectedErr string
	}{
		{
			name:        "git command fails",
			options:     &ListOptions{Sort: SortByActivity},
			gitResponse: "",
			gitError:    assert.AnError,
			wantErr:     true,
			expectedErr: "failed to list worktrees",
		},
		{
			name:        "malformed git output",
			options:     &ListOptions{Sort: SortByActivity},
			gitResponse: "invalid\noutput\nformat",
			gitError:    nil,
			wantErr:     false, // Should handle gracefully
		},
		{
			name: "stale filter with zero days",
			options: &ListOptions{
				Sort:      SortByActivity,
				StaleOnly: true,
				StaleDays: 0,
			},
			gitResponse: "",
			gitError:    nil,
			wantErr:     false,
		},
		{
			name: "multiple conflicting filters",
			options: &ListOptions{
				DirtyOnly: true,
				CleanOnly: true,
				StaleOnly: true,
			},
			wantErr:     true,
			expectedErr: "Cannot specify multiple filters",
		},
		{
			name: "porcelain output with complex data",
			options: &ListOptions{
				Sort:      SortByStatus,
				Porcelain: true,
			},
			gitResponse: "worktree /repo/main\nbranch main\nHEAD abc123\n\nworktree /repo/feature\nbranch feature/test\nHEAD def456\n",
			gitError:    nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			setupWorktreeListMock(mockExecutor, tt.gitResponse, tt.gitError)
			git.DefaultExecutor = mockExecutor

			err := runListCommand(tt.options)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectedErr != "" {
					assert.Contains(t, err.Error(), tt.expectedErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSortWorktrees_EdgeCases(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name      string
		worktrees []git.WorktreeInfo
		sortBy    ListSortOption
		expected  []string // Expected order by path
	}{
		{
			name:      "empty list",
			worktrees: []git.WorktreeInfo{},
			sortBy:    SortByActivity,
			expected:  []string{},
		},
		{
			name: "single item",
			worktrees: []git.WorktreeInfo{
				{Path: "/repo/single", IsCurrent: false, LastActivity: now},
			},
			sortBy:   SortByName,
			expected: []string{"/repo/single"},
		},
		{
			name: "all current worktrees (edge case)",
			worktrees: []git.WorktreeInfo{
				{Path: "/repo/first", IsCurrent: true, LastActivity: now.Add(-1 * time.Hour)},
				{Path: "/repo/second", IsCurrent: true, LastActivity: now.Add(-2 * time.Hour)},
			},
			sortBy:   SortByActivity,
			expected: []string{"/repo/first", "/repo/second"}, // Sort by activity (most recent first)
		},
		{
			name: "same activity times - unstable sort",
			worktrees: []git.WorktreeInfo{
				{Path: "/repo/zebra", IsCurrent: false, LastActivity: now, Branch: "zebra"},
				{Path: "/repo/alpha", IsCurrent: false, LastActivity: now, Branch: "alpha"},
			},
			sortBy:   SortByActivity,
			expected: []string{"/repo/zebra", "/repo/alpha"}, // Go's sort is not stable for equal elements
		},
		{
			name: "same status (all clean) - unstable sort",
			worktrees: []git.WorktreeInfo{
				{Path: "/repo/zebra", IsCurrent: false, Status: git.WorktreeStatus{IsClean: true}, Branch: "zebra", LastActivity: now.Add(-1 * time.Hour)},
				{Path: "/repo/alpha", IsCurrent: false, Status: git.WorktreeStatus{IsClean: true}, Branch: "alpha", LastActivity: now.Add(-2 * time.Hour)},
			},
			sortBy:   SortByStatus,
			expected: []string{"/repo/zebra", "/repo/alpha"}, // Sorts by activity when status is same
		},
		{
			name: "invalid sort option - no sorting applied",
			worktrees: []git.WorktreeInfo{
				{Path: "/repo/old", IsCurrent: false, LastActivity: now.Add(-1 * time.Hour)},
				{Path: "/repo/new", IsCurrent: false, LastActivity: now},
			},
			sortBy:   ListSortOption("invalid"),
			expected: []string{"/repo/old", "/repo/new"}, // Original order preserved
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees := make([]git.WorktreeInfo, len(tt.worktrees))
			copy(worktrees, tt.worktrees)

			service := &ListService{}
			service.sortWorktrees(worktrees, tt.sortBy)

			result := make([]string, len(worktrees))
			for i, wt := range worktrees {
				result[i] = wt.Path
			}

			assert.Equal(t, tt.expected, result)
		})
	}
}
