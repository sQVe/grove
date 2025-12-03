package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

// NewRemoveCmd creates the remove command
func NewRemoveCmd() *cobra.Command {
	var force bool
	var deleteBranch bool

	cmd := &cobra.Command{
		Use:   "remove <worktree>",
		Short: "Remove a worktree",
		Long: `Remove a worktree directory. Optionally delete the branch as well.
Accepts worktree name (directory) or branch name.

Examples:
  grove remove feature-auth        # Remove worktree only
  grove remove --branch feature    # Remove worktree and delete branch
  grove remove --force dirty-work  # Force remove even if dirty/locked`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeRemoveArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args[0], force, deleteBranch)
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Remove even if dirty or locked")
	cmd.Flags().BoolVar(&deleteBranch, "branch", false, "Also delete the branch")
	cmd.Flags().BoolP("help", "h", false, "Help for remove")

	return cmd
}

func runRemove(target string, force, deleteBranch bool) error {
	target = strings.TrimSpace(target)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktreeInfo := git.FindWorktree(infos, target)
	if worktreeInfo == nil {
		return fmt.Errorf("worktree not found: %s", target)
	}

	// Check if user is inside the worktree being deleted
	if cwd == worktreeInfo.Path || strings.HasPrefix(cwd, worktreeInfo.Path+string(filepath.Separator)) {
		return fmt.Errorf("cannot delete current worktree; switch to a different worktree first")
	}

	// Check worktree state unless --force
	if !force {
		// Check dirty state
		hasChanges, _, err := git.CheckGitChanges(worktreeInfo.Path)
		if err != nil {
			return fmt.Errorf("failed to check worktree status: %w", err)
		}
		if hasChanges {
			return fmt.Errorf("worktree has uncommitted changes; use --force to remove anyway")
		}

		// Check locked state
		if git.IsWorktreeLocked(worktreeInfo.Path) {
			return fmt.Errorf("worktree is locked; use --force to remove anyway")
		}
	} else if git.IsWorktreeLocked(worktreeInfo.Path) {
		// Unlock worktree first if locked (git requires double force otherwise)
		if err := git.UnlockWorktree(bareDir, worktreeInfo.Path); err != nil {
			logger.Debug("Failed to unlock worktree: %v", err)
		}
	}

	// Get sync status BEFORE removing worktree if we need to warn about unpushed commits
	// (fast mode doesn't fetch Ahead count, so we must check explicitly)
	var aheadCount int
	if deleteBranch {
		syncStatus := git.GetSyncStatus(worktreeInfo.Path)
		aheadCount = syncStatus.Ahead
	}

	// Remove the worktree
	if err := git.RemoveWorktree(bareDir, worktreeInfo.Path, force); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Optionally delete the branch
	if deleteBranch {
		// Warn about unpushed commits
		if aheadCount > 0 {
			logger.Warning("Branch has %d unpushed commit(s)", aheadCount)
		}

		if err := git.DeleteBranch(bareDir, worktreeInfo.Branch, force); err != nil {
			return fmt.Errorf("worktree removed but failed to delete branch: %w", err)
		}
		logger.Success("Deleted worktree and branch %s", target)
	} else {
		logger.Success("Deleted worktree %s", target)
	}

	return nil
}

func completeRemoveArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete first argument
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, info := range infos {
		// Exclude current worktree
		if cwd != info.Path && !strings.HasPrefix(cwd, info.Path+string(os.PathSeparator)) {
			completions = append(completions, filepath.Base(info.Path))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
