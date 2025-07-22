//go:build !integration
// +build !integration

package logger

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testContextKey string

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	assert.Equal(t, "info", config.Level)
	assert.Equal(t, "text", config.Format)
	assert.Equal(t, os.Stderr, config.Output)
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		expected slog.Level
	}{
		{
			name: "debug level",
			config: Config{
				Level:  "debug",
				Format: "text",
				Output: os.Stderr,
			},
			expected: slog.LevelDebug,
		},
		{
			name: "info level",
			config: Config{
				Level:  "info",
				Format: "text",
				Output: os.Stderr,
			},
			expected: slog.LevelInfo,
		},
		{
			name: "warn level",
			config: Config{
				Level:  "warn",
				Format: "text",
				Output: os.Stderr,
			},
			expected: slog.LevelWarn,
		},
		{
			name: "error level",
			config: Config{
				Level:  "error",
				Format: "text",
				Output: os.Stderr,
			},
			expected: slog.LevelError,
		},
		{
			name: "invalid level defaults to info",
			config: Config{
				Level:  "invalid",
				Format: "text",
				Output: os.Stderr,
			},
			expected: slog.LevelInfo,
		},
		{
			name: "case insensitive level",
			config: Config{
				Level:  "DEBUG",
				Format: "text",
				Output: os.Stderr,
			},
			expected: slog.LevelDebug,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := New(tt.config)
			assert.NotNil(t, logger)
			assert.NotNil(t, logger.Logger)

			// Test that the logger respects the level by trying to log at different levels
			var buf bytes.Buffer
			testConfig := tt.config
			testConfig.Output = &buf
			testLogger := New(testConfig)

			// Log at debug level
			testLogger.Debug("debug message")
			output := buf.String()

			if tt.expected == slog.LevelDebug {
				assert.Contains(t, output, "debug message")
			} else {
				assert.NotContains(t, output, "debug message")
			}
		})
	}
}

func TestNewWithFormats(t *testing.T) {
	tests := []struct {
		name           string
		format         string
		expectedInJSON bool
	}{
		{
			name:           "json format",
			format:         "json",
			expectedInJSON: true,
		},
		{
			name:           "text format",
			format:         "text",
			expectedInJSON: false,
		},
		{
			name:           "invalid format defaults to text",
			format:         "invalid",
			expectedInJSON: false,
		},
		{
			name:           "case insensitive format",
			format:         "JSON",
			expectedInJSON: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:  "info",
				Format: tt.format,
				Output: &buf,
			}

			logger := New(config)
			logger.Info("test message", "key", "value")

			output := buf.String()
			if tt.expectedInJSON {
				assert.Contains(t, output, `"msg":"test message"`)
				assert.Contains(t, output, `"key":"value"`)
			} else {
				assert.Contains(t, output, "test message")
				assert.Contains(t, output, "key=value")
			}
		})
	}
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	ctx := context.WithValue(context.Background(), testContextKey("test-key"), "test-value")
	contextLogger := logger.WithContext(ctx)

	contextLogger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)
	assert.Contains(t, output, `"context"`)
}

func TestWithOperation(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	opLogger := logger.WithOperation("test-operation")

	opLogger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)
	assert.Contains(t, output, `"operation":"test-operation"`)
}

func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	compLogger := logger.WithComponent("test-component")

	compLogger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)
	assert.Contains(t, output, `"component":"test-component"`)
}

func TestWithError(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	testErr := errors.New("test error")
	errLogger := logger.WithError(testErr)

	errLogger.Info("test message")

	output := buf.String()
	assert.Contains(t, output, `"msg":"test message"`)
	assert.Contains(t, output, `"error":"test error"`)
}

func TestDebugOperation(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	logger.DebugOperation("test-op", "key1", "value1", "key2", "value2")

	output := buf.String()
	assert.Contains(t, output, `"msg":"operation"`)
	assert.Contains(t, output, `"op":"test-op"`)
	assert.Contains(t, output, `"key1":"value1"`)
	assert.Contains(t, output, `"key2":"value2"`)
}

func TestInfoOperation(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	logger.InfoOperation("test-op", "key1", "value1")

	output := buf.String()
	assert.Contains(t, output, `"msg":"operation"`)
	assert.Contains(t, output, `"op":"test-op"`)
	assert.Contains(t, output, `"key1":"value1"`)
}

func TestErrorOperation(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "error",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	testErr := errors.New("operation failed")
	logger.ErrorOperation("test-op", testErr, "key1", "value1")

	output := buf.String()
	assert.Contains(t, output, `"msg":"operation failed"`)
	assert.Contains(t, output, `"op":"test-op"`)
	assert.Contains(t, output, `"error":"operation failed"`)
	assert.Contains(t, output, `"key1":"value1"`)
}

func TestGitCommand(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	args := []string{"clone", "--bare", "repo.git"}
	logger.GitCommand("git", args, "timeout", "5s")

	output := buf.String()
	assert.Contains(t, output, `"msg":"git command"`)
	assert.Contains(t, output, `"git_command":"git"`)
	assert.Contains(t, output, `"git_args":["clone","--bare","repo.git"]`)
	assert.Contains(t, output, `"timeout":"5s"`)
}

func TestGitResult(t *testing.T) {
	tests := []struct {
		name            string
		success         bool
		output          string
		expectedMessage string
		expectedLevel   string
	}{
		{
			name:            "successful git command",
			success:         true,
			output:          "command output",
			expectedMessage: "git command completed",
			expectedLevel:   "DEBUG",
		},
		{
			name:            "failed git command",
			success:         false,
			output:          "command failed",
			expectedMessage: "git command failed",
			expectedLevel:   "ERROR",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			config := Config{
				Level:  "debug",
				Format: "json",
				Output: &buf,
			}

			logger := New(config)
			logger.GitResult("git", tt.success, tt.output, "duration", "100ms")

			output := buf.String()
			assert.Contains(t, output, `"msg":"`+tt.expectedMessage+`"`)
			assert.Contains(t, output, `"git_command":"git"`)
			if tt.success {
				assert.Contains(t, output, `"success":true`)
			} else {
				assert.Contains(t, output, `"success":false`)
			}
			assert.Contains(t, output, `"output":"`+tt.output+`"`)
			assert.Contains(t, output, `"duration":"100ms"`)
			assert.Contains(t, output, `"level":"`+tt.expectedLevel+`"`)
		})
	}
}

func TestPerformance(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "info",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	duration := 150 * time.Millisecond
	logger.Performance("test-operation", duration, "component", "test-comp")

	output := buf.String()
	assert.Contains(t, output, `"msg":"performance"`)
	assert.Contains(t, output, `"operation":"test-operation"`)
	assert.Contains(t, output, `"duration":150000000`)
	assert.Contains(t, output, `"component":"test-comp"`)
}

func TestChainedContextMethods(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "debug",
		Format: "json",
		Output: &buf,
	}

	logger := New(config)
	testErr := errors.New("chained error")

	chainedLogger := logger.
		WithComponent("test-comp").
		WithOperation("test-op").
		WithError(testErr)

	chainedLogger.Info("chained message")

	output := buf.String()
	assert.Contains(t, output, `"msg":"chained message"`)
	assert.Contains(t, output, `"component":"test-comp"`)
	assert.Contains(t, output, `"operation":"test-op"`)
	assert.Contains(t, output, `"error":"chained error"`)
}

func TestLoggerLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	config := Config{
		Level:  "warn",
		Format: "text",
		Output: &buf,
	}

	logger := New(config)

	// These should not appear in output
	logger.Debug("debug message")
	logger.Info("info message")

	// These should appear in output
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()
	assert.NotContains(t, output, "debug message")
	assert.NotContains(t, output, "info message")
	assert.Contains(t, output, "warn message")
	assert.Contains(t, output, "error message")
}

func TestLoggerWithNilOutput(t *testing.T) {
	// Test that logger panics with nil output (expected behavior)
	config := Config{
		Level:  "info",
		Format: "text",
		Output: nil,
	}

	// This should panic due to nil output
	require.Panics(t, func() {
		logger := New(config)
		logger.Info("test message")
	})
}
