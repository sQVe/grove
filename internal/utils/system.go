package utils

import (
	"os/exec"
	"time"

	"github.com/sqve/grove/internal/logger"
)

// IsGitAvailable reports whether git is available in the system PATH.
func IsGitAvailable() bool {
	log := logger.WithComponent("system_utils")
	start := time.Now()

	log.DebugOperation("checking git availability in system PATH")

	gitPath, err := exec.LookPath("git")
	duration := time.Since(start)

	if err != nil {
		log.Debug("git not found in PATH", "error", err, "duration", duration)
		return false
	}

	log.Debug("git found in system PATH", "git_path", gitPath, "duration", duration)
	return true
}
