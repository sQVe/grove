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
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-get-integration")

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
			runner := testutils.NewTestRunner(t)
			runner.WithCleanEnvironment().Run(func() {
				_ = os.Setenv("XDG_CONFIG_HOME", configDir)

				viper.Reset()
				config.Initialize()

				if tt.setup != nil {
					tt.setup()
				}

				cmd := NewConfigCmd()
				cmd.SetArgs([]string{"get", tt.key})

				var stdout, stderr strings.Builder
				cmd.SetOut(&stdout)
				cmd.SetErr(&stderr)

				err := cmd.Execute()

				if tt.expectError {
					assert.Contains(t, stderr.String(), "invalid configuration key")
				} else {
					assert.NoError(t, err)
					assert.NotEmpty(t, stdout.String())
					assert.Empty(t, stderr.String())
				}
			})
		})
	}
}

func TestConfigCommand_Integration_GetWithDefault(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-get-default-integration")

	runner := testutils.NewTestRunner(t)
	runner.WithCleanEnvironment().Run(func() {
		_ = os.Setenv("XDG_CONFIG_HOME", configDir)

		viper.Reset()
		config.Initialize()
		config.Set("general.editor", "custom-editor")

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"get", "--default", "general.editor"})

		var stdout, stderr strings.Builder
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		err := cmd.Execute()

		require.NoError(t, err)
		// Should show default value, not the custom one
		assert.Contains(t, stdout.String(), "vi") // default editor
		assert.NotContains(t, stdout.String(), "custom-editor")
	})
}

func TestConfigCommand_Integration_Set(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-set-integration")

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
			runner := testutils.NewTestRunner(t)
			runner.WithCleanEnvironment().Run(func() {
				_ = os.Setenv("XDG_CONFIG_HOME", configDir)

				viper.Reset()
				config.Initialize()

				cmd := NewConfigCmd()
				cmd.SetArgs([]string{"set", tt.key, tt.value})

				var stdout, stderr strings.Builder
				cmd.SetOut(&stdout)
				cmd.SetErr(&stderr)

				err := cmd.Execute()

				if tt.expectError {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
					assert.Contains(t, stdout.String(), "Set "+tt.key)

					if tt.validate != nil {
						tt.validate(t)
					}
				}
			})
		})
	}
}

func TestConfigCommand_Integration_List(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-list-integration")

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
			runner := testutils.NewTestRunner(t)
			runner.WithCleanEnvironment().Run(func() {
				_ = os.Setenv("XDG_CONFIG_HOME", configDir)

				viper.Reset()
				config.Initialize()

				if tt.setup != nil {
					tt.setup()
				}

				cmd := NewConfigCmd()
				cmd.SetArgs(tt.args)

				var stdout, stderr strings.Builder
				cmd.SetOut(&stdout)
				cmd.SetErr(&stderr)

				err := cmd.Execute()

				require.NoError(t, err)
				assert.NotEmpty(t, stdout.String())

				if tt.format == "json" {
					// Validate JSON structure
					var jsonData map[string]interface{}
					err := json.Unmarshal([]byte(stdout.String()), &jsonData)
					require.NoError(t, err)

					// Check expected sections
					assert.Contains(t, jsonData, "general")
					assert.Contains(t, jsonData, "git")
					assert.Contains(t, jsonData, "logging")
					assert.Contains(t, jsonData, "retry")
					assert.Contains(t, jsonData, "worktree")
				} else {
					// Text format should contain section headers
					assert.Contains(t, stdout.String(), "[general]")
					assert.Contains(t, stdout.String(), "[git]")
					assert.Contains(t, stdout.String(), "[logging]")
					assert.Contains(t, stdout.String(), "[retry]")
					assert.Contains(t, stdout.String(), "[worktree]")
				}
			})
		})
	}
}

func TestConfigCommand_Integration_ListInvalidFormat(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-list-invalid")

	runner := testutils.NewTestRunner(t)
	runner.WithCleanEnvironment().Run(func() {
		_ = os.Setenv("XDG_CONFIG_HOME", configDir)

		viper.Reset()
		config.Initialize()

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"list", "--format=invalid"})

		var stdout, stderr strings.Builder
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		err := cmd.Execute()

		assert.Error(t, err)
		assert.Contains(t, stderr.String(), "unsupported format")
	})
}

func TestConfigCommand_Integration_Validate(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-validate-integration")

	t.Run("validate default config", func(t *testing.T) {
		runner := testutils.NewTestRunner(t)
		runner.WithCleanEnvironment().Run(func() {
			_ = os.Setenv("XDG_CONFIG_HOME", configDir)

			viper.Reset()
			config.Initialize()

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"validate"})

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			assert.NoError(t, err)
			assert.Contains(t, stdout.String(), "Configuration is valid")
		})
	})

	t.Run("validate with custom valid config", func(t *testing.T) {
		runner := testutils.NewTestRunner(t)
		runner.WithCleanEnvironment().Run(func() {
			_ = os.Setenv("XDG_CONFIG_HOME", configDir)

			viper.Reset()
			config.Initialize()

			config.Set("general.editor", "code")
			config.Set("git.fetch_timeout", "60s")

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"validate"})

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			assert.NoError(t, err)
			assert.Contains(t, stdout.String(), "Configuration is valid")
		})
	})
}

func TestConfigCommand_Integration_Reset(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-reset-integration")

	t.Run("reset single key", func(t *testing.T) {
		runner := testutils.NewTestRunner(t)
		runner.WithCleanEnvironment().Run(func() {
			_ = os.Setenv("XDG_CONFIG_HOME", configDir)

			viper.Reset()
			config.Initialize()

			// Set a custom value
			config.Set("general.editor", "custom-editor")
			assert.Equal(t, "custom-editor", config.GetString("general.editor"))

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"reset", "general.editor"})

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			assert.NoError(t, err)
			assert.Contains(t, stdout.String(), "Reset general.editor to default value")

			// Should be back to default
			defaultConfig := config.DefaultConfig()
			assert.Equal(t, defaultConfig.General.Editor, config.GetString("general.editor"))
		})
	})

	t.Run("reset invalid key", func(t *testing.T) {
		runner := testutils.NewTestRunner(t)
		runner.WithCleanEnvironment().Run(func() {
			_ = os.Setenv("XDG_CONFIG_HOME", configDir)

			viper.Reset()
			config.Initialize()

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"reset", "invalid.key"})

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			assert.Error(t, err)
			assert.Contains(t, stderr.String(), "invalid configuration key")
		})
	})

	t.Run("reset all with confirm", func(t *testing.T) {
		runner := testutils.NewTestRunner(t)
		runner.WithCleanEnvironment().Run(func() {
			_ = os.Setenv("XDG_CONFIG_HOME", configDir)

			viper.Reset()
			config.Initialize()

			// Set custom values
			config.Set("general.editor", "custom1")
			config.Set("git.fetch_timeout", "123s")

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"reset", "--confirm"})

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			assert.NoError(t, err)
			assert.Contains(t, stdout.String(), "All configuration reset to defaults")

			// Should be back to defaults
			defaultConfig := config.DefaultConfig()
			assert.Equal(t, defaultConfig.General.Editor, config.GetString("general.editor"))
			assert.Equal(t, defaultConfig.Git.FetchTimeout, config.GetDuration("git.fetch_timeout"))
		})
	})
}

func TestConfigCommand_Integration_Path(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-path-integration")

	runner := testutils.NewTestRunner(t)
	runner.WithCleanEnvironment().Run(func() {
		_ = os.Setenv("XDG_CONFIG_HOME", configDir)

		viper.Reset()
		config.Initialize()

		cmd := NewConfigCmd()
		cmd.SetArgs([]string{"path"})

		var stdout, stderr strings.Builder
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)

		err := cmd.Execute()

		assert.NoError(t, err)
		assert.Contains(t, stdout.String(), "Configuration file search paths:")
		assert.Contains(t, stdout.String(), "1.")
		assert.Contains(t, stdout.String(), "2.")

		// Should mention if no config file is found or show the current one
		if strings.Contains(stdout.String(), "Currently used config file:") {
			assert.Contains(t, stdout.String(), ".yaml")
		} else {
			assert.Contains(t, stdout.String(), "No config file found, using defaults")
		}
	})
}

func TestConfigCommand_Integration_Init(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	configDir := helper.CreateTempDir("grove-config-init-integration")

	t.Run("init new config file", func(t *testing.T) {
		runner := testutils.NewTestRunner(t)
		runner.WithCleanEnvironment().Run(func() {
			_ = os.Setenv("XDG_CONFIG_HOME", configDir)

			viper.Reset()
			config.Initialize()

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"init"})

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			assert.NoError(t, err)
			assert.Contains(t, stdout.String(), "Created configuration file:")
			assert.Contains(t, stdout.String(), ".yaml")
		})
	})

	t.Run("init with existing file", func(t *testing.T) {
		runner := testutils.NewTestRunner(t)
		runner.WithCleanEnvironment().Run(func() {
			_ = os.Setenv("XDG_CONFIG_HOME", configDir)

			viper.Reset()
			config.Initialize()

			// Create a config file first
			config.Set("general.editor", "test")
			config.WriteConfig()

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"init"})

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			assert.Error(t, err)
			assert.Contains(t, stderr.String(), "configuration file already exists")
		})
	})

	t.Run("init with force flag", func(t *testing.T) {
		runner := testutils.NewTestRunner(t)
		runner.WithCleanEnvironment().Run(func() {
			_ = os.Setenv("XDG_CONFIG_HOME", configDir)

			viper.Reset()
			config.Initialize()

			// Create a config file first
			config.Set("general.editor", "test")
			config.WriteConfig()

			cmd := NewConfigCmd()
			cmd.SetArgs([]string{"init", "--force"})

			var stdout, stderr strings.Builder
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			assert.NoError(t, err)
			assert.Contains(t, stdout.String(), "Created configuration file:")
		})
	})
}
