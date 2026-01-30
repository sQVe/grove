package commands

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutil"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewFetchCmd(t *testing.T) {
	cmd := NewFetchCmd()

	if cmd.Use != "fetch" {
		t.Errorf("expected Use 'fetch', got '%s'", cmd.Use)
	}

	if cmd.Short != "Fetch all remotes and show changes" {
		t.Errorf("expected Short 'Fetch all remotes and show changes', got '%s'", cmd.Short)
	}

	if cmd.Args == nil {
		t.Fatal("expected Args to be set")
	}

	if cmd.ValidArgsFunction == nil {
		t.Fatal("expected ValidArgsFunction to be set")
	}
}

func TestFetchCmdArgsValidation(t *testing.T) {
	cmd := NewFetchCmd()

	err := cmd.Args(cmd, []string{})
	if err != nil {
		t.Errorf("expected no args to be valid, got error: %v", err)
	}

	err = cmd.Args(cmd, []string{"extra"})
	if err == nil {
		t.Error("expected error for extra args")
	}
}

func TestFetchCmdValidArgsFunction(t *testing.T) {
	cmd := NewFetchCmd()

	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")

	if completions != nil {
		t.Errorf("expected nil completions, got %v", completions)
	}

	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}

func TestRunFetch_NotInWorkspace(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tmpDir := testutil.TempDir(t)
	testutil.Chdir(t, tmpDir)

	err := runFetch(false, false)
	if err == nil {
		t.Error("expected error for non-workspace directory")
	}
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got: %v", err)
	}
}

func TestStripRefPrefix(t *testing.T) {
	tests := []struct {
		name     string
		refName  string
		remote   string
		expected string
	}{
		{"origin main", "refs/remotes/origin/main", "origin", "main"},
		{"origin feature branch", "refs/remotes/origin/feature/test", "origin", "feature/test"},
		{"upstream main", "refs/remotes/upstream/main", "upstream", "main"},
		{"no prefix", "main", "origin", "main"},
		{"wrong remote", "refs/remotes/upstream/main", "origin", "refs/remotes/upstream/main"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripRefPrefix(tt.refName, tt.remote)
			if got != tt.expected {
				t.Errorf("stripRefPrefix(%q, %q) = %q, want %q", tt.refName, tt.remote, got, tt.expected)
			}
		})
	}
}

func TestRemoteResult(t *testing.T) {
	t.Run("result with changes", func(t *testing.T) {
		result := remoteResult{
			Remote: "origin",
			Changes: []git.RefChange{
				{RefName: "refs/remotes/origin/main", Type: git.Updated, OldHash: "abc", NewHash: "def"},
			},
			Error: nil,
		}

		if result.Remote != "origin" {
			t.Errorf("expected Remote 'origin', got '%s'", result.Remote)
		}
		if len(result.Changes) != 1 {
			t.Errorf("expected 1 change, got %d", len(result.Changes))
		}
		if result.Error != nil {
			t.Errorf("expected no error, got %v", result.Error)
		}
	})

	t.Run("result with error", func(t *testing.T) {
		result := remoteResult{
			Remote:  "upstream",
			Changes: nil,
			Error:   errors.New("fetch failed"),
		}

		if result.Remote != "upstream" {
			t.Errorf("expected Remote 'upstream', got '%s'", result.Remote)
		}
		if result.Changes != nil {
			t.Errorf("expected nil changes, got %v", result.Changes)
		}
		if result.Error == nil {
			t.Error("expected error, got nil")
		}
	})
}

func TestNewFetchCmd_Flags(t *testing.T) {
	cmd := NewFetchCmd()

	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Fatal("expected --json flag to exist")
	}
	if jsonFlag.DefValue != "false" {
		t.Errorf("expected --json default value 'false', got '%s'", jsonFlag.DefValue)
	}

	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("expected --verbose flag to exist")
	}
	if verboseFlag.Shorthand != "v" {
		t.Errorf("expected --verbose shorthand 'v', got '%s'", verboseFlag.Shorthand)
	}
}

func TestFetchChangeJSON(t *testing.T) {
	tests := []struct {
		name     string
		change   fetchChangeJSON
		expected map[string]interface{}
	}{
		{
			name: "full change with commit count",
			change: fetchChangeJSON{
				Remote:      "origin",
				RefName:     "main",
				Type:        "Updated",
				OldHash:     "abc123",
				NewHash:     "def456",
				CommitCount: 5,
			},
			expected: map[string]interface{}{
				"remote":       "origin",
				"ref":          "main",
				"type":         "Updated",
				"old_hash":     "abc123",
				"new_hash":     "def456",
				"commit_count": float64(5),
			},
		},
		{
			name: "new ref without old hash",
			change: fetchChangeJSON{
				Remote:  "origin",
				RefName: "feature",
				Type:    "New",
				NewHash: "xyz789",
			},
			expected: map[string]interface{}{
				"remote":   "origin",
				"ref":      "feature",
				"type":     "New",
				"new_hash": "xyz789",
			},
		},
		{
			name: "pruned ref without new hash",
			change: fetchChangeJSON{
				Remote:  "upstream",
				RefName: "old-branch",
				Type:    "Pruned",
				OldHash: "old123",
			},
			expected: map[string]interface{}{
				"remote":   "upstream",
				"ref":      "old-branch",
				"type":     "Pruned",
				"old_hash": "old123",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.change)
			if err != nil {
				t.Fatalf("failed to marshal JSON: %v", err)
			}

			var result map[string]interface{}
			if err := json.Unmarshal(data, &result); err != nil {
				t.Fatalf("failed to unmarshal JSON: %v", err)
			}

			for key, expectedValue := range tt.expected {
				if result[key] != expectedValue {
					t.Errorf("expected %s=%v, got %v", key, expectedValue, result[key])
				}
			}

			for key := range result {
				if _, ok := tt.expected[key]; !ok {
					t.Errorf("unexpected field in JSON: %s=%v", key, result[key])
				}
			}
		})
	}
}
