package commands

import (
	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/commands/create"
	initcmd "github.com/sqve/grove/internal/commands/init"
	"github.com/sqve/grove/internal/commands/list"
	"github.com/sqve/grove/internal/commands/shared"
)

// Re-export shared utilities for backward compatibility
var (
	DefaultExecutorProvider = shared.DefaultExecutorProvider
	NewWorktreeFormatter    = shared.NewWorktreeFormatter
)

// NewCreateCmd returns the create command.
func NewCreateCmd() *cobra.Command {
	return create.NewCreateCmd()
}

// NewInitCmd returns the init command.
func NewInitCmd() *cobra.Command {
	return initcmd.NewInitCmd()
}

// NewListCmd returns the list command.
func NewListCmd() *cobra.Command {
	return list.NewListCmd()
}
