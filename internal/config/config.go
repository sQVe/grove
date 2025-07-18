package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"
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
	// Set config file name (type will be auto-detected from extension)
	viper.SetConfigName("config")

	// Add config paths
	configPaths := GetConfigPaths()
	for _, path := range configPaths {
		viper.AddConfigPath(path)
	}

	// Set environment variables configuration
	viper.SetEnvPrefix("GROVE")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set all defaults
	SetDefaults()

	// Try to read config file
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

// WriteConfig writes the current configuration to file.
func WriteConfig() error {
	return viper.WriteConfig()
}

// SafeWriteConfig writes the current configuration to file if it doesn't exist.
func SafeWriteConfig() error {
	return viper.SafeWriteConfig()
}

// WriteConfigAs writes the current configuration to a specific file.
func WriteConfigAs(filename string) error {
	return viper.WriteConfigAs(filename)
}

// AllSettings returns all configuration settings as a map.
func AllSettings() map[string]interface{} {
	return viper.AllSettings()
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
