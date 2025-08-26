package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

// resolveTargetDirectory resolves the target directory from command arguments
func resolveTargetDirectory(args []string, argIndex int) (string, error) {
	if len(args) <= argIndex {
		return os.Getwd()
	}
	return filepath.Abs(args[argIndex])
}

// getBranchCompletions provides completion suggestions for comma-separated branches
func getBranchCompletions(toComplete string, allBranches []string) []string {
	rawParts := strings.Split(toComplete, ",")
	var parts []string

	// Filter out empty parts except for the last one
	for i, p := range rawParts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" || i == len(rawParts)-1 {
			parts = append(parts, trimmed)
		}
	}

	if len(parts) == 0 {
		parts = []string{""}
	}

	lastPart := parts[len(parts)-1]
	prefixParts := parts[:len(parts)-1]

	selected := make(map[string]bool)
	for _, p := range prefixParts {
		if p != "" {
			selected[p] = true
		}
	}

	prefix := ""
	if len(prefixParts) > 0 {
		prefix = strings.Join(prefixParts, ",") + ","
	}

	var completions []string
	for _, branch := range allBranches {
		if !selected[branch] && strings.HasPrefix(branch, lastPart) {
			completions = append(completions, prefix+branch)
		}
	}

	return completions
}

func NewInitCmd() *cobra.Command {
	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize grove in current repository",
		Long:  `Initialize grove in current repository`,
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}
	initCmd.Flags().BoolP("help", "h", false, "Help for init")

	newCmd := &cobra.Command{
		Use:   "new [directory]",
		Short: "Create a new grove workspace",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir, err := resolveTargetDirectory(args, 0)
			if err != nil {
				return err
			}

			if err := workspace.Initialize(targetDir); err != nil {
				return err
			}

			logger.Info("Initialized grove workspace in: %s", targetDir)
			return nil
		},
	}
	newCmd.Flags().BoolP("help", "h", false, "Help for new")
	initCmd.AddCommand(newCmd)

	var branches string
	var verbose bool
	cloneCmd := &cobra.Command{
		Use:   "clone <url> [directory]",
		Short: "Clone a repository and create a grove workspace",
		Args:  cobra.RangeArgs(1, 2),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("branches") && (branches == "" || branches == `""`) {
				return fmt.Errorf("no branches specified")
			}

			url := args[0]

			targetDir, err := resolveTargetDirectory(args, 1)
			if err != nil {
				return err
			}

			if err := workspace.CloneAndInitialize(url, targetDir, branches, verbose); err != nil {
				return err
			}

			logger.Info("Initialized grove workspace in: %s", targetDir)
			return nil
		},
	}
	cloneCmd.Flags().StringVar(&branches, "branches", "", "Comma-separated list of branches to create worktrees for")
	cloneCmd.Flags().BoolVar(&verbose, "verbose", false, "Show detailed git output during clone and worktree creation")
	_ = cloneCmd.RegisterFlagCompletionFunc("branches", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		url := args[0]
		remoteBranches, err := git.ListRemoteBranches(url)
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		completions := getBranchCompletions(toComplete, remoteBranches)
		return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	})
	cloneCmd.Flags().BoolP("help", "h", false, "Help for clone")
	initCmd.AddCommand(cloneCmd)

	var convertBranches string
	var convertVerbose bool
	convertCmd := &cobra.Command{
		Use:   "convert",
		Short: "Convert existing Git repository to a grove workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			targetDir := "."

			if err := workspace.Convert(targetDir, convertBranches, convertVerbose); err != nil {
				return err
			}

			absPath, err := filepath.Abs(targetDir)
			if err != nil {
				absPath = targetDir
			}
			logger.Success("Repository successfully converted to Grove workspace")
			logger.Info("Converted repository to grove workspace in: %s", absPath)
			return nil
		},
	}
	convertCmd.Flags().StringVar(&convertBranches, "branches", "", "Comma-separated list of branches to create worktrees for")
	convertCmd.Flags().BoolVar(&convertVerbose, "verbose", false, "Show detailed git output during conversion")
	_ = convertCmd.RegisterFlagCompletionFunc("branches", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		// TODO: Implement branch completion
		// - Get local and remote branches from current Git repository
		// - Use getBranchCompletions for comma-separated support
		return nil, cobra.ShellCompDirectiveNoFileComp
	})
	convertCmd.Flags().BoolP("help", "h", false, "Help for convert")
	initCmd.AddCommand(convertCmd)

	return initCmd
}
