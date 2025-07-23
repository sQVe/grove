package commands

import "github.com/charmbracelet/lipgloss"

// List command constants for better maintainability and configuration.

// Default values for list command options.
const (
	DefaultStaleDays    = 30              // Default number of days to consider a worktree stale
	DefaultSortOption   = SortByActivity  // Default sorting option
)

// Time duration constants for activity formatting.
const (
	HoursPerDay  = 24
	DaysPerWeek  = 7
	DaysPerMonth = 30
)

// Terminal and UI constants.
const (
	MinTableWidth          = 17 // Minimum width for table borders and padding
	MinTableWidthVerbose   = 29 // Minimum width for table with verbose PATH column
	MinAvailableWidth      = 20 // Minimum available width for responsive sizing
	MinFlexibleWidth       = 10 // Minimum width for flexible columns
	MinWorktreeWidth       = 8  // Minimum width for worktree name column
	MinBranchWidth         = 10 // Minimum width for branch name column
	StatusColumnWidth      = 20 // Fixed width for STATUS column
	ActivityColumnWidth    = 10 // Fixed width for ACTIVITY column
	DefaultPathColumnWidth = 40 // Default width for PATH column in verbose mode
)

// Column ratio constants for responsive sizing.
const (
	WorktreeColumnRatio = 0.4 // 40% of flexible width for worktree names
	BranchColumnRatio   = 0.6 // 60% of flexible width for branch names
)

// Truncation constants.
const (
	MinTruncationWidth       = 4  // Minimum characters needed for "..." + 1 char
	MinMiddleTruncationWidth = 7  // Minimum characters for middle truncation
	MinBranchTruncationWidth = 10 // Minimum width for intelligent branch truncation
	TruncationTolerance      = 5  // Don't truncate if within this many characters of natural width
)

// Color theme constants for consistent styling.
var (
	PrimaryColor = lipgloss.Color("#8B5CF6") // Purple - for current worktree and highlights
	SuccessColor = lipgloss.Color("#059669") // Green - for clean status (✓)
	WarningColor = lipgloss.Color("#D97706") // Orange - for dirty status (⚠)
	MutedColor   = lipgloss.Color("#9CA3AF") // Gray - for remote status and borders
	HeaderColor  = lipgloss.Color("#6B7280") // Dark gray - for table headers
)

// Status symbols and formatting.
const (
	CleanStatusSymbol = "✓"
	DirtyStatusSymbol = "⚠"
	CurrentMarker     = "*"
	EmptyMarker       = " "
)

// Activity time formatting constants.
const (
	ActivityUnknown = "unknown"
	ActivityJustNow = "just now"
)

// Remote status indicators.
const (
	AheadIndicator  = "↑"
	BehindIndicator = "↓"
	MergedIndicator = "merged"
)

// File count indicators for dirty status.
const (
	ModifiedIndicator = "M"
	StagedIndicator   = "S"
	UntrackedIndicator = "U"
)