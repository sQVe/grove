package completion

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/logger"
)

// BranchCompletion provides completion for branch names.
func BranchCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("branch_completion")

	// Check if we're in a repository
	if !ctx.IsInGroveRepo() {
		log.Debug("not in grove repository, skipping branch completion")
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// Get branch names with timeout
	branches, err := ctx.WithTimeout(func() ([]string, error) {
		return getBranchNames(ctx)
	})
	if err != nil {
		log.Debug("failed to get branch names", "error", err)
		return nil, cobra.ShellCompDirectiveError
	}

	// Filter branches based on current input
	filtered := FilterCompletions(branches, toComplete)

	log.Debug("branch completion results", "total", len(branches), "filtered", len(filtered), "input", toComplete)
	return filtered, cobra.ShellCompDirectiveNoFileComp
}

// getBranchNames retrieves available branch names from the repository.
func getBranchNames(ctx *CompletionContext) ([]string, error) {
	log := logger.WithComponent("branch_completion")

	// Check cache first
	if branches, exists := GetCachedBranches(ctx); exists {
		log.Debug("using cached branch names", "count", len(branches))
		return branches, nil
	}

	// Get local branches
	localBranches, localErr := getLocalBranches(ctx)
	if localErr != nil {
		log.Debug("failed to get local branches", "error", localErr)
		// Continue without local branches
	}

	// Get remote branches (only if network is available)
	var remoteBranches []string
	var remoteErr error
	if ctx.IsNetworkOperationAllowed() {
		remoteBranches, remoteErr = getRemoteBranches(ctx)
		if remoteErr != nil {
			log.Debug("failed to get remote branches", "error", remoteErr)
			// Continue without remote branches
		}
	} else {
		log.Debug("skipping remote branch fetch due to network unavailability")
		remoteErr = nil // Don't treat offline as an error
	}

	// If both operations failed, return error
	if localErr != nil && remoteErr != nil {
		log.Debug("both local and remote branch operations failed")
		return nil, localErr
	}

	// Combine and deduplicate branches
	branchSet := make(map[string]bool)
	var allBranches []string

	// Add local branches
	for _, branch := range localBranches {
		if !branchSet[branch] {
			branchSet[branch] = true
			allBranches = append(allBranches, branch)
		}
	}

	// Add remote branches (without remote prefix)
	for _, branch := range remoteBranches {
		if !branchSet[branch] {
			branchSet[branch] = true
			allBranches = append(allBranches, branch)
		}
	}

	log.Debug("collected branch names", "local", len(localBranches), "remote", len(remoteBranches), "total", len(allBranches))

	// Apply priority ordering
	prioritizedBranches := prioritizeBranches(allBranches)

	// Cache the result
	SetCachedBranches(ctx, prioritizedBranches)

	return prioritizedBranches, nil
}

// getLocalBranches retrieves local branch names.
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

// getRemoteBranches retrieves remote branch names.
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
			// Remove remote prefix (e.g., "origin/main" -> "main")
			if parts := strings.Split(branch, "/"); len(parts) > 1 {
				remoteBranch := strings.Join(parts[1:], "/")
				branches = append(branches, remoteBranch)
			}
		}
	}

	return branches, nil
}

// ParseBranchList parses a comma-separated list of branch names for completion.
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

// GetLastBranchInList returns the last branch in a comma-separated list
// This is useful for completing the current branch being typed.
func GetLastBranchInList(branchStr string) string {
	branches := ParseBranchList(branchStr)
	if len(branches) == 0 {
		return ""
	}
	return branches[len(branches)-1]
}

// CompleteBranchList provides completion for comma-separated branch lists.
func CompleteBranchList(ctx *CompletionContext, currentInput, toComplete string) ([]string, error) {
	log := logger.WithComponent("branch_list_completion")

	// Parse the current input to understand the context
	branches := ParseBranchList(currentInput)

	// Get all available branches
	allBranches, err := getBranchNames(ctx)
	if err != nil {
		log.Debug("failed to get branch names for list completion", "error", err)
		return nil, err
	}

	// Filter out branches that are already in the list
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

	// Filter based on what's being typed
	return FilterCompletions(availableBranches, toComplete), nil
}

// prioritizeBranches orders branches with main/master/develop first, then alphabetically.
func prioritizeBranches(branches []string) []string {
	if len(branches) == 0 {
		return branches
	}

	// Define priority branches in order of preference
	priorityBranches := []string{"main", "master", "develop", "development"}

	var prioritized []string
	var regular []string
	prioritySet := make(map[string]bool)

	// First, collect priority branches that exist
	for _, priority := range priorityBranches {
		for _, branch := range branches {
			if branch == priority {
				prioritized = append(prioritized, branch)
				prioritySet[branch] = true
				break
			}
		}
	}

	// Then, collect all other branches alphabetically
	for _, branch := range branches {
		if !prioritySet[branch] {
			regular = append(regular, branch)
		}
	}

	// Sort regular branches alphabetically
	for i := 0; i < len(regular)-1; i++ {
		for j := i + 1; j < len(regular); j++ {
			if regular[i] > regular[j] {
				regular[i], regular[j] = regular[j], regular[i]
			}
		}
	}

	// Combine priority branches first, then regular branches
	prioritized = append(prioritized, regular...)
	return prioritized
}
