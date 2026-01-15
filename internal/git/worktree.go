package git

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

// WorktreeInfo contains status information about a worktree
type WorktreeInfo struct {
	Path           string // Absolute path to worktree
	Branch         string // Branch name (or commit hash if detached)
	Upstream       string // Upstream branch name (e.g., "origin/main")
	Dirty          bool   // Has uncommitted changes
	Ahead          int    // Commits ahead of upstream
	Behind         int    // Commits behind upstream
	Gone           bool   // Upstream branch deleted
	NoUpstream     bool   // No upstream configured
	Locked         bool   // Worktree is locked
	LockReason     string // Reason for lock (empty if not locked)
	LastCommitTime int64  // Unix timestamp of last commit (0 if unknown)
	Detached       bool   // Worktree is in detached HEAD state
}

// CreateWorktree creates a new worktree from a bare repository
func CreateWorktree(bareRepo, worktreePath, branch string, quiet bool) error {
	if bareRepo == "" {
		return errors.New("bare repository path cannot be empty")
	}
	if worktreePath == "" {
		return errors.New("worktree path cannot be empty")
	}
	if branch == "" {
		return errors.New("branch name cannot be empty")
	}

	logger.Debug("Executing: git worktree add %s %s", worktreePath, branch)
	cmd, cancel := GitCommand("git", "worktree", "add", worktreePath, branch)
	defer cancel()
	cmd.Dir = bareRepo

	return runGitCommand(cmd, quiet)
}

// CreateWorktreeWithNewBranch creates a new worktree with a new branch.
// Uses: git worktree add -b <branch> <path>
func CreateWorktreeWithNewBranch(bareRepo, worktreePath, branch string, quiet bool) error {
	if bareRepo == "" {
		return errors.New("bare repository path cannot be empty")
	}
	if worktreePath == "" {
		return errors.New("worktree path cannot be empty")
	}
	if branch == "" {
		return errors.New("branch name cannot be empty")
	}

	logger.Debug("Executing: git worktree add -b %s %s", branch, worktreePath)
	cmd, cancel := GitCommand("git", "worktree", "add", "-b", branch, worktreePath)
	defer cancel()
	cmd.Dir = bareRepo

	return runGitCommand(cmd, quiet)
}

// CreateWorktreeWithNewBranchFrom creates a new worktree with a new branch based on a specific commit/branch.
// Uses: git worktree add -b <newbranch> <path> <base>
func CreateWorktreeWithNewBranchFrom(bareRepo, worktreePath, branch, base string, quiet bool) error {
	if bareRepo == "" {
		return errors.New("bare repository path cannot be empty")
	}
	if worktreePath == "" {
		return errors.New("worktree path cannot be empty")
	}
	if branch == "" {
		return errors.New("branch name cannot be empty")
	}
	if base == "" {
		return errors.New("base reference cannot be empty")
	}

	logger.Debug("Executing: git worktree add -b %s %s %s", branch, worktreePath, base)
	cmd, cancel := GitCommand("git", "worktree", "add", "-b", branch, worktreePath, base)
	defer cancel()
	cmd.Dir = bareRepo

	return runGitCommand(cmd, quiet)
}

// CreateWorktreeDetached creates a worktree in detached HEAD state at the specified ref.
// Uses: git worktree add --detach <path> <ref>
func CreateWorktreeDetached(bareRepo, worktreePath, ref string, quiet bool) error {
	if bareRepo == "" {
		return errors.New("bare repository path cannot be empty")
	}
	if worktreePath == "" {
		return errors.New("worktree path cannot be empty")
	}
	if ref == "" {
		return errors.New("ref cannot be empty")
	}

	logger.Debug("Executing: git worktree add --detach %s %s", worktreePath, ref)
	cmd, cancel := GitCommand("git", "worktree", "add", "--detach", worktreePath, ref)
	defer cancel()
	cmd.Dir = bareRepo

	return runGitCommand(cmd, quiet)
}

// RemoveWorktree removes a worktree directory
func RemoveWorktree(bareDir, worktreePath string, force bool) error {
	args := []string{"worktree", "remove", worktreePath}
	if force {
		args = append(args, "--force")
	}
	logger.Debug("Executing: git %s in %s", strings.Join(args, " "), bareDir)
	cmd, cancel := GitCommand("git", args...) // nolint:gosec // Worktree path comes from git worktree list
	defer cancel()
	cmd.Dir = bareDir
	return runGitCommand(cmd, true)
}

// RepairWorktree runs git worktree repair to fix worktree paths after directory moves.
func RepairWorktree(bareDir, worktreePath string) error {
	if bareDir == "" {
		return errors.New("bare directory path cannot be empty")
	}

	args := []string{"worktree", "repair"}
	if worktreePath != "" {
		args = append(args, worktreePath)
	}

	logger.Debug("Executing: git %v in %s", args, bareDir)
	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	cmd.Dir = bareDir

	return runGitCommand(cmd, true)
}

// ListWorktrees returns paths to existing worktrees, excluding the main repository
func ListWorktrees(repoPath string) ([]string, error) {
	logger.Debug("Executing: git worktree list --porcelain in %s", repoPath)
	cmd, cancel := GitCommand("git", "worktree", "list", "--porcelain")
	defer cancel()
	cmd.Dir = repoPath

	out, err := executeWithOutputBuffer(cmd)
	if err != nil {
		return nil, err
	}

	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	var worktrees []string
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := scanner.Text()

		// Porcelain format: "worktree /path/to/worktree" on its own line
		if !strings.HasPrefix(line, "worktree ") {
			continue
		}

		worktreePath := strings.TrimPrefix(line, "worktree ")
		// Normalize path separators (git on Windows uses forward slashes)
		worktreePath = filepath.FromSlash(worktreePath)

		absWorktreePath, err := filepath.Abs(worktreePath)
		if err != nil {
			return nil, err
		}

		// Skip the main worktree (same as repo path)
		// Use filepath.Clean for consistent comparison across platforms
		if filepath.Clean(absWorktreePath) == filepath.Clean(absRepoPath) {
			continue
		}

		worktrees = append(worktrees, absWorktreePath)
	}

	return worktrees, scanner.Err()
}

// ListWorktreesWithInfo returns info for all worktrees in a grove workspace.
func ListWorktreesWithInfo(bareDir string, fast bool) ([]*WorktreeInfo, error) {
	paths, err := ListWorktrees(bareDir)
	if err != nil {
		return nil, err
	}

	var infos []*WorktreeInfo
	for _, path := range paths {
		var info *WorktreeInfo
		if fast {
			branch, detached, err := GetCurrentBranchOrDetached(path)
			if err != nil {
				if errors.Is(err, ErrDetachedHead) {
					info = &WorktreeInfo{
						Path:     path,
						Branch:   "(detached)",
						Detached: true,
					}
				} else {
					logger.Warning("Skipping worktree %s (may be corrupted): %v", path, err)
					continue
				}
			} else {
				info = &WorktreeInfo{
					Path:     path,
					Branch:   branch,
					Detached: detached,
				}
			}
		} else {
			var err error
			info, err = GetWorktreeInfo(path)
			if err != nil {
				if errors.Is(err, ErrDetachedHead) {
					info = &WorktreeInfo{
						Path:     path,
						Branch:   "(detached)",
						Detached: true,
					}
				} else {
					logger.Warning("Skipping worktree %s (may be corrupted): %v", path, err)
					continue
				}
			}
		}

		info.Locked = IsWorktreeLocked(path)
		info.LockReason = GetWorktreeLockReason(path)

		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Branch < infos[j].Branch
	})

	return infos, nil
}

// GetWorktreeInfo returns status information for a worktree
func GetWorktreeInfo(path string) (*WorktreeInfo, error) {
	if path == "" {
		return nil, errors.New("worktree path cannot be empty")
	}

	info := &WorktreeInfo{Path: path}

	branch, detached, err := GetCurrentBranchOrDetached(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	info.Branch = branch
	info.Detached = detached

	hasChanges, _, err := CheckGitChanges(path)
	if err != nil {
		return nil, fmt.Errorf("failed to check changes: %w", err)
	}
	info.Dirty = hasChanges

	syncStatus := GetSyncStatus(path)
	if syncStatus.Error != nil {
		logger.Debug("Failed to get sync status for %s: %v", path, syncStatus.Error)
	}
	info.Upstream = syncStatus.Upstream
	info.Ahead = syncStatus.Ahead
	info.Behind = syncStatus.Behind
	info.Gone = syncStatus.Gone
	info.NoUpstream = syncStatus.NoUpstream

	info.LastCommitTime = GetLastCommitTime(path)

	return info, nil
}

// GetWorktreeGitDir returns the gitdir path for a worktree.
// Returns ("", nil) if the path is not a worktree (no .git file).
// Returns an error if the .git file exists but is unreadable or malformed.
func GetWorktreeGitDir(worktreePath string) (string, error) {
	gitFile := filepath.Join(worktreePath, ".git")
	content, err := os.ReadFile(gitFile) //nolint:gosec // path derived from validated workspace
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // Not a worktree - expected case
		}
		return "", fmt.Errorf("failed to read .git file: %w", err)
	}

	line := strings.TrimSpace(string(content))
	if !strings.HasPrefix(line, "gitdir:") {
		return "", fmt.Errorf("invalid .git file format: missing gitdir prefix")
	}

	gitdir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if !filepath.IsAbs(gitdir) {
		gitdir = filepath.Join(worktreePath, gitdir)
	}
	return filepath.Clean(gitdir), nil
}

// IsWorktree checks if the given path is a git worktree
func IsWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")
	return fs.FileExists(gitPath)
}

// FindWorktreeRoot walks up from the given path to find the worktree root.
// Returns the path containing the .git file, or error if not in a worktree.
func FindWorktreeRoot(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	dir := absPath
	for i := 0; i < fs.MaxDirectoryIterations; i++ {
		gitPath := filepath.Join(dir, ".git")
		if fs.FileExists(gitPath) {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a worktree")
		}
		dir = parent
	}
	return "", fmt.Errorf("exceeded maximum directory depth (%d): possible symlink loop", fs.MaxDirectoryIterations)
}

// IsWorktreeLocked checks if a worktree is locked.
func IsWorktreeLocked(worktreePath string) bool {
	gitdir, err := GetWorktreeGitDir(worktreePath)
	if err != nil {
		logger.Debug("Failed to get worktree gitdir for lock check: %v", err)
		return false
	}
	if gitdir == "" {
		return false
	}
	lockFile := filepath.Join(gitdir, "locked")
	_, err = os.Stat(lockFile)
	return err == nil
}

// LockWorktree locks a worktree with an optional reason
func LockWorktree(bareDir, worktreePath, reason string) error {
	args := []string{"worktree", "lock"}
	if reason != "" {
		args = append(args, "--reason", reason)
	}
	args = append(args, worktreePath)
	logger.Debug("Executing: git %s in %s", strings.Join(args, " "), bareDir)
	cmd, cancel := GitCommand("git", args...) //nolint:gosec // Worktree path validated
	defer cancel()
	cmd.Dir = bareDir
	return runGitCommand(cmd, true)
}

// GetWorktreeLockReason returns the lock reason for a worktree.
func GetWorktreeLockReason(worktreePath string) string {
	gitdir, err := GetWorktreeGitDir(worktreePath)
	if err != nil {
		logger.Debug("Failed to get worktree gitdir for lock reason: %v", err)
		return ""
	}
	if gitdir == "" {
		return ""
	}
	lockFile := filepath.Join(gitdir, "locked")
	content, err := os.ReadFile(lockFile) //nolint:gosec // path derived from validated workspace
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

// UnlockWorktree unlocks a locked worktree
func UnlockWorktree(bareDir, worktreePath string) error {
	logger.Debug("Executing: git worktree unlock %s in %s", worktreePath, bareDir)
	cmd, cancel := GitCommand("git", "worktree", "unlock", worktreePath) //nolint:gosec // Worktree path validated
	defer cancel()
	cmd.Dir = bareDir
	return runGitCommand(cmd, true)
}

// FindWorktree finds a worktree by name (directory) or branch.
// Matches by worktree directory basename first, then by branch name.
func FindWorktree(infos []*WorktreeInfo, target string) *WorktreeInfo {
	// First try worktree name (directory basename)
	for _, info := range infos {
		if filepath.Base(info.Path) == target {
			return info
		}
	}

	// Fall back to branch name
	for _, info := range infos {
		if info.Branch == target {
			return info
		}
	}

	return nil
}
