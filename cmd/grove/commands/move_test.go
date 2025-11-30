package commands

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/workspace"
)

func TestNewMoveCmd(t *testing.T) {
	cmd := NewMoveCmd()
	if cmd.Use != "move <old-branch> <new-branch>" {
		t.Errorf("expected Use to be 'move <old-branch> <new-branch>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestNewMoveCmd_RequiresTwoArgs(t *testing.T) {
	cmd := NewMoveCmd()

	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("expected error when no arguments provided")
	}

	err = cmd.Args(cmd, []string{"one"})
	if err == nil {
		t.Error("expected error when only one argument provided")
	}

	err = cmd.Args(cmd, []string{"one", "two", "three"})
	if err == nil {
		t.Error("expected error when too many arguments provided")
	}

	err = cmd.Args(cmd, []string{"old", "new"})
	if err != nil {
		t.Errorf("unexpected error with two arguments: %v", err)
	}
}

func TestRunMove_NotInWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runMove("old-branch", "new-branch")
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunMove_SameBranchName(t *testing.T) {
	err := runMove("same-branch", "same-branch")
	if err == nil {
		t.Fatal("expected error for same branch names")
	}
	if !strings.Contains(err.Error(), "same") {
		t.Errorf("expected error about same branch names, got %v", err)
	}
}

func TestNewMoveCmd_ValidArgsFunction(t *testing.T) {
	cmd := NewMoveCmd()

	if cmd.ValidArgsFunction == nil {
		t.Error("expected ValidArgsFunction to be set")
	}
}
