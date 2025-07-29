package remove

// RemoveService defines the interface for worktree removal operations.
type RemoveService interface {
	// RemoveWorktree removes a single worktree with safety validation.
	RemoveWorktree(path string, options RemoveOptions) error

	// RemoveBulk removes multiple worktrees based on bulk criteria.
	RemoveBulk(criteria BulkCriteria, options RemoveOptions) (RemoveResults, error)

	// ValidateRemoval checks if a worktree can be safely removed.
	ValidateRemoval(path string) (SafetyReport, error)
}

// SafetyChecker defines the interface for safety validation operations.
type SafetyChecker interface {
	// CheckUncommittedChanges validates if worktree has uncommitted changes.
	CheckUncommittedChanges(path string) (bool, error)

	// CheckBranchSafety validates if a branch can be safely deleted.
	CheckBranchSafety(branch string) (BranchSafetyStatus, error)

	// CheckCurrentWorktree validates if worktree is currently active.
	CheckCurrentWorktree(path string) (bool, error)
}

// BranchManager defines the interface for branch cleanup operations.
type BranchManager interface {
	// DeleteBranchSafely removes a branch with safety checks.
	DeleteBranchSafely(branchName string) error

	// CanDeleteBranchAutomatically determines if branch can be auto-deleted.
	CanDeleteBranchAutomatically(branchName string) (bool, string)

	// DeleteRemoteBranch removes the remote tracking branch.
	DeleteRemoteBranch(branchName string) error
}
