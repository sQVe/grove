package hooks

import (
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
)

type HookResult struct {
	Command  string
	ExitCode int
	Stdout   string
	Stderr   string
}

type RunResult struct {
	Succeeded []string
	Failed    *HookResult
}

func GetAddHooks(worktreeDir string) []string {
	cfg, err := config.LoadFromFile(worktreeDir)
	if err != nil {
		// LoadFromFile returns nil error when file doesn't exist,
		// so any error means the file exists but is invalid TOML
		logger.Warning("Config file has errors, hooks disabled: %v", err)
		return nil
	}

	return cfg.Hooks.Add
}
