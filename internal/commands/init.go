package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/utils"
)

func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [directory|remote-url]",
		Short: "Initialize or clone a Git repository optimized for worktrees",
		Long: `Initialize a new bare Git repository or clone an existing one with worktree-optimized structure.

Two modes:
  grove init                    # Initialize new bare repository in current directory
  grove init <directory>        # Initialize new bare repository in specified directory  
  grove init <remote-url>       # Clone existing repository with worktree setup

The repository structure uses a .bare/ subdirectory for git objects and a .git file
pointing to it, allowing the main directory to function as a working directory.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	if !utils.IsGitAvailable() {
		return fmt.Errorf("git is not available in PATH")
	}

	var targetArg string
	if len(args) > 0 {
		targetArg = args[0]
	}

	// Determine if argument is a URL or directory path.
	if targetArg != "" && utils.IsGitURL(targetArg) {
		return runInitRemote(targetArg)
	} else {
		return runInitLocal(targetArg)
	}
}

func runInitLocal(targetDir string) error {
	// Determine target directory
	if targetDir == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	if err := os.MkdirAll(absPath, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", absPath, err)
	}

	gitPath := filepath.Join(absPath, ".git")
	if _, err := os.Stat(gitPath); err == nil {
		return fmt.Errorf("directory %s already contains a .git file or directory", absPath)
	}

	bareDir := filepath.Join(absPath, ".bare")
	if _, err := os.Stat(bareDir); err == nil {
		return fmt.Errorf("directory %s already contains a .bare directory", absPath)
	}

	// Initialize bare repository in .bare subdirectory. AI: Why are we doing this? Add short explanation.
	if err := git.InitBare(bareDir); err != nil {
		return fmt.Errorf("failed to initialize bare repository: %w", err)
	}

	// Create .git file pointing to .bare directory.
	if err := git.CreateGitFile(absPath, bareDir); err != nil {
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	fmt.Printf("Initialized bare Git repository in %s\n", absPath)
	fmt.Printf("Git objects stored in: %s\n", bareDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  grove create <branch-name>  # Create your first worktree")
	fmt.Println("  grove list                  # List all worktrees")

	return nil
}

func runInitRemote(repoURL string) error {
	return runInitRemoteWithExecutor(git.DefaultExecutor, repoURL)
}

func runInitRemoteWithExecutor(executor git.GitExecutor, repoURL string) error {
	targetDir, err := validateAndPrepareDirectory()
	if err != nil {
		return err
	}

	bareDir := filepath.Join(targetDir, ".bare")

	if err := cloneAndSetupRepository(executor, repoURL, targetDir, bareDir); err != nil {
		return err
	}

	if err := configureRemoteTracking(executor, targetDir); err != nil {
		return err
	}

	printSuccessMessage(targetDir, bareDir)
	return nil
}

func validateAndPrepareDirectory() (string, error) {
	targetDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get current directory: %w", err)
	}

	entries, err := os.ReadDir(targetDir)
	if err != nil {
		return "", fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if !utils.IsHidden(entry.Name()) {
			return "", fmt.Errorf("directory %s is not empty", targetDir)
		}
	}

	return targetDir, nil
}

func cloneAndSetupRepository(executor git.GitExecutor, repoURL, targetDir, bareDir string) error {
	// Clone as bare repository into .bare subdirectory. AI: Why are we doing this? Add short explanation.
	fmt.Printf("Cloning %s...\n", repoURL)
	if err := git.CloneBareWithExecutor(executor, repoURL, bareDir); err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	if err := git.CreateGitFile(targetDir, bareDir); err != nil {
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	return nil
}

func configureRemoteTracking(executor git.GitExecutor, targetDir string) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(targetDir); err != nil {
		return fmt.Errorf("failed to change to directory %s: %w", targetDir, err)
	}

	defer func() {
		_ = os.Chdir(originalDir)
	}()

	fmt.Println("Configuring remote tracking...")
	if err := git.ConfigureRemoteTrackingWithExecutor(executor); err != nil {
		return fmt.Errorf("failed to configure remote tracking: %w", err)
	}

	if err := git.SetupUpstreamBranchesWithExecutor(executor); err != nil {
		// Don't fail if this doesn't work - it's not critical.
		fmt.Printf("Warning: failed to set up upstream branches: %v\n", err)
	}

	return nil
}

func printSuccessMessage(targetDir, bareDir string) {
	fmt.Printf("Successfully cloned and configured repository in %s\n", targetDir)
	fmt.Printf("Git objects stored in: %s\n", bareDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  grove create <branch-name>  # Create a worktree for a branch")
	fmt.Println("  grove list                  # List all worktrees")
}
