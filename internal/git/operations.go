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

type GitExecutor interface {
	Execute(args ...string) (string, error)
	ExecuteQuiet(args ...string) (string, error)
	ExecuteWithContext(ctx context.Context, args ...string) (string, error)
}

type DefaultGitExecutor struct{}

func (e *DefaultGitExecutor) Execute(args ...string) (string, error) {
	return ExecuteGit(args...)
}

// Use this for operations where failures are expected and should not be logged as errors.
func (e *DefaultGitExecutor) ExecuteQuiet(args ...string) (string, error) {
	return ExecuteGitQuiet(args...)
}

func (e *DefaultGitExecutor) ExecuteWithContext(ctx context.Context, args ...string) (string, error) {
	return ExecuteGitWithContext(ctx, args...)
}

var DefaultExecutor GitExecutor = &DefaultGitExecutor{}

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

func validatePaths(mainDir, bareDir string) error {
	if strings.Contains(mainDir, "..") || strings.Contains(bareDir, "..") {
		return errors.ErrPathTraversal("paths contain directory traversal sequences")
	}

	absMainDir, err := filepath.Abs(mainDir)
	if err != nil {
		return errors.ErrFileSystem("get absolute path for main directory", err)
	}

	absBareDir, err := filepath.Abs(bareDir)
	if err != nil {
		return errors.ErrFileSystem("get absolute path for bare directory", err)
	}

	if absMainDir != filepath.Clean(absMainDir) {
		return errors.ErrPathTraversal(mainDir).WithContext("type", "unclean_path")
	}

	if absBareDir != filepath.Clean(absBareDir) {
		return errors.ErrPathTraversal(bareDir).WithContext("type", "unclean_path")
	}

	return nil
}

type GitError struct {
	Command  string
	Args     []string
	Stderr   string
	ExitCode int
}

func (e *GitError) Error() string {
	return fmt.Sprintf("git %s failed (exit %d): %s", strings.Join(e.Args, " "), e.ExitCode, e.Stderr)
}

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

// This is useful for operations where failures are expected and should not be logged as errors.
// Successful operations are still logged at debug level.
func ExecuteGitQuiet(args ...string) (string, error) {
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

		// Note: We don't log failures for quiet execution.
		// The caller expects failures and will handle them appropriately.
		return "", gitErr
	}

	output := strings.TrimSpace(string(stdout))
	log.GitResult("git", true, output, "duration", duration)
	return output, nil
}

func CloneBare(repoURL, targetDir string) error {
	return CloneBareWithExecutor(DefaultExecutor, repoURL, targetDir)
}

func CloneBareWithExecutor(executor GitExecutor, repoURL, targetDir string) error {
	log := logger.WithComponent("git_clone")
	start := time.Now()

	log.DebugOperation("cloning bare repository", "repo_url", repoURL, "target_dir", targetDir)

	err := retry.WithConfiguredRetry(context.Background(), func() error {
		_, err := executor.Execute("clone", "--bare", repoURL, targetDir)
		if err != nil {
			if isNetworkError(err) {
				return errors.ErrNetworkTimeout("clone", err)
			}
			if isAuthError(err) {
				return errors.ErrAuthenticationFailed("clone", err)
			}
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

func CreateGitFile(mainDir, bareDir string) error {
	log := logger.WithComponent("git_file")
	log.DebugOperation("creating .git file", "main_dir", mainDir, "bare_dir", bareDir)

	if err := validatePaths(mainDir, bareDir); err != nil {
		log.ErrorOperation("path validation failed", err, "main_dir", mainDir, "bare_dir", bareDir)
		return fmt.Errorf("invalid paths: %w", err)
	}

	gitFilePath := filepath.Join(mainDir, ".git")

	relPath, err := filepath.Rel(mainDir, bareDir)
	if err != nil {
		log.Debug("using absolute path for bare directory", "bare_dir", bareDir, "error", err)
		relPath = bareDir
	}

	// Validate that the relative path doesn't try to escape multiple directory levels.
	// Allow single level traversal (../something) but reject deep traversal (../../something).
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

func ConfigureRemoteTracking() error {
	return ConfigureRemoteTrackingWithExecutor(DefaultExecutor, "origin")
}

func ConfigureRemoteTrackingWithExecutor(executor GitExecutor, remoteName string) error {
	log := logger.WithComponent("remote_tracking")
	start := time.Now()

	log.DebugOperation("configuring remote tracking", "remote", remoteName)

	fetchRefspec := fmt.Sprintf("remote.%s.fetch", remoteName)
	refspecValue := fmt.Sprintf("+refs/heads/*:refs/remotes/%s/*", remoteName)

	log.Debug("setting fetch refspec", "refspec", fetchRefspec, "value", refspecValue)
	_, err := executor.Execute("config", fetchRefspec, refspecValue)
	if err != nil {
		log.ErrorOperation("config fetch refspec failed", err, "remote", remoteName, "refspec", refspecValue)
		return err
	}

	log.Debug("fetching all remote branches", "remote", remoteName)
	err = retry.WithConfiguredRetry(context.Background(), func() error {
		_, err := executor.Execute("fetch")
		if err != nil {
			if isNetworkError(err) {
				return errors.ErrNetworkTimeout("fetch", err)
			}
			if isAuthError(err) {
				return errors.ErrAuthenticationFailed("fetch", err)
			}
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

func SetupUpstreamBranches() error {
	return SetupUpstreamBranchesWithExecutor(DefaultExecutor, "origin")
}

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

		upstreamBranch := fmt.Sprintf("%s/%s", remoteName, branch)
		_, err := executor.Execute("branch", "--set-upstream-to="+upstreamBranch, branch)
		if err != nil {
			continue
		}
	}

	return nil
}

func InitBare(targetDir string) error {
	_, err := ExecuteGit("init", "--bare", targetDir)
	return err
}

func IsTraditionalRepo(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	return info.IsDir()
}

func IsGroveRepo(dir string) bool {
	gitPath := filepath.Join(dir, ".git")
	bareDir := filepath.Join(dir, ".bare")

	gitInfo, err := os.Stat(gitPath)
	if err != nil || gitInfo.IsDir() {
		return false
	}

	bareInfo, err := os.Stat(bareDir)
	if err != nil || !bareInfo.IsDir() {
		return false
	}

	return true
}

func ConvertToGroveStructure(dir string) error {
	return ConvertToGroveStructureWithExecutor(DefaultExecutor, dir)
}

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

type SafetyIssue struct {
	Type        string
	Description string
	Solution    string
}

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

	safetyChecks := []func(GitExecutor) ([]SafetyIssue, error){
		checkGitStatus,
		checkStashedChanges,
		checkUntrackedFiles,
		checkExistingWorktrees,
		checkUnpushedCommits,
		checkLocalOnlyBranches,
	}

	var allIssues []SafetyIssue

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

type GitChangeCounts struct {
	Modified  int
	Added     int
	Deleted   int
	Renamed   int
	Untracked int
}

func (c GitChangeCounts) HasChanges() bool {
	return c.Modified+c.Added+c.Deleted+c.Renamed > 0
}

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

func (c GitChangeCounts) BuildSolution() string {
	if !c.HasChanges() {
		return ""
	}
	return "git add <files> && git commit"
}

func (c GitChangeCounts) ToSafetyIssue() SafetyIssue {
	return SafetyIssue{
		Type:        "uncommitted_changes",
		Description: c.BuildDescription(),
		Solution:    c.BuildSolution(),
	}
}

func parseGitStatusLine(line string) (staged, unstaged rune) {
	if len(line) < 2 {
		return ' ', ' '
	}
	return rune(line[0]), rune(line[1])
}

func countGitChanges(lines []string) GitChangeCounts {
	var counts GitChangeCounts

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}

		staged, unstaged := parseGitStatusLine(line)

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

// It examines the verbose git status output to detect operations in progress.
// This function does not return errors - it continues gracefully if git commands fail.
func checkOngoingGitOperations(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	statusOutput, err := executor.Execute("status")
	if err != nil {
		log := logger.WithComponent("git_operations")
		log.Debug("git status failed during operation check",
			"error", err,
			"reason", "continuing without detailed status - git repository might be corrupted or inaccessible")
		return issues, nil
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

func checkGitStatus(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.Execute("status", "--porcelain=v1")
	if err != nil {
		return nil, fmt.Errorf("failed to check repository status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	hasUncommittedChanges := len(lines) != 1 || lines[0] != ""

	if hasUncommittedChanges {
		counts := countGitChanges(lines)
		if counts.HasChanges() {
			issues = append(issues, counts.ToSafetyIssue())
		}
	}

	ongoingIssues, err := checkOngoingGitOperations(executor)
	if err != nil {
		return issues, nil
	}
	issues = append(issues, ongoingIssues...)

	return issues, nil
}

func checkStashedChanges(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.Execute("stash", "list")
	if err != nil {
		return issues, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return issues, nil
	}

	stashCount := len(lines)
	issues = append(issues, SafetyIssue{
		Type:        "stashed_changes",
		Description: fmt.Sprintf("%d stashed change(s)", stashCount),
		Solution:    "Apply with 'git stash pop' or remove with 'git stash drop'",
	})

	return issues, nil
}

func checkUntrackedFiles(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.Execute("ls-files", "--others", "--exclude-standard")
	if err != nil {
		return issues, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 1 && lines[0] == "" {
		return issues, nil
	}

	fileCount := len(lines)
	issues = append(issues, SafetyIssue{
		Type:        "untracked_files",
		Description: fmt.Sprintf("%d untracked file(s)", fileCount),
		Solution:    "Add to git with 'git add <files>' or add to .gitignore",
	})

	return issues, nil
}

func checkExistingWorktrees(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.ExecuteQuiet("worktree", "list")
	if err != nil {
		log := logger.WithComponent("git_operations")
		log.Debug("git worktree list failed during safety check",
			"error", err,
			"reason", "assuming no worktrees exist - git version might not support worktrees or repository is corrupted")
		return issues, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	var additionalWorktrees []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" && !strings.Contains(line, " (bare)") {
			if !strings.HasSuffix(strings.Fields(line)[0], ".") {
				additionalWorktrees = append(additionalWorktrees, line)
			}
		}
	}

	if len(additionalWorktrees) > 1 {
		issues = append(issues, SafetyIssue{
			Type:        "existing_worktrees",
			Description: fmt.Sprintf("%d existing worktree(s)", len(additionalWorktrees)-1),
			Solution:    "Remove with 'git worktree remove <path>' or 'git worktree prune'",
		})
	}

	return issues, nil
}

func checkUnpushedCommits(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.Execute("for-each-ref", "--format=%(refname:short) %(upstream:short) %(upstream:track)", "refs/heads")
	if err != nil {
		return issues, nil
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		branch := fields[0]
		upstream := fields[1]

		if len(fields) >= 3 && strings.Contains(fields[2], "ahead") {
			trackInfo := strings.Join(fields[2:], " ")
			issues = append(issues, SafetyIssue{
				Type:        "unpushed_commits",
				Description: fmt.Sprintf("Branch '%s' has unpushed commits (%s)", branch, trackInfo),
				Solution:    fmt.Sprintf("Push with 'git push origin %s'", branch),
			})
		} else if upstream != "" {
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

func checkLocalOnlyBranches(executor GitExecutor) ([]SafetyIssue, error) {
	var issues []SafetyIssue

	output, err := executor.Execute("for-each-ref", "--format=%(refname:short) %(upstream)", "refs/heads")
	if err != nil {
		return issues, nil
	}

	var localOnlyBranches []string
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 1 {
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

func formatSafetyIssuesError(issues []SafetyIssue) error {
	var msg strings.Builder
	msg.WriteString("Repository is not ready for conversion:\n")

	for _, issue := range issues {
		msg.WriteString(fmt.Sprintf("  ✗ %s (%s)\n", issue.Description, issue.Solution))
	}

	msg.WriteString("\nPlease resolve these issues before converting to ensure no work is lost.")
	return fmt.Errorf("%s", msg.String())
}

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

	if err := ValidateGroveStructure(dir); err != nil {
		_ = os.Remove(filepath.Join(dir, ".git"))
		_ = os.Rename(bareDir, gitDir)
		return fmt.Errorf("conversion validation failed: %w", err)
	}

	return nil
}

func createProperWorktreeStructure(executor GitExecutor, dir string) error {
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change to directory %s: %w", dir, err)
	}

	defer func() { _ = os.Chdir(originalDir) }()

	defaultBranch, err := DetectDefaultBranch(executor, "origin")
	if err != nil {
		return fmt.Errorf("failed to detect default branch: %w", err)
	}

	currentBranch := defaultBranch
	dirName := BranchToDirectoryName(currentBranch)
	worktreePath := filepath.Join(dir, dirName)

	if _, err := os.Stat(worktreePath); err == nil {
		return nil
	}

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

	if len(workingFiles) > 0 {
		tempDir := filepath.Join(dir, ".grove-temp-files")
		if err := os.MkdirAll(tempDir, 0o755); err != nil {
			return fmt.Errorf("failed to create temporary directory: %w", err)
		}

		for _, file := range workingFiles {
			srcPath := filepath.Join(dir, file)
			dstPath := filepath.Join(tempDir, file)

			parentDir := filepath.Dir(dstPath)
			if err := os.MkdirAll(parentDir, 0o755); err != nil {
				return fmt.Errorf("failed to create parent directory %s: %w", parentDir, err)
			}

			if err := os.Rename(srcPath, dstPath); err != nil {
				return fmt.Errorf("failed to move %s to temporary location: %w", file, err)
			}
		}

		_, err = executor.Execute("config", "--bool", "core.bare", "true")
		if err != nil {
			return fmt.Errorf("failed to set core.bare: %w", err)
		}

		_, err = CreateWorktreeFromExistingBranch(executor, currentBranch, dir)
		if err != nil {
			return fmt.Errorf("failed to create worktree for branch %s: %w", currentBranch, err)
		}

		err = filepath.Walk(tempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if path == tempDir {
				return nil
			}

			relPath, err := filepath.Rel(tempDir, path)
			if err != nil {
				return err
			}

			dstPath := filepath.Join(worktreePath, relPath)

			if info.IsDir() {
				if err := os.MkdirAll(dstPath, info.Mode()); err != nil {
					return fmt.Errorf("failed to create directory %s in worktree: %w", relPath, err)
				}
			} else {
				if _, err := os.Stat(dstPath); err == nil {
					if err := os.Remove(dstPath); err != nil {
						return fmt.Errorf("failed to remove existing file %s: %w", dstPath, err)
					}
				}

				if err := os.Rename(path, dstPath); err != nil {
					return fmt.Errorf("failed to move %s to worktree: %w", relPath, err)
				}
			}

			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to move files to worktree: %w", err)
		}

		if err := os.RemoveAll(tempDir); err != nil {
			return fmt.Errorf("failed to clean up temporary directory: %w", err)
		}
	}

	return nil
}

func ValidateGroveStructure(dir string) error {
	return ValidateGroveStructureWithExecutor(DefaultExecutor, dir)
}

func ValidateGroveStructureWithExecutor(executor GitExecutor, dir string) error {
	log := logger.WithComponent("validation")
	start := time.Now()

	log.DebugOperation("validating Grove structure", "directory", dir)

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
	log.Debug("testing git operations in converted repository", "directory", dir)
	_, err = executor.Execute("status")
	if err != nil {
		log.ErrorOperation("git status test failed", err, "directory", dir)
		return fmt.Errorf("git status failed in converted repository: %w", err)
	}

	log.DebugOperation("Grove structure validation completed successfully", "directory", dir, "duration", time.Since(start))
	return nil
}

func CreateDefaultWorktree(dir string) error {
	return CreateDefaultWorktreeWithExecutor(DefaultExecutor, dir)
}

func CreateDefaultWorktreeWithExecutor(executor GitExecutor, dir string) error {
	return createProperWorktreeStructure(executor, dir)
}

// It performs basic validation to detect common worktree issues without being overly strict.
// This function is designed to catch the specific case where .git files point to invalid.
// locations which causes "fatal: this operation must be run in a work tree" errors.
func IsValidWorktreeDirectory(worktreePath string) error {
	if worktreePath == "" {
		return fmt.Errorf("worktree path cannot be empty")
	}

	if stat, err := os.Stat(worktreePath); err != nil {
		return fmt.Errorf("worktree directory %s does not exist: %w", worktreePath, err)
	} else if !stat.IsDir() {
		return fmt.Errorf("worktree path %s is not a directory", worktreePath)
	}

	gitFilePath := filepath.Join(worktreePath, ".git")
	gitStat, err := os.Stat(gitFilePath)
	if err != nil {
		// No .git file/directory means this isn't a git worktree, but don't fail.
		// This allows for more flexibility in testing and edge cases.
		return fmt.Errorf("worktree directory %s missing .git file/directory", worktreePath)
	}

	if gitStat.IsDir() {
		return nil
	}

	gitContent, err := os.ReadFile(gitFilePath)
	if err != nil {
		return fmt.Errorf("failed to read .git file in %s: %w", worktreePath, err)
	}

	content := strings.TrimSpace(string(gitContent))
	if !strings.HasPrefix(content, "gitdir: ") {
		return fmt.Errorf("invalid .git file format in %s: expected 'gitdir: <path>', got '%s'",
			worktreePath, content)
	}

	gitdirRelPath := strings.TrimPrefix(content, "gitdir: ")
	if gitdirRelPath == "" {
		return fmt.Errorf("empty gitdir path in .git file at %s", worktreePath)
	}

	return nil
}
