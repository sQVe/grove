package git

import (
	"os/exec"

	"github.com/sqve/grove/internal/logger"
)

// InitBare initializes a bare git repository in the specified directory
func InitBare(path string) error {
	logger.Debug("Executing: git init --bare in %s", path)
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = path
	return cmd.Run()
}

// IsInsideGitRepo checks if the given path is inside an existing git repository
func IsInsideGitRepo(path string) bool {
	logger.Debug("Checking if %s is inside git repository", path)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	return cmd.Run() == nil
}
