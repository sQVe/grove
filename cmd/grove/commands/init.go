package commands

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

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

			logger.Info("Initialized grove workspace in: %s", styles.Render(&styles.Path, targetDir))
			return nil
		},
	}
	newCmd.Flags().BoolP("help", "h", false, "Help for new")
	initCmd.AddCommand(newCmd)

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
			logger.Success("Converted repository to grove workspace in: %s", styles.Render(&styles.Path, absPath))
			return nil
		},
	}
	convertCmd.Flags().StringVar(&convertBranches, "branches", "", "Additional branches to create worktrees for (comma-separated, current branch is always included)")
	convertCmd.Flags().BoolVar(&convertVerbose, "verbose", false, "Show detailed git output during conversion")
	_ = convertCmd.RegisterFlagCompletionFunc("branches", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		currentBranch, err := git.GetCurrentBranch(".")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		allBranches, err := git.ListBranches(".")
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		var availableBranches []string
		for _, branch := range allBranches {
			if branch != currentBranch {
				availableBranches = append(availableBranches, branch)
			}
		}

		completions := getBranchCompletions(toComplete, availableBranches)
		return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
	})
	convertCmd.Flags().BoolP("help", "h", false, "Help for convert")
	initCmd.AddCommand(convertCmd)

	return initCmd
}
