//go:build !integration
// +build !integration

package git

import (
	"context"
	"testing"
	"time"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitExecutorWithContext_Cancellation(t *testing.T) {
	tests := []struct {
		name          string
		timeout       time.Duration
		simulateDelay time.Duration
		expectTimeout bool
	}{
		{
			name:          "operation completes within timeout",
			timeout:       100 * time.Millisecond,
			simulateDelay: 50 * time.Millisecond,
			expectTimeout: false,
		},
		{
			name:          "operation times out",
			timeout:       50 * time.Millisecond,
			simulateDelay: 100 * time.Millisecond,
			expectTimeout: true,
		},
		{
			name:          "immediate cancellation",
			timeout:       1 * time.Nanosecond,
			simulateDelay: 10 * time.Millisecond,
			expectTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()

			// Simulate delay in git execution
			if tt.simulateDelay > 0 {
				mockExecutor.SetDelayedResponse("status", "", nil, tt.simulateDelay)
			} else {
				mockExecutor.SetSuccessResponse("status", "clean")
			}

			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			_, err := mockExecutor.ExecuteWithContext(ctx, "status")

			if tt.expectTimeout {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "context")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGitExecutorWithContext_CancellationPropagation(t *testing.T) {
	t.Run("parent context cancellation propagates to child operations", func(t *testing.T) {
		mockExecutor := testutils.NewMockGitExecutor()
		mockExecutor.SetDelayedResponse("fetch", "", nil, 100*time.Millisecond)
		mockExecutor.SetDelayedResponse("merge", "", nil, 100*time.Millisecond)

		parentCtx, parentCancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer parentCancel()

		// First operation should timeout
		_, err1 := mockExecutor.ExecuteWithContext(parentCtx, "fetch")
		require.Error(t, err1)
		assert.Contains(t, err1.Error(), "context")

		// Second operation with same cancelled context should also fail
		_, err2 := mockExecutor.ExecuteWithContext(parentCtx, "merge")
		require.Error(t, err2)
		assert.Contains(t, err2.Error(), "context")
	})
}

func TestGitExecutorWithContext_ResourceCleanup(t *testing.T) {
	t.Run("resources are cleaned up on context cancellation", func(t *testing.T) {
		mockExecutor := testutils.NewMockGitExecutor()

		// Simulate a long-running operation that gets cancelled
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel the context after a short delay
		go func() {
			time.Sleep(10 * time.Millisecond)
			cancel()
		}()

		mockExecutor.SetDelayedResponse("clone https://example.com/repo.git", "", nil, 100*time.Millisecond)

		_, err := mockExecutor.ExecuteWithContext(ctx, "clone", "https://example.com/repo.git")

		require.Error(t, err)
		assert.Contains(t, err.Error(), "context")

		// Verify that the executor can still be used after cancellation
		mockExecutor.SetSuccessResponse("status", "clean")
		_, err = mockExecutor.Execute("status")
		require.NoError(t, err)
	})
}
