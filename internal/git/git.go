package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

// ErrNoUpstreamConfigured is returned when a branch has no upstream configured
var ErrNoUpstreamConfigured = errors.New("branch has no upstream configured")

// remoteBranchCacheDuration is how long remote branch listings are cached
const remoteBranchCacheDuration = 5 * time.Minute

// runGitCommand executes a git command with consistent stderr capture and error handling
func runGitCommand(cmd *exec.Cmd, quiet bool) error {
	if quiet {
		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			if stderr.Len() > 0 {
				return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
			}
			return err
		}
		return nil
	}

	// Verbose mode: stream stdout and stderr
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// InitBare initializes a bare git repository in the specified directory
func InitBare(path string) error {
	if path == "" {
		return errors.New("repository path cannot be empty")
	}
	logger.Debug("Executing: git init --bare in %s", path)
	cmd := exec.Command("git", "init", "--bare")
	cmd.Dir = path
	return runGitCommand(cmd, true) // Always quiet for init
}

// ConfigureBare configures a git repository as bare
func ConfigureBare(path string) error {
	if path == "" {
		return errors.New("repository path cannot be empty")
	}
	logger.Debug("Executing: git config --bool core.bare true in %s", path)
	cmd := exec.Command("git", "config", "--bool", "core.bare", "true")
	cmd.Dir = path
	return runGitCommand(cmd, true)
}

// Clone clones a git repository as bare into the specified path
func Clone(url, path string, quiet bool) error {
	if url == "" {
		return errors.New("repository URL cannot be empty")
	}
	if path == "" {
		return errors.New("destination path cannot be empty")
	}

	var cmd *exec.Cmd
	if quiet {
		logger.Debug("Executing: git clone --bare --quiet %s %s", url, path)
		cmd = exec.Command("git", "clone", "--bare", "--quiet", url, path)
	} else {
		logger.Debug("Executing: git clone --bare %s %s", url, path)
		cmd = exec.Command("git", "clone", "--bare", url, path)
	}

	return runGitCommand(cmd, quiet)
}

// ListBranches returns a list of all branches in a bare repository
func ListBranches(bareRepo string) ([]string, error) {
	logger.Debug("Executing: git branch -a in %s", bareRepo)
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = bareRepo

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
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
	if bareRepo == "" {
		return errors.New("bare repository path cannot be empty")
	}
	if worktreePath == "" {
		return errors.New("worktree path cannot be empty")
	}
	if branch == "" {
		return errors.New("branch name cannot be empty")
	}

	var cmd *exec.Cmd
	if quiet {
		logger.Debug("Executing: git worktree add %s %s (quiet)", worktreePath, branch)
		cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	} else {
		logger.Debug("Executing: git worktree add %s %s", worktreePath, branch)
		cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
	}
	cmd.Dir = bareRepo

	return runGitCommand(cmd, quiet)
}

// IsInsideGitRepo checks if the given path is inside an existing git repository
func IsInsideGitRepo(path string) bool {
	logger.Debug("Checking if %s is inside git repository", path)
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	return cmd.Run() == nil
}

// IsWorktree checks if the given path is a git worktree
func IsWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// HasUncommittedChanges checks if the repository has uncommitted changes
func HasUncommittedChanges(path string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return false, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return false, err
	}

	return strings.TrimSpace(out.String()) != "", nil
}

// ListRemoteBranches returns a list of all branches from a remote repository with transparent caching
func ListRemoteBranches(url string) ([]string, error) { // nolint:unparam // Return value is used in completion and tests
	if url == "" {
		return nil, errors.New("repository URL cannot be empty")
	}
	cacheFile, err := getCacheFile(url)
	if err != nil {
		return listRemoteBranchesLive(url)
	}

	if fileInfo, err := os.Stat(cacheFile); err == nil {
		if time.Since(fileInfo.ModTime()) < remoteBranchCacheDuration {
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
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
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

func sanitizeCacheKey(url string) string {
	filename := strings.ReplaceAll(url, "/", "_")
	filename = strings.ReplaceAll(filename, ":", "_")
	filename = strings.ReplaceAll(filename, "?", "_")
	filename = strings.ReplaceAll(filename, "&", "_")
	filename = strings.ReplaceAll(filename, "=", "_")
	return filename
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

	filename := sanitizeCacheKey(url) + ".txt"

	return filepath.Join(groveCache, filename), nil
}

func writeCacheFile(path string, branches []string) error {
	content := strings.Join(branches, "\n")
	return os.WriteFile(path, []byte(content), fs.FileGit)
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	if path == "" {
		return "", errors.New("repository path cannot be empty")
	}
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

// ListWorktrees returns paths to existing worktrees, excluding the main repository
func ListWorktrees(repoPath string) ([]string, error) {
	logger.Debug("Executing: git worktree list in %s", repoPath)
	cmd := exec.Command("git", "worktree", "list")
	cmd.Dir = repoPath

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	var worktrees []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Split line by whitespace - first field is the path
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		worktreePath := fields[0]

		// Skip the main repository (has "(bare)" suffix or is the main repo)
		if len(fields) > 1 && strings.Contains(line, "(bare)") {
			continue
		}

		// Skip if the worktree path equals the repository path (main repo)
		absWorktreePath, err := filepath.Abs(worktreePath)
		if err != nil {
			return nil, err
		}
		absRepoPath, err := filepath.Abs(repoPath)
		if err != nil {
			return nil, err
		}
		if absWorktreePath == absRepoPath {
			continue
		}

		worktrees = append(worktrees, worktreePath)
	}

	return worktrees, scanner.Err()
}

// HasLockFiles checks if there are any active git lock files
func HasLockFiles(path string) (bool, error) {
	logger.Debug("Checking for lock files in %s", path)
	gitDir := filepath.Join(path, ".git")

	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return false, fmt.Errorf("not a git repository")
	}

	lockFiles, err := filepath.Glob(filepath.Join(gitDir, "*.lock"))
	if err != nil {
		return false, err
	}

	return len(lockFiles) > 0, nil
}

// HasUnresolvedConflicts checks if there are unresolved merge conflicts
func HasUnresolvedConflicts(path string) (bool, error) {
	logger.Debug("Checking for unresolved conflicts in %s", path)
	cmd := exec.Command("git", "ls-files", "-u")
	cmd.Dir = path

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return false, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return false, err
	}

	return strings.TrimSpace(out.String()) != "", nil
}

// HasSubmodules checks if the repository has submodules
func HasSubmodules(path string) (bool, error) {
	logger.Debug("Checking for submodules in %s", path)

	// Check for .gitmodules file first, since it is more reliable than git
	// submodule status.
	gitModulesPath := filepath.Join(path, ".gitmodules")
	if _, err := os.Stat(gitModulesPath); err == nil {
		return true, nil
	}

	cmd := exec.Command("git", "submodule", "status")
	cmd.Dir = path

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return false, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return false, err
	}

	output := strings.TrimSpace(out.String())
	return output != "", nil
}

// HasUnpushedCommits checks if the current branch has unpushed commits
func HasUnpushedCommits(path string) (bool, error) {
	logger.Debug("Checking for unpushed commits in %s", path)

	// First, check if an upstream branch is configured
	cmdUpstream := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	cmdUpstream.Dir = path
	var upstreamStderr bytes.Buffer
	cmdUpstream.Stderr = &upstreamStderr

	if err := cmdUpstream.Run(); err != nil {
		// If git rev-parse @{u} fails, it means no upstream is configured
		return false, fmt.Errorf("%w: %s", ErrNoUpstreamConfigured, strings.TrimSpace(upstreamStderr.String()))
	}

	// If upstream exists, proceed to check for unpushed commits
	cmdLog := exec.Command("git", "log", "@{u}..HEAD", "--oneline")
	cmdLog.Dir = path

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmdLog.Stdout = &out
	cmdLog.Stderr = &stderr

	if err := cmdLog.Run(); err != nil {
		return false, fmt.Errorf("failed to check unpushed commits: %w: %s", err, strings.TrimSpace(stderr.String()))
	}

	output := strings.TrimSpace(out.String())
	return output != "", nil
}

// ListLocalBranches returns a list of all local branches in a repository
func ListLocalBranches(path string) ([]string, error) {
	if path == "" {
		return nil, errors.New("repository path cannot be empty")
	}
	logger.Debug("Listing local branches in %s", path)
	cmd := exec.Command("git", "branch", "--format=%(refname:short)")
	cmd.Dir = path

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	var branches []string
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			branches = append(branches, line)
		}
	}

	return branches, scanner.Err()
}
