package create

type CreateOptions struct {
	BranchName     string
	WorktreePath   string
	BaseBranch     string // Base branch when creating new branches.
	CreateBranch   bool   // Skip prompting when creating new branches.
	Force          bool
	CopyFiles      bool
	CopyPatterns   []string // Overrides default copy configuration.
	CopyEnv        bool     // Shorthand for common environment file patterns.
	SourceWorktree string   // Override automatic source detection.
}

type CreateResult struct {
	WorktreePath string
	BranchName   string
	WasCreated   bool // Whether branch was newly created vs. existing.
	BaseBranch   string
	CopiedFiles  int
}

type BranchInfo struct {
	Name           string
	Exists         bool
	IsRemote       bool // Exists only on remote, requires checkout.
	TrackingBranch string
	RemoteName     string
}

type URLBranchInfo struct {
	RepoURL        string
	BranchName     string
	PRNumber       string // For PR/MR URLs.
	Platform       string // github, gitlab, bitbucket, etc.
	RequiresRemote bool   // Remote must be configured before checkout.
}

type WorktreeOptions struct {
	TrackRemote bool
	Force       bool
}

type CopyOptions struct {
	ConflictStrategy ConflictStrategy
	DryRun           bool // Preview mode without actual changes.
}

type ConflictStrategy string

const (
	ConflictPrompt    ConflictStrategy = "prompt"
	ConflictSkip      ConflictStrategy = "skip"
	ConflictOverwrite ConflictStrategy = "overwrite"
	ConflictBackup    ConflictStrategy = "backup"
)

type FileConflict struct {
	Path       string // Relative to worktree root.
	SourcePath string
	TargetPath string
}
