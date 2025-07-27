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

type ListService struct {
	executor git.GitExecutor
}

func NewListService(executor git.GitExecutor) *ListService {
	return &ListService{
		executor: executor,
	}
}

func (s *ListService) ListWorktrees(options *ListOptions) ([]git.WorktreeInfo, error) {
	logger.Debug("Listing worktrees", "sort", options.Sort, "verbose", options.Verbose)

	repoPath, err := s.findGroveRepository()
	if err != nil {
		return nil, errors.NewGroveError(
			errors.ErrCodeGitOperation,
			"Could not find grove repository: "+err.Error(),
			err,
		)
	}

	var worktrees []git.WorktreeInfo

	if s.hasPerformanceOptimizableFilters(options) {
		logger.Debug("Using early filtering optimization", "filters", s.getActiveFilters(options))
		worktrees, err = s.listWorktreesWithEarlyFiltering(repoPath, options)
	} else {
		logger.Debug("Using traditional listing approach")
		worktrees, err = git.ListWorktreesFromRepo(s.executor, repoPath)
		if err != nil {
			return nil, errors.ErrGitWorktree("list", err)
		}
		worktrees = s.applyFilters(worktrees, options)
	}

	if err != nil {
		return nil, errors.ErrGitWorktree("list", err)
	}

	s.sortWorktrees(worktrees, options.Sort)

	return worktrees, nil
}

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

func (s *ListService) findGroveRepository() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", errors.ErrDirectoryAccess("current directory", err)
	}

	// Start from current directory and walk up to find .bare directory.
	currentPath := cwd
	for {
		bareDir := filepath.Join(currentPath, ".bare")
		if stat, err := os.Stat(bareDir); err == nil && stat.IsDir() {
			return bareDir, nil
		}

		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			break
		}
		currentPath = parent
	}

	return "", errors.ErrRepoNotFound(currentPath)
}

func (s *ListService) hasPerformanceOptimizableFilters(options *ListOptions) bool {
	// Early filtering is beneficial when we have specific filters that can
	// eliminate worktrees, saving us from doing expensive status/activity
	// checks.
	return options.DirtyOnly || options.StaleOnly || options.CleanOnly
}

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

func (s *ListService) listWorktreesWithEarlyFiltering(repoPath string, options *ListOptions) ([]git.WorktreeInfo, error) {
	// For now, implement a simpler optimization: load all worktrees but apply filters more efficiently.
	// This provides some performance benefit without requiring extensive git package refactoring.
	worktrees, err := git.ListWorktreesFromRepo(s.executor, repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	logger.Debug("Loaded worktrees for early filtering", "count", len(worktrees))

	return s.applyFiltersOptimized(worktrees, options), nil
}

func (s *ListService) applyFiltersOptimized(worktrees []git.WorktreeInfo, options *ListOptions) []git.WorktreeInfo {
	if !options.DirtyOnly && !options.StaleOnly && !options.CleanOnly {
		return worktrees
	}

	var filtered []git.WorktreeInfo
	staleThreshold := time.Now().AddDate(0, 0, -options.StaleDays)

	for i := range worktrees {
		wt := &worktrees[i]

		// Apply filters in order of computational cost (cheapest first).

		// 1. Stale filter (requires time comparison - cheapest).
		if options.StaleOnly {
			if wt.LastActivity.IsZero() || !wt.LastActivity.Before(staleThreshold) {
				continue
			}
		}

		// 2. Clean/Dirty filter (requires status check - more expensive).
		if options.DirtyOnly && wt.Status.IsClean {
			continue
		}
		if options.CleanOnly && !wt.Status.IsClean {
			continue
		}

		filtered = append(filtered, *wt)
	}

	logger.Debug("Filter optimization complete", "original", len(worktrees), "filtered", len(filtered))
	return filtered
}
