package testutils

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// MockGitExecutor provides comprehensive Git command mocking for use across all test packages.
// It combines features from all mock implementations: command tracking, call counting,
// helper methods for verification, delay simulation, special command handling,
// and multiple response formats for flexibility.
type MockGitExecutor struct {
	Commands      [][]string
	Responses     map[string]MockResponse
	CallCount     int
	responses     map[string]MockResponse    // Legacy support for simple string responses.
	delays        map[string]time.Duration   // Simulation of command execution delays.
	regexPatterns []RegexPattern             // Flexible command matching via regex patterns.
	contexts      map[string]context.Context // Track contexts for cancellation testing
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
		contexts:      make(map[string]context.Context),
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
	cmdKey := strings.Join(args, " ")
	m.contexts[cmdKey] = ctx

	// Check if context is cancelled before execution..
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	// Simulate delay if set for this command, respecting context cancellation.
	if delay, exists := m.delays[cmdKey]; exists {
		select {
		case <-time.After(delay):
			// Continue with execution.
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}

	return m.executeInternal(args)
}

func (m *MockGitExecutor) executeInternal(args []string) (string, error) {
	m.CallCount++
	m.Commands = append(m.Commands, args)

	cmdKey := strings.Join(args, " ")
	cmdSliceKey := fmt.Sprintf("%v", args)

	if delay, exists := m.delays[cmdKey]; exists {
		time.Sleep(delay)
	}

	// Clone commands require directory creation for test environment compatibility.
	if len(args) >= 3 && args[0] == "clone" && args[1] == "--bare" {
		targetDir := args[3]
		if err := os.MkdirAll(targetDir, 0o750); err != nil {
			return "", err
		}
	}

	if response, exists := m.responses[cmdKey]; exists {
		return response.Output, response.Error
	}

	if response, exists := m.Responses[cmdKey]; exists {
		return response.Output, response.Error
	}

	if response, exists := m.Responses[cmdSliceKey]; exists {
		return response.Output, response.Error
	}

	for pattern, response := range m.Responses {
		if strings.HasPrefix(cmdKey, pattern) {
			return response.Output, response.Error
		}
	}

	for pattern, response := range m.responses {
		if strings.HasPrefix(cmdKey, pattern) {
			return response.Output, response.Error
		}
	}

	if len(args) > 0 {
		for pattern, response := range m.Responses {
			if args[0] == pattern {
				return response.Output, response.Error
			}
		}
	}

	for _, regexPattern := range m.regexPatterns {
		if regexPattern.Pattern.MatchString(cmdKey) {
			return regexPattern.Response.Output, regexPattern.Response.Error
		}
	}

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

// SetDelayedResponse sets a response with a delay for testing context cancellation
func (m *MockGitExecutor) SetDelayedResponse(command, output string, err error, delay time.Duration) {
	m.responses[command] = MockResponse{Output: output, Error: err}
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

// SequentialMockGitExecutor provides advanced mocking for testing scenarios where
// the same command needs to return different responses on successive calls.
// This is particularly useful for testing conflict resolution and retry logic.
type SequentialMockGitExecutor struct {
	responses        map[string][]MockResponse  // Sequential responses for commands
	otherResponses   map[string]MockResponse    // Single responses for other commands
	callCounts       map[string]int             // Track how many times each command was called
	specificCounters map[string]int             // Track specific command patterns (e.g., worktree add calls)
	contexts         map[string]context.Context // Track contexts for cancellation testing
}

// NewSequentialMockGitExecutor creates a new sequential mock git executor.
func NewSequentialMockGitExecutor() *SequentialMockGitExecutor {
	return &SequentialMockGitExecutor{
		responses:        make(map[string][]MockResponse),
		otherResponses:   make(map[string]MockResponse),
		callCounts:       make(map[string]int),
		specificCounters: make(map[string]int),
		contexts:         make(map[string]context.Context),
	}
}

func (m *SequentialMockGitExecutor) Execute(args ...string) (string, error) {
	return m.ExecuteQuiet(args...)
}

// ExecuteQuiet behaves identically to Execute for testing purposes.
func (m *SequentialMockGitExecutor) ExecuteQuiet(args ...string) (string, error) {
	cmdKey := strings.Join(args, " ")

	// Initialize call counts if needed.
	if m.callCounts == nil {
		m.callCounts = make(map[string]int)
	}

	// Track specific command patterns.
	if len(args) >= 2 && args[0] == "worktree" && args[1] == "add" {
		m.specificCounters["worktree_add"]++
	}

	// Check for sequential responses first.
	for pattern, responses := range m.responses {
		if strings.HasPrefix(cmdKey, pattern) || cmdKey == pattern {
			callIndex := m.callCounts[pattern]
			m.callCounts[pattern]++

			if callIndex < len(responses) {
				response := responses[callIndex]
				return response.Output, response.Error
			}
			// If we've exhausted sequential responses, return the last one.
			if len(responses) > 0 {
				response := responses[len(responses)-1]
				return response.Output, response.Error
			}
		}
	}

	// Check other responses.
	if response, exists := m.otherResponses[cmdKey]; exists {
		return response.Output, response.Error
	}

	return "", fmt.Errorf("mock: unhandled git command: %s", cmdKey)
}

func (m *SequentialMockGitExecutor) ExecuteWithContext(ctx context.Context, args ...string) (string, error) {
	cmdKey := strings.Join(args, " ")
	m.contexts[cmdKey] = ctx

	// Check if context is cancelled before execution.
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
	}

	return m.ExecuteQuiet(args...)
}

// SetSequentialResponse sets up multiple responses for the same command pattern.
// Each call to a matching command will return the next response in sequence.
func (m *SequentialMockGitExecutor) SetSequentialResponse(pattern string, responses []MockResponse) {
	m.responses[pattern] = responses
}

// SetSingleResponse sets a single response for a command (same as regular mock).
func (m *SequentialMockGitExecutor) SetSingleResponse(command string, response MockResponse) {
	m.otherResponses[command] = response
}

// GetCallCount returns how many times a command pattern was called.
func (m *SequentialMockGitExecutor) GetCallCount(pattern string) int {
	return m.callCounts[pattern]
}

// GetSpecificCounter returns the count for specific command patterns.
func (m *SequentialMockGitExecutor) GetSpecificCounter(counterName string) int {
	return m.specificCounters[counterName]
}

// Reset clears all configured responses and recorded calls.
func (m *SequentialMockGitExecutor) Reset() {
	m.responses = make(map[string][]MockResponse)
	m.otherResponses = make(map[string]MockResponse)
	m.callCounts = make(map[string]int)
	m.specificCounters = make(map[string]int)
	m.contexts = make(map[string]context.Context)
}
