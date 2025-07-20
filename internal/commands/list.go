package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
)

// Color theme for table styling.
var (
	primaryColor = lipgloss.Color("#8B5CF6")
	successColor = lipgloss.Color("#059669")
	warningColor = lipgloss.Color("#D97706")
	mutedColor   = lipgloss.Color("#9CA3AF")
	headerColor  = lipgloss.Color("#6B7280")
)

// ListSortOption represents the available sorting options for worktree listing.
type ListSortOption string

const (
	SortByActivity ListSortOption = "activity"
	SortByName     ListSortOption = "name"
	SortByStatus   ListSortOption = "status"
)

// ListOptions contains configuration options for the list command.
type ListOptions struct {
	Sort      ListSortOption
	Verbose   bool
	Porcelain bool
	DirtyOnly bool
	StaleOnly bool
	CleanOnly bool
	StaleDays int
}

// NewListCmd creates the list command.
func NewListCmd() *cobra.Command {
	options := &ListOptions{
		Sort:      SortByActivity,
		StaleDays: 30,
	}

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all worktrees with status information",
		Long: `List all Git worktrees in the repository with comprehensive status information.

Shows each worktree with its branch, status, last activity, and remote tracking information.
The current worktree is marked with an asterisk (*).

Status indicators:
  ✓ clean     - No uncommitted changes
  ⚠ dirty     - Has uncommitted changes (shows M/S/U counts)
  ↑N ↓M       - N commits ahead, M commits behind remote
  merged      - Branch has been merged

Examples:
  grove list                    # List all worktrees sorted by activity
  grove list --sort=name        # Sort alphabetically by worktree name
  grove list --dirty            # Show only worktrees with changes
  grove list --stale --days=14  # Show worktrees unused for 14+ days
  grove list --verbose          # Show extended information including paths
  grove list --porcelain        # Machine-readable output

Sorting options: activity (default), name, status`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runListCommand(options)
		},
	}

	// Add flags
	cmd.Flags().StringVar((*string)(&options.Sort), "sort", "activity", "Sort by: activity, name, status")
	cmd.Flags().BoolVarP(&options.Verbose, "verbose", "v", false, "Show extended information including full paths")
	cmd.Flags().BoolVar(&options.Porcelain, "porcelain", false, "Machine-readable output")
	cmd.Flags().BoolVar(&options.DirtyOnly, "dirty", false, "Show only worktrees with uncommitted changes")
	cmd.Flags().BoolVar(&options.StaleOnly, "stale", false, "Show only stale worktrees (unused for specified days)")
	cmd.Flags().BoolVar(&options.CleanOnly, "clean", false, "Show only clean worktrees (no uncommitted changes)")
	cmd.Flags().IntVar(&options.StaleDays, "days", 30, "Number of days to consider a worktree stale (used with --stale)")

	return cmd
}

// runListCommand executes the list command with the given options.
func runListCommand(options *ListOptions) error {
	// Validate mutually exclusive filters
	filterCount := 0
	if options.DirtyOnly {
		filterCount++
	}
	if options.StaleOnly {
		filterCount++
	}
	if options.CleanOnly {
		filterCount++
	}
	if filterCount > 1 {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"Cannot specify multiple filters (--dirty, --stale, --clean) simultaneously",
			nil,
		)
	}

	// Validate sort option
	validSorts := []ListSortOption{SortByActivity, SortByName, SortByStatus}
	validSort := false
	for _, valid := range validSorts {
		if options.Sort == valid {
			validSort = true
			break
		}
	}
	if !validSort {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			fmt.Sprintf("Invalid sort option: %s", options.Sort),
			nil,
		)
	}

	logger.Debug("Listing worktrees", "sort", options.Sort, "verbose", options.Verbose)

	// Find the grove repository root (bare repository)
	repoPath, err := findGroveRepository()
	if err != nil {
		return errors.NewGroveError(
			errors.ErrCodeGitOperation,
			"Could not find grove repository: "+err.Error(),
			err,
		)
	}

	// Get worktree information
	worktrees, err := git.ListWorktreesFromRepo(git.DefaultExecutor, repoPath)
	if err != nil {
		return errors.ErrGitWorktree("list", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

	// Apply filters
	filteredWorktrees := applyFilters(worktrees, options)

	// Sort worktrees
	sortWorktrees(filteredWorktrees, options.Sort)

	// Display results
	if options.Porcelain {
		return displayPorcelainOutput(filteredWorktrees)
	}
	return displayHumanOutput(filteredWorktrees, options.Verbose)
}

// applyFilters filters the worktree list based on the specified options.
func applyFilters(worktrees []git.WorktreeInfo, options *ListOptions) []git.WorktreeInfo {
	if !options.DirtyOnly && !options.StaleOnly && !options.CleanOnly {
		return worktrees
	}

	var filtered []git.WorktreeInfo
	staleThreshold := time.Now().AddDate(0, 0, -options.StaleDays)

	for i := range worktrees {
		wt := &worktrees[i]
		switch {
		case options.DirtyOnly && !wt.Status.IsClean:
			filtered = append(filtered, *wt)
		case options.StaleOnly && !wt.LastActivity.IsZero() && wt.LastActivity.Before(staleThreshold):
			filtered = append(filtered, *wt)
		case options.CleanOnly && wt.Status.IsClean:
			filtered = append(filtered, *wt)
		}
	}

	return filtered
}

// sortWorktrees sorts the worktree list based on the specified sort option.
func sortWorktrees(worktrees []git.WorktreeInfo, sortBy ListSortOption) {
	switch sortBy {
	case SortByActivity:
		sort.Slice(worktrees, func(i, j int) bool {
			return worktrees[i].LastActivity.After(worktrees[j].LastActivity)
		})
	case SortByName:
		sort.Slice(worktrees, func(i, j int) bool {
			return worktrees[i].Path < worktrees[j].Path
		})
	case SortByStatus:
		sort.Slice(worktrees, func(i, j int) bool {
			if worktrees[i].Status.IsClean != worktrees[j].Status.IsClean {
				return !worktrees[i].Status.IsClean
			}
			return worktrees[i].LastActivity.After(worktrees[j].LastActivity)
		})
	}
}

// displayHumanOutput displays worktrees using lipgloss table component.
func displayHumanOutput(worktrees []git.WorktreeInfo, verbose bool) error {
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

		// Marker
		marker := " "
		if wt.IsCurrent {
			marker = lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render("*")
		}

		// Worktree name
		name := getWorktreeName(wt.Path)
		if wt.IsCurrent {
			name = lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(name)
		}

		// Branch name
		branch := wt.Branch
		if wt.IsCurrent {
			branch = lipgloss.NewStyle().Foreground(primaryColor).Bold(true).Render(branch)
		}

		// Status with colors and symbols
		status := ""
		if wt.Status.IsClean {
			status = lipgloss.NewStyle().Foreground(successColor).Bold(true).Render("✓")
		} else {
			var parts []string
			if wt.Status.Modified > 0 {
				parts = append(parts, fmt.Sprintf("%dM", wt.Status.Modified))
			}
			if wt.Status.Staged > 0 {
				parts = append(parts, fmt.Sprintf("%dS", wt.Status.Staged))
			}
			if wt.Status.Untracked > 0 {
				parts = append(parts, fmt.Sprintf("%dU", wt.Status.Untracked))
			}
			warning := lipgloss.NewStyle().Foreground(warningColor).Bold(true).Render("⚠")
			status = warning + " " + strings.Join(parts, ", ")
		}

		// Add remote status if available
		if wt.Remote.HasRemote {
			remoteInfo := ""
			switch {
			case wt.Remote.Ahead > 0 && wt.Remote.Behind > 0:
				remoteInfo = fmt.Sprintf(" ↑%d ↓%d", wt.Remote.Ahead, wt.Remote.Behind)
			case wt.Remote.Ahead > 0:
				remoteInfo = fmt.Sprintf(" ↑%d", wt.Remote.Ahead)
			case wt.Remote.Behind > 0:
				remoteInfo = fmt.Sprintf(" ↓%d", wt.Remote.Behind)
			}
			if remoteInfo != "" {
				status += lipgloss.NewStyle().Foreground(mutedColor).Render(remoteInfo)
			}
		}

		// Activity
		activity := formatActivity(wt.LastActivity)

		// Build row
		row := []string{marker, name, branch, status, activity}
		if verbose {
			row = append(row, wt.Path)
		}

		rows = append(rows, row)
	}

	// Create lipgloss table with better styling
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
	return nil
}

// displayPorcelainOutput displays worktrees in machine-readable format.
func displayPorcelainOutput(worktrees []git.WorktreeInfo) error {
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

// getWorktreeName extracts a display name from the worktree path.
func getWorktreeName(path string) string {
	name := filepath.Base(path)
	if name == "." || name == "/" {
		return "main"
	}
	return name
}

// formatStatus formats the status information for display (plain text).
func formatStatus(status git.WorktreeStatus, remote git.RemoteStatus) string {
	if status.IsClean {
		parts := []string{"✓"}

		if remote.HasRemote {
			switch {
			case remote.Ahead > 0 && remote.Behind > 0:
				parts = append(parts, fmt.Sprintf("↑%d ↓%d", remote.Ahead, remote.Behind))
			case remote.Ahead > 0:
				parts = append(parts, fmt.Sprintf("↑%d", remote.Ahead))
			case remote.Behind > 0:
				parts = append(parts, fmt.Sprintf("↓%d", remote.Behind))
			}
		}

		if remote.IsMerged {
			parts = append(parts, "merged")
		}

		return strings.Join(parts, " ")
	}

	// Format dirty status with counts
	var parts []string
	if status.Modified > 0 {
		parts = append(parts, strconv.Itoa(status.Modified)+"M")
	}
	if status.Staged > 0 {
		parts = append(parts, strconv.Itoa(status.Staged)+"S")
	}
	if status.Untracked > 0 {
		parts = append(parts, strconv.Itoa(status.Untracked)+"U")
	}

	result := "⚠ " + strings.Join(parts, ", ")

	if remote.HasRemote {
		switch {
		case remote.Ahead > 0 && remote.Behind > 0:
			result += fmt.Sprintf(" ↑%d ↓%d", remote.Ahead, remote.Behind)
		case remote.Ahead > 0:
			result += fmt.Sprintf(" ↑%d", remote.Ahead)
		case remote.Behind > 0:
			result += fmt.Sprintf(" ↓%d", remote.Behind)
		}
	}

	return result
}

// formatActivity formats the last activity timestamp for display.
func formatActivity(lastActivity time.Time) string {
	if lastActivity.IsZero() {
		return "unknown"
	}

	now := time.Now()
	duration := now.Sub(lastActivity)

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes < 1 {
			return "just now"
		}
		return fmt.Sprintf("%dm ago", minutes)
	}

	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}

	if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}

	if duration < 30*24*time.Hour {
		weeks := int(duration.Hours() / (7 * 24))
		return fmt.Sprintf("%dw ago", weeks)
	}

	months := int(duration.Hours() / (30 * 24))
	return fmt.Sprintf("%dmo ago", months)
}

// findGroveRepository finds the grove repository root by looking for a .bare directory.
// It starts from the current directory and walks up the directory tree.
func findGroveRepository() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Start from current directory and walk up to find .bare directory
	currentPath := cwd
	for {
		bareDir := filepath.Join(currentPath, ".bare")
		if stat, err := os.Stat(bareDir); err == nil && stat.IsDir() {
			return bareDir, nil
		}

		// Move up one directory
		parent := filepath.Dir(currentPath)
		if parent == currentPath {
			// Reached filesystem root
			break
		}
		currentPath = parent
	}

	return "", fmt.Errorf("no grove repository (.bare directory) found")
}
