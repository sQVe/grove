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

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

// ErrNoUpstreamConfigured is returned when a branch has no upstream configured
var ErrNoUpstreamConfigured = errors.New("branch has no upstream configured")

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

// RestoreNormalConfig restores git repository to normal (non-bare) configuration
func RestoreNormalConfig(path string) error {
	if path == "" {
		return errors.New("repository path cannot be empty")
	}
	logger.Debug("Executing: git config --bool core.bare false in %s", path)
	cmd := exec.Command("git", "config", "--bool", "core.bare", "false")
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
	logger.Debug("Executing: git branch -a --format=%%(refname:short) in %s", bareRepo)
	cmd := exec.Command("git", "branch", "-a", "--format=%(refname:short)")
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

	branchSet := make(map[string]bool)
	scanner := bufio.NewScanner(&out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "origin" {
			continue
		}

		if branchName, ok := strings.CutPrefix(line, "origin/"); ok {
			if branchName != "HEAD" {
				branchSet[branchName] = true
			}
		} else {
			branchSet[line] = true
		}
	}

	var branches []string
	for branch := range branchSet {
		branches = append(branches, branch)
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
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = path
	return cmd.Run() == nil
}

// IsWorktree checks if the given path is a git worktree
func IsWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")
	return fs.FileExists(gitPath)
}

// CheckGitChanges runs git status once and returns both tracked and any changes
func CheckGitChanges(path string) (hasAnyChanges, hasTrackedChanges bool, err error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = path

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return false, false, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		logger.Debug("Git status failed: %v", err)
		return false, false, err
	}

	output := strings.TrimSpace(out.String())
	if output == "" {
		logger.Debug("Repository status: clean (no changes)")
		return false, false, nil
	}

	hasAnyChanges = true

	lines := strings.Split(output, "\n")
	changeCount := len(lines)
	for _, line := range lines {
		if line == "" {
			changeCount--
			continue
		}
		if !strings.HasPrefix(line, "??") {
			hasTrackedChanges = true
			break
		}
	}

	logger.Debug("Repository status: %d changes detected, tracked changes: %t", changeCount, hasTrackedChanges)
	return hasAnyChanges, hasTrackedChanges, nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	if path == "" {
		return "", errors.New("repository path cannot be empty")
	}
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

// GetDefaultBranch returns the default branch for a bare repository
func GetDefaultBranch(bareDir string) (string, error) {
	if bareDir == "" {
		return "", errors.New("repository path cannot be empty")
	}

	// For bare repos, HEAD is at the root
	headFile := filepath.Join(bareDir, "HEAD")

	content, err := os.ReadFile(headFile) // nolint:gosec // Reading git HEAD file
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD: %w", err)
	}

	line := strings.TrimSpace(string(content))

	if after, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
		return after, nil
	}

	return "", fmt.Errorf("could not determine default branch from HEAD")
}

// IsDetachedHead checks if the repository is in detached HEAD state
func IsDetachedHead(path string) (bool, error) {
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
	gitDir := filepath.Join(path, ".git")

	if !fs.DirectoryExists(gitDir) {
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
		if fs.PathExists(filepath.Join(gitDir, marker)) {
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
	gitDir := filepath.Join(path, ".git")

	if !fs.DirectoryExists(gitDir) {
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
	// Check for .gitmodules file first, since it is more reliable than git
	// submodule status.
	gitModulesPath := filepath.Join(path, ".gitmodules")
	if fs.FileExists(gitModulesPath) {
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

// BranchExists checks if a branch exists locally or on the remote
func BranchExists(repoPath, branchName string) (bool, error) {
	if repoPath == "" || branchName == "" {
		return false, errors.New("repository path and branch name cannot be empty")
	}

	cmd := exec.Command("git", "rev-parse", "--verify", "--quiet", branchName) // nolint:gosec // Branch name validated by git
	cmd.Dir = repoPath
	err := cmd.Run()
	if err == nil {
		return true, nil
	}

	remoteBranch := "origin/" + branchName
	cmd = exec.Command("git", "rev-parse", "--verify", "--quiet", remoteBranch) // nolint:gosec // Branch name validated by git
	cmd.Dir = repoPath
	err = cmd.Run()
	if err != nil {
		exitError := &exec.ExitError{}
		if errors.As(err, &exitError) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// ErrConfigNotFound is returned when a config key is not found
var ErrConfigNotFound = errors.New("config key not found")

// IsConfigNotFoundError returns true if error indicates config not found
func IsConfigNotFoundError(err error) bool {
	return errors.Is(err, ErrConfigNotFound)
}

// GetConfig gets a single config value
func GetConfig(key string, global bool) (string, error) {
	logger.Debug("Getting git config: %s (global=%v)", key, global)

	args := []string{"config", "--get"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitCode := cmd.ProcessState.ExitCode(); exitCode == 1 {
			return "", ErrConfigNotFound
		}
		if stderr.Len() > 0 {
			return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", err
	}

	return strings.TrimSpace(stdout.String()), nil
}

// GetConfigs gets all config values for keys with a given prefix
func GetConfigs(prefix string, global bool) (map[string][]string, error) {
	logger.Debug("Getting git configs with prefix: %s (global=%v)", prefix, global)

	args := []string{"config", "--get-regexp"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, prefix)

	cmd := exec.Command("git", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitCode := cmd.ProcessState.ExitCode(); exitCode == 1 {
			return make(map[string][]string), nil
		}
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	configs := make(map[string][]string)
	scanner := bufio.NewScanner(&stdout)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			key, value := parts[0], parts[1]
			configs[key] = append(configs[key], value)
		}
	}

	return configs, scanner.Err()
}

// SetConfig sets a config value, replacing any existing value
func SetConfig(key, value string, global bool) error {
	logger.Debug("Setting git config: %s=%s (global=%v)", key, value, global)

	args := []string{"config"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	cmd := exec.Command("git", args...)
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

// AddConfig adds a value to a multi-value config key
func AddConfig(key, value string, global bool) error {
	logger.Debug("Adding git config: %s=%s (global=%v)", key, value, global)

	args := []string{"config", "--add"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, value)

	cmd := exec.Command("git", args...)
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

// UnsetConfig removes a config key and all its values
func UnsetConfig(key string, global bool) error {
	logger.Debug("Unsetting git config: %s (global=%v)", key, global)

	args := []string{"config", "--unset-all"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key)

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitCode := cmd.ProcessState.ExitCode(); exitCode == 5 {
			return ErrConfigNotFound
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}

	return nil
}

// UnsetConfigValue removes a specific value from a config key using pattern matching
func UnsetConfigValue(key, valuePattern string, global bool) error {
	logger.Debug("Unsetting git config value: %s=%s (global=%v)", key, valuePattern, global)

	args := []string{"config", "--unset-all", "--fixed-value"}
	if global {
		args = append(args, "--global")
	}
	args = append(args, key, valuePattern)

	cmd := exec.Command("git", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if exitCode := cmd.ProcessState.ExitCode(); exitCode == 5 {
			return ErrConfigNotFound
		}
		if stderr.Len() > 0 {
			return fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return err
	}

	return nil
}
