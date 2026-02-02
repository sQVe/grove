package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/github"
	"github.com/sqve/grove/internal/hooks"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

func NewAddCmd() *cobra.Command {
	var baseBranch string
	var name string
	var detach bool
	var prNumber int
	var reset bool
	var from string

	cmd := &cobra.Command{
		Use:   "add [branch|PR-URL|ref]",
		Short: "Add a new worktree",
		Long: `Create a worktree from a branch, pull request, or ref.

The directory name derives from the branch name unless --name is specified.

Examples:
  grove add feat/auth              # Creates ./feat-auth worktree
  grove add feat/auth --name auth  # Creates ./auth worktree
  grove add main                   # Existing branch
  grove add -s feat/auth           # Add and switch to worktree
  grove add --base main feat/auth  # New branch from main
  grove add --detach v1.0.0        # Detached HEAD at tag
  grove add --pr 123               # Creates ./pr-123 worktree
  grove add --from dev feat/auth   # Preserve files from dev worktree (name or branch)`,
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeAddArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switchTo, _ := cmd.Flags().GetBool("switch")
			return runAdd(args, switchTo, baseBranch, name, detach, prNumber, reset, from)
		},
	}

	cmd.Flags().BoolP("switch", "s", false, "Switch to the worktree after creating it")
	cmd.Flags().StringVar(&baseBranch, "base", "", "Create new branch from this base instead of HEAD")
	cmd.Flags().StringVar(&name, "name", "", "Custom directory name for the worktree")
	cmd.Flags().BoolVarP(&detach, "detach", "d", false, "Create worktree in detached HEAD state")
	cmd.Flags().IntVar(&prNumber, "pr", 0, "Pull request number to checkout")
	cmd.Flags().BoolVar(&reset, "reset", false, "Reset diverged PR branch to match remote (discards local commits)")
	cmd.Flags().StringVar(&from, "from", "", "Source worktree for file preservation (name or branch)")
	cmd.Flags().BoolP("help", "h", false, "Help for add")

	_ = cmd.RegisterFlagCompletionFunc("base", completeBaseBranch)
	_ = cmd.RegisterFlagCompletionFunc("name", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("pr", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
	_ = cmd.RegisterFlagCompletionFunc("from", completeFromWorktree)

	return cmd
}

func runAdd(args []string, switchTo bool, baseBranch, name string, detach bool, prNumber int, reset bool, from string) error {
	name = strings.TrimSpace(name)

	// Validate --pr value if provided
	if prNumber < 0 {
		return fmt.Errorf("--pr must be a positive number")
	}

	// Determine if --pr flag is used
	prFlag := prNumber > 0

	// Get positional argument if provided
	var branchOrPR string
	if len(args) > 0 {
		branchOrPR = strings.TrimSpace(args[0])
	}

	// Validate: must provide either --pr or positional arg
	if !prFlag && branchOrPR == "" {
		return fmt.Errorf("requires branch, PR URL, or --pr flag")
	}

	// Validate: --pr cannot be combined with positional argument
	if prFlag && branchOrPR != "" {
		return fmt.Errorf("--pr flag cannot be combined with positional argument")
	}

	// Helpful error for old #N syntax
	if strings.HasPrefix(branchOrPR, "#") {
		return fmt.Errorf("'%s' syntax no longer supported, use: grove add --pr %s",
			branchOrPR, strings.TrimPrefix(branchOrPR, "#"))
	}

	// Validate flag combinations early (before filesystem operations)
	if detach && baseBranch != "" {
		return fmt.Errorf("--detach and --base cannot be used together")
	}

	// Check if positional arg is a PR URL
	isPRURL := branchOrPR != "" && github.IsPRURL(branchOrPR)

	// Validate PR-specific flag conflicts
	if prFlag || isPRURL {
		if baseBranch != "" {
			return fmt.Errorf("--base cannot be used with PR references")
		}
		if detach {
			return fmt.Errorf("--detach cannot be used with PR references")
		}
	}

	// --reset only makes sense with PR checkout
	if reset && !prFlag && !isPRURL {
		return fmt.Errorf("--reset can only be used with PR references")
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

	spin := logger.StartSpinner("Preparing workspace...")
	var sourceWorktree string
	if from != "" {
		infos, err := git.ListWorktreesWithInfo(bareDir, true)
		if err != nil {
			spin.Stop()
			return fmt.Errorf("failed to list worktrees: %w", err)
		}
		info := git.FindWorktree(infos, from)
		if info == nil {
			spin.Stop()
			return fmt.Errorf("worktree %q not found", from)
		}
		sourceWorktree = info.Path
		logger.Debug("Using %s as source for file preservation (--from)", sourceWorktree)
	} else {
		sourceWorktree = findSourceWorktree(cwd, workspaceRoot)
		if sourceWorktree == "" {
			sourceWorktree = findFallbackSourceWorktree(bareDir)
			if sourceWorktree != "" {
				logger.Debug("Using %s as source for file preservation", sourceWorktree)
			}
		}
	}
	spin.Stop()

	// Handle PR via --pr flag
	if prFlag {
		prRef := fmt.Sprintf("#%d", prNumber)
		return runAddFromPR(prRef, switchTo, name, bareDir, workspaceRoot, sourceWorktree, reset)
	}

	// Handle PR via URL
	if isPRURL {
		return runAddFromPR(branchOrPR, switchTo, name, bareDir, workspaceRoot, sourceWorktree, reset)
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
			return fmt.Errorf("worktree already exists for branch %q at %s\n\nHint: Use 'grove list' to see existing worktrees, or use --name to choose a different directory", branch, info.Path)
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
			return git.HintGitTooOld(fmt.Errorf("failed to create worktree: %w", err))
		}
		if remoteExists, _ := git.RemoteBranchExists(bareDir, "origin", branch); remoteExists {
			if err := git.SetUpstreamBranch(worktreePath, "origin/"+branch); err != nil {
				logger.Debug("Failed to set upstream for %s: %v", branch, err)
			}
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
				return git.HintGitTooOld(fmt.Errorf("failed to create worktree: %w", err))
			}
		} else {
			if err := git.CreateWorktreeWithNewBranch(bareDir, worktreePath, branch, true); err != nil {
				return git.HintGitTooOld(fmt.Errorf("failed to create worktree: %w", err))
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

	spin := logger.StartSpinner("Setting up worktree...")
	preserveResult := preserveFilesFromSource(sourceWorktree, worktreePath)
	spin.Stop()
	hookResult := runAddHooks(sourceWorktree, worktreePath)

	if switchTo {
		fmt.Println(worktreePath) // Raw path for shell wrapper to cd into
	} else {
		logger.Success("Created worktree at %s", styles.RenderPath(worktreePath))
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
		return git.HintGitTooOld(fmt.Errorf("failed to create detached worktree: %w", err))
	}

	// Note: Auto-lock not applied for detached worktrees (no branch to lock)

	spin := logger.StartSpinner("Setting up worktree...")
	preserveResult := preserveFilesFromSource(sourceWorktree, worktreePath)
	spin.Stop()
	hookResult := runAddHooks(sourceWorktree, worktreePath)

	if switchTo {
		fmt.Println(worktreePath)
	} else {
		logger.Success("Created detached worktree at %s", styles.RenderPath(worktreePath))
		logPreserveResult(preserveResult)
		logHookResult(hookResult)
	}
	return nil
}

func runAddFromPR(prRef string, switchTo bool, name, bareDir, workspaceRoot, sourceWorktree string, reset bool) error {
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
	spin := logger.StartSpinner(fmt.Sprintf("Fetching PR #%d from %s/%s...", ref.Number, owner, repo))
	prInfo, err := github.FetchPRInfo(owner, repo, ref.Number)
	if err != nil {
		spin.StopWithError("Failed to fetch PR info")
		return err
	}
	spin.Stop()

	branch := prInfo.HeadRef
	dirName := name
	if dirName == "" {
		dirName = fmt.Sprintf("pr-%d", ref.Number)
	}
	worktreePath := filepath.Join(workspaceRoot, dirName)

	// Check if worktree already exists
	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}
	for _, info := range infos {
		if info.Branch == branch {
			return fmt.Errorf("worktree already exists for branch %q at %s\n\nHint: Use 'grove list' to see existing worktrees, or use --name to choose a different directory", branch, info.Path)
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

			spin = logger.StartSpinner(fmt.Sprintf("Adding remote %s for fork...", remoteName))
			if err := git.AddRemote(bareDir, remoteName, remoteURL); err != nil {
				spin.StopWithError("Failed to add remote")
				return fmt.Errorf("failed to add fork remote: %w", err)
			}
			spin.Stop()
			addedRemote = true
		}

		// Cleanup helper: remove remote if we added it and something fails
		cleanupRemote := func() {
			if addedRemote {
				logger.Debug("Cleaning up remote %s after failure", remoteName)
				_ = git.RemoveRemote(bareDir, remoteName)
			}
		}

		spin = logger.StartSpinner(fmt.Sprintf("Fetching branch %s from fork...", branch))
		if err := git.FetchBranch(bareDir, remoteName, branch); err != nil {
			spin.StopWithError("Failed to fetch branch")
			cleanupRemote()
			return fmt.Errorf("failed to fetch fork branch: %w", err)
		}
		spin.Stop()

		// Create worktree tracking the fork's branch
		trackingRef := fmt.Sprintf("%s/%s", remoteName, branch)
		if err := git.CreateWorktree(bareDir, worktreePath, trackingRef, true); err != nil {
			cleanupRemote()
			return git.HintGitTooOld(fmt.Errorf("failed to create worktree: %w", err))
		}
		if err := git.SetUpstreamBranch(worktreePath, trackingRef); err != nil {
			logger.Debug("Failed to set upstream for %s: %v", branch, err)
		}
	} else {
		// Same-repo PR: fetch and create worktree
		spin = logger.StartSpinner(fmt.Sprintf("Fetching branch %s...", branch))
		if err := git.FetchBranch(bareDir, "origin", branch); err != nil {
			spin.StopWithError("Failed to fetch branch")
			return fmt.Errorf("failed to fetch branch: %w", err)
		}
		spin.Stop()

		// Resolve FETCH_HEAD to a commit hash immediately to avoid race conditions.
		// Another fetch could overwrite FETCH_HEAD between our fetch and comparison.
		fetchedHash, err := git.RevParse(bareDir, "FETCH_HEAD")
		if err != nil {
			return fmt.Errorf("failed to resolve fetched commit: %w", err)
		}

		// Check for diverged local branch before creating worktree.
		// If local branch exists and has commits not on remote, fail unless --reset is used.
		localExists, err := git.LocalBranchExists(bareDir, branch)
		if err != nil {
			return fmt.Errorf("failed to check local branch: %w", err)
		}

		if localExists {
			ahead, _, err := git.CompareBranchRefs(bareDir, branch, fetchedHash)
			if err != nil {
				return fmt.Errorf("failed to compare branches: %w", err)
			}

			if ahead > 0 {
				if !reset {
					return fmt.Errorf("local branch %q has %d commit(s) not on remote (PR may have been rebased); use --reset to discard local commits and sync with remote", branch, ahead)
				}
				spin = logger.StartSpinner(fmt.Sprintf("Resetting %s to match remote (discarding %d local commits)...", branch, ahead))
				if err := git.UpdateBranchRef(bareDir, branch, fetchedHash); err != nil {
					spin.StopWithError("Failed to reset branch")
					return fmt.Errorf("failed to reset branch: %w", err)
				}
				spin.Stop()
			}
		}

		if err := git.CreateWorktree(bareDir, worktreePath, branch, true); err != nil {
			return git.HintGitTooOld(fmt.Errorf("failed to create worktree: %w", err))
		}
		if err := git.SetUpstreamBranch(worktreePath, "origin/"+branch); err != nil {
			logger.Debug("Failed to set upstream for %s: %v", branch, err)
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

	setupSpin := logger.StartSpinner("Setting up worktree...")
	preserveResult := preserveFilesFromSource(sourceWorktree, worktreePath)
	setupSpin.Stop()
	hookResult := runAddHooks(sourceWorktree, worktreePath)

	if switchTo {
		fmt.Println(worktreePath)
	} else {
		logger.Success("Created worktree for PR #%d at %s", ref.Number, styles.RenderPath(worktreePath))
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
	if fs.PathsEqual(cwd, workspaceRoot) {
		return ""
	}

	if git.IsWorktree(cwd) {
		return cwd
	}

	dir := cwd
	for !fs.PathsEqual(dir, workspaceRoot) && filepath.Dir(dir) != dir {
		if git.IsWorktree(dir) {
			return dir
		}
		dir = filepath.Dir(dir)
	}

	return ""
}

// findFallbackSourceWorktree returns a worktree to use as source for file
// preservation when the user isn't inside a worktree.
//
// Priority:
//  1. Worktree checked out on default/main/master branch
//  2. Worktree in directory named main/master (handles case where primary
//     worktree is temporarily checked out on a feature branch)
func findFallbackSourceWorktree(bareDir string) string {
	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil || len(infos) == 0 {
		return ""
	}

	// Build candidate list: configured default branch, then common defaults
	// Avoid duplicates if default branch is main or master
	candidates := []string{}
	if defaultBranch, err := git.GetDefaultBranch(bareDir); err == nil && defaultBranch != "" {
		candidates = append(candidates, defaultBranch)
	}
	for _, branch := range []string{"main", "master"} {
		if len(candidates) == 0 || candidates[0] != branch {
			candidates = append(candidates, branch)
		}
	}

	// First, try to find worktree by branch name
	for _, branch := range candidates {
		for _, info := range infos {
			if info.Branch == branch {
				return info.Path
			}
		}
	}

	// Fall back to finding worktree by directory name (handles case where the
	// primary worktree is checked out on a feature branch)
	for _, dirName := range candidates {
		for _, info := range infos {
			if filepath.Base(info.Path) == dirName {
				return info.Path
			}
		}
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
	excludePatterns := config.GetMergedPreserveExcludePatterns(sourceWorktree)

	// Copy preserved files
	result, err := workspace.PreserveFilesToWorktree(sourceWorktree, destWorktree, patterns, ignoredFiles, excludePatterns)
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

	logger.Info("Running %d hook(s)...", len(addHooks))
	return hooks.RunAddHooksStreaming(destWorktree, addHooks, os.Stderr)
}

func logHookResult(result *hooks.RunResult) {
	if result == nil {
		return
	}

	if result.Failed != nil {
		logger.Warning("Hook failed: %s (exit code %d)", result.Failed.Command, result.Failed.ExitCode)
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

func completeFromWorktree(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
		name := filepath.Base(info.Path)
		if strings.HasPrefix(name, toComplete) {
			completions = append(completions, name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
