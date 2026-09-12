package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// openWorktreeInHerdr hands worktreePath to the Herdr CLI and focuses it. Herdr
// keys a workspace on the checkout path, so the path is resolved first: a fresh
// worktree carries the spelling the caller built, while a re-run carries the one
// git recorded, and through a symlinked root those differ.
func openWorktreeInHerdr(bareDir, worktreePath string) error {
	if _, err := exec.LookPath("herdr"); err != nil {
		return fmt.Errorf("cannot open worktree %s in Herdr (ensure herdr is installed and on PATH): %w", worktreePath, err)
	}

	canonical, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		return fmt.Errorf("cannot resolve the path of worktree %s for Herdr: %w", worktreePath, err)
	}

	// Both arguments must land in the same namespace: Herdr resolves the
	// workspace from --cwd and the worktree from --path.
	root, err := filepath.EvalSymlinks(filepath.Dir(bareDir))
	if err != nil {
		return fmt.Errorf("cannot resolve the workspace root of worktree %s for Herdr: %w", canonical, err)
	}

	// Herdr prints a JSON result on stdout, which the shell wrapper would
	// otherwise echo as a cd target, and its failure reason on stderr, which
	// belongs in the error.
	var stderr bytes.Buffer
	command := exec.Command("herdr", "worktree", "open", "--cwd", root, "--path", canonical, "--focus") //nolint:gosec // Paths are passed as arguments, not shell commands.
	command.Stdout = io.Discard
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			reason := strings.TrimSpace(stderr.String())
			if reason == "" {
				reason = "no output on stderr"
			}

			return fmt.Errorf("herdr failed to open worktree %s: %s: %w", canonical, reason, err)
		}

		return fmt.Errorf("herdr could not start for worktree %s: %w", canonical, err)
	}

	_, _ = os.Stderr.Write(stderr.Bytes())
	return nil
}
