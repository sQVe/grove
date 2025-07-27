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
  grove create https://github.com/owner/repo/pull/123    # Create from GitHub PR
  grove create origin/feature-branch        # Create from remote branch

File copying options:
  grove create feature-branch --copy-env    # Copy environment files
  grove create feature-branch --copy ".env,.vscode/"  # Copy specific patterns
  grove create feature-branch --no-copy     # Skip file copying

See 'grove create --help' for complete examples and supported platforms.`,
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
	cmd.Flags().String("base", "", "Base branch for new branch creation")
	cmd.Flags().Bool("force", false, "Force creation even if path exists")
	cmd.Flags().Bool("copy-env", false, "Copy environment files (.env*, .local.*, etc.)")
	cmd.Flags().String("copy", "", "Comma-separated patterns to copy (e.g. '.env,.vscode/')")
	cmd.Flags().Bool("no-copy", false, "Skip all file copying")
	cmd.Flags().String("source", "", "Source worktree for file copying")

	create.RegisterCreateCompletion(cmd)

	return cmd
}
