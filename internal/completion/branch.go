package completion

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/logger"
)

func BranchCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("branch_completion")

	if !ctx.IsInGroveRepo() {
		log.Debug("not in grove repository, skipping branch completion")
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	branches, err := ctx.WithTimeout(func() ([]string, error) {
		return getBranchNames(ctx)
	})
	if err != nil {
		log.Debug("failed to get branch names", "error", err)
		return nil, cobra.ShellCompDirectiveError
	}

	filtered := FilterCompletions(branches, toComplete)

	log.Debug("branch completion results", "total", len(branches), "filtered", len(filtered), "input", toComplete)
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

func getBranchNames(ctx *CompletionContext) ([]string, error) {
	log := logger.WithComponent("branch_completion")

	if branches, exists := GetCachedBranches(ctx); exists {
		log.Debug("using cached branch names", "count", len(branches))
		return branches, nil
	}

	localBranches, localErr := getLocalBranches(ctx)
	if localErr != nil {
		log.Debug("failed to get local branches", "error", localErr)
	}

	// Get remote branches (only if network is available).
	var remoteBranches []string
	var remoteErr error
	if ctx.IsNetworkOperationAllowed() {
		remoteBranches, remoteErr = getRemoteBranches(ctx)
		if remoteErr != nil {
			log.Debug("failed to get remote branches", "error", remoteErr)
		}
	} else {
		log.Debug("skipping remote branch fetch due to network unavailability")
		remoteErr = nil // Don't treat offline as an error
	}

	if localErr != nil && remoteErr != nil {
		log.Debug("both local and remote branch operations failed")
		return nil, localErr
	}

	branchSet := make(map[string]bool)
	var allBranches []string

	for _, branch := range localBranches {
		if !branchSet[branch] {
			branchSet[branch] = true
			allBranches = append(allBranches, branch)
		}
	}

	// Add remote branches (without remote prefix).
	for _, branch := range remoteBranches {
		if !branchSet[branch] {
			branchSet[branch] = true
			allBranches = append(allBranches, branch)
		}
	}

	log.Debug("collected branch names", "local", len(localBranches), "remote", len(remoteBranches), "total", len(allBranches))

	prioritizedBranches := prioritizeBranches(allBranches)

	SetCachedBranches(ctx, prioritizedBranches)

	return prioritizedBranches, nil
}

func getLocalBranches(ctx *CompletionContext) ([]string, error) {
	output, err := ctx.Executor.Execute("branch", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var branches []string

	for _, line := range lines {
		branch := strings.TrimSpace(line)
		if branch != "" {
			branches = append(branches, branch)
		}
	}

	return branches, nil
}

func getRemoteBranches(ctx *CompletionContext) ([]string, error) {
	output, err := ctx.Executor.Execute("branch", "-r", "--format=%(refname:short)")
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var branches []string

	for _, line := range lines {
		branch := strings.TrimSpace(line)
		if branch != "" && !strings.Contains(branch, "->") {
			// Remove remote prefix (e.g., "origin/main" -> "main").
			if parts := strings.Split(branch, "/"); len(parts) > 1 {
				remoteBranch := strings.Join(parts[1:], "/")
				branches = append(branches, remoteBranch)
			}
		}
	}

	return branches, nil
}

func ParseBranchList(branchStr string) []string {
	if branchStr == "" {
		return nil
	}

	var branches []string
	parts := strings.Split(branchStr, ",")

	for _, part := range parts {
		branch := strings.TrimSpace(part)
		if branch != "" {
			branches = append(branches, branch)
		}
	}

	return branches
}

// This is useful for completing the current branch being typed.
func GetLastBranchInList(branchStr string) string {
	branches := ParseBranchList(branchStr)
	if len(branches) == 0 {
		return ""
	}
	return branches[len(branches)-1]
}

func CompleteBranchList(ctx *CompletionContext, currentInput, toComplete string) ([]string, error) {
	log := logger.WithComponent("branch_list_completion")

	branches := ParseBranchList(currentInput)

	allBranches, err := getBranchNames(ctx)
	if err != nil {
		log.Debug("failed to get branch names for list completion", "error", err)
		return nil, err
	}

	// Filter out branches that are already in the list.
	branchSet := make(map[string]bool)
	for _, branch := range branches {
		branchSet[branch] = true
	}

	var availableBranches []string
	for _, branch := range allBranches {
		if !branchSet[branch] {
			availableBranches = append(availableBranches, branch)
		}
	}

	return FilterCompletions(availableBranches, toComplete), nil
}

func prioritizeBranches(branches []string) []string {
	if len(branches) == 0 {
		return branches
	}

	// Define priority branches in order of preference.
	priorityBranches := []string{"main", "master", "develop", "development"}

	var prioritized []string
	var regular []string
	prioritySet := make(map[string]bool)

	for _, priority := range priorityBranches {
		for _, branch := range branches {
			if branch == priority {
				prioritized = append(prioritized, branch)
				prioritySet[branch] = true
				break
			}
		}
	}

	for _, branch := range branches {
		if !prioritySet[branch] {
			regular = append(regular, branch)
		}
	}

	for i := 0; i < len(regular)-1; i++ {
		for j := i + 1; j < len(regular); j++ {
			if regular[i] > regular[j] {
				regular[i], regular[j] = regular[j], regular[i]
			}
		}
	}

	prioritized = append(prioritized, regular...)
	return prioritized
}
