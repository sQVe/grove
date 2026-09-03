package styles

import (
	"os"
	"sync"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/sqve/grove/internal/config"
)

var (
	Dimmed   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // gray
	Error    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	Info     = lipgloss.NewStyle().Foreground(lipgloss.Color("4")) // blue
	Path     = lipgloss.NewStyle().Foreground(lipgloss.Color("6")) // cyan
	Success  = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	Warning  = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	Worktree = lipgloss.NewStyle().Foreground(lipgloss.Color("5")) // magenta

	setTestColors = sync.OnceFunc(func() {
		if os.Getenv("GROVE_TEST_COLORS") == "true" {
			lipgloss.SetColorProfile(termenv.ANSI256)
		}
	})
)

// PrettyPath replaces $HOME with ~ for cleaner output (no styling).
func PrettyPath(path string) string {
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		if len(path) >= len(home) && path[:len(home)] == home {
			path = "~" + path[len(home):]
		}
	}

	return path
}

// RenderPath renders a path with styling, replacing $HOME with ~ for cleaner output.
func RenderPath(path string) string {
	return Render(&Path, PrettyPath(path))
}

func Render(style *lipgloss.Style, text string) string {
	if config.IsPlain() {
		return text
	}

	setTestColors()
	return style.Render(text)
}
