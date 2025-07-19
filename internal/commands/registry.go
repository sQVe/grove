package commands

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

// Registry manages the registration and discovery of Grove commands.
type Registry struct {
	commands map[string]Command
}

// NewRegistry creates a new command registry.
func NewRegistry() *Registry {
	return &Registry{
		commands: make(map[string]Command),
	}
}

// Register adds a command to the registry.
func (r *Registry) Register(cmd Command) error {
	name := cmd.Name()
	if name == "" {
		return fmt.Errorf("command name cannot be empty")
	}

	if _, exists := r.commands[name]; exists {
		return fmt.Errorf("command %s is already registered", name)
	}

	r.commands[name] = cmd
	return nil
}

// Get retrieves a command by name.
func (r *Registry) Get(name string) (Command, bool) {
	cmd, exists := r.commands[name]
	return cmd, exists
}

// List returns all registered command names in alphabetical order.
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.commands))
	for name := range r.commands {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Commands returns all registered commands.
func (r *Registry) Commands() map[string]Command {
	// Return a copy to prevent external modification
	result := make(map[string]Command, len(r.commands))
	for name, cmd := range r.commands {
		result[name] = cmd
	}
	return result
}

// AttachToRoot attaches all registered commands to the provided root command.
func (r *Registry) AttachToRoot(rootCmd *cobra.Command) error {
	for name, cmd := range r.commands {
		cobraCmd := cmd.Command()
		if cobraCmd == nil {
			return fmt.Errorf("command %s returned nil cobra.Command", name)
		}
		
		rootCmd.AddCommand(cobraCmd)
	}
	return nil
}

// DefaultRegistry is the global command registry instance.
var DefaultRegistry = NewRegistry()

// Register adds a command to the default registry.
func Register(cmd Command) error {
	return DefaultRegistry.Register(cmd)
}

// Get retrieves a command from the default registry.
func Get(name string) (Command, bool) {
	return DefaultRegistry.Get(name)
}

// List returns all command names from the default registry.
func List() []string {
	return DefaultRegistry.List()
}

// AttachToRoot attaches all commands from the default registry to the root command.
func AttachToRoot(rootCmd *cobra.Command) error {
	return DefaultRegistry.AttachToRoot(rootCmd)
}

// InitCommand wraps the init command for registry integration.
type InitCommand struct {
	*BaseCommand
}

// NewInitCommand creates a new InitCommand instance.
func NewInitCommand() Command {
	cmd := NewInitCmd()
	base := NewBaseCommand("init", cmd, false)
	return &InitCommand{BaseCommand: base}
}

// ConfigCommand wraps the config command for registry integration.
type ConfigCommand struct {
	*BaseCommand
}

// NewConfigCommand creates a new ConfigCommand instance.
func NewConfigCommand() Command {
	cmd := NewConfigCmd()
	base := NewBaseCommand("config", cmd, true)
	return &ConfigCommand{BaseCommand: base}
}

// RegisterBuiltinCommands registers all built-in Grove commands with the default registry.
func RegisterBuiltinCommands() error {
	commands := []Command{
		NewInitCommand(),
		NewConfigCommand(),
	}

	for _, cmd := range commands {
		if err := Register(cmd); err != nil {
			return fmt.Errorf("failed to register command %s: %w", cmd.Name(), err)
		}
	}

	return nil
}