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
		Use:   "remove <branch>",
		Short: "Remove a worktree",
		Long: `Remove a worktree directory. Optionally delete the branch as well.

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

func runRemove(branch string, force, deleteBranch bool) error {
	branch = strings.TrimSpace(branch)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Find the worktree for the branch
	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktreeInfo *git.WorktreeInfo
	for _, info := range infos {
		if info.Branch == branch {
			worktreeInfo = info
			break
		}
	}

	if worktreeInfo == nil {
		return fmt.Errorf("no worktree found for branch %q", branch)
	}

	// Check if user is inside the worktree being deleted
	if cwd == worktreeInfo.Path || strings.HasPrefix(cwd, worktreeInfo.Path+string(filepath.Separator)) {
		return fmt.Errorf("cannot delete current worktree; switch to a different worktree first")
	}

	worktreeName := filepath.Base(worktreeInfo.Path)

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
		if git.IsWorktreeLocked(bareDir, worktreeName) {
			return fmt.Errorf("worktree is locked; use --force to remove anyway")
		}
	} else if git.IsWorktreeLocked(bareDir, worktreeName) {
		// Unlock worktree first if locked (git requires double force otherwise)
		if err := git.UnlockWorktree(bareDir, worktreeInfo.Path); err != nil {
			logger.Debug("Failed to unlock worktree: %v", err)
		}
	}

	// Remove the worktree
	if err := git.RemoveWorktree(bareDir, worktreeInfo.Path, force); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Optionally delete the branch
	if deleteBranch {
		// Warn about unpushed commits
		if worktreeInfo.Ahead > 0 {
			logger.Warning("Branch has %d unpushed commit(s)", worktreeInfo.Ahead)
		}

		if err := git.DeleteBranch(bareDir, branch, force); err != nil {
			return fmt.Errorf("worktree removed but failed to delete branch: %w", err)
		}
		logger.Success("Deleted worktree and branch %s", branch)
	} else {
		logger.Success("Deleted worktree %s", branch)
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
			completions = append(completions, info.Branch)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
