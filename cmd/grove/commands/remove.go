package commands

import (
	"fmt"
	"path/filepath"
	"slices"
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
	var ignoreMissing bool

	cmd := &cobra.Command{
		Use:   "remove <worktree>...",
		Short: "Remove worktrees",
		Long: `Remove one or more worktrees, optionally deleting their branches.

Accepts worktree names (directories) or branch names.
With --branch, unmerged branches are rejected before removal unless --force is set.

Examples:
  grove remove feat-auth            # Remove worktree
  grove remove --branch feat        # Remove worktree and branch
  grove remove --force wip          # Force remove if dirty or locked
  grove remove feat-auth bugfix-123 # Remove multiple worktrees`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: worktreeCompletion(0, false, notCurrentWorktree),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRemove(args, force, deleteBranch, ignoreMissing)
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "Remove even if dirty or locked; with --branch, delete unmerged and unpushed commits")
	cmd.Flags().BoolVar(&deleteBranch, "branch", false, "Also delete the branch")
	cmd.Flags().BoolVar(&ignoreMissing, "ignore-missing", false, "Skip worktrees that are not found")
	cmd.Flags().BoolP("help", "h", false, "Help for remove")

	return cmd
}

func runRemove(targets []string, force, deleteBranch, ignoreMissing bool) error {
	if len(targets) == 0 {
		return fmt.Errorf("requires at least one worktree")
	}

	cwd, bareDir, infos, err := loadWorkspace(true)
	if err != nil {
		return err
	}

	cleaned := make([]string, 0, len(targets))
	var missing []string
	for _, target := range targets {
		target = strings.TrimSpace(target)
		if ignoreMissing && git.FindWorktree(infos, target) == nil {
			if !slices.Contains(missing, target) {
				missing = append(missing, target)
			}
			continue
		}
		cleaned = append(cleaned, target)
	}

	toRemove, err := resolveWorktrees(infos, cleaned)
	if err != nil {
		return err
	}
	for _, target := range missing {
		logger.Warning("%s: not found (skipped)", target)
	}

	var defaultBranch string
	if deleteBranch && !force {
		defaultBranch, err = git.GetDefaultBranch(bareDir)
		if err != nil {
			logger.Debug("Skipping merge check: could not determine default branch: %v", err)
		}
	}

	// Process each target, accumulate successes and failures
	type removedWorktree struct {
		path          string
		branch        string
		detached      bool
		branchDeleted bool
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

		deleteThisBranch := deleteBranch && !info.Detached
		forceDelete := force
		if deleteThisBranch && !force && defaultBranch != "" {
			merged, mergeErr := git.IsBranchMerged(bareDir, info.Branch, defaultBranch)
			switch {
			case mergeErr != nil:
				logger.Debug("Could not verify merge status for %s: %v", info.Branch, mergeErr)
			case !merged:
				logger.Error("%s: branch is not merged into %s; use --force to delete anyway", info.Branch, defaultBranch)
				failed = append(failed, dirName)
				continue
			default:
				// Git's safe delete does not recognize squash merges.
				forceDelete = true
			}
		}

		// Count commits before removing the worktree so branch deletion can warn.
		var aheadCount, unreachableCount int
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

			if err := git.DeleteBranch(bareDir, info.Branch, forceDelete); err != nil {
				logger.Error("%s: worktree removed but failed to delete branch: %v", displayName, err)
				failed = append(failed, dirName)
				continue
			}
			removed[len(removed)-1].branchDeleted = true
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
				if r.branchDeleted {
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
