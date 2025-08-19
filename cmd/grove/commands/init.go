package commands

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/workspace"
)

// getBranchCompletions provides completion suggestions for comma-separated branches
func getBranchCompletions(toComplete string, allBranches []string) []string {
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

	lastPart := parts[len(parts)-1]
	prefixParts := parts[:len(parts)-1]

	// Track already selected branches
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
	seen := make(map[string]bool) // Prevent duplicates in output

	for _, branch := range allBranches {
		if selected[branch] {
			continue
		}
		if strings.HasPrefix(branch, lastPart) {
			completion := prefix + branch
			if !seen[completion] {
				completions = append(completions, completion)
				seen[completion] = true
			}
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

	newCmd := &cobra.Command{
		Use:   "new [directory]",
		Short: "Create a new grove workspace",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		Run: func(cmd *cobra.Command, args []string) {
			var targetDir string
			if len(args) == 0 {
				var err error
				targetDir, err = os.Getwd()
				if err != nil {
					logger.Error("Failed to get current directory: %v", err)
					os.Exit(1)
				}
			} else {
				targetDir = args[0]
				var err error
				targetDir, err = filepath.Abs(targetDir)
				if err != nil {
					logger.Error("Failed to get absolute path: %v", err)
					os.Exit(1)
				}
			}

			logger.Debug("Initializing grove workspace in: %s", targetDir)

			if err := workspace.Initialize(targetDir); err != nil {
				logger.Error("Failed to initialize workspace: %v", err)
				os.Exit(1)
			}

			logger.Info("Initialized grove workspace in: %s", targetDir)
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
		Run: func(cmd *cobra.Command, args []string) {
			if cmd.Flags().Changed("branches") && (branches == "" || branches == `""`) {
				logger.Error("no branches specified")
				os.Exit(1)
			}

			url := args[0]

			var targetDir string
			if len(args) == 1 {
				var err error
				targetDir, err = os.Getwd()
				if err != nil {
					logger.Error("Failed to get current directory: %v", err)
					os.Exit(1)
				}
			} else {
				targetDir = args[1]
				var err error
				targetDir, err = filepath.Abs(targetDir)
				if err != nil {
					logger.Error("Failed to get absolute path: %v", err)
					os.Exit(1)
				}
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
				os.Exit(1)
			}

			logger.Info("Initialized grove workspace in: %s", targetDir)
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
