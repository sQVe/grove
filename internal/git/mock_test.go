package git

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGitExecutor is a mock implementation of GitExecutor for testing.
type MockGitExecutor struct {
	// Commands stores the executed commands for verification
	Commands [][]string
	// Responses maps command patterns to their responses
	Responses map[string]MockResponse
	// CallCount tracks how many times Execute was called
	CallCount int
}

// MockResponse defines a mock git command response.
type MockResponse struct {
	Output string
	Error  error
}

// NewMockGitExecutor creates a new mock executor with default responses.
func NewMockGitExecutor() *MockGitExecutor {
	return &MockGitExecutor{
		Commands:  [][]string{},
		Responses: make(map[string]MockResponse),
		CallCount: 0,
	}
}

// Execute simulates git command execution.
func (m *MockGitExecutor) Execute(args ...string) (string, error) {
	m.CallCount++
	m.Commands = append(m.Commands, args)

	// Create a command key for lookup
	cmdKey := strings.Join(args, " ")

	// Check for exact match first
	if response, exists := m.Responses[cmdKey]; exists {
		return response.Output, response.Error
	}

	// Check for pattern matches
	for pattern, response := range m.Responses {
		if strings.HasPrefix(cmdKey, pattern) {
			return response.Output, response.Error
		}
	}

	// Default response for unmatched commands
	return "", fmt.Errorf("mock: unhandled git command: %s", cmdKey)
}

// SetResponse configures a response for a specific command pattern.
func (m *MockGitExecutor) SetResponse(pattern, output string, err error) {
	m.Responses[pattern] = MockResponse{
		Output: output,
		Error:  err,
	}
}

// SetSuccessResponse configures a successful response.
func (m *MockGitExecutor) SetSuccessResponse(pattern, output string) {
	m.SetResponse(pattern, output, nil)
}

// SetErrorResponse configures an error response.
func (m *MockGitExecutor) SetErrorResponse(pattern, errMsg string) {
	m.SetResponse(pattern, "", fmt.Errorf("%s", errMsg))
}

// Reset clears all recorded commands and responses.
func (m *MockGitExecutor) Reset() {
	m.Commands = [][]string{}
	m.CallCount = 0
	m.Responses = make(map[string]MockResponse)
}

// LastCommand returns the last executed command.
func (m *MockGitExecutor) LastCommand() []string {
	if len(m.Commands) == 0 {
		return nil
	}
	return m.Commands[len(m.Commands)-1]
}

// HasCommand checks if a specific command was executed.
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

// TestMockGitExecutor tests the mock executor itself.
func TestMockGitExecutor(t *testing.T) {
	mock := NewMockGitExecutor()

	// Test default behavior
	_, err := mock.Execute("unknown", "command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unhandled git command")

	// Test configured response
	mock.SetSuccessResponse("test", "output")
	output, err := mock.Execute("test")
	require.NoError(t, err)
	assert.Equal(t, "output", output)

	// Test error response
	mock.SetErrorResponse("fail", "test error")
	_, err = mock.Execute("fail")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test error")

	// Test command tracking
	assert.Equal(t, 3, mock.CallCount)
	assert.True(t, mock.HasCommand("test"))
	assert.False(t, mock.HasCommand("nonexistent"))
}

func TestCloneBareWithExecutor(t *testing.T) {
	tests := []struct {
		name        string
		repoURL     string
		targetDir   string
		mockOutput  string
		mockError   error
		expectError bool
	}{
		{
			name:        "successful clone",
			repoURL:     "https://github.com/user/repo.git",
			targetDir:   "/tmp/test",
			mockOutput:  "",
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "clone failure",
			repoURL:     "https://invalid.com/repo.git",
			targetDir:   "/tmp/test",
			mockOutput:  "",
			mockError:   fmt.Errorf("clone failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockGitExecutor()
			mock.SetResponse("clone --bare", tt.mockOutput, tt.mockError)

			err := CloneBareWithExecutor(mock, tt.repoURL, tt.targetDir)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify the correct command was called
			assert.True(t, mock.HasCommand("clone", "--bare", tt.repoURL, tt.targetDir))
		})
	}
}

func TestConfigureRemoteTrackingWithExecutor(t *testing.T) {
	tests := []struct {
		name         string
		configError  error
		fetchError   error
		expectError  bool
		expectConfig bool
		expectFetch  bool
	}{
		{
			name:         "successful configuration",
			configError:  nil,
			fetchError:   nil,
			expectError:  false,
			expectConfig: true,
			expectFetch:  true,
		},
		{
			name:         "config command fails",
			configError:  fmt.Errorf("config failed"),
			fetchError:   nil,
			expectError:  true,
			expectConfig: true,
			expectFetch:  false,
		},
		{
			name:         "fetch command fails",
			configError:  nil,
			fetchError:   fmt.Errorf("fetch failed"),
			expectError:  true,
			expectConfig: true,
			expectFetch:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := NewMockGitExecutor()
			mock.SetResponse("config", "", tt.configError)
			mock.SetResponse("fetch", "", tt.fetchError)

			err := ConfigureRemoteTrackingWithExecutor(mock)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify expected commands were called
			if tt.expectConfig {
				assert.True(t, mock.HasCommand("config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"))
			}
			if tt.expectFetch {
				assert.True(t, mock.HasCommand("fetch"))
			}
		})
	}
}

func TestSetupUpstreamBranchesWithExecutor(t *testing.T) {
	tests := getSetupUpstreamTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSetupUpstreamTest(t, tt)
		})
	}
}

type setupUpstreamTestCase struct {
	name           string
	branchOutput   string
	branchError    error
	upstreamErrors map[string]error
	expectError    bool
}

func getSetupUpstreamTestCases() []setupUpstreamTestCase {
	return []setupUpstreamTestCase{
		{
			name:         "successful setup with branches",
			branchOutput: "main\nfeature\ndevelop",
			branchError:  nil,
			upstreamErrors: map[string]error{
				"main":    nil,
				"feature": nil,
				"develop": nil,
			},
			expectError: false,
		},
		{
			name:         "for-each-ref fails",
			branchOutput: "",
			branchError:  fmt.Errorf("refs failed"),
			expectError:  true,
		},
		{
			name:         "no branches",
			branchOutput: "",
			branchError:  nil,
			expectError:  false,
		},
		{
			name:         "some upstream failures (should not error)",
			branchOutput: "main\nfeature",
			branchError:  nil,
			upstreamErrors: map[string]error{
				"main":    nil,
				"feature": fmt.Errorf("no remote branch"),
			},
			expectError: false,
		},
	}
}

func runSetupUpstreamTest(t *testing.T, tt setupUpstreamTestCase) {
	t.Helper()
	mock := NewMockGitExecutor()
	mock.SetResponse("for-each-ref", tt.branchOutput, tt.branchError)

	// Set up upstream responses
	for branch, err := range tt.upstreamErrors {
		pattern := fmt.Sprintf("branch --set-upstream-to=origin/%s %s", branch, branch)
		mock.SetResponse(pattern, "", err)
	}

	err := SetupUpstreamBranchesWithExecutor(mock)

	if tt.expectError {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}

	// Verify for-each-ref was called
	assert.True(t, mock.HasCommand("for-each-ref", "--format=%(refname:short)", "refs/heads"))
}
