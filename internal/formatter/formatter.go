package formatter

import (
	"fmt"
	"strings"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/styles"
)

// Icons for indicators
const (
	iconCurrent = "●"          // filled circle (U+25CF)
	iconLock    = "\U000F033E" // nf-md-lock (U+F033E)
	iconDirty   = "\U000F0992" // nf-md-circle-edit-outline (U+F0992)
)

// ASCII fallbacks for plain mode or when Nerd Fonts disabled
const (
	asciiCurrent = "*"
	asciiLock    = "[locked]"
	asciiDirty   = "[dirty]"
)

// useAsciiIcons returns true if ASCII icons should be used instead of Nerd Fonts
func useAsciiIcons() bool {
	return config.IsPlain() || !config.IsNerdFonts()
}

// CurrentMarker returns the marker for current worktree
// Color mode: ● (filled circle)
// Plain mode: *
// Non-current: space
func CurrentMarker(isCurrent bool) string {
	if !isCurrent {
		return " "
	}
	if useAsciiIcons() {
		return asciiCurrent
	}
	return styles.Render(&styles.Success, iconCurrent)
}

// Lock returns the lock indicator
// Color mode:  (nf-md-lock)
// Plain/no-nerdfonts mode: [locked]
// Not locked: empty string
func Lock(isLocked bool) string {
	if !isLocked {
		return ""
	}
	if useAsciiIcons() {
		return asciiLock
	}
	return styles.Render(&styles.Warning, iconLock)
}

// Dirty returns the dirty indicator
// Color mode: 󰦒 (nf-md-circle-edit-outline)
// Plain/no-nerdfonts mode: [dirty]
// Clean: empty string
func Dirty(isDirty bool) string {
	if !isDirty {
		return ""
	}
	if useAsciiIcons() {
		return asciiDirty
	}
	return styles.Render(&styles.Warning, iconDirty)
}

// Sync returns the sync status indicator
// Color mode: ↑N (green) / ↓N (yellow)
// Plain mode: +N / -N
// Synced: = (dimmed)
// No upstream: empty string
func Sync(ahead, behind int, hasUpstream bool) string {
	if !hasUpstream {
		return ""
	}

	if ahead == 0 && behind == 0 {
		return styles.Render(&styles.Dimmed, "=")
	}

	var parts []string
	if ahead > 0 {
		if config.IsPlain() {
			parts = append(parts, fmt.Sprintf("+%d", ahead))
		} else {
			parts = append(parts, styles.Render(&styles.Success, fmt.Sprintf("↑%d", ahead)))
		}
	}
	if behind > 0 {
		if config.IsPlain() {
			parts = append(parts, fmt.Sprintf("-%d", behind))
		} else {
			parts = append(parts, styles.Render(&styles.Warning, fmt.Sprintf("↓%d", behind)))
		}
	}

	return strings.Join(parts, "")
}

// Gone returns the gone indicator for deleted upstream
// Color mode: × (dimmed)
// Plain mode: gone
func Gone() string {
	if config.IsPlain() {
		return "gone"
	}
	return styles.Render(&styles.Dimmed, "×")
}

// SubItemPrefix returns the prefix for verbose sub-items
// Color mode: ↳
// Plain mode: >
func SubItemPrefix() string {
	if config.IsPlain() {
		return ">"
	}
	return "↳"
}

// BranchName returns the styled branch name
func BranchName(name string) string {
	return styles.Render(&styles.Worktree, name)
}

// WorktreeRow formats a single worktree row for list/status output
// Returns: "marker branch dirty sync lock"
func WorktreeRow(info *git.WorktreeInfo, isCurrent bool, padWidth int) string {
	marker := CurrentMarker(isCurrent)
	dirty := Dirty(info.Dirty)
	lock := Lock(info.Locked)

	var sync string
	if info.Gone {
		sync = Gone()
	} else {
		sync = Sync(info.Ahead, info.Behind, !info.NoUpstream)
	}

	// Build the row with proper spacing
	// Format: marker branch [padding] dirty sync lock
	branchDisplay := info.Branch
	if padWidth > 0 && len(info.Branch) < padWidth {
		branchDisplay = info.Branch + strings.Repeat(" ", padWidth-len(info.Branch))
	}

	parts := []string{marker, styles.Render(&styles.Worktree, branchDisplay)}

	// Add indicators with spacing (lock first, then dirty, then sync)
	indicators := []string{}
	if lock != "" {
		indicators = append(indicators, lock)
	}
	if dirty != "" {
		indicators = append(indicators, dirty)
	}
	if sync != "" {
		indicators = append(indicators, sync)
	}

	if len(indicators) > 0 {
		parts = append(parts, strings.Join(indicators, " "))
	}

	return strings.Join(parts, " ")
}

// VerboseSubItems returns the verbose sub-items for a worktree
// Returns slice of formatted strings like "↳ path: /path/to/worktree"
func VerboseSubItems(info *git.WorktreeInfo) []string {
	prefix := SubItemPrefix()
	var items []string

	// Path is always shown
	items = append(items, fmt.Sprintf("    %s path: %s", prefix, styles.Render(&styles.Path, info.Path)))

	// Upstream if exists
	if info.Upstream != "" {
		items = append(items, fmt.Sprintf("    %s upstream: %s", prefix, info.Upstream))
	}

	// Lock reason only if locked AND has a reason
	if info.Locked && info.LockReason != "" {
		items = append(items, fmt.Sprintf("    %s lock reason: %s", prefix, info.LockReason))
	}

	return items
}
