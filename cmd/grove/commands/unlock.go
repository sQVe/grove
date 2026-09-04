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
		Args: cobra.ArbitraryArgs,
		ValidArgsFunction: worktreeCompletion(0, false, func(_ string, info *git.WorktreeInfo) bool {
			return info.Locked
		}),
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

	_, bareDir, infos, err := loadWorkspace(true)
	if err != nil {
		return err
	}

	cleaned := make([]string, len(targets))
	for i, target := range targets {
		cleaned[i] = strings.TrimSpace(target)
	}
	toUnlock, err := resolveWorktrees(infos, cleaned)
	if err != nil {
		return err
	}

	// Process each target, accumulate failures
	var failed []string
	for _, info := range toUnlock {
		label := formatter.WorktreeLabel(info)
		dirName := filepath.Base(info.Path)

		if !git.IsWorktreeLocked(info.Path) {
			logger.Error("%s: worktree is not locked", label)
			failed = append(failed, dirName)
			continue
		}

		if err := git.UnlockWorktree(bareDir, info.Path); err != nil {
			logger.Error("%s: %v", label, err)
			failed = append(failed, dirName)
			continue
		}

		logger.Success("Unlocked worktree %s", label)
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed: %s", strings.Join(failed, ", "))
	}

	return nil
}
