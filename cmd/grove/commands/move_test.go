package commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutil"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewMoveCmd(t *testing.T) {
	cmd := NewMoveCmd()
	if cmd.Use != "move <worktree> <new-branch>" {
		t.Errorf("expected Use to be 'move <worktree> <new-branch>', got %q", cmd.Use)
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

	tmpDir := testutil.TempDir(t)
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

	// When already has first arg, should not complete further (second arg is free-form)
	_, directive := cmd.ValidArgsFunction(cmd, []string{"existing"}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp when first arg present, got %v", directive)
	}
}

func TestRunMove_CurrentWorktreeHint(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	_ = os.Chdir(mainPath)

	err := runMove("main", "new-branch")
	if err == nil {
		t.Fatal("expected error for current worktree")
	}

	if !strings.Contains(err.Error(), "grove switch") {
		t.Errorf("expected error to contain 'grove switch' hint, got: %v", err)
	}
}
