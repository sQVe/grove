package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
)

const groveGitContent = "gitdir: .bare"

// sanitizeBranchName replaces filesystem-problematic characters with dash
func sanitizeBranchName(branch string) string {
	replacer := strings.NewReplacer(
		"/", "-",
		"<", "-",
		">", "-",
		"|", "-",
		`"`, "-",
	)
	return replacer.Replace(branch)
}

// IsInsideGroveWorkspace checks if the given path is inside an existing grove workspace
func IsInsideGroveWorkspace(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}

	dir := absPath
	for {
		bareDir := filepath.Join(dir, ".bare")
		if fs.DirectoryExists(bareDir) {
			return true
		}

		gitFile := filepath.Join(dir, ".git")
		if content, err := os.ReadFile(gitFile); err == nil { // nolint:gosec // Controlled path for workspace validation
			if strings.TrimSpace(string(content)) == groveGitContent {
				return true
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return false
}

// validateAndPrepareDirectory validates and prepares a directory for grove workspace
func validateAndPrepareDirectory(path string) error {
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
func cloneWithProgress(url, bareDir string, verbose bool) error {
	stop := logger.StartSpinner("Cloning repository...")
	defer stop()

	if err := git.Clone(url, bareDir, !verbose); err != nil {
		return err
	}

	stop()
	logger.Success("Repository cloned")
	return nil
}

// createWorktreesFromBranches creates worktrees for the specified branches, optionally skipping one
func createWorktreesFromBranches(bareDir, branches string, verbose bool, skipBranch string) error {
	filteredBranches := parseBranches(branches, skipBranch)

	if len(filteredBranches) == 0 {
		logger.Debug("No branches to create (all branches filtered out)")
		return nil
	}

	logger.Info("Creating worktrees:")

	for _, branch := range filteredBranches {
		sanitizedName := sanitizeBranchName(branch)
		worktreePath := filepath.Join("..", sanitizedName)

		if err := git.CreateWorktree(bareDir, worktreePath, branch, !verbose); err != nil {
			return fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
		}
	}

	for _, branch := range filteredBranches {
		sanitizedName := sanitizeBranchName(branch)
		fmt.Printf("  %s %s\n", styles.Render(&styles.Success, "✓"), styles.Render(&styles.Worktree, sanitizedName))
	}

	return nil
}

// Initialize creates a new grove workspace in the specified directory
func Initialize(path string) error {
	if err := validateAndPrepareDirectory(path); err != nil {
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

	// Create .git file pointing to .bare directory
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
func CloneAndInitialize(url, path, branches string, verbose bool) error {
	if err := validateAndPrepareDirectory(path); err != nil {
		return err
	}

	bareDir := filepath.Join(path, ".bare")

	if err := cloneWithProgress(url, bareDir, verbose); err != nil {
		if cleanupErr := fs.RemoveAll(bareDir); cleanupErr != nil {
			logger.Warning("Failed to cleanup .bare directory after error: %v", cleanupErr)
		}
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Create .git file pointing to .bare directory
	gitFile := filepath.Join(path, ".git")
	if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	if branches != "" {
		return createWorktreesFromBranches(bareDir, branches, verbose, "")
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
		sanitizedName := sanitizeBranchName(branch)
		worktreePath := filepath.Join("..", sanitizedName)
		absWorktreePath := filepath.Join(filepath.Dir(bareDir), sanitizedName)

		if i == 0 {
			logger.Debug("Executing: git worktree add --no-checkout %s %s in %s", worktreePath, branch, bareDir)
			cmd := exec.Command("git", "worktree", "add", "--no-checkout", worktreePath, branch) // nolint:gosec
			cmd.Dir = bareDir

			var stderr bytes.Buffer
			if verbose {
				cmd.Stdout = os.Stdout
				cmd.Stderr = os.Stderr
			} else {
				cmd.Stderr = &stderr
			}

			if err := cmd.Run(); err != nil {
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
	}
	return createdPaths, nil
}

// moveFilesToFirstWorktree moves all files from targetDir to the first worktree
func moveFilesToFirstWorktree(targetDir string, branches []string, movedFiles *[]string) error {
	firstSanitizedName := sanitizeBranchName(branches[0])
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
		worktreeDirs[sanitizeBranchName(branch)] = true
	}

	// Count files to move for progress reporting
	var filesToMoveCount int
	for _, entry := range entries {
		if entry.Name() != ".bare" && !worktreeDirs[entry.Name()] {
			filesToMoveCount++
		}
	}

	logger.Debug("Preparing to move files to first worktree: %s -> %s, count: %d", targetDir, firstWorktreeAbsPath, filesToMoveCount)
	logger.Info("Reorganizing repository files...")

	// Pre-allocate movedFiles slice if tracking is enabled
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
	firstSanitizedName := sanitizeBranchName(firstBranch)
	firstWorktreeAbsPath := filepath.Join(targetDir, firstSanitizedName)

	logger.Debug("Executing: git checkout -f %s in %s", firstBranch, firstWorktreeAbsPath)
	checkoutCmd := exec.Command("git", "checkout", "-f", firstBranch) // nolint:gosec
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

// createWorktreesForConversion creates worktrees for specified branches and moves files to current branch
func createWorktreesForConversion(targetDir, currentBranch, branches string, verbose bool, ignoredFiles []string, movedFiles *[]string) ([]string, error) {
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

	createdWorktrees, err := createWorktreesOnly(bareDir, cleanedBranches, verbose)
	if err != nil {
		return nil, err
	}

	preservedCount, matchedPatterns, err := preserveIgnoredFilesFromList(targetDir, cleanedBranches, ignoredFiles)
	if err != nil {
		return createdWorktrees, err
	}

	for i, branch := range cleanedBranches {
		sanitizedName := sanitizeBranchName(branch)
		if branch == currentBranch {
			fmt.Printf("  %s %s %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, sanitizedName),
				styles.Render(&styles.Dimmed, "(current)"))
			if i == 0 && preservedCount > 0 {
				var styledPatterns []string
				for _, pattern := range matchedPatterns {
					styledPatterns = append(styledPatterns, styles.Render(&styles.Path, pattern))
				}

				itemText := "file/directory"
				if preservedCount > 1 {
					itemText = "files/directories"
				}

				patternText := "pattern"
				if len(matchedPatterns) > 1 {
					patternText = "patterns"
				}

				fmt.Printf("    %s\n",
					styles.Render(&styles.Dimmed, fmt.Sprintf("↳ Found %d ignored %s matching %s: %s",
						preservedCount,
						itemText,
						patternText,
						strings.Join(styledPatterns, ", "))))
			}
		} else {
			fmt.Printf("  %s %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, sanitizedName))
		}
	}

	if err := moveFilesToFirstWorktree(targetDir, cleanedBranches, movedFiles); err != nil {
		return createdWorktrees, err
	}

	if err := checkoutFirstWorktree(targetDir, cleanedBranches[0], verbose); err != nil {
		return createdWorktrees, err
	}

	return createdWorktrees, nil
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
	logger.Debug("Executing: git ls-files --others --ignored --exclude-standard in %s", dir)
	cmd := exec.Command("git", "ls-files", "--others", "--ignored", "--exclude-standard")
	cmd.Dir = dir

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list ignored files: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	var files []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			files = append(files, trimmed)
		}
	}
	return files, nil
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
func preserveIgnoredFilesFromList(sourceDir string, branches, ignoredFiles []string) (count int, matchedPatterns []string, err error) {
	if len(ignoredFiles) == 0 {
		return 0, nil, nil
	}

	patterns := config.GetPreservePatterns()

	var filesToCopy []string
	matchedPatternsMap := make(map[string]bool)
	for _, file := range ignoredFiles {
		for _, pattern := range patterns {
			if matchesPattern(file, pattern) {
				filesToCopy = append(filesToCopy, file)
				matchedPatternsMap[pattern] = true
				break // File matches this pattern, no need to check others
			}
		}
	}

	if len(filesToCopy) == 0 {
		return 0, nil, nil
	}

	logger.Debug("Preserving ignored files: %v", filesToCopy)

	for _, branch := range branches {
		sanitizedName := sanitizeBranchName(branch)
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

	var ignoredFiles []string
	if branches != "" {
		files, err := findIgnoredFiles(targetDir)
		if err != nil {
			logger.Debug("Failed to find ignored files (continuing anyway): %v", err)
		} else {
			ignoredFiles = files
		}
	}

	currentBranch, err := setupBareRepo(targetDir)
	if err != nil {
		return err
	}

	// From this point on, we have destructive changes that need rollback on failure
	var movedFiles []string
	var createdWorktrees []string
	conversionSucceeded := false

	defer func() { _ = os.Remove(lockFile) }()

	defer func() {
		if conversionSucceeded {
			return
		}

		logger.Error("Conversion failed, attempting to restore repository...")

		for _, worktreePath := range createdWorktrees {
			logger.Debug("Removing worktree: %s", worktreePath)
			if err := fs.RemoveAll(worktreePath); err != nil {
				logger.Warning("Failed to remove worktree %s: %v", worktreePath, err)
			}
		}

		fileRestoreFailed := false
		if len(movedFiles) > 0 && len(createdWorktrees) > 0 {
			firstWorktree := createdWorktrees[0]
			for i := len(movedFiles) - 1; i >= 0; i-- {
				fileName := movedFiles[i]
				src := filepath.Join(firstWorktree, fileName)
				dst := filepath.Join(targetDir, fileName)
				logger.Debug("Moving file back: %s -> %s", src, dst)
				if err := fs.RenameWithFallback(src, dst); err != nil {
					logger.Error("Failed to move file back %s: %v", fileName, err)
					fileRestoreFailed = true
				}
			}
		}

		if fileRestoreFailed {
			logger.Error("CRITICAL: Files are trapped in worktree directories")
			logger.Error("NOT restoring .git to prevent confusion about missing files")
			logger.Error("Your files are safe but located in: %v", createdWorktrees)
			logger.Error("To recover manually:")
			logger.Error("1. Move all files from %s back to repository root", createdWorktrees[0])
			logger.Error("2. Remove worktree directories: %v", createdWorktrees)
			logger.Error("3. Run: mv %s %s", filepath.Join(targetDir, ".bare"), filepath.Join(targetDir, ".git"))
			return
		}

		gitDir := filepath.Join(targetDir, ".git")
		bareDir := filepath.Join(targetDir, ".bare")
		logger.Info("Restoring .git directory...")

		_ = fs.RemoveAll(gitDir)

		if err := fs.RenameWithFallback(bareDir, gitDir); err != nil {
			logger.Error("CRITICAL: Failed to restore .git directory: %v", err)
			logger.Error("Your repository is in an inconsistent state.")
			logger.Error("To recover manually:")
			logger.Error("1. Remove any worktree directories: %v", createdWorktrees)
			logger.Error("2. Run: mv %s %s", bareDir, gitDir)
			return
		}

		// Restore git config to normal (non-bare) state
		if err := git.RestoreNormalConfig(targetDir); err != nil {
			logger.Error("Failed to restore git config: %v", err)
			logger.Warning("Repository directory restored but git config may be inconsistent")
			logger.Warning("Run: git config --bool core.bare false")
		}

		logger.Success("Repository restored to original state")
	}()

	if branches != "" {
		var err error
		createdWorktrees, err = createWorktreesForConversion(targetDir, currentBranch, branches, verbose, ignoredFiles, &movedFiles)
		if err != nil {
			return err
		}
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
