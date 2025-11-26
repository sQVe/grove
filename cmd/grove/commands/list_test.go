package commands

import (
	"errors"
	"os"
	"testing"

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
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag")
	}
}

func TestRunList(t *testing.T) {
	t.Run("returns error when not in workspace", func(t *testing.T) {
		// Save and restore cwd
		origDir, _ := os.Getwd()
		defer func() { _ = os.Chdir(origDir) }()

		tmpDir := t.TempDir()
		_ = os.Chdir(tmpDir)

		err := runList(false, false, false)
		if err == nil {
			t.Error("expected error for non-workspace directory")
		}
		if !errors.Is(err, workspace.ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace, got: %v", err)
		}
	})
}
