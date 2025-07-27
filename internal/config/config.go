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
	ConfigFilePermissions = 0o600
	ConfigDirPermissions = 0o700
)

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

func Initialize() error {
	viper.SetConfigName("config")

	configPaths := GetConfigPaths()
	for _, path := range configPaths {
		viper.AddConfigPath(path)
	}

	viper.SetEnvPrefix("GROVE")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	SetDefaults()

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; using defaults.
			return nil
		}
		return fmt.Errorf("error reading config file: %w", err)
	}

	return nil
}

func Get() (*Config, error) {
	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}
	return &config, nil
}

func GetString(key string) string {
	return viper.GetString(key)
}

func GetInt(key string) int {
	return viper.GetInt(key)
}

func GetBool(key string) bool {
	return viper.GetBool(key)
}

func GetDuration(key string) time.Duration {
	return viper.GetDuration(key)
}

func Set(key string, value interface{}) {
	viper.Set(key, value)
}

func IsSet(key string) bool {
	return viper.IsSet(key)
}

func ConfigFileUsed() string {
	return viper.ConfigFileUsed()
}

func WriteConfig() error {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return fmt.Errorf("no config file in use")
	}

	return secureWriteConfigToFile(configFile)
}

func SafeWriteConfig() error {
	configPaths := GetConfigPaths()
	if len(configPaths) == 0 {
		return fmt.Errorf("no config paths available")
	}

	targetDir := configPaths[0]
	if err := os.MkdirAll(targetDir, ConfigDirPermissions); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configName := "config" // Default config name
	configType := "toml"   // Default config type
	configFile := filepath.Join(targetDir, configName+"."+configType)

	// Check if file already exists (safe write behavior).
	if _, err := os.Stat(configFile); err == nil {
		return fmt.Errorf("config file already exists at %s", configFile)
	}

	return secureWriteConfigToFile(configFile)
}

func WriteConfigAs(filename string) error {
	if strings.Contains(filename, "..") {
		return fmt.Errorf("invalid file path: path traversal detected in %s", filename)
	}

	cleanPath := filepath.Clean(filename)
	if !filepath.IsAbs(cleanPath) {
		absPath, err := filepath.Abs(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to resolve absolute path for %s: %w", filename, err)
		}
		cleanPath = absPath
	}

	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("invalid file path: path traversal detected in %s", filename)
	}

	dir := filepath.Dir(cleanPath)
	if err := os.MkdirAll(dir, ConfigDirPermissions); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	return secureWriteConfigToFile(cleanPath)
}

func AllSettings() map[string]interface{} {
	return viper.AllSettings()
}

// Writes the current viper configuration to a file with secure permissions.
// This function creates the file with 0600 permissions from the start,
// following Go security best practices.
func secureWriteConfigToFile(configPath string) error {
	// Create file with secure permissions (0600 = read/write for owner only).
	file, err := os.OpenFile(configPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, ConfigFilePermissions)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() { _ = file.Close() }()

	allSettings := viper.AllSettings()

	encoder := toml.NewEncoder(file)
	if err := encoder.Encode(allSettings); err != nil {
		return fmt.Errorf("failed to encode config to TOML: %w", err)
	}

	return nil
}

func getDefaultEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	return "vi"
}
