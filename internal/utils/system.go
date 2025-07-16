package utils

import "os/exec"

// IsGitAvailable reports whether git is available in the system PATH.
func IsGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}
