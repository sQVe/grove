package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

const FileName = ".grove.toml"

type FileConfig struct {
	Preserve struct {
		Patterns []string `toml:"patterns"`
	} `toml:"preserve"`
	Hooks struct {
		Create []string `toml:"create"`
	} `toml:"hooks"`
	Plain bool `toml:"plain"`
	Debug bool `toml:"debug"`
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

// WriteToFile uses atomic write (temp file + rename) to prevent corruption.
func WriteToFile(dir string, cfg FileConfig) error {
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
