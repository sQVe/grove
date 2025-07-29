package remove

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
)

// SafetyCheckerImpl implements the SafetyChecker interface with comprehensive validation.
type SafetyCheckerImpl struct {
	executor git.GitExecutor
	logger   *logger.Logger
}

// NewSafetyChecker creates a new SafetyChecker instance.
func NewSafetyChecker(executor git.GitExecutor, log *logger.Logger) SafetyChecker {
	return &SafetyCheckerImpl{
		executor: executor,
		logger:   log,
	}
}

// CheckUncommittedChanges validates if worktree has uncommitted changes.
func (s *SafetyCheckerImpl) CheckUncommittedChanges(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("worktree path cannot be empty")
	}

	// Verify path exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false, fmt.Errorf("worktree path does not exist: %s", path)
	}

	// Get worktree status using existing git package function
	status, err := s.getWorktreeStatus(path)
	if err != nil {
		s.logger.DebugOperation("failed to get worktree status for safety check",
			"path", path,
			"error", err.Error())
		return false, fmt.Errorf("failed to check worktree status: %w", err)
	}

	// Has uncommitted changes if not clean
	hasUncommitted := !status.IsClean

	s.logger.DebugOperation("checked uncommitted changes",
		"path", path,
		"has_uncommitted", hasUncommitted,
		"modified", status.Modified,
		"staged", status.Staged,
		"untracked", status.Untracked)

	return hasUncommitted, nil
}

// CheckBranchSafety validates if a branch can be safely deleted.
func (s *SafetyCheckerImpl) CheckBranchSafety(branch string) (BranchSafetyStatus, error) {
	if branch == "" {
		return BranchSafetyStatus{}, fmt.Errorf("branch name cannot be empty")
	}

	status := BranchSafetyStatus{
		BranchName: branch,
	}

	// Check if branch is merged into default branch
	isMerged, err := s.isBranchMerged(branch)
	if err != nil {
		s.logger.DebugOperation("failed to check if branch is merged",
			"branch", branch,
			"error", err.Error())
		// Continue with other checks even if merge check fails
	} else {
		status.IsMerged = isMerged
	}

	// Check if branch exists on remote
	isPushed, err := s.isBranchPushedToRemote(branch)
	if err != nil {
		s.logger.DebugOperation("failed to check remote branch status",
			"branch", branch,
			"error", err.Error())
		// Continue with other checks even if remote check fails
	} else {
		status.IsPushedToRemote = isPushed
	}

	// Determine deletion safety
	if status.IsMerged || (status.IsPushedToRemote && !status.IsMerged) {
		status.CanDeleteAuto = true
		status.RequiresConfirm = false
		if status.IsMerged {
			status.Reason = "Branch is merged into default branch"
		} else {
			status.Reason = "Branch is pushed to remote but not merged"
		}
	} else {
		status.CanDeleteAuto = false
		status.RequiresConfirm = true
		status.Reason = "Branch is not merged and may contain unique changes"
	}

	s.logger.DebugOperation("checked branch safety",
		"branch", branch,
		"is_merged", status.IsMerged,
		"is_pushed", status.IsPushedToRemote,
		"can_delete_auto", status.CanDeleteAuto,
		"requires_confirm", status.RequiresConfirm)

	return status, nil
}

// CheckCurrentWorktree validates if worktree is currently active.
func (s *SafetyCheckerImpl) CheckCurrentWorktree(path string) (bool, error) {
	if path == "" {
		return false, fmt.Errorf("worktree path cannot be empty")
	}

	// Get current worktree path
	currentPath, err := s.getCurrentWorktreePath()
	if err != nil {
		s.logger.DebugOperation("failed to get current worktree path",
			"error", err.Error())
		return false, fmt.Errorf("failed to determine current worktree: %w", err)
	}

	// If no current path, we're not in a worktree
	if currentPath == "" {
		return false, nil
	}

	// Clean and compare paths
	cleanCurrent, err := filepath.Abs(currentPath)
	if err != nil {
		return false, fmt.Errorf("failed to resolve current path: %w", err)
	}

	cleanTarget, err := filepath.Abs(path)
	if err != nil {
		return false, fmt.Errorf("failed to resolve target path: %w", err)
	}

	isCurrent := cleanCurrent == cleanTarget

	s.logger.DebugOperation("checked current worktree",
		"target_path", path,
		"current_path", currentPath,
		"is_current", isCurrent)

	return isCurrent, nil
}

// getWorktreeStatus wraps the git package function for internal use.
func (s *SafetyCheckerImpl) getWorktreeStatus(worktreePath string) (git.WorktreeStatus, error) {
	output, err := s.executor.ExecuteQuiet("-C", worktreePath, "status", "--porcelain")
	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "must be run in a work tree") {
			s.logger.DebugOperation("git status failed with 'must be run in a work tree' error",
				"path", worktreePath,
				"error", errStr)
			return git.WorktreeStatus{IsClean: true}, nil
		}
		return git.WorktreeStatus{}, err
	}

	status := git.WorktreeStatus{}
	lines := strings.Split(strings.TrimSpace(output), "\n")

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		indexStatus := line[0]
		workTreeStatus := line[1]

		if indexStatus != ' ' && indexStatus != '?' {
			status.Staged++
		}

		if workTreeStatus != ' ' && workTreeStatus != '?' {
			status.Modified++
		}

		if indexStatus == '?' && workTreeStatus == '?' {
			status.Untracked++
		}
	}

	status.IsClean = (status.Modified == 0 && status.Staged == 0 && status.Untracked == 0)
	return status, nil
}

// getCurrentWorktreePath wraps the git package function for internal use.
func (s *SafetyCheckerImpl) getCurrentWorktreePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	output, err := s.executor.ExecuteQuiet("-C", cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		s.logger.DebugOperation("git rev-parse failed, assuming not in git repository",
			"error", err,
			"directory", cwd)
		return "", nil
	}

	return strings.TrimSpace(output), nil
}

// isBranchMerged checks if a branch has been merged into the default branch.
func (s *SafetyCheckerImpl) isBranchMerged(branch string) (bool, error) {
	// Get default branch name
	defaultBranch, err := s.getDefaultBranch()
	if err != nil {
		return false, fmt.Errorf("failed to determine default branch: %w", err)
	}

	// Check if branch is merged into default branch
	output, err := s.executor.ExecuteQuiet("merge-base", "--is-ancestor", branch, defaultBranch)
	if err != nil {
		// If git merge-base fails, the branch likely isn't merged
		return false, nil
	}

	// If command succeeds with empty output, branch is merged
	return strings.TrimSpace(output) == "", nil
}

// isBranchPushedToRemote checks if a branch exists on the remote repository.
func (s *SafetyCheckerImpl) isBranchPushedToRemote(branch string) (bool, error) {
	// Check if remote tracking branch exists
	output, err := s.executor.ExecuteQuiet("branch", "-r", "--list", "origin/"+branch)
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(output) != "", nil
}

// getDefaultBranch determines the default branch name for the repository.
func (s *SafetyCheckerImpl) getDefaultBranch() (string, error) {
	// Try to get default branch from remote
	output, err := s.executor.ExecuteQuiet("symbolic-ref", "refs/remotes/origin/HEAD")
	if err == nil {
		parts := strings.Split(strings.TrimSpace(output), "/")
		if len(parts) > 0 {
			return parts[len(parts)-1], nil
		}
	}

	// Fallback to common default branch names
	commonDefaults := []string{"main", "master", "develop"}
	for _, defaultName := range commonDefaults {
		_, err := s.executor.ExecuteQuiet("rev-parse", "--verify", defaultName)
		if err == nil {
			return defaultName, nil
		}
	}

	return "", fmt.Errorf("could not determine default branch")
}
