package config

import (
	"os"
	"os/exec"
	"strings"
)

// Global holds the global configuration state for Grove
var Global struct {
	Plain bool // Disable colors and symbols
	Debug bool // Enable debug logging
}

// IsPlain returns true if plain output mode is enabled
func IsPlain() bool {
	return Global.Plain
}

// IsDebug returns true if debug logging is enabled
func IsDebug() bool {
	return Global.Debug
}

// LoadFromEnv loads configuration from environment variables
func LoadFromEnv() {
	plain := os.Getenv("GROVE_PLAIN")
	if plain == "1" || plain == "true" {
		Global.Plain = true
	}
	debug := os.Getenv("GROVE_DEBUG")
	if debug == "1" || debug == "true" {
		Global.Debug = true
	}
}

// LoadFromGitConfig loads configuration from git config
func LoadFromGitConfig() {
	if value := getGitConfig("grove.plain"); value != "" {
		if isTruthy(value) {
			Global.Plain = true
		}
	}

	if value := getGitConfig("grove.debug"); value != "" {
		if isTruthy(value) {
			Global.Debug = true
		}
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

// isTruthy checks if a string represents a truthy value
func isTruthy(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	return lower == "true" || lower == "yes" || lower == "on" || lower == "1"
}
