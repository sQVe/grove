package git

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
)

// Regex patterns for parsing git output
var (
	aheadPattern  = regexp.MustCompile(`ahead (\d+)`)
	behindPattern = regexp.MustCompile(`behind (\d+)`)
)

// SyncStatus contains sync information relative to upstream
type SyncStatus struct {
	Upstream   string // Upstream branch name (e.g., "origin/main")
	Ahead      int    // Commits ahead of upstream
	Behind     int    // Commits behind upstream
	Gone       bool   // Upstream branch deleted
	NoUpstream bool   // No upstream configured
	Error      error  // Non-nil if status couldn't be determined due to git error
}

// CheckGitChanges runs git status once and returns both tracked and any changes
func CheckGitChanges(path string) (hasAnyChanges, hasTrackedChanges bool, err error) {
	cmd, cancel := GitCommand("git", "status", "--porcelain")
	defer cancel()
	cmd.Dir = path

	output, err := executeWithOutput(cmd)
	if err != nil {
		logger.Debug("Git status failed: %v", err)
		return false, false, err
	}
	if output == "" {
		logger.Debug("Repository status: clean (no changes)")
		return false, false, nil
	}

	hasAnyChanges = true

	lines := strings.Split(output, "\n")
	changeCount := len(lines)
	for _, line := range lines {
		if line == "" {
			changeCount--
			continue
		}
		if !strings.HasPrefix(line, "??") {
			hasTrackedChanges = true
			break
		}
	}

	logger.Debug("Repository status: %d changes detected, tracked changes: %t", changeCount, hasTrackedChanges)
	return hasAnyChanges, hasTrackedChanges, nil
}

// HasUnresolvedConflicts checks if there are unresolved merge conflicts
func HasUnresolvedConflicts(path string) (bool, error) {
	cmd, cancel := GitCommand("git", "ls-files", "-u")
	defer cancel()
	cmd.Dir = path

	output, err := executeWithOutput(cmd)
	if err != nil {
		return false, err
	}

	return output != "", nil
}

// GetConflictCount returns the number of files with unresolved merge conflicts
func GetConflictCount(path string) (int, error) {
	cmd, cancel := GitCommand("git", "ls-files", "-u")
	defer cancel()
	cmd.Dir = path

	output, err := executeWithOutput(cmd)
	if err != nil {
		return 0, err
	}

	if output == "" {
		return 0, nil
	}

	files := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 4 {
			files[fields[3]] = true
		}
	}

	return len(files), nil
}

// HasOngoingOperation checks for merge/rebase/cherry-pick operations
func HasOngoingOperation(path string) (bool, error) {
	gitDir, err := GetGitDir(path)
	if err != nil {
		return false, err
	}

	markers := []string{
		markerCherryPickHead,
		markerMergeHead,
		markerRevertHead,
		markerRebaseApply,
		markerRebaseMerge,
	}

	for _, marker := range markers {
		if fs.PathExists(filepath.Join(gitDir, marker)) {
			return true, nil
		}
	}

	return false, nil
}

// GetOngoingOperation returns the name of any ongoing git operation, or empty string if none.
// Returns: "merging", "rebasing", "cherry-picking", "reverting", or ""
func GetOngoingOperation(path string) (string, error) {
	gitDir, err := GetGitDir(path)
	if err != nil {
		return "", err
	}

	if fs.PathExists(filepath.Join(gitDir, markerMergeHead)) {
		return "merging", nil
	}
	if fs.PathExists(filepath.Join(gitDir, markerRebaseMerge)) || fs.PathExists(filepath.Join(gitDir, markerRebaseApply)) {
		return "rebasing", nil
	}
	if fs.PathExists(filepath.Join(gitDir, markerCherryPickHead)) {
		return "cherry-picking", nil
	}
	if fs.PathExists(filepath.Join(gitDir, markerRevertHead)) {
		return "reverting", nil
	}

	return "", nil
}

// HasLockFiles checks if there are any active git lock files
func HasLockFiles(path string) (bool, error) {
	gitDir, err := GetGitDir(path)
	if err != nil {
		return false, err
	}

	lockFiles, err := filepath.Glob(filepath.Join(gitDir, "*.lock"))
	if err != nil {
		return false, err
	}

	return len(lockFiles) > 0, nil
}

// HasSubmodules checks if the repository has submodules
func HasSubmodules(path string) (bool, error) {
	// Check for .gitmodules file first, since it is more reliable than git
	// submodule status.
	gitModulesPath := filepath.Join(path, ".gitmodules")
	if fs.FileExists(gitModulesPath) {
		return true, nil
	}

	cmd, cancel := GitCommand("git", "submodule", "status")
	defer cancel()
	cmd.Dir = path

	output, err := executeWithOutput(cmd)
	if err != nil {
		return false, err
	}

	return output != "", nil
}

// HasUnpushedCommits checks if the current branch has unpushed commits
func HasUnpushedCommits(path string) (bool, error) {
	cmdUpstream, cancelUpstream := GitCommand("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	defer cancelUpstream()
	cmdUpstream.Dir = path

	if err := executeWithStderr(cmdUpstream); err != nil {
		return false, fmt.Errorf("%w: %w", ErrNoUpstreamConfigured, err)
	}

	cmdLog, cancelLog := GitCommand("git", "log", "@{u}..HEAD", "--oneline")
	defer cancelLog()
	cmdLog.Dir = path

	output, err := executeWithOutput(cmdLog)
	if err != nil {
		return false, fmt.Errorf("failed to check unpushed commits: %w", err)
	}

	return output != "", nil
}

// GetSyncStatus returns sync status relative to upstream.
// Uses git for-each-ref to reliably detect "gone" upstream branches.
// Check the Error field to distinguish "no upstream" from "git command failed".
func GetSyncStatus(path string) *SyncStatus {
	status := &SyncStatus{}

	// Get current branch name
	cmdBranch, cancelBranch := GitCommand("git", "rev-parse", "--abbrev-ref", "HEAD") // nolint:gosec
	cmdBranch.Dir = path
	var branchOut bytes.Buffer
	cmdBranch.Stdout = &branchOut
	err := cmdBranch.Run()
	cancelBranch()
	if err != nil {
		status.NoUpstream = true
		status.Error = fmt.Errorf("failed to get current branch: %w", err)
		return status
	}
	branch := strings.TrimSpace(branchOut.String())
	if branch == "" || branch == "HEAD" {
		// Detached HEAD - expected condition, not an error
		status.NoUpstream = true
		return status
	}

	// Use for-each-ref to get upstream info and track status in one command
	// Format: "upstream_ref track_info" e.g. "refs/remotes/origin/main [ahead 1, behind 2]" or "refs/remotes/origin/gone [gone]"
	cmdRef, cancelRef := GitCommand("git", "for-each-ref", "--format=%(upstream) %(upstream:track)", fmt.Sprintf("refs/heads/%s", branch)) // nolint:gosec
	defer cancelRef()
	cmdRef.Dir = path
	var refOut bytes.Buffer
	cmdRef.Stdout = &refOut
	if err := cmdRef.Run(); err != nil {
		status.NoUpstream = true
		status.Error = fmt.Errorf("failed to get upstream info: %w", err)
		return status
	}

	output := strings.TrimSpace(refOut.String())
	if output == "" {
		status.NoUpstream = true
		return status
	}

	// Parse upstream ref (first space-separated field)
	parts := strings.SplitN(output, " ", 2)
	upstreamRef := parts[0]
	if upstreamRef == "" {
		status.NoUpstream = true
		return status
	}

	// Convert refs/remotes/origin/main to origin/main
	status.Upstream = strings.TrimPrefix(upstreamRef, "refs/remotes/")

	// Check for track info
	if len(parts) > 1 {
		trackInfo := parts[1]
		if strings.Contains(trackInfo, "[gone]") {
			status.Gone = true
			return status
		}
		// Parse ahead/behind from track info like "[ahead 1, behind 2]" or "[ahead 1]" or "[behind 2]"
		if match := aheadPattern.FindStringSubmatch(trackInfo); match != nil {
			status.Ahead, _ = strconv.Atoi(match[1])
		}
		if match := behindPattern.FindStringSubmatch(trackInfo); match != nil {
			status.Behind, _ = strconv.Atoi(match[1])
		}
	}

	return status
}

// GetLastCommitTime returns the Unix timestamp of the last commit in a repository.
// Returns 0 if the repository has no commits or on error.
func GetLastCommitTime(path string) int64 {
	cmd, cancel := GitCommand("git", "log", "-1", "--format=%ct", "HEAD")
	defer cancel()
	cmd.Dir = path

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return 0
	}

	var timestamp int64
	if _, err := fmt.Sscanf(strings.TrimSpace(out.String()), "%d", &timestamp); err != nil {
		return 0
	}

	return timestamp
}

// GetStashCount returns the number of stashes in a repository
func GetStashCount(path string) (int, error) {
	cmd, cancel := GitCommand("git", "stash", "list")
	defer cancel()
	cmd.Dir = path

	output, err := executeWithOutput(cmd)
	if err != nil {
		return 0, err
	}

	if output == "" {
		return 0, nil
	}

	count := 0
	for _, line := range strings.Split(output, "\n") {
		if line != "" {
			count++
		}
	}
	return count, nil
}
