package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var herdrTicketPrefix = regexp.MustCompile(`^[a-zA-Z]+-\d+(?:-|$)`)

func herdrLabel(branch string) string {
	label := branch[strings.LastIndex(branch, "/")+1:]
	label = herdrTicketPrefix.ReplaceAllString(label, "")
	label = strings.NewReplacer("-", " ", "_", " ").Replace(label)
	label = strings.Join(strings.Fields(label), " ")

	if label == "" {
		return branch
	}

	return label
}

// openWorktreeInHerdr hands worktreePath to the Herdr CLI and focuses it. Herdr
// keys a workspace on the checkout path, so the path is resolved first: a fresh
// worktree carries the spelling the caller built, while a re-run carries the one
// git recorded, and through a symlinked root those differ.
func openWorktreeInHerdr(bareDir, worktreePath, label string) error {
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

	arguments := []string{"worktree", "open", "--cwd", root, "--path", canonical, "--focus"}
	if label != "" {
		arguments = append(arguments, "--label", label)
	}

	// Herdr prints a JSON result on stdout, which the shell wrapper would
	// otherwise echo as a cd target, and its failure reason on stderr, which
	// belongs in the error.
	var stderr bytes.Buffer
	command := exec.Command("herdr", arguments...) //nolint:gosec // Paths are passed as arguments, not shell commands.
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
