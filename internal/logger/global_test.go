//go:build !integration
// +build !integration

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
	logger := GetGlobalLogger()
	assert.NotNil(t, logger)
	assert.NotNil(t, logger.Logger)
}

func TestSetGlobalLogger(t *testing.T) {
	original := GetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}
	newLogger := New(config)

	SetGlobalLogger(newLogger)

	current := GetGlobalLogger()
	assert.Equal(t, newLogger, current)

	Info("test message")
	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)

	SetGlobalLogger(original)
}

func TestConfigure(t *testing.T) {
	original := GetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}
	Configure(config)

	Info("test message")
	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)

	SetGlobalLogger(original)
}

func TestGlobalLoggerThreadSafety(t *testing.T) {
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

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
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

	SetGlobalLogger(original)
}

func TestGlobalLoggerConcurrentLogging(t *testing.T) {
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

	output := buf.String()
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")

	SetGlobalLogger(original)
}

func TestGlobalFunctionsSmoke(t *testing.T) {
	original := GetGlobalLogger()

	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}
	Configure(config)

	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")

	DebugOperation("test-op", "key", "value")
	InfoOperation("test-op", "key", "value")
	testErr := errors.New("test error")
	ErrorOperation("test-op", testErr, "key", "value")

	opLogger := WithOperation("test-operation")
	assert.NotNil(t, opLogger)
	compLogger := WithComponent("test-component")
	assert.NotNil(t, compLogger)
	errLogger := WithError(testErr)
	assert.NotNil(t, errLogger)

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "debug message")
	assert.Contains(t, output, "info message")

	SetGlobalLogger(original)
}

func BenchmarkGlobalLoggerConcurrency(b *testing.B) {
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

	SetGlobalLogger(original)
}

func TestGlobalLoggerRaceCondition(t *testing.T) {
	original := GetGlobalLogger()

	// This test should be run with -race flag.
	const numGoroutines = 10
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < iterations; j++ {
				// Mix of operations that could race.
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

	SetGlobalLogger(original)
}
