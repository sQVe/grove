package app

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/commands"
	"github.com/sqve/grove/internal/completion"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
)

const Version = "v0.1.0"

// NewRootCommand creates and configures the Grove root command
func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     "grove",
		Short:   "Fast, intuitive Git worktree management",
		Version: Version,
		Long: `Grove transforms Git worktrees from a power-user feature into an essential productivity tool.
Manage multiple working directories effortlessly with smart cleanup and seamless integration.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Grove - Git worktree management")
			fmt.Println("Run 'grove --help' for usage information")
		},
	}

	setupRootCommand(rootCmd)
	return rootCmd
}

// setupRootCommand configures flags, commands, and initialization for the root command
func setupRootCommand(rootCmd *cobra.Command) {
	// Disable automatic error printing to avoid duplicate error messages
	rootCmd.SilenceErrors = true

	setupFlags(rootCmd)
	setupInitialization(rootCmd)
	registerCommands(rootCmd)
	setupCompletion(rootCmd)
}

// setupFlags adds persistent flags to the root command
func setupFlags(rootCmd *cobra.Command) {
	rootCmd.PersistentFlags().String("log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().String("log-format", "text", "Log format (text, json)")
	rootCmd.PersistentFlags().Bool("debug", false, "Enable debug logging (shorthand for --log-level=debug)")
}

// setupInitialization configures cobra initialization callback
func setupInitialization(rootCmd *cobra.Command) {
	cobra.OnInitialize(func() { InitializeConfig(rootCmd) })
}

// registerCommands adds all subcommands to the root command
func registerCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(commands.NewInitCmd())
	rootCmd.AddCommand(commands.NewConfigCmd())
	rootCmd.AddCommand(commands.NewListCmd())
	rootCmd.AddCommand(commands.NewCreateCmd())
}

// setupCompletion configures shell completion for the root command
func setupCompletion(rootCmd *cobra.Command) {
	completion.CreateCompletionCommands(rootCmd)
	completion.RegisterCompletionFunctions(rootCmd, git.DefaultExecutor)
}

// InitializeConfig initializes application configuration and logging
func InitializeConfig(rootCmd *cobra.Command) {
	if err := config.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing config: %v\n", err)
		os.Exit(1)
	}

	bindFlags(rootCmd)
	configureLogging(rootCmd)
}

// bindFlags binds cobra flags to viper configuration
func bindFlags(rootCmd *cobra.Command) {
	if err := viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("log-level")); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to bind log-level flag: %v\n", err)
	}
	if err := viper.BindPFlag("logging.format", rootCmd.PersistentFlags().Lookup("log-format")); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to bind log-format flag: %v\n", err)
	}
}

// configureLogging sets up application logging based on flags and configuration
func configureLogging(rootCmd *cobra.Command) {
	// Handle debug flag override for enhanced development experience
	if debug, _ := rootCmd.PersistentFlags().GetBool("debug"); debug {
		viper.Set("logging.level", "debug")
	}

	loggerConfig := logger.Config{
		Level:  config.GetString("logging.level"),
		Format: config.GetString("logging.format"),
		Output: os.Stderr,
	}

	logger.Configure(loggerConfig)
}
