package testutils

import (
	"context"
	"fmt"
	"strings"
)

// MockGitExecutor is a unified mock implementation of the GitExecutor interface
// for use across all test packages.
type MockGitExecutor struct {
	responses map[string]MockResponse
}

// MockResponse represents a mock response for git commands.
type MockResponse struct {
	Output string
	Error  error
}

// NewMockGitExecutor creates a new mock git executor with empty responses.
func NewMockGitExecutor() *MockGitExecutor {
	return &MockGitExecutor{
		responses: make(map[string]MockResponse),
	}
}

// Execute implements the GitExecutor interface by returning pre-configured responses.
func (m *MockGitExecutor) Execute(args ...string) (string, error) {
	return m.executeInternal(args)
}

// ExecuteWithContext implements the GitExecutor interface with context support.
func (m *MockGitExecutor) ExecuteWithContext(ctx context.Context, args ...string) (string, error) {
	// Check if context is cancelled before execution
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	return m.executeInternal(args)
}

// executeInternal contains the common execution logic for both Execute methods.
func (m *MockGitExecutor) executeInternal(args []string) (string, error) {
	cmdKey := strings.Join(args, " ")

	// First check for exact match
	if response, exists := m.responses[cmdKey]; exists {
		return response.Output, response.Error
	}

	// Then check for prefix matches (useful for commands with variable parts)
	for pattern, response := range m.responses {
		if strings.HasPrefix(cmdKey, pattern) {
			return response.Output, response.Error
		}
	}

	// Return error for unhandled commands
	return "", fmt.Errorf("mock: unhandled git command: %s", cmdKey)
}

// SetResponse sets a response for a specific git command.
func (m *MockGitExecutor) SetResponse(command, output string, err error) {
	m.responses[command] = MockResponse{Output: output, Error: err}
}

// SetSuccessResponse sets a successful response for a git command.
func (m *MockGitExecutor) SetSuccessResponse(command, output string) {
	m.responses[command] = MockResponse{Output: output, Error: nil}
}

// SetErrorResponse sets an error response for a git command.
func (m *MockGitExecutor) SetErrorResponse(command string, err error) {
	m.responses[command] = MockResponse{Output: "", Error: err}
}

// SetSafeRepositoryState configures the mock to return responses indicating
// a repository is safe for conversion (no uncommitted changes, stashes, etc.).
func (m *MockGitExecutor) SetSafeRepositoryState() {
	m.SetSuccessResponse("status --porcelain=v1", "")
	m.SetSuccessResponse("status", "On branch main\nnothing to commit, working tree clean\n")
	m.SetSuccessResponse("stash list", "")
	m.SetSuccessResponse("ls-files --others --exclude-standard", "")
	m.SetSuccessResponse("worktree list", "/path/to/repo  abc123 [main]\n")
	m.SetSuccessResponse(
		"for-each-ref --format=%(refname:short) %(upstream:short) %(upstream:track) refs/heads",
		"main origin/main [up to date]\n",
	)
	m.SetSuccessResponse("for-each-ref --format=%(refname:short) %(upstream) refs/heads", "main origin/main\n")
}

// SetUnsafeRepositoryState configures the mock to return responses indicating
// a repository has safety issues (uncommitted changes, stashes, etc.).
func (m *MockGitExecutor) SetUnsafeRepositoryState() {
	m.SetSuccessResponse("status --porcelain=v1", " M file1.txt\nA  file2.txt\n")
	m.SetSuccessResponse(
		"status",
		"On branch main\nChanges to be committed:\n  new file:   file2.txt\n\nChanges not staged for commit:\n  modified:   file1.txt\n",
	)
	m.SetSuccessResponse("stash list", "stash@{0}: WIP on main: abc123 Last commit\n")
	m.SetSuccessResponse("ls-files --others --exclude-standard", "newfile.txt\ntemp.log\n")
	m.SetSuccessResponse("worktree list", "/path/to/repo        abc123 [main]\n/path/to/feature     def456 [feature]\n")
	m.SetSuccessResponse(
		"for-each-ref --format=%(refname:short) %(upstream:short) %(upstream:track) refs/heads",
		"main origin/main [ahead 2]\nfeature origin/feature [ahead 1]\n",
	)
	m.SetSuccessResponse(
		"for-each-ref --format=%(refname:short) %(upstream) refs/heads",
		"main origin/main\nexperiment\ntemp\n",
	)
}

// SetConversionState configures the mock to support repository conversion operations.
func (m *MockGitExecutor) SetConversionState() {
	// Set up responses for conversion process
	m.SetSafeRepositoryState()
	// Add responses for conversion-specific commands
	m.SetSuccessResponse("rev-parse --is-bare-repository", "false")
	m.SetSuccessResponse("config --get core.bare", "false")
	m.SetSuccessResponse("symbolic-ref HEAD", "refs/heads/main")
	m.SetSuccessResponse("rev-parse --abbrev-ref HEAD", "main")
	m.SetSuccessResponse("branch --show-current", "main")
	m.SetSuccessResponse("worktree add", "")
}

// Reset clears all configured responses.
func (m *MockGitExecutor) Reset() {
	m.responses = make(map[string]MockResponse)
}
