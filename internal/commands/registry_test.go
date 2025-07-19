package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockCommand is a test implementation of the Command interface.
type MockCommand struct {
	name           string
	cmd            *cobra.Command
	requiresConfig bool
}

func (m *MockCommand) Name() string                { return m.name }
func (m *MockCommand) Command() *cobra.Command     { return m.cmd }
func (m *MockCommand) RequiresConfig() bool        { return m.requiresConfig }

func newMockCommand(name string, requiresConfig bool) *MockCommand {
	cmd := &cobra.Command{
		Use:   name,
		Short: "Mock command for testing",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	return &MockCommand{
		name:           name,
		cmd:            cmd,
		requiresConfig: requiresConfig,
	}
}

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.commands)
	assert.Empty(t, registry.commands)
}

func TestRegistry_Register(t *testing.T) {
	tests := []struct {
		name        string
		command     Command
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid command",
			command:     newMockCommand("test", false),
			expectError: false,
		},
		{
			name:        "empty name",
			command:     newMockCommand("", false),
			expectError: true,
			errorMsg:    "command name cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			err := registry.Register(tt.command)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				
				// Verify command is registered
				cmd, exists := registry.Get(tt.command.Name())
				assert.True(t, exists)
				assert.Equal(t, tt.command, cmd)
			}
		})
	}
}

func TestRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRegistry()
	cmd1 := newMockCommand("test", false)
	cmd2 := newMockCommand("test", true)

	// Register first command
	err := registry.Register(cmd1)
	require.NoError(t, err)

	// Try to register second command with same name
	err = registry.Register(cmd2)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command test is already registered")

	// Verify original command is still there
	cmd, exists := registry.Get("test")
	assert.True(t, exists)
	assert.Equal(t, cmd1, cmd)
}

func TestRegistry_Get(t *testing.T) {
	registry := NewRegistry()
	testCmd := newMockCommand("test", false)

	// Get non-existent command
	cmd, exists := registry.Get("nonexistent")
	assert.False(t, exists)
	assert.Nil(t, cmd)

	// Register and get existing command
	err := registry.Register(testCmd)
	require.NoError(t, err)

	cmd, exists = registry.Get("test")
	assert.True(t, exists)
	assert.Equal(t, testCmd, cmd)
}

func TestRegistry_List(t *testing.T) {
	registry := NewRegistry()

	// Empty registry
	names := registry.List()
	assert.Empty(t, names)

	// Add commands in non-alphabetical order
	commands := []*MockCommand{
		newMockCommand("zebra", false),
		newMockCommand("alpha", true),
		newMockCommand("beta", false),
	}

	for _, cmd := range commands {
		err := registry.Register(cmd)
		require.NoError(t, err)
	}

	// Verify sorted order
	names = registry.List()
	expected := []string{"alpha", "beta", "zebra"}
	assert.Equal(t, expected, names)
}

func TestRegistry_Commands(t *testing.T) {
	registry := NewRegistry()
	cmd1 := newMockCommand("test1", false)
	cmd2 := newMockCommand("test2", true)

	// Register commands
	err := registry.Register(cmd1)
	require.NoError(t, err)
	err = registry.Register(cmd2)
	require.NoError(t, err)

	// Get all commands
	commands := registry.Commands()
	assert.Len(t, commands, 2)
	assert.Equal(t, cmd1, commands["test1"])
	assert.Equal(t, cmd2, commands["test2"])

	// Verify it's a copy (modifying returned map doesn't affect registry)
	delete(commands, "test1")
	
	cmd, exists := registry.Get("test1")
	assert.True(t, exists)
	assert.Equal(t, cmd1, cmd)
}

func TestRegistry_AttachToRoot(t *testing.T) {
	registry := NewRegistry()
	rootCmd := &cobra.Command{Use: "root"}

	// Empty registry
	err := registry.AttachToRoot(rootCmd)
	require.NoError(t, err)
	assert.Empty(t, rootCmd.Commands())

	// Add commands
	cmd1 := newMockCommand("test1", false)
	cmd2 := newMockCommand("test2", true)
	
	err = registry.Register(cmd1)
	require.NoError(t, err)
	err = registry.Register(cmd2)
	require.NoError(t, err)

	// Attach to root
	err = registry.AttachToRoot(rootCmd)
	require.NoError(t, err)

	// Verify commands are attached
	commands := rootCmd.Commands()
	assert.Len(t, commands, 2)
	
	commandNames := make([]string, len(commands))
	for i, cmd := range commands {
		commandNames[i] = cmd.Use
	}
	assert.Contains(t, commandNames, "test1")
	assert.Contains(t, commandNames, "test2")
}

func TestRegistry_AttachToRoot_NilCommand(t *testing.T) {
	registry := NewRegistry()
	rootCmd := &cobra.Command{Use: "root"}

	// Create a mock command that returns nil
	mockCmd := &MockCommand{
		name:           "badcmd",
		cmd:            nil, // This will cause an error
		requiresConfig: false,
	}

	err := registry.Register(mockCmd)
	require.NoError(t, err)

	// Try to attach - should fail
	err = registry.AttachToRoot(rootCmd)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "command badcmd returned nil cobra.Command")
}

func TestDefaultRegistry_Functions(t *testing.T) {
	// Save original state
	originalCommands := DefaultRegistry.Commands()
	
	// Reset registry for testing
	DefaultRegistry = NewRegistry()
	
	// Restore after test
	defer func() {
		DefaultRegistry = NewRegistry()
		for name, cmd := range originalCommands {
			DefaultRegistry.commands[name] = cmd
		}
	}()

	testCmd := newMockCommand("globaltest", false)

	// Test global Register function
	err := Register(testCmd)
	require.NoError(t, err)

	// Test global Get function
	cmd, exists := Get("globaltest")
	assert.True(t, exists)
	assert.Equal(t, testCmd, cmd)

	// Test global List function
	names := List()
	assert.Contains(t, names, "globaltest")

	// Test global AttachToRoot function
	rootCmd := &cobra.Command{Use: "root"}
	err = AttachToRoot(rootCmd)
	require.NoError(t, err)
	
	commands := rootCmd.Commands()
	assert.Len(t, commands, 1)
	assert.Equal(t, "globaltest", commands[0].Use)
}

func TestBuiltinCommands(t *testing.T) {
	// Create a fresh registry for testing
	registry := NewRegistry()
	
	// Test command creation
	initCmd := NewInitCommand()
	assert.Equal(t, "init", initCmd.Name())
	assert.NotNil(t, initCmd.Command())
	assert.False(t, initCmd.RequiresConfig())

	configCmd := NewConfigCommand()
	assert.Equal(t, "config", configCmd.Name())
	assert.NotNil(t, configCmd.Command())
	assert.True(t, configCmd.RequiresConfig())

	// Test registration
	err := registry.Register(initCmd)
	require.NoError(t, err)
	
	err = registry.Register(configCmd)
	require.NoError(t, err)

	// Verify they were registered correctly
	names := registry.List()
	assert.Contains(t, names, "init")
	assert.Contains(t, names, "config")
}

func TestRegisterBuiltinCommands(t *testing.T) {
	// Save original state
	originalCommands := DefaultRegistry.Commands()
	
	// Reset registry for testing
	DefaultRegistry = NewRegistry()
	
	// Restore after test
	defer func() {
		DefaultRegistry = NewRegistry()
		for name, cmd := range originalCommands {
			DefaultRegistry.commands[name] = cmd
		}
	}()

	// Register builtin commands
	err := RegisterBuiltinCommands()
	require.NoError(t, err)

	// Verify all builtin commands are registered
	names := List()
	assert.Contains(t, names, "init")
	assert.Contains(t, names, "config")

	// Verify commands work correctly
	initCmd, exists := Get("init")
	assert.True(t, exists)
	assert.Equal(t, "init", initCmd.Name())
	assert.False(t, initCmd.RequiresConfig())

	configCmd, exists := Get("config")
	assert.True(t, exists)
	assert.Equal(t, "config", configCmd.Name())
	assert.True(t, configCmd.RequiresConfig())
}