package commands

import (
	"errors"
	"os"
	"testing"

	"github.com/sqve/grove/internal/workspace"
)

func TestNewStatusCmd(t *testing.T) {
	cmd := NewStatusCmd()

	if cmd.Use != "status" {
		t.Errorf("expected Use 'status', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag")
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Error("expected --json flag")
	}
}

func TestNewStatusCmd_FlagDefaults(t *testing.T) {
	cmd := NewStatusCmd()

	verbose, err := cmd.Flags().GetBool("verbose")
	if err != nil {
		t.Fatalf("failed to get verbose flag: %v", err)
	}
	if verbose {
		t.Error("expected verbose to default to false")
	}

	jsonFlag, err := cmd.Flags().GetBool("json")
	if err != nil {
		t.Fatalf("failed to get json flag: %v", err)
	}
	if jsonFlag {
		t.Error("expected json to default to false")
	}
}

func TestNewStatusCmd_ValidArgsFunction(t *testing.T) {
	cmd := NewStatusCmd()

	completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
	if completions != nil {
		t.Errorf("expected nil completions, got %v", completions)
	}
	if directive != 4 { // cobra.ShellCompDirectiveNoFileComp
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}

func TestRunStatus(t *testing.T) {
	t.Run("returns error when not in workspace", func(t *testing.T) {
		origDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(origDir) }()

		tmpDir := t.TempDir()
		_ = os.Chdir(tmpDir)

		err := runStatus(false, false)
		if err == nil {
			t.Error("expected error for non-workspace directory")
		}
		if !errors.Is(err, workspace.ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace, got: %v", err)
		}
	})
}
