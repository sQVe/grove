package commands

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/formatter"
	"github.com/sqve/grove/internal/logger"
)

type execTarget struct {
	label string
	name  string
	path  string
}

type execResult struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	ExitCode int    `json:"exit_code"`

	label  string
	output *bytes.Buffer
	err    error
	stop   bool
}

// ExecError reports an exec command failure with its process exit code.
type ExecError struct {
	message  string
	exitCode int
}

// Error returns the exec failure message.
func (e *ExecError) Error() string {
	return e.message
}

// ExitCode returns the process exit code grove should exit with.
func (e *ExecError) ExitCode() int {
	return e.exitCode
}

// NewExecCmd creates the exec command
func NewExecCmd() *cobra.Command {
	var all bool
	var failFast bool
	var jsonOutput bool
	var parallel int

	cmd := &cobra.Command{
		Use:   "exec [--all | <worktree>...] -- <command>",
		Short: "Execute a command in worktrees",
		Long: `Run a command in one or more worktrees.

With --parallel N, run at most N commands at once. For N > 1, buffer each
worktree's stdout and stderr in memory and print them with its header to stderr
as each command finishes. Parallel commands receive no stdin. The default, N = 1,
streams output and passes stdin through. With --fail-fast, stop starting commands
after a failure and wait for commands already running.

Examples:
  grove exec --all -- npm install                        # All worktrees
  grove exec --all -j 4 -- npm test                       # Four at a time
  grove exec main feature -- npm ci                      # Named worktrees
  grove exec --all --fail-fast -- go build               # Stop on first failure
  grove exec --all --json -- npm test                    # JSON results
  grove exec --all -- bash -c "npm install && npm test"  # Multiple commands`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: worktreeCompletionWithBranches(0, true, true, nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			dashPos := cmd.ArgsLenAtDash()
			if dashPos < 0 {
				return errors.New("missing \"--\" before the command")
			}
			return runExec(all, failFast, jsonOutput, parallel, args[:dashPos], args[dashPos:])
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Execute in all worktrees")
	cmd.Flags().IntVarP(&parallel, "parallel", "j", 1, "Maximum concurrent commands (buffer output when greater than 1)")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop starting commands after first failure")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().BoolP("help", "h", false, "Help for exec")

	return cmd
}

func runParallel(targets []execTarget, n int, run func(execTarget) execResult) []execResult {
	semaphore := make(chan struct{}, n)
	var waitGroup sync.WaitGroup
	var mutex sync.Mutex
	var outputMutex sync.Mutex
	results := make([]execResult, len(targets))
	stopped := false
	started := 0

	for index, target := range targets {
		semaphore <- struct{}{}
		mutex.Lock()
		if stopped {
			mutex.Unlock()
			<-semaphore
			break
		}

		waitGroup.Add(1)
		started++
		go func() {
			defer waitGroup.Done()

			result := run(target)
			output := result.output
			result.output = nil

			// Publish state and free the slot before flushing, so a large block
			// never stalls dispatch. The dispatcher takes the slot then the
			// mutex, so it always sees this stopped value.
			mutex.Lock()
			stopped = stopped || result.stop
			results[index] = result
			mutex.Unlock()
			<-semaphore

			outputMutex.Lock()
			defer outputMutex.Unlock()

			if result.label != "" {
				logger.Info("%s", result.label)
			}
			if output != nil {
				_, _ = output.WriteTo(os.Stderr)
			}
			if result.err != nil {
				logger.Error("%s", result.err)
			}
			if result.label != "" {
				fmt.Fprintln(os.Stderr)
			}
		}()
		mutex.Unlock()
	}

	waitGroup.Wait()

	// Report in target order, so --json matches the sequential array regardless of who finished first.
	return results[:started]
}

func runExec(all, failFast, jsonOutput bool, parallel int, worktrees, command []string) error {
	if parallel < 1 {
		return errors.New("parallel must be at least 1")
	}

	// Validation: must have a command
	if len(command) == 0 {
		return errors.New("no command specified after --")
	}

	// Validation: cannot use both --all and specific worktrees
	if all && len(worktrees) > 0 {
		return errors.New("cannot use --all with specific worktrees")
	}

	// Validation: must specify --all or at least one worktree
	if !all && len(worktrees) == 0 {
		return errors.New("must specify --all or at least one worktree")
	}

	_, _, infos, err := loadWorkspace(true)
	if err != nil {
		return err
	}

	// Determine which worktrees to execute in
	var targets []execTarget
	if all {
		for _, info := range infos {
			targets = append(targets, execTarget{label: formatter.WorktreeLabel(info), name: filepath.Base(info.Path), path: info.Path})
		}
	} else {
		resolved, err := resolveWorktrees(infos, worktrees)
		if err != nil {
			return err
		}
		for _, info := range resolved {
			targets = append(targets, execTarget{label: formatter.WorktreeLabel(info), name: filepath.Base(info.Path), path: info.Path})
		}
	}

	runOne := func(target execTarget, streaming bool) execResult {
		result := execResult{Name: target.name, Path: target.path}
		cmd := exec.Command(command[0], command[1:]...) //nolint:gosec
		cmd.Dir = target.path
		if streaming {
			if !jsonOutput {
				logger.Info("%s", target.label)
			}
			cmd.Stdin = os.Stdin
			if jsonOutput {
				cmd.Stdout = os.Stderr
			} else {
				cmd.Stdout = os.Stdout
			}
			cmd.Stderr = os.Stderr
		} else {
			result.output = &bytes.Buffer{}
			// Sharing the writer makes os/exec copy both streams without concurrent writes.
			cmd.Stdout = result.output
			cmd.Stderr = result.output
			if !jsonOutput {
				result.label = target.label
			}
		}

		if err := cmd.Run(); err != nil {
			result.ExitCode = commandExitCode(err)
			result.stop = failFast
			if !isExitError(err) {
				if streaming {
					logger.Error("%s", err)
				} else {
					result.err = err
				}
			}
		}

		return result
	}

	results := make([]execResult, 0, len(targets))
	if parallel > 1 {
		results = runParallel(targets, parallel, func(target execTarget) execResult {
			return runOne(target, false)
		})
	} else {
		for _, target := range targets {
			result := runOne(target, true)
			results = append(results, result)
			if result.stop {
				break
			}
			if !jsonOutput {
				fmt.Fprintln(os.Stderr) // Blank line between worktrees
			}
		}
	}

	var failed []string
	succeeded := 0
	for _, result := range results {
		if result.ExitCode != 0 {
			failed = append(failed, result.Name)
		} else {
			succeeded++
		}
	}

	total := len(results)
	failCount := len(failed)
	if jsonOutput {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(results); err != nil {
			return fmt.Errorf("encode exec results: %w", err)
		}
	} else {
		switch failCount {
		case 0:
			logger.Success("Executed in %d worktrees", total)
		case total:
			logger.Error("All %d executions failed", total)
		default:
			logger.Warning("Executed in %d worktrees (%d succeeded, %d failed)", total, succeeded, failCount)
		}
		for _, name := range failed {
			logger.Dimmed("%s", name)
		}
		if skipped := len(targets) - total; skipped > 0 {
			logger.Dimmed("Stopped early, %d worktrees skipped", skipped)
		}
	}

	exitCode := 1
	if len(targets) == 1 {
		exitCode = results[0].ExitCode
	}
	switch failCount {
	case 0:
		return nil
	case total:
		return &ExecError{message: "all executions failed", exitCode: exitCode}
	default:
		return &ExecError{message: "some executions failed", exitCode: exitCode}
	}
}

func commandExitCode(err error) int {
	var exitError *exec.ExitError
	if errors.As(err, &exitError) && exitError.ExitCode() >= 0 {
		return exitError.ExitCode()
	}
	return 1
}

func isExitError(err error) bool {
	var exitError *exec.ExitError
	return errors.As(err, &exitError)
}
