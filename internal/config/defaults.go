package config

import (
	"time"

	"github.com/spf13/viper"
)

func SetDefaults() {
	viper.SetDefault("general.editor", getDefaultEditor())
	viper.SetDefault("general.pager", getDefaultPager())
	viper.SetDefault("general.output_format", "text")

	viper.SetDefault("git.default_remote", "origin")
	viper.SetDefault("git.fetch_timeout", 30*time.Second)
	viper.SetDefault("git.max_retries", 3)

	viper.SetDefault("retry.max_attempts", 3)
	viper.SetDefault("retry.base_delay", 1*time.Second)
	viper.SetDefault("retry.max_delay", 10*time.Second)
	viper.SetDefault("retry.jitter_enabled", true)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.format", "text")

	viper.SetDefault("worktree.naming_pattern", "branch")
	viper.SetDefault("worktree.cleanup_threshold", 30*24*time.Hour) // 30 days.
	viper.SetDefault("worktree.base_path", "")                      // Empty means current directory.
	viper.SetDefault("worktree.auto_track_remote", true)
	viper.SetDefault("worktree.copy_files.patterns", []string{})
	viper.SetDefault("worktree.copy_files.source_worktree", "main")
	viper.SetDefault("worktree.copy_files.on_conflict", "prompt")

	viper.SetDefault("create.default_base_branch", "main")
	viper.SetDefault("create.prompt_for_new_branch", true)
	viper.SetDefault("create.auto_create_parents", true)
}

func getDefaultPager() string {
	if pager := viper.GetString("PAGER"); pager != "" {
		return pager
	}
	return "less"
}

func DefaultConfig() *Config {
	v := viper.New()

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
	v.SetDefault("worktree.base_path", "")
	v.SetDefault("worktree.auto_track_remote", true)
	v.SetDefault("worktree.copy_files.patterns", []string{})
	v.SetDefault("worktree.copy_files.source_worktree", "main")
	v.SetDefault("worktree.copy_files.on_conflict", "prompt")

	v.SetDefault("create.default_base_branch", "main")
	v.SetDefault("create.prompt_for_new_branch", true)
	v.SetDefault("create.auto_create_parents", true)

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
