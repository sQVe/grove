package logger

import (
	"sync"
)

var (
	// globalLogger is the global logger instance.
	globalLogger *Logger
	// mu protects globalLogger.
	mu sync.RWMutex
)

// init initializes the global logger with default configuration.
func init() {
	globalLogger = New(DefaultConfig())
}

// SetGlobalLogger sets the global logger instance.
func SetGlobalLogger(logger *Logger) {
	mu.Lock()
	defer mu.Unlock()
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance.
func GetGlobalLogger() *Logger {
	mu.RLock()
	defer mu.RUnlock()
	return globalLogger
}

// Configure configures the global logger with the given config.
func Configure(config Config) {
	SetGlobalLogger(New(config))
}

// Debug logs a debug message using the global logger.
func Debug(msg string, attrs ...any) {
	GetGlobalLogger().Debug(msg, attrs...)
}

// Info logs an info message using the global logger.
func Info(msg string, attrs ...any) {
	GetGlobalLogger().Info(msg, attrs...)
}

// Warn logs a warning message using the global logger.
func Warn(msg string, attrs ...any) {
	GetGlobalLogger().Warn(msg, attrs...)
}

// Error logs an error message using the global logger.
func Error(msg string, attrs ...any) {
	GetGlobalLogger().Error(msg, attrs...)
}

// DebugOperation logs a debug operation using the global logger.
func DebugOperation(operation string, attrs ...any) {
	GetGlobalLogger().DebugOperation(operation, attrs...)
}

// InfoOperation logs an info operation using the global logger.
func InfoOperation(operation string, attrs ...any) {
	GetGlobalLogger().InfoOperation(operation, attrs...)
}

// ErrorOperation logs an error operation using the global logger.
func ErrorOperation(operation string, err error, attrs ...any) {
	GetGlobalLogger().ErrorOperation(operation, err, attrs...)
}

// GitCommand logs a git command using the global logger.
func GitCommand(command string, args []string, attrs ...any) {
	GetGlobalLogger().GitCommand(command, args, attrs...)
}

// GitResult logs a git command result using the global logger.
func GitResult(command string, success bool, output string, attrs ...any) {
	GetGlobalLogger().GitResult(command, success, output, attrs...)
}

// Performance logs performance metrics using the global logger.
func Performance(operation string, duration interface{}, attrs ...any) {
	GetGlobalLogger().Performance(operation, duration, attrs...)
}

// WithOperation returns a new logger with operation context.
func WithOperation(operation string) *Logger {
	return GetGlobalLogger().WithOperation(operation)
}

// WithComponent returns a new logger with component context.
func WithComponent(component string) *Logger {
	return GetGlobalLogger().WithComponent(component)
}

// WithError returns a new logger with error context.
func WithError(err error) *Logger {
	return GetGlobalLogger().WithError(err)
}
