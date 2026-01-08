package git

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/logger"
)

// ListBranches returns a list of all branches in a bare repository
func ListBranches(bareRepo string) ([]string, error) {
	logger.Debug("Executing: git branch -a --format=%%(refname:short) in %s", bareRepo)
	cmd, cancel := GitCommand("git", "branch", "-a", "--format=%(refname:short)")
	defer cancel()
	cmd.Dir = bareRepo

	out, err := executeWithOutputBuffer(cmd)
	if err != nil {
		return nil, err
	}

	branchSet := make(map[string]bool)
	scanner := bufio.NewScanner(out)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || line == "origin" {
			continue
		}

		if branchName, ok := strings.CutPrefix(line, "origin/"); ok {
			if branchName != "HEAD" {
				branchSet[branchName] = true
			}
		} else {
			branchSet[line] = true
		}
	}

	var branches []string
	for branch := range branchSet {
		branches = append(branches, branch)
	}

	return branches, scanner.Err()
}

// BranchExists checks if a branch exists locally or on any remote
func BranchExists(repoPath, branchName string) (bool, error) {
	if repoPath == "" || branchName == "" {
		return false, errors.New("repository path and branch name cannot be empty")
	}

	cmd, cancel := GitCommand("git", "rev-parse", "--verify", "--quiet", branchName) // nolint:gosec // Branch name validated by git
	cmd.Dir = repoPath
	err := cmd.Run()
	cancel()
	if err == nil {
		return true, nil
	}

	remotesCmd, cancelRemotes := GitCommand("git", "remote")
	defer cancelRemotes()
	remotesCmd.Dir = repoPath
	output, err := remotesCmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to list remotes: %w", err)
	}

	remotes := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, remote := range remotes {
		if remote == "" {
			continue
		}
		remoteBranch := remote + "/" + branchName
		cmd, cancelCmd := GitCommand("git", "rev-parse", "--verify", "--quiet", remoteBranch) // nolint:gosec // Branch name validated by git
		cmd.Dir = repoPath
		err := cmd.Run()
		cancelCmd()
		if err == nil {
			return true, nil
		}
	}

	return false, nil
}

// GetCurrentBranch returns the current branch name
func GetCurrentBranch(path string) (string, error) {
	if path == "" {
		return "", errors.New("repository path cannot be empty")
	}

	gitDir, err := resolveGitDir(path)
	if err != nil {
		return "", err
	}

	headFile := filepath.Join(gitDir, "HEAD")
	content, err := os.ReadFile(headFile) // nolint:gosec // Reading git HEAD file
	if err != nil {
		return "", err
	}

	line := strings.TrimSpace(string(content))

	if after, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
		return after, nil
	}

	return "", ErrDetachedHead
}

// ErrDetachedHead is returned when the worktree is in detached HEAD state
var ErrDetachedHead = errors.New("detached HEAD state")

// GetCurrentBranchOrDetached returns the branch name, or the short commit hash if detached.
// Returns (branch, detached, error) where detached indicates if HEAD is detached.
func GetCurrentBranchOrDetached(path string) (branch string, detached bool, err error) {
	branch, err = GetCurrentBranch(path)
	if err == nil {
		return branch, false, nil
	}
	if !errors.Is(err, ErrDetachedHead) {
		return "", false, err
	}

	// Get short commit hash for detached HEAD
	cmd, cancel := GitCommand("git", "rev-parse", "--short", "HEAD")
	defer cancel()
	cmd.Dir = path
	var output []byte
	output, err = cmd.Output()
	if err != nil {
		return "", true, nil // detached but couldn't get hash
	}
	return strings.TrimSpace(string(output)), true, nil
}

// GetDefaultBranch returns the default branch for a bare repository
func GetDefaultBranch(bareDir string) (string, error) {
	if bareDir == "" {
		return "", errors.New("repository path cannot be empty")
	}

	headFile := filepath.Join(bareDir, "HEAD")

	content, err := os.ReadFile(headFile) // nolint:gosec // Reading git HEAD file
	if err != nil {
		return "", fmt.Errorf("failed to read HEAD: %w", err)
	}

	line := strings.TrimSpace(string(content))

	if after, ok := strings.CutPrefix(line, "ref: refs/heads/"); ok {
		return after, nil
	}

	return "", fmt.Errorf("could not determine default branch from HEAD")
}

// IsDetachedHead checks if the repository is in detached HEAD state
func IsDetachedHead(path string) (bool, error) {
	gitDir, err := GetGitDir(path)
	if err != nil {
		return false, err
	}

	headFile := filepath.Join(gitDir, "HEAD")

	content, err := os.ReadFile(headFile) // nolint:gosec // Reading git HEAD file
	if err != nil {
		return false, err
	}

	line := strings.TrimSpace(string(content))

	return !strings.HasPrefix(line, "ref: refs/heads/"), nil
}

// IsUnbornHead checks if the repository has an unborn HEAD (no commits yet).
// An unborn HEAD occurs when HEAD points to a branch ref that doesn't exist,
// which happens in newly initialized repos before the first commit.
func IsUnbornHead(path string) (bool, error) {
	if path == "" {
		return false, errors.New("repository path cannot be empty")
	}

	gitDir, err := resolveGitDir(path)
	if err != nil {
		return false, err
	}

	headFile := filepath.Join(gitDir, "HEAD")
	content, err := os.ReadFile(headFile) // nolint:gosec // Reading git HEAD file
	if err != nil {
		return false, err
	}

	line := strings.TrimSpace(string(content))

	if !strings.HasPrefix(line, "ref: ") {
		return false, nil
	}

	refPath := strings.TrimPrefix(line, "ref: ")

	looseRef := filepath.Join(gitDir, refPath)
	if _, err := os.Stat(looseRef); err == nil {
		return false, nil
	}

	packedRefsPath := filepath.Join(gitDir, "packed-refs")
	if packedRefs, err := os.ReadFile(packedRefsPath); err == nil { // nolint:gosec
		for _, packedLine := range strings.Split(string(packedRefs), "\n") {
			if strings.HasPrefix(packedLine, "#") || strings.HasPrefix(packedLine, "^") {
				continue
			}
			fields := strings.Fields(packedLine)
			if len(fields) >= 2 && fields[1] == refPath {
				return false, nil
			}
		}
	}

	return true, nil
}

// DeleteBranch deletes a local branch
func DeleteBranch(repoPath, branchName string, force bool) error {
	flag := "-d"
	if force {
		flag = "-D"
	}
	logger.Debug("Executing: git branch %s %s in %s", flag, branchName, repoPath)
	cmd, cancel := GitCommand("git", "branch", flag, branchName) //nolint:gosec // Branch name comes from validated input
	defer cancel()
	cmd.Dir = repoPath
	return runGitCommand(cmd, true)
}

// RenameBranch renames a branch using git branch -m
func RenameBranch(repoPath, oldName, newName string) error {
	if repoPath == "" || oldName == "" || newName == "" {
		return errors.New("repository path, old name, and new name cannot be empty")
	}

	logger.Debug("Executing: git branch -m %s %s in %s", oldName, newName, repoPath)
	cmd, cancel := GitCommand("git", "branch", "-m", oldName, newName) // nolint:gosec // Branch names from validated input
	defer cancel()
	cmd.Dir = repoPath

	return runGitCommand(cmd, true)
}

// SetUpstreamBranch sets the upstream tracking branch for a local branch
func SetUpstreamBranch(worktreePath, upstream string) error {
	if worktreePath == "" || upstream == "" {
		return errors.New("worktree path and upstream cannot be empty")
	}

	logger.Debug("Executing: git branch --set-upstream-to=%s in %s", upstream, worktreePath)
	cmd, cancel := GitCommand("git", "branch", "--set-upstream-to="+upstream) // nolint:gosec // Upstream from validated input
	defer cancel()
	cmd.Dir = worktreePath

	return runGitCommand(cmd, true)
}

// LocalBranchExists checks if a local branch (not remote-tracking) exists in the repository.
func LocalBranchExists(repoPath, branch string) (bool, error) {
	if repoPath == "" || branch == "" {
		return false, errors.New("repository path and branch name cannot be empty")
	}

	logger.Debug("Executing: git show-ref --verify --quiet refs/heads/%s in %s", branch, repoPath)
	cmd, cancel := GitCommand("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch) // nolint:gosec
	defer cancel()
	cmd.Dir = repoPath

	if err := cmd.Run(); err != nil {
		// Exit code 1 means ref not found - this is expected, not an error
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// CompareBranchRefs returns how localRef compares to remoteRef.
// Returns (ahead, behind) where:
//   - ahead = commits in localRef not in remoteRef
//   - behind = commits in remoteRef not in localRef
func CompareBranchRefs(repoPath, localRef, remoteRef string) (ahead, behind int, err error) {
	if repoPath == "" || localRef == "" || remoteRef == "" {
		return 0, 0, errors.New("repository path and ref names cannot be empty")
	}

	refRange := localRef + "..." + remoteRef
	logger.Debug("Executing: git rev-list --left-right --count %s in %s", refRange, repoPath)
	cmd, cancel := GitCommand("git", "rev-list", "--left-right", "--count", refRange) // nolint:gosec
	defer cancel()
	cmd.Dir = repoPath

	output, err := executeWithOutput(cmd)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to compare refs: %w", err)
	}

	// Output format: "ahead\tbehind" (e.g., "3\t5")
	_, err = fmt.Sscanf(output, "%d\t%d", &ahead, &behind)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to parse rev-list output %q: %w", output, err)
	}

	return ahead, behind, nil
}

// RevParse resolves a git reference to its full commit hash.
// This is equivalent to `git rev-parse <ref>`.
func RevParse(repoPath, ref string) (string, error) {
	if repoPath == "" {
		return "", errors.New("repository path cannot be empty")
	}
	if ref == "" {
		return "", errors.New("ref cannot be empty")
	}

	logger.Debug("Executing: git rev-parse %s in %s", ref, repoPath)
	cmd, cancel := GitCommand("git", "rev-parse", ref) // nolint:gosec
	defer cancel()
	cmd.Dir = repoPath

	hash, err := executeWithOutput(cmd)
	if err != nil {
		return "", fmt.Errorf("failed to resolve ref %q: %w", ref, err)
	}

	return hash, nil
}

// UpdateBranchRef updates a local branch to point to a target ref.
// This is equivalent to `git update-ref refs/heads/<branch> <target>`.
func UpdateBranchRef(repoPath, branch, targetRef string) error {
	if repoPath == "" || branch == "" || targetRef == "" {
		return errors.New("repository path, branch name, and target ref cannot be empty")
	}

	// First resolve the target ref to a commit hash
	logger.Debug("Executing: git rev-parse %s in %s", targetRef, repoPath)
	resolveCmd, cancelResolve := GitCommand("git", "rev-parse", targetRef) // nolint:gosec
	resolveCmd.Dir = repoPath
	targetHash, err := executeWithOutput(resolveCmd)
	cancelResolve()
	if err != nil {
		return fmt.Errorf("failed to resolve target ref %q: %w", targetRef, err)
	}

	// Now update the branch ref
	refPath := "refs/heads/" + branch
	logger.Debug("Executing: git update-ref %s %s in %s", refPath, targetHash, repoPath)
	cmd, cancel := GitCommand("git", "update-ref", refPath, targetHash) // nolint:gosec
	defer cancel()
	cmd.Dir = repoPath

	return runGitCommand(cmd, true)
}

// IsBranchMerged checks if a branch has been merged into the target branch.
// It detects both regular merges (via ancestry) and squash merges (via patch-id comparison).
func IsBranchMerged(repoPath, branch, targetBranch string) (bool, error) {
	if repoPath == "" {
		return false, errors.New("repository path cannot be empty")
	}
	if branch == "" {
		return false, errors.New("branch name cannot be empty")
	}
	if targetBranch == "" {
		return false, errors.New("target branch name cannot be empty")
	}

	// Check regular merge (ancestry)
	if isMergedByAncestry(repoPath, branch, targetBranch) {
		return true, nil
	}

	// Check squash merge (patch-id comparison)
	return isMergedByPatchID(repoPath, branch, targetBranch)
}

// isMergedByAncestry checks if branch is an ancestor of targetBranch
func isMergedByAncestry(repoPath, branch, targetBranch string) bool {
	// git merge-base --is-ancestor returns 0 if branch is ancestor of targetBranch
	cmd, cancel := GitCommand("git", "merge-base", "--is-ancestor", branch, targetBranch) // nolint:gosec
	defer cancel()
	cmd.Dir = repoPath
	return cmd.Run() == nil
}

// isMergedByPatchID detects squash merges using git cherry.
// Optimized from O(n) subprocess calls to a single git cherry call.
func isMergedByPatchID(repoPath, branch, targetBranch string) (bool, error) {
	// git cherry marks commits with "-" if an equivalent patch exists in target,
	// and "+" if the commit is unique to the branch (not in target).
	// This does the patch-id comparison internally in a single call.
	cmd, cancel := GitCommand("git", "cherry", "-v", targetBranch, branch) // nolint:gosec
	defer cancel()
	cmd.Dir = repoPath
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check patch-id merge status: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		// No commits unique to branch - already handled by ancestry check
		return true, nil
	}

	// If any commit is marked with "+", it's NOT in target (not squash-merged)
	for _, line := range lines {
		if line == "" {
			continue
		}
		// Lines starting with "+ " are NOT in target
		if strings.HasPrefix(line, "+ ") {
			return false, nil
		}
	}

	// All commits marked with "-" means they're all in target (squash-merged)
	return true, nil
}
