package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/cmd/grove/commands"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
)

func main() {
	config.LoadFromGitConfig()
	config.LoadFromEnv()

	rootCmd := &cobra.Command{
		Use:           "grove",
		Short:         "Grove - Git worktree management made simple",
		Long:          `Grove is a tool that makes Git worktrees as simple as switching branches.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("plain") {
				if plain, _ := cmd.Flags().GetBool("plain"); plain {
					config.Global.Plain = true
				} else {
					config.Global.Plain = false
				}
			}
			if cmd.Flags().Changed("debug") {
				if debug, _ := cmd.Flags().GetBool("debug"); debug {
					config.Global.Debug = true
				} else {
					config.Global.Debug = false
				}
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

	rootCmd.PersistentFlags().Bool("plain", false, "Disable colors and symbols")
	rootCmd.PersistentFlags().BoolP("debug", "d", false, "Enable debug logging")
	rootCmd.Flags().BoolP("help", "h", false, "Help for grove")

	rootCmd.AddCommand(commands.NewInitCmd())
	rootCmd.AddCommand(commands.NewCloneCmd())
	rootCmd.AddCommand(commands.NewConfigCmd())
	rootCmd.AddCommand(commands.NewCreateCmd())
	rootCmd.AddCommand(commands.NewListCmd())
	rootCmd.AddCommand(commands.NewSwitchCmd())

	if err := rootCmd.Execute(); err != nil {
		logger.Error("%s", err)
		logger.Dimmed("Run 'grove --help' for usage.")
		os.Exit(1)
	}
}
