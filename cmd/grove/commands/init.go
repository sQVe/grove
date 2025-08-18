package commands

import "github.com/spf13/cobra"

func NewInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize grove in current repository",
		Run: func(cmd *cobra.Command, args []string) {
			// Minimal implementation - do nothing for now
		},
	}
}
