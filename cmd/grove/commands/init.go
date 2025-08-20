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

// parseBranchInput parses comma-separated branch input into parts
func parseBranchInput(toComplete string) (allParts []string, lastPart string, prefixParts []string) {
	rawParts := strings.Split(toComplete, ",")
	parts := make([]string, 0, len(rawParts))

	for i, p := range rawParts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" || i == len(rawParts)-1 {
			parts = append(parts, trimmed)
		}
	}

	if len(parts) == 0 {
		parts = []string{""}
	}

	lastPart = parts[len(parts)-1]
	prefixParts = parts[:len(parts)-1]
	allParts = parts
	return
}

// trackSelectedBranches creates a map of already selected branches
func trackSelectedBranches(prefixParts []string) map[string]bool {
	selected := make(map[string]bool)
	for _, p := range prefixParts {
		if p != "" {
			selected[p] = true
		}
	}
	return selected
}

// filterAvailableBranches filters branches based on prefix and selection
func filterAvailableBranches(allBranches []string, lastPart string, selected map[string]bool) []string {
	var filtered []string
	for _, branch := range allBranches {
		if !selected[branch] && strings.HasPrefix(branch, lastPart) {
			filtered = append(filtered, branch)
		}
	}
	return filtered
}

// formatCompletions formats filtered branches with prefix
func formatCompletions(filteredBranches, prefixParts []string) []string {
	prefix := ""
	if len(prefixParts) > 0 {
		prefix = strings.Join(prefixParts, ",") + ","
	}

	var completions []string
	seen := make(map[string]bool)

	for _, branch := range filteredBranches {
		completion := prefix + branch
		if !seen[completion] {
			completions = append(completions, completion)
			seen[completion] = true
		}
	}

	return completions
}

// getBranchCompletions provides completion suggestions for comma-separated branches
func getBranchCompletions(toComplete string, allBranches []string) []string {
	_, lastPart, prefixParts := parseBranchInput(toComplete)
	selected := trackSelectedBranches(prefixParts)
	filtered := filterAvailableBranches(allBranches, lastPart, selected)
	return formatCompletions(filtered, prefixParts)
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
				logger.Error("Failed to resolve target directory: %v", err)
				return err
			}

			logger.Debug("Initializing grove workspace in: %s", targetDir)

			if err := workspace.Initialize(targetDir); err != nil {
				logger.Error("Failed to initialize workspace: %v", err)
				return err
			}

			logger.Info("Initialized grove workspace in: %s", targetDir)
			return nil
		},
	}
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
				logger.Error("no branches specified")
				return fmt.Errorf("no branches specified")
			}

			url := args[0]

			targetDir, err := resolveTargetDirectory(args, 1)
			if err != nil {
				logger.Error("Failed to resolve target directory: %v", err)
				return err
			}

			logger.Debug("Cloning and initializing grove workspace in: %s", targetDir)
			if branches != "" {
				logger.Debug("Branches requested: %s", branches)
			}
			if verbose {
				logger.Debug("Verbose mode enabled")
			}

			if err := workspace.CloneAndInitialize(url, targetDir, branches, verbose); err != nil {
				logger.Error("Failed to clone and initialize workspace: %v", err)
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
	initCmd.AddCommand(cloneCmd)

	return initCmd
}
