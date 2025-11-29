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

// NewUnlockCmd creates the unlock command
func NewUnlockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unlock <branch>",
		Short: "Unlock a worktree to allow removal",
		Long: `Unlock a worktree so it can be removed by prune or remove commands.

Examples:
  grove unlock feature-auth     # Unlock worktree`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeUnlockArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlock(args[0])
		},
	}

	cmd.Flags().BoolP("help", "h", false, "Help for unlock")

	return cmd
}

func runUnlock(branch string) error {
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

	// Check if actually locked
	if !git.IsWorktreeLocked(bareDir, worktreeName) {
		return fmt.Errorf("worktree is not locked")
	}

	// Unlock the worktree
	if err := git.UnlockWorktree(bareDir, worktreeInfo.Path); err != nil {
		return fmt.Errorf("failed to unlock worktree: %w", err)
	}

	logger.Success("Unlocked worktree %s", branch)

	return nil
}

func completeUnlockArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
		// Only include locked worktrees
		if info.Locked {
			completions = append(completions, info.Branch)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
