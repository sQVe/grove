package commands

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/hooks"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create <branch>",
		Short: "Create a new worktree",
		Long: `Create a new worktree for a branch.

If the branch exists (locally or on remote), creates a worktree for it.
If the branch doesn't exist, creates both the branch and worktree.

Examples:
  grove create feature/auth        # Create worktree for new branch
  grove create main                # Create worktree for existing branch
  grove create -s feature/auth     # Create and switch to worktree`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeCreateArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			switchTo, _ := cmd.Flags().GetBool("switch")
			return runCreate(args[0], switchTo)
		},
	}

	cmd.Flags().BoolP("switch", "s", false, "Switch to the new worktree after creation")
	cmd.Flags().BoolP("help", "h", false, "Help for create")

	return cmd
}

func runCreate(branch string, switchTo bool) error {
	branch = strings.TrimSpace(branch)

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

	sourceWorktree := findSourceWorktree(cwd, workspaceRoot)
	dirName := sanitizeBranchName(branch)
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
		if err := git.CreateWorktree(bareDir, worktreePath, branch, true); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		if err := git.CreateWorktreeWithNewBranch(bareDir, worktreePath, branch, true); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	preserveResult := preserveFilesFromSource(sourceWorktree, worktreePath)
	hookResult := runCreateHooks(sourceWorktree, worktreePath)

	if switchTo {
		fmt.Println(worktreePath) // Path for shell wrapper to cd into
	} else {
		logger.Success("Created worktree at %s", worktreePath)
		logPreserveResult(preserveResult)
		logHookResult(hookResult)
	}
	return nil
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
		if len(result.Copied) == 1 {
			logger.ListItem("Preserved %d file", len(result.Copied))
		} else {
			logger.ListItem("Preserved %d files", len(result.Copied))
		}
		for _, f := range result.Copied {
			logger.ListSubItem("%s", f)
		}
	}

	if len(result.Skipped) > 0 {
		if len(result.Skipped) == 1 {
			logger.Warning("Skipped %d file (already exists)", len(result.Skipped))
		} else {
			logger.Warning("Skipped %d files (already exist)", len(result.Skipped))
		}
		for _, f := range result.Skipped {
			logger.ListSubItem("%s", f)
		}
	}
}

func runCreateHooks(sourceWorktree, destWorktree string) *hooks.RunResult {
	var createHooks []string
	if sourceWorktree != "" {
		createHooks = hooks.GetCreateHooks(sourceWorktree)
	}

	if len(createHooks) == 0 {
		logger.Debug("No create hooks configured")
		return nil
	}

	logger.Debug("Found %d create hooks", len(createHooks))
	return hooks.RunCreateHooks(destWorktree, createHooks)
}

func logHookResult(result *hooks.RunResult) {
	if result == nil {
		return
	}

	if len(result.Succeeded) > 0 {
		for _, cmd := range result.Succeeded {
			logger.ListItem("Ran %s", cmd)
		}
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

func sanitizeBranchName(branch string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		"<", "-",
		">", "-",
		"|", "-",
		`"`, "-",
		"?", "-",
		"*", "-",
		":", "-",
	)
	return replacer.Replace(branch)
}

func completeCreateArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
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
