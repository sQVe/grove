package fs

const (
	// Strict permissions (gosec-compliant defaults)
	DirStrict  = 0o750 // rwxr-x--- - gosec-compliant directory
	FileStrict = 0o600 // rw------- - gosec-compliant file

	// Git-compatible permissions (required for git operations)
	DirGit  = 0o755 // rwxr-xr-x - git-compatible directory
	FileGit = 0o644 // rw-r--r-- - git-compatible file

	// Private permissions for sensitive data
	DirPrivate  = 0o700 // rwx------ - user-only directory
	FilePrivate = 0o600 // rw------- - user-only file
)

// Common git filesystem paths
const (
	GitDir         = ".git"
	GitWorktreeDir = ".git/worktrees"
)

// Grove-specific filesystem constants
const (
	GroveConfig        = ".grove.toml"
	DefaultWorktreeDir = "worktrees"
)
