package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

// globalMu protects access to the Global struct
var globalMu sync.RWMutex

// Global holds the global configuration state for Grove
var Global struct {
	Plain            bool          // Disable colors and symbols
	Debug            bool          // Enable debug logging
	NerdFonts        bool          // Use Nerd Font icons (when not in plain mode)
	PreservePatterns []string      // Patterns for ignored files to preserve in new worktrees
	StaleThreshold   string        // Default threshold for stale worktree detection (e.g., "30d")
	AutoLockPatterns []string      // Patterns for branches to auto-lock when creating worktrees
	Timeout          time.Duration // Command timeout (0 = no timeout)
}

// DefaultConfig contains the default configuration values
var DefaultConfig = struct {
	Plain            bool
	Debug            bool
	NerdFonts        bool
	PreservePatterns []string
	StaleThreshold   string
	AutoLockPatterns []string
	Timeout          time.Duration
}{
	Plain:          false,
	Debug:          false,
	NerdFonts:      true,
	StaleThreshold: "30d",
	Timeout:        30 * time.Second,
	PreservePatterns: []string{
		".env",
		".env.keys",
		".env.local",
		".env.*.local",
		".envrc",
		".grove.toml",
		"*.local.json",
		"*.local.toml",
		"*.local.yaml",
		"*.local.yml",
		"docker-compose.override.yml",
	},
	AutoLockPatterns: []string{
		"develop",
		"main",
		"master",
	},
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

// GetPreservePatterns returns the configured preserve patterns or defaults
func GetPreservePatterns() []string {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if len(Global.PreservePatterns) > 0 {
		return Global.PreservePatterns
	}
	return DefaultConfig.PreservePatterns
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

// GetAutoLockPatterns returns the configured auto-lock patterns or defaults
func GetAutoLockPatterns() []string {
	globalMu.RLock()
	defer globalMu.RUnlock()
	if len(Global.AutoLockPatterns) > 0 {
		return Global.AutoLockPatterns
	}
	return DefaultConfig.AutoLockPatterns
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
	globalMu.Lock()
	defer globalMu.Unlock()
	Global.Plain = DefaultConfig.Plain
	Global.Debug = DefaultConfig.Debug
	Global.NerdFonts = DefaultConfig.NerdFonts
	Global.StaleThreshold = DefaultConfig.StaleThreshold
	Global.Timeout = DefaultConfig.Timeout
	Global.PreservePatterns = make([]string, len(DefaultConfig.PreservePatterns))
	copy(Global.PreservePatterns, DefaultConfig.PreservePatterns)

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

	patterns := getGitConfigs("grove.preserve")
	if len(patterns) > 0 {
		Global.PreservePatterns = patterns
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
	cmd := exec.Command("git", "config", "--get", key)
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
	cmd := exec.Command("git", "config", "--get-all", key)
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
	s = strings.TrimSpace(strings.ToLower(s))
	if len(s) < 2 {
		return false
	}
	unit := s[len(s)-1]
	if unit != 'd' && unit != 'w' && unit != 'm' {
		return false
	}
	num, err := strconv.Atoi(s[:len(s)-1])
	return err == nil && num > 0
}

// Snapshot captures the current Global config state.
// Used for testing to save/restore config.
type Snapshot struct {
	Plain            bool
	Debug            bool
	NerdFonts        bool
	PreservePatterns []string
	StaleThreshold   string
	AutoLockPatterns []string
	Timeout          time.Duration
}

// SaveSnapshot returns a copy of the current Global config.
func SaveSnapshot() Snapshot {
	globalMu.RLock()
	defer globalMu.RUnlock()
	return Snapshot{
		Plain:            Global.Plain,
		Debug:            Global.Debug,
		NerdFonts:        Global.NerdFonts,
		PreservePatterns: append([]string{}, Global.PreservePatterns...),
		StaleThreshold:   Global.StaleThreshold,
		AutoLockPatterns: append([]string{}, Global.AutoLockPatterns...),
		Timeout:          Global.Timeout,
	}
}

// RestoreSnapshot restores Global config from a snapshot.
func RestoreSnapshot(s *Snapshot) {
	globalMu.Lock()
	defer globalMu.Unlock()
	Global.Plain = s.Plain
	Global.Debug = s.Debug
	Global.NerdFonts = s.NerdFonts
	Global.PreservePatterns = append([]string{}, s.PreservePatterns...)
	Global.StaleThreshold = s.StaleThreshold
	Global.AutoLockPatterns = append([]string{}, s.AutoLockPatterns...)
	Global.Timeout = s.Timeout
}
