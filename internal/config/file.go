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

// GetMergedPreservePatterns: TOML > git config > defaults
func GetMergedPreservePatterns(worktreeDir string) []string {
	cfg, err := LoadFromFile(worktreeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v (using fallback)\n", FileName, err)
	} else if len(cfg.Preserve.Patterns) > 0 {
		return cfg.Preserve.Patterns
	}

	patterns := getGitConfigs("grove.preserve")
	if len(patterns) > 0 {
		return patterns
	}

	return DefaultConfig.PreservePatterns
}

// GetMergedPlain: git config > TOML > default
func GetMergedPlain(worktreeDir string) bool {
	if value := getGitConfig("grove.plain"); value != "" {
		return isTruthy(value)
	}

	cfg, err := LoadFromFile(worktreeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v (using fallback)\n", FileName, err)
	} else if cfg.Plain {
		return true
	}

	return DefaultConfig.Plain
}

// GetMergedDebug: git config > TOML > default
func GetMergedDebug(worktreeDir string) bool {
	if value := getGitConfig("grove.debug"); value != "" {
		return isTruthy(value)
	}

	cfg, err := LoadFromFile(worktreeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v (using fallback)\n", FileName, err)
	} else if cfg.Debug {
		return true
	}

	return DefaultConfig.Debug
}

// GetMergedNerdFonts: git config > TOML > default
func GetMergedNerdFonts(worktreeDir string) bool {
	if value := getGitConfig("grove.nerdFonts"); value != "" {
		return isTruthy(value)
	}

	cfg, err := LoadFromFile(worktreeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v (using fallback)\n", FileName, err)
	} else if cfg.NerdFonts != nil {
		return *cfg.NerdFonts
	}

	return DefaultConfig.NerdFonts
}

// GetMergedStaleThreshold: git config > TOML > default
func GetMergedStaleThreshold(worktreeDir string) string {
	if value := getGitConfig("grove.staleThreshold"); value != "" {
		return value
	}

	cfg, err := LoadFromFile(worktreeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v (using fallback)\n", FileName, err)
	} else if cfg.StaleThreshold != "" {
		return cfg.StaleThreshold
	}

	return DefaultConfig.StaleThreshold
}

// GetMergedAutoLockPatterns: TOML > git config > defaults
func GetMergedAutoLockPatterns(worktreeDir string) []string {
	cfg, err := LoadFromFile(worktreeDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to parse %s: %v (using fallback)\n", FileName, err)
	} else if len(cfg.Autolock.Patterns) > 0 {
		return cfg.Autolock.Patterns
	}

	patterns := getGitConfigs("grove.autoLock")
	if len(patterns) > 0 {
		return patterns
	}

	return DefaultConfig.AutoLockPatterns
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
