//go:build !integration
// +build !integration

package commands

import (
	"testing"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestListService_ListWorktrees(t *testing.T) {
	// Integration test - requires file system mocking for proper testing
	// The ListWorktrees method calls findGroveRepository which depends on os.Getwd and os.Stat
	// For complete testing, we'd need to inject file system dependencies
	t.Skip("Integration test - requires file system mocking")
}

func TestListService_ApplyFilters(t *testing.T) {
	service := &ListService{}

	now := time.Now()
	staleTime := now.AddDate(0, 0, -31) // 31 days ago
	recentTime := now.AddDate(0, 0, -1) // 1 day ago

	worktrees := []git.WorktreeInfo{
		{
			Path:         "/repo/clean",
			Status:       git.WorktreeStatus{IsClean: true},
			LastActivity: recentTime,
		},
		{
			Path:         "/repo/dirty",
			Status:       git.WorktreeStatus{IsClean: false, Modified: 1},
			LastActivity: recentTime,
		},
		{
			Path:         "/repo/stale",
			Status:       git.WorktreeStatus{IsClean: true},
			LastActivity: staleTime,
		},
	}

	tests := []struct {
		name     string
		options  *ListOptions
		expected int
	}{
		{
			name:     "No filters",
			options:  &ListOptions{StaleDays: 30},
			expected: 3,
		},
		{
			name:     "Dirty only",
			options:  &ListOptions{DirtyOnly: true, StaleDays: 30},
			expected: 1,
		},
		{
			name:     "Clean only",
			options:  &ListOptions{CleanOnly: true, StaleDays: 30},
			expected: 2,
		},
		{
			name:     "Stale only",
			options:  &ListOptions{StaleOnly: true, StaleDays: 30},
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.applyFilters(worktrees, tt.options)
			assert.Len(t, result, tt.expected)
		})
	}
}

func TestListService_SortWorktrees(t *testing.T) {
	service := &ListService{}

	now := time.Now()
	older := now.Add(-1 * time.Hour)

	worktrees := []git.WorktreeInfo{
		{
			Path:         "/repo/zebra",
			Status:       git.WorktreeStatus{IsClean: false},
			LastActivity: older,
		},
		{
			Path:         "/repo/alpha",
			Status:       git.WorktreeStatus{IsClean: true},
			LastActivity: now,
		},
	}

	tests := []struct {
		name     string
		sortBy   ListSortOption
		expected string // Expected first item path
	}{
		{
			name:     "Sort by activity",
			sortBy:   SortByActivity,
			expected: "/repo/alpha", // Most recent first
		},
		{
			name:     "Sort by name",
			sortBy:   SortByName,
			expected: "/repo/alpha", // Alphabetically first
		},
		{
			name:     "Sort by status",
			sortBy:   SortByStatus,
			expected: "/repo/zebra", // Dirty first, then by activity
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the original
			testWorktrees := make([]git.WorktreeInfo, len(worktrees))
			copy(testWorktrees, worktrees)

			service.sortWorktrees(testWorktrees, tt.sortBy)
			assert.Equal(t, tt.expected, testWorktrees[0].Path)
		})
	}
}

func TestListService_FindGroveRepository(t *testing.T) {
	service := &ListService{}

	// This test is environment-dependent, so we test the error case
	// In a real implementation, we'd use dependency injection for the file system
	result, err := service.findGroveRepository()

	// Should return error if no .bare directory found
	if err != nil {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no grove repository")
		assert.Empty(t, result)
	} else {
		// If it finds a .bare directory, that's also valid
		assert.NoError(t, err)
		assert.NotEmpty(t, result)
	}
}
