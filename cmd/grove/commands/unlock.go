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
		Use:   "unlock <worktree>...",
		Short: "Unlock worktrees to allow removal",
		Long: `Unlock one or more worktrees so they can be removed.

Accepts worktree names (directories) or branch names.

Examples:
  grove unlock feat-auth
  grove unlock feat-auth bugfix-123`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completeUnlockArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runUnlock(args)
		},
	}

	cmd.Flags().BoolP("help", "h", false, "Help for unlock")

	return cmd
}

func runUnlock(targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("requires at least one worktree")
	}

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

	// Validate all targets exist before processing
	var toUnlock []*git.WorktreeInfo
	for _, target := range targets {
		target = strings.TrimSpace(target)
		info := git.FindWorktree(infos, target)
		if info == nil {
			return fmt.Errorf("worktree not found: %s", target)
		}
		toUnlock = append(toUnlock, info)
	}

	// Deduplicate by path
	seen := make(map[string]bool)
	var unique []*git.WorktreeInfo
	for _, info := range toUnlock {
		if seen[info.Path] {
			continue
		}
		seen[info.Path] = true
		unique = append(unique, info)
	}

	// Process each target, accumulate failures
	var failed []string
	for _, info := range unique {
		if !git.IsWorktreeLocked(info.Path) {
			logger.Error("%s: worktree is not locked", info.Branch)
			failed = append(failed, info.Branch)
			continue
		}

		if err := git.UnlockWorktree(bareDir, info.Path); err != nil {
			logger.Error("%s: %v", info.Branch, err)
			failed = append(failed, info.Branch)
			continue
		}

		logger.Success("Unlocked worktree %s", info.Branch)
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed: %s", strings.Join(failed, ", "))
	}

	return nil
}

func completeUnlockArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

	// Build set of already-typed arguments
	alreadyUsed := make(map[string]bool)
	for _, arg := range args {
		alreadyUsed[arg] = true
	}

	var completions []string
	for _, info := range infos {
		name := filepath.Base(info.Path)

		// Skip already-used (check both path basename and branch name)
		if alreadyUsed[name] || alreadyUsed[info.Branch] {
			continue
		}

		// Only include locked worktrees
		if info.Locked {
			completions = append(completions, name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
