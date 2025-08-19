package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/cmd/grove/commands"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
)

func main() {
	config.LoadFromEnv()

	rootCmd := &cobra.Command{
		Use:           "grove",
		Short:         "Grove - Git worktree management made simple",
		Long:          `Grove is a tool that makes Git worktrees as simple as switching branches.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if plain, _ := cmd.Flags().GetBool("plain"); plain && !config.Global.Plain {
				config.Global.Plain = true
			}
			if debug, _ := cmd.Flags().GetBool("debug"); debug && !config.Global.Debug {
				config.Global.Debug = true
			}
			logger.Debug("Grove starting with config: plain=%v, debug=%v",
				config.IsPlain(), config.IsDebug())
		},
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				logger.Error("Failed to display help: %v", err)
			}
		},
	}

	rootCmd.PersistentFlags().Bool("plain", false, "Disable colors, emojis, and formatting")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")

	rootCmd.AddCommand(commands.NewInitCmd())

	if err := rootCmd.ParseFlags(os.Args[1:]); err == nil {
		if plain, _ := rootCmd.Flags().GetBool("plain"); plain {
			config.Global.Plain = true
		}
		if debug, _ := rootCmd.Flags().GetBool("debug"); debug {
			config.Global.Debug = true
		}
	}

	if err := rootCmd.Execute(); err != nil {
		logger.Error("%s", err)
		logger.Dimmed("Run 'grove --help' for usage.")
		os.Exit(1)
	}
}
