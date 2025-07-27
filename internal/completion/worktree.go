package completion

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/logger"
)

func WorktreeCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("worktree_completion")

	if !ctx.IsInGroveRepo() {
		log.Debug("not in grove repository, skipping worktree completion")
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	worktrees, err := ctx.WithTimeout(func() ([]string, error) {
		return getWorktreeNames(ctx)
	})
	if err != nil {
		log.Debug("failed to get worktree names", "error", err)
		return nil, cobra.ShellCompDirectiveError
	}

	filtered := FilterCompletions(worktrees, toComplete)

	log.Debug("worktree completion results", "total", len(worktrees), "filtered", len(filtered), "input", toComplete)
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

func getWorktreeNames(ctx *CompletionContext) ([]string, error) {
	log := logger.WithComponent("worktree_completion")

	if worktrees, exists := GetCachedWorktrees(ctx); exists {
		log.Debug("using cached worktree names", "count", len(worktrees))
		return worktrees, nil
	}

	output, err := ctx.Executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		log.Debug("failed to get worktree list", "error", err)
		return nil, err
	}

	var worktrees []string
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")

			dirName := filepath.Base(path)

			if dirName != "." && dirName != "" {
				worktrees = append(worktrees, dirName)
			}
		}
	}

	log.Debug("collected worktree names", "count", len(worktrees))

	SetCachedWorktrees(ctx, worktrees)

	return worktrees, nil
}

func WorktreePathCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("worktree_path_completion")

	if !ctx.IsInGroveRepo() {
		log.Debug("not in grove repository, skipping worktree path completion")
		return nil, cobra.ShellCompDirectiveDefault
	}

	paths, err := ctx.WithTimeout(func() ([]string, error) {
		return getWorktreePaths(ctx)
	})
	if err != nil {
		log.Debug("failed to get worktree paths", "error", err)
		return nil, cobra.ShellCompDirectiveError
	}

	filtered := FilterCompletions(paths, toComplete)

	log.Debug("worktree path completion results", "total", len(paths), "filtered", len(filtered), "input", toComplete)
	return filtered, cobra.ShellCompDirectiveDefault
}

func getWorktreePaths(ctx *CompletionContext) ([]string, error) {
	log := logger.WithComponent("worktree_path_completion")

	output, err := ctx.Executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		log.Debug("failed to get worktree list", "error", err)
		return nil, err
	}

	var paths []string
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "worktree ") {
			path := strings.TrimPrefix(line, "worktree ")

			if !strings.HasSuffix(path, ".") {
				paths = append(paths, path)
			}
		}
	}

	log.Debug("collected worktree paths", "count", len(paths))
	return paths, nil
}

func BranchToWorktreeName(branchName string) string {
	// Convert filesystem-unsafe characters to hyphens for directory naming.
	name := strings.ReplaceAll(branchName, "/", "-")
	name = strings.ReplaceAll(name, "\\", "-")
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, "?", "-")
	name = strings.ReplaceAll(name, "*", "-")
	name = strings.ReplaceAll(name, "[", "-")
	name = strings.ReplaceAll(name, "]", "-")
	name = strings.ReplaceAll(name, "<", "-")
	name = strings.ReplaceAll(name, ">", "-")
	name = strings.ReplaceAll(name, "|", "-")
	name = strings.ReplaceAll(name, "\"", "-")

	name = strings.Trim(name, "-")

	return name
}

func WorktreeNameToBranch(worktreeName string) string {
	// Best-effort conversion - not always reversible due to character replacements.
	return strings.ReplaceAll(worktreeName, "-", "/")
}

func SuggestWorktreeNamesForBranch(branchName string) []string {
	var suggestions []string

	safeName := BranchToWorktreeName(branchName)
	suggestions = append(suggestions, safeName)

	if branchName == "main" || branchName == "master" {
		suggestions = append(suggestions, branchName)
	}

	if strings.HasPrefix(branchName, "feature/") {
		shortName := strings.TrimPrefix(branchName, "feature/")
		suggestions = append(suggestions, shortName)
	}

	return suggestions
}

func GetWorktreeInfo(ctx *CompletionContext) ([]WorktreeInfo, error) {
	log := logger.WithComponent("worktree_info")

	output, err := ctx.Executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		log.Debug("failed to get worktree info", "error", err)
		return nil, err
	}

	var worktrees []WorktreeInfo
	lines := strings.Split(strings.TrimSpace(output), "\n")

	var currentWorktree *WorktreeInfo
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			if currentWorktree != nil {
				worktrees = append(worktrees, *currentWorktree)
			}
			currentWorktree = &WorktreeInfo{
				Path: strings.TrimPrefix(line, "worktree "),
			}
		} else if currentWorktree != nil {
			switch {
			case strings.HasPrefix(line, "HEAD "):
				currentWorktree.Head = strings.TrimPrefix(line, "HEAD ")
			case strings.HasPrefix(line, "branch "):
				currentWorktree.Branch = strings.TrimPrefix(line, "branch ")
			case line == "bare":
				currentWorktree.IsBare = true
			case line == "detached":
				currentWorktree.IsDetached = true
			}
		}
	}

	if currentWorktree != nil {
		worktrees = append(worktrees, *currentWorktree)
	}

	log.Debug("collected worktree info", "count", len(worktrees))
	return worktrees, nil
}

type WorktreeInfo struct {
	Path       string
	Head       string
	Branch     string
	IsBare     bool
	IsDetached bool
}

func (w WorktreeInfo) Name() string {
	return filepath.Base(w.Path)
}

func (w WorktreeInfo) IsMainWorktree() bool {
	return w.IsBare || w.Name() == "."
}
