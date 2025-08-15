package init

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/commands/shared"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/utils"
)

func NewInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize or clone a Git repository optimized for worktrees",
		Long: `Initialize or clone a Git repository with worktree-optimized structure.

Subcommands:
  new      Create a new empty repository
  clone    Clone an existing repository with worktree setup
  convert  Convert existing traditional Git repository to Grove structure

Examples:
  grove init new                          # Create repository in current directory
  grove init new myproject                # Create repository in myproject/ directory
  grove init clone <url>                  # Clone repository
  grove init clone <url> --branches=main,develop  # Clone and create worktrees
  grove init convert                      # Convert current repo to Grove structure`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			fmt.Fprintf(cmd.ErrOrStderr(), "Error: unknown command %q for %q\n\nRun 'grove init --help' for usage information\n", args[0], cmd.CommandPath())
			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	cmd.AddCommand(newNewCmd())
	cmd.AddCommand(newCloneCmd())
	cmd.AddCommand(newConvertCmd())

	return cmd
}

func newNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new [directory]",
		Short: "Create a new empty repository",
		Long: `Create a new empty Git repository optimized for worktrees.

If no directory is specified, initializes in the current directory.
If a directory is specified, creates it and initializes the repository there.

Git objects are stored in .bare/ subdirectory.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("accepts at most 1 arg(s), received %d\n\nRun 'grove init new --help' for usage information", len(args))
			}
			return nil
		},
		SilenceUsage: true,
		RunE:         runInitNew,
	}
}

func newCloneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clone <url>",
		Short: "Clone an existing repository with worktree setup",
		Long: `Clone an existing Git repository with worktree-optimized structure.

Creates a directory based on the repository name and clones into it.
Supports GitHub, GitLab, Bitbucket, Azure DevOps URLs with branch detection.

Git objects are stored in .bare/ subdirectory.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("accepts 1 arg(s), received %d\n\nRun 'grove init clone --help' for usage information", len(args))
			}
			return nil
		},
		SilenceUsage: true,
		RunE:         runInitClone,
	}

	cmd.Flags().String("branches", "", "Comma-separated list of branches to create worktrees for (e.g., 'main,develop,feature/auth')")

	return cmd
}

func newConvertCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "convert",
		Short: "Convert existing traditional Git repository to Grove structure",
		Long: `Convert an existing traditional Git repository to Grove's worktree-optimized structure.

Must be run from within a traditional Git repository (with .git directory).
Moves existing .git directory to .bare/ and sets up worktree structure.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) != 0 {
				return fmt.Errorf("accepts 0 arg(s), received %d\n\nRun 'grove init convert --help' for usage information", len(args))
			}
			return nil
		},
		SilenceUsage: true,
		RunE:         runInitConvert,
	}
}

func runInitNew(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("init_new")

	log.DebugOperation("starting new repository creation", "args", args)

	if !utils.IsGitAvailable() {
		err := errors.ErrGitNotFound(nil)
		log.ErrorOperation("git availability check failed", err)
		cmd.SilenceUsage = true
		return err
	}

	var targetDir string
	if len(args) > 0 {
		targetDir = args[0]
	}

	if err := runInitLocal(targetDir); err != nil {
		cmd.SilenceUsage = true
		return err
	}

	return nil
}

func runInitClone(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("init_clone")
	start := time.Now()

	repoURL := args[0]
	log.DebugOperation("starting repository clone", "repo_url", repoURL)

	if !utils.IsGitAvailable() {
		err := errors.ErrGitNotFound(nil)
		log.ErrorOperation("git availability check failed", err)
		cmd.SilenceUsage = true
		return err
	}

	branches, _ := cmd.Flags().GetString("branches")

	// Parse smart URLs and extract branch information
	urlInfo, err := utils.ParseGitPlatformURL(repoURL)
	if err == nil {
		// If URL contains branch information, add it to branches list
		if urlInfo.BranchName != "" {
			if branches == "" {
				branches = urlInfo.BranchName
			} else {
				branchList := strings.Split(branches, ",")
				found := false
				for _, b := range branchList {
					if strings.TrimSpace(b) == urlInfo.BranchName {
						found = true
						break
					}
				}
				if !found {
					branches = urlInfo.BranchName + "," + branches
				}
			}
		}

		log.DebugOperation("using smart URL", "original_url", repoURL, "repo_url", urlInfo.RepoURL, "platform", urlInfo.Platform, "branch", urlInfo.BranchName, "pr", urlInfo.PRNumber, "branches", branches, "duration", time.Since(start))

		if urlInfo.PRNumber != "" {
			fmt.Printf("Detected %s pull request #%s\n", urlInfo.Platform, urlInfo.PRNumber)
		}
		if urlInfo.BranchName != "" {
			fmt.Printf("Detected %s branch: %s\n", urlInfo.Platform, urlInfo.BranchName)
		}

		repoURL = urlInfo.RepoURL
	} else if !utils.IsGitURL(repoURL) {
		err := errors.NewGroveError(errors.ErrCodeConfigInvalid, fmt.Sprintf("invalid Git URL: %s", repoURL), nil)
		log.ErrorOperation("URL validation failed", err)
		cmd.SilenceUsage = true
		return err
	}

	if err := runInitRemoteWithDirectory(repoURL, branches); err != nil {
		cmd.SilenceUsage = true
		return err
	}

	return nil
}

func runInitConvert(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("init_convert")

	log.DebugOperation("starting repository conversion")

	if !utils.IsGitAvailable() {
		err := errors.ErrGitNotFound(nil)
		log.ErrorOperation("git availability check failed", err)
		cmd.SilenceUsage = true
		return err
	}

	if err := runInitConvertWithExecutor(shared.DefaultExecutorProvider.GetExecutor()); err != nil {
		cmd.SilenceUsage = true
		return err
	}

	return nil
}

func runInitLocal(targetDir string) error {
	log := logger.WithComponent("init_local")
	start := time.Now()

	log.DebugOperation("starting local repository initialization", "target_dir", targetDir)

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
		err := errors.ErrRepoExists(absPath).
			WithContext("conflict", ".git file or directory")
		log.Debug("existing .git found", "path", gitPath, "error", err)
		return err
	}

	bareDir := filepath.Join(absPath, ".bare")
	log.Debug("checking for existing .bare", "path", bareDir)
	if _, err := os.Stat(bareDir); err == nil {
		err := errors.ErrRepoExists(absPath).
			WithContext("conflict", ".bare directory")
		log.Debug("existing .bare found", "path", bareDir, "error", err)
		return err
	}

	log.Debug("initializing bare repository", "bare_dir", bareDir)
	if err := git.InitBare(bareDir); err != nil {
		log.ErrorOperation("failed to initialize bare repository", err, "bare_dir", bareDir)
		return fmt.Errorf("failed to initialize bare repository: %w", err)
	}

	log.Debug("creating .git file", "target_dir", absPath, "bare_dir", bareDir)
	if err := git.CreateGitFile(absPath, bareDir); err != nil {
		log.ErrorOperation("failed to create .git file", err, "target_dir", absPath, "bare_dir", bareDir)
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	log.DebugOperation("local repository initialization completed", "target_dir", absPath, "bare_dir", bareDir, "duration", time.Since(start))

	fmt.Printf("Initialized bare Git repository in %s\n", absPath)
	fmt.Printf("Git objects stored in: %s\n", bareDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  grove create <branch-name>  # Create your first worktree")
	fmt.Println("  grove list                  # List all worktrees")

	return nil
}

func runInitRemoteWithDirectory(repoURL, branches string) error {
	return RunInitRemoteWithDirectoryAndExecutor(shared.DefaultExecutorProvider.GetExecutor(), repoURL, branches)
}

func RunInitRemoteWithDirectoryAndExecutor(executor git.GitExecutor, repoURL, branches string) error {
	log := logger.WithComponent("init_remote_with_dir")
	start := time.Now()

	log.DebugOperation("starting remote repository initialization with directory creation", "repo_url", repoURL)

	// Extract repository name from URL
	repoName := extractRepositoryName(repoURL)
	if repoName == "" {
		err := errors.NewGroveError(errors.ErrCodeConfigInvalid, "unable to extract repository name from URL", nil)
		log.ErrorOperation("repository name extraction failed", err, "repo_url", repoURL)
		return err
	}

	// Get current directory
	currentDir, err := os.Getwd()
	if err != nil {
		log.ErrorOperation("failed to get current directory", err)
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Create target directory
	targetDir := filepath.Join(currentDir, repoName)
	log.Debug("creating target directory", "target_dir", targetDir)

	// Check if directory already exists
	if _, err := os.Stat(targetDir); err == nil {
		err := errors.ErrRepoExists(targetDir).
			WithContext("conflict", "directory already exists")
		log.Debug("target directory already exists", "target_dir", targetDir, "error", err)
		return err
	}

	// Create the directory
	if err := os.MkdirAll(targetDir, 0o750); err != nil {
		log.ErrorOperation("failed to create target directory", err, "target_dir", targetDir)
		return fmt.Errorf("failed to create directory %s: %w", targetDir, err)
	}

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

	fmt.Println("Creating default worktree...")
	log.Debug("creating default worktree", "target_dir", targetDir)
	if err := git.CreateDefaultWorktreeWithExecutor(executor, targetDir); err != nil {
		log.Warn("default worktree creation failed but continuing", "error", err, "target_dir", targetDir)
		fmt.Printf("Warning: failed to create default worktree: %v\n", err)
	}

	if branches != "" {
		fmt.Println("Creating additional worktrees...")
		branchList := ParseBranches(branches)
		log.Debug("creating additional worktrees", "branches", branchList)
		if err := CreateAdditionalWorktrees(executor, targetDir, branchList); err != nil {
			log.Warn("additional worktree creation failed but continuing", "error", err, "branches", branchList)
			fmt.Printf("Warning: failed to create additional worktrees: %v\n", err)
		}
	}

	log.DebugOperation("remote repository initialization completed", "repo_url", repoURL, "target_dir", targetDir, "duration", time.Since(start))
	printSuccessMessage(targetDir, bareDir)
	return nil
}

func RunInitRemoteWithExecutor(executor git.GitExecutor, repoURL, branches string) error {
	log := logger.WithComponent("init_remote")
	start := time.Now()

	log.DebugOperation("starting remote repository initialization", "repo_url", repoURL)

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

	fmt.Println("Creating default worktree...")
	log.Debug("creating default worktree", "target_dir", targetDir)
	if err := git.CreateDefaultWorktreeWithExecutor(executor, targetDir); err != nil {
		log.Warn("default worktree creation failed but continuing", "error", err, "target_dir", targetDir)
		fmt.Printf("Warning: failed to create default worktree: %v\n", err)
	}

	if branches != "" {
		fmt.Println("Creating additional worktrees...")
		branchList := ParseBranches(branches)
		log.Debug("creating additional worktrees", "branches", branchList)
		if err := CreateAdditionalWorktrees(executor, targetDir, branchList); err != nil {
			log.Warn("additional worktree creation failed but continuing", "error", err, "branches", branchList)
			fmt.Printf("Warning: failed to create additional worktrees: %v\n", err)
		}
	}

	log.DebugOperation("remote repository initialization completed", "repo_url", repoURL, "target_dir", targetDir, "duration", time.Since(start))
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
			return "", errors.ErrRepoInvalid(targetDir, "directory is not empty")
		}
	}

	return targetDir, nil
}

func cloneAndSetupRepository(executor git.GitExecutor, repoURL, targetDir, bareDir string) error {
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

func runInitConvertWithExecutor(executor git.GitExecutor) error {
	log := logger.WithComponent("init_convert")
	start := time.Now()

	log.DebugOperation("starting repository conversion to Grove structure")

	currentDir, err := os.Getwd()
	if err != nil {
		log.ErrorOperation("failed to get current directory", err)
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	log.Debug("determined current directory", "current_dir", currentDir)

	log.Debug("checking repository type", "current_dir", currentDir)
	if !git.IsTraditionalRepo(currentDir) {
		if git.IsGroveRepo(currentDir) {
			err := errors.ErrRepoExists(currentDir).
				WithContext("type", "Grove repository")
			log.Debug("already a Grove repository", "current_dir", currentDir, "error", err)
			return err
		}
		err := errors.ErrRepoNotFound(currentDir).
			WithContext("expected", "traditional Git repository (.git directory)")
		log.Debug("not a traditional Git repository", "current_dir", currentDir, "error", err)
		return err
	}
	log.Debug("confirmed traditional Git repository", "current_dir", currentDir)

	fmt.Printf("Converting traditional Git repository to Grove structure...\n")
	fmt.Printf("Repository: %s\n", currentDir)

	log.Debug("performing conversion", "current_dir", currentDir)
	if err := git.ConvertToGroveStructureWithExecutor(executor, currentDir); err != nil {
		log.Debug("conversion failed", "error", err, "current_dir", currentDir)
		return fmt.Errorf("failed to convert repository: %w", err)
	}

	fmt.Println("Creating default worktree for current branch...")
	log.Debug("creating default worktree", "current_dir", currentDir)
	if err := git.CreateDefaultWorktreeWithExecutor(executor, currentDir); err != nil {
		// Don't fail the conversion if worktree creation fails.
		log.Warn("default worktree creation failed but continuing", "error", err, "current_dir", currentDir)
		fmt.Printf("Warning: failed to create default worktree: %v\n", err)
	}

	bareDir := filepath.Join(currentDir, ".bare")
	log.DebugOperation("repository conversion completed successfully", "current_dir", currentDir, "bare_dir", bareDir, "duration", time.Since(start))

	fmt.Printf("Successfully converted repository to Grove structure in %s\n", currentDir)
	fmt.Printf("Git objects moved to: %s\n", bareDir)
	fmt.Println("\nNext steps:")
	fmt.Println("  grove create <branch-name>  # Create a worktree for a branch")
	fmt.Println("  grove list                  # List all worktrees")

	return nil
}

func extractRepositoryName(repoURL string) string {
	// Remove .git suffix if present
	url := strings.TrimSuffix(repoURL, ".git")

	// Extract the last part of the path
	parts := strings.Split(url, "/")
	if len(parts) == 0 {
		return ""
	}

	repoName := parts[len(parts)-1]

	// Handle SSH URLs like git@github.com:owner/repo
	if strings.Contains(repoName, ":") {
		colonParts := strings.Split(repoName, ":")
		if len(colonParts) > 1 {
			pathParts := strings.Split(colonParts[len(colonParts)-1], "/")
			if len(pathParts) > 0 {
				repoName = pathParts[len(pathParts)-1]
			}
		}
	}

	// Validate the repository name
	if repoName == "" || repoName == "." || repoName == ".." {
		return ""
	}

	return repoName
}

func ParseBranches(branchesStr string) []string {
	if branchesStr == "" {
		return nil
	}

	var branches []string
	for _, branch := range strings.Split(branchesStr, ",") {
		branch = strings.TrimSpace(branch)
		if branch != "" && isValidBranchName(branch) {
			branches = append(branches, branch)
		}
	}
	return branches
}

func isValidBranchName(name string) bool {
	if name == "" {
		return false
	}

	// Git branch naming rules (simplified):.
	// - Cannot start or end with /.
	// - Cannot contain ..
	// - Cannot contain ASCII control characters.
	// - Cannot contain spaces (in most contexts).
	// - Cannot be just -.
	// - Cannot start with -.
	// - Cannot end with .lock.

	if name == "-" || strings.HasPrefix(name, "-") || strings.HasSuffix(name, ".lock") {
		return false
	}

	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		return false
	}

	if strings.Contains(name, "..") {
		return false
	}

	for _, r := range name {
		if r < 32 || r == 127 || r == ' ' {
			return false
		}
	}

	invalidChars := []string{"~", "^", ":", "?", "*", "[", "\\"}
	for _, char := range invalidChars {
		if strings.Contains(name, char) {
			return false
		}
	}

	return true
}

func CreateAdditionalWorktrees(executor git.GitExecutor, targetDir string, branches []string) error {
	if len(branches) == 0 {
		return nil
	}

	log := logger.WithComponent("additional_worktrees")

	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(targetDir); err != nil {
		return fmt.Errorf("failed to change to directory %s: %w", targetDir, err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	output, err := executor.Execute("branch", "-r")
	if err != nil {
		log.Warn("failed to get remote branches", "error", err)
		return fmt.Errorf("failed to get remote branches: %w", err)
	}

	availableBranches := make(map[string]bool)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "->") {
			continue
		}
		if strings.HasPrefix(line, "origin/") {
			branchName := strings.TrimPrefix(line, "origin/")
			availableBranches[branchName] = true
		}
	}

	log.Debug("available remote branches", "branches", availableBranches)

	for _, branch := range branches {
		if !availableBranches[branch] {
			log.Warn("branch not found on remote", "branch", branch)
			fmt.Printf("Warning: branch '%s' not found on remote, skipping\n", branch)
			continue
		}

		dirName := git.BranchToDirectoryName(branch)
		worktreePath := filepath.Join(targetDir, dirName)

		if _, err := os.Stat(worktreePath); err == nil {
			log.Debug("worktree already exists", "branch", branch, "path", worktreePath)
			continue
		}

		fmt.Printf("Creating worktree for branch '%s'...\n", branch)
		log.Debug("creating worktree", "branch", branch, "path", worktreePath)

		_, err := git.CreateWorktreeFromExistingBranch(executor, branch, targetDir)
		if err != nil {
			log.Warn("failed to create worktree", "branch", branch, "error", err)
			fmt.Printf("Warning: failed to create worktree for branch '%s': %v\n", branch, err)
			continue
		}

		log.DebugOperation("worktree created successfully", "branch", branch, "path", worktreePath)
	}

	return nil
}
