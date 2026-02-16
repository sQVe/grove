package commands

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/formatter"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

type execTarget struct {
	label string
	name  string
	path  string
}

// NewExecCmd creates the exec command
func NewExecCmd() *cobra.Command {
	var all bool
	var failFast bool

	cmd := &cobra.Command{
		Use:   "exec [--all | <worktree>...] -- <command>",
		Short: "Execute a command in worktrees",
		Long: `Run a command in one or more worktrees.

Examples:
  grove exec --all -- npm install                        # All worktrees
  grove exec main feature -- npm ci                      # Named worktrees
  grove exec --all --fail-fast -- go build               # Stop on first failure
  grove exec --all -- bash -c "npm install && npm test"  # Multiple commands`,
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completeExecArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse args using ArgsLenAtDash: before -- is worktrees, after is command
			dashPos := cmd.ArgsLenAtDash()
			var worktrees, command []string
			if dashPos < 0 {
				// No -- found, treat all args as command (requires --all)
				command = args
			} else {
				worktrees = args[:dashPos]
				command = args[dashPos:]
			}
			return runExec(all, failFast, worktrees, command)
		},
	}

	cmd.Flags().BoolVarP(&all, "all", "a", false, "Execute in all worktrees")
	cmd.Flags().BoolVar(&failFast, "fail-fast", false, "Stop on first failure")
	cmd.Flags().BoolP("help", "h", false, "Help for exec")

	return cmd
}

func runExec(all, failFast bool, worktrees, command []string) error {
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

	// Get workspace
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Get worktree info
	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Determine which worktrees to execute in
	var targets []execTarget
	if all {
		for _, info := range infos {
			targets = append(targets, execTarget{label: formatter.WorktreeLabel(info), name: filepath.Base(info.Path), path: info.Path})
		}
	} else {
		seen := make(map[string]bool)
		for _, name := range worktrees {
			info := git.FindWorktree(infos, name)
			if info == nil {
				return fmt.Errorf("worktree not found: %s", name)
			}
			if seen[info.Path] {
				continue
			}
			seen[info.Path] = true
			targets = append(targets, execTarget{label: formatter.WorktreeLabel(info), name: filepath.Base(info.Path), path: info.Path})
		}
	}

	// Execute command in each worktree
	var failed []string
	succeeded := 0
	for _, target := range targets {
		logger.Info("%s", target.label)

		cmd := exec.Command(command[0], command[1:]...) //nolint:gosec
		cmd.Dir = target.path
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			failed = append(failed, target.name)
			if failFast {
				return fmt.Errorf("command failed in %s: %w", target.name, err)
			}
		} else {
			succeeded++
		}
		fmt.Fprintln(os.Stderr) // Blank line between worktrees
	}

	// Print summary
	total := len(targets)
	failCount := len(failed)

	switch failCount {
	case 0:
		logger.Success("Executed in %d worktrees", total)
	case total:
		logger.Error("All %d executions failed", total)
		return errors.New("all executions failed")
	default:
		logger.Warning("Executed in %d worktrees (%d succeeded, %d failed)", total, succeeded, failCount)
		return errors.New("some executions failed")
	}

	return nil
}

func completeExecArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if slices.Contains(os.Args, "--") {
		return nil, cobra.ShellCompDirectiveDefault
	}

	cwd, err := os.Getwd()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	// Build set of already-specified worktrees
	alreadyUsed := make(map[string]bool)
	for _, arg := range args {
		alreadyUsed[arg] = true
	}

	// Return worktrees not already specified (by directory name)
	var completions []string
	for _, info := range infos {
		name := filepath.Base(info.Path)
		if !alreadyUsed[name] && !alreadyUsed[info.Branch] {
			completions = append(completions, name)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
