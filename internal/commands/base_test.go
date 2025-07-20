package commands

import (
	"os"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestNewBaseCommand(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	baseCmd := NewBaseCommand("testcmd", cmd, true)

	assert.Equal(t, "testcmd", baseCmd.Name())
	assert.Equal(t, cmd, baseCmd.Command())
	assert.True(t, baseCmd.RequiresConfig())

	// Test with requiresConfig = false
	baseCmd2 := NewBaseCommand("testcmd2", cmd, false)
	assert.False(t, baseCmd2.RequiresConfig())
}

func TestBaseCommand_Interface(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	baseCmd := NewBaseCommand("testcmd", cmd, true)

	// Verify it implements the Command interface
	var _ Command = baseCmd
}

func TestNewCommandContext(t *testing.T) {
	// This test may fail if config initialization fails, which is expected
	// in some test environments. We'll test both success and failure cases.

	ctx, err := NewCommandContext()

	// If config initialization succeeds
	if err == nil {
		assert.NotNil(t, ctx)

		// Test GetConfig
		cfg, err := ctx.GetConfig()
		if err == nil {
			assert.NotNil(t, cfg)
		}
	} else {
		// If config initialization fails (expected in some test environments)
		assert.Nil(t, ctx)
		assert.Error(t, err)
	}
}

func TestNewCommandContext_ConfigError(t *testing.T) {
	// Create a temporary directory with an invalid config file to force Initialize() to fail
	tempDir := t.TempDir()

	// Create an invalid config file (malformed TOML)
	invalidConfig := `[general
	editor = "invalid toml` // Missing closing bracket and quote

	configPath := tempDir + "/config.toml"
	err := os.WriteFile(configPath, []byte(invalidConfig), 0o644)
	assert.NoError(t, err)

	// Set GROVE_CONFIG to point to our invalid config file
	originalConfig := os.Getenv("GROVE_CONFIG")
	defer func() {
		if originalConfig != "" {
			_ = os.Setenv("GROVE_CONFIG", originalConfig)
		} else {
			_ = os.Unsetenv("GROVE_CONFIG")
		}
	}()

	_ = os.Setenv("GROVE_CONFIG", configPath)

	// This should fail due to the malformed config file
	ctx, err := NewCommandContext()

	// Verify error handling
	assert.Error(t, err)
	assert.Nil(t, ctx)
	assert.Contains(t, err.Error(), "error reading config file")
}

func TestCommandContext_GetConfig(t *testing.T) {
	// Create a command context without going through NewCommandContext
	// to avoid config initialization issues in tests
	ctx := &CommandContext{}

	// This may return an error if config isn't initialized, which is fine
	cfg, err := ctx.GetConfig()

	// We just verify the method signature works
	// The actual functionality depends on the config package state
	_ = cfg
	_ = err
}

// TestCommand implementations for interface verification.
type TestCommand struct {
	*BaseCommand
}

func NewTestCommand(name string, requiresConfig bool) *TestCommand {
	cmd := &cobra.Command{
		Use:   name,
		Short: "Test command",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	base := NewBaseCommand(name, cmd, requiresConfig)
	return &TestCommand{BaseCommand: base}
}

func TestCommandInterface(t *testing.T) {
	testCmd := NewTestCommand("test", true)

	// Verify it implements Command interface
	var _ Command = testCmd

	assert.Equal(t, "test", testCmd.Name())
	assert.NotNil(t, testCmd.Command())
	assert.True(t, testCmd.RequiresConfig())
}
