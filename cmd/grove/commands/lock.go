package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/formatter"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

// NewLockCmd creates the lock command
func NewLockCmd() *cobra.Command {
	var reason string

	cmd := &cobra.Command{
		Use:   "lock <worktree>...",
		Short: "Lock worktrees to prevent removal",
		Long: `Lock one or more worktrees to prevent removal.

Locked worktrees resist prune and remove. Use unlock to clear.
Accepts worktree names (directories) or branch names.

Examples:
  grove lock feat-auth                      # Lock worktree
  grove lock feat-auth --reason "WIP"       # Lock with reason
  grove lock feat-auth bugfix-123           # Lock multiple`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completeLockArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLock(args, reason)
		},
	}

	cmd.Flags().StringVar(&reason, "reason", "", "Reason for locking")
	cmd.Flags().BoolP("help", "h", false, "Help for lock")

	_ = cmd.RegisterFlagCompletionFunc("reason", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func runLock(targets []string, reason string) error {
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
	var toLock []*git.WorktreeInfo
	for _, target := range targets {
		target = strings.TrimSpace(target)
		info := git.FindWorktree(infos, target)
		if info == nil {
			return fmt.Errorf("worktree not found: %s", target)
		}
		toLock = append(toLock, info)
	}

	// Deduplicate by path
	seen := make(map[string]bool)
	var unique []*git.WorktreeInfo
	for _, info := range toLock {
		if seen[info.Path] {
			continue
		}
		seen[info.Path] = true
		unique = append(unique, info)
	}

	// Process each target, accumulate failures
	var failed []string
	for _, info := range unique {
		label := formatter.WorktreeLabel(info)
		dirName := filepath.Base(info.Path)

		if git.IsWorktreeLocked(info.Path) {
			existingReason := git.GetWorktreeLockReason(info.Path)
			if existingReason != "" {
				logger.Error("%s: already locked (%q)\n\nHint: Use 'grove unlock %s' to remove the lock", label, existingReason, dirName)
			} else {
				logger.Error("%s: already locked\n\nHint: Use 'grove unlock %s' to remove the lock", label, dirName)
			}
			failed = append(failed, dirName)
			continue
		}

		if err := git.LockWorktree(bareDir, info.Path, reason); err != nil {
			logger.Error("%s: %v", label, err)
			failed = append(failed, dirName)
			continue
		}

		if reason != "" {
			logger.Success("Locked worktree %s (%s)", label, reason)
		} else {
			logger.Success("Locked worktree %s", label)
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed: %s", strings.Join(failed, ", "))
	}

	return nil
}

func completeLockArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

		// Only include non-locked worktrees
		if !info.Locked {
			completions = append(completions, name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
