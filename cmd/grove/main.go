package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/commands"
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
	rootCmd.AddCommand(commands.NewInitCmd())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
