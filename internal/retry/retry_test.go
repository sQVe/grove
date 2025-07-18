package retry

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	groveErrors "github.com/sqve/grove/internal/errors"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxAttempts != 3 {
		t.Errorf("expected MaxAttempts to be 3, got %d", config.MaxAttempts)
	}
	if config.BaseDelay != 1*time.Second {
		t.Errorf("expected BaseDelay to be 1s, got %v", config.BaseDelay)
	}
	if config.MaxDelay != 10*time.Second {
		t.Errorf("expected MaxDelay to be 10s, got %v", config.MaxDelay)
	}
	if !config.JitterEnabled {
		t.Error("expected JitterEnabled to be true")
	}
}

func TestExecuteWithRetry_Success(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		JitterEnabled: false,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return nil
	}

	err := ExecuteWithRetry(context.Background(), config, operation)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 1 {
		t.Errorf("expected operation to be called once, got %d", callCount)
	}
}

func TestExecuteWithRetry_SuccessAfterRetry(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		JitterEnabled: false,
	}

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return groveErrors.ErrNetworkTimeout("test operation", errors.New("network timeout"))
		}
		return nil
	}

	start := time.Now()
	err := ExecuteWithRetry(context.Background(), config, operation)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected operation to be called twice, got %d", callCount)
	}
	// Should have waited at least the base delay
	if duration < config.BaseDelay {
		t.Errorf("expected duration to be at least %v, got %v", config.BaseDelay, duration)
	}
}

func TestExecuteWithRetry_NonRetryableError(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		JitterEnabled: false,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return groveErrors.ErrAuthenticationFailed("test operation", errors.New("auth failed"))
	}

	err := ExecuteWithRetry(context.Background(), config, operation)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected operation to be called once, got %d", callCount)
	}
}

func TestExecuteWithRetry_MaxAttemptsExceeded(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   2,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      1 * time.Second,
		JitterEnabled: false,
	}

	callCount := 0
	operation := func() error {
		callCount++
		return groveErrors.ErrNetworkTimeout("test operation", errors.New("network timeout"))
	}

	err := ExecuteWithRetry(context.Background(), config, operation)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 2 {
		t.Errorf("expected operation to be called twice, got %d", callCount)
	}
	if !errors.Is(err, groveErrors.ErrNetworkTimeout("", nil)) {
		t.Errorf("expected error to wrap network timeout, got %v", err)
	}
}

func TestExecuteWithRetry_ContextCancellation(t *testing.T) {
	config := RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     1 * time.Second,
		MaxDelay:      10 * time.Second,
		JitterEnabled: false,
	}

	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 1 {
			// Cancel context during first retry delay
			go func() {
				time.Sleep(100 * time.Millisecond)
				cancel()
			}()
		}
		return groveErrors.ErrNetworkTimeout("test operation", errors.New("network timeout"))
	}

	err := ExecuteWithRetry(ctx, config, operation)

	if err == nil {
		t.Error("expected error, got nil")
	}
	if callCount != 1 {
		t.Errorf("expected operation to be called once, got %d", callCount)
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context cancelled error, got %v", err)
	}
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
			name:        "retryable error within attempts",
			err:         groveErrors.ErrNetworkTimeout("test", errors.New("timeout")),
			attempt:     1,
			maxAttempts: 3,
			expected:    true,
		},
		{
			name:        "non-retryable error",
			err:         groveErrors.ErrAuthenticationFailed("test", errors.New("auth failed")),
			attempt:     1,
			maxAttempts: 3,
			expected:    false,
		},
		{
			name:        "max attempts reached",
			err:         groveErrors.ErrNetworkTimeout("test", errors.New("timeout")),
			attempt:     3,
			maxAttempts: 3,
			expected:    false,
		},
		{
			name:        "unknown error type",
			err:         errors.New("unknown error"),
			attempt:     1,
			maxAttempts: 3,
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldRetry(tt.err, tt.attempt, tt.maxAttempts)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestIsRetryableErrorCode(t *testing.T) {
	tests := []struct {
		code     string
		expected bool
	}{
		{groveErrors.ErrCodeNetworkTimeout, true},
		{groveErrors.ErrCodeNetworkUnavailable, true},
		{groveErrors.ErrCodeGitOperation, true},
		{groveErrors.ErrCodeAuthenticationFailed, false},
		{groveErrors.ErrCodeInvalidURL, false},
		{groveErrors.ErrCodeGitClone, false},
		{"UNKNOWN_ERROR", false},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			result := isRetryableErrorCode(tt.code)
			if result != tt.expected {
				t.Errorf("expected %v for code %s, got %v", tt.expected, tt.code, result)
			}
		})
	}
}

func TestCalculateDelay(t *testing.T) {
	config := RetryConfig{
		BaseDelay:     1 * time.Second,
		MaxDelay:      10 * time.Second,
		JitterEnabled: false,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{1, 1 * time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 10 * time.Second}, // Capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("attempt_%d", tt.attempt), func(t *testing.T) {
			delay := calculateDelay(tt.attempt, config)
			if delay != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, delay)
			}
		})
	}
}

func TestCalculateDelay_WithJitter(t *testing.T) {
	config := RetryConfig{
		BaseDelay:     1 * time.Second,
		MaxDelay:      10 * time.Second,
		JitterEnabled: true,
	}

	// Test that jitter produces different values
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = calculateDelay(2, config)
	}

	// Check that we get some variation (not all delays are the same)
	allSame := true
	for i := 1; i < len(delays); i++ {
		if delays[i] != delays[0] {
			allSame = false
			break
		}
	}

	if allSame {
		t.Error("expected jitter to produce different delays, but all were the same")
	}

	// Check that all delays are within reasonable bounds (±25% of base delay)
	baseDelay := 2 * time.Second // Expected for attempt 2
	minDelay := time.Duration(float64(baseDelay) * 0.75)
	maxDelay := time.Duration(float64(baseDelay) * 1.25)

	for i, delay := range delays {
		if delay < minDelay || delay > maxDelay {
			t.Errorf("delay %d (%v) is outside expected range [%v, %v]", i, delay, minDelay, maxDelay)
		}
	}
}

func TestWithRetry_ConvenienceFunction(t *testing.T) {
	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return groveErrors.ErrNetworkTimeout("test operation", errors.New("network timeout"))
		}
		return nil
	}

	err := WithRetry(context.Background(), operation)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected operation to be called twice, got %d", callCount)
	}
}

// TestRetryableErrorInterface tests that our GroveError properly implements RetryableError.
func TestRetryableErrorInterface(t *testing.T) {
	retryableErr := groveErrors.ErrNetworkTimeout("test", errors.New("timeout"))
	nonRetryableErr := groveErrors.ErrAuthenticationFailed("test", errors.New("auth failed"))

	var retryable RetryableError

	// Test that retryable error implements the interface
	if !errors.As(retryableErr, &retryable) {
		t.Error("expected retryable error to implement RetryableError interface")
	} else if !retryable.IsRetryable() {
		t.Error("expected retryable error to return true for IsRetryable()")
	}

	// Test that non-retryable error implements the interface but returns false
	if !errors.As(nonRetryableErr, &retryable) {
		t.Error("expected non-retryable error to implement RetryableError interface")
	} else if retryable.IsRetryable() {
		t.Error("expected non-retryable error to return false for IsRetryable()")
	}
}
