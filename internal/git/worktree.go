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

// WorktreeInfo contains comprehensive information about a Git worktree.
type WorktreeInfo struct {
	// Path is the filesystem path to the worktree directory
	Path string

	// Branch is the name of the checked out branch (may be empty for detached HEAD)
	Branch string

	// Head is the commit hash of the current HEAD
	Head string

	// IsCurrent indicates if this is the current worktree (where we're running from)
	IsCurrent bool

	// LastActivity is the timestamp of the last modification to any file in the worktree
	LastActivity time.Time

	// Status contains detailed working directory status information
	Status WorktreeStatus

	// Remote contains information about the remote tracking branch
	Remote RemoteStatus
}

// WorktreeStatus contains detailed status information about a worktree's files.
type WorktreeStatus struct {
	// Modified is the count of modified files
	Modified int

	// Staged is the count of staged files
	Staged int

	// Untracked is the count of untracked files
	Untracked int

	// IsClean indicates if the worktree has no modifications
	IsClean bool
}

// RemoteStatus contains information about the remote tracking branch.
type RemoteStatus struct {
	// HasRemote indicates if the branch has a remote tracking branch
	HasRemote bool

	// Ahead is the number of commits ahead of the remote
	Ahead int

	// Behind is the number of commits behind the remote
	Behind int

	// IsMerged indicates if the branch has been merged into the default branch
	IsMerged bool
}

// CreateWorktreeWithSafeNaming creates a new worktree with filesystem-safe directory naming
// while preserving the original branch name for Git operations.
//
// This function addresses the issue where branch names like "fix/123" would create
// problematic directory structures or incorrect branch names when using git worktree add directly.
//
// Parameters:
//   - executor: GitExecutor interface for running git commands
//   - branchName: The desired branch name (e.g., "fix/123")
//   - basePath: The base directory where worktrees should be created
//
// Returns:
//   - worktreePath: The full path to the created worktree directory
//   - error: Any error encountered during worktree creation
//
// Example:
//
//	path, err := CreateWorktreeWithSafeNaming(executor, "fix/123", "/repo/worktrees")
//	// Creates directory: /repo/worktrees/fix-123
//	// Creates branch: fix/123
func CreateWorktreeWithSafeNaming(executor GitExecutor, branchName, basePath string) (string, error) {
	if branchName == "" {
		return "", fmt.Errorf("branch name cannot be empty")
	}

	if basePath == "" {
		return "", fmt.Errorf("base path cannot be empty")
	}

	// Convert branch name to filesystem-safe directory name
	dirName := BranchToDirectoryName(branchName)
	if dirName == "" {
		return "", fmt.Errorf("could not create valid directory name from branch: %s", branchName)
	}

	// Create the full worktree path
	worktreePath := filepath.Join(basePath, dirName)

	// Normalize the branch name for Git operations
	normalizedBranchName := NormalizeBranchName(branchName)
	if normalizedBranchName == "" {
		return "", fmt.Errorf("could not create valid branch name from: %s", branchName)
	}

	// Create the worktree with explicit branch name using -b flag
	// This ensures the branch name is exactly what we want, not derived from the path
	_, err := executor.Execute("worktree", "add", "-b", normalizedBranchName, worktreePath)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree for branch %s at %s: %w", normalizedBranchName, worktreePath, err)
	}

	return worktreePath, nil
}

// CreateWorktreeFromExistingBranch creates a worktree from an existing branch with safe naming.
// Unlike CreateWorktreeWithSafeNaming, this function doesn't create a new branch but checks out
// an existing one into a filesystem-safe directory.
//
// Parameters:
//   - executor: GitExecutor interface for running git commands
//   - branchName: The existing branch name to check out
//   - basePath: The base directory where worktrees should be created
//
// Returns:
//   - worktreePath: The full path to the created worktree directory
//   - error: Any error encountered during worktree creation
func CreateWorktreeFromExistingBranch(executor GitExecutor, branchName, basePath string) (string, error) {
	if branchName == "" {
		return "", fmt.Errorf("branch name cannot be empty")
	}

	if basePath == "" {
		return "", fmt.Errorf("base path cannot be empty")
	}

	// Convert branch name to filesystem-safe directory name
	dirName := BranchToDirectoryName(branchName)
	if dirName == "" {
		return "", fmt.Errorf("could not create valid directory name from branch: %s", branchName)
	}

	// Create the full worktree path
	worktreePath := filepath.Join(basePath, dirName)

	// Create the worktree from existing branch
	_, err := executor.Execute("worktree", "add", worktreePath, branchName)
	if err != nil {
		return "", fmt.Errorf("failed to create worktree from existing branch %s at %s: %w", branchName, worktreePath, err)
	}

	return worktreePath, nil
}

// RemoveWorktree removes a worktree directory and its Git worktree registration.
//
// Parameters:
//   - executor: GitExecutor interface for running git commands
//   - worktreePath: The path to the worktree directory to remove
//
// Returns:
//   - error: Any error encountered during worktree removal
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

// ListWorktrees returns comprehensive information about all worktrees in the repository.
// This function attempts to find the repository automatically.
//
// This function gathers detailed metadata for each worktree including branch status,
// last activity timestamps, and remote tracking information.
//
// Performance Notes:
//   - Activity detection is limited to 3 directory levels for performance
//   - Large repositories with many worktrees may take several seconds to process
//   - Consider caching results if calling frequently
//
// Parameters:
//   - executor: GitExecutor interface for running git commands
//
// Returns:
//   - []WorktreeInfo: List of worktree information
//   - error: Any error encountered during worktree listing
func ListWorktrees(executor GitExecutor) ([]WorktreeInfo, error) {
	output, err := executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseAndEnhanceWorktrees(executor, output)
}

// ListWorktreesFromRepo returns comprehensive information about all worktrees in a specific repository.
//
// This function gathers detailed metadata for each worktree including branch status,
// last activity timestamps, and remote tracking information.
//
// Performance Notes:
//   - Activity detection is limited to 3 directory levels for performance
//   - Large repositories with many worktrees may take several seconds to process
//   - Consider caching results if calling frequently
//
// Parameters:
//   - executor: GitExecutor interface for running git commands
//   - repoPath: Path to the repository (typically the .bare directory)
//
// Returns:
//   - []WorktreeInfo: List of worktree information
//   - error: Any error encountered during worktree listing
func ListWorktreesFromRepo(executor GitExecutor, repoPath string) ([]WorktreeInfo, error) {
	output, err := executor.Execute("-C", repoPath, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseAndEnhanceWorktrees(executor, output)
}

// parseAndEnhanceWorktrees parses worktree output and enhances with additional information.
func parseAndEnhanceWorktrees(executor GitExecutor, output string) ([]WorktreeInfo, error) {
	worktrees, err := parseWorktreePorcelain(output)
	if err != nil {
		return nil, fmt.Errorf("failed to parse worktree list: %w", err)
	}

	// Get current worktree path for comparison
	currentPath, err := getCurrentWorktreePath(executor)
	if err != nil {
		// Don't fail the entire operation if we can't determine current path
		currentPath = ""
	}

	// Enhance each worktree with additional information
	for i := range worktrees {
		enhanceWorktreeInfo(executor, &worktrees[i], currentPath)
	}

	return worktrees, nil
}

// enhanceWorktreeInfo adds detailed information to a worktree info structure.
func enhanceWorktreeInfo(executor GitExecutor, worktree *WorktreeInfo, currentPath string) {
	// Mark current worktree
	worktree.IsCurrent = (worktree.Path == currentPath)

	// Get last activity timestamp
	if activity, err := getLastActivity(worktree.Path); err == nil {
		worktree.LastActivity = activity
	}

	// Get detailed status information
	if status, err := getWorktreeStatus(executor, worktree.Path); err == nil {
		worktree.Status = status
	}

	// Get remote tracking information
	if remote, err := getRemoteStatus(executor, worktree.Path, worktree.Branch); err == nil {
		worktree.Remote = remote
	}
}

// ListWorktreesPaths returns a simple list of worktree paths for backward compatibility.
//
// Parameters:
//   - executor: GitExecutor interface for running git commands
//
// Returns:
//   - []string: List of worktree paths
//   - error: Any error encountered during worktree listing
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

// parseWorktreePorcelain parses the output of 'git worktree list --porcelain'
// and returns basic WorktreeInfo structures.
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
			// Empty line indicates end of worktree entry
			if current.Path != "" && !isBare {
				// Only add non-bare worktrees to the list
				worktrees = append(worktrees, current)
			}
			current = WorktreeInfo{}
			isBare = false
			continue
		}

		// Parse different line types
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
		// Note: Other fields like "detached" can be added here if needed
	}

	// Add the last worktree if we didn't encounter a trailing empty line
	if current.Path != "" && !isBare {
		worktrees = append(worktrees, current)
	}

	return worktrees, nil
}

// getCurrentWorktreePath returns the path of the current worktree.
func getCurrentWorktreePath(executor GitExecutor) (string, error) {
	// Try to get the current working directory first
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	// Check if the current directory is a git worktree
	output, err := executor.Execute("-C", cwd, "rev-parse", "--show-toplevel")
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

// getLastActivity returns the timestamp of the most recent file modification
// in the worktree directory.
func getLastActivity(worktreePath string) (time.Time, error) {
	const maxDepth = 3
	var lastModTime time.Time
	baseDepth := strings.Count(worktreePath, string(os.PathSeparator))

	err := filepath.Walk(worktreePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			// Skip files we can't access rather than failing completely
			return nil
		}

		// Calculate current depth relative to the base path
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

		// Skip hidden files and directories (except .git which we already handled)
		if strings.HasPrefix(info.Name(), ".") && info.Name() != gitDir {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip common build/cache directories for better performance
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

// getWorktreeStatus returns detailed status information for a worktree.
func getWorktreeStatus(executor GitExecutor, worktreePath string) (WorktreeStatus, error) {
	// Verify the worktree directory exists and is accessible
	if _, err := os.Stat(worktreePath); err != nil {
		return WorktreeStatus{}, fmt.Errorf("worktree directory %s is not accessible: %w", worktreePath, err)
	}

	// Get git status in porcelain format using -C flag to run from worktree directory
	output, err := executor.Execute("-C", worktreePath, "status", "--porcelain")
	if err != nil {
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

		// Count staged files (index changes)
		if indexStatus != ' ' && indexStatus != '?' {
			status.Staged++
		}

		// Count modified files (working tree changes)
		if workTreeStatus != ' ' && workTreeStatus != '?' {
			status.Modified++
		}

		// Count untracked files
		if indexStatus == '?' && workTreeStatus == '?' {
			status.Untracked++
		}
	}

	status.IsClean = (status.Modified == 0 && status.Staged == 0 && status.Untracked == 0)

	return status, nil
}

// getRemoteStatus returns remote tracking information for a branch.
func getRemoteStatus(executor GitExecutor, worktreePath, branchRef string) (RemoteStatus, error) {
	if branchRef == "" {
		return RemoteStatus{}, nil
	}

	// Extract branch name from full reference (e.g., refs/heads/main -> main)
	branchName := branchRef
	if strings.HasPrefix(branchRef, "refs/heads/") {
		branchName = branchRef[11:] // Remove "refs/heads/" prefix
	}

	// Verify the worktree directory exists and is accessible
	if _, err := os.Stat(worktreePath); err != nil {
		return RemoteStatus{}, fmt.Errorf("worktree directory %s is not accessible: %w", worktreePath, err)
	}

	remote := RemoteStatus{}

	// Check if branch has upstream using -C flag to run from worktree directory
	// Use ExecuteQuiet since branches without upstream are expected and normal
	upstreamOutput, err := executor.ExecuteQuiet("-C", worktreePath, "rev-parse", "--abbrev-ref", branchName+"@{upstream}")
	if err == nil && strings.TrimSpace(upstreamOutput) != "" {
		remote.HasRemote = true

		// Get ahead/behind counts
		countOutput, err := executor.Execute("-C", worktreePath, "rev-list", "--count", "--left-right", branchName+"..."+strings.TrimSpace(upstreamOutput))
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

	// TODO: Implement merge status detection in a future enhancement
	// This would require checking if the branch is merged into the default branch
	remote.IsMerged = false

	return remote, nil
}

// Git reference prefixes used for branch name cleaning.
const (
	refsHeadsPrefix         = "refs/heads/"
	refsRemotesOriginPrefix = "refs/remotes/origin/"
	refsRemotesPrefix       = "refs/remotes/"
)

// CleanBranchName removes common git reference prefixes to provide a clean branch name for display.
// This function handles common git reference patterns and returns clean branch names suitable for user display.
//
// Supported patterns:
//   - refs/heads/branch-name -> branch-name (local branches)
//   - refs/remotes/origin/branch-name -> branch-name (origin remote branches)
//   - refs/remotes/upstream/branch-name -> branch-name (other remote branches)
//   - regular branch names remain unchanged
//
// Examples:
//
//	CleanBranchName("refs/heads/main") // returns "main"
//	CleanBranchName("refs/remotes/origin/feature/auth") // returns "feature/auth"
//	CleanBranchName("main") // returns "main" (unchanged)
//	CleanBranchName("") // returns ""
func CleanBranchName(branchRef string) string {
	if branchRef == "" {
		return ""
	}

	// Remove refs/heads/ prefix (local branches)
	if strings.HasPrefix(branchRef, refsHeadsPrefix) {
		return branchRef[len(refsHeadsPrefix):]
	}

	// Remove refs/remotes/origin/ prefix (remote tracking branches)
	if strings.HasPrefix(branchRef, refsRemotesOriginPrefix) {
		return branchRef[len(refsRemotesOriginPrefix):]
	}

	// Remove refs/remotes/<remote>/ prefix (general remote tracking branches)
	if strings.HasPrefix(branchRef, refsRemotesPrefix) {
		parts := strings.SplitN(branchRef[len(refsRemotesPrefix):], "/", 2)
		if len(parts) == 2 {
			return parts[1]
		}
	}

	// Return the original name if no known prefixes match
	return branchRef
}

// splitLines splits a string into lines, handling different line endings.
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
