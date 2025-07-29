package remove

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
)

// NewRemoveCmd creates and returns the remove command with all necessary flags and subcommands.
func NewRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove [worktree-path]",
		Short: "Remove worktrees safely with optional branch cleanup",
		Long: `Remove Git worktrees with comprehensive safety checks and optional branch cleanup.

Grove's remove command provides intelligent worktree cleanup that prevents accidental data loss
while offering powerful bulk operations for efficient repository maintenance.

Examples:
  # Remove a specific worktree with safety checks
  grove remove /path/to/worktree

  # Force remove with uncommitted changes (with confirmation)
  grove remove /path/to/worktree --force

  # Preview what would be removed without actual deletion
  grove remove /path/to/worktree --dry-run

  # Remove worktree and its associated branch
  grove remove /path/to/worktree --delete-branch

  # Remove all merged worktrees
  grove remove --merged

  # Remove worktrees with no activity for 30+ days
  grove remove --stale --days=30

  # Remove all non-current worktrees (with confirmation)
  grove remove --all`,
		Args:              cobra.MaximumNArgs(1),
		RunE:              runRemove,
		ValidArgsFunction: completeWorktreePaths,
	}

	// Individual removal flags.
	cmd.Flags().BoolVar(&removeOptions.Force, "force", false, "Remove worktree even with uncommitted changes (prompts for confirmation)")
	cmd.Flags().BoolVar(&removeOptions.DryRun, "dry-run", false, "Show what would be removed without actual deletion")
	cmd.Flags().BoolVar(&removeOptions.DeleteBranch, "delete-branch", false, "Also delete the associated branch")

	// Bulk removal flags.
	cmd.Flags().BoolVar(&bulkCriteria.Merged, "merged", false, "Remove all worktrees with merged branches")
	cmd.Flags().BoolVar(&bulkCriteria.Stale, "stale", false, "Remove worktrees with no recent activity")
	cmd.Flags().BoolVar(&bulkCriteria.All, "all", false, "Remove all non-current worktrees")
	cmd.Flags().IntVar(&bulkCriteria.DaysOld, "days", DefaultStaleDaysThreshold, "Minimum age in days for --stale operations")

	return cmd
}

var (
	removeOptions RemoveOptions
	bulkCriteria  BulkCriteria
)

func runRemove(cmd *cobra.Command, args []string) error {
	executor := git.DefaultExecutor
	log := logger.WithComponent("remove_command")

	// Initialize remove service with dependencies.
	service := NewRemoveServiceImpl(executor, log)

	// Handle bulk operations.
	if bulkCriteria.Merged || bulkCriteria.Stale || bulkCriteria.All {
		results, err := service.RemoveBulk(bulkCriteria, removeOptions)
		if err != nil {
			return err
		}
		presentBulkResults(&results)
		return nil
	}

	// Handle single worktree removal.
	if len(args) == 0 {
		return cmd.Help()
	}

	worktreePath := args[0]
	err := service.RemoveWorktree(worktreePath, removeOptions)
	if err != nil {
		// Check if this is a branch deletion warning rather than a failure.
		if strings.Contains(err.Error(), "worktree removed successfully, but branch deletion failed") {
			fmt.Printf("Warning: %v\n", err)
			fmt.Printf("Successfully removed worktree: %s\n", worktreePath)
			return nil
		}
		return err
	}

	// Success message for normal completion.
	fmt.Printf("Successfully removed worktree: %s\n", worktreePath)
	return nil
}

func completeWorktreePaths(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	executor := git.DefaultExecutor
	worktrees, err := git.ListWorktrees(executor)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var suggestions []string
	for i := range worktrees {
		// Don't suggest current worktree for removal.
		if !worktrees[i].IsCurrent {
			suggestions = append(suggestions, worktrees[i].Path)
		}
	}

	return suggestions, cobra.ShellCompDirectiveDefault
}

func presentBulkResults(results *RemoveResults) {
	if !results.HasResults() {
		fmt.Println("No worktrees were processed.")
		return
	}

	// Present summary.
	fmt.Printf("\nBulk removal completed:\n")
	fmt.Printf("  Removed: %d\n", len(results.Removed))
	fmt.Printf("  Skipped: %d\n", len(results.Skipped))
	fmt.Printf("  Failed:  %d\n", len(results.Failed))

	// Show removed worktrees.
	if len(results.Removed) > 0 {
		fmt.Printf("\nSuccessfully removed:\n")
		for _, path := range results.Removed {
			fmt.Printf("  ✓ %s\n", path)
		}
	}

	// Show skipped worktrees.
	if len(results.Skipped) > 0 {
		fmt.Printf("\nSkipped:\n")
		for _, skip := range results.Skipped {
			fmt.Printf("  - %s (reason: %s)\n", skip.Path, skip.Reason)
		}
	}

	// Show failed worktrees.
	if len(results.Failed) > 0 {
		fmt.Printf("\nFailed:\n")
		for _, fail := range results.Failed {
			fmt.Printf("  ✗ %s (error: %v)\n", fail.Path, fail.Error)
		}
	}

	// Show performance metrics.
	fmt.Printf("\nSuccess rate: %.1f%%\n", (&results.Summary).SuccessRate())
}
