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

// NewLockCmd creates the lock command
func NewLockCmd() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "lock <branch>",
		Short: "Lock a worktree to prevent removal",
		Long: `Lock a worktree to prevent it from being removed by prune or remove commands.

Locked worktrees are protected from accidental deletion. Use unlock to remove the lock.

Examples:
  grove lock feature-auth                    # Lock worktree
  grove lock feature-auth --reason "WIP"     # Lock with reason`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeLockArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLock(args[0], reason)
		},
	}

	cmd.Flags().StringVarP(&reason, "reason", "r", "", "Reason for locking")
	cmd.Flags().BoolP("help", "h", false, "Help for lock")

	return cmd
}

func runLock(branch, reason string) error {
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

	worktreeName := filepath.Base(worktreeInfo.Path)

	// Check if already locked
	if git.IsWorktreeLocked(bareDir, worktreeName) {
		existingReason := git.GetWorktreeLockReason(bareDir, worktreeName)
		if existingReason != "" {
			return fmt.Errorf("worktree is already locked: %q", existingReason)
		}
		return fmt.Errorf("worktree is already locked")
	}

	// Lock the worktree
	if err := git.LockWorktree(bareDir, worktreeInfo.Path, reason); err != nil {
		return fmt.Errorf("failed to lock worktree: %w", err)
	}

	if reason != "" {
		logger.Success("Locked worktree %s (%s)", branch, reason)
	} else {
		logger.Success("Locked worktree %s", branch)
	}

	return nil
}

func completeLockArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
		// Only include non-locked worktrees
		if !info.Locked {
			completions = append(completions, info.Branch)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
