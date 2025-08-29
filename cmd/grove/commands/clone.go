package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

// resolveTargetDirectory resolves the target directory from command arguments
func resolveTargetDirectory(args []string, argIndex int) (string, error) {
	if len(args) <= argIndex {
		return os.Getwd()
	}
	return filepath.Abs(args[argIndex])
}

func NewCloneCmd() *cobra.Command {
	var branches string
	var verbose bool

	cloneCmd := &cobra.Command{
		Use:   "clone <url> [directory]",
		Short: "Clone a repository and create a grove workspace",
		Args:  cobra.RangeArgs(1, 2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("branches") && len(args) == 0 {
				return fmt.Errorf("--branches requires a repository URL to be specified")
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("branches") && (branches == "" || branches == `""`) {
				return fmt.Errorf("no branches specified")
			}

			url := args[0]

			targetDir, err := resolveTargetDirectory(args, 1)
			if err != nil {
				return err
			}

			if err := workspace.CloneAndInitialize(url, targetDir, branches, verbose); err != nil {
				return err
			}

			logger.Info("Initialized grove workspace in: %s", styles.Render(&styles.Path, targetDir))
			return nil
		},
	}
	cloneCmd.Flags().StringVar(&branches, "branches", "", "Comma-separated list of branches to create worktrees for")
	cloneCmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed git output during clone and worktree creation")
	cloneCmd.Flags().BoolP("help", "h", false, "Help for clone")

	return cloneCmd
}
