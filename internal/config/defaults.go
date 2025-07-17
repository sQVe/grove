package config

import (
	"time"

	"github.com/spf13/viper"
)

// SetDefaults sets all default configuration values
func SetDefaults() {
	// General defaults
	viper.SetDefault("general.editor", getDefaultEditor())
	viper.SetDefault("general.pager", getDefaultPager())
	viper.SetDefault("general.output_format", "text")

	// Git defaults
	viper.SetDefault("git.default_remote", "origin")
	viper.SetDefault("git.fetch_timeout", 30*time.Second)
	viper.SetDefault("git.max_retries", 3)

	// Retry defaults (matching existing retry system)
	viper.SetDefault("retry.max_attempts", 3)
	viper.SetDefault("retry.base_delay", 1*time.Second)
	viper.SetDefault("retry.max_delay", 10*time.Second)
	viper.SetDefault("retry.jitter_enabled", true)

	// Logging defaults (matching existing logger)
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")

	// Worktree defaults
	viper.SetDefault("worktree.naming_pattern", "branch")
	viper.SetDefault("worktree.cleanup_threshold", 30*24*time.Hour) // 30 days
}

// getDefaultPager returns the default pager based on environment
func getDefaultPager() string {
	if pager := viper.GetString("PAGER"); pager != "" {
		return pager
	}
	return "less"
}

// DefaultConfig returns a Config struct with all default values
func DefaultConfig() *Config {
	// Create a temporary viper instance to get defaults
	v := viper.New()

	// Set defaults on the temporary instance
	v.SetDefault("general.editor", getDefaultEditor())
	v.SetDefault("general.pager", getDefaultPager())
	v.SetDefault("general.output_format", "text")

	v.SetDefault("git.default_remote", "origin")
	v.SetDefault("git.fetch_timeout", 30*time.Second)
	v.SetDefault("git.max_retries", 3)

	v.SetDefault("retry.max_attempts", 3)
	v.SetDefault("retry.base_delay", 1*time.Second)
	v.SetDefault("retry.max_delay", 10*time.Second)
	v.SetDefault("retry.jitter_enabled", true)

	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "text")

	v.SetDefault("worktree.naming_pattern", "branch")
	v.SetDefault("worktree.cleanup_threshold", 30*24*time.Hour)

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		// This should never happen with defaults, but handle gracefully
		return &Config{
			General: struct {
				Editor       string `mapstructure:"editor"`
				Pager        string `mapstructure:"pager"`
				OutputFormat string `mapstructure:"output_format"`
			}{
				Editor:       getDefaultEditor(),
				Pager:        getDefaultPager(),
				OutputFormat: "text",
			},
			Git: struct {
				DefaultRemote string        `mapstructure:"default_remote"`
				FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
				MaxRetries    int           `mapstructure:"max_retries"`
			}{
				DefaultRemote: "origin",
				FetchTimeout:  30 * time.Second,
				MaxRetries:    3,
			},
			Retry: struct {
				MaxAttempts int           `mapstructure:"max_attempts"`
				BaseDelay   time.Duration `mapstructure:"base_delay"`
				MaxDelay    time.Duration `mapstructure:"max_delay"`
				Jitter      bool          `mapstructure:"jitter_enabled"`
			}{
				MaxAttempts: 3,
				BaseDelay:   1 * time.Second,
				MaxDelay:    10 * time.Second,
				Jitter:      true,
			},
			Logging: struct {
				Level  string `mapstructure:"level"`
				Format string `mapstructure:"format"`
			}{
				Level:  "info",
				Format: "text",
			},
			Worktree: struct {
				NamingPattern    string        `mapstructure:"naming_pattern"`
				CleanupThreshold time.Duration `mapstructure:"cleanup_threshold"`
			}{
				NamingPattern:    "branch",
				CleanupThreshold: 30 * 24 * time.Hour,
			},
		}
	}
	return &config
}

// ValidLogLevels returns the valid log levels
func ValidLogLevels() []string {
	return []string{"debug", "info", "warn", "error"}
}

// ValidOutputFormats returns the valid output formats
func ValidOutputFormats() []string {
	return []string{"text", "json"}
}

// ValidLogFormats returns the valid log formats
func ValidLogFormats() []string {
	return []string{"text", "json"}
}

// ValidNamingPatterns returns the valid worktree naming patterns
func ValidNamingPatterns() []string {
	return []string{"branch", "slug", "timestamp"}
}
