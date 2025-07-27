package commands

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sqve/grove/internal/git"
)

type WorktreeFormatter struct{}

func NewWorktreeFormatter() *WorktreeFormatter {
	return &WorktreeFormatter{}
}

func (f *WorktreeFormatter) GetWorktreeName(path string) string {
	name := filepath.Base(path)
	if name == "." || name == "/" {
		return "main"
	}
	return name
}

func (f *WorktreeFormatter) FormatActivity(lastActivity time.Time) string {
	if lastActivity.IsZero() {
		return ActivityUnknown
	}

	now := time.Now()
	duration := now.Sub(lastActivity)

	if duration < time.Hour {
		minutes := int(duration.Minutes())
		if minutes < 1 {
			return ActivityJustNow
		}
		return fmt.Sprintf("%dm ago", minutes)
	}

	if duration < HoursPerDay*time.Hour {
		hours := int(duration.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}

	if duration < DaysPerWeek*HoursPerDay*time.Hour {
		days := int(duration.Hours() / HoursPerDay)
		return fmt.Sprintf("%dd ago", days)
	}

	if duration < DaysPerMonth*HoursPerDay*time.Hour {
		weeks := int(duration.Hours() / (DaysPerWeek * HoursPerDay))
		return fmt.Sprintf("%dw ago", weeks)
	}

	months := int(duration.Hours() / (DaysPerMonth * HoursPerDay))
	return fmt.Sprintf("%dmo ago", months)
}

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
		info.Symbol = CleanStatusSymbol
		info.PlainText = CleanStatusSymbol
	} else {
		var parts []string
		if status.Modified > 0 {
			parts = append(parts, strconv.Itoa(status.Modified)+ModifiedIndicator)
		}
		if status.Staged > 0 {
			parts = append(parts, strconv.Itoa(status.Staged)+StagedIndicator)
		}
		if status.Untracked > 0 {
			parts = append(parts, strconv.Itoa(status.Untracked)+UntrackedIndicator)
		}
		info.Symbol = DirtyStatusSymbol
		info.CountsText = strings.Join(parts, ", ")
		info.PlainText = DirtyStatusSymbol + " " + info.CountsText
	}

	if remote.HasRemote {
		switch {
		case remote.Ahead > 0 && remote.Behind > 0:
			info.RemoteText = fmt.Sprintf("%s%d %s%d", AheadIndicator, remote.Ahead, BehindIndicator, remote.Behind)
		case remote.Ahead > 0:
			info.RemoteText = fmt.Sprintf("%s%d", AheadIndicator, remote.Ahead)
		case remote.Behind > 0:
			info.RemoteText = fmt.Sprintf("%s%d", BehindIndicator, remote.Behind)
		}
	}

	parts := []string{info.PlainText}
	if info.RemoteText != "" {
		parts = append(parts, info.RemoteText)
	}
	if remote.IsMerged {
		parts = append(parts, MergedIndicator)
	}
	info.FullPlainText = strings.Join(parts, " ")

	return info
}

func (f *WorktreeFormatter) TruncateText(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if len(text) <= maxWidth {
		return text
	}

	// Need at least 4 characters for "..." + 1 character of content.
	if maxWidth < MinTruncationWidth {
		return text[:maxWidth]
	}

	return text[:maxWidth-3] + "..."
}

func (f *WorktreeFormatter) TruncateTextMiddle(text string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if len(text) <= maxWidth {
		return text
	}

	// Need at least 7 characters for start + "..." + end (minimum 2 chars each side).
	if maxWidth < MinMiddleTruncationWidth {
		return f.TruncateText(text, maxWidth)
	}

	ellipsis := "..."
	availableChars := maxWidth - len(ellipsis)

	// Split available characters between start and end, favoring the start slightly.
	startChars := (availableChars + 1) / 2
	endChars := availableChars - startChars

	return text[:startChars] + ellipsis + text[len(text)-endChars:]
}

func (f *WorktreeFormatter) TruncateBranchName(branchName string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	if len(branchName) <= maxWidth {
		return branchName
	}

	if maxWidth < MinBranchTruncationWidth {
		return f.TruncateText(branchName, maxWidth)
	}

	// Look for common branch patterns with slashes (e.g., feature/description).
	if slashIndex := strings.IndexByte(branchName, '/'); slashIndex != -1 && slashIndex < maxWidth-4 {
		// Try to preserve the prefix (namespace) and truncate the suffix.
		prefix := branchName[:slashIndex+1]
		suffix := branchName[slashIndex+1:]

		remainingWidth := maxWidth - len(prefix)
		if remainingWidth >= 4 {
			truncatedSuffix := f.TruncateText(suffix, remainingWidth)
			return prefix + truncatedSuffix
		}
	}

	return f.TruncateTextMiddle(branchName, maxWidth)
}

type StatusInfo struct {
	IsClean       bool
	Modified      int
	Staged        int
	Untracked     int
	HasRemote     bool
	Ahead         int
	Behind        int
	IsMerged      bool
	Symbol        string // ✓ or ⚠.
	CountsText    string // "2M, 1S" etc.
	RemoteText    string // "↑2 ↓1" etc.
	PlainText     string // "⚠ 2M, 1S" etc.
	FullPlainText string // Complete plain text with remote info.
}
