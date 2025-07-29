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
		Use:   "init [directory|remote-url]",
		Short: "Initialize or clone a Git repository optimized for worktrees",
		Long: `Initialize a new bare Git repository or clone an existing one with worktree-optimized structure.

Basic usage:
  grove init                    # Initialize new bare repository in current directory
  grove init <directory>        # Initialize new bare repository in specified directory
  grove init <remote-url>       # Clone existing repository with worktree setup
  grove init --convert          # Convert existing traditional repo to Grove structure

Multi-branch setup:
  grove init <remote-url> --branches=main,develop,feature/auth

Smart URL parsing supports:
  - GitHub: github.com/owner/repo, github.com/owner/repo/tree/branch, github.com/owner/repo/pull/123
  - GitLab: gitlab.com/owner/repo, gitlab.com/owner/repo/-/tree/branch
  - Bitbucket: bitbucket.org/owner/repo, bitbucket.org/owner/repo/src/branch
  - Azure DevOps: dev.azure.com/org/project/_git/repo
  - Codeberg: codeberg.org/owner/repo, codeberg.org/owner/repo/src/branch/branch
  - And standard Git URLs (.git suffix, SSH format)

Examples:
  grove init https://github.com/owner/repo
  grove init https://github.com/owner/repo/tree/main --branches=develop,staging
  grove init https://gitlab.com/owner/repo --branches=main,feature/auth
  grove init git@github.com:owner/repo.git --branches=main

The repository structure uses a .bare/ subdirectory for git objects and a .git file
pointing to it, allowing the main directory to function as a working directory.`,
		Args: func(cmd *cobra.Command, args []string) error {
			convert, _ := cmd.Flags().GetBool("convert")
			branches, _ := cmd.Flags().GetString("branches")

			if convert && len(args) > 0 {
				return errors.NewGroveError(errors.ErrCodeConfigInvalid, "cannot specify arguments when using --convert flag", nil)
			}
			if convert && branches != "" {
				return errors.NewGroveError(errors.ErrCodeConfigInvalid, "cannot use --branches flag with --convert", nil)
			}
			if branches != "" && len(args) == 0 {
				return errors.NewGroveError(errors.ErrCodeConfigInvalid, "--branches flag requires a remote URL", nil)
			}
			if !convert && len(args) > 1 {
				return errors.NewGroveError(errors.ErrCodeConfigInvalid, "too many arguments", nil)
			}
			return nil
		},
		RunE: runInit,
	}

	cmd.Flags().Bool("convert", false, "Convert existing traditional Git repository to Grove structure")
	cmd.Flags().String("branches", "", "Comma-separated list of branches to create worktrees for (e.g., 'main,develop,feature/auth')")

	return cmd
}

func runInit(cmd *cobra.Command, args []string) error {
	log := logger.WithComponent("init_command")
	start := time.Now()

	log.DebugOperation("starting grove init", "args", args)

	if !utils.IsGitAvailable() {
		err := errors.ErrGitNotFound(nil)
		log.ErrorOperation("git availability check failed", err)
		cmd.SilenceUsage = true
		return err
	}

	convert, _ := cmd.Flags().GetBool("convert")
	log.Debug("init mode determined", "convert", convert, "args_count", len(args))

	if convert {
		log.DebugOperation("running init convert", "duration", time.Since(start))
		if err := runInitConvert(); err != nil {
			cmd.SilenceUsage = true
			return err
		}
		return nil
	}

	var targetArg string
	if len(args) > 0 {
		targetArg = args[0]
	}

	if targetArg != "" {
		// First check if it's a URL (either smart platform URL or standard Git URL).
		urlInfo, err := utils.ParseGitPlatformURL(targetArg)
		switch {
		case err == nil:
			branches, _ := cmd.Flags().GetString("branches")

			// If URL contains branch information, add it to branches list.
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

			log.DebugOperation("running init remote with smart URL", "original_url", targetArg, "repo_url", urlInfo.RepoURL, "platform", urlInfo.Platform, "branch", urlInfo.BranchName, "pr", urlInfo.PRNumber, "branches", branches, "duration", time.Since(start))

			if urlInfo.PRNumber != "" {
				fmt.Printf("Detected %s pull request #%s\n", urlInfo.Platform, urlInfo.PRNumber)
			}
			if urlInfo.BranchName != "" {
				fmt.Printf("Detected %s branch: %s\n", urlInfo.Platform, urlInfo.BranchName)
			}

			if err := runInitRemote(urlInfo.RepoURL, branches); err != nil {
				cmd.SilenceUsage = true
				return err
			}
			return nil
		case utils.IsGitURL(targetArg):
			branches, _ := cmd.Flags().GetString("branches")
			log.DebugOperation("running init remote", "repo_url", targetArg, "branches", branches, "duration", time.Since(start))
			if err := runInitRemote(targetArg, branches); err != nil {
				cmd.SilenceUsage = true
				return err
			}
			return nil
		default:
			log.DebugOperation("running init local", "target_dir", targetArg, "duration", time.Since(start))
			if err := runInitLocal(targetArg); err != nil {
				cmd.SilenceUsage = true
				return err
			}
			return nil
		}
	} else {
		log.DebugOperation("running init local", "target_dir", targetArg, "duration", time.Since(start))
		if err := runInitLocal(targetArg); err != nil {
			cmd.SilenceUsage = true
			return err
		}
		return nil
	}
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

func runInitRemote(repoURL, branches string) error {
	return RunInitRemoteWithExecutor(shared.DefaultExecutorProvider.GetExecutor(), repoURL, branches)
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

func runInitConvert() error {
	return runInitConvertWithExecutor(shared.DefaultExecutorProvider.GetExecutor())
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
