package switchcmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/commands/shared"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
)

type SwitchMode int

const (
	ModeAuto SwitchMode = iota
	ModeEval
	ModeSubshell
	ModeGetPath
)

const (
	defaultWindowsShell = "cmd.exe"
	defaultUnixShell    = "/bin/sh"
)

type SwitchOptions struct {
	Mode         SwitchMode
	Shell        string
	ForceInstall bool
}

func NewSwitchCmd() *cobra.Command {
	options := &SwitchOptions{
		Mode: ModeAuto,
	}

	var getPath, eval, subshell bool

	cmd := &cobra.Command{
		Use:   "switch <worktree-name>",
		Short: "Switch to an existing worktree",
		Long: `Switch to an existing Git worktree by name.

The switch command allows you to quickly change your current working directory
to any existing worktree. It offers multiple execution modes to accommodate
different shell environments and user preferences.

Examples:
  grove switch feature-branch      # Switch to worktree (requires shell integration)
  grove switch --get-path main     # Output the path to main worktree
  grove switch --eval feature      # Output cd command for shell evaluation
  grove switch --subshell main     # Launch new shell in main worktree

Execution modes:
  Default: Uses shell integration if available, falls back to --eval mode
  --get-path: Outputs only the absolute path to the worktree
  --eval: Outputs 'cd /path/to/worktree' for shell evaluation
  --subshell: Launches a new shell in the target directory`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			activeModeFlags := 0
			if getPath {
				activeModeFlags++
				options.Mode = ModeGetPath
			}
			if eval {
				activeModeFlags++
				options.Mode = ModeEval
			}
			if subshell {
				activeModeFlags++
				options.Mode = ModeSubshell
			}

			if activeModeFlags > 1 {
				return errors.NewGroveError(
					errors.ErrCodeConfigInvalid,
					"Cannot specify multiple mode flags (--get-path, --eval, --subshell) simultaneously",
					nil,
				)
			}

			if err := runSwitchCommand(args[0], options); err != nil {
				cmd.SilenceUsage = true
				return err
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&getPath, "get-path", false, "Output only the absolute path to the worktree")
	cmd.Flags().BoolVar(&eval, "eval", false, "Output cd command for shell evaluation")
	cmd.Flags().BoolVar(&subshell, "subshell", false, "Launch new shell in the target directory")
	cmd.Flags().StringVar(&options.Shell, "shell", "", "Target shell for integration (auto-detected if not specified)")
	cmd.Flags().BoolVar(&options.ForceInstall, "force-install", false, "Force shell integration installation")

	return cmd
}

func runSwitchCommand(worktreeName string, options *SwitchOptions) error {
	return runSwitchCommandWithExecutor(shared.DefaultExecutorProvider.GetExecutor(), worktreeName, options)
}

func runSwitchCommandWithExecutor(executor git.GitExecutor, worktreeName string, options *SwitchOptions) error {
	if strings.TrimSpace(worktreeName) == "" {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"worktree name cannot be empty",
			nil,
		)
	}

	service := NewSwitchService(executor)

	switch options.Mode {
	case ModeGetPath:
		return handleGetPathMode(service, worktreeName)
	case ModeEval:
		return handleEvalMode(service, worktreeName)
	case ModeSubshell:
		return handleSubshellMode(service, worktreeName)
	default:
		return handleAutoMode(service, worktreeName, options)
	}
}

func handleGetPathMode(service SwitchService, worktreeName string) error {
	path, err := service.GetWorktreePath(worktreeName)
	if err != nil {
		return err
	}

	fmt.Println(path)
	return nil
}

func handleEvalMode(service SwitchService, worktreeName string) error {
	path, err := service.GetWorktreePath(worktreeName)
	if err != nil {
		return err
	}

	fmt.Printf("cd %q\n", path)
	return nil
}

func handleSubshellMode(service SwitchService, worktreeName string) error {
	path, err := service.GetWorktreePath(worktreeName)
	if err != nil {
		return err
	}

	shell := resolveShell()
	if err := validateShellPath(shell); err != nil {
		return err
	}

	if err := validateDirectory(path); err != nil {
		return err
	}

	return launchSubshell(shell, path, worktreeName)
}

func resolveShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		// Default shell based on platform
		if runtime.GOOS == "windows" {
			shell = defaultWindowsShell
		} else {
			shell = defaultUnixShell
		}
	}
	return shell
}

func validateDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return errors.ErrDirectoryAccess(path, err)
	} else if err != nil {
		return errors.ErrDirectoryAccess(path, err)
	}
	return nil
}

func launchSubshell(shell, path, worktreeName string) error {
	fmt.Printf("Launching subshell in worktree '%s' (%s)\n", worktreeName, path)
	fmt.Printf("Type 'exit' to return to the original directory.\n\n")

	cmd := exec.Command(shell)
	cmd.Dir = path
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			fmt.Sprintf("failed to launch shell '%s'", shell),
			err,
		).WithContext("shell", shell).WithContext("directory", path)
	}
	return nil
}

func validateShellPath(shell string) error {
	if shell == "" {
		return errors.NewGroveError(
			errors.ErrCodeConfigInvalid,
			"shell path cannot be empty",
			nil,
		)
	}

	if _, err := exec.LookPath(shell); err != nil {
		return errors.NewGroveError(
			errors.ErrCodeFileSystem,
			fmt.Sprintf("shell '%s' not found or not executable", shell),
			err,
		).WithContext("shell", shell)
	}

	return nil
}

func handleAutoMode(service SwitchService, worktreeName string, options *SwitchOptions) error {
	// For now, auto mode will fallback to eval mode since shell integration
	// detection and installation will be implemented in later tasks

	// Check if shell integration is available (placeholder for now)
	if checkShellIntegration() {
		// If shell integration is available, this would normally change the directory
		// For now, we'll show what would happen
		path, err := service.GetWorktreePath(worktreeName)
		if err != nil {
			return err
		}
		fmt.Printf("Shell integration not yet implemented. Would switch to: %s\n", path)
		return nil
	}

	fmt.Println("Shell integration not detected. Use one of these alternatives:")
	fmt.Println()
	fmt.Printf("1. Evaluate in your shell:    eval \"$(grove switch --eval %s)\"\n", worktreeName)
	fmt.Printf("2. Launch subshell:           grove switch --subshell %s\n", worktreeName)
	fmt.Printf("3. Get path for manual cd:    grove switch --get-path %s\n", worktreeName)
	fmt.Println()
	fmt.Println("To enable seamless switching, install shell integration:")
	fmt.Println("  grove install-shell-integration")

	return nil
}

func checkShellIntegration() bool {
	// Placeholder implementation - this will be enhanced in later tasks
	// to check for actual shell function presence
	return false
}
