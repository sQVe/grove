package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/validation"
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

// validateBranches validates requested branches against available branches
func validateBranches(
	branches string, availableBranches []string,
) (valid, missing []string) {
	branchList := strings.Split(branches, ",")

	for _, branch := range branchList {
		branch = strings.TrimSpace(branch)
		if branch == "" {
			continue
		}

		found := false
		for _, availBranch := range availableBranches {
			if strings.TrimSpace(availBranch) == branch {
				found = true
				break
			}
		}

		if found {
			valid = append(valid, branch)
		} else {
			missing = append(missing, branch)
		}
	}

	return valid, missing
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
		if validation.DirectoryExists(bareDir) {
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

	if validation.DirectoryExists(path) {
		logger.Debug("Directory %s exists, checking if empty", path)

		isEmpty, err := validation.IsEmptyDir(path)
		if err != nil {
			return fmt.Errorf("failed to check if directory is empty: %w", err)
		}
		if !isEmpty {
			return fmt.Errorf("directory %s is not empty", path)
		}
	} else {
		logger.Debug("Creating new directory: %s", path)
		if err := os.MkdirAll(path, fs.DirGit); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", path, err)
		}
	}

	return nil
}

// createGitFile creates the .git file pointing to .bare directory
func createGitFile(path, bareDir string) error {
	gitFile := filepath.Join(path, ".git")
	if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
		_ = os.RemoveAll(bareDir)
		return fmt.Errorf("failed to create .git file: %w", err)
	}
	logger.Debug("Created .git file pointing to .bare")
	return nil
}

// cloneWithProgress clones a repository with progress indication
func cloneWithProgress(url, bareDir string, verbose bool) error {
	if verbose {
		logger.Info("Cloning repository...")
		return git.Clone(url, bareDir, false)
	}

	stop := logger.StartSpinner("Cloning repository...")
	err := git.Clone(url, bareDir, true)
	stop()

	if err == nil {
		logger.Success("Repository cloned")
	}

	return err
}

// createWorktreesFromBranches creates worktrees for the specified branches
func createWorktreesFromBranches(bareDir, branches string, verbose bool) error {
	availableBranches, err := git.ListBranches(bareDir)
	if err != nil {
		return fmt.Errorf("failed to list branches: %w", err)
	}

	validBranches, missingBranches := validateBranches(branches, availableBranches)

	if len(missingBranches) > 0 {
		if len(availableBranches) == 0 {
			return fmt.Errorf("branches %v do not exist. Repository has no branches", missingBranches)
		}
		return fmt.Errorf("branches %v do not exist. Available branches: %v", missingBranches, availableBranches)
	}

	var stop func()
	if verbose {
		logger.Info("Creating worktrees:")
	} else {
		stop = logger.StartSpinner("Creating worktrees...")
	}

	for _, branch := range validBranches {
		sanitizedName := sanitizeBranchName(branch)
		worktreePath := filepath.Join("..", sanitizedName)

		if err := git.CreateWorktree(bareDir, worktreePath, branch, !verbose); err != nil {
			if stop != nil {
				stop()
			}
			return fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
		}
	}

	if stop != nil {
		stop()
		logger.Success("Creating worktrees:")
	}

	for _, branch := range validBranches {
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
	if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
		return fmt.Errorf("failed to create .bare directory: %w", err)
	}
	logger.Debug("Created .bare directory at %s", bareDir)

	if err := git.InitBare(bareDir); err != nil {
		_ = os.RemoveAll(bareDir)
		return fmt.Errorf("failed to initialize bare git repository: %w", err)
	}
	logger.Debug("Git bare repository initialized successfully")

	return createGitFile(path, bareDir)
}

// CloneAndInitialize clones a repository and creates a grove workspace in the specified directory
func CloneAndInitialize(url, path, branches string, verbose bool) error {
	if err := validateAndPrepareDirectory(path); err != nil {
		return err
	}

	bareDir := filepath.Join(path, ".bare")

	if err := cloneWithProgress(url, bareDir, verbose); err != nil {
		_ = os.RemoveAll(bareDir)
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	if err := createGitFile(path, bareDir); err != nil {
		return err
	}

	if branches != "" {
		return createWorktreesFromBranches(bareDir, branches, verbose)
	}

	return nil
}
