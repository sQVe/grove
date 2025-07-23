package commands

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/utils"
)

// Color theme for table styling (using constants from list_constants.go).

// ColumnWidths contains the calculated maximum widths for table columns.
type ColumnWidths struct {
	Worktree int
	Branch   int
	Status   int
	Activity int
	Path     int // Only used in verbose mode
}

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
		emptyStyle := lipgloss.NewStyle().Foreground(MutedColor).Italic(true)
		fmt.Println(emptyStyle.Render("No worktrees match the specified criteria"))
		return nil
	}

	// Calculate responsive column widths
	colWidths := p.calculateColumnWidths(worktrees, verbose)

	// Build table data
	var rows [][]string

	// Create header row with leading spaces for alignment
	headers := []string{"", " WORKTREE", " BRANCH", " STATUS", " ACTIVITY"}
	if verbose {
		headers = append(headers, " PATH")
	}

	for i := range worktrees {
		wt := &worktrees[i]
		row := p.buildTableRow(wt, verbose, colWidths)
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
		fmt.Printf("branch %s\n", git.CleanBranchName(wt.Branch))
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
func (p *ListPresenter) buildTableRow(wt *git.WorktreeInfo, verbose bool, colWidths ColumnWidths) []string {
	// Marker
	marker := " "
	if wt.IsCurrent {
		marker = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(CurrentMarker)
	}

	// Worktree name - apply truncation if needed
	name := p.formatter.GetWorktreeName(wt.Path)
	if colWidths.Worktree > 0 {
		name = p.formatter.TruncateText(name, colWidths.Worktree)
	}
	if wt.IsCurrent {
		name = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(name)
	}

	// Branch name - apply smart truncation if needed
	branch := git.CleanBranchName(wt.Branch)
	if colWidths.Branch > 0 {
		branch = p.formatter.TruncateBranchName(branch, colWidths.Branch)
	}
	if wt.IsCurrent {
		branch = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(branch)
	}

	// Status with colors and symbols
	status := p.buildStyledStatus(wt.Status, wt.Remote)

	// Activity
	activity := p.formatter.FormatActivity(wt.LastActivity)

	// Build row
	row := []string{marker, name, branch, status, activity}
	if verbose {
		path := wt.Path
		if colWidths.Path > 0 {
			path = p.formatter.TruncateTextMiddle(path, colWidths.Path)
		}
		row = append(row, path)
	}

	return row
}

// buildStyledStatus creates a styled status string with colors and symbols.
func (p *ListPresenter) buildStyledStatus(status git.WorktreeStatus, remote git.RemoteStatus) string {
	statusInfo := p.formatter.FormatStatus(status, remote)

	var result string
	if statusInfo.IsClean {
		result = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render(statusInfo.Symbol)
	} else {
		warning := lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(statusInfo.Symbol)
		result = warning + " " + statusInfo.CountsText
	}

	// Add remote status if available
	if statusInfo.RemoteText != "" {
		remoteStyled := lipgloss.NewStyle().Foreground(MutedColor).Render(" " + statusInfo.RemoteText)
		result += remoteStyled
	}

	return result
}

// calculateColumnWidths determines optimal column widths based on content and terminal size.
func (p *ListPresenter) calculateColumnWidths(worktrees []git.WorktreeInfo, verbose bool) ColumnWidths {
	terminalWidth := utils.GetTerminalWidth()

	// Reserve space for table borders and padding
	// Marker(1) + borders(6) + padding(5*2) = 17 characters minimum
	minTableWidth := MinTableWidth
	if verbose {
		minTableWidth = MinTableWidthVerbose // Extra border + padding for PATH column
	}

	availableWidth := terminalWidth - minTableWidth
	if availableWidth < MinAvailableWidth {
		// Terminal too narrow for responsive sizing
		return ColumnWidths{}
	}

	// Calculate natural content widths
	maxWorktreeWidth := len(" WORKTREE")
	maxBranchWidth := len(" BRANCH")
	maxPathWidth := 0

	for i := range worktrees {
		wt := &worktrees[i]

		worktreeName := p.formatter.GetWorktreeName(wt.Path)
		if len(worktreeName) > maxWorktreeWidth {
			maxWorktreeWidth = len(worktreeName)
		}

		branchName := git.CleanBranchName(wt.Branch)
		if len(branchName) > maxBranchWidth {
			maxBranchWidth = len(branchName)
		}

		if verbose && len(wt.Path) > maxPathWidth {
			maxPathWidth = len(wt.Path)
		}
	}

	// Fixed widths for status and activity columns
	statusWidth := StatusColumnWidth   // STATUS column needs space for symbols and counts
	activityWidth := ActivityColumnWidth // ACTIVITY column ("2d ago", etc)
	reservedWidth := statusWidth + activityWidth

	if verbose {
		reservedWidth += maxPathWidth
	}

	flexibleWidth := availableWidth - reservedWidth
	if flexibleWidth < MinFlexibleWidth {
		// Not enough space for flexible columns
		return ColumnWidths{}
	}

	// Distribute flexible width between worktree and branch columns
	worktreeRatio := WorktreeColumnRatio // 40% for worktree names
	branchRatio := BranchColumnRatio     // 60% for branch names (often longer)

	worktreeWidth := int(float64(flexibleWidth) * worktreeRatio)
	branchWidth := int(float64(flexibleWidth) * branchRatio)

	// Apply minimum and maximum constraints
	if worktreeWidth < MinWorktreeWidth {
		worktreeWidth = MinWorktreeWidth
	}
	if branchWidth < MinBranchWidth {
		branchWidth = MinBranchWidth
	}

	// Don't truncate if natural width is reasonable
	if maxWorktreeWidth <= worktreeWidth+TruncationTolerance {
		worktreeWidth = 0 // No truncation needed
	}
	if maxBranchWidth <= branchWidth+TruncationTolerance {
		branchWidth = 0 // No truncation needed
	}

	pathWidth := 0
	if verbose && maxPathWidth > DefaultPathColumnWidth {
		pathWidth = DefaultPathColumnWidth // Reasonable default for paths
	}

	return ColumnWidths{
		Worktree: worktreeWidth,
		Branch:   branchWidth,
		Status:   0, // Status column doesn't need truncation
		Activity: 0, // Activity column is naturally short
		Path:     pathWidth,
	}
}

// displayTable creates and prints the lipgloss table.
func (p *ListPresenter) displayTable(headers []string, rows [][]string) {
	headerStyle := lipgloss.NewStyle().Foreground(HeaderColor).Bold(false)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(MutedColor)).
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
