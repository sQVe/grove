package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
)

// NewInitCmd creates the init command for initializing a bare repository.
func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [directory]",
		Short: "Initialize a bare Git repository optimized for worktrees",
		Long: `Initialize a bare Git repository in the specified directory (or current directory).
This creates a repository structure optimized for Git worktree management.`,
		Args: cobra.MaximumNArgs(1),
		RunE: runInit,
	}

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	// Determine target directory
	var targetDir string
	if len(args) > 0 {
		targetDir = args[0]
	} else {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %v", err)
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("failed to resolve absolute path: %v", err)
	}

	// Check if git is available
	if !git.IsGitAvailable() {
		return fmt.Errorf("git is not available in PATH")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %v", absPath, err)
	}

	// Change to target directory
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %v", err)
	}

	if err := os.Chdir(absPath); err != nil {
		return fmt.Errorf("failed to change to directory %s: %v", absPath, err)
	}

	// Restore original directory on exit
	defer func() {
		_ = os.Chdir(originalDir)
	}()

	// Check if already a git repository
	isRepo, err := git.IsGitRepository()
	if err != nil {
		return fmt.Errorf("failed to check if directory is a git repository: %v", err)
	}
	if isRepo {
		return fmt.Errorf("directory %s is already a git repository", absPath)
	}

	// Initialize bare repository
	if _, err := git.ExecuteGit("init", "--bare"); err != nil {
		return fmt.Errorf("failed to initialize bare repository: %v", err)
	}

	fmt.Printf("Initialized bare Git repository in %s\n", absPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  grove create <branch-name>  # Create your first worktree")
	fmt.Println("  grove list                  # List all worktrees")

	return nil
}
