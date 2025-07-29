package remove

import (
	"fmt"
	"strings"
)

const (
	// DefaultStaleDaysThreshold is the default number of days after which a worktree is considered stale
	DefaultStaleDaysThreshold = 30

	// MinimumStaleDays is the minimum number of days for stale operations
	MinimumStaleDays = 1
)

// RemoveOptions configures behavior for worktree removal operations.
type RemoveOptions struct {
	// Force skips safety checks and removes worktree regardless of uncommitted changes.
	Force bool

	// DryRun shows what would be removed without performing actual deletion.
	DryRun bool

	// DeleteBranch removes the associated branch along with the worktree.
	DeleteBranch bool

	// Days specifies minimum age for stale worktree operations.
	Days int
}

// Validate checks that RemoveOptions contains valid configuration.
func (o RemoveOptions) Validate() error {
	if o.Days < 0 {
		return fmt.Errorf("days must be non-negative, got %d", o.Days)
	}
	if o.Force && o.DryRun {
		return fmt.Errorf("--force and --dry-run are mutually exclusive")
	}
	return nil
}

// BulkCriteria specifies which worktrees to target for bulk removal operations.
type BulkCriteria struct {
	// Merged removes worktrees with branches merged into the default branch.
	Merged bool

	// Stale removes worktrees with no recent activity.
	Stale bool

	// All removes all non-current worktrees.
	All bool

	// DaysOld specifies minimum age in days for stale operations.
	DaysOld int
}

// Validate checks that BulkCriteria contains valid configuration.
func (c BulkCriteria) Validate() error {
	criteriaCount := 0
	if c.Merged {
		criteriaCount++
	}
	if c.Stale {
		criteriaCount++
	}
	if c.All {
		criteriaCount++
	}

	if criteriaCount == 0 {
		return fmt.Errorf("must specify at least one bulk criteria: --merged, --stale, or --all")
	}
	if criteriaCount > 1 {
		return fmt.Errorf("only one bulk criteria can be specified at a time")
	}

	// Only validate DaysOld when Stale is specified
	if c.Stale && c.DaysOld < MinimumStaleDays {
		return fmt.Errorf("days must be at least %d for stale operations, got %d", MinimumStaleDays, c.DaysOld)
	}

	return nil
}

// IsEmpty returns true if no bulk criteria are specified.
func (c BulkCriteria) IsEmpty() bool {
	return !c.Merged && !c.Stale && !c.All
}

// SafetyReport contains validation results for worktree removal safety.
type SafetyReport struct {
	// Path is the worktree path being validated.
	Path string

	// HasUncommitted indicates if the worktree has uncommitted changes.
	HasUncommitted bool

	// IsCurrent indicates if this is the currently active worktree.
	IsCurrent bool

	// BranchStatus contains branch-specific safety information.
	BranchStatus BranchSafetyStatus

	// Warnings contains user-actionable warning messages.
	Warnings []string

	// CanRemoveSafely indicates if removal can proceed without force flag.
	CanRemoveSafely bool
}

// AddWarning adds a warning message to the safety report.
func (r *SafetyReport) AddWarning(message string) {
	r.Warnings = append(r.Warnings, message)
}

// HasWarnings returns true if the report contains any warnings.
func (r *SafetyReport) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// WarningsText returns all warnings as a formatted string.
func (r *SafetyReport) WarningsText() string {
	if len(r.Warnings) == 0 {
		return ""
	}
	return strings.Join(r.Warnings, "\n")
}

// BranchSafetyStatus contains information about branch deletion safety.
type BranchSafetyStatus struct {
	// BranchName is the name of the branch associated with the worktree.
	BranchName string

	// IsMerged indicates if the branch has been merged into the default branch.
	IsMerged bool

	// IsPushedToRemote indicates if the branch exists on the remote repository.
	IsPushedToRemote bool

	// CanDeleteAuto indicates if the branch can be automatically deleted.
	CanDeleteAuto bool

	// RequiresConfirm indicates if user confirmation is required for branch deletion.
	RequiresConfirm bool

	// Reason provides explanation for the branch safety determination.
	Reason string
}

// IsEmpty returns true if no branch information is available.
func (b *BranchSafetyStatus) IsEmpty() bool {
	return b.BranchName == ""
}

// RemoveResults contains the results of bulk removal operations.
type RemoveResults struct {
	// Removed contains paths of successfully removed worktrees.
	Removed []string

	// Skipped contains worktrees that were skipped with reasons.
	Skipped []RemoveSkip

	// Failed contains worktrees that failed to remove with error details.
	Failed []RemoveFailure

	// Summary provides high-level operation statistics.
	Summary RemoveSummary
}

// HasResults returns true if any operation results are present.
func (r *RemoveResults) HasResults() bool {
	return len(r.Removed) > 0 || len(r.Skipped) > 0 || len(r.Failed) > 0
}

// TotalProcessed returns the total number of worktrees processed.
func (r *RemoveResults) TotalProcessed() int {
	return len(r.Removed) + len(r.Skipped) + len(r.Failed)
}

// RemoveSkip represents a worktree that was skipped during bulk removal.
type RemoveSkip struct {
	// Path is the worktree path that was skipped.
	Path string

	// Reason explains why the worktree was skipped.
	Reason string

	// BranchName is the associated branch name, if available.
	BranchName string
}

// RemoveFailure represents a worktree that failed to be removed.
type RemoveFailure struct {
	// Path is the worktree path that failed to remove.
	Path string

	// Error is the error that occurred during removal.
	Error error

	// BranchName is the associated branch name, if available.
	BranchName string

	// PartialSuccess indicates if some cleanup was completed despite the failure.
	PartialSuccess bool
}

// RemoveSummary provides high-level statistics for bulk operations.
type RemoveSummary struct {
	// Total is the number of worktrees considered for removal.
	Total int

	// Removed is the number of worktrees successfully removed.
	Removed int

	// Skipped is the number of worktrees skipped due to safety checks.
	Skipped int

	// Failed is the number of worktrees that failed to remove.
	Failed int

	// BranchesDeleted is the number of branches deleted along with worktrees.
	BranchesDeleted int

	// Duration is the total time taken for the operation.
	Duration string
}

// SuccessRate returns the percentage of successful removals.
func (s *RemoveSummary) SuccessRate() float64 {
	if s.Total == 0 {
		return 0
	}
	return float64(s.Removed) / float64(s.Total) * 100
}

// DryRunResult contains the results of dry-run operations.
type DryRunResult struct {
	// WorktreePath is the path that would be removed.
	WorktreePath string

	// BranchName is the associated branch name, if available.
	BranchName string

	// WouldDeleteBranch indicates if the branch would also be deleted.
	WouldDeleteBranch bool

	// BranchDeletionReason explains why the branch would or would not be deleted.
	BranchDeletionReason string
}
