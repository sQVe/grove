package workspace

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sqve/grove/internal/logger"
)

const (
	maxLockRetries = 3
	// lockMaxAge is the maximum age of a lock file before it's considered stale.
	// This mitigates PID reuse attacks - if the lock is older than this duration,
	// even if a process with the same PID is running, it's not the original holder.
	lockMaxAge = 30 * time.Minute
)

// AcquireWorkspaceLock attempts to acquire a lock file, with staleness detection.
// If a lock file exists, checks if the owning process is still running.
// Stale locks (from crashed processes) are automatically removed.
func AcquireWorkspaceLock(lockFile string) (*os.File, error) {
	for attempt := range maxLockRetries {
		lockHandle, done, err := tryAcquireLock(lockFile, attempt)
		if done {
			return lockHandle, err
		}
	}
	return nil, fmt.Errorf("failed to acquire lock after %d attempts; if no grove operation is running, remove %s", maxLockRetries, lockFile)
}

// tryAcquireLock makes a single attempt to acquire the lock.
// Returns (handle, true, nil) on success, (nil, true, err) on permanent failure,
// or (nil, false, nil) if a stale lock was removed and retry is needed.
func tryAcquireLock(lockFile string, attempt int) (*os.File, bool, error) {
	lockHandle, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) //nolint:gosec // path derived from validated workspace
	if err == nil {
		_, _ = lockHandle.WriteString(strconv.Itoa(os.Getpid()))
		_ = lockHandle.Sync() // Ensure PID is flushed to disk before returning
		return lockHandle, true, nil
	}

	if !errors.Is(err, os.ErrExist) {
		return nil, true, fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Lock file exists - check if it's stale
	info, statErr := os.Stat(lockFile)
	if statErr != nil {
		// File disappeared between OpenFile and Stat - retry
		return nil, false, nil //nolint:nilerr // intentional: file gone, retry
	}

	content, readErr := os.ReadFile(lockFile) //nolint:gosec // path derived from validated workspace
	if readErr != nil {
		return nil, true, fmt.Errorf("another grove operation is in progress; if this is wrong, remove %s", lockFile)
	}

	pidStr := strings.TrimSpace(string(content))
	pid, parseErr := strconv.Atoi(pidStr)
	if parseErr != nil {
		// Invalid PID means corrupted lock file - treat as stale and retry
		logger.Debug("Lock file contains invalid PID %q, removing stale lock (attempt %d)", pidStr, attempt+1)
		_ = os.Remove(lockFile)
		return nil, false, nil //nolint:nilerr // intentional: invalid PID = stale lock, retry
	}

	// Check if lock is too old (mitigates PID reuse attacks)
	lockAge := time.Since(info.ModTime())
	if lockAge > lockMaxAge {
		logger.Debug("Lock file is %v old (max %v), removing stale lock (attempt %d)", lockAge.Round(time.Second), lockMaxAge, attempt+1)
		_ = os.Remove(lockFile)
		return nil, false, nil // Retry
	}

	if !isProcessRunning(pid) {
		logger.Debug("Lock held by terminated process %d, removing stale lock (attempt %d)", pid, attempt+1)
		// Re-verify file hasn't changed before removing (minimize TOCTOU window)
		if content2, err := os.ReadFile(lockFile); err == nil && strings.TrimSpace(string(content2)) == pidStr { //nolint:gosec // path derived from validated workspace
			_ = os.Remove(lockFile)
		}
		return nil, false, nil // Retry regardless - if remove failed, next attempt will handle it
	}

	return nil, true, fmt.Errorf("another grove operation (PID %d) is in progress; if this is wrong, remove %s", pid, lockFile)
}
