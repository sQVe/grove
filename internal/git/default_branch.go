package git

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/sqve/grove/internal/logger"
)

// DetectDefaultBranch detects the default branch for a repository using a multi-tier fallback strategy.
//
// The detection follows this priority order:
// 1. Local remote HEAD cache (fast, no network)
// 2. Current branch (for conversion scenarios)
// 3. Remote symbolic reference (network, fast)
// 4. Remote show command (network, comprehensive)
// 5. Common branch pattern matching (heuristic)
// 6. First remote branch (last resort)
// 7. Hard-coded "main" (absolute fallback)
//
// Parameters:
// - executor: GitExecutor interface for running git commands
// - remoteName: Name of the remote to query (e.g., "origin", "upstream")
//
// Returns the detected default branch name and any error encountered.
func DetectDefaultBranch(executor GitExecutor, remoteName string) (string, error) {
	log := logger.WithComponent("default_branch")
	start := time.Now()

	log.Debug("starting default branch detection",
		"remote", remoteName,
		"tiers", []string{"local_cache", "current_branch", "remote_symref", "remote_show", "common_patterns", "first_remote", "fallback"},
	)

	// Tier 1: Fast local detection
	log.Debug("tier 1: checking local remote HEAD cache", "remote", remoteName)
	if branch := checkLocalRemoteHead(executor, log, remoteName); branch != "" {
		log.Info("default branch detected via local remote HEAD",
			"branch", branch,
			"remote", remoteName,
			"method", "local_cache",
			"duration", time.Since(start),
		)
		return branch, nil
	}
	log.Debug("tier 1: local remote HEAD check failed")

	log.Debug("tier 1: checking current branch")
	if branch := checkCurrentBranch(executor, log); branch != "" {
		log.Info("default branch detected via current branch",
			"branch", branch,
			"method", "current_branch",
			"duration", time.Since(start),
		)
		return branch, nil
	}
	log.Debug("tier 1: current branch check failed")

	// Tier 2: Network-based detection (with timeout)
	log.Debug("tier 2: starting network-based detection", "timeout", "5s")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debug("tier 2: checking remote symbolic reference", "remote", remoteName)
	if branch := checkRemoteSymref(executor, log, ctx, remoteName); branch != "" {
		log.Info("default branch detected via remote symref",
			"branch", branch,
			"remote", remoteName,
			"method", "remote_symref",
			"duration", time.Since(start),
		)
		return branch, nil
	}
	log.Debug("tier 2: remote symref check failed")

	log.Debug("tier 2: checking remote show command", "remote", remoteName)
	if branch := checkRemoteShow(executor, log, ctx, remoteName); branch != "" {
		log.Info("default branch detected via remote show",
			"branch", branch,
			"remote", remoteName,
			"method", "remote_show",
			"duration", time.Since(start),
		)
		return branch, nil
	}
	log.Debug("tier 2: remote show check failed")

	// Tier 3: Heuristic fallback
	log.Debug("tier 3: trying heuristic fallback methods")
	log.Debug("tier 3: checking common branch patterns", "remote", remoteName)
	if branch := findCommonBranchPattern(executor, log, remoteName); branch != "" {
		log.Info("default branch detected via common pattern",
			"branch", branch,
			"remote", remoteName,
			"method", "common_pattern",
			"duration", time.Since(start),
		)
		return branch, nil
	}
	log.Debug("tier 3: common branch pattern check failed")

	log.Debug("tier 3: getting first remote branch", "remote", remoteName)
	if branch := getFirstRemoteBranch(executor, log, remoteName); branch != "" {
		log.Info("default branch detected via first remote",
			"branch", branch,
			"remote", remoteName,
			"method", "first_remote",
			"duration", time.Since(start),
		)
		return branch, nil
	}
	log.Debug("tier 3: first remote branch check failed")

	// Tier 4: Hard-coded default
	log.Debug("tier 4: using hard-coded default fallback")
	log.Warn("default branch detection exhausted all methods, using fallback",
		"branch", "main",
		"remote", remoteName,
		"method", "fallback",
		"duration", time.Since(start),
	)
	return "main", nil
}

// isValidBranchName validates a branch name according to Git's check-ref-format rules.
// Returns true if the branch name is valid for use as a Git branch name.
//
// Git branch name rules:
// - Cannot contain ASCII control characters (< 0x20 or 0x7F DEL), space, ~, ^, :
// - Cannot contain ?, *, [
// - Cannot contain two consecutive dots (..)
// - Cannot begin or end with /
// - Cannot contain multiple consecutive slashes
// - Cannot end with .
// - Cannot contain sequence @{
// - Cannot be single character @
// - Cannot contain backslash \
// - Cannot begin with a dash - (branch-specific rule)
// - Should not be empty or contain only whitespace
func isValidBranchName(log *logger.Logger, name string) bool {
	if name == "" {
		log.Debug("branch name validation failed: empty name")
		return false
	}

	// Trim whitespace and check if empty
	trimmed := strings.TrimSpace(name)
	if trimmed == "" || trimmed != name {
		log.Debug("branch name validation failed: contains leading/trailing whitespace", "name", name)
		return false // Contains leading/trailing whitespace
	}

	// Cannot be single character @
	if name == "@" {
		log.Debug("branch name validation failed: single @ character", "name", name)
		return false
	}

	// Cannot begin with dash (branch-specific rule)
	if strings.HasPrefix(name, "-") {
		log.Debug("branch name validation failed: begins with dash", "name", name)
		return false
	}

	// Cannot begin or end with /
	if strings.HasPrefix(name, "/") || strings.HasSuffix(name, "/") {
		log.Debug("branch name validation failed: begins or ends with slash", "name", name)
		return false
	}

	// Cannot end with .
	if strings.HasSuffix(name, ".") {
		log.Debug("branch name validation failed: ends with dot", "name", name)
		return false
	}

	// Cannot contain .. (consecutive dots)
	if strings.Contains(name, "..") {
		log.Debug("branch name validation failed: contains consecutive dots", "name", name)
		return false
	}

	// Cannot contain @{
	if strings.Contains(name, "@{") {
		log.Debug("branch name validation failed: contains @{ sequence", "name", name)
		return false
	}

	// Cannot contain multiple consecutive slashes
	if strings.Contains(name, "//") {
		log.Debug("branch name validation failed: contains consecutive slashes", "name", name)
		return false
	}

	// Check for forbidden characters using regex
	// ASCII control chars (< 0x20), DEL (0x7F), space, ~, ^, :, ?, *, [, \
	forbiddenChars := regexp.MustCompile(`[\x00-\x1F\x7F ~^:?*\[\\]`)
	if forbiddenChars.MatchString(name) {
		log.Debug("branch name validation failed: contains forbidden characters", "name", name)
		return false
	}

	// Additional check: no slash-separated component can begin with . or end with .lock
	components := strings.Split(name, "/")
	for _, component := range components {
		if component == "" {
			continue // Skip empty components (handled by consecutive slash check)
		}
		if strings.HasPrefix(component, ".") {
			log.Debug("branch name validation failed: component begins with dot", "name", name, "component", component)
			return false
		}
		if strings.HasSuffix(component, ".lock") {
			log.Debug("branch name validation failed: component ends with .lock", "name", name, "component", component)
			return false
		}
	}

	log.Debug("branch name validation passed", "name", name)
	return true
}

// checkLocalRemoteHead checks if the local remote HEAD is set and returns the branch name.
func checkLocalRemoteHead(executor GitExecutor, log *logger.Logger, remoteName string) string {
	refPath := fmt.Sprintf("refs/remotes/%s/HEAD", remoteName)
	log.Debug("checking local remote HEAD", "ref_path", refPath)

	output, err := executor.Execute("symbolic-ref", refPath)
	if err != nil {
		log.Debug("symbolic-ref command failed", "error", err)
		return ""
	}

	// Output format: refs/remotes/{remote}/main
	trimmed := strings.TrimSpace(output)
	log.Debug("git symbolic-ref output", "output", trimmed)

	expectedPrefix := fmt.Sprintf("refs/remotes/%s/", remoteName)
	if strings.HasPrefix(trimmed, expectedPrefix) {
		branch := strings.TrimPrefix(trimmed, expectedPrefix)
		log.Debug("extracted branch from symref", "branch", branch)

		if isValidBranchName(log, branch) {
			log.Debug("branch name validation passed", "branch", branch)
			return branch
		}
		log.Debug("branch name validation failed", "branch", branch)
	} else {
		log.Debug("symref output format unexpected", "output", trimmed, "expected_prefix", expectedPrefix)
	}

	return ""
}

// checkCurrentBranch returns the currently checked out branch.
// This is useful for conversion scenarios where we want to preserve the user's current context.
func checkCurrentBranch(executor GitExecutor, log *logger.Logger) string {
	log.Debug("checking current branch")

	output, err := executor.Execute("branch", "--show-current")
	if err != nil {
		log.Debug("branch --show-current failed", "error", err)
		return ""
	}

	branch := strings.TrimSpace(output)
	log.Debug("git branch --show-current output", "output", branch)

	if branch == "" {
		log.Debug("no current branch detected (detached HEAD?)")
		return ""
	}

	if isValidBranchName(log, branch) {
		log.Debug("current branch name validation passed", "branch", branch)
		return branch
	}

	log.Debug("current branch name validation failed", "branch", branch)
	return ""
}

// checkRemoteSymref queries the remote for its symbolic reference to HEAD.
func checkRemoteSymref(executor GitExecutor, log *logger.Logger, ctx context.Context, remoteName string) string {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		log.Debug("context already cancelled for remote symref check")
		return ""
	default:
	}

	log.Debug("running git ls-remote --symref", "remote", remoteName)
	output, err := executor.ExecuteWithContext(ctx, "ls-remote", "--symref", remoteName, "HEAD")
	if err != nil {
		log.Debug("ls-remote --symref failed", "remote", remoteName, "error", err)
		return ""
	}

	log.Debug("git ls-remote --symref output", "output", output)

	// Parse output like: "ref: refs/heads/main	HEAD"
	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		log.Debug("parsing symref line", "line", line)
		if strings.HasPrefix(line, "ref: refs/heads/") {
			// Extract branch name from "ref: refs/heads/main	HEAD"
			parts := strings.Fields(line)
			if len(parts) >= 2 && parts[0] == "ref:" {
				ref := parts[1]
				log.Debug("found symref", "ref", ref)
				if strings.HasPrefix(ref, "refs/heads/") {
					branch := strings.TrimPrefix(ref, "refs/heads/")
					log.Debug("extracted branch from symref", "branch", branch)
					if isValidBranchName(log, branch) {
						log.Debug("symref branch name validation passed", "branch", branch)
						return branch
					}
					log.Debug("symref branch name validation failed", "branch", branch)
				}
			}
		}
	}
	log.Debug("no valid symref found in output")
	return ""
}

// checkRemoteShow uses git remote show to get comprehensive remote information.
func checkRemoteShow(executor GitExecutor, log *logger.Logger, ctx context.Context, remoteName string) string {
	// Check if context is already cancelled
	select {
	case <-ctx.Done():
		log.Debug("context already cancelled for remote show check")
		return ""
	default:
	}

	log.Debug("running git remote show", "remote", remoteName)
	output, err := executor.ExecuteWithContext(ctx, "remote", "show", remoteName)
	if err != nil {
		log.Debug("remote show failed", "remote", remoteName, "error", err)
		return ""
	}

	log.Debug("git remote show output", "output", output)

	// Look for "HEAD branch: main" in the output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "HEAD branch: ") {
			log.Debug("found HEAD branch line", "line", trimmed)
			branch := strings.TrimPrefix(trimmed, "HEAD branch: ")
			branch = strings.TrimSpace(branch)
			log.Debug("extracted branch from remote show", "branch", branch)
			if isValidBranchName(log, branch) {
				log.Debug("remote show branch name validation passed", "branch", branch)
				return branch
			}
			log.Debug("remote show branch name validation failed", "branch", branch)
		}
	}
	log.Debug("no HEAD branch found in remote show output")
	return ""
}

// findCommonBranchPattern looks for common branch names in remote branches.
func findCommonBranchPattern(executor GitExecutor, log *logger.Logger, remoteName string) string {
	log.Debug("running git branch -r for pattern matching")
	output, err := executor.Execute("branch", "-r")
	if err != nil {
		log.Debug("branch -r failed", "error", err)
		return ""
	}

	log.Debug("git branch -r output", "output", output)

	// Common branch names in order of preference
	commonBranches := []string{"main", "master", "develop", "trunk"}
	log.Debug("searching for common branch patterns", "patterns", commonBranches)

	lines := strings.Split(output, "\n")
	branches := make([]string, 0)

	// Extract branch names from remote branches
	remotePrefix := remoteName + "/"
	log.Debug("extracting remote branches", "remote_prefix", remotePrefix)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, remotePrefix) && !strings.Contains(trimmed, "HEAD") {
			branch := strings.TrimPrefix(trimmed, remotePrefix)
			branches = append(branches, branch)
			log.Debug("found remote branch", "branch", branch)
		}
	}

	log.Debug("all remote branches discovered", "branches", branches, "count", len(branches))

	// Check for common branch names in preference order
	for _, commonBranch := range commonBranches {
		log.Debug("checking for common pattern", "pattern", commonBranch)
		for _, branch := range branches {
			if branch == commonBranch && isValidBranchName(log, branch) {
				log.Debug("found matching common branch pattern", "branch", branch)
				return branch
			}
		}
	}

	log.Debug("no common branch patterns found")
	return ""
}

// getFirstRemoteBranch returns the first available remote branch as a last resort.
func getFirstRemoteBranch(executor GitExecutor, log *logger.Logger, remoteName string) string {
	log.Debug("running git branch -r for first branch fallback")
	output, err := executor.Execute("branch", "-r")
	if err != nil {
		log.Debug("branch -r failed", "error", err)
		return ""
	}

	log.Debug("git branch -r output for first branch", "output", output)

	lines := strings.Split(output, "\n")
	remotePrefix := remoteName + "/"
	log.Debug("searching for first valid remote branch", "remote_prefix", remotePrefix)

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, remotePrefix) && !strings.Contains(trimmed, "HEAD") {
			branch := strings.TrimPrefix(trimmed, remotePrefix)
			log.Debug("checking first branch candidate", "branch", branch)
			if isValidBranchName(log, branch) {
				log.Debug("found valid first remote branch", "branch", branch)
				return branch
			}
			log.Debug("first branch candidate invalid, continuing", "branch", branch)
		}
	}

	log.Debug("no valid remote branches found for first branch fallback")
	return ""
}
