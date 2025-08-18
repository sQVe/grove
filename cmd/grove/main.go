package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "grove",
		Short: "Grove - Git worktree management made simple",
		Long:  `Grove is a CLI tool that makes Git worktrees as simple as switching branches.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
			}
		},
	}

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize grove in current repository",
		Long:  `Initialize grove in current repository`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Initializing Grove in current repository...")
		},
	}

	rootCmd.AddCommand(initCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
