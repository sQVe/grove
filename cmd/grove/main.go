package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
)

func main() {
	// Load configuration from environment variables first
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

	// Add global flags
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

	rootCmd.AddCommand(initCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
