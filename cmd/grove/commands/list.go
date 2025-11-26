package commands

import (
	"github.com/spf13/cobra"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	var fast bool
	var jsonOutput bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all worktrees with status",
		Long:  `Show all worktrees in the grove workspace with their status and sync information.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(fast, jsonOutput, verbose)
		},
	}

	cmd.Flags().BoolVar(&fast, "fast", false, "Skip sync status for faster output")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show extra details (paths, upstream names)")
	cmd.Flags().BoolP("help", "h", false, "Help for list")

	return cmd
}

func runList(fast, jsonOutput, verbose bool) error {
	// TODO: implement
	return nil
}
