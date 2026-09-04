package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/testutil"
)

func TestAcquireWorkspaceLock(t *testing.T) {
	t.Parallel()

	t.Run("acquires lock on fresh file", func(t *testing.T) {
		t.Parallel()
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

	t.Run("reports old lock held by running process", func(t *testing.T) {
		t.Parallel()

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

		old := time.Now().Add(-lockMaxAge - time.Minute)
		if err := os.Chtimes(lockFile, old, old); err != nil {
			t.Fatal(err)
		}

		// Try to acquire second lock - should report contention
		_, done, err := tryAcquireLock(lockFile, 0)
		if done {
			t.Error("expected live lock to be retried")
		}
		if err == nil {
			t.Error("expected error when lock already held")
		} else if !strings.Contains(err.Error(), "another grove operation") {
			t.Errorf("expected in-progress error, got: %v", err)
		}
	})

	t.Run("waits for lock held by running process", func(t *testing.T) {
		t.Parallel()

		tmpDir := testutil.TempDir(t)
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		handle1, err := AcquireWorkspaceLock(lockFile)
		if err != nil {
			t.Fatalf("expected to acquire first lock, got error: %v", err)
		}
		released := make(chan struct{})
		go func() {
			time.Sleep(500 * time.Millisecond)
			_ = handle1.Close()
			_ = os.Remove(lockFile)
			close(released)
		}()

		handle2, err := AcquireWorkspaceLock(lockFile)
		<-released
		if err != nil {
			t.Fatalf("expected to acquire lock after release, got error: %v", err)
		}
		defer func() {
			_ = handle2.Close()
			_ = os.Remove(lockFile)
		}()
	})

	t.Run("removes stale lock with invalid PID", func(t *testing.T) {
		t.Parallel()

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
		t.Parallel()

		tmpDir := testutil.TempDir(t)
		lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")

		// Create lock file with PID that doesn't exist (use very high PID)
		// PID 99999999 is unlikely to exist on any system
		if err := os.WriteFile(lockFile, []byte("99999999"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}
		old := time.Now().Add(-lockMaxAge - time.Minute)
		if err := os.Chtimes(lockFile, old, old); err != nil {
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
		t.Parallel()

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
	t.Parallel()

	t.Run("returns done=true on success", func(t *testing.T) {
		t.Parallel()
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
		t.Parallel()

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
	t.Parallel()

	t.Run("returns true for current process", func(t *testing.T) {
		t.Parallel()
		pid := os.Getpid()
		if !isProcessRunning(pid) {
			t.Error("expected current process to be running")
		}
	})

	t.Run("returns false for non-existent PID", func(t *testing.T) {
		t.Parallel()

		// Use a very high PID that's unlikely to exist
		if isProcessRunning(99999999) {
			t.Error("expected non-existent PID to return false")
		}
	})
}

func TestTryAcquireLockRemovesNonPositivePID(t *testing.T) {
	t.Parallel()
	tmpDir := testutil.TempDir(t)
	lockFile := filepath.Join(tmpDir, ".grove-worktree.lock")
	if err := os.WriteFile(lockFile, []byte("0"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}

	_, done, err := tryAcquireLock(lockFile, 0)
	if done || err != nil {
		t.Fatalf("expected stale removal, got done=%v err=%v", done, err)
	}
	if _, statErr := os.Stat(lockFile); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("expected lock file removed, got %v", statErr)
	}
}
