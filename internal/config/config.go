package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"
)

// globalMu protects access to the Global struct
var globalMu sync.RWMutex

type settings struct {
	Plain            bool
	Debug            bool
	NerdFonts        bool
	StaleThreshold   string
	AutoLockPatterns []string
	Timeout          time.Duration
}

type defaultSettings struct {
	settings
	PreservePatterns        []string
	PreserveExcludePatterns []string
	PreserveDirectories     []string
	LinkPatterns            []string
}

// Global holds the global configuration state for Grove
var Global settings

// DefaultConfig contains the default configuration values
var DefaultConfig = defaultSettings{
	settings: settings{
		Plain:          false,
		Debug:          false,
		NerdFonts:      true,
		StaleThreshold: "30d",
		Timeout:        30 * time.Second,
		AutoLockPatterns: []string{
			"develop",
			"main",
			"master",
		},
	},
	PreservePatterns: []string{
		".env",
		".env.keys",
		".env.local",
		".env.*.local",
		".envrc",
		".grove.toml",
		"docker-compose.override.yml",
	},
	PreserveExcludePatterns: []string{
		".cache",
		".venv",
		"__pycache__",
		"build",
		"coverage",
		"dist",
		"node_modules",
		"out",
		"target",
		"vendor",
		"venv",
	},
	PreserveDirectories: []string{},
	LinkPatterns:        []string{},
}

// IsPlain returns true if plain output mode is enabled
func IsPlain() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return Global.Plain
}

// IsDebug returns true if debug logging is enabled
func IsDebug() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return Global.Debug
}

// IsNerdFonts returns true if Nerd Font icons should be used
func IsNerdFonts() bool {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return Global.NerdFonts
}

// SetPlain sets the plain output mode
func SetPlain(v bool) {
	globalMu.Lock()
	defer globalMu.Unlock()
	Global.Plain = v
}

// SetDebug sets the debug logging mode
func SetDebug(v bool) {
	globalMu.Lock()
	defer globalMu.Unlock()
	Global.Debug = v
}

// GetStaleThreshold returns the configured stale threshold or default
func GetStaleThreshold() string {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if Global.StaleThreshold != "" {
		return Global.StaleThreshold
	}
	return DefaultConfig.StaleThreshold
}

// ParseDuration parses human-friendly durations like "30d", "2w", and "6m".
func ParseDuration(s string) (time.Duration, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return 0, errors.New("duration cannot be empty")
	}
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	num, err := strconv.Atoi(s[:len(s)-1])
	if err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", s)
	}
	if num <= 0 {
		return 0, fmt.Errorf("duration must be positive: %s", s)
	}

	switch unit := s[len(s)-1]; unit {
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(num) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %c (use d, w, or m)", unit)
	}
}

// GetAutoLockPatterns returns the configured auto-lock patterns or defaults.
// Returns a copy to prevent callers from mutating the original slices.
func GetAutoLockPatterns() []string {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if len(Global.AutoLockPatterns) > 0 {
		return append([]string{}, Global.AutoLockPatterns...)
	}
	return append([]string{}, DefaultConfig.AutoLockPatterns...)
}

// GetTimeout returns the configured command timeout.
// Returns 0 if timeout is disabled (grove.timeout = 0).
func GetTimeout() time.Duration {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return Global.Timeout
}

// ShouldAutoLock checks if a branch name matches any auto-lock pattern.
func ShouldAutoLock(branch string) bool {
	patterns := GetAutoLockPatterns()
	for _, pattern := range patterns {
		if pattern == branch || matchGlobPattern(pattern, branch) {
			return true
		}
	}
	return false
}

func matchGlobPattern(pattern, name string) bool {
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(name, prefix+"/")
	}
	return pattern == name
}

// LoadFromGitConfig loads configuration from git config, merging with defaults
func LoadFromGitConfig() {
	loadGlobalConfig(&FileConfig{})
}

func loadGlobalConfig(fileConfig *FileConfig) {
	globalMu.Lock()
	defer globalMu.Unlock()
	Global = DefaultConfig.settings
	Global.AutoLockPatterns = slices.Clone(DefaultConfig.AutoLockPatterns)

	if fileConfig.Plain != nil {
		Global.Plain = *fileConfig.Plain
	}
	if fileConfig.Debug != nil {
		Global.Debug = *fileConfig.Debug
	}
	if fileConfig.NerdFonts != nil {
		Global.NerdFonts = *fileConfig.NerdFonts
	}
	if isValidStaleThreshold(fileConfig.StaleThreshold) {
		Global.StaleThreshold = fileConfig.StaleThreshold
	}
	if len(fileConfig.Autolock.Patterns) > 0 {
		Global.AutoLockPatterns = append([]string{}, fileConfig.Autolock.Patterns...)
	}

	if value := getGitConfig("grove.plain"); value != "" {
		Global.Plain = isTruthy(value)
	}

	if value := getGitConfig("grove.debug"); value != "" {
		Global.Debug = isTruthy(value)
	}

	if value := getGitConfig("grove.nerdFonts"); value != "" {
		Global.NerdFonts = isTruthy(value)
	}

	if value := getGitConfig("grove.staleThreshold"); value != "" {
		if isValidStaleThreshold(value) {
			Global.StaleThreshold = value
		}
		// Invalid values are silently ignored, using default
	}

	if value := getGitConfig("grove.timeout"); value != "" {
		if d, err := time.ParseDuration(value); err == nil {
			Global.Timeout = d
		}
	}

	autoLockPatterns := getGitConfigs("grove.autoLock")
	if len(autoLockPatterns) > 0 {
		Global.AutoLockPatterns = autoLockPatterns
	}
}

// getGitConfig gets a single config value, returns empty string if not found
func getGitConfig(key string) string {
	return getGitConfigInDir(key, "")
}

// getGitConfigInDir gets a single config value from a specific directory
func getGitConfigInDir(key, dir string) string {
	cmd := exec.Command("git", "config", "--get", key) //nolint:gosec
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return ""
		}
		if IsDebug() {
			fmt.Fprintf(os.Stderr, "[DEBUG] git config error for %s: %v\n", key, err)
		}
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getGitConfigs gets all values for a multi-value config key
func getGitConfigs(key string) []string {
	return getGitConfigsInDir(key, "")
}

// getGitConfigsInDir gets all values for a multi-value config key from a specific directory
func getGitConfigsInDir(key, dir string) []string {
	cmd := exec.Command("git", "config", "--get-all", key) //nolint:gosec
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return nil
		}
		if IsDebug() {
			fmt.Fprintf(os.Stderr, "[DEBUG] git config error for %s: %v\n", key, err)
		}
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var result []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

// isTruthy checks if a string represents a truthy value
func isTruthy(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "true" || lower == "1" || lower == "yes" || lower == "on"
}

// isValidStaleThreshold checks if a stale threshold value has valid format (e.g., "30d", "2w", "1m")
func isValidStaleThreshold(s string) bool {
	_, err := ParseDuration(s)
	return err == nil
}
