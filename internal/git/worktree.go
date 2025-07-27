package git

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/sqve/grove/internal/logger"
)

const gitDir = ".git"

type WorktreeInfo struct {
	// Path is the filesystem path to the worktree directory.
	Path string

	// Branch is the name of the checked out branch (may be empty for detached HEAD).
	Branch string

	// Head is the commit hash of the current HEAD.
	Head string

	IsCurrent bool

	// LastActivity is the timestamp of the last modification to any file in the worktree.
	LastActivity time.Time

	// Status contains detailed working directory status information.
	Status WorktreeStatus

	// Remote contains information about the remote tracking branch.
	Remote RemoteStatus
}

type WorktreeStatus struct {
	// Modified is the count of modified files.
	Modified int

	// Staged is the count of staged files.
	Staged int

	// Untracked is the count of untracked files.
	Untracked int

	// IsClean indicates if the worktree has no modifications.
	IsClean bool
}

type RemoteStatus struct {
	// HasRemote indicates if the branch has a remote tracking branch.
	HasRemote bool

	// Ahead is the number of commits ahead of the remote.
	Ahead int

	// Behind is the number of commits behind the remote.
	Behind int

	// IsMerged indicates if the branch has been merged into the default branch.
	IsMerged bool
}

// Addresses the issue where branch names like "fix/123" would create
// problematic directory structures when using git worktree add directly.
//
// Example: CreateWorktreeWithSafeNaming(executor, "fix/123", "/repo/worktrees")
// Creates directory: /repo/worktrees/fix-123 with branch: fix/123
func CreateWorktreeWithSafeNaming(executor GitExecutor, branchName, basePath string) (string, error) {
	if branchName == "" {
		return "", fmt.Errorf("branch name cannot be empty")
	}

	if basePath == "" {
		return "", fmt.Errorf("base path cannot be empty")
	}

	dirName := BranchToDirectoryName(branchName)
	if dirName == "" {
		return "", fmt.Errorf("could not create valid directory name from branch: %s", branchName)
	}

	worktreePath := filepath.Join(basePath, dirName)

	// Normalize the branch name for Git operations.
	normalizedBranchName := NormalizeBranchName(branchName)
	if normalizedBranchName == "" {
		return "", fmt.Errorf("could not create valid branch name from: %s", branchName)
	}

	// Create the worktree with explicit branch name using -b flag.
	// This ensures the branch name is exactly what we want, not derived from the path.
	_, err := executor.Execute("worktree", "add", "-b", normalizedBranchName, worktreePath)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree for branch %s at %s: %w", normalizedBranchName, worktreePath, err)
	}

	return worktreePath, nil
}

// Unlike CreateWorktreeWithSafeNaming, this doesn't create a new branch but checks out
// an existing one into a filesystem-safe directory.
func CreateWorktreeFromExistingBranch(executor GitExecutor, branchName, basePath string) (string, error) {
	if branchName == "" {
		return "", fmt.Errorf("branch name cannot be empty")
	}

	if basePath == "" {
		return "", fmt.Errorf("base path cannot be empty")
	}

	dirName := BranchToDirectoryName(branchName)
	if dirName == "" {
		return "", fmt.Errorf("could not create valid directory name from branch: %s", branchName)
	}

	worktreePath := filepath.Join(basePath, dirName)

	_, err := executor.Execute("worktree", "add", worktreePath, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree from existing branch %s at %s: %w", branchName, worktreePath, err)
	}

	return worktreePath, nil
}

func RemoveWorktree(executor GitExecutor, worktreePath string) error {
	if worktreePath == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	_, err := executor.Execute("worktree", "remove", worktreePath)
	if err != nil {
		return fmt.Errorf("failed to remove worktree at %s: %w", worktreePath, err)
	}

	return nil
}

// Gathers detailed metadata for each worktree including branch status,
// last activity timestamps, and remote tracking information.
//
// Performance Notes:
// - Activity detection is limited to 3 directory levels for performance
// - Large repositories with many worktrees may take several seconds to process
// - Consider caching results if calling frequently
func ListWorktrees(executor GitExecutor) ([]WorktreeInfo, error) {
	output, err := executor.ExecuteQuiet("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseAndEnhanceWorktrees(executor, output)
}

// Similar to ListWorktrees but operates from a specific repository path.
func ListWorktreesFromRepo(executor GitExecutor, repoPath string) ([]WorktreeInfo, error) {
	output, err := executor.ExecuteQuiet("-C", repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseAndEnhanceWorktrees(executor, output)
}

func parseAndEnhanceWorktrees(executor GitExecutor, output string) ([]WorktreeInfo, error) {
	worktrees, err := parseWorktreePorcelain(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse worktree list: %w", err)
	}

	currentPath, err := getCurrentWorktreePath(executor)
	if err != nil {
		log := logger.WithComponent("worktree_current_path")
		log.Debug("failed to get current worktree path",
			"error", err.Error(),
			"reason", "continuing without current path information")
		currentPath = ""
	}

	for i := range worktrees {
		enhanceWorktreeInfo(executor, &worktrees[i], currentPath)
	}

	return worktrees, nil
}

func enhanceWorktreeInfo(executor GitExecutor, worktree *WorktreeInfo, currentPath string) {
	worktree.IsCurrent = (worktree.Path == currentPath)

	if activity, err := getLastActivity(worktree.Path); err == nil {
		worktree.LastActivity = activity
	}

	if status, err := getWorktreeStatus(executor, worktree.Path); err == nil {
		worktree.Status = status
	}

	if remote, err := getRemoteStatus(executor, worktree.Path, worktree.Branch); err == nil {
		worktree.Remote = remote
	}
}

func ListWorktreesPaths(executor GitExecutor) ([]string, error) {
	worktrees, err := ListWorktrees(executor)
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(worktrees))
	for i := range worktrees {
		paths[i] = worktrees[i].Path
	}

	return paths, nil
}

func parseWorktreePorcelain(output string) ([]WorktreeInfo, error) {
	if output == "" {
		return []WorktreeInfo{}, nil
	}

	var worktrees []WorktreeInfo
	var current WorktreeInfo
	var isBare bool

	lines := splitLines(output)
	for _, line := range lines {
		if line == "" {
			// Empty line indicates end of worktree entry.
			if current.Path != "" && !isBare {
				// Only add non-bare worktrees to the list.
				worktrees = append(worktrees, current)
			}
			current = WorktreeInfo{}
			isBare = false
			continue
		}

		switch {
		case strings.HasPrefix(line, "worktree "):
			current.Path = line[9:] // Remove "worktree " prefix
		case strings.HasPrefix(line, "HEAD "):
			current.Head = line[5:] // Remove "HEAD " prefix
		case strings.HasPrefix(line, "branch "):
			current.Branch = line[7:] // Remove "branch " prefix, keep full reference
		case line == "bare":
			isBare = true
		}
		// Note: Other fields like "detached" can be added here if needed.
	}

	// Add the last worktree if we didn't encounter a trailing empty line.
	if current.Path != "" && !isBare {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

func getCurrentWorktreePath(executor GitExecutor) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	output, err := executor.ExecuteQuiet("-C", cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		log := logger.WithComponent("worktree")
		log.Debug("git rev-parse failed, assuming not in git repository",
			"error", err,
			"directory", cwd,
			"reason", "graceful fallback for non-git directories")
		return "", nil
	}

	return strings.TrimSpace(output), nil
}

// in the worktree directory.
func getLastActivity(worktreePath string) (time.Time, error) {
	const maxDepth = 3
	var lastModTime time.Time
	baseDepth := strings.Count(worktreePath, string(os.PathSeparator))

	err := filepath.Walk(worktreePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files we can't access rather than failing completely.
			return nil
		}

		currentDepth := strings.Count(path, string(os.PathSeparator)) - baseDepth
		if currentDepth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if info.IsDir() && info.Name() == gitDir {
			return filepath.SkipDir
		}

		// Skip hidden files and directories (except .git which we already handled).
		if strings.HasPrefix(info.Name(), ".") && info.Name() != gitDir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common build/cache directories for better performance.
		if info.IsDir() {
			name := info.Name()
			if name == "node_modules" || name == "target" || name == ".next" ||
				name == "dist" || name == "build" || name == "__pycache__" {
				return filepath.SkipDir
			}
		}

		if info.ModTime().After(lastModTime) {
			lastModTime = info.ModTime()
		}

		return nil
	})
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to walk directory %s: %w", worktreePath, err)
	}

	return lastModTime, nil
}

func getWorktreeStatus(executor GitExecutor, worktreePath string) (WorktreeStatus, error) {
	output, err := executor.ExecuteQuiet("-C", worktreePath, "status", "--porcelain")
	if err != nil {
		// Check if this is the specific "must be run in a work tree" error we're trying to fix.
		errStr := err.Error()
		if strings.Contains(errStr, "must be run in a work tree") {
			log := logger.WithComponent("worktree_status")
			log.Debug("git status failed with 'must be run in a work tree' error",
				"path", worktreePath,
				"error", errStr,
				"reason", "returning empty status for invalid worktree")
			// Return empty status for this specific error instead of failing.
			return WorktreeStatus{IsClean: true}, nil
		}

		// For all other errors, return the error as expected.
		return WorktreeStatus{}, fmt.Errorf("failed to get status for worktree %s: %w", worktreePath, err)
	}

	status := WorktreeStatus{}
	lines := splitLines(output)

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		indexStatus := line[0]
		workTreeStatus := line[1]

		if indexStatus != ' ' && indexStatus != '?' {
			status.Staged++
		}

		if workTreeStatus != ' ' && workTreeStatus != '?' {
			status.Modified++
		}

		if indexStatus == '?' && workTreeStatus == '?' {
			status.Untracked++
		}
	}

	status.IsClean = (status.Modified == 0 && status.Staged == 0 && status.Untracked == 0)

	return status, nil
}

func getRemoteStatus(executor GitExecutor, worktreePath, branchRef string) (RemoteStatus, error) {
	if branchRef == "" {
		return RemoteStatus{}, nil
	}

	// Extract branch name from full reference (e.g., refs/heads/main -> main).
	branchName := branchRef
	if strings.HasPrefix(branchRef, "refs/heads/") {
		branchName = branchRef[11:] // Remove "refs/heads/" prefix
	}

	remote := RemoteStatus{}

	// Check if branch has upstream using -C flag to run from worktree directory.
	// Use ExecuteQuiet since branches without upstream are expected and normal.
	upstreamOutput, err := executor.ExecuteQuiet("-C", worktreePath, "rev-parse", "--abbrev-ref", branchName+"@{upstream}")
	if err != nil {
		// Check if this is the specific "must be run in a work tree" error we're trying to fix.
		errStr := err.Error()
		if strings.Contains(errStr, "must be run in a work tree") {
			log := logger.WithComponent("remote_status")
			log.Debug("git rev-parse failed with 'must be run in a work tree' error",
				"path", worktreePath,
				"branch", branchName,
				"error", errStr,
				"reason", "returning empty remote status for invalid worktree")
			// Return empty remote status for this specific error.
			return RemoteStatus{}, nil
		}
		// For all other errors (like no upstream), continue with empty remote (normal behavior).
	}

	if err == nil && strings.TrimSpace(upstreamOutput) != "" {
		remote.HasRemote = true

		countOutput, err := executor.ExecuteQuiet("-C", worktreePath, "rev-list", "--count", "--left-right", branchName+"..."+strings.TrimSpace(upstreamOutput))
		if err == nil {
			parts := strings.Fields(strings.TrimSpace(countOutput))
			if len(parts) == 2 {
				if ahead, err := strconv.Atoi(parts[0]); err == nil {
					remote.Ahead = ahead
				}
				if behind, err := strconv.Atoi(parts[1]); err == nil {
					remote.Behind = behind
				}
			}
		}
	}

	// TODO: Implement merge status detection in a future enhancement.
	// This would require checking if the branch is merged into the default branch.
	remote.IsMerged = false

	return remote, nil
}

// Git reference prefixes used for branch name cleaning.
const (
	refsHeadsPrefix         = "refs/heads/"
	refsRemotesOriginPrefix = "refs/remotes/origin/"
	refsRemotesPrefix       = "refs/remotes/"
)

//
// Supported patterns:.
//   - refs/heads/branch-name -> branch-name (local branches).
//   - refs/remotes/origin/branch-name -> branch-name (origin remote branches).
//   - refs/remotes/upstream/branch-name -> branch-name (other remote branches).
//   - regular branch names remain unchanged.
//
// Examples:.
//
//	CleanBranchName("refs/heads/main") // returns "main".
//	CleanBranchName("refs/remotes/origin/feature/auth") // returns "feature/auth".
//	CleanBranchName("main") // returns "main" (unchanged).
//	CleanBranchName("") // returns "".
func CleanBranchName(branchRef string) string {
	if branchRef == "" {
		return ""
	}

	if strings.HasPrefix(branchRef, refsHeadsPrefix) {
		return branchRef[len(refsHeadsPrefix):]
	}

	if strings.HasPrefix(branchRef, refsRemotesOriginPrefix) {
		return branchRef[len(refsRemotesOriginPrefix):]
	}

	if strings.HasPrefix(branchRef, refsRemotesPrefix) {
		parts := strings.SplitN(branchRef[len(refsRemotesPrefix):], "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}

	return branchRef
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}

	var lines []string
	var current string

	for _, char := range s {
		if char == '\n' {
			lines = append(lines, current)
			current = ""
		} else if char != '\r' {
			current += string(char)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}
