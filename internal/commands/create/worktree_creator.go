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
	executor         git.GitExecutor
	logger           *logger.Logger
	conflictResolver *conflictResolver
}

// NewWorktreeCreator creates a new WorktreeCreator with the provided GitExecutor.
func NewWorktreeCreator(executor git.GitExecutor) *WorktreeCreatorImpl {
	return &WorktreeCreatorImpl{
		executor:         executor,
		logger:           logger.WithComponent("worktree_creator"),
		conflictResolver: newConflictResolver(executor),
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
	if err := op.validateAndPrepare(); err != nil {
		op.rollback()
		return err
	}

	if err := op.createWorktreeAtomically(); err != nil {
		op.rollback()
		return err
	}

	return nil
}

// validateAndPrepare handles pre-creation validation and setup
func (op *worktreeOperation) validateAndPrepare() error {
	// Ensure parent directory exists
	if err := op.ensureParentDirectory(); err != nil {
		return err
	}

	// Note: Path existence checking is now handled by path generator's atomic collision resolution
	// The path generator creates directories atomically to prevent race conditions
	// If path exists, it would have been resolved during path generation phase

	return nil
}

// createWorktreeAtomically performs the actual worktree creation
func (op *worktreeOperation) createWorktreeAtomically() error {
	// Check if branch exists
	branchExists, err := op.worktree.branchExists(op.branchName)
	if err != nil {
		return groveErrors.ErrWorktreeCreation("branch-check", err)
	}

	// Create worktree using appropriate strategy
	if branchExists {
		return op.createFromExistingBranch()
	} else {
		return op.createWithNewBranch()
	}
}

// ensureParentDirectory creates parent directories if needed
func (op *worktreeOperation) ensureParentDirectory() error {
	parentDir := filepath.Dir(op.path)

	// Identify which directories need to be created before creating them
	dirsToCreate := op.identifyDirectoriesToCreate(parentDir)

	// Create the directories
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return groveErrors.ErrDirectoryAccess(parentDir, err)
	}

	// Track in reverse order for proper cleanup.
	for i := len(dirsToCreate) - 1; i >= 0; i-- {
		op.createdDirs = append(op.createdDirs, dirsToCreate[i])
	}

	return nil
}

// Returns directories that need creation in parent-to-child order.
func (op *worktreeOperation) identifyDirectoriesToCreate(targetDir string) []string {
	var dirsToCreate []string
	currentPath := targetDir

	for currentPath != "." && currentPath != "/" {
		if _, err := os.Stat(currentPath); os.IsNotExist(err) {
			// Prepend to maintain parent-to-child order.
			dirsToCreate = append([]string{currentPath}, dirsToCreate...)
			currentPath = filepath.Dir(currentPath)
		} else {
			break
		}
	}

	return dirsToCreate
}

func (op *worktreeOperation) createFromExistingBranch() error {
	_, err := op.worktree.executor.Execute("worktree", "add", op.path, op.branchName)
	if err != nil {
		if op.isBranchInUseError(err) {
			return op.handleBranchConflict(err)
		}
		return groveErrors.ErrWorktreeCreation("existing-branch", err)
	}
	return nil
}

func (op *worktreeOperation) createWithNewBranch() error {
	args := []string{"worktree", "add", "-b", op.branchName, op.path}

	if op.options.BaseBranch != "" {
		args = append(args, op.options.BaseBranch)
	}

	_, err := op.worktree.executor.Execute(args...)
	if err != nil {
		if op.isBranchInUseError(err) {
			return op.handleBranchConflict(err)
		}
		return groveErrors.ErrWorktreeCreation("new-branch", err)
	}

	if op.options.TrackRemote {
		if err := op.setupRemoteTracking(); err != nil {
			// Remote tracking failure is not critical but should be cleaned up
			return groveErrors.ErrWorktreeCreation("remote-tracking", err)
		}
	}

	return nil
}

// isBranchInUseError checks if the error indicates a branch is already in use by another worktree
func (op *worktreeOperation) isBranchInUseError(err error) bool {
	return strings.Contains(err.Error(), "already used by worktree")
}

// handleBranchConflict attempts to resolve branch conflicts using Grove error patterns
func (op *worktreeOperation) handleBranchConflict(err error) error {
	worktreePath := op.extractWorktreePath(err.Error())

	if op.progressCallback != nil {
		op.progressCallback(fmt.Sprintf("Branch '%s' is in use, attempting automatic resolution...", op.branchName))
	}

	op.worktree.logger.DebugOperation("attempting automatic worktree conflict resolution",
		"branch", op.branchName,
		"conflicting_worktree", worktreePath)

	if resolveErr := op.worktree.conflictResolver.resolveWorktreeConflict(op.branchName, worktreePath); resolveErr != nil {
		op.worktree.logger.DebugOperation("automatic conflict resolution failed",
			"branch", op.branchName,
			"conflicting_worktree", worktreePath,
			"error", resolveErr.Error())

		if op.progressCallback != nil {
			if strings.Contains(resolveErr.Error(), "uncommitted changes") {
				op.progressCallback("Cannot resolve automatically: conflicting worktree has uncommitted changes")
			} else {
				op.progressCallback("Automatic conflict resolution failed")
			}
		}

		return groveErrors.ErrBranchInUseByWorktree(op.branchName, worktreePath).
			WithContext("resolution_attempted", true).
			WithContext("resolution_error", resolveErr.Error())
	}

	if op.progressCallback != nil {
		op.progressCallback("Resolved conflict: switched previous worktree to detached HEAD")
	}

	op.worktree.logger.DebugOperation("worktree conflict resolved successfully",
		"branch", op.branchName,
		"conflicting_worktree", worktreePath)

	// Conflict resolved, retry the operation
	return op.retryWorktreeCreation()
}

// extractWorktreePath extracts the worktree path from error message
func (op *worktreeOperation) extractWorktreePath(errorStr string) string {
	if idx := strings.Index(errorStr, "at '"); idx != -1 {
		start := idx + 4
		if end := strings.Index(errorStr[start:], "'"); end != -1 {
			return errorStr[start : start+end]
		}
	}
	return ""
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
