package commands

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigCmd(t *testing.T) {
	cmd := NewConfigCmd()
	assert.Equal(t, "config", cmd.Use)
	assert.Equal(t, "Manage Grove configuration", cmd.Short)
	assert.Len(t, cmd.Commands(), 7) // get, set, list, validate, reset, path, init
}

func TestConfigGetCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		setupConfig func(t *testing.T)
		expectError bool
		expectOut   string
	}{
		{
			name: "get valid key",
			args: []string{"general.editor"},
			setupConfig: func(t *testing.T) {
				config.Set("general.editor", "vim")
			},
			expectOut: "vim",
		},
		{
			name: "get with default flag",
			args: []string{"general.editor", "--default"},
			setupConfig: func(t *testing.T) {
				config.Set("general.editor", "emacs")
			},
			expectOut: "nvim", // default value
		},
		{
			name:        "get invalid key",
			args:        []string{"invalid.key"},
			expectError: true,
		},
		{
			name:        "get no args",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "get too many args",
			args:        []string{"key1", "key2"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper
			viper.Reset()
			config.SetDefaults()

			if tt.setupConfig != nil {
				tt.setupConfig(t)
			}

			cmd := newConfigGetCmd()
			output, err := executeCommand(cmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectOut != "" {
					assert.Contains(t, output, tt.expectOut)
				}
			}
		})
	}
}

func TestConfigSetCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "set invalid key",
			args:        []string{"invalid.key", "value"},
			expectError: true,
		},
		{
			name:        "set invalid int value",
			args:        []string{"git.max_retries", "invalid"},
			expectError: true,
		},
		{
			name:        "set invalid bool value",
			args:        []string{"retry.jitter_enabled", "invalid"},
			expectError: true,
		},
		{
			name:        "set invalid duration value",
			args:        []string{"git.fetch_timeout", "invalid"},
			expectError: true,
		},
		{
			name:        "set no args",
			args:        []string{},
			expectError: true,
		},
		{
			name:        "set one arg",
			args:        []string{"key"},
			expectError: true,
		},
		{
			name:        "set too many args",
			args:        []string{"key", "value", "extra"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newConfigSetCmd()
			_, err := executeCommand(cmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigListCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name: "list text format",
			args: []string{},
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "[general]")
				assert.Contains(t, output, "[git]")
				assert.Contains(t, output, "[logging]")
				assert.Contains(t, output, "[retry]")
				assert.Contains(t, output, "[worktree]")
			},
		},
		{
			name: "list json format",
			args: []string{"--format=json"},
			checkOutput: func(t *testing.T, output string) {
				var data map[string]interface{}
				err := json.Unmarshal([]byte(output), &data)
				assert.NoError(t, err)
				assert.Contains(t, data, "general")
				assert.Contains(t, data, "git")
				assert.Contains(t, data, "logging")
				assert.Contains(t, data, "retry")
				assert.Contains(t, data, "worktree")
			},
		},
		{
			name: "list defaults",
			args: []string{"--defaults"},
			checkOutput: func(t *testing.T, output string) {
				assert.Contains(t, output, "[general]")
				assert.Contains(t, output, "editor = nvim")
			},
		},
		{
			name:        "list invalid format",
			args:        []string{"--format=invalid"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper
			viper.Reset()
			config.SetDefaults()

			cmd := newConfigListCmd()
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.checkOutput != nil {
					tt.checkOutput(t, buf.String())
				}
			}
		})
	}
}

func TestConfigValidateCmd(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func()
		expectError bool
		expectOut   string
	}{
		{
			name: "valid config",
			setupConfig: func() {
				config.SetDefaults()
			},
			expectOut: "Configuration is valid",
		},
		{
			name: "invalid config",
			setupConfig: func() {
				config.SetDefaults()
				config.Set("logging.level", "invalid")
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset viper
			viper.Reset()
			if tt.setupConfig != nil {
				tt.setupConfig()
			}

			cmd := newConfigValidateCmd()
			output, err := executeCommand(cmd, []string{})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectOut != "" {
					assert.Contains(t, output, tt.expectOut)
				}
			}
		})
	}
}

func TestConfigResetCmd(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
	}{
		{
			name:        "reset invalid key",
			args:        []string{"invalid.key"},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newConfigResetCmd()
			_, err := executeCommand(cmd, tt.args)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigPathCmd(t *testing.T) {
	cmd := newConfigPathCmd()
	output, err := executeCommand(cmd, []string{})
	assert.NoError(t, err)
	assert.Contains(t, output, "Configuration file search paths:")
}

func TestParseConfigValue(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		valueStr    string
		expectValue interface{}
		expectError bool
	}{
		{
			name:        "parse string",
			key:         "general.editor",
			valueStr:    "vim",
			expectValue: "vim",
		},
		{
			name:        "parse int",
			key:         "git.max_retries",
			valueStr:    "5",
			expectValue: 5,
		},
		{
			name:        "parse bool true",
			key:         "retry.jitter_enabled",
			valueStr:    "true",
			expectValue: true,
		},
		{
			name:        "parse bool false",
			key:         "retry.jitter_enabled",
			valueStr:    "false",
			expectValue: false,
		},
		{
			name:        "parse duration",
			key:         "git.fetch_timeout",
			valueStr:    "30s",
			expectValue: 30 * time.Second,
		},
		{
			name:        "parse invalid int",
			key:         "git.max_retries",
			valueStr:    "invalid",
			expectError: true,
		},
		{
			name:        "parse invalid bool",
			key:         "retry.jitter_enabled",
			valueStr:    "invalid",
			expectError: true,
		},
		{
			name:        "parse invalid duration",
			key:         "git.fetch_timeout",
			valueStr:    "invalid",
			expectError: true,
		},
		{
			name:        "parse unknown key",
			key:         "unknown.key",
			valueStr:    "value",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := parseConfigValue(tt.key, tt.valueStr)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectValue, value)
			}
		})
	}
}

func TestGetConfigValueByKey(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name        string
		key         string
		expectValue interface{}
	}{
		{
			name:        "get general.editor",
			key:         "general.editor",
			expectValue: cfg.General.Editor,
		},
		{
			name:        "get git.max_retries",
			key:         "git.max_retries",
			expectValue: cfg.Git.MaxRetries,
		},
		{
			name:        "get retry.jitter_enabled",
			key:         "retry.jitter_enabled",
			expectValue: cfg.Retry.Jitter,
		},
		{
			name:        "get invalid key",
			key:         "invalid.key",
			expectValue: nil,
		},
		{
			name:        "get malformed key",
			key:         "invalid",
			expectValue: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value := getConfigValueByKey(cfg, tt.key)
			assert.Equal(t, tt.expectValue, value)
		})
	}
}

func TestStructToMap(t *testing.T) {
	cfg := config.DefaultConfig()
	result := structToMap(cfg)

	assert.Contains(t, result, "general")
	assert.Contains(t, result, "git")
	assert.Contains(t, result, "logging")
	assert.Contains(t, result, "retry")
	assert.Contains(t, result, "worktree")

	general, ok := result["general"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, general, "editor")
	assert.Contains(t, general, "pager")
	assert.Contains(t, general, "output_format")
}

func TestPrintConfigText(t *testing.T) {
	data := map[string]interface{}{
		"general": map[string]interface{}{
			"editor": "vim",
			"pager":  "less",
		},
		"git": map[string]interface{}{
			"max_retries": 3,
		},
	}

	cmd := &cobra.Command{}
	var buf bytes.Buffer
	cmd.SetOut(&buf)

	printConfigText(cmd, data)

	output := buf.String()

	assert.Contains(t, output, "[general]")
	assert.Contains(t, output, "[git]")
	assert.Contains(t, output, "editor = vim")
	assert.Contains(t, output, "max_retries = 3")
}

func TestGetConfigValueByKeySafety(t *testing.T) {
	cfg := config.DefaultConfig()

	tests := []struct {
		name        string
		config      *config.Config
		key         string
		expectValue interface{}
		description string
	}{
		{
			name:        "nil config pointer",
			config:      nil,
			key:         "general.editor",
			expectValue: nil,
			description: "Should handle nil config gracefully",
		},
		{
			name:        "empty key",
			config:      cfg,
			key:         "",
			expectValue: nil,
			description: "Should handle empty key gracefully",
		},
		{
			name:        "malformed key - no dot",
			config:      cfg,
			key:         "generaledit",
			expectValue: nil,
			description: "Should handle malformed key without dot",
		},
		{
			name:        "malformed key - too many dots",
			config:      cfg,
			key:         "general.editor.extra",
			expectValue: nil,
			description: "Should handle key with too many parts",
		},
		{
			name:        "empty section",
			config:      cfg,
			key:         ".editor",
			expectValue: nil,
			description: "Should handle empty section name",
		},
		{
			name:        "empty field",
			config:      cfg,
			key:         "general.",
			expectValue: nil,
			description: "Should handle empty field name",
		},
		{
			name:        "non-existent section",
			config:      cfg,
			key:         "nonexistent.field",
			expectValue: nil,
			description: "Should handle non-existent section",
		},
		{
			name:        "non-existent field",
			config:      cfg,
			key:         "general.nonexistent",
			expectValue: nil,
			description: "Should handle non-existent field",
		},
		{
			name:        "valid key",
			config:      cfg,
			key:         "general.editor",
			expectValue: cfg.General.Editor,
			description: "Should return correct value for valid key",
		},
		{
			name:        "valid duration field",
			config:      cfg,
			key:         "git.fetch_timeout",
			expectValue: cfg.Git.FetchTimeout,
			description: "Should handle duration types correctly",
		},
		{
			name:        "valid bool field",
			config:      cfg,
			key:         "retry.jitter_enabled",
			expectValue: cfg.Retry.Jitter,
			description: "Should handle bool types correctly",
		},
		{
			name:        "valid int field",
			config:      cfg,
			key:         "git.max_retries",
			expectValue: cfg.Git.MaxRetries,
			description: "Should handle int types correctly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This should not panic regardless of input
			result := getConfigValueByKey(tt.config, tt.key)
			assert.Equal(t, tt.expectValue, result, tt.description)
		})
	}
}

func TestGetConfigValueByKeyPanicRecovery(t *testing.T) {
	// Test that the function doesn't crash on edge cases that might cause panics

	// Create a config and then test various edge cases
	cfg := config.DefaultConfig()

	// These calls should not crash the application
	tests := []struct {
		name   string
		config *config.Config
		key    string
	}{
		{"nil config", nil, "general.editor"},
		{"empty key", cfg, ""},
		{"whitespace key", cfg, "   "},
		{"special characters", cfg, "general..editor"},
		{"unicode key", cfg, "généràl.edïtör"},
		{"very long key", cfg, strings.Repeat("a", 1000) + "." + strings.Repeat("b", 1000)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test passes if it doesn't panic
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("getConfigValueByKey panicked: %v", r)
				}
			}()

			result := getConfigValueByKey(tt.config, tt.key)
			// We expect nil for all these edge cases
			assert.Nil(t, result)
		})
	}
}

// Helper function to execute command and capture output.
func executeCommand(cmd *cobra.Command, args []string) (string, error) {
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}
