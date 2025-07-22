package commands

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sqve/grove/internal/git"
)

// Time duration constants for activity formatting.
const (
	hoursPerDay  = 24
	daysPerWeek  = 7
	daysPerMonth = 30
)

// WorktreeFormatter provides shared utilities for formatting worktree information.
type WorktreeFormatter struct{}

// NewWorktreeFormatter creates a new WorktreeFormatter.
func NewWorktreeFormatter() *WorktreeFormatter {
	return &WorktreeFormatter{}
}

// GetWorktreeName extracts a display name from the worktree path.
func (f *WorktreeFormatter) GetWorktreeName(path string) string {
	name := filepath.Base(path)
	if name == "." || name == "/" {
		return "main"
	}
	return name
}

// FormatActivity formats the last activity timestamp for display.
func (f *WorktreeFormatter) FormatActivity(lastActivity time.Time) string {
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

	if duration < hoursPerDay*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}

	if duration < daysPerWeek*hoursPerDay*time.Hour {
		days := int(duration.Hours() / hoursPerDay)
		return fmt.Sprintf("%dd ago", days)
	}

	if duration < daysPerMonth*hoursPerDay*time.Hour {
		weeks := int(duration.Hours() / (daysPerWeek * hoursPerDay))
		return fmt.Sprintf("%dw ago", weeks)
	}

	months := int(duration.Hours() / (daysPerMonth * hoursPerDay))
	return fmt.Sprintf("%dmo ago", months)
}

// FormatStatus formats the status information for display (plain text).
// This consolidates the duplicate logic that was in both displayHumanOutput and formatStatus.
func (f *WorktreeFormatter) FormatStatus(status git.WorktreeStatus, remote git.RemoteStatus) StatusInfo {
	info := StatusInfo{
		IsClean:   status.IsClean,
		Modified:  status.Modified,
		Staged:    status.Staged,
		Untracked: status.Untracked,
		HasRemote: remote.HasRemote,
		Ahead:     remote.Ahead,
		Behind:    remote.Behind,
		IsMerged:  remote.IsMerged,
	}

	if status.IsClean {
		info.Symbol = "✓"
		info.PlainText = "✓"
	} else {
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
		info.Symbol = "⚠"
		info.CountsText = strings.Join(parts, ", ")
		info.PlainText = "⚠ " + info.CountsText
	}

	// Add remote status if available
	if remote.HasRemote {
		switch {
		case remote.Ahead > 0 && remote.Behind > 0:
			info.RemoteText = fmt.Sprintf("↑%d ↓%d", remote.Ahead, remote.Behind)
		case remote.Ahead > 0:
			info.RemoteText = fmt.Sprintf("↑%d", remote.Ahead)
		case remote.Behind > 0:
			info.RemoteText = fmt.Sprintf("↓%d", remote.Behind)
		}
	}

	// Build complete plain text representation
	parts := []string{info.PlainText}
	if info.RemoteText != "" {
		parts = append(parts, info.RemoteText)
	}
	if remote.IsMerged {
		parts = append(parts, "merged")
	}
	info.FullPlainText = strings.Join(parts, " ")

	return info
}

// TruncateText truncates text to the specified maximum width with an ellipsis.
// If the text is shorter than maxWidth, it's returned unchanged.
// For longer text, it truncates and adds "..." while respecting the maxWidth limit.
func (f *WorktreeFormatter) TruncateText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if len(text) <= maxWidth {
		return text
	}

	// Need at least 4 characters for "..." + 1 character of content
	if maxWidth < 4 {
		return text[:maxWidth]
	}

	return text[:maxWidth-3] + "..."
}

// TruncateTextMiddle truncates text in the middle to preserve both start and end.
// This is useful for paths and branch names where both the prefix and suffix matter.
func (f *WorktreeFormatter) TruncateTextMiddle(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if len(text) <= maxWidth {
		return text
	}

	// Need at least 7 characters for start + "..." + end (minimum 2 chars each side)
	if maxWidth < 7 {
		return f.TruncateText(text, maxWidth)
	}

	ellipsis := "..."
	availableChars := maxWidth - len(ellipsis)

	// Split available characters between start and end, favoring the start slightly
	startChars := (availableChars + 1) / 2
	endChars := availableChars - startChars

	return text[:startChars] + ellipsis + text[len(text)-endChars:]
}

// TruncateBranchName intelligently truncates branch names while preserving meaningful parts.
// It handles common patterns like "feature/description" by preserving the namespace.
func (f *WorktreeFormatter) TruncateBranchName(branchName string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if len(branchName) <= maxWidth {
		return branchName
	}

	// For very short widths, just truncate normally
	if maxWidth < 10 {
		return f.TruncateText(branchName, maxWidth)
	}

	// Look for common branch patterns with slashes (e.g., feature/description)
	if slashIndex := strings.IndexByte(branchName, '/'); slashIndex != -1 && slashIndex < maxWidth-4 {
		// Try to preserve the prefix (namespace) and truncate the suffix
		prefix := branchName[:slashIndex+1] // Include the slash
		suffix := branchName[slashIndex+1:]

		remainingWidth := maxWidth - len(prefix)
		if remainingWidth >= 4 { // Need space for at least "..."
			truncatedSuffix := f.TruncateText(suffix, remainingWidth)
			return prefix + truncatedSuffix
		}
	}

	// Fall back to middle truncation for long names without clear structure
	return f.TruncateTextMiddle(branchName, maxWidth)
}

// StatusInfo contains formatted status information that can be styled differently for various outputs.
type StatusInfo struct {
	IsClean       bool
	Modified      int
	Staged        int
	Untracked     int
	HasRemote     bool
	Ahead         int
	Behind        int
	IsMerged      bool
	Symbol        string // ✓ or ⚠
	CountsText    string // "2M, 1S" etc
	RemoteText    string // "↑2 ↓1" etc
	PlainText     string // "⚠ 2M, 1S" etc
	FullPlainText string // Complete plain text with remote info
}
