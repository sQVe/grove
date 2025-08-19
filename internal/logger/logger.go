package logger

import (
	"fmt"
	"os"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/styles"
)

// Debug prints debug information when debug mode is enabled
func Debug(format string, args ...any) {
	if config.IsDebug() {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Success prints success messages
func Success(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Println(message)
	} else {
		fmt.Printf("%s %s\n", styles.Render(&styles.Success, "✓"), styles.Render(&styles.Success, message))
	}
}

// Error prints error messages to stderr
func Error(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Error, "✗"), styles.Render(&styles.Error, message))
	}
}

// Info prints informational messages
func Info(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Println(message)
	} else {
		fmt.Printf("%s %s\n", styles.Render(&styles.Info, "ℹ"), styles.Render(&styles.Info, message))
	}
}

// Warning prints warning messages
func Warning(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Printf("Warning: %s\n", message)
	} else {
		fmt.Printf("%s %s\n", styles.Render(&styles.Warning, "⚠"), styles.Render(&styles.Warning, message))
	}
}

// Dimmed prints dimmed/secondary messages
func Dimmed(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Println(message)
	} else {
		fmt.Println(styles.Render(&styles.Dimmed, message))
	}
}
