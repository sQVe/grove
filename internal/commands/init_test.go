package commands

import (
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitCommandConvertWithMockExecutor(t *testing.T) {
	// Test that the mock executor is properly called for safety checks
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSafeRepositoryState()

	// Test that the mock executor responds to safety check commands
	output, err := mockExecutor.Execute("status", "--porcelain=v1")
	require.NoError(t, err)
	assert.Empty(t, output)

	output, err = mockExecutor.Execute("stash", "list")
	require.NoError(t, err)
	assert.Empty(t, output)

	// Test unhandled command returns error
	_, err = mockExecutor.Execute("unhandled", "command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mock: unhandled git command")
}

func TestInitCommandConvertCannotSpecifyArgs(t *testing.T) {
	cmd := NewInitCmd()
	cmd.SetArgs([]string{"--convert", "some-arg"})

	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot specify arguments when using --convert flag")
}
