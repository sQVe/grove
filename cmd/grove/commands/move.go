package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

func NewMoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "move <old-branch> <new-branch>",
		Short: "Move a branch and its worktree",
		Long: `Move a branch and its associated worktree directory.

This command atomically renames both the git branch and the worktree directory,
updating all necessary references.

Examples:
  grove move feature/old feature/new`,
		Args:              cobra.ExactArgs(2),
		ValidArgsFunction: completeMoveArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMove(args[0], args[1])
		},
	}

	return cmd
}

func runMove(oldBranch, newBranch string) error {
	oldBranch = strings.TrimSpace(oldBranch)
	newBranch = strings.TrimSpace(newBranch)

	if oldBranch == newBranch {
		return fmt.Errorf("old and new branch names are the same")
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

	// Find the worktree for the old branch (fast: false to get Upstream info)
	infos, err := git.ListWorktreesWithInfo(bareDir, false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	var worktreeInfo *git.WorktreeInfo
	for _, info := range infos {
		if info.Branch == oldBranch {
			worktreeInfo = info
			break
		}
	}

	if worktreeInfo == nil {
		return fmt.Errorf("no worktree found for branch %q", oldBranch)
	}

	// Check if user is inside the worktree being renamed
	if cwd == worktreeInfo.Path || strings.HasPrefix(cwd, worktreeInfo.Path+string(filepath.Separator)) {
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
	worktreeName := filepath.Base(worktreeInfo.Path)
	if git.IsWorktreeLocked(bareDir, worktreeName) {
		return fmt.Errorf("worktree is locked; unlock it first with 'git worktree unlock %s'", worktreeName)
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
	lockHandle, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // path derived from validated workspace
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("another grove operation is in progress; if this is wrong, remove %s", lockFile)
		}
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer func() {
		_ = lockHandle.Close()
		_ = os.Remove(lockFile)
	}()

	// Track steps for rollback
	var branchRenamed, dirMoved bool
	oldWorktreePath := worktreeInfo.Path

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
			if err := git.RenameBranch(bareDir, newBranch, oldBranch); err != nil {
				logger.Error("Failed to restore branch name: %v", err)
			}
		}

		// Try to repair worktree after rollback
		_ = git.RepairWorktree(bareDir, oldWorktreePath)
	}()

	// Step 1: Rename the git branch
	if err := git.RenameBranch(bareDir, oldBranch, newBranch); err != nil {
		return fmt.Errorf("failed to rename branch: %w", err)
	}
	branchRenamed = true

	// Step 2: Move the worktree directory
	if err := os.Rename(oldWorktreePath, newWorktreePath); err != nil {
		return fmt.Errorf("failed to move worktree directory: %w", err)
	}
	dirMoved = true

	// Step 3: Repair worktree to update git's registry with new path
	if err := git.RepairWorktree(bareDir, newWorktreePath); err != nil {
		return fmt.Errorf("failed to repair worktree: %w", err)
	}

	// Step 4: Update upstream tracking if configured
	if worktreeInfo.Upstream != "" {
		// Extract remote and branch from upstream (e.g., "origin/feature/old" -> "origin", "feature/old")
		parts := strings.SplitN(worktreeInfo.Upstream, "/", 2)
		if len(parts) == 2 {
			remote := parts[0]
			newUpstream := fmt.Sprintf("%s/%s", remote, newBranch)
			// Check if new upstream exists on remote
			if exists, _ := git.BranchExists(bareDir, newUpstream); exists {
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
		logger.Success("Renamed %s to %s (dir: %s)", oldBranch, newBranch, newDirName)
	} else {
		logger.Success("Renamed %s to %s", oldBranch, newBranch)
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
		if cwd != info.Path && !strings.HasPrefix(cwd, info.Path+string(os.PathSeparator)) {
			completions = append(completions, info.Branch)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
