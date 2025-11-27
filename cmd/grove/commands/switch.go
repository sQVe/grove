package commands

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/workspace"
)

var ErrWorktreeNotFound = errors.New("worktree not found")

func NewSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "switch <branch>",
		Short:             "Switch to a worktree",
		Long:              `Output the path to a worktree for the given branch. Use with the gw shell function for seamless directory switching.`,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeSwitchArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSwitch(args[0])
		},
	}

	cmd.Flags().BoolP("help", "h", false, "Help for switch")

	return cmd
}

func runSwitch(branch string) error {
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
		if info.Path != cwd {
			completions = append(completions, info.Branch)
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}
