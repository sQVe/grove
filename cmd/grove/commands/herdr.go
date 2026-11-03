package commands

import (
	"bytes"
	"encoding/json"
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

// herdrSpace is one local space from `herdr-app ls`.
type herdrSpace struct {
	ID      string     `json:"id"`
	Folder  string     `json:"folder"`
	Machine *string    `json:"machine"`
	Tabs    []herdrTab `json:"tabs"`
}

type herdrTab struct {
	Terminals []herdrTerminal `json:"terminals"`
}

type herdrTerminal struct {
	Terminal string `json:"terminal"`
}

// herdrSpaces indexes the local Herdr spaces in the order herdr-app reported
// them. The order matters: when several spaces share a folder, Grove selects
// the first one.
type herdrSpaces struct {
	list      []herdrSpace
	currentID string
}

// openWorktreeInHerdr shows worktreePath as a Herdr space, focusing the space
// already open on it or creating one named after the label. Herdr keys a space
// on the folder, so the path is resolved first: a fresh worktree carries the
// spelling the caller built, while a re-run carries the one git recorded, and
// through a symlinked root those differ. A detached worktree has no label, so
// its directory names the space.
func openWorktreeInHerdr(worktreePath, label string) error {
	if _, err := exec.LookPath("herdr-app"); err != nil {
		return fmt.Errorf("cannot open worktree %s in Herdr (ensure herdr-app is installed and on PATH): %w", worktreePath, err)
	}

	canonical, err := filepath.EvalSymlinks(worktreePath)
	if err != nil {
		return fmt.Errorf("cannot resolve the path of worktree %s for Herdr: %w", worktreePath, err)
	}

	output, err := runHerdrApp("ls")
	if err != nil {
		return fmt.Errorf("herdr-app failed to list spaces for worktree %s: %w", canonical, err)
	}

	spaces, err := parseHerdrSpaces(output)
	if err != nil {
		return err
	}

	if id := spaces.lookup(canonical); id != "" {
		if _, err := runHerdrApp("do", "space.select", "--id", id); err != nil {
			return fmt.Errorf("herdr-app failed to open worktree %s: %w", canonical, err)
		}

		return nil
	}

	name := label
	if name == "" {
		name = filepath.Base(canonical)
	}
	if _, err := runHerdrApp("do", "space.new", "--name", name, "--folder", canonical); err != nil {
		return fmt.Errorf("herdr-app failed to open worktree %s: %w", canonical, err)
	}

	return nil
}

// runHerdrApp runs herdr-app and returns its stdout. herdr-app prints a JSON
// result on stdout, which the shell wrapper would otherwise echo as a cd
// target, so stdout never reaches grove's stdout. A failure carries herdr-app's
// own reason: the error object it prints on stdout, or its stderr when no
// window is open.
func runHerdrApp(arguments ...string) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	command := exec.Command("herdr-app", arguments...) //nolint:gosec // Paths are passed as arguments, not shell commands.
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w", herdrAppReason(&stdout, &stderr), err)
	}

	_, _ = os.Stderr.Write(stderr.Bytes())
	return stdout.Bytes(), nil
}

// herdrAppReason prefers the JSON error `do` prints on stdout, then the stderr
// line, so both an error object and a no-window failure reach the caller.
func herdrAppReason(stdout, stderr *bytes.Buffer) string {
	var result struct {
		Error string `json:"error"`
	}
	if json.Unmarshal(stdout.Bytes(), &result) == nil && result.Error != "" {
		return result.Error
	}

	if reason := strings.TrimSpace(stderr.String()); reason != "" {
		return reason
	}

	return "no output on stdout or stderr"
}

func parseHerdrSpaces(output []byte) (*herdrSpaces, error) {
	var list struct {
		Spaces []herdrSpace `json:"spaces"`
	}
	if err := json.Unmarshal(output, &list); err != nil {
		return nil, fmt.Errorf("cannot parse herdr-app space list: %w", err)
	}

	currentTerminal := os.Getenv("HC_TERMINAL_ID")
	spaces := &herdrSpaces{}
	for _, space := range list.Spaces {
		if space.Machine != nil {
			continue
		}

		spaces.list = append(spaces.list, space)
		for _, tab := range space.Tabs {
			for _, terminal := range tab.Terminals {
				if currentTerminal != "" && terminal.Terminal == currentTerminal {
					spaces.currentID = space.ID
				}
			}
		}
	}

	return spaces, nil
}

// lookup returns the id of the first local space open on path, or "" when there
// is none. Resolve before removal: once the directory is gone its symlinks
// cannot be followed. Compare with fs.PathsEqual, since Windows paths ignore case.
func (s *herdrSpaces) lookup(path string) string {
	if s == nil {
		return ""
	}

	resolved := resolvePath(path)
	for _, space := range s.list {
		if fs.PathsEqual(resolvePath(space.Folder), resolved) {
			return space.ID
		}
	}

	return ""
}

// current returns the space holding the caller's own terminal, or "".
func (s *herdrSpaces) current() string {
	if s == nil {
		return ""
	}

	return s.currentID
}

// resolvePath follows symlinks so the spelling herdr-app reports and the one git
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

// findHerdrSpaces queries herdr-app before removing worktrees so a missing or
// failing herdr-app leaves them intact.
func findHerdrSpaces() (*herdrSpaces, error) {
	if _, err := exec.LookPath("herdr-app"); err != nil {
		return nil, fmt.Errorf("cannot close Herdr spaces (ensure herdr-app is installed and on PATH): %w", err)
	}

	output, err := runHerdrApp("ls")
	if err != nil {
		return nil, fmt.Errorf("herdr-app failed to list spaces: %w", err)
	}

	return parseHerdrSpaces(output)
}

// closeHerdrSpaces closes the spaces of removed worktrees. The worktrees are
// already gone, so a failed close is only a warning. The caller's own space
// closes last because closing it ends the caller's pane.
func closeHerdrSpaces(ids []string, currentID string) {
	closeCurrent := false
	for _, id := range ids {
		if id == currentID {
			closeCurrent = true
			continue
		}
		closeHerdrSpace(id)
	}

	if closeCurrent {
		closeHerdrSpace(currentID)
	}
}

func closeHerdrSpace(id string) {
	if _, err := runHerdrApp("do", "space.close", "--id", id, "--confirmed", "true"); err != nil {
		logger.Warning("Could not close Herdr space %s: %v", id, err)
	}
}
