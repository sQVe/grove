package remove

import (
	"fmt"
	"strings"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/validation"
)

// CommonDefaultBranches represents typical default branch names in order of preference.
var CommonDefaultBranches = []string{"main", "master", "develop", "dev"}

// BranchManagerImpl implements the BranchManager interface with intelligent branch deletion logic.
type BranchManagerImpl struct {
	executor git.GitExecutor
	logger   *logger.Logger
}

// NewBranchManager creates a new BranchManager instance.
func NewBranchManager(executor git.GitExecutor, log *logger.Logger) BranchManager {
	return &BranchManagerImpl{
		executor: executor,
		logger:   log,
	}
}

// DeleteBranchSafely removes a branch with comprehensive safety checks.
func (b *BranchManagerImpl) DeleteBranchSafely(branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Validate branch name for security.
	if err := validation.ValidateGitBranchName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	b.logger.DebugOperation("starting safe branch deletion",
		"branch", branchName)

	// Check if branch exists locally.
	exists, err := b.branchExists(branchName)
	if err != nil {
		return fmt.Errorf("failed to check if branch exists: %w", err)
	}
	if !exists {
		b.logger.Debug("branch does not exist locally, skipping deletion",
			"branch", branchName)
		return nil
	}

	// Check if we're currently on this branch
	isCurrentBranch, err := b.isCurrentBranch(branchName)
	if err != nil {
		return fmt.Errorf("failed to check current branch: %w", err)
	}
	if isCurrentBranch {
		return fmt.Errorf("cannot delete branch %s: it is currently checked out", branchName)
	}

	// Determine if branch can be deleted automatically
	canDelete, reason := b.CanDeleteBranchAutomatically(branchName)
	if !canDelete {
		return fmt.Errorf("cannot automatically delete branch %s: %s", branchName, reason)
	}

	// Attempt to delete the branch
	err = b.deleteBranchForced(branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	b.logger.Debug("successfully deleted branch",
		"branch", branchName,
		"reason", reason)

	return nil
}

// CanDeleteBranchAutomatically determines if branch can be auto-deleted with reasoning.
func (b *BranchManagerImpl) CanDeleteBranchAutomatically(branchName string) (canDelete bool, reason string) {
	if branchName == "" {
		return false, "branch name is empty"
	}

	// Validate branch name for security.
	if err := validation.ValidateGitBranchName(branchName); err != nil {
		return false, fmt.Sprintf("invalid branch name: %v", err)
	}

	// Check if branch is merged into default branch
	isMerged, err := b.isBranchMerged(branchName)
	if err != nil {
		b.logger.Debug("failed to check if branch is merged",
			"branch", branchName,
			"error", err.Error())
		// Continue with other checks
	} else if isMerged {
		return true, "branch is merged into default branch"
	}

	// Check if branch is pushed to remote and we have all commits locally
	isPushed, err := b.isBranchPushedToRemote(branchName)
	if err != nil {
		b.logger.Debug("failed to check remote branch status",
			"branch", branchName,
			"error", err.Error())
		// Continue with other checks
	} else if isPushed {
		// If pushed to remote, it's safe to delete locally as commits are preserved
		return true, "branch is pushed to remote repository"
	}

	// Check if it's a tracking branch for a merged remote branch
	isTrackingMerged, err := b.isTrackingMergedRemoteBranch(branchName)
	if err != nil {
		b.logger.Debug("failed to check tracking branch status",
			"branch", branchName,
			"error", err.Error())
	} else if isTrackingMerged {
		return true, "local branch tracks a merged remote branch"
	}

	return false, "branch is not merged and may contain unique changes"
}

// DeleteRemoteBranch removes the remote tracking branch.
func (b *BranchManagerImpl) DeleteRemoteBranch(branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	// Validate branch name for security.
	if err := validation.ValidateGitBranchName(branchName); err != nil {
		return fmt.Errorf("invalid branch name: %w", err)
	}

	b.logger.Debug("deleting remote branch",
		"branch", branchName)

	// Check if remote branch exists
	remoteExists, err := b.remoteBranchExists(branchName)
	if err != nil {
		return fmt.Errorf("failed to check remote branch existence: %w", err)
	}
	if !remoteExists {
		b.logger.Debug("remote branch does not exist, skipping deletion",
			"branch", branchName)
		return nil
	}

	// Delete remote branch
	_, err = b.executor.Execute("push", "origin", "--delete", branchName)
	if err != nil {
		return fmt.Errorf("failed to delete remote branch %s: %w", branchName, err)
	}

	b.logger.Debug("successfully deleted remote branch",
		"branch", branchName)

	return nil
}

// branchExists checks if a local branch exists.
func (b *BranchManagerImpl) branchExists(branchName string) (bool, error) {
	output, err := b.executor.ExecuteQuiet("branch", "--list", branchName)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(output) != "", nil
}

// remoteBranchExists checks if a remote branch exists.
func (b *BranchManagerImpl) remoteBranchExists(branchName string) (bool, error) {
	output, err := b.executor.ExecuteQuiet("branch", "-r", "--list", "origin/"+branchName)
	if err != nil {
		return false, nil
	}
	return strings.TrimSpace(output) != "", nil
}

// isCurrentBranch checks if the given branch is currently checked out.
func (b *BranchManagerImpl) isCurrentBranch(branchName string) (bool, error) {
	output, err := b.executor.ExecuteQuiet("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return false, fmt.Errorf("failed to get current branch: %w", err)
	}

	currentBranch := strings.TrimSpace(output)
	return currentBranch == branchName, nil
}

// isBranchMerged checks if a branch has been merged into the default branch.
func (b *BranchManagerImpl) isBranchMerged(branchName string) (bool, error) {
	// Get default branch name
	defaultBranch, err := b.getDefaultBranch()
	if err != nil {
		return false, fmt.Errorf("failed to determine default branch: %w", err)
	}

	// Use git merge-base to check if branch is ancestor of default branch
	_, err = b.executor.ExecuteQuiet("merge-base", "--is-ancestor", branchName, defaultBranch)
	if err != nil {
		// If git merge-base fails, the branch likely isn't merged
		return false, nil
	}

	// Also check that the branch tip is reachable from default branch
	output, err := b.executor.ExecuteQuiet("merge-base", branchName, defaultBranch)
	if err != nil {
		return false, nil
	}

	mergeBase := strings.TrimSpace(output)

	// Get the commit hash of the branch
	branchCommit, err := b.executor.ExecuteQuiet("rev-parse", branchName)
	if err != nil {
		return false, nil
	}

	branchCommit = strings.TrimSpace(branchCommit)

	// Branch is merged if its commit is the merge base (i.e., all commits are in default branch)
	return mergeBase == branchCommit, nil
}

// isBranchPushedToRemote checks if a branch exists on the remote repository.
func (b *BranchManagerImpl) isBranchPushedToRemote(branchName string) (bool, error) {
	return b.remoteBranchExists(branchName)
}

// isTrackingMergedRemoteBranch checks if local branch tracks a remote branch that's been merged.
func (b *BranchManagerImpl) isTrackingMergedRemoteBranch(branchName string) (bool, error) {
	// Get upstream branch if it exists
	upstream, err := b.getUpstreamBranch(branchName)
	if err != nil || upstream == "" {
		return false, nil
	}

	// Check if the upstream branch is merged
	return b.isBranchMerged(upstream)
}

// getUpstreamBranch gets the upstream tracking branch for a local branch.
func (b *BranchManagerImpl) getUpstreamBranch(branchName string) (string, error) {
	output, err := b.executor.ExecuteQuiet("rev-parse", "--abbrev-ref", branchName+"@{upstream}")
	if err != nil {
		return "", nil // No upstream branch
	}
	return strings.TrimSpace(output), nil
}

// getDefaultBranch determines the default branch name for the repository.
func (b *BranchManagerImpl) getDefaultBranch() (string, error) {
	// Try to get default branch from remote HEAD
	output, err := b.executor.ExecuteQuiet("symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		parts := strings.Split(strings.TrimSpace(output), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Try to get from remote's default branch
	output, err = b.executor.ExecuteQuiet("remote", "show", "origin")
	if err == nil {
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "HEAD branch:") {
				parts := strings.Fields(line)
				if len(parts) >= 3 {
					return parts[2], nil
				}
			}
		}
	}

	// Fallback to common default branch names
	for _, defaultName := range CommonDefaultBranches {
		_, err := b.executor.ExecuteQuiet("rev-parse", "--verify", defaultName)
		if err == nil {
			return defaultName, nil
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}

// deleteBranchForced forcefully deletes a local branch.
func (b *BranchManagerImpl) deleteBranchForced(branchName string) error {
	// Use -D flag for forced deletion to handle unmerged branches
	_, err := b.executor.Execute("branch", "-D", branchName)
	return err
}
