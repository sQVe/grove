package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/commands"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
)

var rootCmd = &cobra.Command{
	Use:     "grove",
	Short:   "Fast, intuitive Git worktree management",
	Version: "v0.1.0",
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
	// Initialize Viper configuration
	if err := config.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	// Bind command line flags to Viper
	if err := viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level")); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to bind log-level flag: %v\n", err)
	}
	if err := viper.BindPFlag("logging.format", rootCmd.PersistentFlags().Lookup("log-format")); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to bind log-format flag: %v\n", err)
	}

	// Handle debug flag override
	if debug, _ := rootCmd.PersistentFlags().GetBool("debug"); debug {
		viper.Set("logging.level", "debug")
	}

	// Configure the global logger using Viper values
	loggerConfig := logger.Config{
		Level:  config.GetString("logging.level"),
		Format: config.GetString("logging.format"),
		Output: os.Stderr,
	}

	logger.Configure(loggerConfig)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
