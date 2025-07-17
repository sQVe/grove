package config

import (
	"fmt"
	"slices"
	"strings"
	"time"
)

// ValidationError represents a configuration validation error
type ValidationError struct {
	Field   string
	Value   interface{}
	Message string
}

// Error implements the error interface
func (e ValidationError) Error() string {
	return fmt.Sprintf("config validation error for field '%s': %s (value: %v)", e.Field, e.Message, e.Value)
}

// ValidationErrors represents multiple validation errors
type ValidationErrors []ValidationError

// Error implements the error interface
func (e ValidationErrors) Error() string {
	if len(e) == 0 {
		return "no validation errors"
	}

	var messages []string
	for _, err := range e {
		messages = append(messages, err.Error())
	}
	return fmt.Sprintf("configuration validation failed:\n%s", strings.Join(messages, "\n"))
}

// Validate validates the current configuration
func Validate() error {
	config, err := Get()
	if err != nil {
		return fmt.Errorf("failed to get config for validation: %w", err)
	}

	return ValidateConfig(config)
}

// ValidateConfig validates a configuration struct
func ValidateConfig(config *Config) error {
	var errors ValidationErrors

	// Validate general configuration
	if err := validateGeneral(&config.General); err != nil {
		errors = append(errors, err...)
	}

	// Validate git configuration
	if err := validateGit(&config.Git); err != nil {
		errors = append(errors, err...)
	}

	// Validate retry configuration
	if err := validateRetry(&config.Retry); err != nil {
		errors = append(errors, err...)
	}

	// Validate logging configuration
	if err := validateLogging(&config.Logging); err != nil {
		errors = append(errors, err...)
	}

	// Validate worktree configuration
	if err := validateWorktree(&config.Worktree); err != nil {
		errors = append(errors, err...)
	}

	if len(errors) > 0 {
		return errors
	}

	return nil
}

// validateGeneral validates general configuration
func validateGeneral(config *struct {
	Editor       string `mapstructure:"editor"`
	Pager        string `mapstructure:"pager"`
	OutputFormat string `mapstructure:"output_format"`
},
) ValidationErrors {
	var errors ValidationErrors

	// Validate editor
	if config.Editor == "" {
		errors = append(errors, ValidationError{
			Field:   "general.editor",
			Value:   config.Editor,
			Message: "editor cannot be empty",
		})
	}

	// Validate pager
	if config.Pager == "" {
		errors = append(errors, ValidationError{
			Field:   "general.pager",
			Value:   config.Pager,
			Message: "pager cannot be empty",
		})
	}

	// Validate output format
	validFormats := ValidOutputFormats()
	if !slices.Contains(validFormats, config.OutputFormat) {
		errors = append(errors, ValidationError{
			Field:   "general.output_format",
			Value:   config.OutputFormat,
			Message: fmt.Sprintf("must be one of: %v", validFormats),
		})
	}

	return errors
}

// validateGit validates git configuration
func validateGit(config *struct {
	DefaultRemote string        `mapstructure:"default_remote"`
	FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
	MaxRetries    int           `mapstructure:"max_retries"`
},
) ValidationErrors {
	var errors ValidationErrors

	// Validate default remote
	if config.DefaultRemote == "" {
		errors = append(errors, ValidationError{
			Field:   "git.default_remote",
			Value:   config.DefaultRemote,
			Message: "default remote cannot be empty",
		})
	}

	// Validate fetch timeout
	if config.FetchTimeout <= 0 {
		errors = append(errors, ValidationError{
			Field:   "git.fetch_timeout",
			Value:   config.FetchTimeout,
			Message: "fetch timeout must be positive",
		})
	}

	if config.FetchTimeout > 10*time.Minute {
		errors = append(errors, ValidationError{
			Field:   "git.fetch_timeout",
			Value:   config.FetchTimeout,
			Message: "fetch timeout should not exceed 10 minutes",
		})
	}

	// Validate max retries
	if config.MaxRetries < 0 {
		errors = append(errors, ValidationError{
			Field:   "git.max_retries",
			Value:   config.MaxRetries,
			Message: "max retries cannot be negative",
		})
	}

	if config.MaxRetries > 10 {
		errors = append(errors, ValidationError{
			Field:   "git.max_retries",
			Value:   config.MaxRetries,
			Message: "max retries should not exceed 10",
		})
	}

	return errors
}

// validateRetry validates retry configuration
func validateRetry(config *struct {
	MaxAttempts int           `mapstructure:"max_attempts"`
	BaseDelay   time.Duration `mapstructure:"base_delay"`
	MaxDelay    time.Duration `mapstructure:"max_delay"`
	Jitter      bool          `mapstructure:"jitter_enabled"`
},
) ValidationErrors {
	var errors ValidationErrors

	// Validate max attempts
	if config.MaxAttempts < 1 {
		errors = append(errors, ValidationError{
			Field:   "retry.max_attempts",
			Value:   config.MaxAttempts,
			Message: "max attempts must be at least 1",
		})
	}

	if config.MaxAttempts > 10 {
		errors = append(errors, ValidationError{
			Field:   "retry.max_attempts",
			Value:   config.MaxAttempts,
			Message: "max attempts should not exceed 10",
		})
	}

	// Validate base delay
	if config.BaseDelay <= 0 {
		errors = append(errors, ValidationError{
			Field:   "retry.base_delay",
			Value:   config.BaseDelay,
			Message: "base delay must be positive",
		})
	}

	// Validate max delay
	if config.MaxDelay <= 0 {
		errors = append(errors, ValidationError{
			Field:   "retry.max_delay",
			Value:   config.MaxDelay,
			Message: "max delay must be positive",
		})
	}

	// Validate base delay <= max delay
	if config.BaseDelay > config.MaxDelay {
		errors = append(errors, ValidationError{
			Field:   "retry.base_delay",
			Value:   config.BaseDelay,
			Message: "base delay cannot be greater than max delay",
		})
	}

	return errors
}

// validateLogging validates logging configuration
func validateLogging(config *struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
},
) ValidationErrors {
	var errors ValidationErrors

	// Validate log level
	validLevels := ValidLogLevels()
	if !slices.Contains(validLevels, config.Level) {
		errors = append(errors, ValidationError{
			Field:   "logging.level",
			Value:   config.Level,
			Message: fmt.Sprintf("must be one of: %v", validLevels),
		})
	}

	// Validate log format
	validFormats := ValidLogFormats()
	if !slices.Contains(validFormats, config.Format) {
		errors = append(errors, ValidationError{
			Field:   "logging.format",
			Value:   config.Format,
			Message: fmt.Sprintf("must be one of: %v", validFormats),
		})
	}

	return errors
}

// validateWorktree validates worktree configuration
func validateWorktree(config *struct {
	NamingPattern    string        `mapstructure:"naming_pattern"`
	CleanupThreshold time.Duration `mapstructure:"cleanup_threshold"`
},
) ValidationErrors {
	var errors ValidationErrors

	// Validate naming pattern
	validPatterns := ValidNamingPatterns()
	if !slices.Contains(validPatterns, config.NamingPattern) {
		errors = append(errors, ValidationError{
			Field:   "worktree.naming_pattern",
			Value:   config.NamingPattern,
			Message: fmt.Sprintf("must be one of: %v", validPatterns),
		})
	}

	// Validate cleanup threshold
	if config.CleanupThreshold <= 0 {
		errors = append(errors, ValidationError{
			Field:   "worktree.cleanup_threshold",
			Value:   config.CleanupThreshold,
			Message: "cleanup threshold must be positive",
		})
	}

	// Warn if cleanup threshold is very short (less than 1 day)
	if config.CleanupThreshold < 24*time.Hour {
		errors = append(errors, ValidationError{
			Field:   "worktree.cleanup_threshold",
			Value:   config.CleanupThreshold,
			Message: "cleanup threshold less than 1 day may cause data loss",
		})
	}

	return errors
}

// ValidateKey validates a specific configuration key
func ValidateKey(key string, value interface{}) error {
	// Get current config to validate against
	config, err := Get()
	if err != nil {
		return fmt.Errorf("failed to get config for validation: %w", err)
	}

	// Temporarily set the value
	original := GetString(key)
	Set(key, value)
	defer Set(key, original)

	// Validate the updated config
	return ValidateConfig(config)
}

// IsValidKey checks if a configuration key is valid
func IsValidKey(key string) bool {
	validKeys := []string{
		"general.editor",
		"general.pager",
		"general.output_format",
		"git.default_remote",
		"git.fetch_timeout",
		"git.max_retries",
		"retry.max_attempts",
		"retry.base_delay",
		"retry.max_delay",
		"retry.jitter_enabled",
		"logging.level",
		"logging.format",
		"worktree.naming_pattern",
		"worktree.cleanup_threshold",
	}

	return slices.Contains(validKeys, key)
}

// GetValidKeys returns all valid configuration keys
func GetValidKeys() []string {
	return []string{
		"general.editor",
		"general.pager",
		"general.output_format",
		"git.default_remote",
		"git.fetch_timeout",
		"git.max_retries",
		"retry.max_attempts",
		"retry.base_delay",
		"retry.max_delay",
		"retry.jitter_enabled",
		"logging.level",
		"logging.format",
		"worktree.naming_pattern",
		"worktree.cleanup_threshold",
	}
}
