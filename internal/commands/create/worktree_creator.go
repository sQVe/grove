package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
)

// worktreeOperation encapsulates atomic worktree creation with rollback
type worktreeOperation struct {
	worktree         *WorktreeCreatorImpl
	branchName       string
	path             string
	options          WorktreeOptions
	progressCallback ProgressCallback
	createdDirs      []string // Track created directories for cleanup
	createdFiles     []string // Track created files for cleanup
}

// WorktreeCreatorImpl implements the WorktreeCreator interface.
type WorktreeCreatorImpl struct {
	executor git.GitExecutor
	logger   *logger.Logger
}

// NewWorktreeCreator creates a new WorktreeCreator with the provided GitExecutor.
func NewWorktreeCreator(executor git.GitExecutor) *WorktreeCreatorImpl {
	return &WorktreeCreatorImpl{
		executor: executor,
		logger:   logger.WithComponent("worktree_creator"),
	}
}

// CreateWorktree creates a new Git worktree at the specified path for the given branch.
// It handles both new branch creation and existing branch checkout based on the options.
// Uses atomic operations with comprehensive cleanup on failure.
func (w *WorktreeCreatorImpl) CreateWorktree(branchName, path string, options WorktreeOptions) error {
	return w.CreateWorktreeWithProgress(branchName, path, options, nil)
}

// CreateWorktreeWithProgress creates a new Git worktree with progress reporting.
// It handles both new branch creation and existing branch checkout based on the options.
// Uses atomic operations with comprehensive cleanup on failure.
func (w *WorktreeCreatorImpl) CreateWorktreeWithProgress(branchName, path string, options WorktreeOptions, progressCallback ProgressCallback) error {
	if branchName == "" {
		return groveErrors.ErrWorktreeCreation("validation", fmt.Errorf("branch name cannot be empty"))
	}

	if path == "" {
		return groveErrors.ErrWorktreeCreation("validation", fmt.Errorf("worktree path cannot be empty"))
	}

	// Use atomic worktree creation with rollback capability
	operation := &worktreeOperation{
		worktree:         w,
		branchName:       branchName,
		path:             path,
		options:          options,
		progressCallback: progressCallback,
	}

	return operation.execute()
}

// execute performs the atomic worktree creation with comprehensive rollback
func (op *worktreeOperation) execute() error {
	// Ensure parent directory exists
	if err := op.ensureParentDirectory(); err != nil {
		op.rollback()
		return err
	}

	// Check if path already exists
	if _, err := os.Stat(op.path); err == nil {
		return groveErrors.ErrPathExists(op.path)
	}

	// Check if branch exists
	branchExists, err := op.worktree.branchExists(op.branchName)
	if err != nil {
		op.rollback()
		return groveErrors.ErrWorktreeCreation("branch-check", err)
	}

	// Create worktree atomically
	var worktreeErr error
	if branchExists {
		worktreeErr = op.createFromExistingBranch()
	} else {
		worktreeErr = op.createWithNewBranch()
	}

	if worktreeErr != nil {
		op.rollback()
		return worktreeErr
	}

	return nil
}

// ensureParentDirectory creates parent directories if needed
func (op *worktreeOperation) ensureParentDirectory() error {
	parentDir := filepath.Dir(op.path)
	
	// MkdirAll is idempotent - safe to call even if directory exists
	// This eliminates the race condition between stat check and directory creation
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return groveErrors.ErrDirectoryAccess(parentDir, err)
	}

	// Track parent directory for potential rollback
	// Note: We track it even if it already existed, as rollback logic
	// will safely handle attempting to remove existing directories
	op.createdDirs = append(op.createdDirs, parentDir)
	return nil
}

// createFromExistingBranch creates worktree from existing branch with tracking
func (op *worktreeOperation) createFromExistingBranch() error {
	_, err := op.worktree.executor.Execute("worktree", "add", op.path, op.branchName)
	if err != nil {
		return op.handleWorktreeError(err)
	}
	return nil
}

// createWithNewBranch creates worktree with new branch and optional remote tracking
func (op *worktreeOperation) createWithNewBranch() error {
	args := []string{"worktree", "add", "-b", op.branchName, op.path}
	
	// Add base branch if specified
	if op.options.BaseBranch != "" {
		args = append(args, op.options.BaseBranch)
	}

	_, err := op.worktree.executor.Execute(args...)
	if err != nil {
		return op.handleWorktreeError(err)
	}

	if op.options.TrackRemote {
		if err := op.setupRemoteTracking(); err != nil {
			// Remote tracking failure is not critical but should be cleaned up
			return groveErrors.ErrWorktreeCreation("remote-tracking", err)
		}
	}

	return nil
}

// setupRemoteTracking configures remote tracking for the new branch
func (op *worktreeOperation) setupRemoteTracking() error {
	remote, err := op.worktree.executor.ExecuteQuiet("config", "--get", "clone.defaultRemoteName")
	if err != nil || remote == "" {
		remote = "origin" // Fallback to standard default
	}

	remoteBranch := remote + "/" + op.branchName
	
	// Check if the remote branch exists before setting upstream
	output, err := op.worktree.executor.ExecuteQuiet("branch", "-r", "--list", remoteBranch)
	if err != nil || strings.TrimSpace(output) == "" {
		// Remote branch doesn't exist, skip upstream setup
		op.worktree.logger.Debug("skipping upstream setup - remote branch does not exist", 
			"remote_branch", remoteBranch)
		return nil
	}

	_, err = op.worktree.executor.Execute("-C", op.path, "branch", "--set-upstream-to="+remoteBranch, op.branchName)
	if err != nil {
		return groveErrors.ErrWorktreeCreation("set-upstream", err)
	}

	return nil
}

// rollback performs comprehensive cleanup of partial operations
func (op *worktreeOperation) rollback() {
	// Remove Git worktree if it was created
	if _, err := os.Stat(op.path); err == nil {
		// Try git worktree remove first
		if _, gitErr := op.worktree.executor.ExecuteQuiet("worktree", "remove", "--force", op.path); gitErr != nil {
			// If git worktree remove fails, try manual cleanup
			if fsErr := os.RemoveAll(op.path); fsErr != nil {
				// Log rollback failures at debug level to avoid cluttering user output
				op.worktree.logger.DebugOperation("failed to remove worktree directory during rollback", 
					"path", op.path, 
					"error", fsErr.Error())
			}
		}
	}

	// Clean up any directories we created (in reverse order)
	// Use RemoveAll instead of Remove to handle non-empty directories
	for i := len(op.createdDirs) - 1; i >= 0; i-- {
		dir := op.createdDirs[i]
		// Check if directory exists before attempting removal
		if _, err := os.Stat(dir); err == nil {
			// Only attempt to remove if it's empty or we created nested structure
			if err := os.Remove(dir); err != nil {
				// If Remove fails (directory not empty), that's expected for existing directories
				// We only created the path, not necessarily all contents
				// Log rollback failures at debug level to avoid cluttering user output
				op.worktree.logger.DebugOperation("could not remove directory during rollback (may contain user files)", 
					"directory", dir, 
					"error", err.Error())
			}
		}
	}

	// Clean up any files we created
	for _, file := range op.createdFiles {
		if err := os.Remove(file); err != nil {
			// Log rollback failures at debug level to avoid cluttering user output
			op.worktree.logger.DebugOperation("failed to remove created file during rollback", 
				"file", file, 
				"error", err.Error())
		}
	}
}

// handleWorktreeError provides enhanced error messages for common worktree failures
// and attempts automatic resolution when safe to do so
func (op *worktreeOperation) handleWorktreeError(err error) error {
	errorStr := err.Error()
	
	// Check for "already used by worktree" error
	if strings.Contains(errorStr, "already used by worktree") {
		// Try to extract the worktree path from the error message
		// Error format: "fatal: 'branchname' is already used by worktree at '/path/to/worktree'"
		worktreePath := ""
		if idx := strings.Index(errorStr, "at '"); idx != -1 {
			start := idx + 4
			if end := strings.Index(errorStr[start:], "'"); end != -1 {
				worktreePath = errorStr[start : start+end]
			}
		}
		
		// Attempt automatic conflict resolution if we have a valid worktree path
		if worktreePath != "" {
			// Provide user feedback about conflict resolution attempt
			if op.progressCallback != nil {
				op.progressCallback(fmt.Sprintf("Branch '%s' is in use, attempting automatic resolution...", op.branchName))
			}
			
			op.worktree.logger.DebugOperation("attempting automatic worktree conflict resolution",
				"branch", op.branchName,
				"conflicting_worktree", worktreePath)
			
			if resolveErr := op.worktree.resolveWorktreeConflict(op.branchName, worktreePath); resolveErr != nil {
				op.worktree.logger.DebugOperation("automatic conflict resolution failed",
					"branch", op.branchName,
					"conflicting_worktree", worktreePath,
					"error", resolveErr.Error())
				
				// Inform user that automatic resolution failed
				if op.progressCallback != nil {
					if strings.Contains(resolveErr.Error(), "uncommitted changes") {
						op.progressCallback("Cannot resolve automatically: conflicting worktree has uncommitted changes")
					} else {
						op.progressCallback("Automatic conflict resolution failed")
					}
				}
				
				// Return the original error with additional context about failed resolution
				return groveErrors.ErrBranchInUseByWorktree(op.branchName, worktreePath).
					WithContext("resolution_attempted", true).
					WithContext("resolution_error", resolveErr.Error())
			}
			
			// Inform user about successful resolution
			if op.progressCallback != nil {
				op.progressCallback(fmt.Sprintf("Resolved conflict: switched previous worktree to detached HEAD"))
			}
			
			// Conflict resolved successfully, retry the original worktree creation
			op.worktree.logger.DebugOperation("worktree conflict resolved, retrying creation",
				"branch", op.branchName,
				"conflicting_worktree", worktreePath)
			
			return op.retryWorktreeCreation()
		}
		
		return groveErrors.ErrBranchInUseByWorktree(op.branchName, worktreePath)
	}
	
	// Default to generic worktree creation error
	return groveErrors.ErrWorktreeCreation("create", err)
}

// resolveWorktreeConflict attempts to resolve a branch conflict by switching the conflicting
// worktree to a detached HEAD state, allowing the branch to be used elsewhere
func (w *WorktreeCreatorImpl) resolveWorktreeConflict(branchName, conflictingWorktreePath string) error {
	w.logger.DebugOperation("checking worktree status before conflict resolution",
		"branch", branchName,
		"worktree_path", conflictingWorktreePath)
	
	// First, check if the conflicting worktree has uncommitted changes
	if hasChanges, err := w.worktreeHasUncommittedChanges(conflictingWorktreePath); err != nil {
		return fmt.Errorf("failed to check worktree status: %w", err)
	} else if hasChanges {
		return fmt.Errorf("conflicting worktree has uncommitted changes - manual resolution required")
	}
	
	// Switch the conflicting worktree to detached HEAD at current commit
	w.logger.DebugOperation("switching conflicting worktree to detached HEAD",
		"branch", branchName,
		"worktree_path", conflictingWorktreePath)
	
	_, err := w.executor.Execute("-C", conflictingWorktreePath, "checkout", "--detach")
	if err != nil {
		return fmt.Errorf("failed to detach HEAD in conflicting worktree: %w", err)
	}
	
	w.logger.DebugOperation("successfully resolved worktree conflict",
		"branch", branchName,
		"worktree_path", conflictingWorktreePath)
	
	return nil
}

// worktreeHasUncommittedChanges checks if a worktree has uncommitted changes
func (w *WorktreeCreatorImpl) worktreeHasUncommittedChanges(worktreePath string) (bool, error) {
	// Use git status --porcelain to check for uncommitted changes
	output, err := w.executor.ExecuteQuiet("-C", worktreePath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	
	// If output is empty (after trimming whitespace), there are no uncommitted changes
	return strings.TrimSpace(output) != "", nil
}

// retryWorktreeCreation retries the original worktree creation after conflict resolution
func (op *worktreeOperation) retryWorktreeCreation() error {
	var args []string
	
	// Determine if this is a new branch or existing branch creation
	exists, err := op.worktree.branchExists(op.branchName)
	if err != nil {
		return fmt.Errorf("failed to check branch existence during retry: %w", err)
	}
	
	if exists {
		// Create worktree for existing branch
		args = []string{"worktree", "add", op.path, op.branchName}
	} else {
		// Create worktree with new branch
		args = []string{"worktree", "add", "-b", op.branchName, op.path}
	}
	
	// Track remote if specified
	if op.options.TrackRemote && exists {
		// Add tracking after worktree creation for existing branches
		if _, trackErr := op.worktree.executor.Execute("-C", op.path, "branch", "--set-upstream-to", "origin/"+op.branchName); trackErr != nil {
			op.worktree.logger.DebugOperation("failed to set upstream tracking", 
				"branch", op.branchName, 
				"error", trackErr.Error())
		}
	}
	
	_, err = op.worktree.executor.Execute(args...)
	return err
}

// branchExists checks if a branch exists locally.
func (w *WorktreeCreatorImpl) branchExists(branchName string) (bool, error) {
	_, err := w.executor.ExecuteQuiet("show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err != nil {
		return false, nil
	}
	return true, nil
}

