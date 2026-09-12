package commands

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/logger"
)

func openWorktreeInHerdr(bareDir, worktreePath string) error {
	if _, err := exec.LookPath("herdr"); err != nil {
		return fmt.Errorf("herdr not found for worktree %s: %w", worktreePath, err)
	}

	command := exec.Command("herdr", "worktree", "open", "--cwd", filepath.Dir(bareDir), "--path", worktreePath, "--focus") // nolint:gosec // Paths come from the resolved workspace and are passed as separate arguments.
	// Herdr returns JSON on stdout, which the shell wrapper would otherwise print.
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		return fmt.Errorf("herdr failed to open worktree %s: %s: %w", worktreePath, strings.TrimSpace(stderr.String()), err)
	}

	logger.Success("Opened worktree %s in Herdr", worktreePath)
	return nil
}
