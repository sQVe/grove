package hooks

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
)

type prefixWriter struct {
	prefix string
	target io.Writer
	buf    bytes.Buffer
	mu     *sync.Mutex
}

func newPrefixWriter(prefix string, target io.Writer, mu *sync.Mutex) *prefixWriter {
	return &prefixWriter{prefix: prefix, target: target, mu: mu}
}

func (w *prefixWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	n, err = w.buf.Write(p)
	if err != nil {
		return n, err
	}

	for {
		line, readErr := w.buf.ReadString('\n')
		if readErr != nil {
			if line != "" {
				w.buf.WriteString(line)
			}
			break
		}

		_, writeErr := fmt.Fprintf(w.target, "%s %s", w.prefix, line)
		if writeErr != nil {
			return n, writeErr
		}
	}

	return n, nil
}

func (w *prefixWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	remaining := w.buf.String()
	if remaining != "" {
		_, err := fmt.Fprintf(w.target, "%s %s\n", w.prefix, remaining)
		w.buf.Reset()
		return err
	}
	return nil
}

func RunAddHooksStreaming(workDir string, commands []string, output io.Writer) *RunResult {
	result := &RunResult{}
	if len(commands) == 0 {
		return result
	}

	logger.Debug("Running %d add hooks in %s (streaming)", len(commands), workDir)

	for _, cmdStr := range commands {
		logger.Debug("Executing hook: %s", cmdStr)

		cmd := exec.Command("sh", "-c", cmdStr) //nolint:gosec // User-configured hooks are intentionally executed
		cmd.Dir = workDir

		var mu sync.Mutex
		prefix := styles.Render(&styles.Dimmed, fmt.Sprintf("  [%s]", cmdStr))
		stdout := newPrefixWriter(prefix, output, &mu)
		stderr := newPrefixWriter(prefix, output, &mu)

		cmd.Stdout = stdout
		cmd.Stderr = stderr

		err := cmd.Start()
		if err != nil {
			result.Failed = &HookResult{
				Command:  cmdStr,
				ExitCode: 1,
				Stdout:   "",
				Stderr:   err.Error(),
			}
			return result
		}

		err = cmd.Wait()

		if flushErr := stdout.Flush(); flushErr != nil {
			logger.Debug("Failed to flush stdout: %v", flushErr)
		}
		if flushErr := stderr.Flush(); flushErr != nil {
			logger.Debug("Failed to flush stderr: %v", flushErr)
		}

		if err != nil {
			exitCode := 1
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				exitCode = exitErr.ExitCode()
			}
			result.Failed = &HookResult{
				Command:  cmdStr,
				ExitCode: exitCode,
				Stdout:   "",
				Stderr:   "",
			}
			logger.Debug("Hook failed with exit code %d: %s", exitCode, cmdStr)
			return result
		}

		result.Succeeded = append(result.Succeeded, cmdStr)
		logger.Debug("Hook succeeded: %s", cmdStr)
	}
	return result
}
