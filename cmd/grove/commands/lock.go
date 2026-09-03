package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/formatter"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
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
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: worktreeCompletion(0, false, func(_ string, info *git.WorktreeInfo) bool {
			return !info.Locked
		}),
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

	_, bareDir, infos, err := loadWorkspace(true)
	if err != nil {
		return err
	}

	cleaned := make([]string, len(targets))
	for i, target := range targets {
		cleaned[i] = strings.TrimSpace(target)
	}
	toLock, err := resolveWorktrees(infos, cleaned)
	if err != nil {
		return err
	}

	// Process each target, accumulate failures
	var failed []string
	for _, info := range toLock {
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
