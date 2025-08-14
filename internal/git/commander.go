package git

import (
	"os/exec"
	"strings"
	"time"

	"github.com/sqve/grove/internal/logger"
)

// Commander abstracts Git command execution to enable dependency injection and testing.
// This abstraction prevents tight coupling to the git binary and allows
// mock implementations to replace real Git execution for isolated testing.
type Commander interface {
	// Run executes a Git command with the given arguments in the specified working directory.
	// Returns stdout, stderr, and any execution error.
	// The working directory parameter allows for execution in different repository contexts.
	Run(workDir string, args ...string) (stdout, stderr []byte, err error)

	// RunQuiet executes a Git command without logging failures.
	// This is useful for operations where failures are expected and should not be logged as errors.
	// Returns only the execution error; stdout/stderr are discarded for quiet operations.
	RunQuiet(workDir string, args ...string) error
}

// LiveGitCommander provides production Git command execution with comprehensive logging and error handling.
// This implementation ensures all Git operations are properly tracked and debuggable in production environments.
type LiveGitCommander struct{}

// NewLiveGitCommander creates a new instance of LiveGitCommander.
func NewLiveGitCommander() *LiveGitCommander {
	return &LiveGitCommander{}
}

// Run executes a Git command with structured logging and error handling.
// It captures both stdout and stderr, providing complete command execution details.
func (c *LiveGitCommander) Run(workDir string, args ...string) (stdout, stderr []byte, err error) {
	log := logger.WithComponent("git_commander")
	start := time.Now()

	log.GitCommand("git", args)
	cmd := exec.Command("git", args...)

	if workDir != "" {
		cmd.Dir = workDir
	}

	stdoutBytes, err := cmd.Output()
	duration := time.Since(start)

	if err != nil {
		var stderrBytes []byte
		if exitError, ok := err.(*exec.ExitError); ok {
			stderrBytes = exitError.Stderr
		}

		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}

		gitErr := &GitError{
			Command:  "git",
			Args:     args,
			Stderr:   string(stderrBytes),
			ExitCode: exitCode,
		}

		log.GitResult("git", false, string(stderrBytes), "duration", duration, "workdir", workDir)
		return stdoutBytes, stderrBytes, gitErr
	}

	log.GitResult("git", true, string(stdoutBytes), "duration", duration, "workdir", workDir)
	return stdoutBytes, nil, nil
}

// RunQuiet executes a Git command without logging failures.
// Successful operations are still logged at debug level for debugging purposes.
func (c *LiveGitCommander) RunQuiet(workDir string, args ...string) error {
	log := logger.WithComponent("git_commander")
	start := time.Now()

	log.GitCommand("git", args)
	cmd := exec.Command("git", args...)

	if workDir != "" {
		cmd.Dir = workDir
	}

	stdout, err := cmd.Output()
	duration := time.Since(start)

	if err != nil {
		var stderrString string
		if exitError, ok := err.(*exec.ExitError); ok {
			stderrString = string(exitError.Stderr)
		}

		exitCode := 0
		if cmd.ProcessState != nil {
			exitCode = cmd.ProcessState.ExitCode()
		}

		gitErr := &GitError{
			Command:  "git",
			Args:     args,
			Stderr:   stderrString,
			ExitCode: exitCode,
		}

		// Note: We don't log failures for quiet execution.
		// The caller expects failures and will handle them appropriately.
		return gitErr
	}

	output := strings.TrimSpace(string(stdout))
	log.GitResult("git", true, output, "duration", duration, "workdir", workDir)
	return nil
}

// DefaultCommander provides a default instance of LiveGitCommander for production use.
var DefaultCommander Commander = NewLiveGitCommander()
