package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitialize(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.toml")
	configContent := `
[general]
editor = "vim"
pager = "less"
output_format = "json"

[git]
default_remote = "upstream"
fetch_timeout = "60s"
max_retries = 5

[retry]
max_attempts = 5
base_delay = "2s"
max_delay = "30s"
jitter_enabled = false

[logging]
level = "debug"
format = "json"

[worktree]
naming_pattern = "slug"
cleanup_threshold = "7d"
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	require.NoError(t, err)

	originalViper := viper.GetViper()
	defer func() {
		viper.Reset()
		// Restore original viper instance
		for key, value := range originalViper.AllSettings() {
			viper.Set(key, value)
		}
	}()

	// Reset viper for clean test
	viper.Reset()
	viper.AddConfigPath(tmpDir)

	// Test initialization
	err = Initialize()
	require.NoError(t, err)

	// Verify configuration was loaded
	assert.Equal(t, "vim", GetString("general.editor"))
	assert.Equal(t, "json", GetString("general.output_format"))
	assert.Equal(t, "upstream", GetString("git.default_remote"))
	assert.Equal(t, 60*time.Second, GetDuration("git.fetch_timeout"))
	assert.Equal(t, 5, GetInt("retry.max_attempts"))
	assert.Equal(t, false, GetBool("retry.jitter_enabled"))
}

func TestInitializeWithoutConfigFile(t *testing.T) {
	// Create a temporary directory without config file
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Reset viper for clean test
	originalViper := viper.GetViper()
	defer func() {
		viper.Reset()
		// Restore original viper instance
		for key, value := range originalViper.AllSettings() {
			viper.Set(key, value)
		}
	}()

	viper.Reset()
	viper.AddConfigPath(tmpDir)

	// Test initialization without config file
	err = Initialize()
	require.NoError(t, err)

	// Verify defaults are used
	assert.Equal(t, "info", GetString("logging.level"))
	assert.Equal(t, "text", GetString("logging.format"))
	assert.Equal(t, "origin", GetString("git.default_remote"))
	assert.Equal(t, 3, GetInt("retry.max_attempts"))
}

func TestGet(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()
	SetDefaults()

	config, err := Get()
	require.NoError(t, err)

	// Verify default values
	assert.Equal(t, "text", config.General.OutputFormat)
	assert.Equal(t, "origin", config.Git.DefaultRemote)
	assert.Equal(t, 30*time.Second, config.Git.FetchTimeout)
	assert.Equal(t, 3, config.Retry.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.Retry.BaseDelay)
	assert.Equal(t, 10*time.Second, config.Retry.MaxDelay)
	assert.Equal(t, true, config.Retry.Jitter)
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
	assert.Equal(t, "branch", config.Worktree.NamingPattern)
	assert.Equal(t, 30*24*time.Hour, config.Worktree.CleanupThreshold)
}

func TestEnvironmentVariables(t *testing.T) {
	// Reset viper for clean test
	viper.Reset()

	// Set environment variables
	require.NoError(t, os.Setenv("GROVE_LOGGING_LEVEL", "debug"))
	require.NoError(t, os.Setenv("GROVE_GIT_DEFAULT_REMOTE", "upstream"))
	require.NoError(t, os.Setenv("GROVE_RETRY_MAX_ATTEMPTS", "5"))
	defer func() {
		_ = os.Unsetenv("GROVE_LOGGING_LEVEL")
		_ = os.Unsetenv("GROVE_GIT_DEFAULT_REMOTE")
		_ = os.Unsetenv("GROVE_RETRY_MAX_ATTEMPTS")
	}()

	// Initialize configuration
	err := Initialize()
	require.NoError(t, err)

	// Verify environment variables are used
	assert.Equal(t, "debug", GetString("logging.level"))
	assert.Equal(t, "upstream", GetString("git.default_remote"))
	assert.Equal(t, 5, GetInt("retry.max_attempts"))
}

func TestSetAndIsSet(t *testing.T) {
	viper.Reset()
	SetDefaults()

	// Test setting a value
	Set("test.key", "test_value")
	assert.Equal(t, "test_value", GetString("test.key"))
	assert.True(t, IsSet("test.key"))

	// Test checking an unset value
	assert.False(t, IsSet("unset.key"))
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Verify all default values are set correctly
	assert.Equal(t, "text", config.General.OutputFormat)
	assert.Equal(t, "origin", config.Git.DefaultRemote)
	assert.Equal(t, 30*time.Second, config.Git.FetchTimeout)
	assert.Equal(t, 3, config.Git.MaxRetries)
	assert.Equal(t, 3, config.Retry.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.Retry.BaseDelay)
	assert.Equal(t, 10*time.Second, config.Retry.MaxDelay)
	assert.Equal(t, true, config.Retry.Jitter)
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
	assert.Equal(t, "branch", config.Worktree.NamingPattern)
	assert.Equal(t, 30*24*time.Hour, config.Worktree.CleanupThreshold)
}

func TestValidConstants(t *testing.T) {
	// Test ValidLogLevels
	logLevels := ValidLogLevels()
	assert.Contains(t, logLevels, "debug")
	assert.Contains(t, logLevels, "info")
	assert.Contains(t, logLevels, "warn")
	assert.Contains(t, logLevels, "error")

	// Test ValidOutputFormats
	outputFormats := ValidOutputFormats()
	assert.Contains(t, outputFormats, "text")
	assert.Contains(t, outputFormats, "json")

	// Test ValidLogFormats
	logFormats := ValidLogFormats()
	assert.Contains(t, logFormats, "text")
	assert.Contains(t, logFormats, "json")

	// Test ValidNamingPatterns
	namingPatterns := ValidNamingPatterns()
	assert.Contains(t, namingPatterns, "branch")
	assert.Contains(t, namingPatterns, "slug")
	assert.Contains(t, namingPatterns, "timestamp")
}

func TestGetDefaultEditor(t *testing.T) {
	// Save original environment
	originalEditor := os.Getenv("EDITOR")
	originalVisual := os.Getenv("VISUAL")
	defer func() {
		if originalEditor != "" {
			_ = os.Setenv("EDITOR", originalEditor)
		} else {
			_ = os.Unsetenv("EDITOR")
		}
		if originalVisual != "" {
			_ = os.Setenv("VISUAL", originalVisual)
		} else {
			_ = os.Unsetenv("VISUAL")
		}
	}()

	// Test with EDITOR environment variable
	_ = os.Unsetenv("VISUAL")
	require.NoError(t, os.Setenv("EDITOR", "nano"))

	editor := getDefaultEditor()
	assert.Equal(t, "nano", editor)

	// Test with VISUAL environment variable (should take precedence)
	require.NoError(t, os.Setenv("VISUAL", "emacs"))
	require.NoError(t, os.Setenv("EDITOR", "nano"))

	editor = getDefaultEditor()
	assert.Equal(t, "emacs", editor) // VISUAL has precedence over EDITOR

	// Test fallback to vi when neither is set
	_ = os.Unsetenv("EDITOR")
	_ = os.Unsetenv("VISUAL")

	editor = getDefaultEditor()
	assert.Equal(t, "vi", editor)
}

func TestConfigFileOperations(t *testing.T) {
	// Create a temporary directory
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Reset viper for clean test
	viper.Reset()
	viper.AddConfigPath(tmpDir)
	viper.SetConfigName("config")
	viper.SetConfigType("toml")

	// Set some test values
	Set("logging.level", "debug")
	Set("git.default_remote", "upstream")

	// Test WriteConfigAs
	configPath := filepath.Join(tmpDir, "test_config.toml")
	err = WriteConfigAs(configPath)
	require.NoError(t, err)

	// Verify file was created
	assert.FileExists(t, configPath)

	// Test reading the config back
	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	assert.Equal(t, "debug", viper.GetString("logging.level"))
	assert.Equal(t, "upstream", viper.GetString("git.default_remote"))
}

func TestAllSettings(t *testing.T) {
	viper.Reset()
	SetDefaults()

	// Set some test values
	Set("test.key1", "value1")
	Set("test.key2", "value2")

	settings := AllSettings()

	// Verify settings contains our test values
	assert.NotNil(t, settings["test"])
	testSettings := settings["test"].(map[string]interface{})
	assert.Equal(t, "value1", testSettings["key1"])
	assert.Equal(t, "value2", testSettings["key2"])

	// Verify default values are also present
	assert.NotNil(t, settings["logging"])
	assert.NotNil(t, settings["git"])
	assert.NotNil(t, settings["retry"])
}

func TestConfigFileUsed(t *testing.T) {
	// Create a temporary directory with config file
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.toml")
	err = os.WriteFile(configPath, []byte(`
[logging]
level = "debug"
`), 0o644)
	require.NoError(t, err)

	// Reset viper and initialize
	viper.Reset()
	viper.AddConfigPath(tmpDir)

	err = Initialize()
	require.NoError(t, err)

	// Verify ConfigFileUsed returns the correct path
	usedPath := ConfigFileUsed()
	assert.Equal(t, configPath, usedPath)
}

func TestConfigWithJSONFormat(t *testing.T) {
	// Create a temporary directory with JSON config
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.json")
	jsonContent := `{
		"general": {
			"editor": "code",
			"output_format": "json"
		},
		"logging": {
			"level": "debug",
			"format": "json"
		}
	}`
	err = os.WriteFile(configPath, []byte(jsonContent), 0o644)
	require.NoError(t, err)

	// Reset viper and initialize
	viper.Reset()
	viper.AddConfigPath(tmpDir)

	err = Initialize()
	require.NoError(t, err)

	// Verify JSON config was loaded
	assert.Equal(t, "code", GetString("general.editor"))
	assert.Equal(t, "json", GetString("general.output_format"))
	assert.Equal(t, "debug", GetString("logging.level"))
	assert.Equal(t, "json", GetString("logging.format"))
}

func TestConfigWithYAMLFormat(t *testing.T) {
	// Create a temporary directory with YAML config
	tmpDir, err := os.MkdirTemp("", "grove-config-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	configPath := filepath.Join(tmpDir, "config.yaml")
	yamlContent := `
general:
  editor: vim
  output_format: text
logging:
  level: warn
  format: text
git:
  default_remote: origin
  fetch_timeout: 45s
`
	err = os.WriteFile(configPath, []byte(yamlContent), 0o644)
	require.NoError(t, err)

	// Reset viper and initialize
	viper.Reset()
	viper.AddConfigPath(tmpDir)

	err = Initialize()
	require.NoError(t, err)

	// Verify YAML config was loaded
	assert.Equal(t, "vim", GetString("general.editor"))
	assert.Equal(t, "text", GetString("general.output_format"))
	assert.Equal(t, "warn", GetString("logging.level"))
	assert.Equal(t, "origin", GetString("git.default_remote"))
	assert.Equal(t, 45*time.Second, GetDuration("git.fetch_timeout"))
}

func TestWriteConfigSecurity(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grove-security-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test WriteConfig with secure permissions
	t.Run("WriteConfig sets secure permissions", func(t *testing.T) {
		configPath := filepath.Join(tmpDir, "test-write-config.toml")

		// Reset viper and set test config
		viper.Reset()
		SetDefaults()
		Set("general.editor", "test-editor")
		viper.SetConfigFile(configPath)

		// Write config
		err := WriteConfig()
		require.NoError(t, err)

		// Check file exists and has correct permissions
		stat, err := os.Stat(configPath)
		require.NoError(t, err)

		// Check permissions (0o600 = -rw-------)
		expectedPerm := os.FileMode(0o600)
		actualPerm := stat.Mode().Perm()
		assert.Equal(t, expectedPerm, actualPerm, "Config file should have 0600 permissions")
	})

	// Test SafeWriteConfig with secure permissions
	t.Run("SafeWriteConfig sets secure permissions", func(t *testing.T) {
		// Reset viper and set test config
		viper.Reset()
		SetDefaults()
		Set("general.editor", "test-editor")

		// Get the expected config path and clean up any existing file
		configPaths := GetConfigPaths()
		require.NotEmpty(t, configPaths)
		expectedPath := filepath.Join(configPaths[0], "config.toml")

		// Clean up any existing config file
		_ = os.Remove(expectedPath)
		defer func() { _ = os.Remove(expectedPath) }() // Clean up after test

		// Safe write config
		err := SafeWriteConfig()
		require.NoError(t, err)

		// Check file exists and has correct permissions
		stat, err := os.Stat(expectedPath)
		require.NoError(t, err)

		// Check permissions
		expectedPerm := os.FileMode(0o600)
		actualPerm := stat.Mode().Perm()
		assert.Equal(t, expectedPerm, actualPerm, "Config file should have 0600 permissions")
	})
}

func TestWriteConfigAsPathValidation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grove-path-validation-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Setup viper with test config
	viper.Reset()
	SetDefaults()
	Set("general.editor", "test-editor")

	tests := []struct {
		name        string
		path        string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid absolute path",
			path:        filepath.Join(tmpDir, "valid-config.toml"),
			expectError: false,
		},
		{
			name:        "valid relative path",
			path:        "valid-relative-config.toml",
			expectError: false,
		},
		{
			name:        "path traversal with double dots",
			path:        "../../../etc/passwd.toml",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "path traversal in absolute path",
			path:        "/tmp/../../../etc/passwd.toml",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
		{
			name:        "nested path traversal",
			path:        "config/../../../sensitive-file.toml",
			expectError: true,
			errorMsg:    "path traversal detected",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WriteConfigAs(tt.path)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)

				// For valid paths, check file was created with correct permissions
				cleanPath := filepath.Clean(tt.path)
				if !filepath.IsAbs(cleanPath) {
					cleanPath, _ = filepath.Abs(cleanPath)
				}

				stat, err := os.Stat(cleanPath)
				require.NoError(t, err)

				expectedPerm := os.FileMode(0o600)
				actualPerm := stat.Mode().Perm()
				assert.Equal(t, expectedPerm, actualPerm, "Config file should have 0600 permissions")

				// Cleanup
				_ = os.Remove(cleanPath)
			}
		})
	}
}

func TestWriteConfigAsDirectoryCreation(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "grove-dir-creation-test")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Setup viper with test config
	viper.Reset()
	SetDefaults()
	Set("general.editor", "test-editor")

	// Test creating config in nested directory
	nestedPath := filepath.Join(tmpDir, "nested", "deep", "config.toml")

	err = WriteConfigAs(nestedPath)
	require.NoError(t, err)

	// Check directory was created with correct permissions
	nestedDir := filepath.Join(tmpDir, "nested", "deep")
	stat, err := os.Stat(nestedDir)
	require.NoError(t, err)
	assert.True(t, stat.IsDir())

	// Check directory permissions (0o700 = drwx------)
	expectedDirPerm := os.FileMode(0o700)
	actualDirPerm := stat.Mode().Perm()
	assert.Equal(t, expectedDirPerm, actualDirPerm, "Config directory should have 0700 permissions")

	// Check file was created with correct permissions
	fileStat, err := os.Stat(nestedPath)
	require.NoError(t, err)

	expectedFilePerm := os.FileMode(0o600)
	actualFilePerm := fileStat.Mode().Perm()
	assert.Equal(t, expectedFilePerm, actualFilePerm, "Config file should have 0600 permissions")
}
