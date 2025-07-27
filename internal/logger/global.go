package logger

import (
	"sync"
)

var (
	globalLogger *Logger
	// mu protects globalLogger.
	mu sync.RWMutex
)

// init initializes the global logger with default configuration.
func init() {
	globalLogger = New(DefaultConfig())
}

func SetGlobalLogger(logger *Logger) {
	mu.Lock()
	defer mu.Unlock()
	globalLogger = logger
}

func GetGlobalLogger() *Logger {
	mu.RLock()
	defer mu.RUnlock()
	return globalLogger
}

func Configure(config Config) {
	SetGlobalLogger(New(config))
}

func Debug(msg string, attrs ...any) {
	GetGlobalLogger().Debug(msg, attrs...)
}

func Info(msg string, attrs ...any) {
	GetGlobalLogger().Info(msg, attrs...)
}

func Warn(msg string, attrs ...any) {
	GetGlobalLogger().Warn(msg, attrs...)
}

func Error(msg string, attrs ...any) {
	GetGlobalLogger().Error(msg, attrs...)
}

func DebugOperation(operation string, attrs ...any) {
	GetGlobalLogger().DebugOperation(operation, attrs...)
}

func InfoOperation(operation string, attrs ...any) {
	GetGlobalLogger().InfoOperation(operation, attrs...)
}

func ErrorOperation(operation string, err error, attrs ...any) {
	GetGlobalLogger().ErrorOperation(operation, err, attrs...)
}

func GitCommand(command string, args []string, attrs ...any) {
	GetGlobalLogger().GitCommand(command, args, attrs...)
}

func GitResult(command string, success bool, output string, attrs ...any) {
	GetGlobalLogger().GitResult(command, success, output, attrs...)
}

func Performance(operation string, duration interface{}, attrs ...any) {
	GetGlobalLogger().Performance(operation, duration, attrs...)
}

func WithOperation(operation string) *Logger {
	return GetGlobalLogger().WithOperation(operation)
}

func WithComponent(component string) *Logger {
	return GetGlobalLogger().WithComponent(component)
}

func WithError(err error) *Logger {
	return GetGlobalLogger().WithError(err)
}
