package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Logger wraps slog.Logger to provide Grove-specific logging functionality.
type Logger struct {
	*slog.Logger
}

// Config holds configuration for the logger.
type Config struct {
	Level  string // debug, info, warn, error
	Format string // text, json
	Output io.Writer
}

// DefaultConfig returns the default logger configuration.
func DefaultConfig() Config {
	return Config{
		Level:  "info",
		Format: "text",
		Output: os.Stderr,
	}
}

// New creates a new logger with the specified configuration.
func New(config Config) *Logger {
	var level slog.Level
	switch strings.ToLower(config.Level) {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: level,
	}

	switch strings.ToLower(config.Format) {
	case "json":
		handler = slog.NewJSONHandler(config.Output, opts)
	default:
		handler = slog.NewTextHandler(config.Output, opts)
	}

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithContext returns a new logger with the given context.
func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.With(slog.Any("context", ctx)),
	}
}

// WithOperation returns a new logger with operation context.
func (l *Logger) WithOperation(operation string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("operation", operation)),
	}
}

// WithComponent returns a new logger with component context.
func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("component", component)),
	}
}

// WithError returns a new logger with error context.
func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.With(slog.Any("error", err)),
	}
}

// DebugOperation logs a debug message with operation timing.
func (l *Logger) DebugOperation(operation string, attrs ...any) {
	l.Debug("operation", append([]any{"op", operation}, attrs...)...)
}

// InfoOperation logs an info message with operation timing.
func (l *Logger) InfoOperation(operation string, attrs ...any) {
	l.Info("operation", append([]any{"op", operation}, attrs...)...)
}

// ErrorOperation logs an error message with operation context.
func (l *Logger) ErrorOperation(operation string, err error, attrs ...any) {
	allAttrs := append([]any{"op", operation, "error", err}, attrs...)
	l.Error("operation failed", allAttrs...)
}

// GitCommand logs a git command execution.
func (l *Logger) GitCommand(command string, args []string, attrs ...any) {
	allAttrs := append([]any{
		"git_command", command,
		"git_args", args,
	}, attrs...)
	l.Debug("git command", allAttrs...)
}

// GitResult logs a git command result.
func (l *Logger) GitResult(command string, success bool, output string, attrs ...any) {
	allAttrs := append([]any{
		"git_command", command,
		"success", success,
		"output", output,
	}, attrs...)

	if success {
		l.Debug("git command completed", allAttrs...)
	} else {
		l.Error("git command failed", allAttrs...)
	}
}

// Performance logs performance metrics.
func (l *Logger) Performance(operation string, duration interface{}, attrs ...any) {
	allAttrs := append([]any{
		"operation", operation,
		"duration", duration,
	}, attrs...)
	l.Info("performance", allAttrs...)
}
