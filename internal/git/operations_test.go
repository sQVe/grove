package git

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type MockCommander struct {
	mock.Mock
}

func (m *MockCommander) Run(workDir string, args ...string) (stdout, stderr []byte, err error) {
	mockArgs := m.Called(workDir, args)
	return mockArgs.Get(0).([]byte), mockArgs.Get(1).([]byte), mockArgs.Error(2)
}

func (m *MockCommander) RunQuiet(workDir string, args ...string) error {
	mockArgs := m.Called(workDir, args)
	return mockArgs.Error(0)
}

func TestDefaultGitExecutor_Interface(t *testing.T) {
	executor := &DefaultGitExecutor{}

	// Verify interface compliance
	var _ GitExecutor = executor

	// Test methods exist and don't panic
	assert.NotPanics(t, func() {
		_, _ = executor.Execute("--version")
	})

	assert.NotPanics(t, func() {
		_, _ = executor.ExecuteQuiet("--version")
	})

	assert.NotPanics(t, func() {
		_, _ = executor.ExecuteWithContext(context.Background(), "--version")
	})
}

func TestNewCommanderAdapter(t *testing.T) {
	mockCommander := &MockCommander{}

	adapter := NewCommanderAdapter(mockCommander)

	assert.NotNil(t, adapter)
	assert.Equal(t, mockCommander, adapter.commander)
	assert.Equal(t, "", adapter.workDir)
}

func TestNewCommanderAdapterWithWorkDir(t *testing.T) {
	mockCommander := &MockCommander{}
	workDir := "/test/work/dir"

	adapter := NewCommanderAdapterWithWorkDir(mockCommander, workDir)

	assert.NotNil(t, adapter)
	assert.Equal(t, mockCommander, adapter.commander)
	assert.Equal(t, workDir, adapter.workDir)
}

func TestCommanderAdapter_Execute_Success(t *testing.T) {
	mockCommander := &MockCommander{}
	adapter := NewCommanderAdapter(mockCommander)

	expectedStdout := []byte("test output\n")
	expectedStderr := []byte("")

	mockCommander.On("Run", "", []string{"status", "--porcelain"}).
		Return(expectedStdout, expectedStderr, nil)

	result, err := adapter.Execute("status", "--porcelain")

	assert.NoError(t, err)
	assert.Equal(t, "test output", result) // Should be trimmed
	mockCommander.AssertExpectations(t)
}

func TestCommanderAdapter_Execute_WithWorkDir(t *testing.T) {
	mockCommander := &MockCommander{}
	workDir := "/test/repo"
	adapter := NewCommanderAdapterWithWorkDir(mockCommander, workDir)

	expectedStdout := []byte("  branch output  \n")
	expectedStderr := []byte("")

	mockCommander.On("Run", workDir, []string{"branch", "-a"}).
		Return(expectedStdout, expectedStderr, nil)

	result, err := adapter.Execute("branch", "-a")

	assert.NoError(t, err)
	assert.Equal(t, "branch output", result)
	mockCommander.AssertExpectations(t)
}

func TestCommanderAdapter_Execute_Error(t *testing.T) {
	mockCommander := &MockCommander{}
	adapter := NewCommanderAdapter(mockCommander)

	expectedErr := errors.New("git command failed")

	mockCommander.On("Run", "", []string{"invalid", "command"}).
		Return([]byte(""), []byte("error output"), expectedErr)

	result, err := adapter.Execute("invalid", "command")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, "", result)
	mockCommander.AssertExpectations(t)
}

func TestCommanderAdapter_ExecuteQuiet_Success(t *testing.T) {
	mockCommander := &MockCommander{}
	adapter := NewCommanderAdapter(mockCommander)

	mockCommander.On("RunQuiet", "", []string{"rev-parse", "--git-dir"}).
		Return(nil)
	mockCommander.On("Run", "", []string{"rev-parse", "--git-dir"}).
		Return([]byte(".git\n"), []byte(""), nil)

	result, err := adapter.ExecuteQuiet("rev-parse", "--git-dir")

	assert.NoError(t, err)
	assert.Equal(t, ".git", result)
	mockCommander.AssertExpectations(t)
}

func TestCommanderAdapter_ExecuteQuiet_QuietFails(t *testing.T) {
	mockCommander := &MockCommander{}
	adapter := NewCommanderAdapter(mockCommander)

	expectedErr := errors.New("git operation failed")

	mockCommander.On("RunQuiet", "", []string{"invalid", "command"}).
		Return(expectedErr)

	result, err := adapter.ExecuteQuiet("invalid", "command")

	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, "", result)
	mockCommander.AssertExpectations(t)
}

func TestCommanderAdapter_ExecuteWithContext_Success(t *testing.T) {
	mockCommander := &MockCommander{}
	adapter := NewCommanderAdapter(mockCommander)

	ctx := context.Background()
	expectedStdout := []byte("commit hash\n")

	mockCommander.On("Run", "", []string{"rev-parse", "HEAD"}).
		Return(expectedStdout, []byte(""), nil)

	result, err := adapter.ExecuteWithContext(ctx, "rev-parse", "HEAD")

	assert.NoError(t, err)
	assert.Equal(t, "commit hash", result)
	mockCommander.AssertExpectations(t)
}

func TestCommanderAdapter_EmptyArguments(t *testing.T) {
	mockCommander := &MockCommander{}
	adapter := NewCommanderAdapter(mockCommander)

	expectedStdout := []byte("git version 2.x.x")

	mockCommander.On("Run", "", mock.MatchedBy(func(args []string) bool {
		return len(args) == 0
	})).
		Return(expectedStdout, []byte(""), nil)

	result, err := adapter.Execute()

	assert.NoError(t, err)
	assert.Equal(t, "git version 2.x.x", result)
	mockCommander.AssertExpectations(t)
}

func TestCommanderAdapter_OutputTrimming(t *testing.T) {
	mockCommander := &MockCommander{}
	adapter := NewCommanderAdapter(mockCommander)

	outputWithWhitespace := []byte("  \n\ttest output\n\n  ")

	mockCommander.On("Run", "", []string{"status"}).
		Return(outputWithWhitespace, []byte(""), nil)

	result, err := adapter.Execute("status")

	assert.NoError(t, err)
	assert.Equal(t, "test output", result)
	mockCommander.AssertExpectations(t)
}
