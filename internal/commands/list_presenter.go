package commands

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/sqve/grove/internal/git"
)

// Color theme for table styling.
var (
	primaryColor = lipgloss.Color("#8B5CF6")
	successColor = lipgloss.Color("#059669")
	warningColor = lipgloss.Color("#D97706")
	mutedColor   = lipgloss.Color("#9CA3AF")
	headerColor  = lipgloss.Color("#6B7280")
)

// ListPresenter handles the display formatting and styling for worktree listings.
type ListPresenter struct {
	formatter *WorktreeFormatter
}

// NewListPresenter creates a new ListPresenter.
func NewListPresenter() *ListPresenter {
	return &ListPresenter{
		formatter: NewWorktreeFormatter(),
	}
}

// DisplayHuman displays worktrees using lipgloss table component with rich styling.
func (p *ListPresenter) DisplayHuman(worktrees []git.WorktreeInfo, verbose bool) error {
	if len(worktrees) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(mutedColor).Italic(true)
		fmt.Println(emptyStyle.Render("No worktrees match the specified criteria"))
		return nil
	}

	// Build table data
	var rows [][]string

	// Create header row with leading spaces for alignment
	headers := []string{"", " WORKTREE", " BRANCH", " STATUS", " ACTIVITY"}
	if verbose {
		headers = append(headers, " PATH")
	}

	for i := range worktrees {
		wt := &worktrees[i]
		row := p.buildTableRow(wt, verbose)
		rows = append(rows, row)
	}

	// Create and display lipgloss table
	p.displayTable(headers, rows)
	return nil
}

// DisplayPorcelain displays worktrees in machine-readable format.
func (p *ListPresenter) DisplayPorcelain(worktrees []git.WorktreeInfo) error {
	for i := range worktrees {
		wt := &worktrees[i]
		current := "false"
		if wt.IsCurrent {
			current = "true"
		}

		fmt.Printf("worktree %s\n", wt.Path)
		fmt.Printf("branch %s\n", wt.Branch)
		fmt.Printf("head %s\n", wt.Head)
		fmt.Printf("current %s\n", current)

		if !wt.LastActivity.IsZero() {
			fmt.Printf("activity %d\n", wt.LastActivity.Unix())
		}

		fmt.Printf("status %d %d %d %t\n",
			wt.Status.Modified, wt.Status.Staged, wt.Status.Untracked, wt.Status.IsClean)

		if wt.Remote.HasRemote {
			fmt.Printf("remote %d %d %t\n", wt.Remote.Ahead, wt.Remote.Behind, wt.Remote.IsMerged)
		}

		fmt.Println() // Empty line to separate entries
	}

	return nil
}

// buildTableRow creates a styled table row for the given worktree.
func (p *ListPresenter) buildTableRow(wt *git.WorktreeInfo, verbose bool) []string {
	// Marker
	marker := " "
	if wt.IsCurrent {
		marker = lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("*")
	}

	// Worktree name
	name := p.formatter.GetWorktreeName(wt.Path)
	if wt.IsCurrent {
		name = lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(name)
	}

	// Branch name
	branch := wt.Branch
	if wt.IsCurrent {
		branch = lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(branch)
	}

	// Status with colors and symbols
	status := p.buildStyledStatus(wt.Status, wt.Remote)

	// Activity
	activity := p.formatter.FormatActivity(wt.LastActivity)

	// Build row
	row := []string{marker, name, branch, status, activity}
	if verbose {
		row = append(row, wt.Path)
	}

	return row
}

// buildStyledStatus creates a styled status string with colors and symbols.
func (p *ListPresenter) buildStyledStatus(status git.WorktreeStatus, remote git.RemoteStatus) string {
	statusInfo := p.formatter.FormatStatus(status, remote)

	var result string
	if statusInfo.IsClean {
		result = lipgloss.NewStyle().Foreground(successColor).Bold(true).Render(statusInfo.Symbol)
	} else {
		warning := lipgloss.NewStyle().Foreground(warningColor).Bold(true).Render(statusInfo.Symbol)
		result = warning + " " + statusInfo.CountsText
	}

	// Add remote status if available
	if statusInfo.RemoteText != "" {
		remoteStyled := lipgloss.NewStyle().Foreground(mutedColor).Render(" " + statusInfo.RemoteText)
		result += remoteStyled
	}

	return result
}

// displayTable creates and prints the lipgloss table.
func (p *ListPresenter) displayTable(headers []string, rows [][]string) {
	headerStyle := lipgloss.NewStyle().Foreground(headerColor).Bold(false)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(mutedColor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			// Add padding to all cells
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers(headers...).
		Rows(rows...)

	// Print the table
	fmt.Println(t)
}