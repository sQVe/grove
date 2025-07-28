//go:build !integration
// +build !integration

package config

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorCount  int
	}{
		{
			name:        "valid default config",
			config:      DefaultConfig(),
			expectError: false,
		},
		{
			name: "invalid output format",
			config: &Config{
				General: struct {
					Editor       string `mapstructure:"editor"`
					Pager        string `mapstructure:"pager"`
					OutputFormat string `mapstructure:"output_format"`
				}{
					Editor:       "vim",
					Pager:        "less",
					OutputFormat: "invalid",
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
					NamingPattern    string          `mapstructure:"naming_pattern"`
					CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
					BasePath         string          `mapstructure:"base_path"`
					AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
					CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
				}{
					NamingPattern:    "branch",
					CleanupThreshold: 30 * 24 * time.Hour,
					BasePath:         "",
					AutoTrackRemote:  true,
					CopyFiles: CopyFilesConfig{
						Patterns:       []string{},
						SourceWorktree: "main",
						OnConflict:     "prompt",
					},
				},
				Create: struct {
					DefaultBaseBranch  string `mapstructure:"default_base_branch"`
					PromptForNewBranch bool   `mapstructure:"prompt_for_new_branch"`
					AutoCreateParents  bool   `mapstructure:"auto_create_parents"`
				}{
					DefaultBaseBranch:  "main",
					PromptForNewBranch: true,
					AutoCreateParents:  true,
				},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "empty editor",
			config: &Config{
				General: struct {
					Editor       string `mapstructure:"editor"`
					Pager        string `mapstructure:"pager"`
					OutputFormat string `mapstructure:"output_format"`
				}{
					Editor:       "",
					Pager:        "less",
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
					NamingPattern    string          `mapstructure:"naming_pattern"`
					CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
					BasePath         string          `mapstructure:"base_path"`
					AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
					CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
				}{
					NamingPattern:    "branch",
					CleanupThreshold: 30 * 24 * time.Hour,
					BasePath:         "",
					AutoTrackRemote:  true,
					CopyFiles: CopyFilesConfig{
						Patterns:       []string{},
						SourceWorktree: "main",
						OnConflict:     "prompt",
					},
				},
				Create: struct {
					DefaultBaseBranch  string `mapstructure:"default_base_branch"`
					PromptForNewBranch bool   `mapstructure:"prompt_for_new_branch"`
					AutoCreateParents  bool   `mapstructure:"auto_create_parents"`
				}{
					DefaultBaseBranch:  "main",
					PromptForNewBranch: true,
					AutoCreateParents:  true,
				},
			},
			expectError: true,
			errorCount:  1,
		},
		{
			name: "invalid retry configuration",
			config: &Config{
				General: struct {
					Editor       string `mapstructure:"editor"`
					Pager        string `mapstructure:"pager"`
					OutputFormat string `mapstructure:"output_format"`
				}{
					Editor:       "vim",
					Pager:        "less",
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
					MaxAttempts: 0,
					BaseDelay:   5 * time.Second,
					MaxDelay:    1 * time.Second, // Invalid: base > max.
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
					NamingPattern    string          `mapstructure:"naming_pattern"`
					CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
					BasePath         string          `mapstructure:"base_path"`
					AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
					CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
				}{
					NamingPattern:    "branch",
					CleanupThreshold: 30 * 24 * time.Hour,
					BasePath:         "",
					AutoTrackRemote:  true,
					CopyFiles: CopyFilesConfig{
						Patterns:       []string{},
						SourceWorktree: "main",
						OnConflict:     "prompt",
					},
				},
				Create: struct {
					DefaultBaseBranch  string `mapstructure:"default_base_branch"`
					PromptForNewBranch bool   `mapstructure:"prompt_for_new_branch"`
					AutoCreateParents  bool   `mapstructure:"auto_create_parents"`
				}{
					DefaultBaseBranch:  "main",
					PromptForNewBranch: true,
					AutoCreateParents:  true,
				},
			},
			expectError: true,
			errorCount:  2, // Invalid max_attempts and delay relationship.
		},
		{
			name: "multiple validation errors",
			config: &Config{
				General: struct {
					Editor       string `mapstructure:"editor"`
					Pager        string `mapstructure:"pager"`
					OutputFormat string `mapstructure:"output_format"`
				}{
					Editor:       "",
					Pager:        "",
					OutputFormat: "invalid",
				},
				Git: struct {
					DefaultRemote string        `mapstructure:"default_remote"`
					FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
					MaxRetries    int           `mapstructure:"max_retries"`
				}{
					DefaultRemote: "",
					FetchTimeout:  -1 * time.Second,
					MaxRetries:    -1,
				},
				Retry: struct {
					MaxAttempts int           `mapstructure:"max_attempts"`
					BaseDelay   time.Duration `mapstructure:"base_delay"`
					MaxDelay    time.Duration `mapstructure:"max_delay"`
					Jitter      bool          `mapstructure:"jitter_enabled"`
				}{
					MaxAttempts: 0,
					BaseDelay:   -1 * time.Second,
					MaxDelay:    -1 * time.Second,
					Jitter:      true,
				},
				Logging: struct {
					Level  string `mapstructure:"level"`
					Format string `mapstructure:"format"`
				}{
					Level:  "invalid",
					Format: "invalid",
				},
				Worktree: struct {
					NamingPattern    string          `mapstructure:"naming_pattern"`
					CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
					BasePath         string          `mapstructure:"base_path"`
					AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
					CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
				}{
					NamingPattern:    "invalid",
					CleanupThreshold: -1 * time.Hour,
					BasePath:         "",
					AutoTrackRemote:  true,
					CopyFiles: CopyFilesConfig{
						Patterns:       []string{},
						SourceWorktree: "main",
						OnConflict:     "prompt",
					},
				},
				Create: struct {
					DefaultBaseBranch  string `mapstructure:"default_base_branch"`
					PromptForNewBranch bool   `mapstructure:"prompt_for_new_branch"`
					AutoCreateParents  bool   `mapstructure:"auto_create_parents"`
				}{
					DefaultBaseBranch:  "main",
					PromptForNewBranch: true,
					AutoCreateParents:  true,
				},
			},
			expectError: true,
			errorCount:  14, // Errors across all validation sections.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.expectError {
				require.Error(t, err)
				var validationErrors ValidationErrors
				require.ErrorAs(t, err, &validationErrors)
				assert.Len(t, validationErrors, tt.errorCount)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateGeneral(t *testing.T) {
	tests := []struct {
		name   string
		config *struct {
			Editor       string `mapstructure:"editor"`
			Pager        string `mapstructure:"pager"`
			OutputFormat string `mapstructure:"output_format"`
		}
		expectErrors int
	}{
		{
			name: "valid config",
			config: &struct {
				Editor       string `mapstructure:"editor"`
				Pager        string `mapstructure:"pager"`
				OutputFormat string `mapstructure:"output_format"`
			}{
				Editor:       "vim",
				Pager:        "less",
				OutputFormat: "text",
			},
			expectErrors: 0,
		},
		{
			name: "empty editor",
			config: &struct {
				Editor       string `mapstructure:"editor"`
				Pager        string `mapstructure:"pager"`
				OutputFormat string `mapstructure:"output_format"`
			}{
				Editor:       "",
				Pager:        "less",
				OutputFormat: "text",
			},
			expectErrors: 1,
		},
		{
			name: "invalid output format",
			config: &struct {
				Editor       string `mapstructure:"editor"`
				Pager        string `mapstructure:"pager"`
				OutputFormat string `mapstructure:"output_format"`
			}{
				Editor:       "vim",
				Pager:        "less",
				OutputFormat: "invalid",
			},
			expectErrors: 1,
		},
		{
			name: "multiple errors",
			config: &struct {
				Editor       string `mapstructure:"editor"`
				Pager        string `mapstructure:"pager"`
				OutputFormat string `mapstructure:"output_format"`
			}{
				Editor:       "",
				Pager:        "",
				OutputFormat: "invalid",
			},
			expectErrors: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateGeneral(tt.config)
			assert.Len(t, errors, tt.expectErrors)
		})
	}
}

func TestValidateGit(t *testing.T) {
	tests := []struct {
		name   string
		config *struct {
			DefaultRemote string        `mapstructure:"default_remote"`
			FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
			MaxRetries    int           `mapstructure:"max_retries"`
		}
		expectErrors int
	}{
		{
			name: "valid config",
			config: &struct {
				DefaultRemote string        `mapstructure:"default_remote"`
				FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
				MaxRetries    int           `mapstructure:"max_retries"`
			}{
				DefaultRemote: "origin",
				FetchTimeout:  30 * time.Second,
				MaxRetries:    3,
			},
			expectErrors: 0,
		},
		{
			name: "empty default remote",
			config: &struct {
				DefaultRemote string        `mapstructure:"default_remote"`
				FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
				MaxRetries    int           `mapstructure:"max_retries"`
			}{
				DefaultRemote: "",
				FetchTimeout:  30 * time.Second,
				MaxRetries:    3,
			},
			expectErrors: 1,
		},
		{
			name: "invalid timeout",
			config: &struct {
				DefaultRemote string        `mapstructure:"default_remote"`
				FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
				MaxRetries    int           `mapstructure:"max_retries"`
			}{
				DefaultRemote: "origin",
				FetchTimeout:  -1 * time.Second,
				MaxRetries:    3,
			},
			expectErrors: 1,
		},
		{
			name: "timeout too long",
			config: &struct {
				DefaultRemote string        `mapstructure:"default_remote"`
				FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
				MaxRetries    int           `mapstructure:"max_retries"`
			}{
				DefaultRemote: "origin",
				FetchTimeout:  15 * time.Minute,
				MaxRetries:    3,
			},
			expectErrors: 1,
		},
		{
			name: "negative retries",
			config: &struct {
				DefaultRemote string        `mapstructure:"default_remote"`
				FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
				MaxRetries    int           `mapstructure:"max_retries"`
			}{
				DefaultRemote: "origin",
				FetchTimeout:  30 * time.Second,
				MaxRetries:    -1,
			},
			expectErrors: 1,
		},
		{
			name: "too many retries",
			config: &struct {
				DefaultRemote string        `mapstructure:"default_remote"`
				FetchTimeout  time.Duration `mapstructure:"fetch_timeout"`
				MaxRetries    int           `mapstructure:"max_retries"`
			}{
				DefaultRemote: "origin",
				FetchTimeout:  30 * time.Second,
				MaxRetries:    15,
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateGit(tt.config)
			assert.Len(t, errors, tt.expectErrors)
		})
	}
}

func TestValidateRetry(t *testing.T) {
	tests := []struct {
		name   string
		config *struct {
			MaxAttempts int           `mapstructure:"max_attempts"`
			BaseDelay   time.Duration `mapstructure:"base_delay"`
			MaxDelay    time.Duration `mapstructure:"max_delay"`
			Jitter      bool          `mapstructure:"jitter_enabled"`
		}
		expectErrors int
	}{
		{
			name: "valid config",
			config: &struct {
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
			expectErrors: 0,
		},
		{
			name: "zero max attempts",
			config: &struct {
				MaxAttempts int           `mapstructure:"max_attempts"`
				BaseDelay   time.Duration `mapstructure:"base_delay"`
				MaxDelay    time.Duration `mapstructure:"max_delay"`
				Jitter      bool          `mapstructure:"jitter_enabled"`
			}{
				MaxAttempts: 0,
				BaseDelay:   1 * time.Second,
				MaxDelay:    10 * time.Second,
				Jitter:      true,
			},
			expectErrors: 1,
		},
		{
			name: "too many attempts",
			config: &struct {
				MaxAttempts int           `mapstructure:"max_attempts"`
				BaseDelay   time.Duration `mapstructure:"base_delay"`
				MaxDelay    time.Duration `mapstructure:"max_delay"`
				Jitter      bool          `mapstructure:"jitter_enabled"`
			}{
				MaxAttempts: 15,
				BaseDelay:   1 * time.Second,
				MaxDelay:    10 * time.Second,
				Jitter:      true,
			},
			expectErrors: 1,
		},
		{
			name: "negative base delay",
			config: &struct {
				MaxAttempts int           `mapstructure:"max_attempts"`
				BaseDelay   time.Duration `mapstructure:"base_delay"`
				MaxDelay    time.Duration `mapstructure:"max_delay"`
				Jitter      bool          `mapstructure:"jitter_enabled"`
			}{
				MaxAttempts: 3,
				BaseDelay:   -1 * time.Second,
				MaxDelay:    10 * time.Second,
				Jitter:      true,
			},
			expectErrors: 1,
		},
		{
			name: "base delay greater than max delay",
			config: &struct {
				MaxAttempts int           `mapstructure:"max_attempts"`
				BaseDelay   time.Duration `mapstructure:"base_delay"`
				MaxDelay    time.Duration `mapstructure:"max_delay"`
				Jitter      bool          `mapstructure:"jitter_enabled"`
			}{
				MaxAttempts: 3,
				BaseDelay:   10 * time.Second,
				MaxDelay:    5 * time.Second,
				Jitter:      true,
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateRetry(tt.config)
			assert.Len(t, errors, tt.expectErrors)
		})
	}
}

func TestValidateLogging(t *testing.T) {
	tests := []struct {
		name   string
		config *struct {
			Level  string `mapstructure:"level"`
			Format string `mapstructure:"format"`
		}
		expectErrors int
	}{
		{
			name: "valid config",
			config: &struct {
				Level  string `mapstructure:"level"`
				Format string `mapstructure:"format"`
			}{
				Level:  "info",
				Format: "text",
			},
			expectErrors: 0,
		},
		{
			name: "invalid level",
			config: &struct {
				Level  string `mapstructure:"level"`
				Format string `mapstructure:"format"`
			}{
				Level:  "invalid",
				Format: "text",
			},
			expectErrors: 1,
		},
		{
			name: "invalid format",
			config: &struct {
				Level  string `mapstructure:"level"`
				Format string `mapstructure:"format"`
			}{
				Level:  "info",
				Format: "invalid",
			},
			expectErrors: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateLogging(tt.config)
			assert.Len(t, errors, tt.expectErrors)
		})
	}
}

func TestValidateWorktree(t *testing.T) {
	tests := []struct {
		name   string
		config *struct {
			NamingPattern    string          `mapstructure:"naming_pattern"`
			CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
			BasePath         string          `mapstructure:"base_path"`
			AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
			CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
		}
		expectErrors int
	}{
		{
			name: "valid config",
			config: &struct {
				NamingPattern    string          `mapstructure:"naming_pattern"`
				CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
				BasePath         string          `mapstructure:"base_path"`
				AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
				CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
			}{
				NamingPattern:    "branch",
				CleanupThreshold: 30 * 24 * time.Hour,
				BasePath:         "",
				AutoTrackRemote:  true,
				CopyFiles: CopyFilesConfig{
					Patterns:       []string{},
					SourceWorktree: "main",
					OnConflict:     "prompt",
				},
			},
			expectErrors: 0,
		},
		{
			name: "invalid naming pattern",
			config: &struct {
				NamingPattern    string          `mapstructure:"naming_pattern"`
				CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
				BasePath         string          `mapstructure:"base_path"`
				AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
				CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
			}{
				NamingPattern:    "invalid",
				CleanupThreshold: 30 * 24 * time.Hour,
				BasePath:         "",
				AutoTrackRemote:  true,
				CopyFiles: CopyFilesConfig{
					Patterns:       []string{},
					SourceWorktree: "main",
					OnConflict:     "prompt",
				},
			},
			expectErrors: 1,
		},
		{
			name: "negative cleanup threshold",
			config: &struct {
				NamingPattern    string          `mapstructure:"naming_pattern"`
				CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
				BasePath         string          `mapstructure:"base_path"`
				AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
				CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
			}{
				NamingPattern:    "branch",
				CleanupThreshold: -1 * time.Hour,
				BasePath:         "",
				AutoTrackRemote:  true,
				CopyFiles: CopyFilesConfig{
					Patterns:       []string{},
					SourceWorktree: "main",
					OnConflict:     "prompt",
				},
			},
			expectErrors: 2, // Negative threshold plus data loss warning.
		},
		{
			name: "short cleanup threshold (warning)",
			config: &struct {
				NamingPattern    string          `mapstructure:"naming_pattern"`
				CleanupThreshold time.Duration   `mapstructure:"cleanup_threshold"`
				BasePath         string          `mapstructure:"base_path"`
				AutoTrackRemote  bool            `mapstructure:"auto_track_remote"`
				CopyFiles        CopyFilesConfig `mapstructure:"copy_files"`
			}{
				NamingPattern:    "branch",
				CleanupThreshold: 1 * time.Hour,
				BasePath:         "",
				AutoTrackRemote:  true,
				CopyFiles: CopyFilesConfig{
					Patterns:       []string{},
					SourceWorktree: "main",
					OnConflict:     "prompt",
				},
			},
			expectErrors: 1, // Data loss warning below 24h threshold.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors := validateWorktree(tt.config)
			assert.Len(t, errors, tt.expectErrors)
		})
	}
}

func TestValidate(t *testing.T) {
	viper.Reset()
	SetDefaults()

	err := Validate()
	assert.NoError(t, err)

	Set("logging.level", "invalid")
	err = Validate()
	assert.Error(t, err)

	var validationErrors ValidationErrors
	assert.ErrorAs(t, err, &validationErrors)
	assert.Len(t, validationErrors, 1)
	assert.Contains(t, validationErrors[0].Error(), "logging.level")
}

func TestIsValidKey(t *testing.T) {
	tests := []struct {
		key   string
		valid bool
	}{
		{"general.editor", true},
		{"general.pager", true},
		{"general.output_format", true},
		{"git.default_remote", true},
		{"git.fetch_timeout", true},
		{"git.max_retries", true},
		{"retry.max_attempts", true},
		{"retry.base_delay", true},
		{"retry.max_delay", true},
		{"retry.jitter_enabled", true},
		{"logging.level", true},
		{"logging.format", true},
		{"worktree.naming_pattern", true},
		{"worktree.cleanup_threshold", true},
		{"invalid.key", false},
		{"general.invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, tt.valid, IsValidKey(tt.key))
		})
	}
}

func TestGetValidKeys(t *testing.T) {
	keys := GetValidKeys()

	expectedKeys := []string{
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
		"worktree.base_path",
		"worktree.auto_track_remote",
		"worktree.copy_files.patterns",
		"worktree.copy_files.source_worktree",
		"worktree.copy_files.on_conflict",
		"create.default_base_branch",
		"create.prompt_for_new_branch",
		"create.auto_create_parents",
	}

	assert.ElementsMatch(t, expectedKeys, keys)
}

func TestValidationError(t *testing.T) {
	err := ValidationError{
		Field:   "test.field",
		Value:   "test_value",
		Message: "test message",
	}

	expected := "config validation error for field 'test.field': test message (value: test_value)"
	assert.Equal(t, expected, err.Error())
}

func TestValidationErrors(t *testing.T) {
	errors := ValidationErrors{
		{Field: "field1", Value: "value1", Message: "message1"},
		{Field: "field2", Value: "value2", Message: "message2"},
	}

	errorMsg := errors.Error()
	assert.Contains(t, errorMsg, "configuration validation failed:")
	assert.Contains(t, errorMsg, "field1")
	assert.Contains(t, errorMsg, "field2")
	assert.Contains(t, errorMsg, "message1")
	assert.Contains(t, errorMsg, "message2")

	emptyErrors := ValidationErrors{}
	assert.Equal(t, "no validation errors", emptyErrors.Error())
}
