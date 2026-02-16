package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/formatter"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

// NewRemoveCmd creates the remove command
func NewRemoveCmd() *cobra.Command {
	var force bool
	var deleteBranch bool

	cmd := &cobra.Command{
		Use:   "remove <worktree>...",
		Short: "Remove worktrees",
		Long: `Remove one or more worktrees, optionally deleting their branches.

Accepts worktree names (directories) or branch names.

Examples:
  grove remove feat-auth            # Remove worktree
  grove remove --branch feat        # Remove worktree and branch
  grove remove --force wip          # Force remove if dirty or locked
  grove remove feat-auth bugfix-123 # Remove multiple worktrees`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completeRemoveArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args, force, deleteBranch)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Remove even if dirty or locked")
	cmd.Flags().BoolVar(&deleteBranch, "branch", false, "Also delete the branch")
	cmd.Flags().BoolP("help", "h", false, "Help for remove")

	return cmd
}

func runRemove(targets []string, force, deleteBranch bool) error {
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
	var toRemove []*git.WorktreeInfo
	for _, target := range targets {
		target = strings.TrimSpace(target)
		info := git.FindWorktree(infos, target)
		if info == nil {
			return fmt.Errorf("worktree not found: %s", target)
		}
		toRemove = append(toRemove, info)
	}

	// Deduplicate by path
	seen := make(map[string]bool)
	var unique []*git.WorktreeInfo
	for _, info := range toRemove {
		if seen[info.Path] {
			continue
		}
		seen[info.Path] = true
		unique = append(unique, info)
	}

	// Process each target, accumulate successes and failures
	type removedWorktree struct {
		path   string
		branch string
	}
	var removed []removedWorktree
	var failed []string

	var spin *logger.Spinner
	if len(unique) > 1 {
		spin = logger.StartSpinner(fmt.Sprintf("Removing worktrees (0/%d)...", len(unique)))
	}

	for i, info := range unique {
		if spin != nil {
			spin.Update(fmt.Sprintf("Removing worktrees (%d/%d)...", i+1, len(unique)))
		}

		displayName := formatter.WorktreeLabel(info)
		dirName := filepath.Base(info.Path)

		// Check if user is inside the worktree being deleted
		if fs.PathsEqual(cwd, info.Path) || fs.PathHasPrefix(cwd, info.Path) {
			logger.Error("%s: cannot delete current worktree\n\nHint: Switch to a different worktree first with 'grove switch <worktree>'", displayName)
			failed = append(failed, dirName)
			continue
		}

		// Check worktree state unless --force
		if !force {
			hasChanges, _, err := git.CheckGitChanges(info.Path)
			if err != nil {
				logger.Error("%s: failed to check worktree status: %v", displayName, err)
				failed = append(failed, dirName)
				continue
			}
			if hasChanges {
				logger.Error("%s: worktree has uncommitted changes; use --force to remove anyway", displayName)
				failed = append(failed, dirName)
				continue
			}

			if git.IsWorktreeLocked(info.Path) {
				logger.Error("%s: worktree is locked; use --force to remove anyway", displayName)
				failed = append(failed, dirName)
				continue
			}
		} else if git.IsWorktreeLocked(info.Path) {
			// Unlock worktree first if locked (git requires double force otherwise)
			if err := git.UnlockWorktree(bareDir, info.Path); err != nil {
				logger.Debug("Failed to unlock worktree: %v", err)
			}
		}

		// Get sync status BEFORE removing worktree if we need to warn about unpushed commits
		var aheadCount int
		if deleteBranch {
			syncStatus := git.GetSyncStatus(info.Path)
			aheadCount = syncStatus.Ahead
		}

		// Remove the worktree
		if err := git.RemoveWorktree(bareDir, info.Path, force); err != nil {
			logger.Error("%s: failed to remove worktree: %v", displayName, err)
			failed = append(failed, dirName)
			continue
		}

		// Optionally delete the branch
		if deleteBranch {
			if aheadCount > 0 {
				logger.Warning("%s: branch has %d unpushed commit(s)", info.Branch, aheadCount)
			}

			if err := git.DeleteBranch(bareDir, info.Branch, force); err != nil {
				logger.Error("%s: worktree removed but failed to delete branch: %v", displayName, err)
				failed = append(failed, dirName)
				continue
			}
		}
		removed = append(removed, removedWorktree{path: info.Path, branch: info.Branch})
	}

	if spin != nil {
		spin.Stop()
	}

	// Print summary
	if len(removed) > 0 {
		if len(removed) == 1 {
			logger.Success("Removed worktree %s", styles.RenderPath(removed[0].path))
			if deleteBranch {
				logger.ListSubItem("deleted branch %s", removed[0].branch)
			}
		} else {
			if deleteBranch {
				logger.Success("Removed %d worktrees and branches:", len(removed))
			} else {
				logger.Success("Removed %d worktrees:", len(removed))
			}
			for _, r := range removed {
				if deleteBranch {
					logger.ListSubItem("%s (branch %s)", styles.RenderPath(r.path), r.branch)
				} else {
					logger.ListSubItem("%s", styles.RenderPath(r.path))
				}
			}
		}
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed: %s", strings.Join(failed, ", "))
	}

	return nil
}

func completeRemoveArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

		// Exclude current worktree (use fs.PathsEqual for cross-platform comparison)
		if !fs.PathsEqual(cwd, info.Path) && !fs.PathHasPrefix(cwd, info.Path) {
			completions = append(completions, name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
