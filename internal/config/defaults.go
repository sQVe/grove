package config

import (
	"time"

	"github.com/spf13/viper"
)

// Single source of truth for all configuration defaults.
var defaultValues = map[string]interface{}{
	"general.editor":                      getDefaultEditor,
	"general.pager":                       getDefaultPager,
	"general.output_format":               "text",
	"git.default_remote":                  "origin",
	"git.fetch_timeout":                   30 * time.Second,
	"git.max_retries":                     3,
	"retry.max_attempts":                  3,
	"retry.base_delay":                    1 * time.Second,
	"retry.max_delay":                     10 * time.Second,
	"retry.jitter_enabled":                true,
	"logging.level":                       "info",
	"logging.format":                      "text",
	"worktree.naming_pattern":             "branch",
	"worktree.cleanup_threshold":          30 * 24 * time.Hour,
	"worktree.base_path":                  "",
	"worktree.auto_track_remote":          true,
	"worktree.copy_files.patterns":        []string{},
	"worktree.copy_files.source_worktree": "main",
	"worktree.copy_files.on_conflict":     "prompt",
	"create.default_base_branch":          "main",
	"create.prompt_for_new_branch":        true,
	"create.auto_create_parents":          true,
	"errors.max_context_entries":          10,
}

func SetDefaults() {
	for key, value := range defaultValues {
		// Handle function values that need to be called.
		if fn, ok := value.(func() string); ok {
			viper.SetDefault(key, fn())
		} else {
			viper.SetDefault(key, value)
		}
	}
}

func getDefaultPager() string {
	if pager := viper.GetString("PAGER"); pager != "" {
		return pager
	}
	return "less"
}

func DefaultConfig() *Config {
	v := viper.New()

	// Apply defaults from single source of truth.
	for key, value := range defaultValues {
		// Handle function values that need to be called.
		if fn, ok := value.(func() string); ok {
			v.SetDefault(key, fn())
		} else {
			v.SetDefault(key, value)
		}
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		// Fallback if unmarshal fails with default values.
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
		}
	}
	return &config
}

func ValidLogLevels() []string {
	return []string{"debug", "info", "warn", "error"}
}

func ValidOutputFormats() []string {
	return []string{"text", "json"}
}

func ValidLogFormats() []string {
	return []string{"text", "json"}
}

func ValidNamingPatterns() []string {
	return []string{"branch", "slug", "timestamp"}
}

func ValidConflictStrategies() []string {
	return []string{"prompt", "skip", "overwrite", "backup"}
}
