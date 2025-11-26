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

// ListItem prints a list item (used for worktree creation output, etc.)
func ListItem(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Printf("  - %s\n", message)
	} else {
		fmt.Printf("  %s %s\n", styles.Render(&styles.Success, "✓"), message)
	}
}

// ListItemWithNote prints a list item with an optional note in parentheses
func ListItemWithNote(main, note string) {
	if config.IsPlain() {
		if note != "" {
			fmt.Printf("  - %s (%s)\n", main, note)
		} else {
			fmt.Printf("  - %s\n", main)
		}
	} else {
		if note != "" {
			fmt.Printf("  %s %s %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, main),
				styles.Render(&styles.Dimmed, "("+note+")"))
		} else {
			fmt.Printf("  %s %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, main))
		}
	}
}

// ListSubItem prints an indented sub-item (used for additional details under a list item)
func ListSubItem(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if config.IsPlain() {
		fmt.Printf("      %s\n", message)
	} else {
		fmt.Printf("    %s\n", styles.Render(&styles.Dimmed, "↳ "+message))
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
