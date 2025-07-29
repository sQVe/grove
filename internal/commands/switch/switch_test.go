package switchcmd

import (
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNewSwitchCmd(t *testing.T) {
	cmd := NewSwitchCmd()

	assert.NotNil(t, cmd, "NewSwitchCmd should return a command")
	assert.Equal(t, "switch", cmd.Use[:6], "Command should be named 'switch'")
	assert.Equal(t, "Switch to an existing worktree", cmd.Short, "Command should have correct short description")
}

func TestNewSwitchService(t *testing.T) {
	executor := &testutils.MockGitExecutor{}
	service := NewSwitchService(executor)

	assert.NotNil(t, service, "NewSwitchService should return a service")
}
