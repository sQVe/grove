package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/config"
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
func CloneAndInitialize(url, path, branches string) error {
	if err := validateAndPrepareDirectory(path); err != nil {
		return err
	}

	bareDir := filepath.Join(path, ".bare")
	if err := git.Clone(url, bareDir); err != nil {
		_ = os.RemoveAll(bareDir)
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	logger.Debug("Repository cloned to %s", bareDir)

	if err := createGitFile(path, bareDir); err != nil {
		return err
	}

	if branches != "" {
		branchList := strings.Split(branches, ",")
		availableBranches, err := git.ListBranches(bareDir)
		if err != nil {
			return fmt.Errorf("failed to list branches: %w", err)
		}

		var missingBranches []string

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
			if !found {
				missingBranches = append(missingBranches, branch)
				continue
			}

			sanitizedName := sanitizeBranchName(branch)
			worktreePath := filepath.Join("..", sanitizedName)
			logger.Debug("Creating worktree for branch %s at %s", branch, worktreePath)

			if err := git.CreateWorktree(bareDir, worktreePath, branch); err != nil {
				return fmt.Errorf("failed to create worktree for branch '%s': %w", branch, err)
			}

			logger.Debug("Created worktree for branch %s", branch)
		}

		if len(missingBranches) > 0 {
			if len(availableBranches) == 0 {
				return fmt.Errorf("branches %v do not exist. Repository has no branches", missingBranches)
			}
			return fmt.Errorf("branches %v do not exist. Available branches: %v", missingBranches, availableBranches)
		}
	}

	return nil
}

// CloneAndInitializeWithVerbose clones a repository and creates a grove workspace with verbose control
func CloneAndInitializeWithVerbose(url, path, branches string, verbose bool) error {
	if err := validateAndPrepareDirectory(path); err != nil {
		return err
	}

	bareDir := filepath.Join(path, ".bare")

	// Count valid branches for output (but don't show confusing mapping)
	var validBranchCount int
	if branches != "" {
		branchList := strings.Split(branches, ",")
		for _, branch := range branchList {
			branch = strings.TrimSpace(branch)
			if branch != "" {
				validBranchCount++
			}
		}
	}

	var cloneErr error
	if verbose {
		logger.Info("Cloning repository...")
		cloneErr = git.Clone(url, bareDir)
	} else {
		stop := logger.StartSpinner("Cloning repository...")
		cloneErr = git.CloneQuiet(url, bareDir)
		stop()
		if cloneErr == nil {
			logger.Success("Repository cloned")
		}
	}

	if cloneErr != nil {
		_ = os.RemoveAll(bareDir)
		return fmt.Errorf("failed to clone repository: %w", cloneErr)
	}

	if err := createGitFile(path, bareDir); err != nil {
		return err
	}

	if branches != "" {
		var worktreeMessage string
		if validBranchCount == 1 {
			worktreeMessage = "Creating 1 worktree:"
		} else {
			worktreeMessage = fmt.Sprintf("Creating %d worktrees:", validBranchCount)
		}

		var stopWorktree func()
		if verbose {
			logger.Info("%s", worktreeMessage)
		} else {
			stopWorktree = logger.StartSpinner(worktreeMessage)
		}

		branchList := strings.Split(branches, ",")
		availableBranches, err := git.ListBranches(bareDir)
		if err != nil {
			if stopWorktree != nil {
				stopWorktree()
			}
			return fmt.Errorf("failed to list branches: %w", err)
		}

		var missingBranches []string

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
			if !found {
				missingBranches = append(missingBranches, branch)
				continue
			}

			sanitizedName := sanitizeBranchName(branch)
			worktreePath := filepath.Join("..", sanitizedName)

			var worktreeErr error
			if verbose {
				worktreeErr = git.CreateWorktree(bareDir, worktreePath, branch)
			} else {
				worktreeErr = git.CreateWorktreeQuiet(bareDir, worktreePath, branch)
			}

			if worktreeErr != nil {
				if stopWorktree != nil {
					stopWorktree()
				}
				return fmt.Errorf("failed to create worktree for branch '%s': %w", branch, worktreeErr)
			}
		}

		if stopWorktree != nil {
			stopWorktree()
			if !config.IsPlain() {
				logger.Info("%s", worktreeMessage)
			}
		}

		// Show individual worktree results after spinner stops
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
				sanitizedName := sanitizeBranchName(branch)
				fmt.Printf("  %s %s\n", styles.Render(&styles.Success, "✓"), sanitizedName)
			}
		}

		if len(missingBranches) > 0 {
			if len(availableBranches) == 0 {
				return fmt.Errorf("branches %v do not exist. Repository has no branches", missingBranches)
			}
			return fmt.Errorf("branches %v do not exist. Available branches: %v", missingBranches, availableBranches)
		}
	}

	return nil
}
