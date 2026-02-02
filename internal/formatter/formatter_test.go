package formatter

import (
	"strings"
	"testing"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
)

func TestCurrentMarker(t *testing.T) {
	tests := []struct {
		name      string
		isCurrent bool
		plain     bool
		nerdFonts bool
		wantExact string // Use for plain mode (no ANSI codes)
	}{
		{"not current returns space", false, false, true, " "},
		{"current plain mode returns asterisk", true, true, true, "*"},
		{"current styled no nerd fonts returns asterisk", true, false, false, "*"},
		{"current styled with nerd fonts contains icon", true, false, true, iconCurrent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Global.Plain = tt.plain
			config.Global.NerdFonts = tt.nerdFonts

			got := CurrentMarker(tt.isCurrent)

			// Plain mode returns exact value; styled mode may wrap with ANSI codes
			if tt.plain {
				if got != tt.wantExact {
					t.Errorf("CurrentMarker(%v) = %q, want %q", tt.isCurrent, got, tt.wantExact)
				}
			} else {
				if !strings.Contains(got, tt.wantExact) {
					t.Errorf("CurrentMarker(%v) = %q, want to contain %q", tt.isCurrent, got, tt.wantExact)
				}
			}
		})
	}
}

func TestLock(t *testing.T) {
	tests := []struct {
		name      string
		isLocked  bool
		plain     bool
		nerdFonts bool
		wantExact string // Use for plain mode or when not locked
	}{
		{"not locked returns empty", false, false, true, ""},
		{"locked plain mode", true, true, true, "[locked]"},
		{"locked no nerd fonts", true, false, false, "[locked]"},
		{"locked styled with nerd fonts contains icon", true, false, true, iconLock},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Global.Plain = tt.plain
			config.Global.NerdFonts = tt.nerdFonts

			got := Lock(tt.isLocked)

			// Plain mode or unlocked returns exact value; styled+locked may have ANSI codes
			if tt.plain || !tt.isLocked {
				if got != tt.wantExact {
					t.Errorf("Lock(%v) = %q, want %q", tt.isLocked, got, tt.wantExact)
				}
			} else {
				if !strings.Contains(got, tt.wantExact) {
					t.Errorf("Lock(%v) = %q, want to contain %q", tt.isLocked, got, tt.wantExact)
				}
			}
		})
	}
}

func TestDirty(t *testing.T) {
	tests := []struct {
		name      string
		isDirty   bool
		plain     bool
		nerdFonts bool
		wantExact string // Use for plain mode or when not dirty
	}{
		{"not dirty returns empty", false, false, true, ""},
		{"dirty plain mode", true, true, true, "[dirty]"},
		{"dirty no nerd fonts", true, false, false, "[dirty]"},
		{"dirty styled with nerd fonts contains icon", true, false, true, iconDirty},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Global.Plain = tt.plain
			config.Global.NerdFonts = tt.nerdFonts

			got := Dirty(tt.isDirty)

			// Plain mode or not dirty returns exact value; styled+dirty may have ANSI codes
			if tt.plain || !tt.isDirty {
				if got != tt.wantExact {
					t.Errorf("Dirty(%v) = %q, want %q", tt.isDirty, got, tt.wantExact)
				}
			} else {
				if !strings.Contains(got, tt.wantExact) {
					t.Errorf("Dirty(%v) = %q, want to contain %q", tt.isDirty, got, tt.wantExact)
				}
			}
		})
	}
}

func TestSync(t *testing.T) {
	tests := []struct {
		name        string
		ahead       int
		behind      int
		hasUpstream bool
		plain       bool
		wantEmpty   bool
		wantContain string
	}{
		{"no upstream returns empty", 0, 0, false, false, true, ""},
		{"in sync returns equals", 0, 0, true, false, false, "="},
		{"ahead only plain", 5, 0, true, true, false, "+5"},
		{"behind only plain", 0, 3, true, true, false, "-3"},
		{"ahead and behind plain", 2, 4, true, true, false, "+2"},
		{"ahead styled", 5, 0, true, false, false, "5"},
		{"behind styled", 0, 3, true, false, false, "3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Global.Plain = tt.plain

			got := Sync(tt.ahead, tt.behind, tt.hasUpstream)

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("Sync(%d, %d, %v) = %q, want empty", tt.ahead, tt.behind, tt.hasUpstream, got)
				}
				return
			}

			if !strings.Contains(got, tt.wantContain) {
				t.Errorf("Sync(%d, %d, %v) = %q, want to contain %q", tt.ahead, tt.behind, tt.hasUpstream, got, tt.wantContain)
			}
		})
	}
}

func TestGone(t *testing.T) {
	t.Run("plain mode returns gone", func(t *testing.T) {
		config.Global.Plain = true

		got := Gone()

		if got != "gone" {
			t.Errorf("Gone() = %q, want %q", got, "gone")
		}
	})

	t.Run("styled mode returns x symbol", func(t *testing.T) {
		config.Global.Plain = false
		t.Setenv("GROVE_TEST_COLORS", "true")

		got := Gone()

		if !strings.Contains(got, "×") {
			t.Errorf("Gone() = %q, want to contain ×", got)
		}
	})
}

func TestSubItemPrefix(t *testing.T) {
	t.Run("plain mode returns greater than", func(t *testing.T) {
		config.Global.Plain = true

		got := SubItemPrefix()

		if got != ">" {
			t.Errorf("SubItemPrefix() = %q, want %q", got, ">")
		}
	})

	t.Run("styled mode returns arrow", func(t *testing.T) {
		config.Global.Plain = false

		got := SubItemPrefix()

		if got != "↳" {
			t.Errorf("SubItemPrefix() = %q, want %q", got, "↳")
		}
	})
}

func TestWorktreeRow(t *testing.T) {
	tests := []struct {
		name      string
		info      *git.WorktreeInfo
		isCurrent bool
		padWidth  int
		plain     bool
		wantParts []string
	}{
		{
			name:      "simple clean worktree",
			info:      &git.WorktreeInfo{Branch: "main", Path: "/tmp/main"},
			isCurrent: false,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"main"},
		},
		{
			name:      "current worktree",
			info:      &git.WorktreeInfo{Branch: "main", Path: "/tmp/main"},
			isCurrent: true,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"*", "main"},
		},
		{
			name:      "dirty worktree",
			info:      &git.WorktreeInfo{Branch: "feature", Path: "/tmp/feature", Dirty: true},
			isCurrent: false,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"feature", "[dirty]"},
		},
		{
			name:      "locked worktree",
			info:      &git.WorktreeInfo{Branch: "feature", Path: "/tmp/feature", Locked: true},
			isCurrent: false,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"feature", "[locked]"},
		},
		{
			name:      "gone upstream",
			info:      &git.WorktreeInfo{Branch: "feature", Path: "/tmp/feature", Gone: true},
			isCurrent: false,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"feature", "gone"},
		},
		{
			name:      "ahead of upstream",
			info:      &git.WorktreeInfo{Branch: "feature", Path: "/tmp/feature", Ahead: 3},
			isCurrent: false,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"feature", "+3"},
		},
		{
			name:      "behind upstream",
			info:      &git.WorktreeInfo{Branch: "feature", Path: "/tmp/feature", Behind: 2},
			isCurrent: false,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"feature", "-2"},
		},
		{
			name:      "in sync with upstream",
			info:      &git.WorktreeInfo{Branch: "feature", Path: "/tmp/feature", Upstream: "origin/feature"},
			isCurrent: false,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"feature", "="},
		},
		{
			name:      "multiple indicators",
			info:      &git.WorktreeInfo{Branch: "feature", Path: "/tmp/feature", Dirty: true, Locked: true},
			isCurrent: true,
			padWidth:  0,
			plain:     true,
			wantParts: []string{"*", "feature", "[locked]", "[dirty]"},
		},
		{
			name:      "padded branch name",
			info:      &git.WorktreeInfo{Branch: "main", Path: "/tmp/main"},
			isCurrent: false,
			padWidth:  10,
			plain:     true,
			wantParts: []string{"main"},
		},
	}

	// Additional test for UTF-8 worktree name padding
	t.Run("UTF-8 worktree name padding uses character count", func(t *testing.T) {
		config.Global.Plain = true
		config.Global.NerdFonts = false

		// "日本語" is 3 characters but 9 bytes
		info := &git.WorktreeInfo{Branch: "main", Path: "/tmp/日本語"}
		namePadWidth := 10 // Need to pad to 10 characters

		got := WorktreeRow(info, false, namePadWidth, 0)

		// Should have 7 spaces (10 - 3 chars), not 1 space (10 - 9 bytes)
		expectedSpaces := 7
		nameWithPad := "日本語" + strings.Repeat(" ", expectedSpaces)
		if !strings.Contains(got, nameWithPad) {
			t.Errorf("UTF-8 padding incorrect. Expected name + %d spaces, got: %q", expectedSpaces, got)
		}
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Global.Plain = tt.plain
			config.Global.NerdFonts = false

			got := WorktreeRow(tt.info, tt.isCurrent, tt.padWidth, 0)

			for _, part := range tt.wantParts {
				if !strings.Contains(got, part) {
					t.Errorf("WorktreeRow() = %q, want to contain %q", got, part)
				}
			}
		})
	}
}

func TestWorktreeLabel(t *testing.T) {
	tests := []struct {
		name     string
		info     *git.WorktreeInfo
		expected string
	}{
		{
			name: "formats directory and branch",
			info: &git.WorktreeInfo{
				Path:   "/workspace/pr-1729",
				Branch: "hup-1566-new-training-wizard-page",
			},
			expected: "pr-1729 [hup-1566-new-training-wizard-page]",
		},
		{
			name: "handles detached worktree",
			info: &git.WorktreeInfo{
				Path:     "/workspace/experiment",
				Branch:   "abc1234",
				Detached: true,
			},
			expected: "experiment [abc1234]",
		},
		{
			name: "handles nested path",
			info: &git.WorktreeInfo{
				Path:   "/home/user/projects/grove/worktrees/feature-auth",
				Branch: "feat/authentication",
			},
			expected: "feature-auth [feat/authentication]",
		},
		{
			name: "handles empty path gracefully",
			info: &git.WorktreeInfo{
				Path:   "",
				Branch: "main",
			},
			expected: " [main]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := WorktreeLabel(tt.info)
			if got != tt.expected {
				t.Errorf("WorktreeLabel() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestVerboseSubItems(t *testing.T) {
	t.Run("includes path", func(t *testing.T) {
		config.Global.Plain = true
		info := &git.WorktreeInfo{
			Branch: "main",
			Path:   "/tmp/workspace/main",
		}

		items := VerboseSubItems(info)

		if len(items) < 1 {
			t.Fatal("expected at least 1 item")
		}

		found := false
		for _, item := range items {
			if strings.Contains(item, "path:") && strings.Contains(item, "/tmp/workspace/main") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("VerboseSubItems() missing path item, got %v", items)
		}
	})

	t.Run("includes upstream when set", func(t *testing.T) {
		config.Global.Plain = true
		info := &git.WorktreeInfo{
			Branch:   "feature",
			Path:     "/tmp/workspace/feature",
			Upstream: "origin/feature",
		}

		items := VerboseSubItems(info)

		found := false
		for _, item := range items {
			if strings.Contains(item, "upstream:") && strings.Contains(item, "origin/feature") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("VerboseSubItems() missing upstream item, got %v", items)
		}
	})

	t.Run("includes lock reason when locked with reason", func(t *testing.T) {
		config.Global.Plain = true
		info := &git.WorktreeInfo{
			Branch:     "feature",
			Path:       "/tmp/workspace/feature",
			Locked:     true,
			LockReason: "WIP - do not remove",
		}

		items := VerboseSubItems(info)

		found := false
		for _, item := range items {
			if strings.Contains(item, "lock reason:") && strings.Contains(item, "WIP - do not remove") {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("VerboseSubItems() missing lock reason item, got %v", items)
		}
	})

	t.Run("excludes lock reason when not locked", func(t *testing.T) {
		config.Global.Plain = true
		info := &git.WorktreeInfo{
			Branch:     "feature",
			Path:       "/tmp/workspace/feature",
			Locked:     false,
			LockReason: "should not appear",
		}

		items := VerboseSubItems(info)

		for _, item := range items {
			if strings.Contains(item, "lock reason:") {
				t.Errorf("VerboseSubItems() should not include lock reason when not locked, got %v", items)
			}
		}
	})

	t.Run("uses correct prefix", func(t *testing.T) {
		config.Global.Plain = true
		info := &git.WorktreeInfo{
			Branch: "main",
			Path:   "/tmp/main",
		}

		items := VerboseSubItems(info)

		if len(items) < 1 {
			t.Fatal("expected at least 1 item")
		}

		if !strings.Contains(items[0], ">") {
			t.Errorf("VerboseSubItems() should use > prefix in plain mode, got %q", items[0])
		}
	})

	t.Run("uses styled prefix", func(t *testing.T) {
		config.Global.Plain = false
		info := &git.WorktreeInfo{
			Branch: "main",
			Path:   "/tmp/main",
		}

		items := VerboseSubItems(info)

		if len(items) < 1 {
			t.Fatal("expected at least 1 item")
		}

		if !strings.Contains(items[0], "↳") {
			t.Errorf("VerboseSubItems() should use ↳ prefix in styled mode, got %q", items[0])
		}
	})
}

func TestUseAsciiIcons(t *testing.T) {
	tests := []struct {
		name      string
		plain     bool
		nerdFonts bool
		want      bool
	}{
		{"plain mode uses ascii", true, true, true},
		{"plain mode with no nerd fonts uses ascii", true, false, true},
		{"styled with nerd fonts uses icons", false, true, false},
		{"styled without nerd fonts uses ascii", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.Global.Plain = tt.plain
			config.Global.NerdFonts = tt.nerdFonts

			got := useAsciiIcons()

			if got != tt.want {
				t.Errorf("useAsciiIcons() = %v, want %v", got, tt.want)
			}
		})
	}
}
