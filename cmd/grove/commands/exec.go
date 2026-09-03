package commands

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

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

	cmd := &cobra.Command{
		Use:   "exec [--all | <worktree>...] -- <command>",
		Short: "Execute a command in worktrees",
		Long: `Run a command in one or more worktrees.

Examples:
  grove exec --all -- npm install                        # All worktrees
  grove exec main feature -- npm ci                      # Named worktrees
  grove exec --all --fail-fast -- go build               # Stop on first failure
  grove exec --all --json -- npm test                    # JSON results
  grove exec --all -- bash -c "npm install && npm test"  # Multiple commands`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: worktreeCompletion(0, true, nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			dashPos := cmd.ArgsLenAtDash()
			if dashPos < 0 {
				return errors.New("missing \"--\" before the command")
			}
			return runExec(all, failFast, jsonOutput, args[:dashPos], args[dashPos:])
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Execute in all worktrees")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first failure")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output results as JSON")
	cmd.Flags().BoolP("help", "h", false, "Help for exec")

	return cmd
}

func runExec(all, failFast, jsonOutput bool, worktrees, command []string) error {
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

	// Execute command in each worktree
	var failed []string
	results := make([]execResult, 0, len(targets))
	succeeded := 0
	for _, target := range targets {
		if !jsonOutput {
			logger.Info("%s", target.label)
		}

		cmd := exec.Command(command[0], command[1:]...) //nolint:gosec
		cmd.Dir = target.path
		cmd.Stdin = os.Stdin
		if jsonOutput {
			cmd.Stdout = os.Stderr
		} else {
			cmd.Stdout = os.Stdout
		}
		cmd.Stderr = os.Stderr

		exitCode := 0
		if err := cmd.Run(); err != nil {
			exitCode = commandExitCode(err)
			failed = append(failed, target.name)
			if !isExitError(err) {
				logger.Error("%s", err)
			}
		} else {
			succeeded++
		}
		results = append(results, execResult{Name: target.name, Path: target.path, ExitCode: exitCode})

		if failFast && exitCode != 0 {
			break
		}
		if !jsonOutput {
			fmt.Fprintln(os.Stderr) // Blank line between worktrees
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
