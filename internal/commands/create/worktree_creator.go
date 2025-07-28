package create

import (
	"fmt"
	"os"
	"path/filepath"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
)

// WorktreeCreatorImpl implements the WorktreeCreator interface.
type WorktreeCreatorImpl struct {
	executor git.GitExecutor
}

// NewWorktreeCreator creates a new WorktreeCreator with the provided GitExecutor.
func NewWorktreeCreator(executor git.GitExecutor) *WorktreeCreatorImpl {
	return &WorktreeCreatorImpl{
		executor: executor,
	}
}

// CreateWorktree creates a new Git worktree at the specified path for the given branch.
// It handles both new branch creation and existing branch checkout based on the options.
func (w *WorktreeCreatorImpl) CreateWorktree(branchName, path string, options WorktreeOptions) error {
	if branchName == "" {
		return groveErrors.ErrWorktreeCreation("validation", fmt.Errorf("branch name cannot be empty"))
	}

	if path == "" {
		return groveErrors.ErrWorktreeCreation("validation", fmt.Errorf("worktree path cannot be empty"))
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return groveErrors.ErrDirectoryAccess(filepath.Dir(path), err)
	}

	if _, err := os.Stat(path); err == nil && !options.Force {
		return groveErrors.ErrPathExists(path)
	}

	branchExists, err := w.branchExists(branchName)
	if err != nil {
		return groveErrors.ErrWorktreeCreation("branch-check", err)
	}

	var worktreeErr error
	if branchExists {
		worktreeErr = w.createFromExistingBranch(branchName, path)
	} else {
		worktreeErr = w.createWithNewBranch(branchName, path, options.TrackRemote)
	}

	if worktreeErr != nil {
		w.cleanup(path)
		return worktreeErr
	}

	return nil
}

// branchExists checks if a branch exists locally.
func (w *WorktreeCreatorImpl) branchExists(branchName string) (bool, error) {
	_, err := w.executor.ExecuteQuiet("show-ref", "--verify", "--quiet", "refs/heads/"+branchName)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// createFromExistingBranch creates a worktree from an existing branch.
func (w *WorktreeCreatorImpl) createFromExistingBranch(branchName, path string) error {
	_, err := w.executor.Execute("worktree", "add", path, branchName)
	if err != nil {
		return groveErrors.ErrGitWorktree("add", err)
	}
	return nil
}

// createWithNewBranch creates a worktree with a new branch.
func (w *WorktreeCreatorImpl) createWithNewBranch(branchName, path string, trackRemote bool) error {
	args := []string{"worktree", "add", "-b", branchName, path}

	_, err := w.executor.Execute(args...)
	if err != nil {
		return groveErrors.ErrWorktreeCreation("create", err)
	}

	if trackRemote {
		if err := w.setupRemoteTracking(branchName, path); err != nil {
			// Remote tracking failure is not critical, log but don't fail.
			// The worktree was created successfully.
			return groveErrors.ErrWorktreeCreation("remote-tracking", err)
		}
	}

	return nil
}

// setupRemoteTracking configures the new branch to track a remote branch.
func (w *WorktreeCreatorImpl) setupRemoteTracking(branchName, worktreePath string) error {
	remote, err := w.executor.ExecuteQuiet("config", "--get", "clone.defaultRemoteName")
	if err != nil || remote == "" {
		remote = "origin" // Fallback to standard default.
	}

	remoteBranch := remote + "/" + branchName
	_, err = w.executor.Execute("-C", worktreePath, "branch", "--set-upstream-to="+remoteBranch, branchName)
	if err != nil {
		return groveErrors.ErrWorktreeCreation("set-upstream", err)
	}

	return nil
}

// cleanup removes a partially created worktree directory on failure.
func (w *WorktreeCreatorImpl) cleanup(path string) {
	if _, err := os.Stat(path); err == nil {
		if _, gitErr := w.executor.ExecuteQuiet("worktree", "remove", "--force", path); gitErr != nil {
			fmt.Printf("Warning: failed to remove worktree via git: %v\n", gitErr)
		}
		if fsErr := os.RemoveAll(path); fsErr != nil {
			fmt.Printf("Warning: failed to remove directory %s: %v\n", path, fsErr)
		}
	}
}
