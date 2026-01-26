package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

func NewMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <worktree> <new-branch>",
		Short: "Move a branch and its worktree",
		Long: `Rename a branch and its worktree directory atomically.

Accepts worktree name (directory) or branch name.

Example:
  grove move feat/old feat/new`,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeMoveArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMove(args[0], args[1])
		},
	}

	return cmd
}

func runMove(target, newBranch string) error {
	target = strings.TrimSpace(target)
	newBranch = strings.TrimSpace(newBranch)

	if target == newBranch {
		return fmt.Errorf("source and destination are the same: %s", target)
	}

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	workspaceRoot := filepath.Dir(bareDir)

	// Find the worktree (fast: false to get Upstream info)
	infos, err := git.ListWorktreesWithInfo(bareDir, false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktreeInfo := git.FindWorktree(infos, target)
	if worktreeInfo == nil {
		return fmt.Errorf("worktree not found: %s", target)
	}

	if worktreeInfo.Branch == newBranch {
		return fmt.Errorf("worktree already has branch %s", newBranch)
	}

	// Check if user is inside the worktree being renamed
	if fs.PathsEqual(cwd, worktreeInfo.Path) || fs.PathHasPrefix(cwd, worktreeInfo.Path) {
		return fmt.Errorf("cannot rename current worktree; switch to a different worktree first")
	}

	// Check new branch doesn't already exist
	newExists, err := git.BranchExists(bareDir, newBranch)
	if err != nil {
		return fmt.Errorf("failed to check if branch exists: %w", err)
	}
	if newExists {
		return fmt.Errorf("branch %q already exists", newBranch)
	}

	// Check worktree is not dirty
	hasChanges, _, err := git.CheckGitChanges(worktreeInfo.Path)
	if err != nil {
		return fmt.Errorf("failed to check worktree status: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("worktree has uncommitted changes; commit or stash them first")
	}

	// Check worktree is not locked
	if git.IsWorktreeLocked(worktreeInfo.Path) {
		return fmt.Errorf("worktree is locked; unlock it first with 'grove unlock %s'", target)
	}

	// Calculate new worktree path
	newDirName := workspace.SanitizeBranchName(newBranch)
	newWorktreePath := filepath.Join(workspaceRoot, newDirName)

	// Check if new directory already exists
	if _, err := os.Stat(newWorktreePath); err == nil {
		return fmt.Errorf("directory %q already exists", newWorktreePath)
	}

	// Acquire workspace lock
	lockFile := filepath.Join(workspaceRoot, ".grove-worktree.lock")
	lockHandle, err := workspace.AcquireWorkspaceLock(lockFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = lockHandle.Close()
		_ = os.Remove(lockFile)
	}()

	// Track steps for rollback
	var branchRenamed, dirMoved bool
	oldWorktreePath := worktreeInfo.Path
	spin := logger.StartSpinner(fmt.Sprintf("Moving worktree %s to %s...", target, newBranch))

	defer func() {
		if !branchRenamed && !dirMoved {
			return
		}

		// Rollback on failure
		logger.Warning("Attempting rollback...")

		if dirMoved {
			if err := os.Rename(newWorktreePath, oldWorktreePath); err != nil {
				logger.Error("Failed to restore directory: %v", err)
			}
		}

		if branchRenamed {
			if err := git.RenameBranch(bareDir, newBranch, worktreeInfo.Branch); err != nil {
				logger.Error("Failed to restore branch name: %v", err)
			}
		}

		// Try to repair worktree after rollback
		_ = git.RepairWorktree(bareDir, oldWorktreePath)
	}()

	// Step 1: Rename the git branch
	if err := git.RenameBranch(bareDir, worktreeInfo.Branch, newBranch); err != nil {
		spin.StopWithError("Failed to rename branch")
		return fmt.Errorf("failed to rename branch: %w", err)
	}
	branchRenamed = true

	// Step 2: Move the worktree directory
	if err := os.Rename(oldWorktreePath, newWorktreePath); err != nil {
		spin.StopWithError("Failed to move directory")
		return fmt.Errorf("failed to move worktree directory: %w", err)
	}
	dirMoved = true

	// Step 3: Repair worktree to update git's registry with new path
	if err := git.RepairWorktree(bareDir, newWorktreePath); err != nil {
		spin.StopWithError("Failed to repair worktree")
		return fmt.Errorf("failed to repair worktree: %w", err)
	}
	spin.Stop()

	// Step 4: Update upstream tracking if configured
	if worktreeInfo.Upstream != "" {
		// Extract remote and branch from upstream (e.g., "origin/feature/old" -> "origin", "feature/old")
		parts := strings.SplitN(worktreeInfo.Upstream, "/", 2)
		if len(parts) == 2 {
			remote := parts[0]
			newUpstream := fmt.Sprintf("%s/%s", remote, newBranch)
			// Check if new upstream exists on remote
			exists, err := git.BranchExists(bareDir, newUpstream)
			if err != nil {
				logger.Warning("Failed to check if upstream exists: %v", err)
			} else if exists {
				if err := git.SetUpstreamBranch(newWorktreePath, newUpstream); err != nil {
					logger.Warning("Failed to update upstream: %v", err)
				}
			}
		}
	}

	// Success - clear rollback flags
	branchRenamed = false
	dirMoved = false

	if newDirName != newBranch {
		logger.Success("Renamed %s to %s (dir: %s)", target, newBranch, newDirName)
	} else {
		logger.Success("Renamed %s to %s", target, newBranch)
	}

	return nil
}

func completeMoveArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	// Only complete first argument (old branch name)
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
		// Exclude current worktree
		if !fs.PathsEqual(cwd, info.Path) && !fs.PathHasPrefix(cwd, info.Path) {
			completions = append(completions, filepath.Base(info.Path))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
