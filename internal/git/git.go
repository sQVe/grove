package git

import (
	"os/exec"
)

// InitBare initializes a bare git repository in the specified directory
func InitBare(path string) error {
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = path
	return cmd.Run()
}
