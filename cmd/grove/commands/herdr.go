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

type herdrSpace struct {
	ID      string     `json:"id"`
	Folder  string     `json:"folder"`
	Machine *string    `json:"machine"`
	Tabs    []herdrTab `json:"tabs"`

	resolvedFolder string
}

type herdrTab struct {
	Terminals []herdrTerminal `json:"terminals"`
}

type herdrTerminal struct {
	Terminal string `json:"terminal"`
}

type herdrSpaces struct {
	list      []herdrSpace
	currentID string
}

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

		space.resolvedFolder = resolvePath(space.Folder)
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
// is none. Space folders are resolved once in parseHerdrSpaces, before any
// worktree is removed. Resolve path before its own directory is removed: once
// the directory is gone its symlinks cannot be followed. Compare with
// fs.PathsEqual, since Windows paths ignore case.
func (s *herdrSpaces) lookup(path string) string {
	if s == nil {
		return ""
	}

	resolved := resolvePath(path)
	for _, space := range s.list {
		if fs.PathsEqual(space.resolvedFolder, resolved) {
			return space.ID
		}
	}

	return ""
}

func (s *herdrSpaces) current() string {
	if s == nil {
		return ""
	}

	return s.currentID
}

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
