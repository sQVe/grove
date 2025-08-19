package styles

import (
	"os"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/sqve/grove/internal/config"
)

var (
	Success = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	Error   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	Info    = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	Dimmed  = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
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
