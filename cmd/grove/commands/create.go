package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
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
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Get workspace root (parent of .bare)
	workspaceRoot := filepath.Dir(bareDir)

	// Sanitize branch name for directory
	dirName := sanitizeBranchName(branch)
	worktreePath := filepath.Join(workspaceRoot, dirName)

	// Check if branch already has a worktree (check this first for better error message)
	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}
	for _, info := range infos {
		if info.Branch == branch {
			return fmt.Errorf("worktree already exists for branch %q at %s", branch, info.Path)
		}
	}

	// Check if worktree directory already exists
	if _, err := os.Stat(worktreePath); err == nil {
		return fmt.Errorf("directory already exists: %s", worktreePath)
	}

	// Check if branch exists
	exists, err := git.BranchExists(bareDir, branch)
	if err != nil {
		return fmt.Errorf("failed to check branch: %w", err)
	}

	if exists {
		// Create worktree for existing branch
		if err := git.CreateWorktree(bareDir, worktreePath, branch, true); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	} else {
		// Create worktree with new branch
		if err := git.CreateWorktreeWithNewBranch(bareDir, worktreePath, branch, true); err != nil {
			return fmt.Errorf("failed to create worktree: %w", err)
		}
	}

	if switchTo {
		// Output just the path for shell wrapper to cd into
		fmt.Println(worktreePath)
	} else {
		logger.Success("Created worktree at %s", worktreePath)
	}
	return nil
}

// sanitizeBranchName converts branch name to safe directory name
func sanitizeBranchName(branch string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"<", "-",
		">", "-",
		"|", "-",
		`"`, "-",
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

	// Get all branches (for existing branch completion)
	branches, err := git.ListBranches(bareDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Get existing worktrees to exclude
	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	existingBranches := make(map[string]bool)
	for _, info := range infos {
		existingBranches[info.Branch] = true
	}

	// Filter out branches that already have worktrees
	var completions []string
	for _, b := range branches {
		if !existingBranches[b] {
			completions = append(completions, b)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
