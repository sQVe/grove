package logger

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/sqve/grove/internal/styles"
)

// Logger state - initialized once at startup
var state struct {
	plain bool
	debug bool
}

// Init initializes the logger with the given settings.
// Should be called once at startup after config is loaded.
func Init(plain, debug bool) {
	state.plain = plain
	state.debug = debug
}

// isPlain returns true if plain output mode is enabled
func isPlain() bool {
	return state.plain
}

// isDebug returns true if debug logging is enabled
func isDebug() bool {
	return state.debug
}

// Debug prints debug information when debug mode is enabled
func Debug(format string, args ...any) {
	if isDebug() {
		fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
	}
}

// Success prints success messages
func Success(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Println(message)
	} else {
		fmt.Printf("%s %s\n", styles.Render(&styles.Success, "✓"), message)
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

// Info prints informational messages
func Info(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Println(message)
	} else {
		fmt.Printf("%s %s\n", styles.Render(&styles.Info, "→"), message)
	}
}

// Warning prints warning messages
func Warning(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Printf("Warning: %s\n", message)
	} else {
		fmt.Printf("%s %s\n", styles.Render(&styles.Warning, "⚠"), message)
	}
}

// Dimmed prints dimmed/secondary messages
func Dimmed(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Println(message)
	} else {
		fmt.Println(styles.Render(&styles.Dimmed, message))
	}
}

// ListItem prints a list item (used for worktree creation output, etc.)
func ListItem(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	if isPlain() {
		fmt.Printf("  - %s\n", message)
	} else {
		fmt.Printf("  %s %s\n", styles.Render(&styles.Success, "✓"), message)
	}
}

// ListItemWithNote prints a list item with an optional note in parentheses
func ListItemWithNote(main, note string) {
	if isPlain() {
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
	if isPlain() {
		fmt.Printf("    > %s\n", message)
	} else {
		fmt.Printf("    %s %s\n", styles.Render(&styles.Dimmed, "↳"), message)
	}
}

// ListItemGroup prints a group of items under a header
// Used for multi-item lists like preserved files
func ListItemGroup(header string, items []string) {
	if len(items) == 0 {
		return
	}
	ListSubItem("%s", header)
	for _, item := range items {
		if isPlain() {
			fmt.Printf("        %s\n", item)
		} else {
			fmt.Printf("        %s\n", styles.Render(&styles.Dimmed, item))
		}
	}
}

// WorktreeListItem prints a worktree entry in list format
func WorktreeListItem(name string, current bool, status, syncStatus string, nameWidth int, lockIndicator string) {
	marker := " "
	if current {
		marker = "*"
	}

	namePadded := fmt.Sprintf("%-*s", nameWidth, name)

	if isPlain() {
		if lockIndicator != "" {
			fmt.Printf("%s %s %s %s %s\n", marker, namePadded, status, lockIndicator, syncStatus)
		} else {
			fmt.Printf("%s %s %s %s\n", marker, namePadded, status, syncStatus)
		}
	} else {
		markerStyled := " "
		if current {
			markerStyled = styles.Render(&styles.Success, "●")
		}
		nameStyled := styles.Render(&styles.Worktree, name) + namePadded[len(name):]
		var statusStyled string
		if status == "[dirty]" {
			statusStyled = styles.Render(&styles.Warning, status)
		} else {
			statusStyled = styles.Render(&styles.Dimmed, status)
		}
		if lockIndicator != "" {
			fmt.Printf("%s %s %s %s %s\n", markerStyled, nameStyled, statusStyled, lockIndicator, syncStatus)
		} else {
			fmt.Printf("%s %s %s %s\n", markerStyled, nameStyled, statusStyled, syncStatus)
		}
	}
}

// StartSpinner starts a spinner with a message and returns a stop function
func StartSpinner(message string) func() {
	if isPlain() {
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
			time.Sleep(10 * time.Millisecond) // Let spinner goroutine clear the line before exit
		})
	}
}
