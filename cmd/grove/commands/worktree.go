package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/workspace"
)

func loadWorkspace(fast bool) (cwd, bareDir string, infos []*git.WorktreeInfo, err error) {
	cwd, err = os.Getwd()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err = workspace.FindBareDir(cwd)
	if err != nil {
		return "", "", nil, err
	}

	infos, err = git.ListWorktreesWithInfo(bareDir, fast)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return cwd, bareDir, infos, nil
}

func resolveWorktrees(infos []*git.WorktreeInfo, targets []string) ([]*git.WorktreeInfo, error) {
	seen := make(map[string]bool)
	resolved := make([]*git.WorktreeInfo, 0, len(targets))
	for _, target := range targets {
		info := git.FindWorktree(infos, target)
		if info == nil {
			return nil, fmt.Errorf("worktree not found: %s", target)
		}
		if !seen[info.Path] {
			seen[info.Path] = true
			resolved = append(resolved, info)
		}
	}
	return resolved, nil
}

func worktreeCompletion(maxArgs int, afterDash bool, include func(string, *git.WorktreeInfo) bool) cobra.CompletionFunc {
	return func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if afterDash && slices.Contains(os.Args, "--") {
			return nil, cobra.ShellCompDirectiveDefault
		}
		if maxArgs > 0 && len(args) >= maxArgs {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}

		cwd, _, infos, err := loadWorkspace(true)
		if err != nil {
			return nil, cobra.ShellCompDirectiveError
		}

		used := make(map[string]bool, len(args))
		for _, arg := range args {
			used[arg] = true
		}

		describeBranch := cmd.Name() == "switch" || cmd.Name() == "remove" || cmd.Name() == "exec"
		var completions []string
		for _, info := range infos {
			name := filepath.Base(info.Path)
			if used[name] || used[info.Branch] || (include != nil && !include(cwd, info)) {
				continue
			}

			if strings.HasPrefix(name, toComplete) {
				completion := name
				if describeBranch && !info.Detached && info.Branch != "" {
					completion += "\t" + info.Branch
				}

				completions = append(completions, completion)
			}

			if describeBranch && !info.Detached && info.Branch != "" && info.Branch != name && strings.HasPrefix(info.Branch, toComplete) {
				completions = append(completions, info.Branch)
			}
		}
		return completions, cobra.ShellCompDirectiveNoFileComp
	}
}

func notCurrentWorktree(cwd string, info *git.WorktreeInfo) bool {
	return !fs.PathsEqual(cwd, info.Path) && !fs.PathHasPrefix(cwd, info.Path)
}
