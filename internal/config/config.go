package config

import "os"

// Global holds the global configuration state for Grove CLI
var Global struct {
	Plain bool // Disable colors, emojis, and formatting
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
	if os.Getenv("GROVE_PLAIN") == "1" {
		Global.Plain = true
	}
	if os.Getenv("GROVE_DEBUG") == "1" {
		Global.Debug = true
	}
}
