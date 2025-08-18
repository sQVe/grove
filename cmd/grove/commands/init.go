package commands

import (
	"os"

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
			logger.Debug("Starting initialization process")
			logger.Success("Initialized Grove in current repository")
		},
	}

	newCmd := &cobra.Command{
		Use:   "new [directory]",
		Short: "Create a new grove workspace",
		Args:  cobra.MaximumNArgs(1),
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

			logger.Success("Initialized Grove workspace in %s", targetDir)
		},
	}
	initCmd.AddCommand(newCmd)

	return initCmd
}
