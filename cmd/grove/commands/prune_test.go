package commands

import (
	"errors"
	"os"
	"testing"
	"time"

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
	if cmd.Flags().Lookup("force") == nil {
		t.Error("expected --force flag")
	}
	if cmd.Flags().Lookup("stale") == nil {
		t.Error("expected --stale flag")
	}
	if cmd.Flags().Lookup("merged") == nil {
		t.Error("expected --merged flag")
	}
}

func TestRunPrune(t *testing.T) {
	t.Run("returns error when not in workspace", func(t *testing.T) {
		// Save and restore cwd
		origDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(origDir) }()

		tmpDir := t.TempDir()
		_ = os.Chdir(tmpDir)

		err := runPrune(false, false, "", false)
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

	tests := []struct {
		name      string
		timestamp int64
		expected  string
	}{
		{"zero timestamp", 0, ""},
		{"today", now.Unix(), "today"},
		{"yesterday", now.Add(-25 * time.Hour).Unix(), "yesterday"},
		{"3 days ago", now.Add(-3 * 24 * time.Hour).Unix(), "3 days ago"},
		{"1 week ago", now.Add(-8 * 24 * time.Hour).Unix(), "1 week ago"},
		{"2 weeks ago", now.Add(-15 * 24 * time.Hour).Unix(), "2 weeks ago"},
		{"1 month ago", now.Add(-35 * 24 * time.Hour).Unix(), "1 month ago"},
		{"3 months ago", now.Add(-95 * 24 * time.Hour).Unix(), "3 months ago"},
		{"1 year ago", now.Add(-400 * 24 * time.Hour).Unix(), "1 year ago"},
		{"2 years ago", now.Add(-800 * 24 * time.Hour).Unix(), "2 years ago"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.timestamp)
			if got != tt.expected {
				t.Errorf("formatAge(%d) = %q, want %q", tt.timestamp, got, tt.expected)
			}
		})
	}
}
