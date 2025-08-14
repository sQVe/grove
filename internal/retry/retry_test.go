package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	groveErrors "github.com/sqve/grove/internal/errors"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, 3, config.MaxAttempts)
	assert.Equal(t, 1*time.Second, config.BaseDelay)
	assert.Equal(t, 10*time.Second, config.MaxDelay)
	assert.True(t, config.JitterEnabled)
}

func TestCalculateDelay(t *testing.T) {
	config := RetryConfig{
		BaseDelay:     1 * time.Second,
		MaxDelay:      10 * time.Second,
		JitterEnabled: false,
	}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"first retry", 1, 1 * time.Second},
		{"second retry", 2, 2 * time.Second},
		{"third retry", 3, 4 * time.Second},
		{"fourth retry", 4, 8 * time.Second},
		{"capped at max", 5, 10 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := calculateDelay(tt.attempt, config)
			assert.Equal(t, tt.expected, delay)
		})
	}
}

func TestCalculateDelayWithJitter(t *testing.T) {
	config := RetryConfig{
		BaseDelay:     1 * time.Second,
		MaxDelay:      10 * time.Second,
		JitterEnabled: true,
	}

	delay := calculateDelay(1, config)

	// With jitter, delay should be within ±25% of base delay
	expectedMin := 750 * time.Millisecond
	expectedMax := 1250 * time.Millisecond

	assert.True(t, delay >= expectedMin && delay <= expectedMax,
		"Delay %v should be between %v and %v", delay, expectedMin, expectedMax)
}

func TestCalculateDelayNegativeJitter(t *testing.T) {
	config := RetryConfig{
		BaseDelay:     10 * time.Millisecond, // Very small base delay
		MaxDelay:      10 * time.Second,
		JitterEnabled: true,
	}

	delay := calculateDelay(1, config)

	// Even with negative jitter, delay should not be less than base delay
	assert.True(t, delay >= config.BaseDelay,
		"Delay %v should not be less than base delay %v", delay, config.BaseDelay)
}

type mockRetryableError struct {
	retryable bool
}

func (e *mockRetryableError) Error() string {
	return "mock retryable error"
}

func (e *mockRetryableError) IsRetryable() bool {
	return e.retryable
}

func TestShouldRetry(t *testing.T) {
	tests := []struct {
		name        string
		err         error
		attempt     int
		maxAttempts int
		expected    bool
	}{
		{
			name:        "max attempts reached",
			err:         errors.New("test error"),
			attempt:     3,
			maxAttempts: 3,
			expected:    false,
		},
		{
			name:        "retryable error",
			err:         &mockRetryableError{retryable: true},
			attempt:     1,
			maxAttempts: 3,
			expected:    true,
		},
		{
			name:        "non-retryable error",
			err:         &mockRetryableError{retryable: false},
			attempt:     1,
			maxAttempts: 3,
			expected:    false,
		},
		{
			name:        "grove error network timeout",
			err:         groveErrors.NewGroveError(groveErrors.ErrCodeNetworkTimeout, "timeout", nil),
			attempt:     1,
			maxAttempts: 3,
			expected:    true,
		},
		{
			name:        "grove error auth failed",
			err:         groveErrors.NewGroveError(groveErrors.ErrCodeAuthenticationFailed, "auth failed", nil),
			attempt:     1,
			maxAttempts: 3,
			expected:    false,
		},
		{
			name:        "unknown error",
			err:         errors.New("unknown error"),
			attempt:     1,
			maxAttempts: 3,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.err, tt.attempt, tt.maxAttempts)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExecuteWithRetrySuccess(t *testing.T) {
	ctx := context.Background()
	config := RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     1 * time.Millisecond,
		MaxDelay:      10 * time.Millisecond,
		JitterEnabled: false,
	}

	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 1 {
			return &mockRetryableError{retryable: true}
		}
		return nil
	}

	err := ExecuteWithRetry(ctx, config, operation)

	assert.NoError(t, err)
	assert.Equal(t, 2, callCount)
}

func TestExecuteWithRetryContextCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	config := RetryConfig{
		MaxAttempts:   5,
		BaseDelay:     10 * time.Millisecond,
		MaxDelay:      100 * time.Millisecond,
		JitterEnabled: false,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return &mockRetryableError{retryable: true}
	}

	err := ExecuteWithRetry(ctx, config, operation)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retry operation cancelled")
	assert.GreaterOrEqual(t, callCount, 1)
	assert.Less(t, callCount, 5)
}
