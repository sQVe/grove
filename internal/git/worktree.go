package git

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

// WorktreeInfo contains status information about a worktree
type WorktreeInfo struct {
	Path           string `json:"path"`                  // Absolute path to worktree
	Branch         string `json:"branch"`                // Branch name (or commit hash if detached)
	Upstream       string `json:"upstream,omitempty"`    // Upstream branch name (e.g., "origin/main")
	Dirty          bool   `json:"dirty"`                 // Has uncommitted changes
	Ahead          int    `json:"ahead"`                 // Commits ahead of upstream
	Behind         int    `json:"behind"`                // Commits behind upstream
	Gone           bool   `json:"gone"`                  // Upstream branch deleted
	NoUpstream     bool   `json:"no_upstream"`           // No upstream configured
	Locked         bool   `json:"locked"`                // Worktree is locked
	LockReason     string `json:"lock_reason,omitempty"` // Reason for lock (empty if not locked)
	LastCommitTime int64  `json:"-"`                     // Unix timestamp of last commit (0 if unknown)
	Detached       bool   `json:"detached"`              // Worktree is in detached HEAD state
	Prunable       bool   `json:"-"`                     // Git marks the worktree metadata as prunable
}

type worktreeListEntry struct {
	Path       string
	Branch     string
	Detached   bool
	Locked     bool
	LockReason string
	Prunable   bool
}

const (
	gitWorktreeSubcommand = "worktree"
	detachedBranch        = "(detached)"
)

// CreateWorktreeOptions configures worktree creation.
type CreateWorktreeOptions struct {
	Branch     string
	NewBranch  bool
	Base       string
	Detach     bool
	NoCheckout bool
}

// CreateWorktree creates a worktree from a bare repository.
func CreateWorktree(bareRepo, worktreePath string, opts CreateWorktreeOptions, quiet bool) error {
	if bareRepo == "" {
		return errors.New("bare repository path cannot be empty")
	}
	if worktreePath == "" {
		return errors.New("worktree path cannot be empty")
	}
	if opts.Branch == "" && opts.Detach {
		return errors.New("ref cannot be empty")
	}
	if opts.Branch == "" {
		return errors.New("branch name cannot be empty")
	}

	args := createWorktreeArgs(worktreePath, opts)
	logger.Debug("Executing: git %s", strings.Join(args, " "))
	cmd, cancel := GitCommand("git", args...)
	defer cancel()
	cmd.Dir = bareRepo

	return WrapGitTooOldError(runGitCommand(cmd, quiet))
}

func createWorktreeArgs(worktreePath string, opts CreateWorktreeOptions) []string {
	args := []string{gitWorktreeSubcommand, "add", "--relative-paths"}
	if opts.NewBranch {
		args = append(args, "-b", opts.Branch)
	}
	if opts.Detach {
		args = append(args, "--detach")
	}
	if opts.NoCheckout {
		args = append(args, "--no-checkout")
	}
	args = append(args, worktreePath)
	if opts.Base != "" {
		args = append(args, opts.Base)
	} else if !opts.NewBranch {
		args = append(args, opts.Branch)
	}
	return args
}

// RemoveWorktree removes a worktree directory
func RemoveWorktree(bareDir, worktreePath string, force bool) error {
	args := []string{gitWorktreeSubcommand, "remove", worktreePath}
	if force {
		args = append(args, "--force")
	}
	logger.Debug("Executing: git %s in %s", strings.Join(args, " "), bareDir)
	cmd, cancel := GitCommand("git", args...) // nolint:gosec // Worktree path comes from git worktree list
	defer cancel()
	cmd.Dir = bareDir
	return runGitCommand(cmd, true)
}

// PruneWorktrees removes git-prunable worktree metadata.
func PruneWorktrees(bareDir string) error {
	if bareDir == "" {
		return errors.New("bare directory path cannot be empty")
	}

	logger.Debug("Executing: git worktree prune in %s", bareDir)
	cmd, cancel := GitCommand("git", gitWorktreeSubcommand, "prune")
	defer cancel()
	cmd.Dir = bareDir
	return runGitCommand(cmd, true)
}

// RepairWorktree runs git worktree repair to fix worktree paths after directory moves.
func RepairWorktree(bareDir, worktreePath string) error {
	if bareDir == "" {
		return errors.New("bare directory path cannot be empty")
	}

	args := []string{gitWorktreeSubcommand, "repair"}
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
	entries, err := listWorktreeEntries(repoPath)
	if err != nil {
		return nil, err
	}

	worktrees := make([]string, 0, len(entries))
	for _, entry := range entries {
		worktrees = append(worktrees, entry.Path)
	}
	return worktrees, nil
}

func listWorktreeEntries(repoPath string) ([]worktreeListEntry, error) {
	logger.Debug("Executing: git worktree list --porcelain -z in %s", repoPath)
	cmd, cancel := GitCommand("git", gitWorktreeSubcommand, "list", "--porcelain", "-z")
	defer cancel()
	cmd.Dir = repoPath

	out, err := executeWithOutputBuffer(cmd)
	if err != nil {
		return nil, err
	}

	return parseWorktreeListPorcelain(out, repoPath)
}

func parseWorktreeListPorcelain(r io.Reader, repoPath string) ([]worktreeListEntry, error) {
	absRepoPath, err := filepath.Abs(repoPath)
	if err != nil {
		return nil, err
	}

	var entries []worktreeListEntry
	var entry worktreeListEntry

	flush := func() error {
		if entry.Path == "" {
			return nil
		}

		worktreePath := filepath.FromSlash(entry.Path)
		absWorktreePath, err := filepath.Abs(worktreePath)
		if err != nil {
			return err
		}
		if !fs.PathsEqual(absWorktreePath, absRepoPath) {
			entry.Path = absWorktreePath
			entries = append(entries, entry)
		}
		entry = worktreeListEntry{}
		return nil
	}

	output, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}
	for _, field := range bytes.Split(output, []byte{0}) {
		line := string(field)
		if line == "" {
			if err := flush(); err != nil {
				return nil, err
			}
			continue
		}

		switch {
		case strings.HasPrefix(line, "worktree "):
			entry.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			entry.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "detached":
			entry.Detached = true
			entry.Branch = detachedBranch
		case line == "locked" || strings.HasPrefix(line, "locked "):
			entry.Locked = true
			entry.LockReason = strings.TrimSpace(strings.TrimPrefix(line, "locked"))
		case line == "prunable" || strings.HasPrefix(line, "prunable "):
			entry.Prunable = true
		}
	}
	if err := flush(); err != nil {
		return nil, err
	}

	return entries, nil
}

// worktreeFallbackInfo builds a WorktreeInfo for an entry whose status read
// failed. A detached HEAD or a locked entry is surfaced; anything else is a
// genuinely corrupt admin entry (gitdir present but unreadable) and is skipped
// with a warning. The bool is false when the caller should skip the entry.
func worktreeFallbackInfo(path string, entry worktreeListEntry, err error) (*WorktreeInfo, bool) {
	switch {
	case errors.Is(err, ErrDetachedHead):
		return &WorktreeInfo{Path: path, Branch: detachedBranch, Detached: true}, true
	case entry.Locked:
		return &WorktreeInfo{Path: path, Branch: entry.Branch, Detached: entry.Detached}, true
	default:
		logger.Warning("Skipping worktree %s (may be corrupted): %v", path, err)
		return nil, false
	}
}

// ListWorktreesWithInfo returns info for all worktrees in a grove workspace.
func ListWorktreesWithInfo(bareDir string, fast bool) ([]*WorktreeInfo, error) {
	entries, err := listWorktreeEntries(bareDir)
	if err != nil {
		return nil, err
	}

	var infos []*WorktreeInfo
	for _, entry := range entries {
		path := entry.Path
		var info *WorktreeInfo
		// git-prunable entries have no backing path, so they are not usable
		// worktrees for callers that switch/exec/add against them. Only
		// `grove prune` acts on them, via ListPrunableWorktrees.
		if entry.Prunable {
			continue
		}

		// Both modes validate the worktree by reading its HEAD, so a registered
		// entry whose gitdir is present but unreadable is skipped as corrupt
		// rather than returned as usable. Fast mode reads only the branch;
		// full mode also collects status.
		if fast {
			branch, detached, err := GetCurrentBranchOrDetached(path)
			if err != nil {
				var ok bool
				if info, ok = worktreeFallbackInfo(path, entry, err); !ok {
					continue
				}
			} else {
				info = &WorktreeInfo{Path: path, Branch: branch, Detached: detached}
			}
		} else {
			var err error
			if info, err = GetWorktreeInfo(path); err != nil {
				var ok bool
				if info, ok = worktreeFallbackInfo(path, entry, err); !ok {
					continue
				}
			}
		}

		info.Locked = entry.Locked
		info.LockReason = entry.LockReason

		infos = append(infos, info)
	}

	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Branch < infos[j].Branch
	})

	return infos, nil
}

// ListPrunableWorktrees returns the worktrees git has marked prunable, i.e.
// their backing path is gone. These are excluded from ListWorktreesWithInfo
// because they are not usable worktrees; only `grove prune` acts on them.
func ListPrunableWorktrees(bareDir string) ([]*WorktreeInfo, error) {
	if bareDir == "" {
		return nil, errors.New("bare directory path cannot be empty")
	}

	entries, err := listWorktreeEntries(bareDir)
	if err != nil {
		return nil, err
	}

	var infos []*WorktreeInfo
	for _, entry := range entries {
		if !entry.Prunable {
			continue
		}
		infos = append(infos, &WorktreeInfo{
			Path:       entry.Path,
			Branch:     entry.Branch,
			Detached:   entry.Detached,
			Locked:     entry.Locked,
			LockReason: entry.LockReason,
			Prunable:   true,
		})
	}
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

// IsWorktree checks if the given path is a git worktree
func IsWorktree(path string) bool {
	gitPath := filepath.Join(path, ".git")
	return fs.FileExists(gitPath)
}

// FindWorktreeRoot walks up from the given path to find the worktree root.
// Returns the path containing the .git file, or error if not in a worktree.
func FindWorktreeRoot(startPath string) (string, error) {
	dir, err := fs.WalkUp(startPath, func(dir string) bool {
		return fs.FileExists(filepath.Join(dir, ".git"))
	})
	if err != nil {
		return "", err
	}
	if dir == "" {
		return "", fmt.Errorf("not in a worktree")
	}
	return dir, nil
}

// IsWorktreeLocked checks if a worktree is locked.
func IsWorktreeLocked(worktreePath string) bool {
	gitdir, err := GetGitDir(worktreePath)
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
	gitdir, err := GetGitDir(worktreePath)
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
