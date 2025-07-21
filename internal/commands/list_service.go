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

	// Get worktree information
	worktrees, err := git.ListWorktreesFromRepo(s.executor, repoPath)
	if err != nil {
		return nil, errors.ErrGitWorktree("list", err)
	}

	// Apply filters
	filteredWorktrees := s.applyFilters(worktrees, options)

	// Sort worktrees
	s.sortWorktrees(filteredWorktrees, options.Sort)

	return filteredWorktrees, nil
}

// applyFilters filters the worktree list based on the specified options.
func (s *ListService) applyFilters(worktrees []git.WorktreeInfo, options *ListOptions) []git.WorktreeInfo {
	if !options.DirtyOnly && !options.StaleOnly && !options.CleanOnly {
		return worktrees
	}

	var filtered []git.WorktreeInfo
	staleThreshold := time.Now().AddDate(0, 0, -options.StaleDays)

	for _, wt := range worktrees {
		switch {
		case options.DirtyOnly && !wt.Status.IsClean:
			filtered = append(filtered, wt)
		case options.StaleOnly && !wt.LastActivity.IsZero() && wt.LastActivity.Before(staleThreshold):
			filtered = append(filtered, wt)
		case options.CleanOnly && wt.Status.IsClean:
			filtered = append(filtered, wt)
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
		return "", fmt.Errorf("failed to get current directory: %w", err)
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

	return "", fmt.Errorf("no grove repository (.bare directory) found")
}