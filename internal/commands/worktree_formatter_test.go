package commands

import (
	"testing"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/stretchr/testify/assert"
)

func TestWorktreeFormatter_GetWorktreeName(t *testing.T) {
	formatter := NewWorktreeFormatter()

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{"Regular path", "/path/to/feature-branch", "feature-branch"},
		{"Current directory", ".", "main"},
		{"Root directory", "/", "main"},
		{"Single name", "main", "main"},
		{"Complex path", "/home/user/projects/repo/worktrees/bug-fix", "bug-fix"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.GetWorktreeName(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorktreeFormatter_FormatActivity(t *testing.T) {
	formatter := NewWorktreeFormatter()
	now := time.Now()

	tests := []struct {
		name     string
		activity time.Time
		expected string
	}{
		{"Zero time", time.Time{}, "unknown"},
		{"Just now", now.Add(-30 * time.Second), "just now"},
		{"Minutes ago", now.Add(-5 * time.Minute), "5m ago"},
		{"Hours ago", now.Add(-3 * time.Hour), "3h ago"},
		{"Days ago", now.Add(-2 * 24 * time.Hour), "2d ago"},
		{"Weeks ago", now.Add(-2 * 7 * 24 * time.Hour), "2w ago"},
		{"Months ago", now.Add(-2 * 30 * 24 * time.Hour), "2mo ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatActivity(tt.activity)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorktreeFormatter_FormatStatus(t *testing.T) {
	formatter := NewWorktreeFormatter()

	tests := []struct {
		name           string
		status         git.WorktreeStatus
		remote         git.RemoteStatus
		expectedSymbol string
		expectedClean  bool
	}{
		{
			name:           "Clean status",
			status:         git.WorktreeStatus{IsClean: true},
			remote:         git.RemoteStatus{},
			expectedSymbol: "✓",
			expectedClean:  true,
		},
		{
			name: "Dirty with modifications",
			status: git.WorktreeStatus{
				IsClean:  false,
				Modified: 2,
				Staged:   1,
			},
			remote:         git.RemoteStatus{},
			expectedSymbol: "⚠",
			expectedClean:  false,
		},
		{
			name:   "Clean with remote ahead",
			status: git.WorktreeStatus{IsClean: true},
			remote: git.RemoteStatus{
				HasRemote: true,
				Ahead:     2,
			},
			expectedSymbol: "✓",
			expectedClean:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.FormatStatus(tt.status, tt.remote)
			
			assert.Equal(t, tt.expectedSymbol, result.Symbol)
			assert.Equal(t, tt.expectedClean, result.IsClean)
			
			// Verify plain text contains expected elements
			if tt.expectedClean {
				assert.Contains(t, result.PlainText, "✓")
			} else {
				assert.Contains(t, result.PlainText, "⚠")
			}
		})
	}
}

func TestWorktreeFormatter_FormatStatusWithRemote(t *testing.T) {
	formatter := NewWorktreeFormatter()

	status := git.WorktreeStatus{IsClean: true}
	remote := git.RemoteStatus{
		HasRemote: true,
		Ahead:     2,
		Behind:    1,
	}

	result := formatter.FormatStatus(status, remote)
	
	assert.Equal(t, "✓", result.Symbol)
	assert.Equal(t, "↑2 ↓1", result.RemoteText)
	assert.Contains(t, result.FullPlainText, "✓")
	assert.Contains(t, result.FullPlainText, "↑2 ↓1")
}

func TestWorktreeFormatter_FormatStatusDirtyWithCounts(t *testing.T) {
	formatter := NewWorktreeFormatter()

	status := git.WorktreeStatus{
		IsClean:   false,
		Modified:  3,
		Staged:    2,
		Untracked: 1,
	}
	remote := git.RemoteStatus{}

	result := formatter.FormatStatus(status, remote)
	
	assert.Equal(t, "⚠", result.Symbol)
	assert.Equal(t, "3M, 2S, 1U", result.CountsText)
	assert.Equal(t, "⚠ 3M, 2S, 1U", result.PlainText)
}