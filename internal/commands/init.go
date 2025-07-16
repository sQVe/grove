package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/utils"
)

func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [directory|remote-url]",
		Short: "Initialize or clone a Git repository optimized for worktrees",
		Long: `Initialize a new bare Git repository or clone an existing one with worktree-optimized structure.

Three modes:
  grove init                    # Initialize new bare repository in current directory
  grove init <directory>        # Initialize new bare repository in specified directory  
  grove init <remote-url>       # Clone existing repository with worktree setup
  grove init --convert          # Convert existing traditional repo to Grove structure

The repository structure uses a .bare/ subdirectory for git objects and a .git file
pointing to it, allowing the main directory to function as a working directory.`,
		Args: func(cmd *cobra.Command, args []string) error {
			convert, _ := cmd.Flags().GetBool("convert")
			if convert && len(args) > 0 {
				_ = cmd.Usage()
				return fmt.Errorf("cannot specify arguments when using --convert flag")
			}
			if !convert && len(args) > 1 {
				_ = cmd.Usage()
				return fmt.Errorf("too many arguments")
			}
			return nil
		},
		RunE: runInit,
	}

	cmd.Flags().Bool("convert", false, "Convert existing traditional Git repository to Grove structure")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("init_command")
	start := time.Now()

	log.InfoOperation("starting grove init", "args", args)

	if !utils.IsGitAvailable() {
		err := fmt.Errorf("git is not available in PATH")
		log.ErrorOperation("git availability check failed", err)
		return err
	}

	convert, _ := cmd.Flags().GetBool("convert")
	log.Debug("init mode determined", "convert", convert, "args_count", len(args))

	if convert {
		log.InfoOperation("running init convert", "duration", time.Since(start))
		return runInitConvert()
	}

	var targetArg string
	if len(args) > 0 {
		targetArg = args[0]
	}

	// Determine if argument is a URL or directory path.
	if targetArg != "" && utils.IsGitURL(targetArg) {
		log.InfoOperation("running init remote", "repo_url", targetArg, "duration", time.Since(start))
		return runInitRemote(targetArg)
	} else {
		log.InfoOperation("running init local", "target_dir", targetArg, "duration", time.Since(start))
		return runInitLocal(targetArg)
	}
}

func runInitLocal(targetDir string) error {
	log := logger.WithComponent("init_local")
	start := time.Now()

	log.InfoOperation("starting local repository initialization", "target_dir", targetDir)

	// Determine target directory
	if targetDir == "" {
		var err error
		targetDir, err = os.Getwd()
		if err != nil {
			log.ErrorOperation("failed to get current directory", err)
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		log.Debug("using current directory", "target_dir", targetDir)
	}

	absPath, err := filepath.Abs(targetDir)
	if err != nil {
		log.ErrorOperation("failed to resolve absolute path", err, "target_dir", targetDir)
		return fmt.Errorf("failed to resolve absolute path: %w", err)
	}
	log.Debug("resolved absolute path", "abs_path", absPath)

	log.Debug("creating target directory", "path", absPath)
	if err := os.MkdirAll(absPath, 0o750); err != nil {
		log.ErrorOperation("failed to create directory", err, "path", absPath)
		return fmt.Errorf("failed to create directory %s: %w", absPath, err)
	}

	gitPath := filepath.Join(absPath, ".git")
	log.Debug("checking for existing .git", "path", gitPath)
	if _, err := os.Stat(gitPath); err == nil {
		err := fmt.Errorf("directory %s already contains a .git file or directory", absPath)
		log.ErrorOperation("existing .git found", err, "path", gitPath)
		return err
	}

	bareDir := filepath.Join(absPath, ".bare")
	log.Debug("checking for existing .bare", "path", bareDir)
	if _, err := os.Stat(bareDir); err == nil {
		err := fmt.Errorf("directory %s already contains a .bare directory", absPath)
		log.ErrorOperation("existing .bare found", err, "path", bareDir)
		return err
	}

	// Initialize bare repository in .bare subdirectory.
	log.Debug("initializing bare repository", "bare_dir", bareDir)
	if err := git.InitBare(bareDir); err != nil {
		log.ErrorOperation("failed to initialize bare repository", err, "bare_dir", bareDir)
		return fmt.Errorf("failed to initialize bare repository: %w", err)
	}

	// Create .git file pointing to .bare directory.
	log.Debug("creating .git file", "target_dir", absPath, "bare_dir", bareDir)
	if err := git.CreateGitFile(absPath, bareDir); err != nil {
		log.ErrorOperation("failed to create .git file", err, "target_dir", absPath, "bare_dir", bareDir)
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	log.InfoOperation("local repository initialization completed", "target_dir", absPath, "bare_dir", bareDir, "duration", time.Since(start))

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
	log := logger.WithComponent("init_remote")
	start := time.Now()

	log.InfoOperation("starting remote repository initialization", "repo_url", repoURL)

	log.Debug("validating and preparing directory")
	targetDir, err := validateAndPrepareDirectory()
	if err != nil {
		log.ErrorOperation("directory validation failed", err, "repo_url", repoURL)
		return err
	}
	log.Debug("directory validation completed", "target_dir", targetDir)

	bareDir := filepath.Join(targetDir, ".bare")
	log.Debug("determined bare directory path", "bare_dir", bareDir)

	log.Debug("cloning and setting up repository")
	if err := cloneAndSetupRepository(executor, repoURL, targetDir, bareDir); err != nil {
		log.ErrorOperation("clone and setup failed", err, "repo_url", repoURL, "target_dir", targetDir)
		return err
	}

	log.Debug("configuring remote tracking")
	if err := configureRemoteTracking(executor, targetDir); err != nil {
		log.ErrorOperation("remote tracking configuration failed", err, "target_dir", targetDir)
		return err
	}

	// Create default worktree for the detected default branch
	fmt.Println("Creating default worktree...")
	log.Debug("creating default worktree", "target_dir", targetDir)
	if err := git.CreateDefaultWorktreeWithExecutor(executor, targetDir); err != nil {
		// Don't fail the init if worktree creation fails
		log.Warn("default worktree creation failed but continuing", "error", err, "target_dir", targetDir)
		fmt.Printf("Warning: failed to create default worktree: %v\n", err)
	}

	log.InfoOperation("remote repository initialization completed", "repo_url", repoURL, "target_dir", targetDir, "duration", time.Since(start))
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
	// Clone as bare repository into .bare subdirectory.
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
	if err := git.ConfigureRemoteTrackingWithExecutor(executor, "origin"); err != nil {
		return fmt.Errorf("failed to configure remote tracking: %w", err)
	}

	if err := git.SetupUpstreamBranchesWithExecutor(executor, "origin"); err != nil {
		// Don't fail if this doesn't work - it's not critical.
		fmt.Printf("Warning: failed to set up upstream branches: %v\n", err)
	}

	return nil
}

func printSuccessMessage(targetDir, bareDir string) {
	fmt.Printf("Successfully cloned and configured repository in %s\n", targetDir)
	fmt.Printf("Git objects stored in: %s\n", bareDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  cd <worktree-name>          # Switch to the created worktree")
	fmt.Println("  grove create <branch-name>  # Create a worktree for a branch")
	fmt.Println("  grove list                  # List all worktrees")
}

func runInitConvert() error {
	return runInitConvertWithExecutor(git.DefaultExecutor)
}

func runInitConvertWithExecutor(executor git.GitExecutor) error {
	log := logger.WithComponent("init_convert")
	start := time.Now()

	log.InfoOperation("starting repository conversion to Grove structure")

	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.ErrorOperation("failed to get current directory", err)
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	log.Debug("determined current directory", "current_dir", currentDir)

	// Check if this is a traditional Git repository
	log.Debug("checking repository type", "current_dir", currentDir)
	if !git.IsTraditionalRepo(currentDir) {
		if git.IsGroveRepo(currentDir) {
			err := fmt.Errorf("directory %s is already a Grove repository", currentDir)
			log.ErrorOperation("already a Grove repository", err, "current_dir", currentDir)
			return err
		}
		err := fmt.Errorf("directory %s does not contain a traditional Git repository (.git directory not found)", currentDir)
		log.ErrorOperation("not a traditional Git repository", err, "current_dir", currentDir)
		return err
	}
	log.Debug("confirmed traditional Git repository", "current_dir", currentDir)

	fmt.Printf("Converting traditional Git repository to Grove structure...\n")
	fmt.Printf("Repository: %s\n", currentDir)

	// Perform the conversion
	log.Debug("performing conversion", "current_dir", currentDir)
	if err := git.ConvertToGroveStructureWithExecutor(executor, currentDir); err != nil {
		log.ErrorOperation("conversion failed", err, "current_dir", currentDir)
		return fmt.Errorf("failed to convert repository: %w", err)
	}

	// Create default worktree for the current branch
	fmt.Println("Creating default worktree for current branch...")
	log.Debug("creating default worktree", "current_dir", currentDir)
	if err := git.CreateDefaultWorktreeWithExecutor(executor, currentDir); err != nil {
		// Don't fail the conversion if worktree creation fails
		log.Warn("default worktree creation failed but continuing", "error", err, "current_dir", currentDir)
		fmt.Printf("Warning: failed to create default worktree: %v\n", err)
	}

	// Print success message
	bareDir := filepath.Join(currentDir, ".bare")
	log.InfoOperation("repository conversion completed successfully", "current_dir", currentDir, "bare_dir", bareDir, "duration", time.Since(start))

	fmt.Printf("Successfully converted repository to Grove structure in %s\n", currentDir)
	fmt.Printf("Git objects moved to: %s\n", bareDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  grove create <branch-name>  # Create a worktree for a branch")
	fmt.Println("  grove list                  # List all worktrees")

	return nil
}
