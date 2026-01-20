package git

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

// ErrNoUpstreamConfigured is returned when a branch has no upstream configured
var ErrNoUpstreamConfigured = errors.New("branch has no upstream configured")

var ErrGitTooOld = errors.New("git version too old")

const MinGitVersion = "2.48"

func WrapGitTooOldError(err error) error {
	if err == nil {
		return nil
	}
	errStr := err.Error()
	if strings.Contains(errStr, "unknown option") && strings.Contains(errStr, "relative-paths") {
		return fmt.Errorf("%w: %w", ErrGitTooOld, err)
	}
	return err
}

func IsGitTooOld(err error) bool {
	return errors.Is(err, ErrGitTooOld)
}

func HintGitTooOld(err error) error {
	if err != nil && IsGitTooOld(err) {
		logger.Warning("Grove requires Git %s+ for portable worktrees", MinGitVersion)
		logger.Info("Run 'grove doctor' to check your environment")
	}
	return err
}

// Git operation marker files/directories
const (
	markerMergeHead      = "MERGE_HEAD"
	markerCherryPickHead = "CHERRY_PICK_HEAD"
	markerRevertHead     = "REVERT_HEAD"
	markerRebaseApply    = "rebase-apply"
	markerRebaseMerge    = "rebase-merge"
)

// GitCommand creates an exec.Cmd with timeout context if configured.
// Returns the command and a cancel function that must be called when done.
func GitCommand(name string, arg ...string) (*exec.Cmd, context.CancelFunc) {
	timeout := config.GetTimeout()
	if timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		return exec.CommandContext(ctx, name, arg...), cancel
	}
	return exec.Command(name, arg...), func() {}
}

// runGitCommand executes a git command with consistent stderr capture and error handling
func runGitCommand(cmd *exec.Cmd, quiet bool) error {
	if quiet {
		return executeWithStderr(cmd)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// executeWithStderr runs cmd and returns error with stderr context if available.
func executeWithStderr(cmd *exec.Cmd) error {
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

// executeWithOutput runs cmd and returns stdout as string, with stderr in error if failed.
func executeWithOutput(cmd *exec.Cmd) (string, error) {
	out, err := executeWithOutputBuffer(cmd)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// executeWithOutputBuffer runs cmd and returns stdout buffer, with stderr in error if failed.
// Use this when you need to process output line-by-line with a scanner.
func executeWithOutputBuffer(cmd *exec.Cmd) (*bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if stderr.Len() > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}
	return &stdout, nil
}

// resolveGitDir returns the actual git directory for a repository or worktree.
func resolveGitDir(path string) (string, error) {
	gitPath := filepath.Join(path, ".git")

	info, err := os.Stat(gitPath)
	if err != nil {
		return "", err
	}

	if info.IsDir() {
		return gitPath, nil
	}

	content, err := os.ReadFile(gitPath) // nolint:gosec // Reading git pointer file
	if err != nil {
		return "", err
	}

	line := strings.TrimSpace(string(content))
	if after, ok := strings.CutPrefix(line, "gitdir: "); ok {
		if filepath.IsAbs(after) {
			return after, nil
		}
		return filepath.Join(path, after), nil
	}

	return "", fmt.Errorf("invalid .git file format")
}

// InitBare initializes a bare git repository in the specified directory
func InitBare(path string) error {
	if path == "" {
		return errors.New("repository path cannot be empty")
	}
	logger.Debug("Executing: git init --bare in %s", path)
	cmd, cancel := GitCommand("git", "init", "--bare")
	defer cancel()
	cmd.Dir = path
	return runGitCommand(cmd, true) // Always quiet for init
}

// ConfigureBare configures a git repository as bare
func ConfigureBare(path string) error {
	if path == "" {
		return errors.New("repository path cannot be empty")
	}
	logger.Debug("Executing: git config --bool core.bare true in %s", path)
	cmd, cancel := GitCommand("git", "config", "--bool", "core.bare", "true")
	defer cancel()
	cmd.Dir = path
	return runGitCommand(cmd, true)
}

// RestoreNormalConfig restores git repository to normal (non-bare) configuration
func RestoreNormalConfig(path string) error {
	if path == "" {
		return errors.New("repository path cannot be empty")
	}
	logger.Debug("Executing: git config --bool core.bare false in %s", path)
	cmd, cancel := GitCommand("git", "config", "--bool", "core.bare", "false")
	defer cancel()
	cmd.Dir = path
	return runGitCommand(cmd, true)
}

// Clone clones a git repository as bare into the specified path
func Clone(url, path string, quiet, shallow bool) error {
	if url == "" {
		return errors.New("repository URL cannot be empty")
	}
	if path == "" {
		return errors.New("destination path cannot be empty")
	}

	args := []string{"clone", "--bare"}
	if quiet {
		args = append(args, "--quiet")
	}
	if shallow {
		args = append(args, "--depth", "1")
	}
	args = append(args, url, path)

	logger.Debug("Executing: git %s", strings.Join(args, " "))
	cmd, cancel := GitCommand("git", args...)
	defer cancel()

	return runGitCommand(cmd, quiet)
}

// FetchPrune runs git fetch --prune to update remote tracking refs and remove stale ones
func FetchPrune(repoPath string) error {
	logger.Debug("Executing: git fetch --prune in %s", repoPath)
	cmd, cancel := GitCommand("git", "fetch", "--prune")
	defer cancel()
	cmd.Dir = repoPath
	return runGitCommand(cmd, true)
}

// ConfigureFetchRefspec configures the fetch refspec for a remote to enable
// tracking of remote branches. This is needed after bare clones which don't
// automatically set up the refspec.
func ConfigureFetchRefspec(repoPath, remote string) error {
	if repoPath == "" {
		return errors.New("repository path cannot be empty")
	}
	if remote == "" {
		return errors.New("remote name cannot be empty")
	}

	refspec := fmt.Sprintf("+refs/heads/*:refs/remotes/%s/*", remote)
	key := fmt.Sprintf("remote.%s.fetch", remote)

	logger.Debug("Configuring fetch refspec: %s=%s in %s", key, refspec, repoPath)
	cmd, cancel := GitCommand("git", "config", key, refspec)
	defer cancel()
	cmd.Dir = repoPath

	return runGitCommand(cmd, true)
}

// FetchBranch fetches a specific branch from a remote.
func FetchBranch(repoPath, remote, branch string) error {
	if repoPath == "" {
		return errors.New("repository path cannot be empty")
	}
	if remote == "" {
		return errors.New("remote name cannot be empty")
	}
	if branch == "" {
		return errors.New("branch name cannot be empty")
	}

	logger.Debug("Executing: git fetch %s %s in %s", remote, branch, repoPath)
	cmd, cancel := GitCommand("git", "fetch", remote, branch) // nolint:gosec // Validated input
	defer cancel()
	cmd.Dir = repoPath

	return runGitCommand(cmd, true)
}

// RefExists checks if a ref (commit, tag, branch) exists
func RefExists(repoPath, ref string) error {
	cmd, cancel := GitCommand("git", "rev-parse", "--verify", "--quiet", ref)
	defer cancel()
	cmd.Dir = repoPath
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ref %q not found: %w", ref, err)
	}
	return nil
}

// IsInsideGitRepo checks if the given path is inside an existing git repository
func IsInsideGitRepo(path string) bool {
	cmd, cancel := GitCommand("git", "rev-parse", "--show-toplevel")
	defer cancel()
	cmd.Dir = path
	return cmd.Run() == nil
}

// GetGitDir returns the path to the git directory for the given path.
// For worktrees, this resolves the gitdir from the .git file.
func GetGitDir(path string) (string, error) {
	gitPath := filepath.Join(path, ".git")

	if fs.DirectoryExists(gitPath) {
		return gitPath, nil
	}

	if fs.FileExists(gitPath) {
		content, err := os.ReadFile(gitPath) // nolint:gosec // Path is constructed internally
		if err != nil {
			return "", err
		}

		line := strings.TrimSpace(string(content))
		if !strings.HasPrefix(line, "gitdir: ") {
			return "", fmt.Errorf("invalid .git file format")
		}

		gitdir := strings.TrimPrefix(line, "gitdir: ")
		if !filepath.IsAbs(gitdir) {
			gitdir = filepath.Join(path, gitdir)
		}
		return filepath.Clean(gitdir), nil
	}

	return "", fmt.Errorf("not a git repository")
}

// AddRemote adds a new remote to the repository.
func AddRemote(repoPath, name, url string) error {
	if repoPath == "" {
		return errors.New("repository path cannot be empty")
	}
	if name == "" {
		return errors.New("remote name cannot be empty")
	}
	if url == "" {
		return errors.New("remote URL cannot be empty")
	}

	logger.Debug("Executing: git remote add %s %s in %s", name, url, repoPath)
	cmd, cancel := GitCommand("git", "remote", "add", name, url) // nolint:gosec // Validated input
	defer cancel()
	cmd.Dir = repoPath

	return runGitCommand(cmd, true)
}

// RemoveRemote removes a remote with the given name.
func RemoveRemote(repoPath, name string) error {
	if repoPath == "" {
		return errors.New("repository path cannot be empty")
	}
	if name == "" {
		return errors.New("remote name cannot be empty")
	}

	logger.Debug("Executing: git remote remove %s in %s", name, repoPath)
	cmd, cancel := GitCommand("git", "remote", "remove", name) // nolint:gosec // Validated input
	defer cancel()
	cmd.Dir = repoPath

	return runGitCommand(cmd, true)
}

// RemoteExists checks if a remote with the given name exists.
func RemoteExists(repoPath, name string) (bool, error) {
	if repoPath == "" {
		return false, errors.New("repository path cannot be empty")
	}
	if name == "" {
		return false, errors.New("remote name cannot be empty")
	}

	cmd, cancel := GitCommand("git", "remote", "get-url", name) // nolint:gosec // Validated input
	defer cancel()
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		// Exit code 2 means remote not found - this is expected, not an error
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 2 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// GetRemoteURL returns the URL for a named remote.
func GetRemoteURL(repoPath, name string) (string, error) {
	if repoPath == "" {
		return "", errors.New("repository path cannot be empty")
	}
	if name == "" {
		return "", errors.New("remote name cannot be empty")
	}

	cmd, cancel := GitCommand("git", "remote", "get-url", name) // nolint:gosec // Validated input
	defer cancel()
	cmd.Dir = repoPath

	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get URL for remote %q: %w", name, err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

// ListIgnoredFiles returns a list of git-ignored files in the given directory.
func ListIgnoredFiles(dir string) ([]string, error) {
	logger.Debug("Executing: git ls-files --others --ignored --exclude-standard in %s", dir)
	cmd, cancel := GitCommand("git", "ls-files", "--others", "--ignored", "--exclude-standard")
	defer cancel()
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list ignored files: %w", err)
	}

	if len(output) == 0 {
		return nil, nil
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}

// IsRemoteReachable checks if a remote is accessible.
func IsRemoteReachable(repoPath, remote string) bool {
	if repoPath == "" || remote == "" {
		return false
	}

	logger.Debug("Checking if remote %s is reachable in %s", remote, repoPath)
	cmd, cancel := GitCommand("git", "ls-remote", "--heads", remote) //nolint:gosec
	defer cancel()
	cmd.Dir = repoPath

	cmd.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -o BatchMode=yes -o ConnectTimeout=5",
	)

	return cmd.Run() == nil
}

// ListRemotes returns a list of configured remote names for the repository.
func ListRemotes(repoPath string) ([]string, error) {
	if repoPath == "" {
		return nil, errors.New("repository path cannot be empty")
	}

	logger.Debug("Executing: git remote in %s", repoPath)
	cmd, cancel := GitCommand("git", "remote")
	defer cancel()
	cmd.Dir = repoPath

	output, err := executeWithOutputBuffer(cmd)
	if err != nil {
		return nil, err
	}

	var remotes []string
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			remotes = append(remotes, line)
		}
	}

	return remotes, scanner.Err()
}
