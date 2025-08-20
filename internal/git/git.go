package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sqve/grove/internal/fs"
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
func Clone(url, path string, quiet bool) error {
	if quiet {
		logger.Debug("Executing: git clone --bare --quiet %s %s", url, path)
		cmd := exec.Command("git", "clone", "--bare", "--quiet", url, path)

		// Capture output for error reporting but don't stream it
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			if stderr.Len() > 0 {
				return fmt.Errorf("%w: %s", err, stderr.String())
			}
			return err
		}

		return nil
	} else {
		logger.Debug("Executing: git clone --bare %s %s", url, path)
		cmd := exec.Command("git", "clone", "--bare", url, path)

		// Let git's output stream through to user
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}
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

		line = strings.TrimPrefix(line, "* ")

		if after, ok := strings.CutPrefix(line, "remotes/origin/"); ok {
			branch := after
			if branch != "HEAD" {
				branches = append(branches, branch)
			}
		} else if line != "HEAD" {
			branches = append(branches, line)
		}
	}

	return branches, scanner.Err()
}

// CreateWorktree creates a new worktree from a bare repository
func CreateWorktree(bareRepo, worktreePath, branch string, quiet bool) error {
	if quiet {
		logger.Debug("Executing: git worktree add %s %s (quiet)", worktreePath, branch)
		cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
		cmd.Dir = bareRepo

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			if stderr.Len() > 0 {
				return fmt.Errorf("%w: %s", err, stderr.String())
			}
			return err
		}

		return nil
	} else {
		logger.Debug("Executing: git worktree add %s %s", worktreePath, branch)
		cmd := exec.Command("git", "worktree", "add", worktreePath, branch)
		cmd.Dir = bareRepo

		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		return cmd.Run()
	}
}

// IsInsideGitRepo checks if the given path is inside an existing git repository
func IsInsideGitRepo(path string) bool {
	logger.Debug("Checking if %s is inside git repository", path)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	return cmd.Run() == nil
}

// ListRemoteBranches returns a list of all branches from a remote repository with transparent caching
func ListRemoteBranches(url string) ([]string, error) { // nolint:unparam // Return value is used in completion and tests
	cacheFile, err := getCacheFile(url)
	if err != nil {
		return listRemoteBranchesLive(url)
	}

	if fileInfo, err := os.Stat(cacheFile); err == nil {
		if time.Since(fileInfo.ModTime()) < 5*time.Minute {
			content, err := os.ReadFile(cacheFile) // nolint:gosec // Reading controlled cache file
			if err == nil {
				lines := strings.Split(strings.TrimSpace(string(content)), "\n")
				if len(lines) == 1 && lines[0] == "" {
					return []string{}, nil
				}
				return lines, nil
			}
		}
	}

	branches, err := listRemoteBranchesLive(url)
	if err != nil {
		return nil, err
	}

	_ = writeCacheFile(cacheFile, branches)

	return branches, nil
}

func listRemoteBranchesLive(url string) ([]string, error) {
	logger.Debug("Executing: git ls-remote --heads %s", url)
	cmd := exec.Command("git", "ls-remote", "--heads", url)

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

		parts := strings.Fields(line)
		if len(parts) >= 2 && strings.HasPrefix(parts[1], "refs/heads/") {
			branch := strings.TrimPrefix(parts[1], "refs/heads/")
			branches = append(branches, branch)
		}
	}

	return branches, scanner.Err()
}

func getCacheFile(url string) (string, error) {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	groveCache := filepath.Join(cacheDir, "grove", "branches")
	if err := os.MkdirAll(groveCache, fs.DirGit); err != nil {
		return "", err
	}

	filename := strings.ReplaceAll(url, "/", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "&", "_")
	filename = strings.ReplaceAll(filename, "=", "_")
	filename += ".txt"

	return filepath.Join(groveCache, filename), nil
}

func writeCacheFile(path string, branches []string) error {
	content := strings.Join(branches, "\n")
	return os.WriteFile(path, []byte(content), fs.FileGit)
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	logger.Debug("Getting current branch from %s", path)
	headFile := filepath.Join(path, ".git", "HEAD")

	content, err := os.ReadFile(headFile) // nolint:gosec // Reading git HEAD file
	if err != nil {
		return "", err
	}

	line := strings.TrimSpace(string(content))

	if after, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
		return after, nil
	}

	return "", fmt.Errorf("detached HEAD state")
}

// IsDetachedHead checks if the repository is in detached HEAD state
func IsDetachedHead(path string) (bool, error) {
	logger.Debug("Checking detached HEAD in %s", path)
	headFile := filepath.Join(path, ".git", "HEAD")

	content, err := os.ReadFile(headFile) // nolint:gosec // Reading git HEAD file
	if err != nil {
		return false, err
	}

	line := strings.TrimSpace(string(content))

	return !strings.HasPrefix(line, "ref: refs/heads/"), nil
}

// HasOngoingOperation checks for merge/rebase/cherry-pick operations
func HasOngoingOperation(path string) (bool, error) {
	logger.Debug("Checking ongoing operations in %s", path)
	gitDir := filepath.Join(path, ".git")

	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false, fmt.Errorf("not a git repository")
	}

	markers := []string{
		"CHERRY_PICK_HEAD",
		"MERGE_HEAD",
		"REVERT_HEAD",
		"rebase-apply",
		"rebase-merge",
	}

	for _, marker := range markers {
		if _, err := os.Stat(filepath.Join(gitDir, marker)); err == nil {
			return true, nil
		}
	}

	return false, nil
}
