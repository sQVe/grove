package config

import (
	"os"
	"os/exec"
	"strings"
)

// Global holds the global configuration state for Grove
var Global struct {
	Plain            bool     // Disable colors and symbols
	Debug            bool     // Enable debug logging
	PreservePatterns []string // Patterns for ignored files to preserve in new worktrees
}

// DefaultConfig contains the default configuration values
var DefaultConfig = struct {
	Plain            bool
	Debug            bool
	PreservePatterns []string
}{
	Plain: false,
	Debug: false,
	PreservePatterns: []string{
		".env",
		".env.local",
		".env.development.local",
		".env.test.local",
		".env.production.local",
		"*.local.json",
		"*.local.yaml",
		"*.local.yml",
		"*.local.toml",
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
	Global.PreservePatterns = make([]string, len(DefaultConfig.PreservePatterns))
	copy(Global.PreservePatterns, DefaultConfig.PreservePatterns)

	if value := getGitConfig("grove.plain"); value != "" {
		Global.Plain = isTruthy(value)
	}

	if value := getGitConfig("grove.debug"); value != "" {
		Global.Debug = isTruthy(value)
	}

	patterns := getGitConfigs("grove.preserve")
	if len(patterns) > 0 {
		Global.PreservePatterns = patterns
	}
}

// getGitConfig gets a single config value, returns empty string if not found
func getGitConfig(key string) string {
	cmd := exec.Command("git", "config", "--get", key)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// getGitConfigs gets all values for a multi-value config key
func getGitConfigs(key string) []string {
	cmd := exec.Command("git", "config", "--get-all", key)
	output, err := cmd.Output()
	if err != nil {
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
