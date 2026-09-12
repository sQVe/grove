package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// openWorktreeInHerdr hands worktreePath to the Herdr CLI and focuses it. Herdr
// keys a workspace on the checkout path, so the path is resolved first: a fresh
// worktree carries the spelling the caller built, while a re-run carries the one
// git recorded, and through a symlinked root those differ.
func openWorktreeInHerdr(bareDir, worktreePath string) error {
	canonical, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		return fmt.Errorf("worktree at %s is ready, but its path could not be resolved for Herdr: %w", worktreePath, err)
	}

	// Both arguments must land in the same namespace: Herdr resolves the
	// workspace from --cwd and the worktree from --path.
	root, err := filepath.EvalSymlinks(filepath.Dir(bareDir))
	if err != nil {
		return fmt.Errorf("worktree at %s is ready, but the workspace root could not be resolved for Herdr: %w", canonical, err)
	}

	command := exec.Command("herdr", "worktree", "open", "--cwd", root, "--path", canonical, "--focus") //nolint:gosec // Paths are passed as arguments, not shell commands.
	command.Stderr = os.Stderr

	if err := command.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("worktree at %s is ready, but Herdr could not open it (ensure herdr is installed and on PATH): %w", canonical, err)
		}

		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return fmt.Errorf("worktree at %s is ready, but Herdr exited with an error; see its output above: %w", canonical, err)
		}

		return fmt.Errorf("worktree at %s is ready, but Herdr could not start: %w", canonical, err)
	}

	return nil
}
