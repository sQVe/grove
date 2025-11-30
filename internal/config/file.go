package config

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

const FileName = ".grove.toml"

//go:embed grove.template.toml
var initTemplate string

type FileConfig struct {
	Preserve struct {
		Patterns []string `toml:"patterns"`
	} `toml:"preserve"`
	Hooks struct {
		Add []string `toml:"add"`
	} `toml:"hooks"`
	Autolock struct {
		Patterns []string `toml:"patterns"`
	} `toml:"autolock"`
	Plain          bool   `toml:"plain"`
	Debug          bool   `toml:"debug"`
	NerdFonts      *bool  `toml:"nerd_fonts"`
	StaleThreshold string `toml:"stale_threshold"`
}

// LoadFromFile returns empty config if file missing, error if file invalid.
func LoadFromFile(dir string) (FileConfig, error) {
	var cfg FileConfig
	path := filepath.Join(dir, FileName)

	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func FileConfigExists(dir string) bool {
	path := filepath.Join(dir, FileName)
	_, err := os.Stat(path)
	return err == nil
}

// loadConfigWithWarning loads the TOML config and prints a warning on parse error.
// Returns the config and whether it was successfully loaded.
func loadConfigWithWarning(dir string) (FileConfig, bool) {
	cfg, err := LoadFromFile(dir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v (using fallback)\n", FileName, err)
		return cfg, false
	}
	return cfg, true
}

// getMergedBool implements: git config > TOML > default
func getMergedBool(worktreeDir, gitKey string, tomlExtract func(FileConfig) *bool, defaultValue bool) bool {
	if value := getGitConfigInDir(gitKey, worktreeDir); value != "" {
		return isTruthy(value)
	}
	if cfg, ok := loadConfigWithWarning(worktreeDir); ok {
		if v := tomlExtract(cfg); v != nil {
			return *v
		}
	}
	return defaultValue
}

// getMergedString implements: git config > TOML > default
func getMergedString(worktreeDir, gitKey string, tomlExtract func(FileConfig) string, defaultValue string) string {
	if value := getGitConfigInDir(gitKey, worktreeDir); value != "" {
		return value
	}
	if cfg, ok := loadConfigWithWarning(worktreeDir); ok {
		if v := tomlExtract(cfg); v != "" {
			return v
		}
	}
	return defaultValue
}

// getMergedPatterns implements: TOML > git config > default
func getMergedPatterns(worktreeDir, gitKey string, tomlExtract func(FileConfig) []string, defaultValue []string) []string {
	if cfg, ok := loadConfigWithWarning(worktreeDir); ok {
		if patterns := tomlExtract(cfg); len(patterns) > 0 {
			return patterns
		}
	}
	if patterns := getGitConfigsInDir(gitKey, worktreeDir); len(patterns) > 0 {
		return patterns
	}
	return defaultValue
}

// GetMergedPreservePatterns: TOML > git config > defaults
func GetMergedPreservePatterns(worktreeDir string) []string {
	return getMergedPatterns(worktreeDir, "grove.preserve",
		func(cfg FileConfig) []string { return cfg.Preserve.Patterns },
		DefaultConfig.PreservePatterns)
}

// GetMergedAutoLockPatterns: TOML > git config > defaults
func GetMergedAutoLockPatterns(worktreeDir string) []string {
	return getMergedPatterns(worktreeDir, "grove.autoLock",
		func(cfg FileConfig) []string { return cfg.Autolock.Patterns },
		DefaultConfig.AutoLockPatterns)
}

// GetMergedPlain: git config > TOML > default
func GetMergedPlain(worktreeDir string) bool {
	return getMergedBool(worktreeDir, "grove.plain",
		func(cfg FileConfig) *bool {
			if cfg.Plain {
				return &cfg.Plain
			}
			return nil
		},
		DefaultConfig.Plain)
}

// GetMergedDebug: git config > TOML > default
func GetMergedDebug(worktreeDir string) bool {
	return getMergedBool(worktreeDir, "grove.debug",
		func(cfg FileConfig) *bool {
			if cfg.Debug {
				return &cfg.Debug
			}
			return nil
		},
		DefaultConfig.Debug)
}

// GetMergedNerdFonts: git config > TOML > default
func GetMergedNerdFonts(worktreeDir string) bool {
	return getMergedBool(worktreeDir, "grove.nerdFonts",
		func(cfg FileConfig) *bool { return cfg.NerdFonts },
		DefaultConfig.NerdFonts)
}

// GetMergedStaleThreshold: git config > TOML > default
func GetMergedStaleThreshold(worktreeDir string) string {
	return getMergedString(worktreeDir, "grove.staleThreshold",
		func(cfg FileConfig) string { return cfg.StaleThreshold },
		DefaultConfig.StaleThreshold)
}

// WriteToFile uses atomic write (temp file + rename) to prevent corruption.
func WriteToFile(dir string, cfg *FileConfig) error {
	path := filepath.Join(dir, FileName)
	tmpPath := fmt.Sprintf("%s.tmp.%d.%d", path, os.Getpid(), time.Now().UnixNano())

	f, err := os.Create(tmpPath) //nolint:gosec // Path is derived from user-provided dir, not external input
	if err != nil {
		return err
	}

	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		_ = f.Close()
		_ = os.Remove(tmpPath)
		return err
	}

	if err := f.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}

	return os.Rename(tmpPath, path)
}

// WriteTemplateToFile writes the default config template with comments.
func WriteTemplateToFile(dir string) error {
	path := filepath.Join(dir, FileName)
	return os.WriteFile(path, []byte(initTemplate), 0o644) //nolint:gosec // Fixed permissions for config file
}
