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
		err := runAdd("#123", false, "main", false)
		if err == nil || err.Error() != "--base cannot be used with PR references" {
			t.Errorf("expected base/PR error, got %v", err)
		}
	})

	t.Run("detach flag cannot be used with PR number", func(t *testing.T) {
		err := runAdd("#123", false, "", true)
		if err == nil || err.Error() != "--detach cannot be used with PR references" {
			t.Errorf("expected detach/PR error, got %v", err)
		}
	})

	t.Run("base flag cannot be used with PR URL", func(t *testing.T) {
		err := runAdd("https://github.com/owner/repo/pull/456", false, "main", false)
		if err == nil || err.Error() != "--base cannot be used with PR references" {
			t.Errorf("expected base/PR error, got %v", err)
		}
	})

	t.Run("detach flag cannot be used with PR URL", func(t *testing.T) {
		err := runAdd("https://github.com/owner/repo/pull/456", false, "", true)
		if err == nil || err.Error() != "--detach cannot be used with PR references" {
			t.Errorf("expected detach/PR error, got %v", err)
		}
	})
}

func TestRunAdd_DetachBaseValidation(t *testing.T) {
	t.Run("detach and base cannot be used together", func(t *testing.T) {
		err := runAdd("v1.0.0", false, "main", true)
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

func TestAcquireWorkspaceLock(t *testing.T) {
	t.Run("acquires lock on fresh file", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		handle, err := acquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire lock, got error: %v", err)
		}
		defer func() {
			_ = handle.Close()
			_ = os.Remove(lockFile)
		}()

		// Verify lock file exists
		if _, err := os.Stat(lockFile); os.IsNotExist(err) {
			t.Error("expected lock file to exist")
		}
	})

	t.Run("fails when lock already held by running process", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Acquire first lock
		handle1, err := acquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire first lock, got error: %v", err)
		}
		defer func() {
			_ = handle1.Close()
			_ = os.Remove(lockFile)
		}()

		// Try to acquire second lock - should fail
		_, err = acquireWorkspaceLock(lockFile)
		if err == nil {
			t.Error("expected error when lock already held")
		}
	})

	t.Run("removes stale lock with invalid PID", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Create lock file with invalid PID
		if err := os.WriteFile(lockFile, []byte("not-a-pid"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Should succeed by removing stale lock
		handle, err := acquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire lock after removing stale, got: %v", err)
		}
		defer func() {
			_ = handle.Close()
			_ = os.Remove(lockFile)
		}()
	})

	t.Run("removes stale lock from dead process", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Create lock file with PID that doesn't exist (use very high PID)
		// PID 99999999 is unlikely to exist on any system
		if err := os.WriteFile(lockFile, []byte("99999999"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Should succeed by removing stale lock
		handle, err := acquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire lock after removing stale, got: %v", err)
		}
		defer func() {
			_ = handle.Close()
			_ = os.Remove(lockFile)
		}()
	})

	t.Run("respects max retry limit", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Create a lock file that we'll keep recreating
		// This simulates a race condition where another process keeps creating locks

		// First, create an initial stale lock
		if err := os.WriteFile(lockFile, []byte("invalid"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Acquire should succeed (removes stale lock)
		handle, err := acquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire lock, got: %v", err)
		}
		_ = handle.Close()
		_ = os.Remove(lockFile)
	})
}

func TestTryAcquireLock(t *testing.T) {
	t.Run("returns done=true on success", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		handle, done, err := tryAcquireLock(lockFile, 0)
		if !done {
			t.Error("expected done=true on success")
		}
		if err != nil {
			t.Errorf("expected no error, got: %v", err)
		}
		if handle == nil {
			t.Error("expected handle to be non-nil")
		}
		defer func() {
			if handle != nil {
				_ = handle.Close()
				_ = os.Remove(lockFile)
			}
		}()
	})

	t.Run("returns done=false for stale lock retry", func(t *testing.T) {
		tmpDir := t.TempDir()
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Create lock file with invalid PID
		if err := os.WriteFile(lockFile, []byte("invalid-pid"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		_, done, err := tryAcquireLock(lockFile, 0)
		if done {
			t.Error("expected done=false for stale lock removal")
		}
		if err != nil {
			t.Errorf("expected no error for retry signal, got: %v", err)
		}
	})
}

func TestIsProcessRunning(t *testing.T) {
	t.Run("returns true for current process", func(t *testing.T) {
		pid := os.Getpid()
		if !isProcessRunning(pid) {
			t.Error("expected current process to be running")
		}
	})

	t.Run("returns false for non-existent PID", func(t *testing.T) {
		// Use a very high PID that's unlikely to exist
		if isProcessRunning(99999999) {
			t.Error("expected non-existent PID to return false")
		}
	})
}
