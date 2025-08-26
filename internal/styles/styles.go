package styles

import (
	"os"

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
)

func Render(style *lipgloss.Style, text string) string {
	if config.IsPlain() {
		return text
	}

	// Allow us to force enable colors in certain tests, since lipgloss disables
	// colors in test enviroments.
	if os.Getenv("GROVE_TEST_COLORS") == "true" {
		lipgloss.SetColorProfile(termenv.ANSI256)
	}

	return style.Render(text)
}
