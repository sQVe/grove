package commands

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/workspace"
)

//go:embed shell/grove.sh
var shellPOSIX string

//go:embed shell/grove.fish
var shellFish string

//go:embed shell/grove.ps1
var shellPowerShell string

var ErrWorktreeNotFound = errors.New("worktree not found")

// Shell type constants
const (
	shellBashType       = "bash"
	shellZshType        = "zsh"
	shellFishType       = "fish"
	shellShType         = "sh"
	shellPowerShellType = "powershell"
)

func NewSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <worktree>",
		Short: "Switch to a worktree",
		Long: `Switch to a worktree by name or branch.

Requires shell integration:
  eval "$(grove switch shell-init)"

Accepts worktree name (directory) or branch name.

Examples:
  grove switch main        # Switch to main worktree
  grove switch feat-auth   # Switch by directory name
  grove switch feat/auth   # Switch by branch name`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeSwitchArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSwitch(args[0])
		},
	}

	cmd.Flags().BoolP("help", "h", false, "Help for switch")

	cmd.AddCommand(newShellInitCmd())

	return cmd
}

func newShellInitCmd() *cobra.Command {
	var shellType string

	cmd := &cobra.Command{
		Use:   "shell-init",
		Short: "Output shell function for directory switching",
		Long: `Print shell integration for 'grove switch' and 'grove add --switch'.

Supported: bash, zsh, fish, sh, powershell

Add to shell config:
  bash/zsh/sh: eval "$(grove switch shell-init)"
  fish:        grove switch shell-init | source
  powershell:  grove switch shell-init | Invoke-Expression`,
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := shellType
			if shell == "" {
				shell = detectShell()
			}
			return printShellIntegration(shell)
		},
	}

	cmd.Flags().StringVar(&shellType, "shell", "", "Shell type (bash, zsh, fish, sh, powershell)")

	_ = cmd.RegisterFlagCompletionFunc("shell", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"bash", "zsh", "fish", "sh", "powershell"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// detectShell attempts to determine the current shell from environment
func detectShell() string {
	// Check SHELL environment variable (Unix)
	shell := os.Getenv("SHELL")
	if shell != "" {
		base := filepath.Base(shell)
		switch base {
		case shellBashType, shellZshType, shellShType, "dash", "ash":
			return shellShType // Use POSIX for all sh-compatible shells
		case shellFishType:
			return shellFishType
		}
	}

	// Check for PowerShell on Windows
	if psModulePath := os.Getenv("PSModulePath"); psModulePath != "" {
		return shellPowerShellType
	}

	// Default to sh (POSIX, most portable)
	return shellShType
}

// printShellIntegration outputs the appropriate shell integration script
func printShellIntegration(shell string) error {
	switch shell {
	case shellBashType, shellZshType, shellShType, "dash", "ash", "posix":
		fmt.Print(shellPOSIX)
	case shellFishType:
		fmt.Print(shellFish)
	case shellPowerShellType, "pwsh":
		fmt.Print(shellPowerShell)
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, sh, powershell)", shell)
	}
	return nil
}

func runSwitch(target string) error {
	target = strings.TrimSpace(target)

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	infos, err := git.ListWorktreesWithInfo(bareDir, true)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// First try to match by worktree name (directory basename)
	for _, info := range infos {
		if filepath.Base(info.Path) == target {
			fmt.Println(info.Path)
			return nil
		}
	}

	// Fall back to matching by branch name (backwards compatibility)
	for _, info := range infos {
		if info.Branch == target {
			fmt.Println(info.Path)
			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrWorktreeNotFound, target)
}

func completeSwitchArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
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

	var completions []string
	for _, info := range infos {
		// Exclude current worktree (check if cwd is at root or inside this worktree)
		inWorktree := fs.PathsEqual(cwd, info.Path) || fs.PathHasPrefix(cwd, info.Path)
		if !inWorktree {
			// Suggest worktree name (directory basename)
			completions = append(completions, filepath.Base(info.Path))
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
