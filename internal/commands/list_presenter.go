package commands

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/utils"
)

type ColumnWidths struct {
	Worktree int
	Branch   int
	Status   int
	Activity int
	Path     int // Only used in verbose mode.
}

type ListPresenter struct {
	formatter *WorktreeFormatter
}

func NewListPresenter() *ListPresenter {
	return &ListPresenter{
		formatter: NewWorktreeFormatter(),
	}
}

func (p *ListPresenter) DisplayHuman(worktrees []git.WorktreeInfo, verbose bool) error {
	if len(worktrees) == 0 {
		emptyStyle := lipgloss.NewStyle().Foreground(MutedColor).Italic(true)
		fmt.Println(emptyStyle.Render("No worktrees match the specified criteria"))
		return nil
	}

	colWidths := p.calculateColumnWidths(worktrees, verbose)

	var rows [][]string

	headers := []string{"", " WORKTREE", " BRANCH", " STATUS", " ACTIVITY"}
	if verbose {
		headers = append(headers, " PATH")
	}

	for i := range worktrees {
		wt := &worktrees[i]
		row := p.buildTableRow(wt, verbose, colWidths)
		rows = append(rows, row)
	}

	p.displayTable(headers, rows)
	return nil
}

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

		fmt.Println()
	}

	return nil
}

func (p *ListPresenter) buildTableRow(wt *git.WorktreeInfo, verbose bool, colWidths ColumnWidths) []string {
	marker := " "
	if wt.IsCurrent {
		marker = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(CurrentMarker)
	}

	name := p.formatter.GetWorktreeName(wt.Path)
	if colWidths.Worktree > 0 {
		name = p.formatter.TruncateText(name, colWidths.Worktree)
	}
	if wt.IsCurrent {
		name = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(name)
	}

	branch := git.CleanBranchName(wt.Branch)
	if colWidths.Branch > 0 {
		branch = p.formatter.TruncateBranchName(branch, colWidths.Branch)
	}
	if wt.IsCurrent {
		branch = lipgloss.NewStyle().Foreground(PrimaryColor).Bold(true).Render(branch)
	}

	status := p.buildStyledStatus(wt.Status, wt.Remote)
	activity := p.formatter.FormatActivity(wt.LastActivity)
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

func (p *ListPresenter) buildStyledStatus(status git.WorktreeStatus, remote git.RemoteStatus) string {
	statusInfo := p.formatter.FormatStatus(status, remote)

	var result string
	if statusInfo.IsClean {
		result = lipgloss.NewStyle().Foreground(SuccessColor).Bold(true).Render(statusInfo.Symbol)
	} else {
		warning := lipgloss.NewStyle().Foreground(WarningColor).Bold(true).Render(statusInfo.Symbol)
		result = warning + " " + statusInfo.CountsText
	}

	if statusInfo.RemoteText != "" {
		remoteStyled := lipgloss.NewStyle().Foreground(MutedColor).Render(" " + statusInfo.RemoteText)
		result += remoteStyled
	}

	return result
}

func (p *ListPresenter) calculateColumnWidths(worktrees []git.WorktreeInfo, verbose bool) ColumnWidths {
	terminalWidth := utils.GetTerminalWidth()

	// Reserve space for table borders and padding.
	// Marker(1) + borders(6) + padding(5*2) = 17 characters minimum.
	minTableWidth := MinTableWidth
	if verbose {
		minTableWidth = MinTableWidthVerbose
	}

	availableWidth := terminalWidth - minTableWidth
	if availableWidth < MinAvailableWidth {
		return ColumnWidths{}
	}

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

	// Fixed widths for status and activity columns.
	statusWidth := StatusColumnWidth     // STATUS column needs space for symbols and counts.
	activityWidth := ActivityColumnWidth // ACTIVITY column ("2d ago", etc).
	reservedWidth := statusWidth + activityWidth

	if verbose {
		reservedWidth += maxPathWidth
	}

	flexibleWidth := availableWidth - reservedWidth
	if flexibleWidth < MinFlexibleWidth {
		return ColumnWidths{}
	}

	// Distribute flexible width between worktree and branch columns.
	worktreeRatio := WorktreeColumnRatio // 40% for worktree names.
	branchRatio := BranchColumnRatio     // 60% for branch names (often longer).

	worktreeWidth := int(float64(flexibleWidth) * worktreeRatio)
	branchWidth := int(float64(flexibleWidth) * branchRatio)

	if worktreeWidth < MinWorktreeWidth {
		worktreeWidth = MinWorktreeWidth
	}
	if branchWidth < MinBranchWidth {
		branchWidth = MinBranchWidth
	}

	// Don't truncate if natural width is reasonable.
	if maxWorktreeWidth <= worktreeWidth+TruncationTolerance {
		worktreeWidth = 0
	}
	if maxBranchWidth <= branchWidth+TruncationTolerance {
		branchWidth = 0
	}

	pathWidth := 0
	if verbose && maxPathWidth > DefaultPathColumnWidth {
		pathWidth = DefaultPathColumnWidth
	}

	return ColumnWidths{
		Worktree: worktreeWidth,
		Branch:   branchWidth,
		Status:   0,
		Activity: 0,
		Path:     pathWidth,
	}
}

func (p *ListPresenter) displayTable(headers []string, rows [][]string) {
	headerStyle := lipgloss.NewStyle().Foreground(HeaderColor).Bold(false)

	t := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(lipgloss.NewStyle().Foreground(MutedColor)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle().Padding(0, 1)
		}).
		Headers(headers...).
		Rows(rows...)

	fmt.Println(t)
}
