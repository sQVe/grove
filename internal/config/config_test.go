package config

import (
	"os"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultValues(t *testing.T) {
	// Test default config structure values
	config := &Config{}

	// Verify zero values
	assert.Equal(t, "", config.General.Editor)
	assert.Equal(t, "", config.General.Pager)
	assert.Equal(t, "", config.General.OutputFormat)
	assert.Equal(t, "", config.Git.DefaultRemote)
	assert.Equal(t, time.Duration(0), config.Git.FetchTimeout)
	assert.Equal(t, 0, config.Git.MaxRetries)
}

func TestGetConfigPaths(t *testing.T) {
	paths := GetConfigPaths()

	// Should return at least one path
	assert.NotEmpty(t, paths)

	// Should include current directory or current directory path
	cwd, err := os.Getwd()
	assert.NoError(t, err)
	assert.Contains(t, paths, cwd)

	// Paths should be strings
	for _, path := range paths {
		assert.IsType(t, "", path)
	}
}

func TestInitialize_NoConfigFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-config-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Reset viper state
	viper.Reset()

	// Change to temp directory
	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Initialize should not fail even without config file
	err = Initialize()
	assert.NoError(t, err)

	// Viper should be configured
	assert.Equal(t, "GROVE", viper.GetEnvPrefix())
}

func TestInitialize_WithConfigFile(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-config-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Reset viper state
	viper.Reset()

	originalDir, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalDir) }()

	err = os.Chdir(tempDir)
	require.NoError(t, err)

	// Create a config file
	configContent := `
[general]
editor = "vim"
pager = "less"
output_format = "table"

[git]
default_remote = "origin"
fetch_timeout = "30s"
max_retries = 3

[logging]
level = "info"
format = "text"
`

	err = os.WriteFile("config.toml", []byte(configContent), ConfigFilePermissions)
	require.NoError(t, err)

	err = Initialize()
	assert.NoError(t, err)

	// Check that values were loaded
	assert.Equal(t, "vim", viper.GetString("general.editor"))
	assert.Equal(t, "origin", viper.GetString("git.default_remote"))
	assert.Equal(t, 3, viper.GetInt("git.max_retries"))
}

func TestEnvironmentVariables(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "grove-config-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Reset viper state
	viper.Reset()

	// Set environment variable
	err = os.Setenv("GROVE_GENERAL_EDITOR", "emacs")
	require.NoError(t, err)
	defer func() { _ = os.Unsetenv("GROVE_GENERAL_EDITOR") }()

	err = Initialize()
	assert.NoError(t, err)

	// Environment variable should override
	assert.Equal(t, "emacs", viper.GetString("general.editor"))
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Check default values are set correctly
	assert.NotNil(t, config)
	assert.Equal(t, "text", config.General.OutputFormat)
	assert.Equal(t, "origin", config.Git.DefaultRemote)
	assert.Equal(t, 3, config.Git.MaxRetries)
	assert.Equal(t, "info", config.Logging.Level)
	assert.Equal(t, "text", config.Logging.Format)
}

func TestGetString(t *testing.T) {
	viper.Reset()
	viper.Set("test.string", "test_value")

	result := GetString("test.string")
	assert.Equal(t, "test_value", result)

	// Non-existent key should return empty string
	result = GetString("nonexistent.key")
	assert.Equal(t, "", result)
}

func TestGetInt(t *testing.T) {
	viper.Reset()
	viper.Set("test.int", 42)

	result := GetInt("test.int")
	assert.Equal(t, 42, result)

	// Non-existent key should return 0
	result = GetInt("nonexistent.key")
	assert.Equal(t, 0, result)
}

func TestGetBool(t *testing.T) {
	viper.Reset()
	viper.Set("test.bool", true)

	result := GetBool("test.bool")
	assert.True(t, result)

	// Non-existent key should return false
	result = GetBool("nonexistent.key")
	assert.False(t, result)
}

func TestGetDuration(t *testing.T) {
	viper.Reset()
	viper.Set("test.duration", "30s")

	result := GetDuration("test.duration")
	assert.Equal(t, 30*time.Second, result)

	// Non-existent key should return 0
	result = GetDuration("nonexistent.key")
	assert.Equal(t, time.Duration(0), result)
}

func TestConfigFilePermissions(t *testing.T) {
	assert.Equal(t, int(0o600), int(ConfigFilePermissions))
}

func TestConfigDirPermissions(t *testing.T) {
	assert.Equal(t, int(0o700), int(ConfigDirPermissions))
}
