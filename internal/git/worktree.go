package git

import (
	"fmt"
	"path/filepath"
)

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

// ListWorktrees returns a list of all worktrees in the repository.
//
// Parameters:
//   - executor: GitExecutor interface for running git commands
//
// Returns:
//   - []string: List of worktree paths
//   - error: Any error encountered during worktree listing
func ListWorktrees(executor GitExecutor) ([]string, error) {
	output, err := executor.Execute("worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Parse the porcelain output to extract worktree paths
	// The format is: "worktree <path>" for each worktree
	var worktrees []string
	lines := splitLines(output)

	for _, line := range lines {
		if len(line) > 9 && line[:9] == "worktree " {
			worktreePath := line[9:]
			worktrees = append(worktrees, worktreePath)
		}
	}

	return worktrees, nil
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
			if current != "" {
				lines = append(lines, current)
				current = ""
			}
		} else if char != '\r' {
			current += string(char)
		}
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}
