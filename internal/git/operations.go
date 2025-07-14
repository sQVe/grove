package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitError represents an error from a git command execution.
type GitError struct {
	Command  string
	Args     []string
	Stderr   string
	ExitCode int
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %s failed (exit %d): %s", strings.Join(e.Args, " "), e.ExitCode, e.Stderr)
}

// ExecuteGit runs a git command with the given arguments and returns stdout.
// If the command fails, it returns a GitError with stderr and exit code.
func ExecuteGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)

	stdout, err := cmd.Output()
	if err != nil {
		var stderr string
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr = string(exitErr.Stderr)
		}

		return "", &GitError{
			Command:  "git",
			Args:     args,
			Stderr:   stderr,
			ExitCode: cmd.ProcessState.ExitCode(),
		}
	}

	return strings.TrimSpace(string(stdout)), nil
}

// IsGitRepository checks if the current directory is inside a git repository.
func IsGitRepository() (bool, error) {
	_, err := ExecuteGit("rev-parse", "--git-dir")
	if err != nil {
		if gitErr, ok := err.(*GitError); ok && gitErr.ExitCode == 128 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// IsGitAvailable checks if git is available in the system PATH.
func IsGitAvailable() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

// GetRepositoryRoot returns the root directory of the current git repository.
func GetRepositoryRoot() (string, error) {
	output, err := ExecuteGit("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return output, nil
}

// ValidateRepository checks if we're in a valid git repository with commits.
func ValidateRepository() error {
	if !IsGitAvailable() {
		return fmt.Errorf("git is not available in PATH")
	}

	isRepo, err := IsGitRepository()
	if err != nil {
		return fmt.Errorf("failed to check git repository: %v", err)
	}
	if !isRepo {
		return fmt.Errorf("not in a git repository")
	}

	// Check if repository has any commits
	_, err = ExecuteGit("rev-parse", "HEAD")
	if err != nil {
		if gitErr, ok := err.(*GitError); ok && strings.Contains(gitErr.Stderr, "bad revision") {
			return fmt.Errorf("repository has no commits")
		}
		return fmt.Errorf("failed to validate repository: %v", err)
	}

	return nil
}
