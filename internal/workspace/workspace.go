package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	logger.Info("Creating worktrees...")

	for _, branch := range filteredBranches {
		sanitizedName := sanitizeBranchName(branch)
		worktreePath := filepath.Join("..", sanitizedName)

		if err := git.CreateWorktree(bareDir, worktreePath, branch, !verbose); err != nil {
			return fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
		}
	}

	for _, branch := range filteredBranches {
		sanitizedName := sanitizeBranchName(branch)
		fmt.Printf("  %s %s\n", styles.Render(&styles.Success, "✓"), sanitizedName)
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

	_, hasChanges, err := git.CheckGitChanges(targetDir)
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

	logger.Info("Moving .git directory to .bare...")
	if err := fs.RenameWithFallback(gitDir, bareDir); err != nil {
		return "", fmt.Errorf("failed to move .git to .bare: %w (recovery: ensure %s exists)", err, gitDir)
	}

	logger.Info("Configuring repository as bare...")
	if err := git.ConfigureBare(bareDir); err != nil {
		if renameErr := fs.RenameWithFallback(bareDir, gitDir); renameErr != nil {
			logger.Error("Failed to restore .git directory: %v", renameErr)
		}
		return "", fmt.Errorf("failed to configure as bare repository: %w (recovery: mv %s %s)", err, bareDir, gitDir)
	}

	logger.Debug("Repository converted to bare successfully, current branch: %s", currentBranch)
	return currentBranch, nil
}

// createMainWorktree creates worktree for current branch and moves files into it
func createMainWorktree(targetDir, currentBranch string, verbose bool) error {
	bareDir := filepath.Join(targetDir, ".bare")
	branches := []string{currentBranch}

	if err := createWorktreesOnly(bareDir, branches, verbose); err != nil {
		gitDir := filepath.Join(targetDir, ".git")
		if renameErr := fs.RenameWithFallback(bareDir, gitDir); renameErr != nil {
			logger.Error("Failed to restore .git directory: %v", renameErr)
		}
		return err
	}

	if err := moveFilesToFirstWorktree(targetDir, branches); err != nil {
		return err
	}

	if err := checkoutFirstWorktree(targetDir, currentBranch, verbose); err != nil {
		return err
	}

	return nil
}

// createWorktreesOnly creates worktrees for all specified branches
func createWorktreesOnly(bareDir string, branches []string, verbose bool) error {
	logger.Info("Creating worktrees...")
	for i, branch := range branches {
		sanitizedName := sanitizeBranchName(branch)
		worktreePath := filepath.Join("..", sanitizedName)

		if i == 0 {
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
					return fmt.Errorf("failed to create worktree for branch '%s': %w: %s", branch, err, strings.TrimSpace(stderr.String()))
				}
				return fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
			}
		} else {
			if err := git.CreateWorktree(bareDir, worktreePath, branch, !verbose); err != nil {
				return fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
			}
		}
	}
	return nil
}

// moveFilesToFirstWorktree moves all files from targetDir to the first worktree
func moveFilesToFirstWorktree(targetDir string, branches []string) error {
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

	var filesToMove []string
	for _, entry := range entries {
		if entry.Name() == ".bare" || worktreeDirs[entry.Name()] {
			continue
		}
		filesToMove = append(filesToMove, entry.Name())
	}

	logger.Debug("Preparing to move files to first worktree: %s -> %s, count: %d", targetDir, firstWorktreeAbsPath, len(filesToMove))
	logger.Info("Moving files to worktree...")

	movedCount := 0
	for _, entry := range entries {
		if entry.Name() == ".bare" || worktreeDirs[entry.Name()] {
			continue
		}

		oldPath := filepath.Join(targetDir, entry.Name())
		newPath := filepath.Join(firstWorktreeAbsPath, entry.Name())
		if err := fs.RenameWithFallback(oldPath, newPath); err != nil {
			return fmt.Errorf("failed to move %s to worktree: %w (moved %d/%d files)", entry.Name(), err, movedCount, len(filesToMove))
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

	logger.Info("Checking out worktree...")
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
			return fmt.Errorf("failed to checkout worktree: %w: %s", err, strings.TrimSpace(checkoutStderr.String()))
		}
		return fmt.Errorf("failed to checkout worktree: %w", err)
	}
	return nil
}

// createWorktreesForConversion creates worktrees for specified branches and moves files to first one
func createWorktreesForConversion(targetDir, branches string, verbose bool) error {
	bareDir := filepath.Join(targetDir, ".bare")
	cleanedBranches := parseBranches(branches, "")

	if len(cleanedBranches) == 0 {
		return fmt.Errorf("no valid branches specified")
	}

	if err := createWorktreesOnly(bareDir, cleanedBranches, verbose); err != nil {
		gitDir := filepath.Join(targetDir, ".git")
		if renameErr := fs.RenameWithFallback(bareDir, gitDir); renameErr != nil {
			logger.Error("Failed to restore .git directory: %v", renameErr)
		}
		return err
	}

	if err := moveFilesToFirstWorktree(targetDir, cleanedBranches); err != nil {
		return err
	}

	if err := checkoutFirstWorktree(targetDir, cleanedBranches[0], verbose); err != nil {
		return err
	}

	for _, branch := range cleanedBranches {
		sanitizedName := sanitizeBranchName(branch)
		fmt.Printf("  %s %s\n", styles.Render(&styles.Success, "✓"), sanitizedName)
	}

	return nil
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

	currentBranch, err := setupBareRepo(targetDir)
	if err != nil {
		return err
	}

	if branches != "" {
		if err := createWorktreesForConversion(targetDir, branches, verbose); err != nil {
			return err
		}
	} else {
		if err := createMainWorktree(targetDir, currentBranch, verbose); err != nil {
			return err
		}
	}

	gitFile := filepath.Join(targetDir, ".git")
	if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	return nil
}
