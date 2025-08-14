package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/commands/shared"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
)

type ListSortOption string

const (
	SortByActivity ListSortOption = "activity"
	SortByName     ListSortOption = "name"
	SortByStatus   ListSortOption = "status"

	DefaultSortOption = SortByActivity
)

type ListOptions struct {
	Sort      ListSortOption
	Verbose   bool
	Porcelain bool
	DirtyOnly bool
	StaleOnly bool
	CleanOnly bool
	StaleDays int
}

func NewListCmd() *cobra.Command {
	options := &ListOptions{
		Sort:      DefaultSortOption,
		StaleDays: shared.DefaultStaleDays,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all worktrees with status information",
		Long: `List all Git worktrees in the repository with comprehensive status information.

Shows each worktree with its branch, status, last activity, and remote tracking information.
The current worktree is marked with an asterisk (*).

Status indicators:
  ✓ clean     - No uncommitted changes
  ⚠ dirty     - Has uncommitted changes (shows M/S/U counts)
  ↑N ↓M       - N commits ahead, M commits behind remote
  merged      - Branch has been merged

Examples:
  grove list                    # List all worktrees sorted by activity
  grove list --sort=name        # Sort alphabetically by worktree name
  grove list --dirty            # Show only worktrees with changes
  grove list --stale --days=14  # Show worktrees unused for 14+ days
  grove list --verbose          # Show extended information including paths
  grove list --porcelain        # Machine-readable output

Sorting options: activity (default), name, status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := runListCommand(options); err != nil {
				cmd.SilenceUsage = true
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar((*string)(&options.Sort), "sort", "activity", "Sort by: activity, name, status")
	cmd.Flags().BoolVarP(&options.Verbose, "verbose", "v", false, "Show extended information including full paths")
	cmd.Flags().BoolVar(&options.Porcelain, "porcelain", false, "Machine-readable output")
	cmd.Flags().BoolVar(&options.DirtyOnly, "dirty", false, "Show only worktrees with uncommitted changes")
	cmd.Flags().BoolVar(&options.StaleOnly, "stale", false, "Show only stale worktrees (unused for specified days)")
	cmd.Flags().BoolVar(&options.CleanOnly, "clean", false, "Show only clean worktrees (no uncommitted changes)")
	cmd.Flags().IntVar(&options.StaleDays, "days", shared.DefaultStaleDays, "Number of days to consider a worktree stale (used with --stale)")

	return cmd
}

func runListCommand(options *ListOptions) error {
	return runListCommandWithExecutor(shared.DefaultExecutorProvider.GetExecutor(), options)
}

func runListCommandWithExecutor(executor git.GitExecutor, options *ListOptions) error {
	if err := validateListOptions(options); err != nil {
		return err
	}

	service := NewListService(executor)
	presenter := NewListPresenter()

	worktrees, err := service.ListWorktrees(options)
	if err != nil {
		return err
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	if options.Porcelain {
		return presenter.DisplayPorcelain(worktrees)
	}
	return presenter.DisplayHuman(worktrees, options.Verbose)
}

func validateListOptions(options *ListOptions) error {
	// Check for conflicting filters with specific error messages
	if options.DirtyOnly && options.CleanOnly {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"cannot use --dirty and --clean flags together",
			nil,
		)
	}
	if options.DirtyOnly && options.StaleOnly {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"cannot use --dirty and --stale flags together",
			nil,
		)
	}
	if options.CleanOnly && options.StaleOnly {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"cannot use --clean and --stale flags together",
			nil,
		)
	}

	// Check for conflicting output formats
	if options.Porcelain && options.Verbose {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"cannot use --porcelain and --verbose flags together",
			nil,
		)
	}

	// Validate stale days is positive when stale filter is used
	if options.StaleOnly && options.StaleDays < 0 {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"stale days must be positive",
			nil,
		)
	}

	// Validate sort option
	validSorts := []ListSortOption{SortByActivity, SortByName, SortByStatus}
	validSort := false
	for _, valid := range validSorts {
		if options.Sort == valid {
			validSort = true
			break
		}
	}
	if !validSort {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"invalid sort option",
			nil,
		)
	}

	return nil
}
