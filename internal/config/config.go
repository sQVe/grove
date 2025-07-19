package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/viper"
)

const (
	// ConfigFilePermissions defines secure permissions for configuration files (read/write for owner only).
	ConfigFilePermissions = 0o600
	// ConfigDirPermissions defines secure permissions for configuration directories (full access for owner only).
	ConfigDirPermissions = 0o700
)

// Config represents the complete configuration structure for Grove.
type Config struct {
	General struct {
		Editor       string `mapstructure:"editor"`
		Pager        string `mapstructure:"pager"`
		OutputFormat string `mapstructure:"output_format"`
	} `mapstructure:"general"`

	Git struct {
		DefaultRemote string        `mapstructure:"default_remote"`
		FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
		MaxRetries    int           `mapstructure:"max_retries"`
	} `mapstructure:"git"`

	Retry struct {
		MaxAttempts int           `mapstructure:"max_attempts"`
		BaseDelay   time.Duration `mapstructure:"base_delay"`
		MaxDelay    time.Duration `mapstructure:"max_delay"`
		Jitter      bool          `mapstructure:"jitter_enabled"`
	} `mapstructure:"retry"`

	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`

	Worktree struct {
		NamingPattern    string        `mapstructure:"naming_pattern"`
		CleanupThreshold time.Duration `mapstructure:"cleanup_threshold"`
	} `mapstructure:"worktree"`
}

// Initialize sets up Viper configuration with proper defaults and file paths.
func Initialize() error {
	viper.SetConfigName("config")

	configPaths := GetConfigPaths()
	for _, path := range configPaths {
		viper.AddConfigPath(path)
	}

	viper.SetEnvPrefix("GROVE")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set all defaults
	SetDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; using defaults
			return nil
		}
		// Config file was found but another error was produced
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

// Get returns the current configuration as a Config struct.
func Get() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}
	return &config, nil
}

// GetString returns a string configuration value.
func GetString(key string) string {
	return viper.GetString(key)
}

// GetInt returns an integer configuration value.
func GetInt(key string) int {
	return viper.GetInt(key)
}

// GetBool returns a boolean configuration value.
func GetBool(key string) bool {
	return viper.GetBool(key)
}

// GetDuration returns a duration configuration value.
func GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}

// Set sets a configuration value.
func Set(key string, value interface{}) {
	viper.Set(key, value)
}

// IsSet checks if a configuration key is set.
func IsSet(key string) bool {
	return viper.IsSet(key)
}

// ConfigFileUsed returns the path to the config file being used.
func ConfigFileUsed() string {
	return viper.ConfigFileUsed()
}

// WriteConfig writes the current configuration to file with secure permissions.
func WriteConfig() error {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return fmt.Errorf("no config file in use")
	}

	return secureWriteConfigToFile(configFile)
}

// SafeWriteConfig writes the current configuration to file if it doesn't exist with secure permissions.
func SafeWriteConfig() error {
	// Get the first config path where we should write the config
	configPaths := GetConfigPaths()
	if len(configPaths) == 0 {
		return fmt.Errorf("no config paths available")
	}

	// Use the first available path
	targetDir := configPaths[0]
	if err := os.MkdirAll(targetDir, ConfigDirPermissions); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Construct the config file path
	configName := "config" // Default config name
	configType := "toml"   // Default config type
	configFile := filepath.Join(targetDir, configName+"."+configType)

	// Check if file already exists (safe write behavior)
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config file already exists at %s", configFile)
	}

	// Write the config with our secure function
	return secureWriteConfigToFile(configFile)
}

// WriteConfigAs writes the current configuration to a specific file with secure permissions and path validation.
func WriteConfigAs(filename string) error {
	// Validate file path to prevent directory traversal attacks
	if strings.Contains(filename, "..") {
		return fmt.Errorf("invalid file path: path traversal detected in %s", filename)
	}

	// Clean the path and resolve to absolute
	cleanPath := filepath.Clean(filename)
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for %s: %w", filename, err)
		}
		cleanPath = absPath
	}

	// Additional check after path resolution
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid file path: path traversal detected in %s", filename)
	}

	// Ensure the directory exists with secure permissions
	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, ConfigDirPermissions); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	// Write config with secure permissions
	return secureWriteConfigToFile(cleanPath)
}

// AllSettings returns all configuration settings as a map.
func AllSettings() map[string]interface{} {
	return viper.AllSettings()
}

// secureWriteConfigToFile writes the current viper configuration to a file with secure permissions.
// This function creates the file with 0600 permissions from the start, following Go security best practices.
func secureWriteConfigToFile(configPath string) error {
	// Create file with secure permissions (0600 = read/write for owner only)
	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, ConfigFilePermissions)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Get all current configuration settings
	allSettings := viper.AllSettings()

	// Encode configuration as TOML
	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(allSettings); err != nil {
		return fmt.Errorf("failed to encode config to TOML: %w", err)
	}

	return nil
}

// getDefaultEditor returns the default editor based on environment variables.
func getDefaultEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vi"
}
