package commands

import (
	"errors"
	"os"
	"testing"

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
}

func TestRunPrune(t *testing.T) {
	t.Run("returns error when not in workspace", func(t *testing.T) {
		// Save and restore cwd
		origDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(origDir) }()

		tmpDir := t.TempDir()
		_ = os.Chdir(tmpDir)

		err := runPrune(false, false)
		if err == nil {
			t.Error("expected error for non-workspace directory")
		}
		if !errors.Is(err, workspace.ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace, got: %v", err)
		}
	})
}
