package commands

import (
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

func NewInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize grove in current repository",
		Long:  `Initialize grove in current repository`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	newCmd := &cobra.Command{
		Use:   "new [directory]",
		Short: "Create a new grove workspace",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		Run: func(cmd *cobra.Command, args []string) {
			var targetDir string
			if len(args) == 0 {
				var err error
				targetDir, err = os.Getwd()
				if err != nil {
					logger.Error("Failed to get current directory: %v", err)
					os.Exit(1)
				}
			} else {
				targetDir = args[0]
			}

			logger.Debug("Initializing grove workspace in: %s", targetDir)

			if err := workspace.Initialize(targetDir); err != nil {
				logger.Error("Failed to initialize workspace: %v", err)
				os.Exit(1)
			}

			logger.Success("Initialized grove workspace in: %s", targetDir)
		},
	}
	initCmd.AddCommand(newCmd)

	cloneCmd := &cobra.Command{
		Use:   "clone <url> [directory]",
		Short: "Clone a repository and create a grove workspace",
		Args:  cobra.RangeArgs(1, 2),
		Run: func(cmd *cobra.Command, args []string) {
			url := args[0]

			if !isValidGitURL(url) {
				logger.Error("Specified URL is not a valid Git repository URL")
				os.Exit(1)
			}

			logger.Error("Clone functionality not yet implemented")
			os.Exit(1)
		},
	}
	initCmd.AddCommand(cloneCmd)

	return initCmd
}

func isValidGitURL(url string) bool {
	return strings.HasPrefix(url, "http://") ||
		strings.HasPrefix(url, "https://") ||
		strings.HasPrefix(url, "git@") ||
		strings.Contains(url, ".git")
}
