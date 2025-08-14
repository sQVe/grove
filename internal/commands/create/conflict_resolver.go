package create

import (
	"fmt"
	"strings"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
)

// conflictResolver handles worktree branch conflicts by managing
// detachment of conflicting branches.
type conflictResolver struct {
	commander git.Commander
	logger    *logger.Logger
}

func newConflictResolver(commander git.Commander) *conflictResolver {
	return &conflictResolver{
		commander: commander,
		logger:    logger.WithComponent("conflict_resolver"),
	}
}

// resolveWorktreeConflict attempts to resolve a branch conflict by switching the conflicting
// worktree to a detached HEAD state, allowing the branch to be used elsewhere.
func (cr *conflictResolver) resolveWorktreeConflict(branchName, conflictingWorktreePath string) error {
	cr.logger.DebugOperation("checking worktree status before conflict resolution",
		"branch", branchName,
		"worktree_path", conflictingWorktreePath)

	if isMain, err := cr.isMainWorktree(conflictingWorktreePath); err != nil {
		return groveErrors.ErrGitOperation("worktree-list", err).
			WithContext("operation", "determine_main_worktree").
			WithContext("worktree_path", conflictingWorktreePath)
	} else if isMain {
		return groveErrors.ErrWorktreeCreation("main-worktree-conflict",
			fmt.Errorf("cannot automatically resolve conflict with main worktree")).
			WithContext("worktree_path", conflictingWorktreePath).
			WithContext("reason", "main_worktree_protection")
	}

	if hasChanges, err := cr.worktreeHasUncommittedChanges(conflictingWorktreePath); err != nil {
		return groveErrors.ErrGitOperation("status", err).
			WithContext("operation", "check_uncommitted_changes").
			WithContext("worktree_path", conflictingWorktreePath)
	} else if hasChanges {
		return groveErrors.ErrWorktreeCreation("uncommitted-changes",
			fmt.Errorf("conflicting worktree has uncommitted changes")).
			WithContext("worktree_path", conflictingWorktreePath).
			WithContext("reason", "data_protection")
	}

	cr.logger.DebugOperation("switching conflicting worktree to detached HEAD",
		"branch", branchName,
		"worktree_path", conflictingWorktreePath)

	_, _, err := cr.commander.Run(conflictingWorktreePath, "checkout", "--detach")
	if err != nil {
		return groveErrors.ErrGitOperation("checkout --detach", err).
			WithContext("worktree_path", conflictingWorktreePath).
			WithContext("branch", branchName).
			WithContext("operation", "detach_head")
	}

	cr.logger.DebugOperation("successfully resolved worktree conflict",
		"branch", branchName,
		"worktree_path", conflictingWorktreePath)

	return nil
}

func (cr *conflictResolver) isMainWorktree(worktreePath string) (bool, error) {
	// Works with both regular worktrees and bare repositories.
	stdout, _, err := cr.commander.Run(".", "worktree", "list", "--porcelain")
	output := strings.TrimSpace(string(stdout))
	if err != nil {
		return false, err
	}

	// Find the main worktree (first non-bare worktree).
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var currentWorktreePath string
	var isBare bool

	for _, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			if currentWorktreePath != "" && !isBare {
				return currentWorktreePath == worktreePath, nil
			}
			currentWorktreePath = ""
			isBare = false
		case strings.HasPrefix(line, "worktree "):
			currentWorktreePath = strings.TrimPrefix(line, "worktree ")
		case line == "bare":
			isBare = true
		}
	}

	if currentWorktreePath != "" && !isBare {
		return currentWorktreePath == worktreePath, nil
	}

	return false, nil
}

func (cr *conflictResolver) worktreeHasUncommittedChanges(worktreePath string) (bool, error) {
	stdout, _, err := cr.commander.Run(worktreePath, "status", "--porcelain")
	output := strings.TrimSpace(string(stdout))
	if err != nil {
		return false, err
	}

	return strings.TrimSpace(output) != "", nil
}
