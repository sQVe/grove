package logger

import (
	"fmt"
	"os"
	"sync"
	"time"

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
		fmt.Printf("%s %s\n", styles.Render(&styles.Success, "✓"), message)
	}
}

// Error prints error messages to stderr
func Error(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Error, "✗"), message)
	}
}

// Info prints informational messages
func Info(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Println(message)
	} else {
		fmt.Printf("%s %s\n", styles.Render(&styles.Info, "→"), message)
	}
}

// Warning prints warning messages
func Warning(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Printf("Warning: %s\n", message)
	} else {
		fmt.Printf("%s %s\n", styles.Render(&styles.Warning, "⚠"), message)
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

// StartSpinner starts a spinner with a message and returns a stop function
func StartSpinner(message string) func() {
	if config.IsPlain() {
		fmt.Printf("%s %s\n", styles.Render(&styles.Info, "→"), message)
		return func() {}
	}

	done := make(chan bool)
	var once sync.Once

	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		ticker := time.NewTicker(80 * time.Millisecond)
		defer ticker.Stop()

		i := 0
		for {
			select {
			case <-done:
				fmt.Print("\r\033[K") // Clear line
				return
			case <-ticker.C:
				fmt.Printf("\r%s %s",
					styles.Render(&styles.Info, frames[i]),
					message)
				i = (i + 1) % len(frames)
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(done)
			time.Sleep(10 * time.Millisecond) // Give goroutine time to clear the line
		})
	}
}
