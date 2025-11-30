package workspace

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/sqve/grove/internal/logger"
)

const maxLockRetries = 3

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
		return lockHandle, true, nil
	}

	if !errors.Is(err, os.ErrExist) {
		return nil, true, fmt.Errorf("failed to acquire lock: %w", err)
	}

	// Lock file exists - check if it's stale
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

	if !isProcessRunning(pid) {
		logger.Debug("Lock held by terminated process %d, removing stale lock (attempt %d)", pid, attempt+1)
		_ = os.Remove(lockFile)
		return nil, false, nil // Retry
	}

	return nil, true, fmt.Errorf("another grove operation (PID %d) is in progress; if this is wrong, remove %s", pid, lockFile)
}
