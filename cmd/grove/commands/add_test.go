package commands

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewAddCmd(t *testing.T) {
	cmd := NewAddCmd()
	if cmd.Use != "add <branch|#PR|PR-URL|ref>" {
		t.Errorf("expected Use to be 'add <branch|#PR|PR-URL|ref>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestNewAddCmd_HasSwitchFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("switch")
	if flag == nil {
		t.Fatal("expected --switch flag to exist")
	}
	if flag.Shorthand != "s" {
		t.Errorf("expected shorthand 's', got %q", flag.Shorthand)
	}
}

func TestNewAddCmd_HasBaseFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("base")
	if flag == nil {
		t.Fatal("expected --base flag to exist")
	}
}

func TestNewAddCmd_HasDetachFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("detach")
	if flag == nil {
		t.Fatal("expected --detach flag to exist")
	}
}

func TestRunAdd_NotInWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runAdd("feature-test", false, "", false)
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestFindSourceWorktree(t *testing.T) {
	t.Run("returns empty at workspace root", func(t *testing.T) {
		workspaceRoot := t.TempDir()

		result := findSourceWorktree(workspaceRoot, workspaceRoot)
		if result != "" {
			t.Errorf("expected empty string at workspace root, got %q", result)
		}
	})

	t.Run("returns worktree path when in worktree", func(t *testing.T) {
		workspaceRoot := t.TempDir()

		// Create a fake worktree directory with .git file
		worktreeDir := filepath.Join(workspaceRoot, "main")
		if err := os.MkdirAll(worktreeDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		gitFile := filepath.Join(worktreeDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../.bare"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(worktreeDir, workspaceRoot)
		if result != worktreeDir {
			t.Errorf("expected %q, got %q", worktreeDir, result)
		}
	})

	t.Run("returns worktree from subdirectory", func(t *testing.T) {
		workspaceRoot := t.TempDir()

		// Create worktree with subdirectory
		worktreeDir := filepath.Join(workspaceRoot, "main")
		subDir := filepath.Join(worktreeDir, "src", "pkg")
		if err := os.MkdirAll(subDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		gitFile := filepath.Join(worktreeDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../.bare"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(subDir, workspaceRoot)
		if result != worktreeDir {
			t.Errorf("expected %q, got %q", worktreeDir, result)
		}
	})

	t.Run("returns empty when not in worktree", func(t *testing.T) {
		workspaceRoot := t.TempDir()

		// Create a directory that's not a worktree
		otherDir := filepath.Join(workspaceRoot, "other")
		if err := os.MkdirAll(otherDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(otherDir, workspaceRoot)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}
