package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/retry"
)

// GitExecutor defines the interface for executing git commands.
type GitExecutor interface {
	Execute(args ...string) (string, error)
	ExecuteWithContext(ctx context.Context, args ...string) (string, error)
}

// DefaultGitExecutor implements GitExecutor using real git commands.
type DefaultGitExecutor struct{}

// Execute runs a real git command.
func (e *DefaultGitExecutor) Execute(args ...string) (string, error) {
	return ExecuteGit(args...)
}

// ExecuteWithContext runs a real git command with context support for cancellation.
func (e *DefaultGitExecutor) ExecuteWithContext(ctx context.Context, args ...string) (string, error) {
	return ExecuteGitWithContext(ctx, args...)
}

// DefaultExecutor is the default git command executor.
var DefaultExecutor GitExecutor = &DefaultGitExecutor{}

// isNetworkError checks if an error is likely a network-related issue
func isNetworkError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "network") ||
		strings.Contains(errStr, "connection") ||
		strings.Contains(errStr, "dns") ||
		strings.Contains(errStr, "resolve") ||
		strings.Contains(errStr, "unreachable")
}

// isAuthError checks if an error is likely an authentication issue
func isAuthError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "authentication") ||
		strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "access denied") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "403") ||
		strings.Contains(errStr, "401")
}

// validatePaths validates that the provided paths are safe to use.
func validatePaths(mainDir, bareDir string) error {
	// Check for directory traversal in original paths before conversion
	if strings.Contains(mainDir, "..") || strings.Contains(bareDir, "..") {
		return errors.ErrPathTraversal("paths contain directory traversal sequences")
	}

	// Convert to absolute paths for validation
	absMainDir, err := filepath.Abs(mainDir)
	if err != nil {
		return errors.ErrFileSystem("get absolute path for main directory", err)
	}

	absBareDir, err := filepath.Abs(bareDir)
	if err != nil {
		return errors.ErrFileSystem("get absolute path for bare directory", err)
	}

	// Ensure paths are clean and don't contain unnecessary separators
	if absMainDir != filepath.Clean(absMainDir) {
		return errors.ErrPathTraversal(mainDir).WithContext("type", "unclean_path")
	}

	if absBareDir != filepath.Clean(absBareDir) {
		return errors.ErrPathTraversal(bareDir).WithContext("type", "unclean_path")
	}

	return nil
}

// GitError represents an error from a git command execution.
type GitError struct {
	Command  string
	Args     []string
	Stderr   string
	ExitCode int
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %s failed (exit %d): %s", strings.Join(e.Args, " "), e.ExitCode, e.Stderr)
}

// ExecuteGit runs a git command with the given arguments and returns stdout.
// If the command fails, it returns a GitError with stderr and exit code.
func ExecuteGit(args ...string) (string, error) {
	log := logger.WithComponent("git_executor")
	start := time.Now()

	log.GitCommand("git", args)
	cmd := exec.Command("git", args...)

	stdout, err := cmd.Output()
	duration := time.Since(start)

	if err != nil {
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}

		gitErr := &GitError{
			Command:  "git",
			Args:     args,
			Stderr:   stderr,
			ExitCode: cmd.ProcessState.ExitCode(),
		}

		log.GitResult("git", false, stderr, "duration", duration)
		return "", gitErr
	}

	output := strings.TrimSpace(string(stdout))
	log.GitResult("git", true, output, "duration", duration)
	return output, nil
}

// ExecuteGitWithContext runs a git command with context support for cancellation.
// If the command fails, it returns a GitError with stderr and exit code.
func ExecuteGitWithContext(ctx context.Context, args ...string) (string, error) {
	log := logger.WithComponent("git_executor")
	start := time.Now()

	log.GitCommand("git", args, "with_context", true)
	cmd := exec.CommandContext(ctx, "git", args...)

	stdout, err := cmd.Output()
	duration := time.Since(start)

	if err != nil {
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}

		gitErr := &GitError{
			Command:  "git",
			Args:     args,
			Stderr:   stderr,
			ExitCode: cmd.ProcessState.ExitCode(),
		}

		log.GitResult("git", false, stderr, "duration", duration, "with_context", true)
		return "", gitErr
	}

	output := strings.TrimSpace(string(stdout))
	log.GitResult("git", true, output, "duration", duration, "with_context", true)
	return output, nil
}

// CloneBare runs git clone --bare for the given repository URL.
func CloneBare(repoURL, targetDir string) error {
	return CloneBareWithExecutor(DefaultExecutor, repoURL, targetDir)
}

// CloneBareWithExecutor runs git clone --bare using the specified executor.
func CloneBareWithExecutor(executor GitExecutor, repoURL, targetDir string) error {
	log := logger.WithComponent("git_clone")
	start := time.Now()

	log.DebugOperation("cloning bare repository", "repo_url", repoURL, "target_dir", targetDir)

	// Use configured retry mechanism for clone operation
	err := retry.WithConfiguredRetry(context.Background(), func() error {
		_, err := executor.Execute("clone", "--bare", repoURL, targetDir)
		if err != nil {
			// Classify error for retry logic
			if isNetworkError(err) {
				return errors.ErrNetworkTimeout("clone", err)
			}
			if isAuthError(err) {
				return errors.ErrAuthenticationFailed("clone", err)
			}
			// Default to git operation error (retryable)
			return errors.ErrGitOperation("clone", err)
		}
		return nil
	})
	if err != nil {
		log.ErrorOperation("clone bare failed", err, "repo_url", repoURL, "target_dir", targetDir, "duration", time.Since(start))
		return err
	}

	log.DebugOperation("clone bare completed", "repo_url", repoURL, "target_dir", targetDir, "duration", time.Since(start))
	return nil
}

// CreateGitFile writes a .git file with gitdir pointing to the bare repository.
func CreateGitFile(mainDir, bareDir string) error {
	log := logger.WithComponent("git_file")
	log.DebugOperation("creating .git file", "main_dir", mainDir, "bare_dir", bareDir)

	// Validate input paths for security
	if err := validatePaths(mainDir, bareDir); err != nil {
		log.ErrorOperation("path validation failed", err, "main_dir", mainDir, "bare_dir", bareDir)
		return fmt.Errorf("invalid paths: %w", err)
	}

	gitFilePath := filepath.Join(mainDir, ".git")

	// Make bareDir relative to mainDir if possible, otherwise use absolute path
	relPath, err := filepath.Rel(mainDir, bareDir)
	if err != nil {
		log.Debug("using absolute path for bare directory", "bare_dir", bareDir, "error", err)
		relPath = bareDir
	}

	// Validate that the relative path doesn't try to escape multiple directory levels
	// Allow single level traversal (../something) but reject deep traversal (../../something)
	if strings.HasPrefix(relPath, "../../") || strings.Contains(relPath, "/../..") {
		err := fmt.Errorf("relative path contains directory traversal: %s", relPath)
		log.ErrorOperation("path traversal validation failed", err, "rel_path", relPath)
		return err
	}

	content := fmt.Sprintf("gitdir: %s\n", relPath)
	log.Debug("writing .git file", "path", gitFilePath, "content", content)

	if err := os.WriteFile(gitFilePath, []byte(content), 0o600); err != nil {
		log.ErrorOperation("writing .git file failed", err, "path", gitFilePath)
		return err
	}

	log.DebugOperation(".git file created successfully", "path", gitFilePath, "gitdir", relPath)
	return nil
}

// ConfigureRemoteTracking sets up fetch refspec and fetches all remote branches.
func ConfigureRemoteTracking() error {
	return ConfigureRemoteTrackingWithExecutor(DefaultExecutor, "origin")
}

// ConfigureRemoteTrackingWithExecutor sets up fetch refspec using the specified executor.
func ConfigureRemoteTrackingWithExecutor(executor GitExecutor, remoteName string) error {
	log := logger.WithComponent("remote_tracking")
	start := time.Now()

	log.DebugOperation("configuring remote tracking", "remote", remoteName)

	// Set up fetch refspec to get all remote branches
	fetchRefspec := fmt.Sprintf("remote.%s.fetch", remoteName)
	refspecValue := fmt.Sprintf("+refs/heads/*:refs/remotes/%s/*", remoteName)

	log.Debug("setting fetch refspec", "refspec", fetchRefspec, "value", refspecValue)
	_, err := executor.Execute("config", fetchRefspec, refspecValue)
	if err != nil {
		log.ErrorOperation("config fetch refspec failed", err, "remote", remoteName, "refspec", refspecValue)
		return err
	}

	// Fetch all remote branches with configured retry
	log.Debug("fetching all remote branches", "remote", remoteName)
	err = retry.WithConfiguredRetry(context.Background(), func() error {
		_, err := executor.Execute("fetch")
		if err != nil {
			// Classify error for retry logic
			if isNetworkError(err) {
				return errors.ErrNetworkTimeout("fetch", err)
			}
			if isAuthError(err) {
				return errors.ErrAuthenticationFailed("fetch", err)
			}
			// Default to git operation error (retryable)
			return errors.ErrGitOperation("fetch", err)
		}
		return nil
	})
	if err != nil {
		log.ErrorOperation("fetch remote branches failed", err, "remote", remoteName, "duration", time.Since(start))
		return err
	}

	log.DebugOperation("remote tracking configured successfully", "remote", remoteName, "duration", time.Since(start))
	return nil
}

// SetupUpstreamBranches configures branch.*.remote for existing local branches.
func SetupUpstreamBranches() error {
	return SetupUpstreamBranchesWithExecutor(DefaultExecutor, "origin")
}

// SetupUpstreamBranchesWithExecutor configures upstream tracking using the specified executor.
func SetupUpstreamBranchesWithExecutor(executor GitExecutor, remoteName string) error {
	output, err := executor.Execute("for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return err
	}

	branches := strings.Split(strings.TrimSpace(output), "\n")
	for _, branch := range branches {
		if branch == "" {
			continue
		}

		// Set upstream tracking for each branch
		upstreamBranch := fmt.Sprintf("%s/%s", remoteName, branch)
		_, err := executor.Execute("branch", "--set-upstream-to="+upstreamBranch, branch)
		if err != nil {
			// Continue if this branch doesn't exist on remote
			continue
		}
	}

	return nil
}

// InitBare runs git init --bare in the target directory.
func InitBare(targetDir string) error {
	_, err := ExecuteGit("init", "--bare", targetDir)
	return err
}

// IsTraditionalRepo checks if the current directory contains a traditional Git repository.
// Returns true if there's a .git directory (not file), false otherwise.
func IsTraditionalRepo(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsGroveRepo checks if the current directory contains a Grove-structured repository.
// Returns true if there's a .git file pointing to a .bare directory.
func IsGroveRepo(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	bareDir := filepath.Join(dir, ".bare")

	// Check if .git is a file (not directory)
	gitInfo, err := os.Stat(gitPath)
	if err != nil || gitInfo.IsDir() {
		return false
	}

	// Check if .bare directory exists
	bareInfo, err := os.Stat(bareDir)
	if err != nil || !bareInfo.IsDir() {
		return false
	}

	return true
}

// ConvertToGroveStructure converts a traditional Git repository to Grove's worktree structure.
// This moves the .git directory to .bare and creates a .git file pointing to it.
func ConvertToGroveStructure(dir string) error {
	return ConvertToGroveStructureWithExecutor(DefaultExecutor, dir)
}

// ConvertToGroveStructureWithExecutor converts using the specified executor.
func ConvertToGroveStructureWithExecutor(executor GitExecutor, dir string) error {
	log := logger.WithComponent("conversion")
	start := time.Now()

	log.DebugOperation("starting Grove structure conversion", "directory", dir)

	log.Debug("validating conversion preconditions", "directory", dir)
	if err := validateConversionPreconditions(dir); err != nil {
		log.Debug("conversion precondition validation failed", "error", err, "directory", dir)
		return err
	}

	log.Debug("checking repository cleanliness", "directory", dir)
	if err := checkRepositoryClean(executor, dir); err != nil {
		log.Debug("repository cleanliness check failed", "error", err, "directory", dir)
		return err
	}

	log.Debug("performing conversion", "directory", dir)
	if err := performConversion(dir); err != nil {
		log.Debug("conversion failed", "error", err, "directory", dir, "duration", time.Since(start))
		return err
	}

	log.DebugOperation("Grove structure conversion completed successfully", "directory", dir, "duration", time.Since(start))
	return nil
}

// validateConversionPreconditions checks that the directory is ready for conversion.
func validateConversionPreconditions(dir string) error {
	if !IsTraditionalRepo(dir) {
		return fmt.Errorf("directory does not contain a traditional Git repository")
	}

	bareDir := filepath.Join(dir, ".bare")
	if _, err := os.Stat(bareDir); err == nil {
		return fmt.Errorf(".bare directory already exists in %s", dir)
	}

	return nil
}

// SafetyIssue represents a repository safety issue that prevents conversion.
type SafetyIssue struct {
	Type        string
	Description string
	Solution    string
}

// checkRepositoryClean verifies that the repository has no uncommitted changes.
func checkRepositoryClean(executor GitExecutor, dir string) error {
	issues, err := checkRepositorySafetyForConversion(executor, dir)
	if err != nil {
		return err
	}

	if len(issues) > 0 {
		return formatSafetyIssuesError(issues)
	}

	return nil
}

// checkRepositorySafetyForConversion performs comprehensive safety checks before conversion.
func checkRepositorySafetyForConversion(executor GitExecutor, dir string) ([]SafetyIssue, error) {
	log := logger.WithComponent("safety_checks")
	start := time.Now()

	log.DebugOperation("starting repository safety checks", "directory", dir)

	originalDir, err := os.Getwd()
	if err != nil {
		log.ErrorOperation("failed to get current directory", err, "directory", dir)
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		log.ErrorOperation("failed to change directory", err, "directory", dir)
		return nil, fmt.Errorf("failed to change to directory %s: %w", dir, err)
	}

	defer func() { _ = os.Chdir(originalDir) }()

	// Define all safety check functions
	safetyChecks := []func(GitExecutor) ([]SafetyIssue, error){
		checkGitStatus,
		checkStashedChanges,
		checkUntrackedFiles,
		checkExistingWorktrees,
		checkUnpushedCommits,
		checkLocalOnlyBranches,
	}

	var allIssues []SafetyIssue

	// Run all safety checks
	for i, check := range safetyChecks {
		log.Debug("running safety check", "check_index", i+1, "total_checks", len(safetyChecks))
		issues, err := check(executor)
		if err != nil {
			log.ErrorOperation("safety check failed", err, "check_index", i+1, "directory", dir)
			return nil, err
		}
		allIssues = append(allIssues, issues...)
		log.Debug("safety check completed", "check_index", i+1, "issues_found", len(issues))
	}

	log.DebugOperation("repository safety checks completed", "directory", dir, "total_issues", len(allIssues), "duration", time.Since(start))
	return allIssues, nil
}

// GitChangeCounts represents the counts of different types of git changes.
type GitChangeCounts struct {
	Modified  int
	Added     int
	Deleted   int
	Renamed   int
	Untracked int
}

// HasChanges returns true if there are any uncommitted changes.
func (c GitChangeCounts) HasChanges() bool {
	return c.Modified+c.Added+c.Deleted+c.Renamed > 0
}

// BuildDescription creates a human-readable description of the changes.
func (c GitChangeCounts) BuildDescription() string {
	if !c.HasChanges() {
		return ""
	}

	var parts []string
	if c.Modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", c.Modified))
	}
	if c.Added > 0 {
		parts = append(parts, fmt.Sprintf("%d added", c.Added))
	}
	if c.Deleted > 0 {
		parts = append(parts, fmt.Sprintf("%d deleted", c.Deleted))
	}
	if c.Renamed > 0 {
		parts = append(parts, fmt.Sprintf("%d renamed", c.Renamed))
	}

	return fmt.Sprintf("Uncommitted changes (%s)", strings.Join(parts, ", "))
}

// BuildSolution creates solution text for the changes.
func (c GitChangeCounts) BuildSolution() string {
	if !c.HasChanges() {
		return ""
	}
	return "git add <files> && git commit"
}

// ToSafetyIssue converts the change counts to a SafetyIssue.
func (c GitChangeCounts) ToSafetyIssue() SafetyIssue {
	return SafetyIssue{
		Type:        "uncommitted_changes",
		Description: c.BuildDescription(),
		Solution:    c.BuildSolution(),
	}
}

// parseGitStatusLine parses a single git status line and returns the staged and unstaged status.
// Git status uses a two-character format: first character is staged, second is unstaged.
// Examples: " M" = unstaged modified, "A " = staged added, "MM" = modified in both staged and unstaged.
func parseGitStatusLine(line string) (staged, unstaged rune) {
	if len(line) < 2 {
		return ' ', ' '
	}
	return rune(line[0]), rune(line[1])
}

// countGitChanges parses git status output and counts different types of changes.
// It processes each line from `git status --porcelain=v1` and categorizes changes by type.
// Returns a GitChangeCounts struct with counts for modified, added, deleted, renamed, and untracked files.
func countGitChanges(lines []string) GitChangeCounts {
	var counts GitChangeCounts

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		staged, unstaged := parseGitStatusLine(line)

		// Count staged changes
		switch staged {
		case 'M':
			counts.Modified++
		case 'A':
			counts.Added++
		case 'D':
			counts.Deleted++
		case 'R', 'C':
			counts.Renamed++
		}

		// Count unstaged changes
		switch unstaged {
		case 'M':
			counts.Modified++
		case 'D':
			counts.Deleted++
		case '?':
			counts.Untracked++
		}
	}

	return counts
}

// checkOngoingGitOperations checks for ongoing git operations like rebase or merge.
// It examines the verbose git status output to detect operations in progress.
// Returns safety issues for rebase, merge, cherry-pick, and bisect operations.
// This function does not return errors - it continues gracefully if git commands fail.
func checkOngoingGitOperations(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	statusOutput, err := executor.Execute("status")
	if err != nil {
		return issues, nil // Continue without detailed status if this fails
	}

	if strings.Contains(statusOutput, "rebase in progress") {
		issues = append(issues, SafetyIssue{
			Type:        "ongoing_rebase",
			Description: "Git rebase in progress",
			Solution:    "Complete with 'git rebase --continue' or abort with 'git rebase --abort'",
		})
	}

	if strings.Contains(statusOutput, "merge in progress") {
		issues = append(issues, SafetyIssue{
			Type:        "ongoing_merge",
			Description: "Git merge in progress",
			Solution:    "Complete with 'git merge --continue' or abort with 'git merge --abort'",
		})
	}

	if strings.Contains(statusOutput, "cherry-pick in progress") {
		issues = append(issues, SafetyIssue{
			Type:        "ongoing_cherry_pick",
			Description: "Git cherry-pick in progress",
			Solution:    "Complete with 'git cherry-pick --continue' or abort with 'git cherry-pick --abort'",
		})
	}

	if strings.Contains(statusOutput, "bisect in progress") {
		issues = append(issues, SafetyIssue{
			Type:        "ongoing_bisect",
			Description: "Git bisect in progress",
			Solution:    "Complete bisect or abort with 'git bisect reset'",
		})
	}

	return issues, nil
}

// checkGitStatus checks for uncommitted changes and ongoing git operations.
func checkGitStatus(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	// Get git status output
	output, err := executor.Execute("status", "--porcelain=v1")
	if err != nil {
		return nil, fmt.Errorf("failed to check repository status: %w", err)
	}

	// Parse output and count changes
	lines := strings.Split(strings.TrimSpace(output), "\n")
	hasUncommittedChanges := len(lines) != 1 || lines[0] != ""

	if hasUncommittedChanges {
		counts := countGitChanges(lines)
		if counts.HasChanges() {
			issues = append(issues, counts.ToSafetyIssue())
		}
	}

	// Check for ongoing git operations
	ongoingIssues, err := checkOngoingGitOperations(executor)
	if err != nil {
		return issues, nil // Continue if we can't check ongoing operations
	}
	issues = append(issues, ongoingIssues...)

	return issues, nil
}

// checkStashedChanges checks for stashed changes.
func checkStashedChanges(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.Execute("stash", "list")
	if err != nil {
		return issues, nil // If stash command fails, assume no stashes
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return issues, nil // No stashes
	}

	stashCount := len(lines)
	issues = append(issues, SafetyIssue{
		Type:        "stashed_changes",
		Description: fmt.Sprintf("%d stashed change(s)", stashCount),
		Solution:    "Apply with 'git stash pop' or remove with 'git stash drop'",
	})

	return issues, nil
}

// checkUntrackedFiles checks for untracked files.
func checkUntrackedFiles(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.Execute("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return issues, nil // If command fails, assume no untracked files
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return issues, nil // No untracked files
	}

	fileCount := len(lines)
	issues = append(issues, SafetyIssue{
		Type:        "untracked_files",
		Description: fmt.Sprintf("%d untracked file(s)", fileCount),
		Solution:    "Add to git with 'git add <files>' or add to .gitignore",
	})

	return issues, nil
}

// checkExistingWorktrees checks for existing worktrees.
func checkExistingWorktrees(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.Execute("worktree", "list")
	if err != nil {
		return issues, nil // If worktree command fails, assume no worktrees
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	// Filter out the main worktree (current directory)
	var additionalWorktrees []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(line, " (bare)") {
			// Check if this is not the main worktree
			if !strings.HasSuffix(strings.Fields(line)[0], ".") {
				additionalWorktrees = append(additionalWorktrees, line)
			}
		}
	}

	if len(additionalWorktrees) > 1 { // More than just the main worktree
		issues = append(issues, SafetyIssue{
			Type:        "existing_worktrees",
			Description: fmt.Sprintf("%d existing worktree(s)", len(additionalWorktrees)-1),
			Solution:    "Remove with 'git worktree remove <path>' or 'git worktree prune'",
		})
	}

	return issues, nil
}

// checkUnpushedCommits checks for unpushed commits on tracked branches.
func checkUnpushedCommits(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	// Get all local branches with their upstream tracking info
	output, err := executor.Execute("for-each-ref", "--format=%(refname:short) %(upstream:short) %(upstream:track)", "refs/heads")
	if err != nil {
		return issues, nil // If command fails, continue without this check
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue // No upstream
		}

		branch := fields[0]
		upstream := fields[1]

		if len(fields) >= 3 && strings.Contains(fields[2], "ahead") {
			// Extract number of commits ahead
			trackInfo := strings.Join(fields[2:], " ")
			issues = append(issues, SafetyIssue{
				Type:        "unpushed_commits",
				Description: fmt.Sprintf("Branch '%s' has unpushed commits (%s)", branch, trackInfo),
				Solution:    fmt.Sprintf("Push with 'git push origin %s'", branch),
			})
		} else if upstream != "" {
			// Double-check with git log if tracking info isn't clear
			logOutput, logErr := executor.Execute("rev-list", "--count", upstream+".."+branch)
			if logErr == nil {
				commitCount := strings.TrimSpace(logOutput)
				if commitCount != "0" {
					issues = append(issues, SafetyIssue{
						Type:        "unpushed_commits",
						Description: fmt.Sprintf("Branch '%s' has %s unpushed commit(s)", branch, commitCount),
						Solution:    fmt.Sprintf("Push with 'git push origin %s'", branch),
					})
				}
			}
		}
	}

	return issues, nil
}

// checkLocalOnlyBranches checks for branches that exist only locally.
func checkLocalOnlyBranches(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	// Get all local branches with their upstream info
	output, err := executor.Execute("for-each-ref", "--format=%(refname:short) %(upstream)", "refs/heads")
	if err != nil {
		return issues, nil // If command fails, continue without this check
	}

	var localOnlyBranches []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 1 {
			// No upstream configured
			localOnlyBranches = append(localOnlyBranches, fields[0])
		}
	}

	if len(localOnlyBranches) > 0 {
		branchList := strings.Join(localOnlyBranches, ", ")
		issues = append(issues, SafetyIssue{
			Type:        "local_only_branches",
			Description: fmt.Sprintf("Local-only branch(es): %s", branchList),
			Solution:    "Push with 'git push -u origin <branch>' or delete with 'git branch -d <branch>'",
		})
	}

	return issues, nil
}

// formatSafetyIssuesError creates a comprehensive error message from safety issues.
func formatSafetyIssuesError(issues []SafetyIssue) error {
	var msg strings.Builder
	msg.WriteString("Repository is not ready for conversion:\n")

	for _, issue := range issues {
		msg.WriteString(fmt.Sprintf("  ✗ %s (%s)\n", issue.Description, issue.Solution))
	}

	msg.WriteString("\nPlease resolve these issues before converting to ensure no work is lost.")
	return fmt.Errorf("%s", msg.String())
}

// performConversion executes the actual conversion with rollback on failure.
func performConversion(dir string) error {
	gitDir := filepath.Join(dir, ".git")
	bareDir := filepath.Join(dir, ".bare")
	backupDir := filepath.Join(dir, ".git.backup")

	if err := os.Rename(gitDir, backupDir); err != nil {
		return fmt.Errorf("failed to create backup of .git directory: %w", err)
	}

	if err := os.Rename(backupDir, bareDir); err != nil {
		_ = os.Rename(backupDir, gitDir)
		return fmt.Errorf("failed to move .git to .bare: %w", err)
	}

	if err := CreateGitFile(dir, bareDir); err != nil {
		_ = os.Rename(bareDir, gitDir)
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	// Validate the conversion was successful
	if err := ValidateGroveStructure(dir); err != nil {
		_ = os.Remove(filepath.Join(dir, ".git"))
		_ = os.Rename(bareDir, gitDir)
		return fmt.Errorf("conversion validation failed: %w", err)
	}

	return nil
}

// createProperWorktreeStructure creates a proper worktree structure after conversion.
func createProperWorktreeStructure(executor GitExecutor, dir string) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change to directory %s: %w", dir, err)
	}

	defer func() { _ = os.Chdir(originalDir) }()

	// Detect the default branch for the repository
	defaultBranch, err := DetectDefaultBranch(executor, "origin")
	if err != nil {
		return fmt.Errorf("failed to detect default branch: %w", err)
	}

	// Use the detected default branch as the worktree branch
	currentBranch := defaultBranch

	// Create worktree directory path using filesystem-safe naming
	dirName := BranchToDirectoryName(currentBranch)
	worktreePath := filepath.Join(dir, dirName)

	// Check if worktree directory already exists
	if _, err := os.Stat(worktreePath); err == nil {
		// Directory exists, skip creation
		return nil
	}

	// Get list of files in the current directory (excluding .bare and .git)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	var workingFiles []string
	for _, entry := range entries {
		name := entry.Name()
		if name != ".bare" && name != ".git" && name != ".git.backup" {
			workingFiles = append(workingFiles, name)
		}
	}

	// If there are working files, we need to create a proper worktree structure
	if len(workingFiles) > 0 {
		// Create temporary directory to hold files during conversion
		tempDir := filepath.Join(dir, ".grove-temp-files")
		if err := os.MkdirAll(tempDir, 0o755); err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}

		// Move all working files to temporary directory to preserve them
		for _, file := range workingFiles {
			srcPath := filepath.Join(dir, file)
			dstPath := filepath.Join(tempDir, file)

			// Create parent directory if needed
			parentDir := filepath.Dir(dstPath)
			if err := os.MkdirAll(parentDir, 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}

			// Move the file/directory
			if err := os.Rename(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to move %s to temporary location: %w", file, err)
			}
		}

		// Configure the repository as bare to allow worktree creation
		_, err = executor.Execute("config", "--bool", "core.bare", "true")
		if err != nil {
			return fmt.Errorf("failed to set core.bare: %w", err)
		}

		// Create the worktree - this will populate it with the files from the branch
		_, err = CreateWorktreeFromExistingBranch(executor, currentBranch, dir)
		if err != nil {
			return fmt.Errorf("failed to create worktree for branch %s: %w", currentBranch, err)
		}

		// Move all files from temporary directory to worktree, preserving gitignored files
		err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if path == tempDir {
				return nil // Skip the root temp directory itself
			}

			// Calculate relative path from temp dir
			relPath, err := filepath.Rel(tempDir, path)
			if err != nil {
				return err
			}

			// Destination path in worktree
			dstPath := filepath.Join(worktreePath, relPath)

			if info.IsDir() {
				// If it's a directory, ensure it exists in worktree
				if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
					return fmt.Errorf("failed to create directory %s in worktree: %w", relPath, err)
				}
			} else {
				// If destination file already exists (from git worktree add), remove it first
				// This handles the case where the file is tracked by git
				if _, err := os.Stat(dstPath); err == nil {
					if err := os.Remove(dstPath); err != nil {
						return fmt.Errorf("failed to remove existing file %s: %w", dstPath, err)
					}
				}

				// Move the file to worktree
				if err := os.Rename(path, dstPath); err != nil {
					return fmt.Errorf("failed to move %s to worktree: %w", relPath, err)
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to move files to worktree: %w", err)
		}

		// Clean up temporary directory
		if err := os.RemoveAll(tempDir); err != nil {
			// Log warning but don't fail the conversion
			// The conversion succeeded, just cleanup failed
			return fmt.Errorf("failed to clean up temporary directory: %w", err)
		}
	}

	return nil
}

// ValidateGroveStructure verifies that a Grove repository structure is valid and functional.
func ValidateGroveStructure(dir string) error {
	return ValidateGroveStructureWithExecutor(DefaultExecutor, dir)
}

// ValidateGroveStructureWithExecutor validates using the specified executor.
func ValidateGroveStructureWithExecutor(executor GitExecutor, dir string) error {
	log := logger.WithComponent("validation")
	start := time.Now()

	log.DebugOperation("validating Grove structure", "directory", dir)

	// Check that .git file exists and is not a directory
	gitPath := filepath.Join(dir, ".git")
	log.Debug("checking .git file", "path", gitPath)
	gitInfo, err := os.Stat(gitPath)
	if err != nil {
		log.ErrorOperation(".git file validation failed", err, "path", gitPath)
		return fmt.Errorf(".git file does not exist: %w", err)
	}
	if gitInfo.IsDir() {
		err := fmt.Errorf(".git should be a file, not a directory")
		log.ErrorOperation(".git file type validation failed", err, "path", gitPath)
		return err
	}

	// Check that .bare directory exists
	bareDir := filepath.Join(dir, ".bare")
	log.Debug("checking .bare directory", "path", bareDir)
	bareInfo, err := os.Stat(bareDir)
	if err != nil {
		log.ErrorOperation(".bare directory validation failed", err, "path", bareDir)
		return fmt.Errorf(".bare directory does not exist: %w", err)
	}
	if !bareInfo.IsDir() {
		err := fmt.Errorf(".bare should be a directory")
		log.ErrorOperation(".bare directory type validation failed", err, "path", bareDir)
		return err
	}

	// Validate .git file content
	log.Debug("validating .git file content", "path", gitPath)
	gitContent, err := os.ReadFile(gitPath)
	if err != nil {
		log.ErrorOperation(".git file content read failed", err, "path", gitPath)
		return fmt.Errorf("failed to read .git file: %w", err)
	}

	expectedContent := fmt.Sprintf("gitdir: %s\n", ".bare")
	if string(gitContent) != expectedContent {
		err := fmt.Errorf(".git file content is invalid, expected 'gitdir: .bare\\n', got '%s'", string(gitContent))
		log.ErrorOperation(".git file content validation failed", err, "expected", expectedContent, "actual", string(gitContent))
		return err
	}

	// Test that git operations work
	originalDir, err := os.Getwd()
	if err != nil {
		log.ErrorOperation("failed to get current directory", err)
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		log.ErrorOperation("failed to change directory", err, "directory", dir)
		return fmt.Errorf("failed to change to directory %s: %w", dir, err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// Test basic git operation
	log.Debug("testing git operations in converted repository", "directory", dir)
	_, err = executor.Execute("status")
	if err != nil {
		log.ErrorOperation("git status test failed", err, "directory", dir)
		return fmt.Errorf("git status failed in converted repository: %w", err)
	}

	log.DebugOperation("Grove structure validation completed successfully", "directory", dir, "duration", time.Since(start))
	return nil
}

// CreateDefaultWorktree creates a worktree for the current branch after conversion.
func CreateDefaultWorktree(dir string) error {
	return CreateDefaultWorktreeWithExecutor(DefaultExecutor, dir)
}

// CreateDefaultWorktreeWithExecutor creates a worktree using the specified executor.
func CreateDefaultWorktreeWithExecutor(executor GitExecutor, dir string) error {
	return createProperWorktreeStructure(executor, dir)
}
