package commands

import (
	"errors"
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutil"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()

	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("fast") == nil {
		t.Error("expected --fast flag")
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Error("expected --json flag")
	}
	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("expected --verbose flag")
	}
	if verboseFlag.Shorthand != "v" {
		t.Errorf("expected verbose shorthand 'v', got %q", verboseFlag.Shorthand)
	}
	if cmd.Flags().Lookup("filter") == nil {
		t.Error("expected --filter flag")
	}
}

func TestRunList(t *testing.T) {
	t.Run("returns error when not in workspace", func(t *testing.T) {
		// Save and restore cwd
		defer testutil.SaveCwd(t)()

		tmpDir := testutil.TempDir(t)
		testutil.Chdir(t, tmpDir)

		err := runList(false, false, false, "")
		if err == nil {
			t.Error("expected error for non-workspace directory")
		}
		if !errors.Is(err, workspace.ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace, got: %v", err)
		}
	})
}

func TestParseFilters(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"single filter", "dirty", []string{"dirty"}},
		{"multiple filters", "dirty,locked", []string{"dirty", "locked"}},
		{"with spaces", " dirty , locked ", []string{"dirty", "locked"}},
		{"empty string", "", nil},
		{"uppercase converted", "DIRTY,LOCKED", []string{"dirty", "locked"}},
		{"mixed case", "Dirty,AHEAD,behind", []string{"dirty", "ahead", "behind"}},
		{"trailing comma", "dirty,", []string{"dirty"}},
		{"leading comma", ",dirty", []string{"dirty"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseFilters(tt.input)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("parseFilters(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestMatchesAnyFilter(t *testing.T) {
	tests := []struct {
		name    string
		info    *git.WorktreeInfo
		filters []string
		want    bool
	}{
		{"dirty matches dirty", &git.WorktreeInfo{Dirty: true}, []string{"dirty"}, true},
		{"clean does not match dirty", &git.WorktreeInfo{Dirty: false}, []string{"dirty"}, false},
		{"ahead matches ahead", &git.WorktreeInfo{Ahead: 2}, []string{"ahead"}, true},
		{"zero ahead does not match", &git.WorktreeInfo{Ahead: 0}, []string{"ahead"}, false},
		{"behind matches behind", &git.WorktreeInfo{Behind: 1}, []string{"behind"}, true},
		{"zero behind does not match", &git.WorktreeInfo{Behind: 0}, []string{"behind"}, false},
		{"gone matches gone", &git.WorktreeInfo{Gone: true}, []string{"gone"}, true},
		{"not gone does not match", &git.WorktreeInfo{Gone: false}, []string{"gone"}, false},
		{"locked matches locked", &git.WorktreeInfo{Locked: true}, []string{"locked"}, true},
		{"not locked does not match", &git.WorktreeInfo{Locked: false}, []string{"locked"}, false},
		{"OR logic: dirty or locked, dirty true", &git.WorktreeInfo{Dirty: true, Locked: false}, []string{"dirty", "locked"}, true},
		{"OR logic: dirty or locked, locked true", &git.WorktreeInfo{Dirty: false, Locked: true}, []string{"dirty", "locked"}, true},
		{"OR logic: both true", &git.WorktreeInfo{Dirty: true, Locked: true}, []string{"dirty", "locked"}, true},
		{"OR logic: neither matches", &git.WorktreeInfo{Dirty: false, Locked: false}, []string{"dirty", "locked"}, false},
		{"empty filters matches nothing", &git.WorktreeInfo{Dirty: true}, []string{}, false},
		{"unknown filter ignored", &git.WorktreeInfo{Dirty: true}, []string{"unknown"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAnyFilter(tt.info, tt.filters)
			if got != tt.want {
				t.Errorf("matchesAnyFilter(%+v, %v) = %v, want %v", tt.info, tt.filters, got, tt.want)
			}
		})
	}
}

func TestFilterWorktrees(t *testing.T) {
	infos := []*git.WorktreeInfo{
		{Branch: "main", Dirty: false, Locked: true},
		{Branch: "feature", Dirty: true, Locked: false},
		{Branch: "old", Gone: true, Locked: false},
		{Branch: "clean", Dirty: false, Locked: false},
	}

	t.Run("empty filter returns all", func(t *testing.T) {
		got := filterWorktrees(infos, "")
		if len(got) != 4 {
			t.Errorf("expected 4 worktrees, got %d", len(got))
		}
	})

	t.Run("filter dirty", func(t *testing.T) {
		got := filterWorktrees(infos, "dirty")
		if len(got) != 1 || got[0].Branch != "feature" {
			t.Errorf("expected [feature], got %v", got)
		}
	})

	t.Run("filter locked", func(t *testing.T) {
		got := filterWorktrees(infos, "locked")
		if len(got) != 1 || got[0].Branch != "main" {
			t.Errorf("expected [main], got %v", got)
		}
	})

	t.Run("filter gone", func(t *testing.T) {
		got := filterWorktrees(infos, "gone")
		if len(got) != 1 || got[0].Branch != "old" {
			t.Errorf("expected [old], got %v", got)
		}
	})

	t.Run("filter dirty,locked OR logic", func(t *testing.T) {
		got := filterWorktrees(infos, "dirty,locked")
		if len(got) != 2 {
			t.Errorf("expected 2 worktrees, got %d", len(got))
		}
		branches := make(map[string]bool)
		for _, info := range got {
			branches[info.Branch] = true
		}
		if !branches["main"] || !branches["feature"] {
			t.Errorf("expected main and feature, got %v", got)
		}
	})
}

func TestCompleteFilterValues(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		wantLen    int
		wantFirst  string
	}{
		{
			name:       "empty returns all filters",
			toComplete: "",
			wantLen:    5,
			wantFirst:  "dirty",
		},
		{
			name:       "partial match d",
			toComplete: "d",
			wantLen:    1,
			wantFirst:  "dirty",
		},
		{
			name:       "partial match a",
			toComplete: "a",
			wantLen:    1,
			wantFirst:  "ahead",
		},
		{
			name:       "after comma returns remaining filters",
			toComplete: "dirty,",
			wantLen:    4,
			wantFirst:  "dirty,ahead",
		},
		{
			name:       "partial after comma",
			toComplete: "dirty,l",
			wantLen:    1,
			wantFirst:  "dirty,locked",
		},
		{
			name:       "multiple selected excludes them",
			toComplete: "dirty,ahead,",
			wantLen:    3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			completions, directive := completeFilterValues(nil, nil, tt.toComplete)
			if len(completions) != tt.wantLen {
				t.Errorf("completeFilterValues(%q) returned %d completions, want %d: %v",
					tt.toComplete, len(completions), tt.wantLen, completions)
			}
			if tt.wantFirst != "" && len(completions) > 0 && completions[0] != tt.wantFirst {
				t.Errorf("completeFilterValues(%q) first = %q, want %q",
					tt.toComplete, completions[0], tt.wantFirst)
			}
			wantDirective := cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
			if directive != wantDirective {
				t.Errorf("completeFilterValues(%q) directive = %v, want %v", tt.toComplete, directive, wantDirective)
			}
		})
	}
}
