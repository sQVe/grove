package formatter

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/styles"
)

// Indicator legend (color / plain):
// Current: ● / *
// Lock: nf-md-lock / [locked]
// Dirty: nf-md-circle-edit-outline / [dirty]
// Sync: green ↑N, yellow ↓N (plain +N/-N), "=" when in sync, blank if no upstream
// Gone: dim × / gone
// Verbose prefix: ↳ / >

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
func Gone() string {
	if config.IsPlain() {
		return "gone"
	}
	return styles.Render(&styles.Dimmed, "×")
}

// SubItemPrefix returns the prefix for verbose sub-items
func SubItemPrefix() string {
	if config.IsPlain() {
		return ">"
	}
	return "↳"
}

// WorktreeRow formats a single worktree row for list/status output
// Format: marker name [branch] indicators
func WorktreeRow(info *git.WorktreeInfo, isCurrent bool, namePadWidth, branchPadWidth int) string {
	marker := CurrentMarker(isCurrent)
	dirty := Dirty(info.Dirty)
	lock := Lock(info.Locked)

	var sync string
	if info.Gone {
		sync = Gone()
	} else {
		sync = Sync(info.Ahead, info.Behind, !info.NoUpstream)
	}

	// Worktree name (directory basename)
	name := filepath.Base(info.Path)
	nameLen := utf8.RuneCountInString(name)
	nameDisplay := name
	if namePadWidth > 0 && nameLen < namePadWidth {
		nameDisplay = name + strings.Repeat(" ", namePadWidth-nameLen)
	}

	// Branch display: [branch] or (detached)
	var branchDisplay string
	if info.Detached {
		branchDisplay = styles.Render(&styles.Dimmed, "(detached)")
	} else {
		branchDisplay = styles.Render(&styles.Dimmed, "["+info.Branch+"]")
	}

	// Pad the branch display for alignment (calculate visible length excluding ANSI codes)
	branchVisibleLen := len(info.Branch) + 2 // brackets add 2 chars
	if info.Detached {
		branchVisibleLen = 10 // "(detached)" is 10 chars
	}
	if branchPadWidth > 0 && branchVisibleLen < branchPadWidth {
		branchDisplay += strings.Repeat(" ", branchPadWidth-branchVisibleLen)
	}

	parts := []string{marker, styles.Render(&styles.Worktree, nameDisplay), branchDisplay}

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

// WorktreeLabel returns a simple label for a worktree: "directory [branch]"
func WorktreeLabel(info *git.WorktreeInfo) string {
	dir := filepath.Base(info.Path)
	if dir == "" || dir == "." {
		dir = info.Path
	}
	return fmt.Sprintf("%s [%s]", dir, info.Branch)
}

// VerboseSubItems returns the verbose sub-items for a worktree
func VerboseSubItems(info *git.WorktreeInfo) []string {
	prefix := SubItemPrefix()
	var items []string

	items = append(items, fmt.Sprintf("    %s path: %s", prefix, styles.RenderPath(info.Path)))

	if info.Upstream != "" {
		items = append(items, fmt.Sprintf("    %s upstream: %s", prefix, info.Upstream))
	}

	if info.Locked && info.LockReason != "" {
		items = append(items, fmt.Sprintf("    %s lock reason: %s", prefix, info.LockReason))
	}

	return items
}
