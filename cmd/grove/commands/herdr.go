package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
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

	if _, err := runHerdr(arguments...); err != nil {
		return fmt.Errorf("herdr failed to open worktree %s: %w", canonical, err)
	}

	return nil
}

// runHerdr runs the Herdr CLI and returns its stdout. Herdr prints a JSON
// result on stdout, which the shell wrapper would otherwise echo as a cd
// target, and its failure reason on stderr, which belongs in the error.
func runHerdr(arguments ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command("herdr", arguments...) //nolint:gosec // Paths are passed as arguments, not shell commands.
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			reason := strings.TrimSpace(stderr.String())
			if reason == "" {
				reason = "no output on stderr"
			}

			return nil, fmt.Errorf("%s: %w", reason, err)
		}

		return nil, fmt.Errorf("herdr could not start: %w", err)
	}

	_, _ = os.Stderr.Write(stderr.Bytes())
	return stdout.Bytes(), nil
}

// herdrWorkspaces maps normalized worktree paths to Herdr workspace IDs.
type herdrWorkspaces map[string]string

// findHerdrWorkspaces queries Herdr before removing worktrees so a missing or
// failing Herdr leaves them intact.
func findHerdrWorkspaces(bareDir string) (herdrWorkspaces, error) {
	if _, err := exec.LookPath("herdr"); err != nil {
		return nil, fmt.Errorf("cannot close Herdr workspaces (ensure herdr is installed and on PATH): %w", err)
	}

	root, err := filepath.EvalSymlinks(filepath.Dir(bareDir))
	if err != nil {
		return nil, fmt.Errorf("cannot resolve the workspace root for Herdr: %w", err)
	}

	output, err := runHerdr("worktree", "list", "--cwd", root)
	if err != nil {
		return nil, fmt.Errorf("herdr failed to list worktrees: %w", err)
	}

	var list struct {
		Result struct {
			Worktrees []struct {
				Path            string `json:"path"`
				OpenWorkspaceID string `json:"open_workspace_id"`
			} `json:"worktrees"`
		} `json:"result"`
	}
	if err := json.Unmarshal(output, &list); err != nil {
		return nil, fmt.Errorf("cannot parse herdr worktree list: %w", err)
	}

	workspaces := herdrWorkspaces{}
	for _, worktree := range list.Result.Worktrees {
		if worktree.OpenWorkspaceID != "" {
			workspaces[resolvePath(worktree.Path)] = worktree.OpenWorkspaceID
		}
	}

	return workspaces, nil
}

// lookup returns the workspace open on path, or "" when there is none. Resolve
// before removal: once the directory is gone its symlinks cannot be followed.
// Compare with fs.PathsEqual, not a map key, since Windows paths ignore case.
func (w herdrWorkspaces) lookup(path string) string {
	if len(w) == 0 {
		return ""
	}

	resolved := resolvePath(path)
	for worktreePath, id := range w {
		if fs.PathsEqual(worktreePath, resolved) {
			return id
		}
	}

	return ""
}

// resolvePath follows symlinks so the spelling Herdr reports and the one git
// recorded compare equal. It resolves the nearest existing ancestor too, because
// prunable worktrees may already be missing.
func resolvePath(path string) string {
	path = filepath.Clean(path)
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}

	parent := filepath.Dir(path)
	if parent == path {
		return path
	}

	return filepath.Join(resolvePath(parent), filepath.Base(path))
}

// closeHerdrWorkspaces closes the workspaces of removed worktrees. The worktrees
// are already gone, so a failed close is only a warning. The caller's own
// workspace closes last because closing it ends the caller's pane.
func closeHerdrWorkspaces(ids []string) {
	current := os.Getenv("HERDR_WORKSPACE_ID")
	closeCurrent := false
	for _, id := range ids {
		if id == current {
			closeCurrent = true
			continue
		}
		closeHerdrWorkspace(id)
	}

	if closeCurrent {
		closeHerdrWorkspace(current)
	}
}

func closeHerdrWorkspace(id string) {
	if _, err := runHerdr("workspace", "close", id); err != nil {
		logger.Warning("Could not close Herdr workspace %s: %v", id, err)
	}
}
