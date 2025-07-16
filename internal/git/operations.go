package git

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitExecutor defines the interface for executing git commands.
type GitExecutor interface {
	Execute(args ...string) (string, error)
}

// DefaultGitExecutor implements GitExecutor using real git commands.
type DefaultGitExecutor struct{}

// Execute runs a real git command.
func (e *DefaultGitExecutor) Execute(args ...string) (string, error) {
	return ExecuteGit(args...)
}

// DefaultExecutor is the default git command executor.
var DefaultExecutor GitExecutor = &DefaultGitExecutor{}

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

// Runs a git command with the given arguments and returns stdout.
// If the command fails, it returns a GitError with stderr and exit code.
func ExecuteGit(args ...string) (string, error) {
	cmd := exec.Command("git", args...)

	stdout, err := cmd.Output()
	if err != nil {
		var stderr string
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			stderr = string(exitErr.Stderr)
		}

		return "", &GitError{
			Command:  "git",
			Args:     args,
			Stderr:   stderr,
			ExitCode: cmd.ProcessState.ExitCode(),
		}
	}

	return strings.TrimSpace(string(stdout)), nil
}

// Runs git clone --bare for the given repository URL.
func CloneBare(repoURL, targetDir string) error {
	return CloneBareWithExecutor(DefaultExecutor, repoURL, targetDir)
}

// CloneBareWithExecutor runs git clone --bare using the specified executor.
func CloneBareWithExecutor(executor GitExecutor, repoURL, targetDir string) error {
	_, err := executor.Execute("clone", "--bare", repoURL, targetDir)
	return err
}

// Writes a .git file with gitdir pointing to the bare repository.
func CreateGitFile(mainDir, bareDir string) error {
	gitFilePath := filepath.Join(mainDir, ".git")

	// Make bareDir relative to mainDir if possible, otherwise use absolute path
	relPath, err := filepath.Rel(mainDir, bareDir)
	if err != nil {
		relPath = bareDir
	}

	content := fmt.Sprintf("gitdir: %s\n", relPath)
	return os.WriteFile(gitFilePath, []byte(content), 0o600)
}

// Sets up fetch refspec and fetches all remote branches.
func ConfigureRemoteTracking() error {
	return ConfigureRemoteTrackingWithExecutor(DefaultExecutor)
}

// ConfigureRemoteTrackingWithExecutor sets up fetch refspec using the specified executor.
func ConfigureRemoteTrackingWithExecutor(executor GitExecutor) error {
	// Set up fetch refspec to get all remote branches
	_, err := executor.Execute("config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*")
	if err != nil {
		return err
	}

	// Fetch all remote branches
	_, err = executor.Execute("fetch")
	return err
}

// Configures branch.*.remote for existing local branches.
func SetupUpstreamBranches() error {
	return SetupUpstreamBranchesWithExecutor(DefaultExecutor)
}

// SetupUpstreamBranchesWithExecutor configures upstream tracking using the specified executor.
func SetupUpstreamBranchesWithExecutor(executor GitExecutor) error {
	// Get all local branches
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
		_, err := executor.Execute("branch", "--set-upstream-to=origin/"+branch, branch)
		if err != nil {
			// Continue if this branch doesn't exist on remote
			continue
		}
	}

	return nil
}

// Runs git init --bare in the target directory.
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
	if err := validateConversionPreconditions(dir); err != nil {
		return err
	}

	if err := checkRepositoryClean(executor, dir); err != nil {
		return err
	}

	return performConversion(dir)
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
	originalDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		return nil, fmt.Errorf("failed to change to directory %s: %w", dir, err)
	}

	defer func() { _ = os.Chdir(originalDir) }()

	var issues []SafetyIssue

	// Check for uncommitted changes and ongoing operations
	if statusIssues, err := checkGitStatus(executor); err != nil {
		return nil, err
	} else {
		issues = append(issues, statusIssues...)
	}

	// Check for stashed changes
	if stashIssues, err := checkStashedChanges(executor); err != nil {
		return nil, err
	} else {
		issues = append(issues, stashIssues...)
	}

	// Check for untracked files
	if untrackedIssues, err := checkUntrackedFiles(executor); err != nil {
		return nil, err
	} else {
		issues = append(issues, untrackedIssues...)
	}

	// Check for existing worktrees
	if worktreeIssues, err := checkExistingWorktrees(executor); err != nil {
		return nil, err
	} else {
		issues = append(issues, worktreeIssues...)
	}

	// Check for unpushed commits
	if unpushedIssues, err := checkUnpushedCommits(executor); err != nil {
		return nil, err
	} else {
		issues = append(issues, unpushedIssues...)
	}

	// Check for local-only branches
	if localBranchIssues, err := checkLocalOnlyBranches(executor); err != nil {
		return nil, err
	} else {
		issues = append(issues, localBranchIssues...)
	}

	return issues, nil
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

	// Create backup of original .git directory
	if err := os.Rename(gitDir, backupDir); err != nil {
		return fmt.Errorf("failed to create backup of .git directory: %w", err)
	}

	// Move .git directory to .bare
	if err := os.Rename(backupDir, bareDir); err != nil {
		_ = os.Rename(backupDir, gitDir)
		return fmt.Errorf("failed to move .git to .bare: %w", err)
	}

	// Create .git file pointing to .bare
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

// ValidateGroveStructure verifies that a Grove repository structure is valid and functional.
func ValidateGroveStructure(dir string) error {
	return ValidateGroveStructureWithExecutor(DefaultExecutor, dir)
}

// ValidateGroveStructureWithExecutor validates using the specified executor.
func ValidateGroveStructureWithExecutor(executor GitExecutor, dir string) error {
	// Check that .git file exists and is not a directory
	gitPath := filepath.Join(dir, ".git")
	gitInfo, err := os.Stat(gitPath)
	if err != nil {
		return fmt.Errorf(".git file does not exist: %w", err)
	}
	if gitInfo.IsDir() {
		return fmt.Errorf(".git should be a file, not a directory")
	}

	// Check that .bare directory exists
	bareDir := filepath.Join(dir, ".bare")
	bareInfo, err := os.Stat(bareDir)
	if err != nil {
		return fmt.Errorf(".bare directory does not exist: %w", err)
	}
	if !bareInfo.IsDir() {
		return fmt.Errorf(".bare should be a directory")
	}

	// Validate .git file content
	gitContent, err := os.ReadFile(gitPath)
	if err != nil {
		return fmt.Errorf("failed to read .git file: %w", err)
	}

	expectedContent := fmt.Sprintf("gitdir: %s\n", ".bare")
	if string(gitContent) != expectedContent {
		return fmt.Errorf(".git file content is invalid, expected 'gitdir: .bare\\n', got '%s'", string(gitContent))
	}

	// Test that git operations work
	originalDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	if err := os.Chdir(dir); err != nil {
		return fmt.Errorf("failed to change to directory %s: %w", dir, err)
	}
	defer func() { _ = os.Chdir(originalDir) }()

	// Test basic git operation
	_, err = executor.Execute("status")
	if err != nil {
		return fmt.Errorf("git status failed in converted repository: %w", err)
	}

	return nil
}
