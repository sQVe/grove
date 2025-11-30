//go:build !windows

package commands

import (
	"os"
	"syscall"
)

// isProcessRunning checks if a process with the given PID is still running.
// On Unix, FindProcess always succeeds, so we send signal 0 to verify.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 doesn't actually send a signal, just checks if process exists
	return process.Signal(syscall.Signal(0)) == nil
}
