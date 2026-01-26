package logger

import (
	"fmt"
	"io"
	"os"
	"sync/atomic"

	"github.com/sqve/grove/internal/styles"
)

// Logger state - initialized once at startup, read from goroutines
var (
	plainMode atomic.Bool
	debugMode atomic.Bool
	output    atomic.Pointer[io.Writer]
)

// Init initializes the logger with the given settings.
// Should be called once at startup after config is loaded.
func Init(plain, debug bool) {
	plainMode.Store(plain)
	debugMode.Store(debug)
}

// SetOutput sets the output writer for logging.
// Pass nil to reset to os.Stderr. Returns the previous writer.
func SetOutput(w io.Writer) io.Writer {
	prev := output.Load()
	if w == nil {
		output.Store(nil)
	} else {
		output.Store(&w)
	}
	if prev == nil {
		return os.Stderr
	}
	return *prev
}

// getOutput returns the current output writer
func getOutput() io.Writer {
	if w := output.Load(); w != nil {
		return *w
	}
	return os.Stderr
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
		_, _ = fmt.Fprintf(getOutput(), "[DEBUG] "+format+"\n", args...)
	}
}

// Success prints success messages to stderr
func Success(format string, args ...any) {
	out := getOutput()
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		_, _ = fmt.Fprintln(out, message)
	} else {
		_, _ = fmt.Fprintf(out, "%s %s\n", styles.Render(&styles.Success, "✓"), message)
	}
}

// Error prints error messages to stderr
func Error(format string, args ...any) {
	out := getOutput()
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		_, _ = fmt.Fprintf(out, "Error: %s\n", message)
	} else {
		_, _ = fmt.Fprintf(out, "%s %s\n", styles.Render(&styles.Error, "✗"), message)
	}
}

// Info prints informational messages to stderr
func Info(format string, args ...any) {
	out := getOutput()
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		_, _ = fmt.Fprintln(out, message)
	} else {
		_, _ = fmt.Fprintf(out, "%s %s\n", styles.Render(&styles.Info, "→"), message)
	}
}

// Warning prints warning messages to stderr
func Warning(format string, args ...any) {
	out := getOutput()
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		_, _ = fmt.Fprintf(out, "Warning: %s\n", message)
	} else {
		_, _ = fmt.Fprintf(out, "%s %s\n", styles.Render(&styles.Warning, "⚠"), message)
	}
}

// Dimmed prints dimmed/secondary messages to stderr
func Dimmed(format string, args ...any) {
	out := getOutput()
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		_, _ = fmt.Fprintln(out, message)
	} else {
		_, _ = fmt.Fprintln(out, styles.Render(&styles.Dimmed, message))
	}
}

// ListItemWithNote prints a list item with an optional note in parentheses to stderr
func ListItemWithNote(main, note string) {
	out := getOutput()
	if isPlain() {
		if note != "" {
			_, _ = fmt.Fprintf(out, "  - %s (%s)\n", main, note)
		} else {
			_, _ = fmt.Fprintf(out, "  - %s\n", main)
		}
	} else {
		if note != "" {
			_, _ = fmt.Fprintf(out, "  %s %s %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, main),
				styles.Render(&styles.Dimmed, "("+note+")"))
		} else {
			_, _ = fmt.Fprintf(out, "  %s %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, main))
		}
	}
}

// ListSubItem prints an indented sub-item to stderr (used for additional details under a list item)
func ListSubItem(format string, args ...any) {
	out := getOutput()
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		_, _ = fmt.Fprintf(out, "    > %s\n", message)
	} else {
		_, _ = fmt.Fprintf(out, "    %s %s\n", styles.Render(&styles.Dimmed, "↳"), message)
	}
}

// ListItemGroup prints a group of items under a header to stderr
// Used for multi-item lists like preserved files
func ListItemGroup(header string, items []string) {
	if len(items) == 0 {
		return
	}
	out := getOutput()
	ListSubItem("%s", header)
	for _, item := range items {
		if isPlain() {
			_, _ = fmt.Fprintf(out, "        %s\n", item)
		} else {
			_, _ = fmt.Fprintf(out, "        %s\n", styles.Render(&styles.Dimmed, item))
		}
	}
}
