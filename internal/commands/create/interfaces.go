package create

type CreateService interface {
	Create(options *CreateOptions) (*CreateResult, error)
}

type BranchResolver interface {
	ResolveBranch(name, base string, createIfMissing bool) (*BranchInfo, error)
	ResolveURL(url string) (*URLBranchInfo, error)
	ResolveRemoteBranch(remoteBranch string) (*BranchInfo, error)
	RemoteExists(remoteName string) bool
}

type PathGenerator interface {
	GeneratePath(branchName, basePath string) (string, error)
}

type WorktreeCreator interface {
	CreateWorktree(branchName, path string, options WorktreeOptions) error
	CreateWorktreeWithProgress(branchName, path string, options WorktreeOptions, progressCallback ProgressCallback) error
}

type FileManager interface {
	CopyFiles(sourceWorktree, targetWorktree string, patterns []string, options CopyOptions) error
	DiscoverSourceWorktree() (string, error)
	FindWorktreeByBranch(branchName string) (string, error)
	GetCurrentWorktreePath() (string, error)
	ResolveConflicts(conflicts []FileConflict, strategy ConflictStrategy) error
}
