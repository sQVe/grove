package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/commands"
	"github.com/sqve/grove/internal/logger"
)

var rootCmd = &cobra.Command{
	Use:   "grove",
	Short: "Fast, intuitive Git worktree management",
	Long: `Grove transforms Git worktrees from a power-user feature into an essential productivity tool.
Manage multiple working directories effortlessly with smart cleanup and seamless integration.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Grove - Git worktree management")
		fmt.Println("Run 'grove --help' for usage information")
	},
}

func init() {
	// Disable automatic error printing to avoid duplicate error messages
	rootCmd.SilenceErrors = true

	// Add logging flags
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "text", "Log format (text, json)")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging (shorthand for --log-level=debug)")

	// Configure logger before running any commands
	cobra.OnInitialize(initConfig)

	rootCmd.AddCommand(commands.NewInitCmd())
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	// Get flag values
	logLevel, _ := rootCmd.PersistentFlags().GetString("log-level")
	logFormat, _ := rootCmd.PersistentFlags().GetString("log-format")
	debug, _ := rootCmd.PersistentFlags().GetBool("debug")

	// If debug flag is set, override log level
	if debug {
		logLevel = "debug"
	}

	// Configure the global logger
	config := logger.Config{
		Level:  logLevel,
		Format: logFormat,
		Output: os.Stderr,
	}

	logger.Configure(config)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
