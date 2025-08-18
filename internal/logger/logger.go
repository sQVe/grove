package logger

import (
	"fmt"
	"os"

	"github.com/sqve/grove/internal/config"
)

// Debug prints debug information when debug mode is enabled
func Debug(format string, args ...interface{}) {
	if config.IsDebug() {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Success prints success messages
func Success(format string, args ...interface{}) {
	if config.IsPlain() {
		fmt.Printf(format+"\n", args...)
	} else {
		fmt.Printf("✓ "+format+"\n", args...)
	}
}
