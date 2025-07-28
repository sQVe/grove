package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type Logger struct {
	*slog.Logger
}

type Config struct {
	Level  string // debug, info, warn, error.
	Format string // text, json.
	Output io.Writer
}

func DefaultConfig() Config {
	return Config{
		Level:  "info",
		Format: "text",
		Output: os.Stderr,
	}
}

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

func (l *Logger) WithContext(ctx context.Context) *Logger {
	return &Logger{
		Logger: l.With(slog.Any("context", ctx)),
	}
}

func (l *Logger) WithOperation(operation string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("operation", operation)),
	}
}

func (l *Logger) WithComponent(component string) *Logger {
	return &Logger{
		Logger: l.With(slog.String("component", component)),
	}
}

func (l *Logger) WithError(err error) *Logger {
	return &Logger{
		Logger: l.With(slog.Any("error", err)),
	}
}

func (l *Logger) DebugOperation(operation string, attrs ...any) {
	l.Debug("operation", append([]any{"op", operation}, attrs...)...)
}

func (l *Logger) InfoOperation(operation string, attrs ...any) {
	l.Info("operation", append([]any{"op", operation}, attrs...)...)
}

func (l *Logger) ErrorOperation(operation string, err error, attrs ...any) {
	allAttrs := append([]any{"op", operation, "error", err}, attrs...)
	// Log operation failures at debug level to avoid duplicate error messages
	// The actual error should be returned to the user by the calling code.
	l.Debug("operation failed", allAttrs...)
}

func (l *Logger) GitCommand(command string, args []string, attrs ...any) {
	allAttrs := append([]any{
		"git_command", command,
		"git_args", args,
	}, attrs...)
	l.Debug("git command", allAttrs...)
}

func (l *Logger) GitResult(command string, success bool, output string, attrs ...any) {
	allAttrs := append([]any{
		"git_command", command,
		"success", success,
		"output", output,
	}, attrs...)

	if success {
		l.Debug("git command completed", allAttrs...)
	} else {
		// Log git failures at debug level to avoid cluttering user output
		// Actual user-facing errors should come from the calling code.
		l.Debug("git command failed", allAttrs...)
	}
}

func (l *Logger) Performance(operation string, duration interface{}, attrs ...any) {
	allAttrs := append([]any{
		"operation", operation,
		"duration", duration,
	}, attrs...)
	l.Info("performance", allAttrs...)
}
