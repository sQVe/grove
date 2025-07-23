package commands

import "github.com/sqve/grove/internal/git"

// ExecutorProvider manages git executor instances for commands.
// This provides better dependency injection and testability.
type ExecutorProvider struct {
	executor git.GitExecutor
}

// NewExecutorProvider creates a new ExecutorProvider with the default git executor.
func NewExecutorProvider() *ExecutorProvider {
	return &ExecutorProvider{
		executor: git.DefaultExecutor,
	}
}

// NewExecutorProviderWithExecutor creates a new ExecutorProvider with a custom executor.
// This is primarily used for testing.
func NewExecutorProviderWithExecutor(executor git.GitExecutor) *ExecutorProvider {
	return &ExecutorProvider{
		executor: executor,
	}
}

// GetExecutor returns the configured git executor.
func (ep *ExecutorProvider) GetExecutor() git.GitExecutor {
	return ep.executor
}

// CreateListService creates a new ListService with the configured executor.
func (ep *ExecutorProvider) CreateListService() *ListService {
	return NewListService(ep.executor)
}

// Global executor provider instance for commands.
// This can be replaced for testing or different executor configurations.
var DefaultExecutorProvider = NewExecutorProvider()

// SetExecutorProvider sets the global executor provider.
// This is primarily used for testing.
func SetExecutorProvider(provider *ExecutorProvider) {
	DefaultExecutorProvider = provider
}

// ResetExecutorProvider resets the global executor provider to default.
// This is primarily used for testing cleanup.
func ResetExecutorProvider() {
	DefaultExecutorProvider = NewExecutorProvider()
}
