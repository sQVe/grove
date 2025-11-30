//go:build windows

package workspace

import (
	"golang.org/x/sys/windows"
)

// isProcessRunning checks if a process with the given PID is still running.
// On Windows, we try to open the process handle to verify it exists.
func isProcessRunning(pid int) bool {
	handle, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, uint32(pid))
	if err != nil {
		return false
	}
	_ = windows.CloseHandle(handle)
	return true
}
