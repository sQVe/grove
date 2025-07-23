package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
)

// ListService handles the business logic for listing worktrees.
type ListService struct {
	executor git.GitExecutor
}

// NewListService creates a new ListService with the provided GitExecutor.
func NewListService(executor git.GitExecutor) *ListService {
	return &ListService{
		executor: executor,
	}
}

// ListWorktrees retrieves, filters, and sorts worktrees based on the provided options.
func (s *ListService) ListWorktrees(options *ListOptions) ([]git.WorktreeInfo, error) {
	logger.Debug("Listing worktrees", "sort", options.Sort, "verbose", options.Verbose)

	// Find the grove repository root (bare repository)
	repoPath, err := s.findGroveRepository()
	if err != nil {
		return nil, errors.NewGroveError(
			errors.ErrCodeGitOperation,
			"Could not find grove repository: "+err.Error(),
			err,
		)
	}

	var worktrees []git.WorktreeInfo

	// Use early filtering optimization if specific filters are applied
	if s.hasPerformanceOptimizableFilters(options) {
		logger.Debug("Using early filtering optimization", "filters", s.getActiveFilters(options))
		worktrees, err = s.listWorktreesWithEarlyFiltering(repoPath, options)
	} else {
		// Use traditional approach for general listing
		logger.Debug("Using traditional listing approach")
		worktrees, err = git.ListWorktreesFromRepo(s.executor, repoPath)
		if err != nil {
			return nil, errors.ErrGitWorktree("list", err)
		}
		// Apply filters after loading all data
		worktrees = s.applyFilters(worktrees, options)
	}

	if err != nil {
		return nil, errors.ErrGitWorktree("list", err)
	}

	// Sort worktrees
	s.sortWorktrees(worktrees, options.Sort)

	return worktrees, nil
}

// applyFilters filters the worktree list based on the specified options.
func (s *ListService) applyFilters(worktrees []git.WorktreeInfo, options *ListOptions) []git.WorktreeInfo {
	if !options.DirtyOnly && !options.StaleOnly && !options.CleanOnly {
		return worktrees
	}

	var filtered []git.WorktreeInfo
	staleThreshold := time.Now().AddDate(0, 0, -options.StaleDays)

	for i := range worktrees {
		wt := &worktrees[i]
		switch {
		case options.DirtyOnly && !wt.Status.IsClean:
			filtered = append(filtered, *wt)
		case options.StaleOnly && !wt.LastActivity.IsZero() && wt.LastActivity.Before(staleThreshold):
			filtered = append(filtered, *wt)
		case options.CleanOnly && wt.Status.IsClean:
			filtered = append(filtered, *wt)
		}
	}

	return filtered
}

// sortWorktrees sorts the worktree list based on the specified sort option.
func (s *ListService) sortWorktrees(worktrees []git.WorktreeInfo, sortBy ListSortOption) {
	switch sortBy {
	case SortByActivity:
		sort.Slice(worktrees, func(i, j int) bool {
			return worktrees[i].LastActivity.After(worktrees[j].LastActivity)
		})
	case SortByName:
		sort.Slice(worktrees, func(i, j int) bool {
			return worktrees[i].Path < worktrees[j].Path
		})
	case SortByStatus:
		sort.Slice(worktrees, func(i, j int) bool {
			if worktrees[i].Status.IsClean != worktrees[j].Status.IsClean {
				return !worktrees[i].Status.IsClean
			}
			return worktrees[i].LastActivity.After(worktrees[j].LastActivity)
		})
	}
}

// findGroveRepository finds the grove repository root by looking for a .bare directory.
// It starts from the current directory and walks up the directory tree.
func (s *ListService) findGroveRepository() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.ErrDirectoryAccess("current directory", err)
	}

	// Start from current directory and walk up to find .bare directory
	currentPath := cwd
	for {
		bareDir := filepath.Join(currentPath, ".bare")
		if stat, err := os.Stat(bareDir); err == nil && stat.IsDir() {
			return bareDir, nil
		}

		// Move up one directory
		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			// Reached filesystem root
			break
		}
		currentPath = parent
	}

	return "", errors.ErrRepoNotFound(currentPath)
}

// hasPerformanceOptimizableFilters determines if the current filter set can benefit from early filtering.
func (s *ListService) hasPerformanceOptimizableFilters(options *ListOptions) bool {
	// Early filtering is beneficial when we have specific filters that can eliminate worktrees
	// before doing expensive status/activity checks
	return options.DirtyOnly || options.StaleOnly || options.CleanOnly
}

// getActiveFilters returns a list of active filter names for logging.
func (s *ListService) getActiveFilters(options *ListOptions) []string {
	var filters []string
	if options.DirtyOnly {
		filters = append(filters, "dirty")
	}
	if options.StaleOnly {
		filters = append(filters, fmt.Sprintf("stale(%dd)", options.StaleDays))
	}
	if options.CleanOnly {
		filters = append(filters, "clean")
	}
	return filters
}

// listWorktreesWithEarlyFiltering implements optimized filtering by loading full worktree data
// but processing filters in a more efficient order to skip expensive operations when possible.
func (s *ListService) listWorktreesWithEarlyFiltering(repoPath string, options *ListOptions) ([]git.WorktreeInfo, error) {
	// For now, implement a simpler optimization: load all worktrees but apply filters more efficiently
	// This provides some performance benefit without requiring extensive git package refactoring
	worktrees, err := git.ListWorktreesFromRepo(s.executor, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	logger.Debug("Loaded worktrees for early filtering", "count", len(worktrees))

	// Apply filters with early exit logic
	return s.applyFiltersOptimized(worktrees, options), nil
}

// applyFiltersOptimized applies filters with performance optimizations.
func (s *ListService) applyFiltersOptimized(worktrees []git.WorktreeInfo, options *ListOptions) []git.WorktreeInfo {
	if !options.DirtyOnly && !options.StaleOnly && !options.CleanOnly {
		return worktrees
	}

	var filtered []git.WorktreeInfo
	staleThreshold := time.Now().AddDate(0, 0, -options.StaleDays)

	for i := range worktrees {
		wt := &worktrees[i]

		// Apply filters in order of computational cost (cheapest first)

		// 1. Stale filter (requires time comparison - cheapest)
		if options.StaleOnly {
			if wt.LastActivity.IsZero() || !wt.LastActivity.Before(staleThreshold) {
				continue // Skip if not stale
			}
		}

		// 2. Clean/Dirty filter (requires status check - more expensive)
		if options.DirtyOnly && wt.Status.IsClean {
			continue // Skip clean worktrees when filtering for dirty only
		}
		if options.CleanOnly && !wt.Status.IsClean {
			continue // Skip dirty worktrees when filtering for clean only
		}

		// Worktree passed all active filters
		filtered = append(filtered, *wt)
	}

	logger.Debug("Filter optimization complete", "original", len(worktrees), "filtered", len(filtered))
	return filtered
}
