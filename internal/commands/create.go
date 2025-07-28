package commands

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sqve/grove/internal/commands/create"
)

func NewCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "create [branch-name|url] [path]",
		Short: "Create a new Git worktree from a branch or URL",
		Long: `Create a new Git worktree from an existing or new branch with intelligent automation.

Basic usage:
  grove create feature-branch              # Create worktree for existing branch
  grove create feature-branch ./custom     # Create at specific path
  grove create --create new-feature        # Create new branch without prompting
  grove create --base main new-feature     # Create from specific base branch

URL and remote branch support:
  grove create https://github.com/owner/repo/pull/123
  grove create https://gitlab.com/owner/repo/-/merge_requests/456
  grove create https://bitbucket.org/owner/repo/pull-requests/789
  grove create https://github.com/owner/repo/tree/feature-branch
  grove create origin/feature-branch
  grove create upstream/hotfix-123

Supported platforms: GitHub, GitLab, Bitbucket, Azure DevOps, Codeberg, Gitea

File copying options:
  grove create feature-branch --copy-env               # Copy environment files (.env*, *.local.*)
  grove create feature-branch --copy ".env*,.vscode/"  # Copy specific patterns
  grove create feature-branch --no-copy                # Skip all file copying
  grove create feature-branch --source main            # Copy from specific source worktree

File copying patterns support glob syntax. Common patterns:
  .env*             # All environment files
  .vscode/          # VS Code settings
  .idea/            # IntelliJ IDEA settings
  *.local.*         # Local configuration files
  .gitignore.local  # Local gitignore

By default, files are copied from the main worktree based on configuration.
Use --no-copy to disable, --copy-env for quick environment setup, or --copy with
custom patterns for specific files.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := create.ValidateCreateArgs(args); err != nil {
				return err
			}

			noCopy, _ := cmd.Flags().GetBool("no-copy")
			copyEnv, _ := cmd.Flags().GetBool("copy-env")
			copyPatterns, _ := cmd.Flags().GetString("copy")

			return create.ValidateFlags(noCopy, copyEnv, copyPatterns)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options, err := create.ParseCreateOptions(cmd, args)
			if err != nil {
				return err
			}

			// TODO: Initialize services and execute create command.
			fmt.Printf("Creating worktree for: %s\n", options.BranchName)
			if options.WorktreePath != "" {
				fmt.Printf("Path: %s\n", options.WorktreePath)
			}
			fmt.Println("Implementation in progress...")

			return nil
		},
	}

	cmd.Flags().BoolP("create", "c", false, "Create new branch without prompting")
	cmd.Flags().String("base", "", "Base branch for new branch creation (default: current branch)")
	cmd.Flags().Bool("force", false, "Force creation even if path exists")
	cmd.Flags().Bool("copy-env", false, "Copy common environment files (.env*, *.local.*, docker-compose.override.yml)")
	cmd.Flags().String("copy", "", "Comma-separated glob patterns to copy (supports .env*,.vscode/,.idea/ etc.)")
	cmd.Flags().Bool("no-copy", false, "Skip all file copying (overrides config and other copy flags)")
	cmd.Flags().String("source", "", "Source worktree path for file copying (default: main worktree from config)")

	create.RegisterCreateCompletion(cmd)

	return cmd
}
