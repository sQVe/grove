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

	// Verify command structure
	if cmd.Use != "add <branch|#PR|PR-URL|ref>" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Verify required flags exist with correct configuration
	flags := []struct {
		name      string
		shorthand string
	}{
		{"switch", "s"},
		{"base", ""},
		{"detach", "d"},
		{"name", ""},
	}

	for _, f := range flags {
		flag := cmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("expected --%s flag to exist", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("--%s: expected shorthand %q, got %q", f.name, f.shorthand, flag.Shorthand)
		}
	}

	// Verify ValidArgsFunction is set
	if cmd.ValidArgsFunction == nil {
		t.Error("expected ValidArgsFunction to be set")
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
	if flag.DefValue != "" {
		t.Errorf("expected default value '', got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %q", flag.Value.Type())
	}
}

func TestNewAddCmd_HasDetachFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("detach")
	if flag == nil {
		t.Fatal("expected --detach flag to exist")
	}
	if flag.Shorthand != "d" {
		t.Errorf("expected shorthand 'd', got %q", flag.Shorthand)
	}
	if flag.DefValue != "false" {
		t.Errorf("expected default value 'false', got %q", flag.DefValue)
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("expected bool type, got %q", flag.Value.Type())
	}
}

func TestNewAddCmd_HasNameFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("name")
	if flag == nil {
		t.Fatal("expected --name flag to exist")
	}
	if flag.DefValue != "" {
		t.Errorf("expected default value '', got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %q", flag.Value.Type())
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

	err = runAdd("feature-test", false, "", "", false)
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunAdd_PRValidation(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	t.Run("base flag cannot be used with PR number", func(t *testing.T) {
		err := runAdd("#123", false, "main", "", false)
		if err == nil || err.Error() != "--base cannot be used with PR references" {
			t.Errorf("expected base/PR error, got %v", err)
		}
	})

	t.Run("detach flag cannot be used with PR number", func(t *testing.T) {
		err := runAdd("#123", false, "", "", true)
		if err == nil || err.Error() != "--detach cannot be used with PR references" {
			t.Errorf("expected detach/PR error, got %v", err)
		}
	})

	t.Run("base flag cannot be used with PR URL", func(t *testing.T) {
		err := runAdd("https://github.com/owner/repo/pull/456", false, "main", "", false)
		if err == nil || err.Error() != "--base cannot be used with PR references" {
			t.Errorf("expected base/PR error, got %v", err)
		}
	})

	t.Run("detach flag cannot be used with PR URL", func(t *testing.T) {
		err := runAdd("https://github.com/owner/repo/pull/456", false, "", "", true)
		if err == nil || err.Error() != "--detach cannot be used with PR references" {
			t.Errorf("expected detach/PR error, got %v", err)
		}
	})
}

func TestRunAdd_DetachBaseValidation(t *testing.T) {
	t.Run("detach and base cannot be used together", func(t *testing.T) {
		err := runAdd("v1.0.0", false, "main", "", true)
		if err == nil || err.Error() != "--detach and --base cannot be used together" {
			t.Errorf("expected detach/base error, got %v", err)
		}
	})
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
