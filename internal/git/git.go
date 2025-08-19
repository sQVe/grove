package git

import (
	"os"
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

// Clone clones a git repository as bare into the specified path
func Clone(url, path string) error {
	logger.Debug("Executing: git clone --bare %s %s", url, path)
	cmd := exec.Command("git", "clone", "--bare", url, path)

	// Let git's output stream through to user
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// IsInsideGitRepo checks if the given path is inside an existing git repository
func IsInsideGitRepo(path string) bool {
	logger.Debug("Checking if %s is inside git repository", path)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	return cmd.Run() == nil
}
