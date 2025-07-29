package remove

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestNewRemoveCmd(t *testing.T) {
	cmd := NewRemoveCmd()

	assert.Equal(t, "remove [worktree-path]", cmd.Use)
	assert.Equal(t, "Remove worktrees safely with optional branch cleanup", cmd.Short)
	assert.NotEmpty(t, cmd.Long)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.ValidArgsFunction)

	// Check flags are properly defined
	forceFlag := cmd.Flags().Lookup("force")
	assert.NotNil(t, forceFlag)

	dryRunFlag := cmd.Flags().Lookup("dry-run")
	assert.NotNil(t, dryRunFlag)

	deleteBranchFlag := cmd.Flags().Lookup("delete-branch")
	assert.NotNil(t, deleteBranchFlag)
}

func TestCompleteWorktreePaths_Basic(t *testing.T) {
	// Basic test that the function doesn't panic
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetResponse("worktree list --porcelain", "", nil)

	// Replace the default executor temporarily
	originalExecutor := git.DefaultExecutor
	git.DefaultExecutor = mockExecutor
	defer func() { git.DefaultExecutor = originalExecutor }()

	_, directive := completeWorktreePaths(nil, []string{}, "")

	// Just verify it returns without error
	assert.Equal(t, cobra.ShellCompDirectiveDefault, directive)
}

func TestPresentBulkResults_Basic(t *testing.T) {
	// Test that the function doesn't panic with various inputs
	tests := []RemoveResults{
		{}, // Empty results
		{
			Removed: []string{"/path1"},
			Summary: RemoveSummary{Total: 1, Removed: 1},
		},
	}

	for _, results := range tests {
		assert.NotPanics(t, func() {
			presentBulkResults(&results)
		})
	}
}
