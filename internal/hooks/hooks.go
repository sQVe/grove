package hooks

import (
	"bytes"
	"errors"
	"os/exec"

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

// RunCreateHooks runs commands sequentially, stops on first failure.
func RunCreateHooks(workDir string, commands []string) *RunResult {
	result := &RunResult{}

	if len(commands) == 0 {
		return result
	}

	logger.Debug("Running %d create hooks in %s", len(commands), workDir)

	for _, cmdStr := range commands {
		logger.Debug("Executing hook: %s", cmdStr)

		cmd := exec.Command("sh", "-c", cmdStr) //nolint:gosec // User-configured hooks are intentionally executed
		cmd.Dir = workDir

		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr

		err := cmd.Run()
		if err != nil {
			exitCode := 1
			exitErr := &exec.ExitError{}
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			}

			result.Failed = &HookResult{
				Command:  cmdStr,
				ExitCode: exitCode,
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
			}

			logger.Debug("Hook failed with exit code %d: %s", exitCode, cmdStr)
			return result
		}

		result.Succeeded = append(result.Succeeded, cmdStr)
		logger.Debug("Hook succeeded: %s", cmdStr)
	}

	return result
}

func GetCreateHooks(worktreeDir string) []string {
	cfg, err := config.LoadFromFile(worktreeDir)
	if err != nil {
		logger.Debug("Failed to load config for hooks: %v", err)
		return nil
	}

	return cfg.Hooks.Create
}
