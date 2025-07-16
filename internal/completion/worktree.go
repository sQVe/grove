package completion

import (
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/logger"
)

// WorktreeCompletion provides completion for worktree directory names
func WorktreeCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("worktree_completion")

	// Check if we're in a repository
	if !ctx.IsInGroveRepo() {
		log.Debug("not in grove repository, skipping worktree completion")
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get worktree names with timeout
	worktrees, err := ctx.WithTimeout(func() ([]string, error) {
		return getWorktreeNames(ctx)
	})
	if err != nil {
		log.Debug("failed to get worktree names", "error", err)
		return nil, cobra.ShellCompDirectiveError
	}

	// Filter worktrees based on current input
	filtered := FilterCompletions(worktrees, toComplete)

	log.Debug("worktree completion results", "total", len(worktrees), "filtered", len(filtered), "input", toComplete)
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// getWorktreeNames retrieves worktree directory names
func getWorktreeNames(ctx *CompletionContext) ([]string, error) {
	log := logger.WithComponent("worktree_completion")

	// Check cache first
	if worktrees, exists := GetCachedWorktrees(ctx); exists {
		log.Debug("using cached worktree names", "count", len(worktrees))
		return worktrees, nil
	}

	// Get worktree list
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
			// Extract path from "worktree /path/to/worktree"
			path := strings.TrimPrefix(line, "worktree ")

			// Get directory name (last component of path)
			dirName := filepath.Base(path)

			// Skip the main worktree (usually ".")
			if dirName != "." && dirName != "" {
				worktrees = append(worktrees, dirName)
			}
		}
	}

	log.Debug("collected worktree names", "count", len(worktrees))

	// Cache the result
	SetCachedWorktrees(ctx, worktrees)

	return worktrees, nil
}

// WorktreePathCompletion provides completion for worktree paths
func WorktreePathCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("worktree_path_completion")

	// Check if we're in a repository
	if !ctx.IsInGroveRepo() {
		log.Debug("not in grove repository, skipping worktree path completion")
		return nil, cobra.ShellCompDirectiveDefault
	}

	// Get worktree paths with timeout
	paths, err := ctx.WithTimeout(func() ([]string, error) {
		return getWorktreePaths(ctx)
	})
	if err != nil {
		log.Debug("failed to get worktree paths", "error", err)
		return nil, cobra.ShellCompDirectiveError
	}

	// Filter paths based on current input
	filtered := FilterCompletions(paths, toComplete)

	log.Debug("worktree path completion results", "total", len(paths), "filtered", len(filtered), "input", toComplete)
	return filtered, cobra.ShellCompDirectiveDefault
}

// getWorktreePaths retrieves worktree full paths
func getWorktreePaths(ctx *CompletionContext) ([]string, error) {
	log := logger.WithComponent("worktree_path_completion")

	// Get worktree list
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
			// Extract path from "worktree /path/to/worktree"
			path := strings.TrimPrefix(line, "worktree ")

			// Skip the main worktree (usually current directory)
			if !strings.HasSuffix(path, ".") {
				paths = append(paths, path)
			}
		}
	}

	log.Debug("collected worktree paths", "count", len(paths))
	return paths, nil
}

// BranchToWorktreeName converts a branch name to a filesystem-safe worktree directory name
func BranchToWorktreeName(branchName string) string {
	// Replace characters that are problematic in directory names
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

	// Remove leading/trailing dashes
	name = strings.Trim(name, "-")

	return name
}

// WorktreeNameToBranch converts a worktree directory name back to a branch name
func WorktreeNameToBranch(worktreeName string) string {
	// This is a best-effort conversion since the transformation is not always reversible
	// In practice, Grove should store the mapping between worktree names and branches
	return strings.ReplaceAll(worktreeName, "-", "/")
}

// SuggestWorktreeNamesForBranch suggests worktree directory names for a branch
func SuggestWorktreeNamesForBranch(branchName string) []string {
	var suggestions []string

	// Primary suggestion: filesystem-safe version
	safeName := BranchToWorktreeName(branchName)
	suggestions = append(suggestions, safeName)

	// Alternative suggestion: simple name for main branches
	if branchName == "main" || branchName == "master" {
		suggestions = append(suggestions, branchName)
	}

	// For feature branches, suggest shortened versions
	if strings.HasPrefix(branchName, "feature/") {
		shortName := strings.TrimPrefix(branchName, "feature/")
		suggestions = append(suggestions, shortName)
	}

	return suggestions
}

// GetWorktreeInfo retrieves information about existing worktrees
func GetWorktreeInfo(ctx *CompletionContext) ([]WorktreeInfo, error) {
	log := logger.WithComponent("worktree_info")

	// Get worktree list with detailed information
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
			// Start of new worktree info
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

	// Add the last worktree
	if currentWorktree != nil {
		worktrees = append(worktrees, *currentWorktree)
	}

	log.Debug("collected worktree info", "count", len(worktrees))
	return worktrees, nil
}

// WorktreeInfo represents information about a worktree
type WorktreeInfo struct {
	Path       string
	Head       string
	Branch     string
	IsBare     bool
	IsDetached bool
}

// Name returns the directory name of the worktree
func (w WorktreeInfo) Name() string {
	return filepath.Base(w.Path)
}

// IsMainWorktree checks if this is the main worktree
func (w WorktreeInfo) IsMainWorktree() bool {
	return w.IsBare || w.Name() == "."
}
