package remove

import (
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestRemoveService_Creation(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	log := newTestLogger()

	service := NewRemoveServiceImpl(mockExecutor, log)
	assert.NotNil(t, service)
}

func TestRemoveService_EmptyPath(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	log := newTestLogger()
	service := NewRemoveServiceImpl(mockExecutor, log)

	err := service.RemoveWorktree("", RemoveOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree path cannot be empty")
}

func TestRemoveService_InvalidOptions(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	log := newTestLogger()
	service := NewRemoveServiceImpl(mockExecutor, log)

	err := service.RemoveWorktree("/test/path", RemoveOptions{Days: -1})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid options")
}

func TestRemoveService_BulkInvalidCriteria(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	log := newTestLogger()
	service := NewRemoveServiceImpl(mockExecutor, log)

	_, err := service.RemoveBulk(BulkCriteria{}, RemoveOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid bulk criteria")
}
