package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

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

// CloneQuiet clones a repository with minimal output
func CloneQuiet(url, path string) error {
	logger.Debug("Executing: git clone --bare --quiet %s %s", url, path)
	cmd := exec.Command("git", "clone", "--bare", "--quiet", url, path)

	// Capture output for error reporting but don't stream it
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Include captured stderr in error for debugging
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}

	return nil
}

// ListBranches returns a list of all branches in a bare repository
func ListBranches(bareRepo string) ([]string, error) {
	logger.Debug("Executing: git branch -a in %s", bareRepo)
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = bareRepo

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	var branches []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Remove "* " prefix for current branch
		line = strings.TrimPrefix(line, "* ")

		// Handle remote branches
		if strings.HasPrefix(line, "remotes/origin/") {
			branch := strings.TrimPrefix(line, "remotes/origin/")
			if branch != "HEAD" {
				branches = append(branches, branch)
			}
		} else if line != "HEAD" {
			// Handle direct branches (common in bare repos)
			branches = append(branches, line)
		}
	}

	return branches, scanner.Err()
}

// CreateWorktree creates a new worktree from a bare repository
func CreateWorktree(bareRepo, worktreePath, branch string) error {
	logger.Debug("Executing: git worktree add %s %s", worktreePath, branch)
	cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
	cmd.Dir = bareRepo

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// CreateWorktreeQuiet creates a new worktree with minimal output
func CreateWorktreeQuiet(bareRepo, worktreePath, branch string) error {
	logger.Debug("Executing: git worktree add %s %s (quiet)", worktreePath, branch)
	cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
	cmd.Dir = bareRepo

	// Capture output for error reporting but don't stream it
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		// Include captured stderr in error for debugging
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, stderr.String())
		}
		return err
	}

	return nil
}

// IsInsideGitRepo checks if the given path is inside an existing git repository
func IsInsideGitRepo(path string) bool {
	logger.Debug("Checking if %s is inside git repository", path)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	return cmd.Run() == nil
}
