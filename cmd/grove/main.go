package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

func main() {
	config.LoadFromEnv()

	rootCmd := &cobra.Command{
		Use:   "grove",
		Short: "Grove - Git worktree management made simple",
		Long:  `Grove is a CLI tool that makes Git worktrees as simple as switching branches.`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if plain, _ := cmd.Flags().GetBool("plain"); plain && !config.Global.Plain {
				config.Global.Plain = true
			}
			if debug, _ := cmd.Flags().GetBool("debug"); debug && !config.Global.Debug {
				config.Global.Debug = true
			}
			logger.Debug("Grove CLI starting with config: plain=%v, debug=%v",
				config.IsPlain(), config.IsDebug())
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
			}
		},
	}

	rootCmd.PersistentFlags().Bool("plain", false, "Disable colors, emojis, and formatting")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize grove in current repository",
		Long:  `Initialize grove in current repository`,
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

	rootCmd.AddCommand(initCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
