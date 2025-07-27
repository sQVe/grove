package commands

import "github.com/sqve/grove/internal/git"

// This provides better dependency injection and testability.
type ExecutorProvider struct {
	executor git.GitExecutor
}

func NewExecutorProvider() *ExecutorProvider {
	return &ExecutorProvider{
		executor: git.DefaultExecutor,
	}
}

// This is primarily used for testing.
func NewExecutorProviderWithExecutor(executor git.GitExecutor) *ExecutorProvider {
	return &ExecutorProvider{
		executor: executor,
	}
}

func (ep *ExecutorProvider) GetExecutor() git.GitExecutor {
	return ep.executor
}

func (ep *ExecutorProvider) CreateListService() *ListService {
	return NewListService(ep.executor)
}

// Global executor provider instance for commands.
// This can be replaced for testing or different executor configurations.
var DefaultExecutorProvider = NewExecutorProvider()

// This is primarily used for testing.
func SetExecutorProvider(provider *ExecutorProvider) {
	DefaultExecutorProvider = provider
}

// This is primarily used for testing cleanup.
func ResetExecutorProvider() {
	DefaultExecutorProvider = NewExecutorProvider()
}
