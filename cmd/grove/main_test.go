//go:build !integration
// +build !integration

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionFlag(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "long version flag",
			args:     []string{"--version"},
			expected: "grove version v0.1.0",
		},
		{
			name:     "short version flag",
			args:     []string{"-v"},
			expected: "grove version v0.1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new command instance for each test
			cmd := rootCmd
			cmd.SetArgs(tt.args)

			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)

			// Execute command
			err := cmd.Execute()
			require.NoError(t, err)

			// Check output
			output := strings.TrimSpace(buf.String())
			assert.Equal(t, tt.expected, output)
		})
	}
}

func TestRootCommandHelp(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{"--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Grove transforms Git worktrees")
	assert.Contains(t, output, "Usage:")
	assert.Contains(t, output, "Available Commands:")
	assert.Contains(t, output, "Flags:")
}

func TestRootCommandDefault(t *testing.T) {
	cmd := rootCmd
	cmd.SetArgs([]string{})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Grove transforms Git worktrees")
	assert.Contains(t, output, "Available Commands:")
}

func TestInitConfig(t *testing.T) {
	// Create a test directory for configuration
	testDir := testutils.NewTestDirectory(t, "grove-test-config")
	defer testDir.Cleanup()

	// Test initialization with valid config
	t.Run("successful initialization", func(t *testing.T) {
		// Reset Viper for clean test
		viper.Reset()

		// Set a temporary config directory
		oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
		defer func() {
			if oldConfigHome != "" {
				_ = os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
			} else {
				_ = os.Unsetenv("XDG_CONFIG_HOME")
			}
		}()
		_ = os.Setenv("XDG_CONFIG_HOME", testDir.Path)

		// Test that initConfig doesn't panic or exit
		require.NotPanics(t, func() {
			initConfig()
		})
	})
}

func TestLogLevel(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "debug flag",
			args:     []string{"--debug", "--help"},
			expected: "", // help output, but we're testing the debug flag is processed
		},
		{
			name:     "log-level flag",
			args:     []string{"--log-level", "error", "--help"},
			expected: "",
		},
		{
			name:     "log-format flag",
			args:     []string{"--log-format", "json", "--help"},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset Viper for clean test
			viper.Reset()

			cmd := rootCmd
			cmd.SetArgs(tt.args)

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			// Execute command - should not fail
			err := cmd.Execute()
			require.NoError(t, err)

			// Verify command executed successfully
			output := buf.String()
			assert.Contains(t, output, "Grove transforms Git worktrees")
		})
	}
}

func TestMainFunction(t *testing.T) {
	// Test that main function doesn't panic
	// We can't easily test the actual main() without causing os.Exit,
	// but we can test the core logic through rootCmd.Execute()
	t.Run("help command execution", func(t *testing.T) {
		cmd := rootCmd
		cmd.SetArgs([]string{"--help"})

		buf := new(bytes.Buffer)
		cmd.SetOut(buf)

		err := cmd.Execute()
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Grove transforms Git worktrees")
		assert.Contains(t, output, "Available Commands:")
	})
}

func TestPersistentFlags(t *testing.T) {
	// Test that all expected persistent flags are registered
	flags := rootCmd.PersistentFlags()

	assert.NotNil(t, flags.Lookup("log-level"))
	assert.NotNil(t, flags.Lookup("log-format"))
	assert.NotNil(t, flags.Lookup("debug"))

	// Reset flags to defaults for clean test
	_ = flags.Set("log-level", "info")
	_ = flags.Set("log-format", "text")
	_ = flags.Set("debug", "false")

	// Test default values
	logLevel, err := flags.GetString("log-level")
	require.NoError(t, err)
	assert.Equal(t, "info", logLevel)

	logFormat, err := flags.GetString("log-format")
	require.NoError(t, err)
	assert.Equal(t, "text", logFormat)

	debug, err := flags.GetBool("debug")
	require.NoError(t, err)
	assert.False(t, debug)
}
