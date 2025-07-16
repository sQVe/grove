package utils

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
)

// GitExecutor defines interface for git command execution.
type GitExecutor interface {
	Execute(args ...string) (string, error)
}

// IsGitRepository reports whether the current directory is inside a git repository.
func IsGitRepository(executor GitExecutor) (bool, error) {
	_, err := executor.Execute("rev-parse", "--git-dir")
	if err != nil {
		// Simple heuristic: if exit code looks like 128, assume it's "not a repo"
		if strings.Contains(err.Error(), "exit 128") {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// GetRepositoryRoot returns the root directory of the current git repository.
func GetRepositoryRoot(executor GitExecutor) (string, error) {
	output, err := executor.Execute("rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return output, nil
}

// ValidateRepository returns an error if not in a valid git repository with commits.
func ValidateRepository(executor GitExecutor) error {
	_, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git is not available in PATH")
	}

	isRepo, err := IsGitRepository(executor)
	if err != nil {
		return fmt.Errorf("failed to check git repository: %w", err)
	}
	if !isRepo {
		return fmt.Errorf("not in a git repository")
	}

	// Check if repository has any commits
	_, err = executor.Execute("rev-parse", "HEAD")
	if err != nil {
		if strings.Contains(err.Error(), "bad revision") {
			return fmt.Errorf("repository has no commits")
		}
		return fmt.Errorf("failed to validate repository: %w", err)
	}

	return nil
}

// IsGitURL reports whether str matches recognized git repository URL patterns.
func IsGitURL(str string) bool {
	// Check for common git URL patterns
	patterns := []string{
		`^https?://.*\.git$`,                            // https://github.com/user/repo.git
		`^https?://github\.com/[\w\-\.]+/[\w\-\.]+/?$`,  // https://github.com/user/repo
		`^https?://gitlab\.com/[\w\-\.]+/[\w\-\.]+/?$`,  // https://gitlab.com/user/repo
		`^git@[\w\.-]+:[\w\-\.]+/[\w\-\.]+\.git$`,       // git@github.com:user/repo.git
		`^ssh://git@[\w\.-]+/[\w\-\.]+/[\w\-\.]+\.git$`, // SSH format: ssh://git@github.com/user/repo.git
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, str); matched {
			return true
		}
	}

	// Check if it's a valid URL with git scheme
	if u, err := url.Parse(str); err == nil {
		if u.Scheme == "git" {
			return true
		}
	}

	return false
}
