// Package retry provides exponential backoff retry mechanisms for network operations.
package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/logger"
)

var log = logger.WithComponent("retry")

// RetryConfig defines the configuration for retry operations.
type RetryConfig struct {
	MaxAttempts   int           // Maximum number of retry attempts (including initial attempt)
	BaseDelay     time.Duration // Base delay for exponential backoff
	MaxDelay      time.Duration // Maximum delay between retries
	JitterEnabled bool          // Whether to add jitter to prevent thundering herd
}

// DefaultConfig returns a sensible default retry configuration.
func DefaultConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:   3,
		BaseDelay:     1 * time.Second,
		MaxDelay:      10 * time.Second,
		JitterEnabled: true,
	}
}

// GetConfig returns the retry configuration from Viper settings.
// Falls back to default values if configuration is not properly initialized.
func GetConfig() RetryConfig {
	maxAttempts := config.GetInt("retry.max_attempts")
	baseDelay := config.GetDuration("retry.base_delay")
	maxDelay := config.GetDuration("retry.max_delay")
	jitterEnabled := config.GetBool("retry.jitter_enabled")

	// If configuration is not initialized (all values are zero), use defaults
	if maxAttempts == 0 && baseDelay == 0 && maxDelay == 0 {
		return DefaultConfig()
	}

	return RetryConfig{
		MaxAttempts:   maxAttempts,
		BaseDelay:     baseDelay,
		MaxDelay:      maxDelay,
		JitterEnabled: jitterEnabled,
	}
}

// RetryableError defines the interface for errors that can be retried.
type RetryableError interface {
	IsRetryable() bool
}

// ExecuteWithRetry executes the given function with exponential backoff retry.
// It only retries if the error implements RetryableError and returns true for IsRetryable().
func ExecuteWithRetry(ctx context.Context, retryConfig RetryConfig, operation func() error) error {
	return ExecuteWithRetryContext(ctx, retryConfig, func(ctx context.Context) error {
		return operation()
	})
}

// ExecuteWithRetryContext executes the given context-aware function with exponential backoff retry.
// The operation function receives the context and should respect cancellation.
// It only retries if the error implements RetryableError and returns true for IsRetryable().
func ExecuteWithRetryContext(ctx context.Context, retryConfig RetryConfig, operation func(context.Context) error) error {
	var lastErr error

	for attempt := 1; attempt <= retryConfig.MaxAttempts; attempt++ {
		// Check context cancellation before each attempt
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry operation cancelled before attempt %d: %w", attempt, ctx.Err())
		default:
		}

		// Execute the operation with context
		err := operation(ctx)
		if err == nil {
			// Success - log if this was a retry
			if attempt > 1 {
				log.Debug("Operation succeeded after retry", "attempt", attempt)
			}
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !shouldRetry(err, attempt, retryConfig.MaxAttempts) {
			break
		}

		// Calculate delay for next attempt
		delay := calculateDelay(attempt, retryConfig)

		// Log retry attempt
		log.Debug("Operation failed, retrying",
			"attempt", attempt,
			"max_attempts", retryConfig.MaxAttempts,
			"error", err,
			"delay", delay)

		// Wait before next attempt, respecting context cancellation
		select {
		case <-ctx.Done():
			return fmt.Errorf("retry operation cancelled: %w", ctx.Err())
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All attempts failed
	return fmt.Errorf("operation failed after %d attempts: %w", retryConfig.MaxAttempts, lastErr)
}

// shouldRetry determines if an error should be retried based on the error type and attempt count.
func shouldRetry(err error, attempt, maxAttempts int) bool {
	// Don't retry if we've reached max attempts
	if attempt >= maxAttempts {
		return false
	}

	// Check if error is retryable
	var retryableErr RetryableError
	if errors.As(err, &retryableErr) {
		return retryableErr.IsRetryable()
	}

	// Check if it's a GroveError with retryable error code
	var groveErr *errors.GroveError
	if errors.As(err, &groveErr) {
		return isRetryableErrorCode(groveErr.Code)
	}

	// Default to not retrying unknown errors
	return false
}

// isRetryableErrorCode checks if a Grove error code represents a retryable condition.
func isRetryableErrorCode(code string) bool {
	switch code {
	case errors.ErrCodeNetworkTimeout,
		errors.ErrCodeNetworkUnavailable,
		errors.ErrCodeGitOperation: // Some git operations might be retryable
		return true
	case errors.ErrCodeGitClone,
		errors.ErrCodeInvalidURL,
		errors.ErrCodeAuthenticationFailed:
		return false
	default:
		return false
	}
}

// calculateDelay calculates the delay for the next retry attempt using exponential backoff.
func calculateDelay(attempt int, retryConfig RetryConfig) time.Duration {
	// Calculate exponential backoff: baseDelay * 2^(attempt-1)
	exponentialDelay := float64(retryConfig.BaseDelay) * math.Pow(2, float64(attempt-1))

	// Cap at maximum delay
	if exponentialDelay > float64(retryConfig.MaxDelay) {
		exponentialDelay = float64(retryConfig.MaxDelay)
	}

	delay := time.Duration(exponentialDelay)

	// Add jitter if enabled (±25% random variation)
	if retryConfig.JitterEnabled {
		jitter := float64(delay) * 0.25 * (rand.Float64()*2 - 1) // -25% to +25%
		delay = time.Duration(float64(delay) + jitter)

		// Ensure delay is not negative
		if delay < 0 {
			delay = retryConfig.BaseDelay
		}
	}

	return delay
}

// WithRetry is a convenience function that uses the default retry configuration.
func WithRetry(ctx context.Context, operation func() error) error {
	return ExecuteWithRetry(ctx, DefaultConfig(), operation)
}

// WithConfiguredRetry is a convenience function that uses the configured retry settings.
func WithConfiguredRetry(ctx context.Context, operation func() error) error {
	return ExecuteWithRetry(ctx, GetConfig(), operation)
}
