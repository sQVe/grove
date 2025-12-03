package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/github"
	"github.com/sqve/grove/internal/hooks"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

func NewAddCmd() *cobra.Command {
	var baseBranch string
	var name string
	var detach bool

	cmd := &cobra.Command{
		Use:   "add <branch|#PR|PR-URL|ref>",
		Short: "Add a new worktree",
		Long: `Add a new worktree for a branch, pull request, or ref.

The directory name is derived from the branch name unless --name is specified.

Examples:
  grove add feature/auth              # Creates ./feature-auth worktree
  grove add feature/auth --name auth  # Creates ./auth worktree
  grove add main                      # Existing branch
  grove add -s feature/auth           # Add and switch to worktree
  grove add --base main feature/auth  # New branch from main
  grove add --detach v1.0.0           # Detached HEAD at tag
  grove add #123                      # PR by number
  grove add PR-URL                    # PR by URL`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeAddArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switchTo, _ := cmd.Flags().GetBool("switch")
			return runAdd(args[0], switchTo, baseBranch, name, detach)
		},
	}

	cmd.Flags().BoolP("switch", "s", false, "Switch to the new worktree after creation")
	cmd.Flags().StringVar(&baseBranch, "base", "", "Create new branch from this base instead of HEAD")
	cmd.Flags().StringVar(&name, "name", "", "Custom directory name for the worktree")
	cmd.Flags().BoolVar(&detach, "detach", false, "Create worktree in detached HEAD state")
	cmd.Flags().BoolP("help", "h", false, "Help for add")

	_ = cmd.RegisterFlagCompletionFunc("base", completeBaseBranch)

	return cmd
}

func runAdd(branchOrPR string, switchTo bool, baseBranch, name string, detach bool) error {
	branchOrPR = strings.TrimSpace(branchOrPR)
	name = strings.TrimSpace(name)

	// Validate flag combinations early (before filesystem operations)
	if detach && baseBranch != "" {
		return fmt.Errorf("--detach and --base cannot be used together")
	}

	isPR := github.IsPRReference(branchOrPR)
	if isPR {
		if baseBranch != "" {
			return fmt.Errorf("--base cannot be used with PR references")
		}
		if detach {
			return fmt.Errorf("--detach cannot be used with PR references")
		}
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

	// Acquire workspace lock to prevent concurrent worktree creation
	lockFile := filepath.Join(workspaceRoot, ".grove-worktree.lock")
	lockHandle, err := workspace.AcquireWorkspaceLock(lockFile)
	if err != nil {
		return err
	}
	defer func() {
		_ = lockHandle.Close()
		_ = os.Remove(lockFile)
	}()

	sourceWorktree := findSourceWorktree(cwd, workspaceRoot)

	// Handle PR reference
	if isPR {
		return runAddFromPR(branchOrPR, switchTo, name, bareDir, workspaceRoot, sourceWorktree)
	}

	// Detached worktree
	if detach {
		return runAddDetached(branchOrPR, switchTo, name, bareDir, workspaceRoot, sourceWorktree)
	}

	// Regular branch creation
	return runAddFromBranch(branchOrPR, switchTo, baseBranch, name, bareDir, workspaceRoot, sourceWorktree)
}

func runAddFromBranch(branch string, switchTo bool, baseBranch, name, bareDir, workspaceRoot, sourceWorktree string) error {
	dirName := name
	if dirName == "" {
		dirName = workspace.SanitizeBranchName(branch)
	}
	worktreePath := filepath.Join(workspaceRoot, dirName)

	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}
	for _, info := range infos {
		if info.Branch == branch {
			return fmt.Errorf("worktree already exists for branch %q at %s", branch, info.Path)
		}
	}

	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("directory already exists: %s", worktreePath)
	}

	exists, err := git.BranchExists(bareDir, branch)
	if err != nil {
		return fmt.Errorf("failed to check branch: %w", err)
	}

	if exists {
		if baseBranch != "" {
			return fmt.Errorf("--base cannot be used with existing branch %q", branch)
		}
		if err := git.CreateWorktree(bareDir, worktreePath, branch, true); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		if baseBranch != "" {
			// Validate base branch exists
			baseExists, err := git.BranchExists(bareDir, baseBranch)
			if err != nil {
				return fmt.Errorf("failed to check base branch: %w", err)
			}
			if !baseExists {
				return fmt.Errorf("base branch %q does not exist", baseBranch)
			}
			if err := git.CreateWorktreeWithNewBranchFrom(bareDir, worktreePath, branch, baseBranch, true); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		} else {
			if err := git.CreateWorktreeWithNewBranch(bareDir, worktreePath, branch, true); err != nil {
				return fmt.Errorf("failed to create worktree: %w", err)
			}
		}
	}

	// Auto-lock if branch matches auto-lock patterns
	if config.ShouldAutoLock(branch) {
		if err := git.LockWorktree(bareDir, worktreePath, "Auto-locked (grove.autoLock)"); err != nil {
			logger.Debug("Failed to auto-lock worktree: %v", err)
		} else {
			logger.Debug("Auto-locked worktree for branch %s", branch)
		}
	}

	preserveResult := preserveFilesFromSource(sourceWorktree, worktreePath)
	hookResult := runAddHooks(sourceWorktree, worktreePath)

	if switchTo {
		fmt.Println(worktreePath) // Path for shell wrapper to cd into
	} else {
		logger.Success("Created worktree at %s", worktreePath)
		logPreserveResult(preserveResult)
		logHookResult(hookResult)
	}
	return nil
}

func runAddDetached(ref string, switchTo bool, name, bareDir, workspaceRoot, sourceWorktree string) error {
	dirName := name
	if dirName == "" {
		dirName = workspace.SanitizeBranchName(ref)
	}
	worktreePath := filepath.Join(workspaceRoot, dirName)

	// Check directory doesn't already exist
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("directory already exists: %s", worktreePath)
	}

	// Validate ref exists
	if err := git.RefExists(bareDir, ref); err != nil {
		return fmt.Errorf("ref %q does not exist", ref)
	}

	if err := git.CreateWorktreeDetached(bareDir, worktreePath, ref, true); err != nil {
		return fmt.Errorf("failed to create detached worktree: %w", err)
	}

	// Note: Auto-lock not applied for detached worktrees (no branch to lock)

	preserveResult := preserveFilesFromSource(sourceWorktree, worktreePath)
	hookResult := runAddHooks(sourceWorktree, worktreePath)

	if switchTo {
		fmt.Println(worktreePath)
	} else {
		logger.Success("Created detached worktree at %s", worktreePath)
		logPreserveResult(preserveResult)
		logHookResult(hookResult)
	}
	return nil
}

func runAddFromPR(prRef string, switchTo bool, name, bareDir, workspaceRoot, sourceWorktree string) error {
	// Check gh is available
	if err := github.CheckGhAvailable(); err != nil {
		return err
	}

	// Parse PR reference
	ref, err := github.ParsePRReference(prRef)
	if err != nil {
		return err
	}

	// If no owner/repo in ref, get from workspace's origin
	owner, repo := ref.Owner, ref.Repo
	if owner == "" || repo == "" {
		repoRef, err := getRepoFromOrigin(bareDir)
		if err != nil {
			return fmt.Errorf("PR number requires workspace context: %w", err)
		}
		owner, repo = repoRef.Owner, repoRef.Repo
	}

	// Fetch PR info
	logger.Info("Fetching PR #%d from %s/%s...", ref.Number, owner, repo)
	prInfo, err := github.FetchPRInfo(owner, repo, ref.Number)
	if err != nil {
		return err
	}

	branch := prInfo.HeadRef
	dirName := name
	if dirName == "" {
		dirName = workspace.SanitizeBranchName(branch)
	}
	worktreePath := filepath.Join(workspaceRoot, dirName)

	// Check if worktree already exists
	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}
	for _, info := range infos {
		if info.Branch == branch {
			return fmt.Errorf("worktree already exists for branch %q at %s", branch, info.Path)
		}
	}

	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("directory already exists: %s", worktreePath)
	}

	// Handle fork PRs: add remote and fetch
	if prInfo.IsFork {
		remoteName := fmt.Sprintf("pr-%d-%s", ref.Number, prInfo.HeadOwner)

		// Check if remote already exists
		exists, err := git.RemoteExists(bareDir, remoteName)
		if err != nil {
			return fmt.Errorf("failed to check remote: %w", err)
		}

		// Track if we added a new remote (for cleanup on failure)
		addedRemote := false

		if !exists {
			remoteURL, err := github.GetRepoCloneURL(prInfo.HeadOwner, prInfo.HeadRepo)
			if err != nil {
				return fmt.Errorf("failed to get fork URL: %w", err)
			}

			logger.Info("Adding remote %s for fork...", remoteName)
			if err := git.AddRemote(bareDir, remoteName, remoteURL); err != nil {
				return fmt.Errorf("failed to add fork remote: %w", err)
			}
			addedRemote = true
		}

		// Cleanup helper: remove remote if we added it and something fails
		cleanupRemote := func() {
			if addedRemote {
				logger.Debug("Cleaning up remote %s after failure", remoteName)
				_ = git.RemoveRemote(bareDir, remoteName)
			}
		}

		logger.Info("Fetching branch %s from fork...", branch)
		if err := git.FetchBranch(bareDir, remoteName, branch); err != nil {
			cleanupRemote()
			return fmt.Errorf("failed to fetch fork branch: %w", err)
		}

		// Create worktree tracking the fork's branch
		trackingRef := fmt.Sprintf("%s/%s", remoteName, branch)
		if err := git.CreateWorktree(bareDir, worktreePath, trackingRef, true); err != nil {
			cleanupRemote()
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		// Same-repo PR: fetch and create worktree
		logger.Info("Fetching branch %s...", branch)
		if err := git.FetchBranch(bareDir, "origin", branch); err != nil {
			return fmt.Errorf("failed to fetch branch: %w", err)
		}

		if err := git.CreateWorktree(bareDir, worktreePath, branch, true); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	// Auto-lock if branch matches auto-lock patterns
	if config.ShouldAutoLock(branch) {
		if err := git.LockWorktree(bareDir, worktreePath, "Auto-locked (grove.autoLock)"); err != nil {
			logger.Debug("Failed to auto-lock worktree: %v", err)
		} else {
			logger.Debug("Auto-locked worktree for branch %s", branch)
		}
	}

	preserveResult := preserveFilesFromSource(sourceWorktree, worktreePath)
	hookResult := runAddHooks(sourceWorktree, worktreePath)

	if switchTo {
		fmt.Println(worktreePath)
	} else {
		logger.Success("Created worktree for PR #%d at %s", ref.Number, worktreePath)
		logPreserveResult(preserveResult)
		logHookResult(hookResult)
	}
	return nil
}

// getRepoFromOrigin extracts owner/repo from the origin remote URL.
func getRepoFromOrigin(bareDir string) (*github.RepoRef, error) {
	url, err := git.GetRemoteURL(bareDir, "origin")
	if err != nil {
		return nil, err
	}
	return github.ParseRepoURL(url)
}

func findSourceWorktree(cwd, workspaceRoot string) string {
	if cwd == workspaceRoot {
		return ""
	}

	if git.IsWorktree(cwd) {
		return cwd
	}

	dir := cwd
	for dir != workspaceRoot && dir != "/" {
		if git.IsWorktree(dir) {
			return dir
		}
		dir = filepath.Dir(dir)
	}

	return ""
}

func preserveFilesFromSource(sourceWorktree, destWorktree string) *workspace.PreserveResult {
	if sourceWorktree == "" {
		logger.Debug("No source worktree, skipping file preservation")
		return nil
	}

	// Find ignored files in source worktree
	ignoredFiles, err := workspace.FindIgnoredFilesInWorktree(sourceWorktree)
	if err != nil {
		logger.Debug("Failed to find ignored files: %v", err)
		return nil
	}

	if len(ignoredFiles) == 0 {
		logger.Debug("No ignored files found in source worktree")
		return nil
	}

	// Get preserve patterns (uses merged config: TOML > git config > defaults)
	patterns := config.GetMergedPreservePatterns(sourceWorktree)

	// Copy preserved files
	result, err := workspace.PreserveFilesToWorktree(sourceWorktree, destWorktree, patterns, ignoredFiles)
	if err != nil {
		logger.Debug("Failed to preserve files: %v", err)
		return nil
	}

	return result
}

func logPreserveResult(result *workspace.PreserveResult) {
	if result == nil {
		return
	}

	if len(result.Copied) > 0 {
		header := fmt.Sprintf("preserved %d files:", len(result.Copied))
		if len(result.Copied) == 1 {
			header = "preserved 1 file:"
		}
		logger.ListItemGroup(header, result.Copied)
	}

	if len(result.Skipped) > 0 {
		if len(result.Skipped) == 1 {
			logger.Warning("Skipped 1 file (already exists): %s", result.Skipped[0])
		} else {
			header := fmt.Sprintf("skipped %d files (already exist):", len(result.Skipped))
			logger.ListSubItem("%s", header)
			for _, f := range result.Skipped {
				logger.Dimmed("        %s", f)
			}
		}
	}
}

func runAddHooks(sourceWorktree, destWorktree string) *hooks.RunResult {
	var addHooks []string
	if sourceWorktree != "" {
		addHooks = hooks.GetAddHooks(sourceWorktree)
	}

	if len(addHooks) == 0 {
		logger.Debug("No add hooks configured")
		return nil
	}

	logger.Debug("Found %d add hooks", len(addHooks))
	return hooks.RunAddHooks(destWorktree, addHooks)
}

func logHookResult(result *hooks.RunResult) {
	if result == nil {
		return
	}

	if len(result.Succeeded) > 0 {
		header := fmt.Sprintf("ran %d hooks:", len(result.Succeeded))
		if len(result.Succeeded) == 1 {
			header = "ran 1 hook:"
		}
		logger.ListItemGroup(header, result.Succeeded)
	}

	if result.Failed != nil {
		logger.Warning("Hook failed: %s (exit code %d)", result.Failed.Command, result.Failed.ExitCode)
		if config.IsDebug() {
			if result.Failed.Stdout != "" {
				logger.Debug("stdout: %s", result.Failed.Stdout)
			}
			if result.Failed.Stderr != "" {
				logger.Debug("stderr: %s", result.Failed.Stderr)
			}
		}
	}
}

func completeAddArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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

	branches, err := git.ListBranches(bareDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	existingBranches := make(map[string]bool)
	for _, info := range infos {
		existingBranches[info.Branch] = true
	}

	var completions []string
	for _, b := range branches {
		if !existingBranches[b] {
			completions = append(completions, b)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func completeBaseBranch(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	branches, err := git.ListBranches(bareDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var completions []string
	for _, b := range branches {
		if strings.HasPrefix(b, toComplete) {
			completions = append(completions, b)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp
}
