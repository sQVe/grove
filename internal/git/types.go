package git

// BranchName represents a git branch name.
// Using a distinct type prevents mixing up branch names with other strings.
type BranchName string

// WorktreePath represents an absolute path to a worktree directory.
type WorktreePath string

// RepoPath represents an absolute path to a git repository (bare or normal).
type RepoPath string

// RemoteName represents a git remote name (e.g., "origin").
type RemoteName string
