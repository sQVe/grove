package git

// BranchName represents a git branch name.
// Using a distinct type prevents mixing up branch names with other strings.
type BranchName string

// String returns the branch name as a string.
func (b BranchName) String() string {
	return string(b)
}

// WorktreePath represents an absolute path to a worktree directory.
type WorktreePath string

// String returns the path as a string.
func (p WorktreePath) String() string {
	return string(p)
}

// RepoPath represents an absolute path to a git repository (bare or normal).
type RepoPath string

// String returns the path as a string.
func (p RepoPath) String() string {
	return string(p)
}

// RemoteName represents a git remote name (e.g., "origin").
type RemoteName string

// String returns the remote name as a string.
func (r RemoteName) String() string {
	return string(r)
}
