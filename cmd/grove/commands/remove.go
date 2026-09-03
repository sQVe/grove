package commands

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/formatter"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
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
		ValidArgsFunction: worktreeCompletion(0, false, notCurrentWorktree),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args, force, deleteBranch)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Remove even if dirty or locked; with --branch, delete unmerged and unpushed commits")
	cmd.Flags().BoolVar(&deleteBranch, "branch", false, "Also delete the branch")
	cmd.Flags().BoolP("help", "h", false, "Help for remove")

	return cmd
}

func runRemove(targets []string, force, deleteBranch bool) error {
	if len(targets) == 0 {
		return fmt.Errorf("requires at least one worktree")
	}

	cwd, bareDir, infos, err := loadWorkspace(true)
	if err != nil {
		return err
	}

	cleaned := make([]string, len(targets))
	for i, target := range targets {
		cleaned[i] = strings.TrimSpace(target)
	}
	toRemove, err := resolveWorktrees(infos, cleaned)
	if err != nil {
		return err
	}

	// Process each target, accumulate successes and failures
	type removedWorktree struct {
		path     string
		branch   string
		detached bool
	}
	var removed []removedWorktree
	var deletedBranches int
	var failed []string

	var spin *logger.Spinner
	if len(toRemove) > 1 {
		spin = logger.StartSpinner(fmt.Sprintf("Removing worktrees (0/%d)...", len(toRemove)))
	}

	for i, info := range toRemove {
		if spin != nil {
			spin.Update(fmt.Sprintf("Removing worktrees (%d/%d)...", i+1, len(toRemove)))
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
		}

		// Count commits before removing the worktree so branch deletion can warn.
		var aheadCount, unreachableCount int
		deleteThisBranch := deleteBranch && !info.Detached
		if deleteThisBranch {
			if force {
				unreachableCount, err = git.CountUnreachableCommits(bareDir, info.Branch)
				if err != nil {
					logger.Error("%s: failed to count commits: %v", displayName, err)
					failed = append(failed, dirName)
					continue
				}
			} else {
				syncStatus := git.GetSyncStatus(info.Path)
				aheadCount = syncStatus.Ahead
			}
		}

		if force && git.IsWorktreeLocked(info.Path) {
			// Unlock worktree first if locked (git requires double force otherwise)
			if err := git.UnlockWorktree(bareDir, info.Path); err != nil {
				logger.Debug("Failed to unlock worktree: %v", err)
			}
		}

		// Remove the worktree
		if err := git.RemoveWorktree(bareDir, info.Path, force); err != nil {
			logger.Error("%s: failed to remove worktree: %v", displayName, err)
			failed = append(failed, dirName)
			continue
		}
		removed = append(removed, removedWorktree{path: info.Path, branch: info.Branch, detached: info.Detached})

		// Optionally delete the branch
		if deleteThisBranch {
			if unreachableCount > 0 {
				logger.Warning("%s: %d commit(s) not on any other ref will be lost", info.Branch, unreachableCount)
			} else if aheadCount > 0 {
				logger.Warning("%s: branch has %d unpushed commit(s)", info.Branch, aheadCount)
			}

			if err := git.DeleteBranch(bareDir, info.Branch, force); err != nil {
				logger.Error("%s: worktree removed but failed to delete branch: %v", displayName, err)
				failed = append(failed, dirName)
				continue
			}
			deletedBranches++
		}
	}

	if spin != nil {
		spin.Stop()
	}

	// Print summary
	if len(removed) > 0 {
		if len(removed) == 1 {
			logger.Success("Removed worktree %s", styles.RenderPath(removed[0].path))
			if deletedBranches == 1 {
				logger.ListSubItem("deleted branch %s", removed[0].branch)
			}
		} else {
			if deletedBranches == len(removed) {
				logger.Success("Removed %d worktrees and branches:", len(removed))
			} else {
				logger.Success("Removed %d worktrees:", len(removed))
			}
			for _, r := range removed {
				if deleteBranch && !r.detached {
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
