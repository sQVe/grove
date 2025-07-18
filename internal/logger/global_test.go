package logger

import (
	"bytes"
	"errors"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalLoggerInitialization(t *testing.T) {
	// Test that global logger is initialized with default config
	logger := GetGlobalLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
}

func TestSetGlobalLogger(t *testing.T) {
	// Save original logger
	original := GetGlobalLogger()

	// Create a new logger
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}
	newLogger := New(config)

	// Set new global logger
	SetGlobalLogger(newLogger)

	// Test that the global logger was changed
	current := GetGlobalLogger()
	assert.Equal(t, newLogger, current)

	// Test that global functions use new logger
	Info("test message")
	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)

	// Restore original logger
	SetGlobalLogger(original)
}

func TestConfigure(t *testing.T) {
	// Save original logger
	original := GetGlobalLogger()

	// Configure with new settings
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}
	Configure(config)

	// Test that configuration was applied
	Info("test message")
	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)

	// Restore original logger
	SetGlobalLogger(original)
}

func TestGlobalLoggerThreadSafety(t *testing.T) {
	// Save original logger
	original := GetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "text",
		Output: &buf,
	}

	const numGoroutines = 100
	const operationsPerGoroutine = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent set/get operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				// Alternate between setting and getting
				if j%2 == 0 {
					testLogger := New(config)
					SetGlobalLogger(testLogger)
				} else {
					logger := GetGlobalLogger()
					assert.NotNil(t, logger)
				}
			}
		}(i)
	}

	wg.Wait()

	// Restore original logger
	SetGlobalLogger(original)
}

func TestGlobalLoggerConcurrentLogging(t *testing.T) {
	// Save original logger
	original := GetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "text",
		Output: &buf,
	}
	Configure(config)

	const numGoroutines = 50
	const logsPerGoroutine = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent logging operations
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < logsPerGoroutine; j++ {
				Debug("debug message", "goroutine", id, "iteration", j)
				Info("info message", "goroutine", id, "iteration", j)
				Warn("warn message", "goroutine", id, "iteration", j)
				Error("error message", "goroutine", id, "iteration", j)
			}
		}(i)
	}

	wg.Wait()

	// Verify that all messages were logged
	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")

	// Restore original logger
	SetGlobalLogger(original)
}

// Test a few key global functions work (smoke tests).
func TestGlobalFunctionsSmoke(t *testing.T) {
	// Save original logger
	original := GetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}
	Configure(config)

	// Test basic logging functions
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	// Test operation functions
	DebugOperation("test-op", "key", "value")
	InfoOperation("test-op", "key", "value")
	testErr := errors.New("test error")
	ErrorOperation("test-op", testErr, "key", "value")

	// Test context functions return loggers
	opLogger := WithOperation("test-operation")
	assert.NotNil(t, opLogger)
	compLogger := WithComponent("test-component")
	assert.NotNil(t, compLogger)
	errLogger := WithError(testErr)
	assert.NotNil(t, errLogger)

	// Just verify we got some output
	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")

	// Restore original logger
	SetGlobalLogger(original)
}

func BenchmarkGlobalLoggerConcurrency(b *testing.B) {
	// Save original logger
	original := GetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Output: &buf,
	}
	Configure(config)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			Info("benchmark message", "key", "value")
		}
	})

	// Restore original logger
	SetGlobalLogger(original)
}

func TestGlobalLoggerRaceCondition(t *testing.T) {
	// Save original logger
	original := GetGlobalLogger()

	// This test should be run with -race flag
	const numGoroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				// Mix of operations that could race
				if j%3 == 0 {
					var buf bytes.Buffer
					config := Config{
						Level:  "debug",
						Format: "json",
						Output: &buf,
					}
					Configure(config)
				}

				Info("race test", "goroutine", id, "iteration", j)

				logger := GetGlobalLogger()
				require.NotNil(t, logger)
			}
		}(i)
	}

	wg.Wait()

	// Restore original logger
	SetGlobalLogger(original)
}
