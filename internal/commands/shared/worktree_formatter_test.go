//go:build !integration
// +build !integration

package shared

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

func TestWorktreeFormatter_TruncateText(t *testing.T) {
	formatter := NewWorktreeFormatter()

	tests := []struct {
		name     string
		text     string
		maxWidth int
		expected string
	}{
		{
			name:     "text shorter than max width",
			text:     "short",
			maxWidth: 10,
			expected: "short",
		},
		{
			name:     "text equal to max width",
			text:     "exact",
			maxWidth: 5,
			expected: "exact",
		},
		{
			name:     "text longer than max width",
			text:     "this-is-a-very-long-text",
			maxWidth: 15,
			expected: "this-is-a-ve...",
		},
		{
			name:     "max width zero",
			text:     "test",
			maxWidth: 0,
			expected: "",
		},
		{
			name:     "max width negative",
			text:     "test",
			maxWidth: -1,
			expected: "",
		},
		{
			name:     "max width very small",
			text:     "test",
			maxWidth: 2,
			expected: "te",
		},
		{
			name:     "max width just enough for ellipsis",
			text:     "testing",
			maxWidth: 4,
			expected: "t...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.TruncateText(tt.text, tt.maxWidth)
			assert.Equal(t, tt.expected, result)

			if tt.maxWidth > 0 {
				assert.LessOrEqual(t, len(result), tt.maxWidth)
			}
		})
	}
}

func TestWorktreeFormatter_TruncateTextMiddle(t *testing.T) {
	formatter := NewWorktreeFormatter()

	tests := []struct {
		name     string
		text     string
		maxWidth int
		expected string
	}{
		{
			name:     "text shorter than max width",
			text:     "short",
			maxWidth: 10,
			expected: "short",
		},
		{
			name:     "text longer with middle truncation",
			text:     "this-is-a-very-long-filename.txt",
			maxWidth: 20,
			expected: "this-is-a...name.txt",
		},
		{
			name:     "max width too small for middle truncation",
			text:     "testing",
			maxWidth: 5,
			expected: "te...",
		},
		{
			name:     "minimum width for middle truncation",
			text:     "test-file",
			maxWidth: 7,
			expected: "te...le",
		},
		{
			name:     "max width zero",
			text:     "test",
			maxWidth: 0,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.TruncateTextMiddle(tt.text, tt.maxWidth)
			assert.Equal(t, tt.expected, result)

			if tt.maxWidth > 0 {
				assert.LessOrEqual(t, len(result), tt.maxWidth)
			}
		})
	}
}

func TestWorktreeFormatter_TruncateBranchName(t *testing.T) {
	formatter := NewWorktreeFormatter()

	tests := []struct {
		name     string
		branch   string
		maxWidth int
		expected string
	}{
		{
			name:     "branch shorter than max width",
			branch:   "main",
			maxWidth: 10,
			expected: "main",
		},
		{
			name:     "feature branch with namespace preservation",
			branch:   "feature/very-long-description-here",
			maxWidth: 20,
			expected: "feature/very-long...",
		},
		{
			name:     "bugfix branch with namespace preservation",
			branch:   "bugfix/fix-issue-123-with-authentication",
			maxWidth: 25,
			expected: "bugfix/fix-issue-123-w...",
		},
		{
			name:     "long branch without slash - middle truncation",
			branch:   "very-long-branch-name-without-slashes",
			maxWidth: 20,
			expected: "very-long...-slashes",
		},
		{
			name:     "max width too small",
			branch:   "feature/test",
			maxWidth: 5,
			expected: "fe...",
		},
		{
			name:     "max width zero",
			branch:   "test",
			maxWidth: 0,
			expected: "",
		},
		{
			name:     "nested namespace preservation",
			branch:   "team/backend/feature/new-api-endpoint",
			maxWidth: 25,
			expected: "team/backend/feature/n...",
		},
		{
			name:     "short namespace, can't preserve",
			branch:   "a/very-long-description-that-cannot-fit",
			maxWidth: 8,
			expected: "a/ver...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.TruncateBranchName(tt.branch, tt.maxWidth)
			assert.Equal(t, tt.expected, result)

			if tt.maxWidth > 0 {
				assert.LessOrEqual(t, len(result), tt.maxWidth)
			}
		})
	}
}
