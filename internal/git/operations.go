package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitExecutor defines the interface for executing git commands.
type GitExecutor interface {
	Execute(args ...string) (string, error)
}

// DefaultGitExecutor implements GitExecutor using real git commands.
type DefaultGitExecutor struct{}

// Execute runs a real git command.
func (e *DefaultGitExecutor) Execute(args ...string) (string, error) {
	return ExecuteGit(args...)
}

// DefaultExecutor is the default git command executor.
var DefaultExecutor GitExecutor = &DefaultGitExecutor{}

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

// Runs a git command with the given arguments and returns stdout.
// If the command fails, it returns a GitError with stderr and exit code.
func ExecuteGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)

	stdout, err := cmd.Output()
	if err != nil {
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
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

// Runs git clone --bare for the given repository URL.
func CloneBare(repoURL, targetDir string) error {
	return CloneBareWithExecutor(DefaultExecutor, repoURL, targetDir)
}

// CloneBareWithExecutor runs git clone --bare using the specified executor.
func CloneBareWithExecutor(executor GitExecutor, repoURL, targetDir string) error {
	_, err := executor.Execute("clone", "--bare", repoURL, targetDir)
	return err
}

// Writes a .git file with gitdir pointing to the bare repository.
func CreateGitFile(mainDir, bareDir string) error {
	gitFilePath := filepath.Join(mainDir, ".git")

	// Make bareDir relative to mainDir if possible, otherwise use absolute path
	relPath, err := filepath.Rel(mainDir, bareDir)
	if err != nil {
		relPath = bareDir
	}

	content := fmt.Sprintf("gitdir: %s\n", relPath)
	return os.WriteFile(gitFilePath, []byte(content), 0600)
}

// Sets up fetch refspec and fetches all remote branches.
func ConfigureRemoteTracking() error {
	return ConfigureRemoteTrackingWithExecutor(DefaultExecutor)
}

// ConfigureRemoteTrackingWithExecutor sets up fetch refspec using the specified executor.
func ConfigureRemoteTrackingWithExecutor(executor GitExecutor) error {
	// Set up fetch refspec to get all remote branches
	_, err := executor.Execute("config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	if err != nil {
		return err
	}

	// Fetch all remote branches
	_, err = executor.Execute("fetch")
	return err
}

// Configures branch.*.remote for existing local branches.
func SetupUpstreamBranches() error {
	return SetupUpstreamBranchesWithExecutor(DefaultExecutor)
}

// SetupUpstreamBranchesWithExecutor configures upstream tracking using the specified executor.
func SetupUpstreamBranchesWithExecutor(executor GitExecutor) error {
	// Get all local branches
	output, err := executor.Execute("for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return err
	}

	branches := strings.Split(strings.TrimSpace(output), "\n")
	for _, branch := range branches {
		if branch == "" {
			continue
		}

		// Set upstream tracking for each branch
		_, err := executor.Execute("branch", "--set-upstream-to=origin/"+branch, branch)
		if err != nil {
			// Continue if this branch doesn't exist on remote
			continue
		}
	}

	return nil
}

// Runs git init --bare in the target directory.
func InitBare(targetDir string) error {
	_, err := ExecuteGit("init", "--bare", targetDir)
	return err
}
