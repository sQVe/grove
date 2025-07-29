//go:build integration
// +build integration

package commands

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigCommand_Integration_Get(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-get-integration")
	defer testDir.Cleanup()

	// Set up clean environment
	setupConfigTestEnvironment(t, testDir.Path)

	tests := []struct {
		name        string
		key         string
		expectError bool
		setup       func()
	}{
		{
			name: "get valid key from defaults",
			key:  "general.editor",
		},
		{
			name: "get git timeout",
			key:  "git.fetch_timeout",
		},
		{
			name: "get logging level",
			key:  "logging.level",
		},
		{
			name: "get retry settings",
			key:  "retry.max_attempts",
		},
		{
			name: "get worktree settings",
			key:  "worktree.naming_pattern",
		},
		{
			name:        "get invalid key",
			key:         "invalid.key",
			expectError: true,
		},
		{
			name:        "get malformed key",
			key:         "malformed",
			expectError: true,
		},
		{
			name: "get custom set value",
			key:  "general.editor",
			setup: func() {
				config.Set("general.editor", "nano")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config for each test
			viper.Reset()
			config.Initialize()

			if tt.setup != nil {
				tt.setup()
			}

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"get", tt.key})

			output := captureOutput(t, func() error {
				return cmd.Execute()
			})

			if tt.expectError {
				assert.Contains(t, output.stderr, "invalid configuration key")
			} else {
				assert.NoError(t, output.err)
				assert.NotEmpty(t, output.stdout)
				assert.Empty(t, output.stderr)
			}
		})
	}
}

func TestConfigCommand_Integration_GetWithDefault(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-get-default-integration")
	defer testDir.Cleanup()

	setupConfigTestEnvironment(t, testDir.Path)

	// Set a custom value
	viper.Reset()
	config.Initialize()
	config.Set("general.editor", "custom-editor")

	cmd := NewConfigCmd()
	cmd.SetArgs([]string{"get", "--default", "general.editor"})

	output := captureOutput(t, func() error {
		return cmd.Execute()
	})

	require.NoError(t, output.err)
	// Should show default value, not the custom one
	assert.Contains(t, output.stdout, "vim") // default editor
	assert.NotContains(t, output.stdout, "custom-editor")
}

func TestConfigCommand_Integration_Set(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-set-integration")
	defer testDir.Cleanup()

	setupConfigTestEnvironment(t, testDir.Path)

	tests := []struct {
		name        string
		key         string
		value       string
		expectError bool
		validate    func(t *testing.T)
	}{
		{
			name:  "set string value",
			key:   "general.editor",
			value: "emacs",
			validate: func(t *testing.T) {
				assert.Equal(t, "emacs", config.GetString("general.editor"))
			},
		},
		{
			name:  "set duration value",
			key:   "git.fetch_timeout",
			value: "120s",
			validate: func(t *testing.T) {
				assert.Equal(t, "2m0s", config.GetDuration("git.fetch_timeout").String())
			},
		},
		{
			name:  "set integer value",
			key:   "retry.max_attempts",
			value: "5",
			validate: func(t *testing.T) {
				assert.Equal(t, 5, config.GetInt("retry.max_attempts"))
			},
		},
		{
			name:  "set boolean value",
			key:   "retry.jitter_enabled",
			value: "false",
			validate: func(t *testing.T) {
				assert.False(t, config.GetBool("retry.jitter_enabled"))
			},
		},
		{
			name:        "set invalid key",
			key:         "invalid.key",
			value:       "value",
			expectError: true,
		},
		{
			name:        "set invalid duration",
			key:         "git.fetch_timeout",
			value:       "invalid-duration",
			expectError: true,
		},
		{
			name:        "set invalid integer",
			key:         "retry.max_attempts",
			value:       "not-a-number",
			expectError: true,
		},
		{
			name:        "set invalid boolean",
			key:         "retry.jitter_enabled",
			value:       "maybe",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config for each test
			viper.Reset()
			config.Initialize()

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"set", tt.key, tt.value})

			output := captureOutput(t, func() error {
				return cmd.Execute()
			})

			if tt.expectError {
				assert.Error(t, output.err)
			} else {
				assert.NoError(t, output.err)
				assert.Contains(t, output.stdout, "Set "+tt.key)

				if tt.validate != nil {
					tt.validate(t)
				}
			}
		})
	}
}

func TestConfigCommand_Integration_List(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-list-integration")
	defer testDir.Cleanup()

	setupConfigTestEnvironment(t, testDir.Path)

	tests := []struct {
		name   string
		args   []string
		format string
		setup  func()
	}{
		{
			name:   "list default format",
			args:   []string{"list"},
			format: "text",
		},
		{
			name:   "list text format",
			args:   []string{"list", "--format=text"},
			format: "text",
		},
		{
			name:   "list json format",
			args:   []string{"list", "--format=json"},
			format: "json",
		},
		{
			name:   "list with defaults flag",
			args:   []string{"list", "--defaults"},
			format: "text",
		},
		{
			name:   "list json with defaults",
			args:   []string{"list", "--format=json", "--defaults"},
			format: "json",
		},
		{
			name:   "list with custom values",
			args:   []string{"list"},
			format: "text",
			setup: func() {
				config.Set("general.editor", "nano")
				config.Set("git.fetch_timeout", "45s")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset config for each test
			viper.Reset()
			config.Initialize()

			if tt.setup != nil {
				tt.setup()
			}

			cmd := NewConfigCmd()
			cmd.SetArgs(tt.args)

			output := captureOutput(t, func() error {
				return cmd.Execute()
			})

			require.NoError(t, output.err)
			assert.NotEmpty(t, output.stdout)

			if tt.format == "json" {
				// Validate JSON structure
				var jsonData map[string]interface{}
				err := json.Unmarshal([]byte(output.stdout), &jsonData)
				require.NoError(t, err)

				// Check expected sections
				assert.Contains(t, jsonData, "general")
				assert.Contains(t, jsonData, "git")
				assert.Contains(t, jsonData, "logging")
				assert.Contains(t, jsonData, "retry")
				assert.Contains(t, jsonData, "worktree")
			} else {
				// Text format should contain section headers
				assert.Contains(t, output.stdout, "[general]")
				assert.Contains(t, output.stdout, "[git]")
				assert.Contains(t, output.stdout, "[logging]")
				assert.Contains(t, output.stdout, "[retry]")
				assert.Contains(t, output.stdout, "[worktree]")
			}
		})
	}
}

func TestConfigCommand_Integration_ListInvalidFormat(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-list-invalid")
	defer testDir.Cleanup()

	setupConfigTestEnvironment(t, testDir.Path)

	viper.Reset()
	config.Initialize()

	cmd := NewConfigCmd()
	cmd.SetArgs([]string{"list", "--format=invalid"})

	output := captureOutput(t, func() error {
		return cmd.Execute()
	})

	assert.Error(t, output.err)
	assert.Contains(t, output.stderr, "unsupported format")
}

func TestConfigCommand_Integration_Validate(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-validate-integration")
	defer testDir.Cleanup()

	setupConfigTestEnvironment(t, testDir.Path)

	t.Run("validate default config", func(t *testing.T) {
		viper.Reset()
		config.Initialize()

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"validate"})

		output := captureOutput(t, func() error {
			return cmd.Execute()
		})

		assert.NoError(t, output.err)
		assert.Contains(t, output.stdout, "Configuration is valid")
	})

	t.Run("validate with custom valid config", func(t *testing.T) {
		viper.Reset()
		config.Initialize()

		config.Set("general.editor", "code")
		config.Set("git.fetch_timeout", "60s")

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"validate"})

		output := captureOutput(t, func() error {
			return cmd.Execute()
		})

		assert.NoError(t, output.err)
		assert.Contains(t, output.stdout, "Configuration is valid")
	})
}

func TestConfigCommand_Integration_Reset(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-reset-integration")
	defer testDir.Cleanup()

	setupConfigTestEnvironment(t, testDir.Path)

	t.Run("reset single key", func(t *testing.T) {
		viper.Reset()
		config.Initialize()

		// Set a custom value
		config.Set("general.editor", "custom-editor")
		assert.Equal(t, "custom-editor", config.GetString("general.editor"))

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"reset", "general.editor"})

		output := captureOutput(t, func() error {
			return cmd.Execute()
		})

		assert.NoError(t, output.err)
		assert.Contains(t, output.stdout, "Reset general.editor to default value")

		// Should be back to default
		defaultConfig := config.DefaultConfig()
		assert.Equal(t, defaultConfig.General.Editor, config.GetString("general.editor"))
	})

	t.Run("reset invalid key", func(t *testing.T) {
		viper.Reset()
		config.Initialize()

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"reset", "invalid.key"})

		output := captureOutput(t, func() error {
			return cmd.Execute()
		})

		assert.Error(t, output.err)
		assert.Contains(t, output.stderr, "invalid configuration key")
	})

	t.Run("reset all with confirm", func(t *testing.T) {
		viper.Reset()
		config.Initialize()

		// Set custom values
		config.Set("general.editor", "custom1")
		config.Set("git.fetch_timeout", "123s")

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"reset", "--confirm"})

		output := captureOutput(t, func() error {
			return cmd.Execute()
		})

		assert.NoError(t, output.err)
		assert.Contains(t, output.stdout, "All configuration reset to defaults")

		// Should be back to defaults
		defaultConfig := config.DefaultConfig()
		assert.Equal(t, defaultConfig.General.Editor, config.GetString("general.editor"))
		assert.Equal(t, defaultConfig.Git.FetchTimeout, config.GetDuration("git.fetch_timeout"))
	})
}

func TestConfigCommand_Integration_Path(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-path-integration")
	defer testDir.Cleanup()

	setupConfigTestEnvironment(t, testDir.Path)

	viper.Reset()
	config.Initialize()

	cmd := NewConfigCmd()
	cmd.SetArgs([]string{"path"})

	output := captureOutput(t, func() error {
		return cmd.Execute()
	})

	assert.NoError(t, output.err)
	assert.Contains(t, output.stdout, "Configuration file search paths:")
	assert.Contains(t, output.stdout, "1.")
	assert.Contains(t, output.stdout, "2.")

	// Should mention if no config file is found or show the current one
	if strings.Contains(output.stdout, "Currently used config file:") {
		assert.Contains(t, output.stdout, ".yaml")
	} else {
		assert.Contains(t, output.stdout, "No config file found, using defaults")
	}
}

func TestConfigCommand_Integration_Init(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-config-init-integration")
	defer testDir.Cleanup()

	setupConfigTestEnvironment(t, testDir.Path)

	t.Run("init new config file", func(t *testing.T) {
		viper.Reset()
		config.Initialize()

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"init"})

		output := captureOutput(t, func() error {
			return cmd.Execute()
		})

		assert.NoError(t, output.err)
		assert.Contains(t, output.stdout, "Created configuration file:")
		assert.Contains(t, output.stdout, ".yaml")
	})

	t.Run("init with existing file", func(t *testing.T) {
		viper.Reset()
		config.Initialize()

		// Create a config file first
		config.Set("general.editor", "test")
		config.WriteConfig()

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"init"})

		output := captureOutput(t, func() error {
			return cmd.Execute()
		})

		assert.Error(t, output.err)
		assert.Contains(t, output.stderr, "configuration file already exists")
	})

	t.Run("init with force flag", func(t *testing.T) {
		viper.Reset()
		config.Initialize()

		// Create a config file first
		config.Set("general.editor", "test")
		config.WriteConfig()

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"init", "--force"})

		output := captureOutput(t, func() error {
			return cmd.Execute()
		})

		assert.NoError(t, output.err)
		assert.Contains(t, output.stdout, "Created configuration file:")
	})
}

// Helper functions

func setupConfigTestEnvironment(t *testing.T, configDir string) {
	t.Helper()

	// Set XDG_CONFIG_HOME to test directory
	oldConfigHome := os.Getenv("XDG_CONFIG_HOME")
	t.Cleanup(func() {
		if oldConfigHome != "" {
			os.Setenv("XDG_CONFIG_HOME", oldConfigHome)
		} else {
			os.Unsetenv("XDG_CONFIG_HOME")
		}
	})

	os.Setenv("XDG_CONFIG_HOME", configDir)
}

type commandOutput struct {
	stdout string
	stderr string
	err    error
}

func captureOutput(t *testing.T, fn func() error) commandOutput {
	t.Helper()

	// Create a new command instance to capture its output
	oldStdout := os.Stdout
	oldStderr := os.Stderr

	stdoutR, stdoutW, _ := os.Pipe()
	stderrR, stderrW, _ := os.Pipe()

	os.Stdout = stdoutW
	os.Stderr = stderrW

	// Create channels to capture output
	stdoutCh := make(chan string, 1)
	stderrCh := make(chan string, 1)

	go func() {
		buf := make([]byte, 1024)
		var output strings.Builder
		for {
			n, err := stdoutR.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		stdoutCh <- output.String()
	}()

	go func() {
		buf := make([]byte, 1024)
		var output strings.Builder
		for {
			n, err := stderrR.Read(buf)
			if n > 0 {
				output.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
		stderrCh <- output.String()
	}()

	// Execute the function
	err := fn()

	// Restore original stdout/stderr
	os.Stdout = oldStdout
	os.Stderr = oldStderr
	stdoutW.Close()
	stderrW.Close()

	// Get captured output
	stdout := <-stdoutCh
	stderr := <-stderrCh

	return commandOutput{
		stdout: stdout,
		stderr: stderr,
		err:    err,
	}
}
