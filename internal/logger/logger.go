package logger

import (
	"fmt"
	"os"

	"github.com/sqve/grove/internal/config"
)

// Debug prints debug information when debug mode is enabled
func Debug(format string, args ...any) {
	if config.IsDebug() {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Success prints success messages
func Success(format string, args ...any) {
	if config.IsPlain() {
		fmt.Printf(format+"\n", args...)
	} else {
		fmt.Printf("✓ "+format+"\n", args...)
	}
}

// Error prints error messages to stderr
func Error(format string, args ...any) {
	if config.IsPlain() {
		fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	} else {
		fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
	}
}
