package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Global holds the global configuration state for Grove
var Global struct {
	Plain            bool     // Disable colors and symbols
	Debug            bool     // Enable debug logging
	PreservePatterns []string // Patterns for ignored files to preserve in new worktrees
	StaleThreshold   string   // Default threshold for stale worktree detection (e.g., "30d")
	AutoLockPatterns []string // Patterns for branches to auto-lock when creating worktrees
}

// DefaultConfig contains the default configuration values
var DefaultConfig = struct {
	Plain            bool
	Debug            bool
	PreservePatterns []string
	StaleThreshold   string
	AutoLockPatterns []string
}{
	Plain:          false,
	Debug:          false,
	StaleThreshold: "30d",
	PreservePatterns: []string{
		".env",
		".env.keys",
		".env.local",
		".env.*.local",
		".envrc",
		"docker-compose.override.yml",
		"*.local.json",
		"*.local.toml",
		"*.local.yaml",
		"*.local.yml",
	},
	AutoLockPatterns: []string{
		"develop",
		"main",
		"master",
	},
}

// IsPlain returns true if plain output mode is enabled
func IsPlain() bool {
	return Global.Plain
}

// IsDebug returns true if debug logging is enabled
func IsDebug() bool {
	return Global.Debug
}

// GetPreservePatterns returns the configured preserve patterns or defaults
func GetPreservePatterns() []string {
	if len(Global.PreservePatterns) > 0 {
		return Global.PreservePatterns
	}
	return DefaultConfig.PreservePatterns
}

// GetStaleThreshold returns the configured stale threshold or default
func GetStaleThreshold() string {
	if Global.StaleThreshold != "" {
		return Global.StaleThreshold
	}
	return DefaultConfig.StaleThreshold
}

// GetAutoLockPatterns returns the configured auto-lock patterns or defaults
func GetAutoLockPatterns() []string {
	if len(Global.AutoLockPatterns) > 0 {
		return Global.AutoLockPatterns
	}
	return DefaultConfig.AutoLockPatterns
}

// ShouldAutoLock checks if a branch name matches any auto-lock pattern
func ShouldAutoLock(branch string) bool {
	patterns := GetAutoLockPatterns()
	for _, pattern := range patterns {
		// Exact match
		if pattern == branch {
			return true
		}
		// Glob pattern match (e.g., "release/*")
		if matchGlobPattern(pattern, branch) {
			return true
		}
	}
	return false
}

// matchGlobPattern performs simple glob matching supporting * wildcard
func matchGlobPattern(pattern, name string) bool {
	// Simple pattern matching: only support trailing /* for now
	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return strings.HasPrefix(name, prefix+"/")
	}
	// Exact match fallback
	return pattern == name
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() {
	plain := os.Getenv("GROVE_PLAIN")
	if plain != "" {
		Global.Plain = isTruthy(plain)
	}
	debug := os.Getenv("GROVE_DEBUG")
	if debug != "" {
		Global.Debug = isTruthy(debug)
	}
}

// LoadFromGitConfig loads configuration from git config, merging with defaults
func LoadFromGitConfig() {
	Global.Plain = DefaultConfig.Plain
	Global.Debug = DefaultConfig.Debug
	Global.StaleThreshold = DefaultConfig.StaleThreshold
	Global.PreservePatterns = make([]string, len(DefaultConfig.PreservePatterns))
	copy(Global.PreservePatterns, DefaultConfig.PreservePatterns)

	if value := getGitConfig("grove.plain"); value != "" {
		Global.Plain = isTruthy(value)
	}

	if value := getGitConfig("grove.debug"); value != "" {
		Global.Debug = isTruthy(value)
	}

	if value := getGitConfig("grove.staleThreshold"); value != "" {
		Global.StaleThreshold = value
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
	cmd := exec.Command("git", "config", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			// Exit code 1 = key not found (expected)
			if exitErr.ExitCode() == 1 {
				return ""
			}
		}
		// Real error (git not found, permission denied, etc.)
		// Log directly to stderr to avoid import cycle with logger package
		if Global.Debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] git config error for %s: %v\n", key, err)
		}
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getGitConfigs gets all values for a multi-value config key
func getGitConfigs(key string) []string {
	cmd := exec.Command("git", "config", "--get-all", key)
	output, err := cmd.Output()
	if err != nil {
		exitErr := &exec.ExitError{}
		if errors.As(err, &exitErr) {
			// Exit code 1 = key not found (expected)
			if exitErr.ExitCode() == 1 {
				return nil
			}
		}
		// Real error (git not found, permission denied, etc.)
		// Log directly to stderr to avoid import cycle with logger package
		if Global.Debug {
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
