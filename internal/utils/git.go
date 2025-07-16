package utils

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sqve/grove/internal/logger"
)

// GitExecutor defines interface for git command execution.
type GitExecutor interface {
	Execute(args ...string) (string, error)
}

// IsGitRepository reports whether the current directory is inside a git repository.
func IsGitRepository(executor GitExecutor) (bool, error) {
	log := logger.WithComponent("git_utils")
	start := time.Now()

	log.DebugOperation("checking if current directory is git repository")

	output, err := executor.Execute("rev-parse", "--git-dir")
	duration := time.Since(start)

	if err != nil {
		log.Debug("git rev-parse --git-dir failed", "error", err, "duration", duration)
		// Simple heuristic: if exit code looks like 128, assume it's "not a repo"
		if strings.Contains(err.Error(), "exit 128") {
			log.Debug("directory is not a git repository", "reason", "exit_code_128", "duration", duration)
			return false, nil
		}
		log.ErrorOperation("unexpected error during git repository check", err, "duration", duration)
		return false, err
	}

	log.Debug("git repository detected", "git_dir", strings.TrimSpace(output), "duration", duration)
	return true, nil
}

// GetRepositoryRoot returns the root directory of the current git repository.
func GetRepositoryRoot(executor GitExecutor) (string, error) {
	log := logger.WithComponent("git_utils")
	start := time.Now()

	log.DebugOperation("getting git repository root directory")

	output, err := executor.Execute("rev-parse", "--show-toplevel")
	duration := time.Since(start)

	if err != nil {
		log.ErrorOperation("failed to get repository root", err, "duration", duration)
		return "", err
	}

	root := strings.TrimSpace(output)
	log.Debug("repository root determined", "root", root, "duration", duration)
	return root, nil
}

// ValidateRepository returns an error if not in a valid git repository with commits.
func ValidateRepository(executor GitExecutor) error {
	log := logger.WithComponent("git_utils")
	start := time.Now()

	log.InfoOperation("validating git repository")

	// Check git availability
	log.Debug("checking git availability in PATH")
	_, err := exec.LookPath("git")
	if err != nil {
		log.ErrorOperation("git not available in PATH", err, "duration", time.Since(start))
		return fmt.Errorf("git is not available in PATH")
	}
	log.Debug("git found in PATH")

	// Check if we're in a git repository
	log.Debug("checking if current directory is a git repository")
	isRepo, err := IsGitRepository(executor)
	if err != nil {
		log.ErrorOperation("failed to check git repository", err, "duration", time.Since(start))
		return fmt.Errorf("failed to check git repository: %w", err)
	}
	if !isRepo {
		err := fmt.Errorf("not in a git repository")
		log.ErrorOperation("validation failed", err, "reason", "not_git_repo", "duration", time.Since(start))
		return err
	}
	log.Debug("confirmed we are in a git repository")

	// Check if repository has any commits
	log.Debug("checking if repository has commits")
	_, err = executor.Execute("rev-parse", "HEAD")
	if err != nil {
		if strings.Contains(err.Error(), "bad revision") {
			err := fmt.Errorf("repository has no commits")
			log.ErrorOperation("validation failed", err, "reason", "no_commits", "duration", time.Since(start))
			return err
		}
		log.ErrorOperation("unexpected error validating repository commits", err, "duration", time.Since(start))
		return fmt.Errorf("failed to validate repository: %w", err)
	}
	log.Debug("repository has commits")

	log.InfoOperation("git repository validation completed successfully", "duration", time.Since(start))
	return nil
}

// IsGitURL reports whether str matches recognized git repository URL patterns.
func IsGitURL(str string) bool {
	log := logger.WithComponent("git_utils")
	start := time.Now()

	log.DebugOperation("checking if string is git URL", "input", str)

	if str == "" {
		log.Debug("git URL check failed: empty string", "duration", time.Since(start))
		return false
	}

	// Check for common git URL patterns
	patterns := []string{
		`^https?://.*\.git$`,                            // https://github.com/user/repo.git
		`^https?://github\.com/[\w\-\.]+/[\w\-\.]+/?$`,  // https://github.com/user/repo
		`^https?://gitlab\.com/[\w\-\.]+/[\w\-\.]+/?$`,  // https://gitlab.com/user/repo
		`^git@[\w\.-]+:[\w\-\.]+/[\w\-\.]+\.git$`,       // git@github.com:user/repo.git
		`^ssh://git@[\w\.-]+/[\w\-\.]+/[\w\-\.]+\.git$`, // SSH format: ssh://git@github.com/user/repo.git
	}

	log.Debug("checking against git URL patterns", "pattern_count", len(patterns))
	for i, pattern := range patterns {
		if matched, err := regexp.MatchString(pattern, str); err == nil && matched {
			log.Debug("git URL pattern matched", "pattern_index", i, "pattern", pattern, "input", str, "duration", time.Since(start))
			return true
		} else if err != nil {
			log.Debug("regex pattern match error", "pattern", pattern, "error", err)
		}
	}

	// Check if it's a valid URL with git scheme
	log.Debug("checking for git scheme URL")
	if u, err := url.Parse(str); err == nil {
		if u.Scheme == "git" {
			log.Debug("git scheme URL detected", "url", str, "scheme", u.Scheme, "duration", time.Since(start))
			return true
		}
		log.Debug("URL parsed but not git scheme", "scheme", u.Scheme)
	} else {
		log.Debug("URL parse failed", "error", err)
	}

	log.Debug("string is not a git URL", "input", str, "duration", time.Since(start))
	return false
}
