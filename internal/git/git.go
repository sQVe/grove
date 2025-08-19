package git

import (
	"bufio"
	"bytes"
	"crypto/sha1" // nolint:gosec // Used for filename hashing, not cryptographic security
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

		if strings.HasPrefix(line, "remotes/origin/") {
			branch := strings.TrimPrefix(line, "remotes/origin/")
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
	cacheDir := os.Getenv("TEST_CACHE_DIR")
	if cacheDir == "" {
		var err error
		cacheDir, err = os.UserCacheDir()
		if err != nil {
			return "", err
		}
	}

	groveCache := filepath.Join(cacheDir, "grove", "branches")
	if err := os.MkdirAll(groveCache, fs.DirGit); err != nil {
		return "", err
	}

	hash := sha1.Sum([]byte(url)) // nolint:gosec // Used for filename hashing, not cryptographic security
	filename := fmt.Sprintf("%x.txt", hash)

	return filepath.Join(groveCache, filename), nil
}

func writeCacheFile(path string, branches []string) error {
	content := strings.Join(branches, "\n")
	return os.WriteFile(path, []byte(content), fs.FileGit)
}
