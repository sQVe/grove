package testutils

import (
	"github.com/sqve/grove/internal/git"
	"github.com/stretchr/testify/mock"
)

// MockGitCommander enables isolated testing by replacing real Git operations with predictable mock responses.
// This allows testing command logic without requiring actual Git repositories or network access.
type MockGitCommander struct {
	mock.Mock
}

// Run executes a mocked Git command with the given arguments.
// Expectations must be set using On() method before calling this method.
// Returns mocked stdout, stderr, and error based on configured expectations.
func (m *MockGitCommander) Run(workDir string, args ...string) (stdout, stderr []byte, err error) {
	// Convert variadic args to slice for mock framework
	mockArgs := []interface{}{workDir}
	for _, arg := range args {
		mockArgs = append(mockArgs, arg)
	}

	called := m.Called(mockArgs...)

	return called.Get(0).([]byte), called.Get(1).([]byte), called.Error(2)
}

// RunQuiet executes a mocked Git command without logging failures.
// Expectations must be set using On() method before calling this method.
// Returns mocked error based on configured expectations.
func (m *MockGitCommander) RunQuiet(workDir string, args ...string) error {
	// Convert variadic args to slice for mock framework
	mockArgs := []interface{}{workDir}
	for _, arg := range args {
		mockArgs = append(mockArgs, arg)
	}

	called := m.Called(mockArgs...)

	return called.Error(0)
}

// NewMockGitCommander creates a new MockGitCommander instance.
// This factory function ensures consistent mock creation across tests.
func NewMockGitCommander() *MockGitCommander {
	return &MockGitCommander{}
}

// ExpectRun sets up an expectation for the Run method with specific arguments.
// This is a convenience method for common test scenarios.
func (m *MockGitCommander) ExpectRun(workDir string, args []string) *mock.Call {
	// Convert to interface slice for mock framework
	mockArgs := []interface{}{workDir}
	for _, arg := range args {
		mockArgs = append(mockArgs, arg)
	}

	return m.On("Run", mockArgs...)
}

// ExpectRunQuiet sets up an expectation for the RunQuiet method with specific arguments.
// This is a convenience method for common test scenarios.
func (m *MockGitCommander) ExpectRunQuiet(workDir string, args []string) *mock.Call {
	// Convert to interface slice for mock framework
	mockArgs := []interface{}{workDir}
	for _, arg := range args {
		mockArgs = append(mockArgs, arg)
	}

	return m.On("RunQuiet", mockArgs...)
}

// Ensure MockGitCommander implements git.Commander interface at compile time.
var _ git.Commander = (*MockGitCommander)(nil)
