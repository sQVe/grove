package git

// Client defines the interface for git operations.
// This enables test doubles for code that depends on git.
type Client interface {
	// Repository inspection
	IsInsideGitRepo(path string) bool
	IsWorktree(path string) bool
	GetCurrentBranch(path string) (string, error)
	GetDefaultBranch(path string) (string, error)

	// Branch operations
	BranchExists(path, branch string) (bool, error)
	ListBranches(bareRepo string) ([]string, error)

	// Worktree operations
	ListWorktrees(path string) ([]string, error)
	ListWorktreesWithInfo(bareDir string, fast bool) ([]*WorktreeInfo, error)
	CreateWorktree(bareDir, path, branch string, quiet bool) error
	RemoveWorktree(bareDir, path string, force bool) error
	LockWorktree(bareDir, path, reason string) error
	UnlockWorktree(bareDir, path string) error

	// Status checks
	HasUnresolvedConflicts(path string) (bool, error)
	HasOngoingOperation(path string) (bool, error)
	CheckGitChanges(path string) (hasAnyChanges, hasTrackedChanges bool, err error)
	HasUnpushedCommits(path string) (bool, error)
}

// DefaultClient is the production implementation that shells out to git.
type DefaultClient struct{}

// NewClient returns the default git client.
func NewClient() Client {
	return &DefaultClient{}
}

// IsInsideGitRepo delegates to the package function.
func (c *DefaultClient) IsInsideGitRepo(path string) bool {
	return IsInsideGitRepo(path)
}

// IsWorktree delegates to the package function.
func (c *DefaultClient) IsWorktree(path string) bool {
	return IsWorktree(path)
}

// GetCurrentBranch delegates to the package function.
func (c *DefaultClient) GetCurrentBranch(path string) (string, error) {
	return GetCurrentBranch(path)
}

// GetDefaultBranch delegates to the package function.
func (c *DefaultClient) GetDefaultBranch(path string) (string, error) {
	return GetDefaultBranch(path)
}

// BranchExists delegates to the package function.
func (c *DefaultClient) BranchExists(path, branch string) (bool, error) {
	return BranchExists(path, branch)
}

// ListBranches delegates to the package function.
func (c *DefaultClient) ListBranches(bareRepo string) ([]string, error) {
	return ListBranches(bareRepo)
}

// ListWorktrees delegates to the package function.
func (c *DefaultClient) ListWorktrees(path string) ([]string, error) {
	return ListWorktrees(path)
}

// ListWorktreesWithInfo delegates to the package function.
func (c *DefaultClient) ListWorktreesWithInfo(bareDir string, fast bool) ([]*WorktreeInfo, error) {
	return ListWorktreesWithInfo(bareDir, fast)
}

// CreateWorktree delegates to the package function.
func (c *DefaultClient) CreateWorktree(bareDir, path, branch string, quiet bool) error {
	return CreateWorktree(bareDir, path, branch, quiet)
}

// RemoveWorktree delegates to the package function.
func (c *DefaultClient) RemoveWorktree(bareDir, path string, force bool) error {
	return RemoveWorktree(bareDir, path, force)
}

// LockWorktree delegates to the package function.
func (c *DefaultClient) LockWorktree(bareDir, path, reason string) error {
	return LockWorktree(bareDir, path, reason)
}

// UnlockWorktree delegates to the package function.
func (c *DefaultClient) UnlockWorktree(bareDir, path string) error {
	return UnlockWorktree(bareDir, path)
}

// HasUnresolvedConflicts delegates to the package function.
func (c *DefaultClient) HasUnresolvedConflicts(path string) (bool, error) {
	return HasUnresolvedConflicts(path)
}

// HasOngoingOperation delegates to the package function.
func (c *DefaultClient) HasOngoingOperation(path string) (bool, error) {
	return HasOngoingOperation(path)
}

// CheckGitChanges delegates to the package function.
func (c *DefaultClient) CheckGitChanges(path string) (hasAnyChanges, hasTrackedChanges bool, err error) {
	return CheckGitChanges(path)
}

// HasUnpushedCommits delegates to the package function.
func (c *DefaultClient) HasUnpushedCommits(path string) (bool, error) {
	return HasUnpushedCommits(path)
}
