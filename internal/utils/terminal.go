package utils

import (
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

// DefaultTerminalWidth is the fallback terminal width when detection fails.
const DefaultTerminalWidth = 80

// Falls back to DefaultTerminalWidth if detection fails or if not running in a terminal.
func GetTerminalWidth() int {
	// Try to get width from COLUMNS environment variable first (useful for testing).
	if cols := os.Getenv("COLUMNS"); cols != "" {
		if width, err := strconv.Atoi(cols); err == nil && width > 0 {
			return width
		}
	}

	if width := getTerminalWidthUnix(); width > 0 {
		return width
	}

	return DefaultTerminalWidth
}

// getTerminalWidthUnix attempts to get terminal width using unix system calls.
func getTerminalWidthUnix() int {
	// Try stdout first, then stderr, then stdin.
	fds := []int{int(os.Stdout.Fd()), int(os.Stderr.Fd()), int(os.Stdin.Fd())}

	for _, fd := range fds {
		if winsize, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ); err == nil {
			if winsize.Col > 0 {
				return int(winsize.Col)
			}
		}
	}

	return 0
}

// IsInteractiveTerminal reports whether we're running in an interactive terminal.
// This is useful for deciding whether to apply truncation or not.
func IsInteractiveTerminal() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return (fi.Mode() & os.ModeCharDevice) != 0
}
