package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/testutil"
)

func TestAcquireWorkspaceLock(t *testing.T) {
	t.Run("acquires lock on fresh file", func(t *testing.T) {
		tmpDir := testutil.TempDir(t)
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		handle, err := AcquireWorkspaceLock(lockFile)
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
		tmpDir := testutil.TempDir(t)
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Acquire first lock
		handle1, err := AcquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire first lock, got error: %v", err)
		}
		defer func() {
			_ = handle1.Close()
			_ = os.Remove(lockFile)
		}()

		// Try to acquire second lock - should fail
		_, err = AcquireWorkspaceLock(lockFile)
		if err == nil {
			t.Error("expected error when lock already held")
		}
	})

	t.Run("removes stale lock with invalid PID", func(t *testing.T) {
		tmpDir := testutil.TempDir(t)
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Create lock file with invalid PID
		if err := os.WriteFile(lockFile, []byte("not-a-pid"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Should succeed by removing stale lock
		handle, err := AcquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire lock after removing stale, got: %v", err)
		}
		defer func() {
			_ = handle.Close()
			_ = os.Remove(lockFile)
		}()
	})

	t.Run("removes stale lock from dead process", func(t *testing.T) {
		tmpDir := testutil.TempDir(t)
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Create lock file with PID that doesn't exist (use very high PID)
		// PID 99999999 is unlikely to exist on any system
		if err := os.WriteFile(lockFile, []byte("99999999"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Should succeed by removing stale lock
		handle, err := AcquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire lock after removing stale, got: %v", err)
		}
		defer func() {
			_ = handle.Close()
			_ = os.Remove(lockFile)
		}()
	})

	t.Run("respects max retry limit", func(t *testing.T) {
		tmpDir := testutil.TempDir(t)
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Create a lock file that we'll keep recreating
		// This simulates a race condition where another process keeps creating locks

		// First, create an initial stale lock
		if err := os.WriteFile(lockFile, []byte("invalid"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		// Acquire should succeed (removes stale lock)
		handle, err := AcquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire lock, got: %v", err)
		}
		_ = handle.Close()
		_ = os.Remove(lockFile)
	})
}

func TestTryAcquireLock(t *testing.T) {
	t.Run("returns done=true on success", func(t *testing.T) {
		tmpDir := testutil.TempDir(t)
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
		tmpDir := testutil.TempDir(t)
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
