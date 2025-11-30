package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
)

const groveGitContent = "gitdir: .bare"

// ErrNotInWorkspace is returned when not inside a grove workspace
var ErrNotInWorkspace = errors.New("not in a grove workspace")

// SanitizeBranchName replaces filesystem-problematic characters with dash.
// Includes all characters that are unsafe on Windows filesystems.
func SanitizeBranchName(branch string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"\\", "-",
		"<", "-",
		">", "-",
		"|", "-",
		`"`, "-",
		"?", "-",
		"*", "-",
		":", "-",
	)
	return replacer.Replace(branch)
}

// FindBareDir finds the .bare directory for a grove workspace
// by walking up the directory tree from the given path
func FindBareDir(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	dir := absPath
	for i := 0; i < fs.MaxDirectoryIterations; i++ {
		bareDir := filepath.Join(dir, ".bare")
		if fs.DirectoryExists(bareDir) {
			return bareDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInWorkspace
		}
		dir = parent
	}
	return "", fmt.Errorf("exceeded maximum directory depth (%d): possible symlink loop", fs.MaxDirectoryIterations)
}

// IsInsideGroveWorkspace checks if the given path is inside an existing grove workspace
func IsInsideGroveWorkspace(path string) bool {
	_, err := FindBareDir(path)
	return err == nil
}

// ResolveConfigDir finds the appropriate directory for reading config.
// If inside a worktree, returns that worktree's root.
// If at workspace root, returns the default branch worktree or first available worktree.
func ResolveConfigDir(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	dir := absPath
	for i := 0; i < fs.MaxDirectoryIterations; i++ {
		bareDir := filepath.Join(dir, ".bare")
		if fs.DirectoryExists(bareDir) {
			return findCanonicalConfigDir(bareDir, dir)
		}

		gitPath := filepath.Join(dir, ".git")
		if info, err := os.Stat(gitPath); err == nil && !info.IsDir() {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInWorkspace
		}
		dir = parent
	}

	return "", fmt.Errorf("exceeded maximum directory depth (%d): possible symlink loop", fs.MaxDirectoryIterations)
}

// findCanonicalConfigDir finds the config directory when at workspace root.
// Priority: default branch worktree > first worktree in list.
func findCanonicalConfigDir(bareDir, workspaceRoot string) (string, error) {
	defaultBranch, err := git.GetDefaultBranch(bareDir)
	if err == nil && defaultBranch != "" {
		defaultWorktree := filepath.Join(workspaceRoot, SanitizeBranchName(defaultBranch))
		if fs.DirectoryExists(defaultWorktree) {
			return defaultWorktree, nil
		}
	}

	worktrees, err := git.ListWorktrees(bareDir)
	if err != nil {
		return "", fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		return "", fmt.Errorf("no worktrees found in workspace")
	}

	return worktrees[0], nil
}

// ValidateAndPrepareDirectory validates and prepares a directory for grove workspace.
// It rejects directories inside existing git repos or grove workspaces,
// rejects non-empty directories, and creates the directory if it doesn't exist.
func ValidateAndPrepareDirectory(path string) error {
	if git.IsInsideGitRepo(path) {
		return fmt.Errorf("cannot initialize grove inside existing git repository")
	}

	if IsInsideGroveWorkspace(path) {
		return fmt.Errorf("cannot initialize grove inside existing grove workspace")
	}

	if fs.DirectoryExists(path) {

		isEmpty, err := fs.IsEmptyDir(path)
		if err != nil {
			return fmt.Errorf("failed to check if directory is empty: %w", err)
		}
		if !isEmpty {
			return fmt.Errorf("directory %s is not empty", path)
		}
	} else {
		if err := fs.CreateDirectory(path, fs.DirGit); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	return nil
}

// cloneWithProgress clones a repository with progress indication
func cloneWithProgress(url, bareDir string, verbose, shallow bool) error {
	stop := logger.StartSpinner("Cloning repository...")
	defer stop()

	if err := git.Clone(url, bareDir, !verbose, shallow); err != nil {
		return err
	}

	stop()
	logger.Success("Repository cloned")
	return nil
}

// createWorktreesFromBranches creates worktrees for the specified branches, skipping skipBranch if set.
// Returns absolute paths of created worktrees so callers can clean them up on failure.
func createWorktreesFromBranches(bareDir, branches string, verbose bool, skipBranch string) ([]string, error) {
	filteredBranches := parseBranches(branches, skipBranch)

	if len(filteredBranches) == 0 {
		logger.Debug("No branches to create (all branches filtered out)")
		return nil, nil
	}

	logger.Info("Creating worktrees:")
	var createdPaths []string

	for _, branch := range filteredBranches {
		sanitizedName := SanitizeBranchName(branch)
		worktreePath := filepath.Join("..", sanitizedName)
		absWorktreePath := filepath.Join(filepath.Dir(bareDir), sanitizedName)

		if err := git.CreateWorktree(bareDir, worktreePath, branch, !verbose); err != nil {
			for i := len(createdPaths) - 1; i >= 0; i-- {
				if removeErr := fs.RemoveAll(createdPaths[i]); removeErr != nil {
					logger.Warning("Failed to cleanup worktree %s: %v", createdPaths[i], removeErr)
				}
			}
			return createdPaths, fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
		}
		createdPaths = append(createdPaths, absWorktreePath)

		if config.ShouldAutoLock(branch) {
			if err := git.LockWorktree(bareDir, absWorktreePath, "Auto-locked (grove.autoLock)"); err != nil {
				logger.Debug("Failed to auto-lock worktree: %v", err)
			} else {
				logger.Debug("Auto-locked worktree for branch %s", branch)
			}
		}
	}

	for _, branch := range filteredBranches {
		sanitizedName := SanitizeBranchName(branch)
		logger.ListItemWithNote(sanitizedName, "")
	}

	return createdPaths, nil
}

// Initialize creates a new grove workspace in the specified directory
func Initialize(path string) error {
	if err := ValidateAndPrepareDirectory(path); err != nil {
		return err
	}

	bareDir := filepath.Join(path, ".bare")
	if err := fs.CreateDirectory(bareDir, fs.DirGit); err != nil {
		return fmt.Errorf("failed to create .bare directory: %w", err)
	}

	if err := git.InitBare(bareDir); err != nil {
		if cleanupErr := fs.RemoveAll(bareDir); cleanupErr != nil {
			logger.Warning("Failed to cleanup .bare directory after error: %v", cleanupErr)
		}
		return fmt.Errorf("failed to initialize bare git repository: %w", err)
	}

	gitFile := filepath.Join(path, ".git")
	if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
		if cleanupErr := fs.RemoveAll(bareDir); cleanupErr != nil {
			logger.Warning("Failed to cleanup .bare directory after error: %v", cleanupErr)
		}
		return fmt.Errorf("failed to create .git file: %w", err)
	}
	return nil
}

// CloneAndInitialize clones a repository and creates a grove workspace in the specified directory
func CloneAndInitialize(url, path, branches string, verbose, shallow bool) error {
	if err := ValidateAndPrepareDirectory(path); err != nil {
		return err
	}

	bareDir := filepath.Join(path, ".bare")
	gitFile := filepath.Join(path, ".git")

	cleanup := func(worktrees []string) {
		if err := os.Remove(gitFile); err != nil && !errors.Is(err, os.ErrNotExist) {
			logger.Warning("Failed to remove .git file during cleanup: %v", err)
		}
		if err := fs.RemoveAll(bareDir); err != nil {
			logger.Warning("Failed to remove .bare during cleanup: %v", err)
		}
		for i := len(worktrees) - 1; i >= 0; i-- {
			if err := fs.RemoveAll(worktrees[i]); err != nil {
				logger.Warning("Failed to remove worktree %s during cleanup: %v", worktrees[i], err)
			}
		}
	}

	if err := cloneWithProgress(url, bareDir, verbose, shallow); err != nil {
		cleanup(nil)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
		cleanup(nil)
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	branchesToCreate := branches
	if branchesToCreate == "" {
		defaultBranch, err := git.GetDefaultBranch(bareDir)
		if err != nil {
			cleanup(nil)
			return fmt.Errorf("failed to determine default branch: %w", err)
		}
		branchesToCreate = defaultBranch
	}

	createdWorktrees, err := createWorktreesFromBranches(bareDir, branchesToCreate, verbose, "")
	if err != nil {
		cleanup(createdWorktrees)
		return err
	}

	return nil
}

// validateRepoForConversion performs all pre-conversion validation checks
func validateRepoForConversion(targetDir string) error {
	if IsInsideGroveWorkspace(targetDir) {
		return fmt.Errorf("already a grove workspace")
	}

	if !git.IsInsideGitRepo(targetDir) {
		return fmt.Errorf("not a git repository")
	}

	if git.IsWorktree(targetDir) {
		return fmt.Errorf("cannot convert: repository is already a worktree")
	}

	hasLocks, err := git.HasLockFiles(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check for lock files: %w", err)
	}
	if hasLocks {
		return fmt.Errorf("cannot convert: repository has active lock files")
	}

	hasSubmodules, err := git.HasSubmodules(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check for submodules: %w", err)
	}
	if hasSubmodules {
		return fmt.Errorf("cannot convert: repository has submodules")
	}

	hasConflicts, err := git.HasUnresolvedConflicts(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check for unresolved conflicts: %w", err)
	}
	if hasConflicts {
		return fmt.Errorf("cannot convert: repository has unresolved conflicts")
	}

	hasChanges, _, err := git.CheckGitChanges(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check for uncommitted changes: %w", err)
	}
	if hasChanges {
		return fmt.Errorf("cannot convert: repository has uncommitted changes")
	}

	hasUnpushed, err := git.HasUnpushedCommits(targetDir)
	if err != nil && !errors.Is(err, git.ErrNoUpstreamConfigured) {
		return fmt.Errorf("failed to check for unpushed commits: %w", err)
	}
	if hasUnpushed {
		return fmt.Errorf("cannot convert: repository has unpushed commits")
	}

	detached, err := git.IsDetachedHead(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check HEAD state: %w", err)
	}
	if detached {
		return fmt.Errorf("cannot convert: repository is in detached HEAD state")
	}

	unborn, err := git.IsUnbornHead(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check for unborn HEAD: %w", err)
	}
	if unborn {
		return fmt.Errorf("cannot convert: repository has no commits (unborn HEAD)")
	}

	hasOngoing, err := git.HasOngoingOperation(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check for ongoing operations: %w", err)
	}
	if hasOngoing {
		return fmt.Errorf("cannot convert: repository has ongoing merge/rebase/cherry-pick")
	}

	worktrees, err := git.ListWorktrees(targetDir)
	if err != nil {
		return fmt.Errorf("failed to check for existing worktrees: %w", err)
	}
	if len(worktrees) > 0 {
		return fmt.Errorf("cannot convert: repository has existing worktrees")
	}

	return nil
}

// setupBareRepo moves .git to .bare and configures it as bare repository
func setupBareRepo(targetDir string) (string, error) {
	currentBranch, err := git.GetCurrentBranch(targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	gitDir := filepath.Join(targetDir, ".git")
	bareDir := filepath.Join(targetDir, ".bare")
	logger.Debug("Preparing to convert repository to bare: %s -> %s", gitDir, bareDir)

	logger.Info("Moving %s directory to %s...", styles.Render(&styles.Path, ".git"), styles.Render(&styles.Path, ".bare"))
	if err := fs.RenameWithFallback(gitDir, bareDir); err != nil {
		return "", fmt.Errorf("failed to move .git to .bare: %w (recovery: ensure %s exists)", err, gitDir)
	}

	logger.Info("Configuring repository as bare...")
	if err := git.ConfigureBare(bareDir); err != nil {
		return "", fmt.Errorf("failed to configure as bare repository: %w", err)
	}

	logger.Debug("Repository converted to bare successfully, current branch: %s", currentBranch)
	return currentBranch, nil
}

// createMainWorktree creates worktree for current branch and moves files into it
func createMainWorktree(targetDir, currentBranch string, verbose bool, movedFiles *[]string) ([]string, error) {
	bareDir := filepath.Join(targetDir, ".bare")
	branches := []string{currentBranch}

	createdWorktrees, err := createWorktreesOnly(bareDir, branches, verbose)
	if err != nil {
		return nil, err
	}

	if err := moveFilesToFirstWorktree(targetDir, branches, movedFiles); err != nil {
		return createdWorktrees, err
	}

	if err := checkoutFirstWorktree(targetDir, currentBranch, verbose); err != nil {
		return createdWorktrees, err
	}

	return createdWorktrees, nil
}

// createWorktreesOnly creates worktrees for all specified branches
func createWorktreesOnly(bareDir string, branches []string, verbose bool) ([]string, error) {
	logger.Info("Creating worktrees...")
	var createdPaths []string

	for i, branch := range branches {
		sanitizedName := SanitizeBranchName(branch)
		worktreePath := filepath.Join("..", sanitizedName)
		absWorktreePath := filepath.Join(filepath.Dir(bareDir), sanitizedName)

		if i == 0 {
			logger.Debug("Executing: git worktree add --no-checkout %s %s in %s", worktreePath, branch, bareDir)
			cmd, cancel := git.GitCommand("git", "worktree", "add", "--no-checkout", worktreePath, branch) // nolint:gosec
			cmd.Dir = bareDir

			var stderr bytes.Buffer
			if verbose {
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
			} else {
				cmd.Stderr = &stderr
			}

			err := cmd.Run()
			cancel()
			if err != nil {
				if !verbose && stderr.Len() > 0 {
					return createdPaths, fmt.Errorf("failed to create worktree for branch '%s': %w: %s", branch, err, strings.TrimSpace(stderr.String()))
				}
				return createdPaths, fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
			}
		} else {
			if err := git.CreateWorktree(bareDir, worktreePath, branch, !verbose); err != nil {
				return createdPaths, fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
			}
		}
		createdPaths = append(createdPaths, absWorktreePath)

		if config.ShouldAutoLock(branch) {
			if err := git.LockWorktree(bareDir, absWorktreePath, "Auto-locked (grove.autoLock)"); err != nil {
				logger.Debug("Failed to auto-lock worktree: %v", err)
			} else {
				logger.Debug("Auto-locked worktree for branch %s", branch)
			}
		}
	}
	return createdPaths, nil
}

// moveFilesToFirstWorktree moves all files from targetDir to the first worktree
func moveFilesToFirstWorktree(targetDir string, branches []string, movedFiles *[]string) error {
	if len(branches) == 0 {
		return errors.New("no branches provided")
	}
	firstSanitizedName := SanitizeBranchName(branches[0])
	firstWorktreeAbsPath := filepath.Join(targetDir, firstSanitizedName)

	if !fs.PathExists(firstWorktreeAbsPath) {
		return fmt.Errorf("worktree directory %s was not created as expected", firstWorktreeAbsPath)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return fmt.Errorf("failed to read directory entries: %w", err)
	}

	worktreeDirs := make(map[string]bool)
	for _, branch := range branches {
		worktreeDirs[SanitizeBranchName(branch)] = true
	}

	var filesToMoveCount int
	for _, entry := range entries {
		if entry.Name() != ".bare" && !worktreeDirs[entry.Name()] {
			filesToMoveCount++
		}
	}

	logger.Debug("Preparing to move files to first worktree: %s -> %s, count: %d", targetDir, firstWorktreeAbsPath, filesToMoveCount)
	logger.Info("Reorganizing repository files...")

	if movedFiles != nil && filesToMoveCount > 0 {
		*movedFiles = make([]string, 0, filesToMoveCount)
	}

	movedCount := 0
	for _, entry := range entries {
		if entry.Name() == ".bare" || worktreeDirs[entry.Name()] || entry.Name() == ".grove-convert.lock" {
			continue
		}

		oldPath := filepath.Join(targetDir, entry.Name())
		newPath := filepath.Join(firstWorktreeAbsPath, entry.Name())
		if err := fs.RenameWithFallback(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move %s to worktree: %w (moved %d/%d files)", entry.Name(), err, movedCount, filesToMoveCount)
		}
		if movedFiles != nil {
			*movedFiles = append(*movedFiles, entry.Name())
		}
		movedCount++
	}

	logger.Debug("Successfully moved %d files to worktree", movedCount)
	return nil
}

// checkoutFirstWorktree checks out the first worktree
func checkoutFirstWorktree(targetDir, firstBranch string, verbose bool) error {
	firstSanitizedName := SanitizeBranchName(firstBranch)
	firstWorktreeAbsPath := filepath.Join(targetDir, firstSanitizedName)

	logger.Debug("Executing: git checkout -f %s in %s", firstBranch, firstWorktreeAbsPath)
	checkoutCmd, cancel := git.GitCommand("git", "checkout", "-f", firstBranch) // nolint:gosec
	defer cancel()
	checkoutCmd.Dir = firstWorktreeAbsPath

	var checkoutStderr bytes.Buffer
	if verbose {
		checkoutCmd.Stdout = os.Stdout
		checkoutCmd.Stderr = os.Stderr
	} else {
		checkoutCmd.Stderr = &checkoutStderr
	}

	if err := checkoutCmd.Run(); err != nil {
		if !verbose && checkoutStderr.Len() > 0 {
			return fmt.Errorf("failed to restore files in worktree: %w: %s", err, strings.TrimSpace(checkoutStderr.String()))
		}
		return fmt.Errorf("failed to restore files in worktree: %w", err)
	}
	return nil
}

// conversionOpts holds options for worktree creation during conversion.
type conversionOpts struct {
	TargetDir        string
	CurrentBranch    string
	Branches         string
	Verbose          bool
	IgnoredFiles     []string
	PreservePatterns []string
}

// conversionResult holds the results of worktree creation during conversion.
type conversionResult struct {
	Worktrees  []string
	MovedFiles []string
}

// createWorktreesForConversion creates worktrees for specified branches and moves files to current branch
func createWorktreesForConversion(opts *conversionOpts) (*conversionResult, error) {
	targetDir := opts.TargetDir
	currentBranch := opts.CurrentBranch
	branches := opts.Branches
	verbose := opts.Verbose
	ignoredFiles := opts.IgnoredFiles
	preservePatterns := opts.PreservePatterns
	result := &conversionResult{}
	bareDir := filepath.Join(targetDir, ".bare")
	cleanedBranches := parseBranches(branches, "")

	if len(cleanedBranches) == 0 {
		return nil, fmt.Errorf("no valid branches specified")
	}

	found := false
	for i, branch := range cleanedBranches {
		if branch == currentBranch {
			found = true
			if i != 0 {
				cleanedBranches = append([]string{currentBranch}, append(cleanedBranches[:i], cleanedBranches[i+1:]...)...)
			}
			break
		}
	}
	if !found {
		cleanedBranches = append([]string{currentBranch}, cleanedBranches...)
	}

	worktrees, err := createWorktreesOnly(bareDir, cleanedBranches, verbose)
	if err != nil {
		return nil, err
	}
	result.Worktrees = worktrees

	preservedCount, matchedPatterns, err := preserveIgnoredFilesFromList(targetDir, cleanedBranches, ignoredFiles, preservePatterns)
	if err != nil {
		return result, err
	}

	for i, branch := range cleanedBranches {
		sanitizedName := SanitizeBranchName(branch)
		if branch == currentBranch {
			logger.ListItemWithNote(sanitizedName, "current")
			if i == 0 && preservedCount > 0 {
				itemText := "file/directory"
				if preservedCount > 1 {
					itemText = "files/directories"
				}

				patternText := "pattern"
				if len(matchedPatterns) > 1 {
					patternText = "patterns"
				}

				logger.ListSubItem("Found %d ignored %s matching %s: %s",
					preservedCount,
					itemText,
					patternText,
					strings.Join(matchedPatterns, ", "))
			}
		} else {
			logger.ListItemWithNote(sanitizedName, "")
		}
	}

	if err := moveFilesToFirstWorktree(targetDir, cleanedBranches, &result.MovedFiles); err != nil {
		return result, err
	}

	if err := checkoutFirstWorktree(targetDir, cleanedBranches[0], verbose); err != nil {
		return result, err
	}

	return result, nil
}

// parseBranches splits and cleans a comma-separated list of branches
func parseBranches(branches, skipBranch string) []string {
	branchList := strings.Split(branches, ",")
	var cleanedBranches []string
	for _, branch := range branchList {
		branch = strings.TrimSpace(branch)
		if branch != "" && branch != skipBranch {
			cleanedBranches = append(cleanedBranches, branch)
		}
	}
	logger.Debug("Parsed branches for worktree creation: %v", cleanedBranches)
	return cleanedBranches
}

// findIgnoredFiles returns a list of git-ignored files in the given directory
func findIgnoredFiles(dir string) ([]string, error) {
	return git.ListIgnoredFiles(dir)
}

// matchesPattern checks if a file path matches a single pattern
func matchesPattern(filePath, pattern string) bool {
	fileName := filepath.Base(filePath)
	if matched, _ := filepath.Match(pattern, fileName); matched {
		return true
	}
	if matched, _ := filepath.Match(pattern, filePath); matched {
		return true
	}
	return false
}

// preserveIgnoredFilesFromList copies matching ignored files to all worktrees
// If patterns is nil, it will be loaded from config (only works if .git exists in sourceDir)
func preserveIgnoredFilesFromList(sourceDir string, branches, ignoredFiles, patterns []string) (count int, matchedPatterns []string, err error) {
	if len(ignoredFiles) == 0 {
		return 0, nil, nil
	}

	if patterns == nil {
		patterns = config.GetMergedPreservePatterns(sourceDir)
	}

	var filesToCopy []string
	matchedPatternsMap := make(map[string]bool)
	for _, file := range ignoredFiles {
		for _, pattern := range patterns {
			if matchesPattern(file, pattern) {
				filesToCopy = append(filesToCopy, file)
				matchedPatternsMap[pattern] = true
				break
			}
		}
	}

	if len(filesToCopy) == 0 {
		return 0, nil, nil
	}

	logger.Debug("Preserving ignored files: %v", filesToCopy)

	for _, branch := range branches {
		sanitizedName := SanitizeBranchName(branch)
		worktreeDir := filepath.Join(sourceDir, sanitizedName)

		for _, file := range filesToCopy {
			sourcePath := filepath.Join(sourceDir, file)
			destPath := filepath.Join(worktreeDir, file)

			destDir := filepath.Dir(destPath)
			if err := os.MkdirAll(destDir, fs.DirGit); err != nil {
				return 0, nil, fmt.Errorf("failed to create directory for preserved file %s: %w", file, err)
			}

			if err := fs.CopyFile(sourcePath, destPath, fs.FileGit); err != nil {
				return 0, nil, fmt.Errorf("failed to preserve file %s in worktree %s: %w", file, sanitizedName, err)
			}
		}
	}

	for pattern := range matchedPatternsMap {
		matchedPatterns = append(matchedPatterns, pattern)
	}

	return len(filesToCopy), matchedPatterns, nil
}

// validateBranchesForConvert validates that all specified branches exist before conversion
func validateBranchesForConvert(targetDir, branches string) error {
	cleanedBranches := parseBranches(branches, "")
	logger.Debug("Validating branches exist: %v", cleanedBranches)

	for _, branch := range cleanedBranches {
		exists, err := git.BranchExists(targetDir, branch)
		if err != nil {
			logger.Debug("Branch validation failed for '%s': %v", branch, err)
			return fmt.Errorf("failed to check branch '%s': %w", branch, err)
		}
		if !exists {
			logger.Debug("Branch validation failed: branch '%s' does not exist", branch)
			return fmt.Errorf("branch '%s' does not exist", branch)
		}
	}

	return nil
}

func Convert(targetDir, branches string, verbose bool) error {
	if err := validateRepoForConversion(targetDir); err != nil {
		return err
	}

	if branches != "" {
		if err := validateBranchesForConvert(targetDir, branches); err != nil {
			return err
		}
	}

	lockFile := filepath.Join(targetDir, ".grove-convert.lock")
	lockHandle, err := os.OpenFile(lockFile, os.O_CREATE|os.O_EXCL|os.O_WRONLY, fs.FileGit) // nolint:gosec // Controlled path from validated directory
	if err != nil {
		if errors.Is(err, os.ErrExist) {
			return fmt.Errorf("conversion already in progress or failed previously; remove %s to proceed", lockFile)
		}
		return fmt.Errorf("failed to acquire conversion lock: %w", err)
	}
	_ = lockHandle.Close()
	defer func() { _ = os.Remove(lockFile) }()

	var ignoredFiles []string
	var preservePatterns []string
	if branches != "" {
		files, err := findIgnoredFiles(targetDir)
		if err != nil {
			logger.Debug("Failed to find ignored files (continuing anyway): %v", err)
		} else {
			ignoredFiles = files
		}
		// Get preserve patterns BEFORE moving .git to .bare (git config needs .git)
		preservePatterns = config.GetMergedPreservePatterns(targetDir)
	}

	currentBranch, err := setupBareRepo(targetDir)
	if err != nil {
		return err
	}

	// From this point on, we have destructive changes that need rollback on failure
	var movedFiles []string
	var createdWorktrees []string
	conversionSucceeded := false

	defer func() {
		if conversionSucceeded {
			return
		}

		logger.Error("Conversion failed, attempting to restore repository...")
		var restoreErrors []string

		// Step 1: Remove created worktrees (best effort)
		for _, worktreePath := range createdWorktrees {
			logger.Debug("Removing worktree: %s", worktreePath)
			if err := fs.RemoveAll(worktreePath); err != nil {
				logger.Warning("Failed to remove worktree %s: %v", worktreePath, err)
			}
		}

		// Step 2: Move files back from first worktree (best effort, track failures)
		if len(movedFiles) > 0 && len(createdWorktrees) > 0 {
			firstWorktree := createdWorktrees[0]
			for i := len(movedFiles) - 1; i >= 0; i-- {
				fileName := movedFiles[i]
				src := filepath.Join(firstWorktree, fileName)
				dst := filepath.Join(targetDir, fileName)
				logger.Debug("Moving file back: %s -> %s", src, dst)
				if err := fs.RenameWithFallback(src, dst); err != nil {
					logger.Error("Failed to move file back %s: %v", fileName, err)
					restoreErrors = append(restoreErrors, fileName)
				}
			}
		}

		// Step 3: Always attempt to restore .git directory
		gitDir := filepath.Join(targetDir, ".git")
		bareDir := filepath.Join(targetDir, ".bare")
		logger.Info("Restoring .git directory...")

		_ = fs.RemoveAll(gitDir) // Remove any partial .git file

		if err := fs.RenameWithFallback(bareDir, gitDir); err != nil {
			logger.Error("Failed to restore .git directory: %v", err)
			restoreErrors = append(restoreErrors, ".git")
		}

		// Step 4: Always attempt to restore git config
		if err := git.RestoreNormalConfig(targetDir); err != nil {
			logger.Error("Failed to restore git config: %v", err)
			restoreErrors = append(restoreErrors, "git config")
		}

		// Report final status
		if len(restoreErrors) > 0 {
			logger.Error("CRITICAL: Restoration incomplete. Failed to restore: %v", restoreErrors)
			logger.Error("Your repository may be in an inconsistent state.")

			// Write recovery instructions to a file so they're not lost in terminal scrollback
			recoveryFile := filepath.Join(targetDir, ".grove-recovery.txt")
			var recoveryInstructions strings.Builder
			recoveryInstructions.WriteString("Grove conversion failed. Manual recovery steps:\n\n")
			recoveryInstructions.WriteString(fmt.Sprintf("Failed to restore: %v\n\n", restoreErrors))
			if len(createdWorktrees) > 0 {
				recoveryInstructions.WriteString(fmt.Sprintf("1. Check for files in: %s\n", createdWorktrees[0]))
			}
			recoveryInstructions.WriteString("2. Ensure .git directory exists and is not bare\n")
			recoveryInstructions.WriteString("3. Run: git config --bool core.bare false\n")
			recoveryInstructions.WriteString("\nDelete this file after recovery is complete.\n")

			if err := os.WriteFile(recoveryFile, []byte(recoveryInstructions.String()), fs.FileStrict); err != nil {
				logger.Error("Failed to write recovery file: %v", err)
				// Try fallback location
				fallbackFile := filepath.Join(os.TempDir(), fmt.Sprintf("grove-recovery-%d.txt", os.Getpid()))
				if fallbackErr := os.WriteFile(fallbackFile, []byte(recoveryInstructions.String()), fs.FileStrict); fallbackErr == nil {
					logger.Error("Recovery instructions saved to: %s", fallbackFile)
				}
			} else {
				logger.Error("Recovery instructions saved to: %s", recoveryFile)
			}

			logger.Error("To recover manually:")
			if len(createdWorktrees) > 0 {
				logger.Error("1. Check for files in: %s", createdWorktrees[0])
			}
			logger.Error("2. Ensure .git directory exists and is not bare")
			logger.Error("3. Run: git config --bool core.bare false")
		} else {
			logger.Success("Repository restored to original state")
		}
	}()

	if branches != "" {
		result, err := createWorktreesForConversion(&conversionOpts{
			TargetDir:        targetDir,
			CurrentBranch:    currentBranch,
			Branches:         branches,
			Verbose:          verbose,
			IgnoredFiles:     ignoredFiles,
			PreservePatterns: preservePatterns,
		})
		if err != nil {
			if result != nil {
				createdWorktrees = result.Worktrees
				movedFiles = result.MovedFiles
			}
			return err
		}
		createdWorktrees = result.Worktrees
		movedFiles = result.MovedFiles
	} else {
		var err error
		createdWorktrees, err = createMainWorktree(targetDir, currentBranch, verbose, &movedFiles)
		if err != nil {
			return err
		}
	}

	gitFile := filepath.Join(targetDir, ".git")
	if err := fs.WriteFileAtomic(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	conversionSucceeded = true
	return nil
}
