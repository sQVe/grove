package commands

import (
	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
)

// Command represents a Grove command that can be registered with the command registry.
type Command interface {
	// Name returns the command name (e.g., "init", "config", "list").
	Name() string

	// Command returns the cobra.Command instance for this command.
	Command() *cobra.Command

	// RequiresConfig indicates whether this command requires configuration to be initialized.
	RequiresConfig() bool
}

// BaseCommand provides common functionality for all Grove commands.
type BaseCommand struct {
	name           string
	cmd            *cobra.Command
	requiresConfig bool
}

// NewBaseCommand creates a new BaseCommand with the given parameters.
func NewBaseCommand(name string, cmd *cobra.Command, requiresConfig bool) *BaseCommand {
	return &BaseCommand{
		name:           name,
		cmd:            cmd,
		requiresConfig: requiresConfig,
	}
}

// Name returns the command name.
func (b *BaseCommand) Name() string {
	return b.name
}

// Command returns the cobra.Command instance.
func (b *BaseCommand) Command() *cobra.Command {
	return b.cmd
}

// RequiresConfig indicates whether this command requires configuration.
func (b *BaseCommand) RequiresConfig() bool {
	return b.requiresConfig
}

// CommandContext provides shared context and utilities for command execution.
type CommandContext struct {
	Config config.Config
}

// NewCommandContext creates a new command context.
func NewCommandContext() (*CommandContext, error) {
	// Initialize configuration if needed
	if err := config.Initialize(); err != nil {
		return nil, err
	}

	return &CommandContext{}, nil
}

// GetConfig returns the current configuration.
func (ctx *CommandContext) GetConfig() (*config.Config, error) {
	return config.Get()
}
