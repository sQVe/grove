package shared

import "github.com/charmbracelet/lipgloss"

const (
	DefaultStaleDays = 30
)

const (
	HoursPerDay  = 24
	DaysPerWeek  = 7
	DaysPerMonth = 30
)

const (
	MinTableWidth          = 17
	MinTableWidthVerbose   = 29
	MinAvailableWidth      = 20
	MinFlexibleWidth       = 10
	MinWorktreeWidth       = 8
	MinBranchWidth         = 10
	StatusColumnWidth      = 20
	ActivityColumnWidth    = 10
	DefaultPathColumnWidth = 40
)

const (
	WorktreeColumnRatio = 0.4 // 40% of flexible width for worktree names.
	BranchColumnRatio   = 0.6 // 60% of flexible width for branch names.
)

const (
	MinTruncationWidth       = 4  // Minimum characters needed for "..." + 1 char.
	MinMiddleTruncationWidth = 7  // Minimum characters for middle truncation.
	MinBranchTruncationWidth = 10 // Minimum width for intelligent branch truncation.
	TruncationTolerance      = 5  // Don't truncate if within this many characters of natural width.
)

var (
	PrimaryColor = lipgloss.Color("#8B5CF6") // Purple - for current worktree and highlights.
	SuccessColor = lipgloss.Color("#059669") // Green - for clean status (✓).
	WarningColor = lipgloss.Color("#D97706") // Orange - for dirty status (⚠).
	MutedColor   = lipgloss.Color("#9CA3AF") // Gray - for remote status and borders.
	HeaderColor  = lipgloss.Color("#6B7280") // Dark gray - for table headers.
)

const (
	CleanStatusSymbol = "✓"
	DirtyStatusSymbol = "⚠"
	CurrentMarker     = "*"
	EmptyMarker       = " "
)

const (
	ActivityUnknown = "unknown"
	ActivityJustNow = "just now"
)

const (
	AheadIndicator  = "↑"
	BehindIndicator = "↓"
	MergedIndicator = "merged"
)

const (
	ModifiedIndicator  = "M"
	StagedIndicator    = "S"
	UntrackedIndicator = "U"
)
