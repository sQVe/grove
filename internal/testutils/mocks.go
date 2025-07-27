package testutils

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// for use across all test packages. It combines features from all mock implementations:.
// - Command tracking and call counting.
// - Helper methods for verification.
// - Delay simulation capability.
// - Special command handling.
// - Multiple response formats for flexibility.
type MockGitExecutor struct {
	// Commands stores the executed commands for verification.
	Commands [][]string
	// Responses maps command patterns to their responses.
	Responses map[string]MockResponse
	// CallCount tracks how many times Execute was called.
	CallCount int
	// responses provides legacy support for simple string responses.
	responses map[string]MockResponse
	// delays allows simulation of command execution delays.
	delays map[string]time.Duration
	// regexPatterns stores regex patterns and their responses for flexible command matching.
	regexPatterns []RegexPattern
}

type RegexPattern struct {
	Pattern  *regexp.Regexp
	Response MockResponse
}

type MockResponse struct {
	Output string
	Error  error
}

func NewMockGitExecutor() *MockGitExecutor {
	return &MockGitExecutor{
		Commands:      [][]string{},
		Responses:     make(map[string]MockResponse),
		CallCount:     0,
		responses:     make(map[string]MockResponse),
		delays:        make(map[string]time.Duration),
		regexPatterns: []RegexPattern{},
	}
}

func (m *MockGitExecutor) Execute(args ...string) (string, error) {
	return m.executeInternal(args)
}

// Behaves identically to Execute for testing purposes.
func (m *MockGitExecutor) ExecuteQuiet(args ...string) (string, error) {
	return m.executeInternal(args)
}

func (m *MockGitExecutor) ExecuteWithContext(ctx context.Context, args ...string) (string, error) {
	// Check if context is cancelled before execution.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	return m.executeInternal(args)
}

// executeInternal contains the common execution logic for both Execute methods.
func (m *MockGitExecutor) executeInternal(args []string) (string, error) {
	m.CallCount++
	m.Commands = append(m.Commands, args)

	// Create a command key for lookup.
	cmdKey := strings.Join(args, " ")
	cmdSliceKey := fmt.Sprintf("%v", args)

	// Handle delay if configured.
	if delay, exists := m.delays[cmdKey]; exists {
		time.Sleep(delay)
	}

	// Special handling for clone command to create directory (from commands mock).
	if len(args) >= 3 && args[0] == "clone" && args[1] == "--bare" {
		targetDir := args[3]
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			return "", err
		}
	}

	// Check responses map (new format).
	if response, exists := m.responses[cmdKey]; exists {
		return response.Output, response.Error
	}

	// Check Responses map (old format with exact match).
	if response, exists := m.Responses[cmdKey]; exists {
		return response.Output, response.Error
	}

	// Check for slice-based key (from utils mock).
	if response, exists := m.Responses[cmdSliceKey]; exists {
		return response.Output, response.Error
	}

	// Check for pattern matches (useful for commands with variable parts).
	for pattern, response := range m.Responses {
		if strings.HasPrefix(cmdKey, pattern) {
			return response.Output, response.Error
		}
	}

	// Check for pattern matches in responses map.
	for pattern, response := range m.responses {
		if strings.HasPrefix(cmdKey, pattern) {
			return response.Output, response.Error
		}
	}

	// Check for simple command matches (from commands mock).
	if len(args) > 0 {
		for pattern, response := range m.Responses {
			if args[0] == pattern {
				return response.Output, response.Error
			}
		}
	}

	// Check for regex pattern matches (most flexible matching).
	for _, regexPattern := range m.regexPatterns {
		if regexPattern.Pattern.MatchString(cmdKey) {
			return regexPattern.Response.Output, regexPattern.Response.Error
		}
	}

	// Return error for unhandled commands.
	return "", fmt.Errorf("mock: unhandled git command: %s", cmdKey)
}

func (m *MockGitExecutor) SetResponse(command, output string, err error) {
	m.responses[command] = MockResponse{Output: output, Error: err}
}

func (m *MockGitExecutor) SetResponseSlice(args []string, output string, err error) {
	key := fmt.Sprintf("%v", args)
	m.Responses[key] = MockResponse{Output: output, Error: err}
}

func (m *MockGitExecutor) SetSuccessResponse(command, output string) {
	m.responses[command] = MockResponse{Output: output, Error: nil}
}

func (m *MockGitExecutor) SetErrorResponse(command string, err error) {
	m.responses[command] = MockResponse{Output: "", Error: err}
}

func (m *MockGitExecutor) SetErrorResponseWithMessage(command, errMsg string) {
	m.responses[command] = MockResponse{Output: "", Error: fmt.Errorf("%s", errMsg)}
}

func (m *MockGitExecutor) SetDelay(command string, delay time.Duration) {
	m.delays[command] = delay
}

// This provides more flexible command matching than string-based patterns.
func (m *MockGitExecutor) SetResponsePattern(pattern *regexp.Regexp, output string, err error) {
	m.regexPatterns = append(m.regexPatterns, RegexPattern{
		Pattern:  pattern,
		Response: MockResponse{Output: output, Error: err},
	})
}

func (m *MockGitExecutor) SetSuccessResponsePattern(pattern *regexp.Regexp, output string) {
	m.SetResponsePattern(pattern, output, nil)
}

func (m *MockGitExecutor) SetErrorResponsePattern(pattern *regexp.Regexp, err error) {
	m.SetResponsePattern(pattern, "", err)
}

func (m *MockGitExecutor) SetErrorResponsePatternWithMessage(pattern *regexp.Regexp, errMsg string) {
	m.SetResponsePattern(pattern, "", fmt.Errorf("%s", errMsg))
}

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

func (m *MockGitExecutor) SetConversionState() {
	m.SetSafeRepositoryState()
	m.SetSuccessResponse("rev-parse --is-bare-repository", "false")
	m.SetSuccessResponse("config --get core.bare", "false")
	m.SetSuccessResponse("symbolic-ref HEAD", "refs/heads/main")
	m.SetSuccessResponse("rev-parse --abbrev-ref HEAD", "main")
	m.SetSuccessResponse("branch --show-current", "main")
	m.SetSuccessResponse("worktree add", "")
}

func (m *MockGitExecutor) LastCommand() []string {
	if len(m.Commands) == 0 {
		return nil
	}
	return m.Commands[len(m.Commands)-1]
}

func (m *MockGitExecutor) HasCommand(expected ...string) bool {
	for _, cmd := range m.Commands {
		if len(cmd) == len(expected) {
			match := true
			for i, arg := range expected {
				if cmd[i] != arg {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

// Reset clears all configured responses and recorded commands.
func (m *MockGitExecutor) Reset() {
	m.Commands = [][]string{}
	m.CallCount = 0
	m.Responses = make(map[string]MockResponse)
	m.responses = make(map[string]MockResponse)
	m.delays = make(map[string]time.Duration)
	m.regexPatterns = []RegexPattern{}
}
