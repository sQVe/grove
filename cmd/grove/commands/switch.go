package commands

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/workspace"
)

var ErrWorktreeNotFound = errors.New("worktree not found")

func NewSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "switch <branch>",
		Short: "Switch to a worktree",
		Long: `Output the path to a worktree for the given branch.

Setup shell integration for seamless directory switching:
  eval "$(grove switch shell-init)"

Then use 'grove switch <branch>' to switch between worktrees.`,
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
	return &cobra.Command{
		Use:    "shell-init",
		Short:  "Output shell function for directory switching",
		Long:   `Output a shell function that wraps grove to enable seamless directory changes with 'grove switch' and 'grove add --switch'. Add to your shell config with: eval "$(grove switch shell-init)"`,
		Hidden: true,
		Args:   cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			shellFunc := `# Grove shell integration
# Wraps grove to enable 'grove switch' and 'grove add --switch' to change directories
grove() {
    if [[ "$1" == "switch" ]]; then
        local target exit_code
        target="$(command grove switch "${@:2}")"
        exit_code=$?
        if [[ $exit_code -eq 0 && -d "$target" ]]; then
            cd "$target"
        else
            [[ -n "$target" ]] && printf '%s\n' "$target"
            return $exit_code
        fi
    elif [[ "$1" == "add" ]] && [[ " ${*:2} " =~ " -s " || " ${*:2} " =~ " --switch " ]]; then
        local target exit_code
        target="$(command grove add "${@:2}")"
        exit_code=$?
        if [[ $exit_code -eq 0 && -d "$target" ]]; then
            cd "$target"
        else
            [[ -n "$target" ]] && printf '%s\n' "$target"
            return $exit_code
        fi
    else
        command grove "$@"
    fi
}
`
			fmt.Print(shellFunc)
			return nil
		},
	}
}

func runSwitch(branch string) error {
	branch = strings.TrimSpace(branch)

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

	for _, info := range infos {
		if info.Branch == branch {
			fmt.Println(info.Path)
			return nil
		}
	}

	return fmt.Errorf("%w: %s", ErrWorktreeNotFound, branch)
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
		inWorktree := cwd == info.Path || strings.HasPrefix(cwd, info.Path+string(os.PathSeparator))
		if !inWorktree {
			completions = append(completions, info.Branch)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
