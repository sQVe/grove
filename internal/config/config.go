package config

import "os"

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
	// TODO: implement git config loading
}

// GetDefaultPreserveIgnoredPatterns returns default patterns to preserve
func GetDefaultPreserveIgnoredPatterns() []string {
	return []string{
		".env",
		".env.local",
		".env.development.local",
		"*.local.json",
		"*.local.yaml",
		"*.local.yml",
		"*.local.toml",
	}
}

// GetPreserveIgnoredPatterns returns all preserve ignored patterns
func GetPreserveIgnoredPatterns() []string {
	// TODO: implement git config reading for grove.convert.preserveIgnored
	return GetDefaultPreserveIgnoredPatterns()
}
