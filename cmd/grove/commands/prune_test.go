package commands

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewPruneCmd(t *testing.T) {
	cmd := NewPruneCmd()

	if cmd.Use != "prune" {
		t.Errorf("expected Use 'prune', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("commit") == nil {
		t.Error("expected --commit flag")
	}
	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("expected --force flag")
	}
	if forceFlag.Shorthand != "f" {
		t.Errorf("expected force shorthand 'f', got %q", forceFlag.Shorthand)
	}
	if cmd.Flags().Lookup("stale") == nil {
		t.Error("expected --stale flag")
	}
	if cmd.Flags().Lookup("merged") == nil {
		t.Error("expected --merged flag")
	}
	if cmd.Flags().Lookup("detached") == nil {
		t.Error("expected --detached flag")
	}
}

func TestRunPrune(t *testing.T) {
	t.Run("returns error when not in workspace", func(t *testing.T) {
		// Save and restore cwd
		origDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(origDir) }()

		tmpDir := t.TempDir()
		_ = os.Chdir(tmpDir)

		err := runPrune(false, false, "", false, false)
		if err == nil {
			t.Error("expected error for non-workspace directory")
		}
		if !errors.Is(err, workspace.ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace, got: %v", err)
		}
	})
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"30 days", "30d", 30 * 24 * time.Hour, false},
		{"2 weeks", "2w", 14 * 24 * time.Hour, false},
		{"6 months", "6m", 180 * 24 * time.Hour, false},
		{"1 day", "1d", 24 * time.Hour, false},
		{"1 week", "1w", 7 * 24 * time.Hour, false},
		{"1 month", "1m", 30 * 24 * time.Hour, false},
		{"uppercase D", "30D", 30 * 24 * time.Hour, false},
		{"uppercase W", "2W", 14 * 24 * time.Hour, false},
		{"uppercase M", "6M", 180 * 24 * time.Hour, false},
		{"empty string", "", 0, true},
		{"invalid format", "abc", 0, true},
		{"missing number", "d", 0, true},
		{"invalid unit", "30x", 0, true},
		{"negative number", "-5d", 0, true},
		{"zero", "0d", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDuration(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseDuration(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if got != tt.expected {
				t.Errorf("parseDuration(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	now := time.Now()

	// Helper to compute timestamp from offset
	ts := func(offset time.Duration) int64 {
		return now.Add(offset).Unix()
	}

	tests := []struct {
		name      string
		timestamp int64
		expected  string
	}{
		// Edge cases
		{"zero timestamp", 0, ""},

		// Today boundary: < 24 hours
		{"just now", ts(0), "today"},
		{"23 hours ago still today", ts(-23 * time.Hour), "today"},

		// Yesterday boundary: 24-48 hours
		{"25 hours ago is yesterday", ts(-25 * time.Hour), "yesterday"},
		{"47 hours ago still yesterday", ts(-47 * time.Hour), "yesterday"},

		// Days ago boundary: 2-6 days
		{"49 hours ago is 2 days ago", ts(-49 * time.Hour), "2 days ago"},
		{"3 days ago", ts(-3 * 24 * time.Hour), "3 days ago"},
		{"6 days ago", ts(-6 * 24 * time.Hour), "6 days ago"},

		// Weeks boundary: 7-29 days
		{"8 days ago is 1 week ago", ts(-8 * 24 * time.Hour), "1 week ago"},
		{"13 days ago still 1 week ago", ts(-13 * 24 * time.Hour), "1 week ago"},
		{"15 days ago is 2 weeks ago", ts(-15 * 24 * time.Hour), "2 weeks ago"},
		{"28 days ago is 4 weeks ago", ts(-28 * 24 * time.Hour), "4 weeks ago"},

		// Months boundary: 30-364 days
		{"35 days ago is 1 month ago", ts(-35 * 24 * time.Hour), "1 month ago"},
		{"65 days ago is 2 months ago", ts(-65 * 24 * time.Hour), "2 months ago"},
		{"95 days ago is 3 months ago", ts(-95 * 24 * time.Hour), "3 months ago"},

		// Years boundary: 365+ days
		{"400 days ago is 1 year ago", ts(-400 * 24 * time.Hour), "1 year ago"},
		{"800 days ago is 2 years ago", ts(-800 * 24 * time.Hour), "2 years ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.timestamp)
			if got != tt.expected {
				t.Errorf("formatAge() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDetermineSkipReason(t *testing.T) {
	tests := []struct {
		name     string
		info     *git.WorktreeInfo
		cwd      string
		force    bool
		expected skipReason
	}{
		{
			name:     "current worktree is protected",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/main"},
			cwd:      "/tmp/workspace/main",
			force:    false,
			expected: skipCurrent,
		},
		{
			name:     "subdirectory of current worktree is protected",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/main"},
			cwd:      "/tmp/workspace/main/src/app",
			force:    false,
			expected: skipCurrent,
		},
		{
			name:     "dirty worktree without force",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/feature", Dirty: true},
			cwd:      "/tmp/workspace/main",
			force:    false,
			expected: skipDirty,
		},
		{
			name:     "dirty worktree with force",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/feature", Dirty: true},
			cwd:      "/tmp/workspace/main",
			force:    true,
			expected: skipNone,
		},
		{
			name:     "locked worktree without force",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/feature", Locked: true},
			cwd:      "/tmp/workspace/main",
			force:    false,
			expected: skipLocked,
		},
		{
			name:     "locked worktree with force",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/feature", Locked: true},
			cwd:      "/tmp/workspace/main",
			force:    true,
			expected: skipNone,
		},
		{
			name:     "unpushed commits without force",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/feature", Ahead: 3},
			cwd:      "/tmp/workspace/main",
			force:    false,
			expected: skipUnpushed,
		},
		{
			name:     "unpushed commits with force",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/feature", Ahead: 3},
			cwd:      "/tmp/workspace/main",
			force:    true,
			expected: skipNone,
		},
		{
			name:     "clean worktree",
			info:     &git.WorktreeInfo{Path: "/tmp/workspace/feature"},
			cwd:      "/tmp/workspace/main",
			force:    false,
			expected: skipNone,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := determineSkipReason(tt.info, tt.cwd, tt.force)
			if got != tt.expected {
				t.Errorf("determineSkipReason() = %q, want %q", got, tt.expected)
			}
		})
	}
}
