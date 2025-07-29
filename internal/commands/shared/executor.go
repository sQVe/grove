package shared

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

func NewExecutorProviderWithExecutor(executor git.GitExecutor) *ExecutorProvider {
	return &ExecutorProvider{
		executor: executor,
	}
}

func (ep *ExecutorProvider) GetExecutor() git.GitExecutor {
	return ep.executor
}

// Global executor provider instance for commands.
// This can be replaced for testing or different executor configurations.
var DefaultExecutorProvider = NewExecutorProvider()

func SetExecutorProvider(provider *ExecutorProvider) {
	DefaultExecutorProvider = provider
}

func ResetExecutorProvider() {
	DefaultExecutorProvider = NewExecutorProvider()
}
