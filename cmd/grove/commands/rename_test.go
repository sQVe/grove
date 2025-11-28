package commands

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/workspace"
)

func TestNewRenameCmd(t *testing.T) {
	cmd := NewRenameCmd()
	if cmd.Use != "rename <old-branch> <new-branch>" {
		t.Errorf("expected Use to be 'rename <old-branch> <new-branch>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestRunRename_NotInWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runRename("old-branch", "new-branch")
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunRename_SameBranchName(t *testing.T) {
	err := runRename("same-branch", "same-branch")
	if err == nil {
		t.Fatal("expected error for same branch names")
	}
	if !strings.Contains(err.Error(), "same") {
		t.Errorf("expected error about same branch names, got %v", err)
	}
}
