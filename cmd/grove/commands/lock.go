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
		Use:   "lock <worktree>",
		Short: "Lock a worktree to prevent removal",
		Long: `Lock a worktree to prevent removal.

Locked worktrees resist prune and remove. Use unlock to clear.
Accepts worktree name (directory) or branch name.

Examples:
  grove lock feat-auth                 # Lock worktree
  grove lock feat-auth --reason "WIP"  # Lock with reason`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeLockArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLock(args[0], reason)
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Reason for locking")
	cmd.Flags().BoolP("help", "h", false, "Help for lock")

	return cmd
}

func runLock(target, reason string) error {
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

	// Check if already locked
	if git.IsWorktreeLocked(worktreeInfo.Path) {
		existingReason := git.GetWorktreeLockReason(worktreeInfo.Path)
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
		logger.Success("Locked worktree %s (%s)", target, reason)
	} else {
		logger.Success("Locked worktree %s", target)
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
			completions = append(completions, filepath.Base(info.Path))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
