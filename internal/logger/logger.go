package logger

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/sqve/grove/internal/styles"
)

// Logger state - initialized once at startup, read from goroutines
var (
	plainMode atomic.Bool
	debugMode atomic.Bool
)

// Init initializes the logger with the given settings.
// Should be called once at startup after config is loaded.
func Init(plain, debug bool) {
	plainMode.Store(plain)
	debugMode.Store(debug)
}

// isPlain returns true if plain output mode is enabled
func isPlain() bool {
	return plainMode.Load()
}

// isDebug returns true if debug logging is enabled
func isDebug() bool {
	return debugMode.Load()
}

// Debug prints debug information when debug mode is enabled
func Debug(format string, args ...any) {
	if isDebug() {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Success prints success messages to stderr
func Success(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Fprintln(os.Stderr, message)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Success, "✓"), message)
	}
}

// Error prints error messages to stderr
func Error(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Fprintf(os.Stderr, "Error: %s\n", message)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Error, "✗"), message)
	}
}

// Info prints informational messages to stderr
func Info(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Fprintln(os.Stderr, message)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Info, "→"), message)
	}
}

// Warning prints warning messages to stderr
func Warning(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", message)
	} else {
		fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Warning, "⚠"), message)
	}
}

// Dimmed prints dimmed/secondary messages to stderr
func Dimmed(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Fprintln(os.Stderr, message)
	} else {
		fmt.Fprintln(os.Stderr, styles.Render(&styles.Dimmed, message))
	}
}

// ListItemWithNote prints a list item with an optional note in parentheses to stderr
func ListItemWithNote(main, note string) {
	if isPlain() {
		if note != "" {
			fmt.Fprintf(os.Stderr, "  - %s (%s)\n", main, note)
		} else {
			fmt.Fprintf(os.Stderr, "  - %s\n", main)
		}
	} else {
		if note != "" {
			fmt.Fprintf(os.Stderr, "  %s %s %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, main),
				styles.Render(&styles.Dimmed, "("+note+")"))
		} else {
			fmt.Fprintf(os.Stderr, "  %s %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, main))
		}
	}
}

// ListSubItem prints an indented sub-item to stderr (used for additional details under a list item)
func ListSubItem(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Fprintf(os.Stderr, "    > %s\n", message)
	} else {
		fmt.Fprintf(os.Stderr, "    %s %s\n", styles.Render(&styles.Dimmed, "↳"), message)
	}
}

// ListItemGroup prints a group of items under a header to stderr
// Used for multi-item lists like preserved files
func ListItemGroup(header string, items []string) {
	if len(items) == 0 {
		return
	}
	ListSubItem("%s", header)
	for _, item := range items {
		if isPlain() {
			fmt.Fprintf(os.Stderr, "        %s\n", item)
		} else {
			fmt.Fprintf(os.Stderr, "        %s\n", styles.Render(&styles.Dimmed, item))
		}
	}
}

// StartSpinner starts a spinner with a message and returns a stop function
// Output goes to stderr to keep stdout clean for program output
func StartSpinner(message string) func() {
	if isPlain() {
		fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Info, "→"), message)
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
				fmt.Fprint(os.Stderr, "\r\033[K") // Clear line
				return
			case <-ticker.C:
				fmt.Fprintf(os.Stderr, "\r%s %s",
					styles.Render(&styles.Info, frames[i]),
					message)
				i = (i + 1) % len(frames)
			}
		}
	}()

	return func() {
		once.Do(func() {
			close(done)
			time.Sleep(10 * time.Millisecond) // Let spinner goroutine clear the line before exit
		})
	}
}
