package create

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
)

// FileManagerImpl implements the FileManager interface for copying files between worktrees.
type FileManagerImpl struct {
	executor git.GitExecutor
}

// NewFileManager creates a new FileManager with the provided GitExecutor.
func NewFileManager(executor git.GitExecutor) *FileManagerImpl {
	return &FileManagerImpl{
		executor: executor,
	}
}

// CopyFiles copies files matching the provided patterns from source to target worktree.
func (f *FileManagerImpl) CopyFiles(sourceWorktree, targetWorktree string, patterns []string, options CopyOptions) error {
	if sourceWorktree == "" {
		return fmt.Errorf("source worktree path cannot be empty")
	}

	if targetWorktree == "" {
		return fmt.Errorf("target worktree path cannot be empty")
	}

	if len(patterns) == 0 {
		return nil
	}

	if err := f.validateWorktreePath(sourceWorktree); err != nil {
		return fmt.Errorf("invalid source worktree: %w", err)
	}

	if err := f.validateWorktreePath(targetWorktree); err != nil {
		return fmt.Errorf("invalid target worktree: %w", err)
	}

	var conflicts []FileConflict
	var copiedCount int

	for _, pattern := range patterns {
		matches, err := f.findMatchingFiles(sourceWorktree, pattern)
		if err != nil {
			return fmt.Errorf("failed to find files matching pattern %s: %w", pattern, err)
		}

		for _, match := range matches {
			sourcePath := filepath.Join(sourceWorktree, match)
			targetPath := filepath.Join(targetWorktree, match)

			if _, err := os.Stat(targetPath); err == nil {
				conflicts = append(conflicts, FileConflict{
					Path:       match,
					SourcePath: sourcePath,
					TargetPath: targetPath,
				})
				continue
			}

			if !options.DryRun {
				if err := f.copyFile(sourcePath, targetPath); err != nil {
					return fmt.Errorf("failed to copy file %s: %w", match, err)
				}
			}
			copiedCount++
		}
	}

	if len(conflicts) > 0 {
		if err := f.ResolveConflicts(conflicts, options.ConflictStrategy); err != nil {
			return fmt.Errorf("failed to resolve file conflicts: %w", err)
		}
	}

	return nil
}

// GetCurrentWorktreePath returns the path of the current worktree.
func (f *FileManagerImpl) GetCurrentWorktreePath() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	output, err := f.executor.ExecuteQuiet("-C", cwd, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", groveErrors.ErrGitOperation("rev-parse --show-toplevel", err)
	}

	return strings.TrimSpace(output), nil
}

// DiscoverSourceWorktree attempts to find the default branch's worktree for file copying.
func (f *FileManagerImpl) DiscoverSourceWorktree() (string, error) {
	// First, try to detect the default branch with any available remote
	remoteName := f.getFirstRemote()
	if remoteName != "" {
		defaultBranch, err := git.DetectDefaultBranch(f.executor, remoteName)
		if err == nil {
			// Look for a worktree that has the default branch
			worktreePath, err := f.FindWorktreeByBranch(defaultBranch)
			if err == nil && worktreePath != "" {
				return worktreePath, nil
			}
		}
	}

	// If no remote or detection failed, try to find the current HEAD branch
	currentBranch, err := f.getCurrentBranch()
	if err == nil && currentBranch != "" {
		worktreePath, err := f.FindWorktreeByBranch(currentBranch)
		if err == nil && worktreePath != "" {
			return worktreePath, nil
		}
	}

	// If all else fails, fall back to common paths
	return f.discoverSourceWorktreeFallback()
}

// getFirstRemote returns the name of the first available remote, or empty string if none
func (f *FileManagerImpl) getFirstRemote() string {
	output, err := f.executor.ExecuteQuiet("remote")
	if err != nil {
		return ""
	}
	
	remotes := strings.Split(strings.TrimSpace(output), "\n")
	if len(remotes) > 0 && strings.TrimSpace(remotes[0]) != "" {
		return strings.TrimSpace(remotes[0])
	}
	return ""
}

// getCurrentBranch returns the name of the current branch
func (f *FileManagerImpl) getCurrentBranch() (string, error) {
	output, err := f.executor.ExecuteQuiet("branch", "--show-current")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

// discoverSourceWorktreeFallback uses the original logic as fallback
func (f *FileManagerImpl) discoverSourceWorktreeFallback() (string, error) {
	// Check if we're in a bare repository first
	isBare, err := f.executor.ExecuteQuiet("rev-parse", "--is-bare-repository")
	if err != nil {
		return "", groveErrors.ErrGitOperation("rev-parse --is-bare-repository", err)
	}
	
	var repoRoot string
	if strings.TrimSpace(isBare) == "true" {
		// In bare repository, get the git directory path
		gitDir, err := f.executor.ExecuteQuiet("rev-parse", "--git-dir")
		if err != nil {
			return "", groveErrors.ErrGitOperation("rev-parse --git-dir", err)
		}
		repoRoot = strings.TrimSpace(gitDir)
	} else {
		// In regular repository, get the working tree root
		root, err := f.executor.ExecuteQuiet("rev-parse", "--show-toplevel")
		if err != nil {
			return "", groveErrors.ErrGitOperation("rev-parse --show-toplevel", err)
		}
		repoRoot = strings.TrimSpace(root)
	}

	worktreeList, err := f.executor.ExecuteQuiet("worktree", "list", "--porcelain")
	if err != nil {
		return "", groveErrors.ErrGitOperation("worktree list --porcelain", err)
	}

	lines := strings.Split(strings.TrimSpace(worktreeList), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "worktree ") {
			worktreePath := strings.TrimPrefix(line, "worktree ")

			if worktreePath == repoRoot {
				return worktreePath, nil
			}

			for j := i + 1; j < len(lines) && !strings.HasPrefix(lines[j], "worktree "); j++ {
				if strings.TrimSpace(lines[j]) == "bare" {
					break
				}
			}
		}
	}

	commonPaths := []string{
		filepath.Join(filepath.Dir(repoRoot), "main"),
		filepath.Join(filepath.Dir(repoRoot), "master"),
		repoRoot, // Fallback to repo root.
	}

	for _, path := range commonPaths {
		if err := f.validateWorktreePath(path); err == nil {
			return path, nil
		}
	}

	return "", groveErrors.ErrSourceWorktreeNotFound("")
}

// FindWorktreeByBranch finds the worktree path for a specific branch name.
func (f *FileManagerImpl) FindWorktreeByBranch(branchName string) (string, error) {
	worktreeList, err := f.executor.ExecuteQuiet("worktree", "list", "--porcelain")
	if err != nil {
		return "", groveErrors.ErrGitOperation("worktree list --porcelain", err)
	}

	lines := strings.Split(strings.TrimSpace(worktreeList), "\n")
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "worktree ") {
			worktreePath := strings.TrimPrefix(line, "worktree ")
			
			// Look ahead for the branch information
			for j := i + 1; j < len(lines) && !strings.HasPrefix(lines[j], "worktree "); j++ {
				branchLine := strings.TrimSpace(lines[j])
				if strings.HasPrefix(branchLine, "branch refs/heads/") {
					currentBranch := strings.TrimPrefix(branchLine, "branch refs/heads/")
					if currentBranch == branchName {
						return worktreePath, nil
					}
					break // Found branch info for this worktree, move to next
				}
			}
		}
	}

	return "", fmt.Errorf("no worktree found for branch '%s'", branchName)
}

// ResolveConflicts handles file conflicts based on the specified strategy.
func (f *FileManagerImpl) ResolveConflicts(conflicts []FileConflict, strategy ConflictStrategy) error {
	if len(conflicts) == 0 {
		return nil
	}

	switch strategy {
	case ConflictSkip:
		return nil

	case ConflictOverwrite:
		for _, conflict := range conflicts {
			if err := f.copyFile(conflict.SourcePath, conflict.TargetPath); err != nil {
				return fmt.Errorf("failed to overwrite %s: %w", conflict.Path, err)
			}
		}
		return nil

	case ConflictBackup:
		for _, conflict := range conflicts {
			timestamp := time.Now().Format("20060102_150405")
			backupPath := conflict.TargetPath + ".backup." + timestamp

			if err := f.copyFile(conflict.TargetPath, backupPath); err != nil {
				return fmt.Errorf("failed to create backup for %s: %w", conflict.Path, err)
			}

			if err := f.copyFile(conflict.SourcePath, conflict.TargetPath); err != nil {
				return fmt.Errorf("failed to copy %s after backup: %w", conflict.Path, err)
			}
		}
		return nil

	case ConflictPrompt:
		return f.promptForConflictResolution(conflicts)

	default:
		return fmt.Errorf("unknown conflict strategy: %s", strategy)
	}
}

// findMatchingFiles finds files in the source worktree matching the given pattern.
func (f *FileManagerImpl) findMatchingFiles(worktreePath, pattern string) ([]string, error) {
	var matches []string

	err := filepath.Walk(worktreePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if strings.Contains(path, ".git/") || strings.HasSuffix(path, ".git") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(worktreePath, path)
		if err != nil {
			return err
		}

		matched, err := filepath.Match(pattern, relPath)
		if err != nil {
			return fmt.Errorf("invalid pattern %s: %w", pattern, err)
		}

		if matched || strings.HasPrefix(relPath, strings.TrimSuffix(pattern, "*")) {
			if !info.IsDir() {
				matches = append(matches, relPath)
			}
		}

		return nil
	})

	return matches, err
}

// copyFile copies a file from source to destination, creating directories as needed.
func (f *FileManagerImpl) copyFile(srcPath, dstPath string) error {
	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(dstPath), err)
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer func() {
		if closeErr := src.Close(); closeErr != nil {
			// Log error but don't override main error.
			fmt.Printf("Warning: failed to close source file: %v\n", closeErr)
		}
	}()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer func() {
		if closeErr := dst.Close(); closeErr != nil {
			// Log error but don't override main error.
			fmt.Printf("Warning: failed to close destination file: %v\n", closeErr)
		}
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to copy file contents: %w", err)
	}

	srcInfo, err := src.Stat()
	if err != nil {
		return fmt.Errorf("failed to get source file info: %w", err)
	}

	if err := dst.Chmod(srcInfo.Mode()); err != nil {
		return fmt.Errorf("failed to set file permissions: %w", err)
	}

	return nil
}

// validateWorktreePath verifies that the path exists and appears to be a valid worktree.
func (f *FileManagerImpl) validateWorktreePath(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("path does not exist: %s", path)
	}

	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}

	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		return fmt.Errorf("path does not appear to be a Git worktree: %s", path)
	}

	return nil
}

// promptForConflictResolution prompts the user for each conflict resolution.
func (f *FileManagerImpl) promptForConflictResolution(conflicts []FileConflict) error {
	reader := bufio.NewReader(os.Stdin)

	for _, conflict := range conflicts {
		fmt.Printf("File conflict: %s\n", conflict.Path)
		fmt.Printf("  Source: %s\n", conflict.SourcePath)
		fmt.Printf("  Target: %s\n", conflict.TargetPath)
		fmt.Print("Choose action: [s]kip, [o]verwrite, [b]ackup and overwrite: ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("failed to read user input: %w", err)
		}

		response = strings.ToLower(strings.TrimSpace(response))
		switch response {
		case "s", "skip":
			continue // Skip this file.
		case "o", "overwrite":
			if err := f.copyFile(conflict.SourcePath, conflict.TargetPath); err != nil {
				return fmt.Errorf("failed to overwrite %s: %w", conflict.Path, err)
			}
		case "b", "backup":
			timestamp := time.Now().Format("20060102_150405")
			backupPath := conflict.TargetPath + ".backup." + timestamp

			if err := f.copyFile(conflict.TargetPath, backupPath); err != nil {
				return fmt.Errorf("failed to create backup for %s: %w", conflict.Path, err)
			}

			if err := f.copyFile(conflict.SourcePath, conflict.TargetPath); err != nil {
				return fmt.Errorf("failed to copy %s after backup: %w", conflict.Path, err)
			}
		default:
			fmt.Printf("Invalid choice '%s'. Skipping file.\n", response)
		}
	}

	return nil
}
