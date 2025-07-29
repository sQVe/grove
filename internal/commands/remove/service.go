package remove

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/validation"
)

// RemoveServiceImpl implements the RemoveService interface with comprehensive worktree removal capabilities.
type RemoveServiceImpl struct {
	executor      git.GitExecutor
	logger        *logger.Logger
	safetyChecker SafetyChecker
	branchManager BranchManager
}

// NewRemoveServiceImpl creates a new RemoveService instance with all necessary dependencies.
func NewRemoveServiceImpl(executor git.GitExecutor, log *logger.Logger) RemoveService {
	return &RemoveServiceImpl{
		executor:      executor,
		logger:        log,
		safetyChecker: NewSafetyChecker(executor, log),
		branchManager: NewBranchManager(executor, log),
	}
}

// RemoveWorktree removes a single worktree with comprehensive safety validation.
func (s *RemoveServiceImpl) RemoveWorktree(path string, options RemoveOptions) error {
	if path == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	// Validate path for security (prevent path traversal attacks)
	if err := validation.ValidatePath(path); err != nil {
		return fmt.Errorf("invalid worktree path: %w", err)
	}

	// Validate options
	if err := options.Validate(); err != nil {
		return fmt.Errorf("invalid options: %w", err)
	}

	s.logger.DebugOperation("starting worktree removal",
		"path", path,
		"force", options.Force,
		"dry_run", options.DryRun,
		"delete_branch", options.DeleteBranch)

	// Clean the path
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to resolve worktree path: %w", err)
	}

	// Validate removal safety unless forced
	if !options.Force {
		report, err := s.ValidateRemoval(cleanPath)
		if err != nil {
			return fmt.Errorf("failed to validate removal: %w", err)
		}

		if !report.CanRemoveSafely {
			var errorParts []string
			errorParts = append(errorParts, fmt.Sprintf("cannot safely remove worktree at %s", cleanPath))

			if report.HasWarnings() {
				errorParts = append(errorParts, "Issues found:", report.WarningsText())
			}

			errorParts = append(errorParts, "Use --force to override safety checks (with confirmation)")
			return fmt.Errorf("%s", strings.Join(errorParts, "\n"))
		}
	}

	// Handle dry-run mode
	if options.DryRun {
		return s.handleDryRun(cleanPath, options)
	}

	// Get worktree info before removal for branch operations
	worktreeInfo, err := s.getWorktreeInfo(cleanPath)
	if err != nil {
		s.logger.DebugOperation("failed to get worktree info before removal",
			"path", cleanPath,
			"error", err.Error())
		// Continue with removal but branch deletion won't be possible
		if options.DeleteBranch {
			s.logger.DebugOperation("branch deletion requested but worktree info unavailable",
				"path", cleanPath)
		}
	}

	// Perform the actual removal
	err = s.performRemoval(cleanPath)
	if err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	// Handle branch deletion if requested
	if options.DeleteBranch && worktreeInfo != nil && worktreeInfo.Branch != "" {
		err = s.handleBranchDeletion(worktreeInfo.Branch, options.Force)
		if err != nil {
			s.logger.DebugOperation("worktree removed but branch deletion failed",
				"path", cleanPath,
				"branch", worktreeInfo.Branch,
				"error", err.Error())
			// Don't fail the entire operation if branch deletion fails
			// Return a warning message that the caller can display
			return fmt.Errorf("worktree removed successfully, but branch deletion failed: %w", err)
		}
	}

	s.logger.DebugOperation("worktree removal completed successfully",
		"path", cleanPath,
		"branch_deleted", options.DeleteBranch && worktreeInfo != nil)

	return nil
}

// RemoveBulk removes multiple worktrees based on bulk criteria.
func (s *RemoveServiceImpl) RemoveBulk(criteria BulkCriteria, options RemoveOptions) (RemoveResults, error) {
	// Validate criteria
	if err := criteria.Validate(); err != nil {
		return RemoveResults{}, fmt.Errorf("invalid bulk criteria: %w", err)
	}

	// Validate options
	if err := options.Validate(); err != nil {
		return RemoveResults{}, fmt.Errorf("invalid options: %w", err)
	}

	s.logger.DebugOperation("starting bulk worktree removal",
		"merged", criteria.Merged,
		"stale", criteria.Stale,
		"all", criteria.All,
		"days_old", criteria.DaysOld)

	// Get list of all worktrees
	allWorktrees, err := git.ListWorktrees(s.executor)
	if err != nil {
		return RemoveResults{}, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Filter worktrees based on criteria
	candidates := s.filterWorktreesByCriteria(allWorktrees, criteria)

	s.logger.DebugOperation("identified candidate worktrees for bulk removal",
		"total_worktrees", len(allWorktrees),
		"candidates", len(candidates))

	if len(candidates) == 0 {
		return RemoveResults{
			Summary: RemoveSummary{
				Total: len(allWorktrees),
			},
		}, nil
	}

	// Handle dry-run mode for bulk operations
	if options.DryRun {
		return s.handleBulkDryRun(candidates, options), nil
	}

	// Perform bulk removal
	results := s.performBulkRemoval(candidates, options)

	s.logger.DebugOperation("bulk removal completed",
		"total_processed", results.TotalProcessed(),
		"removed", len(results.Removed),
		"skipped", len(results.Skipped),
		"failed", len(results.Failed))

	return results, nil
}

// ValidateRemoval checks if a worktree can be safely removed.
func (s *RemoveServiceImpl) ValidateRemoval(path string) (SafetyReport, error) {
	report := SafetyReport{
		Path:            path,
		CanRemoveSafely: true,
	}

	// Check if path exists
	exists, err := s.pathExists(path)
	if err != nil {
		return report, fmt.Errorf("failed to check path existence: %w", err)
	}
	if !exists {
		report.AddWarning(fmt.Sprintf("Path does not exist: %s", path))
		report.CanRemoveSafely = false
		return report, nil
	}

	// Check if worktree is currently active
	isCurrent, err := s.safetyChecker.CheckCurrentWorktree(path)
	if err != nil {
		s.logger.DebugOperation("failed to check current worktree status",
			"path", path,
			"error", err.Error())
		report.AddWarning("Could not verify if worktree is currently active - proceeding with caution")
		// Continue with other checks but mark as unsafe due to uncertainty
		report.CanRemoveSafely = false
	} else {
		report.IsCurrent = isCurrent
		if isCurrent {
			report.AddWarning("Cannot remove currently active worktree")
			report.CanRemoveSafely = false
		}
	}

	// Check for uncommitted changes
	hasUncommitted, err := s.safetyChecker.CheckUncommittedChanges(path)
	if err != nil {
		s.logger.DebugOperation("failed to check uncommitted changes",
			"path", path,
			"error", err.Error())
		report.AddWarning("Could not verify uncommitted changes - proceeding with caution")
		// Continue with other checks but mark as unsafe due to uncertainty
		report.CanRemoveSafely = false
	} else {
		report.HasUncommitted = hasUncommitted
		if hasUncommitted {
			report.AddWarning("Worktree has uncommitted changes that will be lost")
			report.CanRemoveSafely = false
		}
	}

	// Get branch safety information
	worktreeInfo, err := s.getWorktreeInfo(path)
	if err == nil && worktreeInfo != nil && worktreeInfo.Branch != "" {
		branchStatus, err := s.safetyChecker.CheckBranchSafety(worktreeInfo.Branch)
		if err == nil {
			report.BranchStatus = branchStatus
		}
	}

	s.logger.DebugOperation("completed safety validation",
		"path", path,
		"can_remove_safely", report.CanRemoveSafely,
		"has_uncommitted", report.HasUncommitted,
		"is_current", report.IsCurrent,
		"warnings_count", len(report.Warnings))

	return report, nil
}

// performRemoval executes the actual worktree removal using the existing git function.
func (s *RemoveServiceImpl) performRemoval(path string) error {
	return git.RemoveWorktree(s.executor, path)
}

// handleDryRun provides detailed preview of what would be removed.
func (s *RemoveServiceImpl) handleDryRun(path string, options RemoveOptions) error {
	result := DryRunResult{
		WorktreePath: path,
	}

	// Check what would happen with branch deletion
	if options.DeleteBranch {
		worktreeInfo, err := s.getWorktreeInfo(path)
		if err == nil && worktreeInfo != nil && worktreeInfo.Branch != "" {
			result.BranchName = worktreeInfo.Branch
			canDelete, reason := s.branchManager.CanDeleteBranchAutomatically(worktreeInfo.Branch)
			result.WouldDeleteBranch = canDelete
			result.BranchDeletionReason = reason
		}
	}

	s.logger.DebugOperation("dry-run completed for single worktree",
		"path", result.WorktreePath,
		"branch", result.BranchName,
		"would_delete_branch", result.WouldDeleteBranch,
		"reason", result.BranchDeletionReason)

	// For now, print directly to maintain current CLI behavior
	// TODO: Return DryRunResult struct to command layer and remove fmt.Printf calls from service layer
	fmt.Printf("DRY RUN: Would remove worktree at: %s\n", result.WorktreePath)
	if options.DeleteBranch && result.BranchName != "" {
		if result.WouldDeleteBranch {
			fmt.Printf("DRY RUN: Would also delete branch: %s (%s)\n", result.BranchName, result.BranchDeletionReason)
		} else {
			fmt.Printf("DRY RUN: Would NOT delete branch: %s (%s)\n", result.BranchName, result.BranchDeletionReason)
		}
	}

	return nil
}

// handleBranchDeletion manages branch cleanup after worktree removal.
func (s *RemoveServiceImpl) handleBranchDeletion(branchName string, force bool) error {
	if !force {
		// Check if branch can be automatically deleted
		canDelete, reason := s.branchManager.CanDeleteBranchAutomatically(branchName)
		if !canDelete {
			return fmt.Errorf("cannot automatically delete branch %s: %s. Use --force for confirmation prompt", branchName, reason)
		}
	}

	// Attempt to delete the branch
	err := s.branchManager.DeleteBranchSafely(branchName)
	if err != nil {
		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)
	}

	s.logger.DebugOperation("successfully deleted branch", "branch", branchName)
	return nil
}

// getWorktreeInfo retrieves information about a specific worktree.
func (s *RemoveServiceImpl) getWorktreeInfo(path string) (*git.WorktreeInfo, error) {
	worktrees, err := git.ListWorktrees(s.executor)
	if err != nil {
		return nil, err
	}

	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	for i := range worktrees {
		wtCleanPath, err := filepath.Abs(worktrees[i].Path)
		if err != nil {
			continue
		}
		if wtCleanPath == cleanPath {
			return &worktrees[i], nil
		}
	}

	return nil, fmt.Errorf("worktree not found: %s", path)
}

// pathExists checks if a path exists on the filesystem.
func (s *RemoveServiceImpl) pathExists(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("path cannot be empty")
	}

	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check path existence: %w", err)
	}
	return true, nil
}

// filterWorktreesByCriteria filters worktrees based on bulk removal criteria.
func (s *RemoveServiceImpl) filterWorktreesByCriteria(worktrees []git.WorktreeInfo, criteria BulkCriteria) []git.WorktreeInfo {
	var candidates []git.WorktreeInfo

	for i := range worktrees {
		// Skip current worktree
		if worktrees[i].IsCurrent {
			continue
		}

		shouldInclude := false

		if criteria.Merged && worktrees[i].Remote.IsMerged {
			shouldInclude = true
		}

		if criteria.Stale {
			// Check if worktree is stale based on last activity
			if s.isWorktreeStale(&worktrees[i], criteria.DaysOld) {
				shouldInclude = true
			}
		}

		if criteria.All {
			shouldInclude = true
		}

		if shouldInclude {
			candidates = append(candidates, worktrees[i])
		}
	}

	return candidates
}

// isWorktreeStale checks if a worktree is older than the specified number of days.
func (s *RemoveServiceImpl) isWorktreeStale(wt *git.WorktreeInfo, daysOld int) bool {
	if wt.LastActivity.IsZero() {
		// If no last activity is recorded, consider it stale if it's older than the threshold
		// We can fall back to checking directory modification time if needed
		s.logger.DebugOperation("worktree has no recorded activity, checking directory modification time",
			"path", wt.Path)

		if info, err := os.Stat(wt.Path); err == nil {
			threshold := time.Now().AddDate(0, 0, -daysOld)
			return info.ModTime().Before(threshold)
		}

		// If we can't check anything, assume it's not stale to be safe
		return false
	}

	threshold := time.Now().AddDate(0, 0, -daysOld)
	isStale := wt.LastActivity.Before(threshold)

	s.logger.DebugOperation("checked worktree staleness",
		"path", wt.Path,
		"last_activity", wt.LastActivity.Format(time.RFC3339),
		"threshold", threshold.Format(time.RFC3339),
		"days_old", daysOld,
		"is_stale", isStale)

	return isStale
}

// handleBulkDryRun provides preview for bulk operations.
func (s *RemoveServiceImpl) handleBulkDryRun(candidates []git.WorktreeInfo, options RemoveOptions) RemoveResults {
	s.logger.DebugOperation("dry-run for bulk operation", "candidate_count", len(candidates))

	var removed []string
	for i := range candidates {
		removed = append(removed, candidates[i].Path)
		s.logger.DebugOperation("would remove worktree in bulk dry-run",
			"path", candidates[i].Path,
			"branch", candidates[i].Branch)
	}

	// For now, print directly to maintain current CLI behavior
	// TODO: Return structured bulk dry-run data to command layer and remove fmt.Printf calls from service layer
	fmt.Printf("DRY RUN: Would remove %d worktrees:\n", len(candidates))
	for i := range candidates {
		fmt.Printf("  - %s (branch: %s)\n", candidates[i].Path, candidates[i].Branch)
	}

	return RemoveResults{
		Removed: removed,
		Summary: RemoveSummary{
			Total:   len(candidates),
			Removed: len(candidates),
		},
	}
}

// performBulkRemoval executes bulk worktree removal operations.
func (s *RemoveServiceImpl) performBulkRemoval(candidates []git.WorktreeInfo, options RemoveOptions) RemoveResults {
	results := RemoveResults{}

	for i := range candidates {
		s.processSingleWorktreeInBulk(&candidates[i], options, &results)
	}

	s.updateBulkRemovalSummary(&results, len(candidates))
	return results
}

// processSingleWorktreeInBulk handles removal of a single worktree in bulk operation.
func (s *RemoveServiceImpl) processSingleWorktreeInBulk(wt *git.WorktreeInfo, options RemoveOptions, results *RemoveResults) {
	// Validate safety for each worktree
	if !options.Force {
		if shouldSkip, reason := s.shouldSkipWorktreeRemoval(wt.Path); shouldSkip {
			results.Skipped = append(results.Skipped, RemoveSkip{
				Path:       wt.Path,
				BranchName: wt.Branch,
				Reason:     reason,
			})
			return
		}
	}

	// Attempt worktree removal
	if err := s.performRemoval(wt.Path); err != nil {
		results.Failed = append(results.Failed, RemoveFailure{
			Path:       wt.Path,
			BranchName: wt.Branch,
			Error:      err,
		})
		return
	}

	results.Removed = append(results.Removed, wt.Path)

	// Handle branch deletion if requested
	if options.DeleteBranch && wt.Branch != "" {
		s.handleBranchDeletionInBulk(wt, options.Force, results)
	}
}

// shouldSkipWorktreeRemoval determines if a worktree should be skipped during bulk removal.
func (s *RemoveServiceImpl) shouldSkipWorktreeRemoval(path string) (shouldSkip bool, reason string) {
	report, err := s.ValidateRemoval(path)
	if err != nil {
		return true, fmt.Sprintf("Safety validation failed: %v", err)
	}
	if !report.CanRemoveSafely {
		return true, "Failed comprehensive safety validation"
	}
	return false, ""
}

// handleBranchDeletionInBulk manages branch deletion during bulk operations.
func (s *RemoveServiceImpl) handleBranchDeletionInBulk(wt *git.WorktreeInfo, force bool, results *RemoveResults) {
	err := s.handleBranchDeletion(wt.Branch, force)
	if err != nil {
		s.logger.DebugOperation("branch deletion failed in bulk operation",
			"path", wt.Path,
			"branch", wt.Branch,
			"error", err.Error())
		// Continue with other worktrees - don't fail the bulk operation
	} else {
		results.Summary.BranchesDeleted++
	}
}

// updateBulkRemovalSummary calculates and updates the final summary statistics.
func (s *RemoveServiceImpl) updateBulkRemovalSummary(results *RemoveResults, totalCandidates int) {
	results.Summary = RemoveSummary{
		Total:           totalCandidates,
		Removed:         len(results.Removed),
		Skipped:         len(results.Skipped),
		Failed:          len(results.Failed),
		BranchesDeleted: results.Summary.BranchesDeleted, // Preserve count from branch operations
	}
}
