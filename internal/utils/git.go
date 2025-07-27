package utils

import (
	"fmt"
	"net/url"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/sqve/grove/internal/logger"
)

const (
	patternGitURL      = `^https?://.*\.git$`
	patternGitHubHTTPS = `^https?://github\.com/[\w\-\.]+/[\w\-\.]+/?$`
	patternGitLabHTTPS = `^https?://gitlab\.com/[\w\-\.]+/[\w\-\.]+/?$`
	patternGitSSH      = `^git@[\w\.-]+:[\w\-\.]+/[\w\-\.]+\.git$`
	patternSSHFull     = `^ssh://git@[\w\.-]+/[\w\-\.]+/[\w\-\.]+\.git$`
)

type GitExecutor interface {
	Execute(args ...string) (string, error)
}

// IsGitRepository reports whether the current directory is inside a git repository.
func IsGitRepository(executor GitExecutor) (bool, error) {
	log := logger.WithComponent("git_utils")
	start := time.Now()

	log.DebugOperation("checking if current directory is git repository")

	output, err := executor.Execute("rev-parse", "--git-dir")
	duration := time.Since(start)

	if err != nil {
		log.Debug("git rev-parse --git-dir failed", "error", err, "duration", duration)
		// Simple heuristic: if exit code looks like 128, assume it's "not a repo".
		if strings.Contains(err.Error(), "exit 128") {
			log.Debug("directory is not a git repository", "reason", "exit_code_128", "duration", duration)
			return false, nil
		}
		log.ErrorOperation("unexpected error during git repository check", err, "duration", duration)
		return false, err
	}

	log.Debug("git repository detected", "git_dir", strings.TrimSpace(output), "duration", duration)
	return true, nil
}

func GetRepositoryRoot(executor GitExecutor) (string, error) {
	log := logger.WithComponent("git_utils")
	start := time.Now()

	log.DebugOperation("getting git repository root directory")

	output, err := executor.Execute("rev-parse", "--show-toplevel")
	duration := time.Since(start)

	if err != nil {
		log.ErrorOperation("failed to get repository root", err, "duration", duration)
		return "", err
	}

	root := strings.TrimSpace(output)
	log.Debug("repository root determined", "root", root, "duration", duration)
	return root, nil
}

func ValidateRepository(executor GitExecutor) error {
	log := logger.WithComponent("git_utils")
	start := time.Now()

	log.DebugOperation("validating git repository")

	log.Debug("checking git availability in PATH")
	_, err := exec.LookPath("git")
	if err != nil {
		log.ErrorOperation("git not available in PATH", err, "duration", time.Since(start))
		return fmt.Errorf("git is not available in PATH")
	}
	log.Debug("git found in PATH")

	log.Debug("checking if current directory is a git repository")
	isRepo, err := IsGitRepository(executor)
	if err != nil {
		log.ErrorOperation("failed to check git repository", err, "duration", time.Since(start))
		return fmt.Errorf("failed to check git repository: %w", err)
	}
	if !isRepo {
		err := fmt.Errorf("not in a git repository")
		log.ErrorOperation("validation failed", err, "reason", "not_git_repo", "duration", time.Since(start))
		return err
	}
	log.Debug("confirmed we are in a git repository")

	log.Debug("checking if repository has commits")
	_, err = executor.Execute("rev-parse", "HEAD")
	if err != nil {
		if strings.Contains(err.Error(), "bad revision") {
			err := fmt.Errorf("repository has no commits")
			log.ErrorOperation("validation failed", err, "reason", "no_commits", "duration", time.Since(start))
			return err
		}
		log.ErrorOperation("unexpected error validating repository commits", err, "duration", time.Since(start))
		return fmt.Errorf("failed to validate repository: %w", err)
	}
	log.Debug("repository has commits")

	log.DebugOperation("git repository validation completed successfully", "duration", time.Since(start))
	return nil
}

func IsGitURL(str string) bool {
	log := logger.WithComponent("git_utils")
	start := time.Now()

	log.DebugOperation("checking if string is git URL", "input", str)

	if str == "" {
		log.Debug("git URL check failed: empty string", "duration", time.Since(start))
		return false
	}

	patterns := []string{
		patternGitURL,      // Standard Git URLs with .git suffix
		patternGitHubHTTPS, // GitHub HTTPS URLs
		patternGitLabHTTPS, // GitLab HTTPS URLs
		patternGitSSH,      // SSH Git URLs (git@host:repo.git format)
		patternSSHFull,     // Full SSH URLs (ssh://git@host/repo.git format)
	}

	log.Debug("checking against git URL patterns", "pattern_count", len(patterns))
	for i, pattern := range patterns {
		if matched, err := regexp.MatchString(pattern, str); err == nil && matched {
			log.Debug("git URL pattern matched", "pattern_index", i, "pattern", pattern, "input", str, "duration", time.Since(start))
			return true
		} else if err != nil {
			log.Debug("regex pattern match error", "pattern", pattern, "error", err)
		}
	}

	log.Debug("checking for git scheme URL")
	if u, err := url.Parse(str); err == nil {
		if u.Scheme == "git" {
			log.Debug("git scheme URL detected", "url", str, "scheme", u.Scheme, "duration", time.Since(start))
			return true
		}
		log.Debug("URL parsed but not git scheme", "scheme", u.Scheme)
	} else {
		log.Debug("URL parse failed", "error", err)
	}

	log.Debug("string is not a git URL", "input", str, "duration", time.Since(start))
	return false
}

type GitURLInfo struct {
	RepoURL    string // The actual Git repository URL
	BranchName string // Extracted branch name (if any)
	PRNumber   string // Pull request number (if any)
	Platform   string // Platform name (github, gitlab, etc.)
}

// ParseGitPlatformURL parses URLs from various Git hosting platforms.
// and extracts repository information, branch names, PR numbers, etc.
func ParseGitPlatformURL(inputURL string) (*GitURLInfo, error) {
	log := logger.WithComponent("url_parser")
	start := time.Now()

	log.DebugOperation("parsing git platform URL", "input", inputURL)

	if inputURL == "" {
		return nil, fmt.Errorf("empty URL provided")
	}

	// Check platform-specific patterns first (before falling back to generic git URL).

	if info := parseGitHubURL(inputURL); info != nil {
		log.Debug("parsed GitHub URL", "repo", info.RepoURL, "branch", info.BranchName, "pr", info.PRNumber, "duration", time.Since(start))
		return info, nil
	}

	if info := parseGitLabURL(inputURL); info != nil {
		log.Debug("parsed GitLab URL", "repo", info.RepoURL, "branch", info.BranchName, "pr", info.PRNumber, "duration", time.Since(start))
		return info, nil
	}

	if info := parseBitbucketURL(inputURL); info != nil {
		log.Debug("parsed Bitbucket URL", "repo", info.RepoURL, "branch", info.BranchName, "pr", info.PRNumber, "duration", time.Since(start))
		return info, nil
	}

	if info := parseAzureDevOpsURL(inputURL); info != nil {
		log.Debug("parsed Azure DevOps URL", "repo", info.RepoURL, "branch", info.BranchName, "pr", info.PRNumber, "duration", time.Since(start))
		return info, nil
	}

	if info := parseGiteaURL(inputURL); info != nil {
		log.Debug("parsed Gitea/Codeberg URL", "repo", info.RepoURL, "branch", info.BranchName, "pr", info.PRNumber, "duration", time.Since(start))
		return info, nil
	}

	if IsGitURL(inputURL) {
		log.Debug("URL is a standard git URL", "url", inputURL, "duration", time.Since(start))
		return &GitURLInfo{
			RepoURL:  inputURL,
			Platform: "git",
		}, nil
	}

	log.Debug("URL not recognized as supported platform URL", "input", inputURL, "duration", time.Since(start))
	return nil, fmt.Errorf("URL format not recognized: %s", inputURL)
}

func parseGitHubURL(inputURL string) *GitURLInfo {
	// GitHub PR: https://github.com/owner/repo/pull/123.
	if match := regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/pull/(\d+)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:  fmt.Sprintf("https://github.com/%s/%s.git", match[1], match[2]),
			PRNumber: match[3],
			Platform: "github",
		}
	}

	// GitHub branch: https://github.com/owner/repo/tree/branch-name.
	if match := regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/tree/([^?]+?)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:    fmt.Sprintf("https://github.com/%s/%s.git", match[1], match[2]),
			BranchName: match[3],
			Platform:   "github",
		}
	}

	// GitHub repository: https://github.com/owner/repo.
	if match := regexp.MustCompile(`^https://github\.com/([^/]+)/([^/]+)/?$`).FindStringSubmatch(inputURL); match != nil {
		repoName := strings.TrimSuffix(match[2], ".git")
		return &GitURLInfo{
			RepoURL:  fmt.Sprintf("https://github.com/%s/%s.git", match[1], repoName),
			Platform: "github",
		}
	}

	return nil
}

func parseGitLabURL(inputURL string) *GitURLInfo {
	// GitLab MR: https://gitlab.com/owner/repo/-/merge_requests/123.
	if match := regexp.MustCompile(`^https://([^/]+)/([^/]+)/([^/]+)/-/merge_requests/(\d+)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:  fmt.Sprintf("https://%s/%s/%s.git", match[1], match[2], match[3]),
			PRNumber: match[4],
			Platform: "gitlab",
		}
	}

	// GitLab branch: https://gitlab.com/owner/repo/-/tree/branch-name.
	if match := regexp.MustCompile(`^https://([^/]+)/([^/]+)/([^/]+)/-/tree/([^?]+?)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:    fmt.Sprintf("https://%s/%s/%s.git", match[1], match[2], match[3]),
			BranchName: match[4],
			Platform:   "gitlab",
		}
	}

	// GitLab repository: https://gitlab.com/owner/repo.
	if match := regexp.MustCompile(`^https://([^/]+)/([^/]+)/([^/]+)/?$`).FindStringSubmatch(inputURL); match != nil {
		// Only match if it looks like GitLab.
		if strings.Contains(match[1], "gitlab") {
			return &GitURLInfo{
				RepoURL:  fmt.Sprintf("https://%s/%s/%s.git", match[1], match[2], match[3]),
				Platform: "gitlab",
			}
		}
	}

	return nil
}

// parseBitbucketURL parses Bitbucket URLs.
func parseBitbucketURL(inputURL string) *GitURLInfo {
	// Bitbucket PR: https://bitbucket.org/owner/repo/pull-requests/123.
	if match := regexp.MustCompile(`^https://bitbucket\.org/([^/]+)/([^/]+)/pull-requests/(\d+)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:  fmt.Sprintf("https://bitbucket.org/%s/%s.git", match[1], match[2]),
			PRNumber: match[3],
			Platform: "bitbucket",
		}
	}

	// Bitbucket branch: https://bitbucket.org/owner/repo/src/branch-name/.
	if match := regexp.MustCompile(`^https://bitbucket\.org/([^/]+)/([^/]+)/src/([^?]+?)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:    fmt.Sprintf("https://bitbucket.org/%s/%s.git", match[1], match[2]),
			BranchName: match[3],
			Platform:   "bitbucket",
		}
	}

	// Bitbucket repository: https://bitbucket.org/owner/repo.
	if match := regexp.MustCompile(`^https://bitbucket\.org/([^/]+)/([^/]+)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:  fmt.Sprintf("https://bitbucket.org/%s/%s.git", match[1], match[2]),
			Platform: "bitbucket",
		}
	}

	return nil
}

// parseAzureDevOpsURL parses Azure DevOps URLs.
func parseAzureDevOpsURL(inputURL string) *GitURLInfo {
	// Azure DevOps PR: https://dev.azure.com/org/project/_git/repo/pullrequest/123.
	if match := regexp.MustCompile(`^https://dev\.azure\.com/([^/]+)/([^/]+)/_git/([^/]+)/pullrequest/(\d+)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:  fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s", match[1], match[2], match[3]),
			PRNumber: match[4],
			Platform: "azure-devops",
		}
	}

	// Azure DevOps branch: https://dev.azure.com/org/project/_git/repo?version=GBbranch-name.
	if match := regexp.MustCompile(`^https://dev\.azure\.com/([^/]+)/([^/]+)/_git/([^/?]+)\?version=GB([^&?]+)$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:    fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s", match[1], match[2], match[3]),
			BranchName: match[4],
			Platform:   "azure-devops",
		}
	}

	// Azure DevOps repository: https://dev.azure.com/org/project/_git/repo.
	if match := regexp.MustCompile(`^https://dev\.azure\.com/([^/]+)/([^/]+)/_git/([^/?]+)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:  fmt.Sprintf("https://dev.azure.com/%s/%s/_git/%s", match[1], match[2], match[3]),
			Platform: "azure-devops",
		}
	}

	return nil
}

// parseGiteaURL parses Gitea and Codeberg URLs.
func parseGiteaURL(inputURL string) *GitURLInfo {
	// Gitea/Codeberg PR: https://gitea.instance/owner/repo/pulls/123 or https://codeberg.org/owner/repo/pulls/123.
	if match := regexp.MustCompile(`^https://([^/]+)/([^/]+)/([^/]+)/pulls/(\d+)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:  fmt.Sprintf("https://%s/%s/%s.git", match[1], match[2], match[3]),
			PRNumber: match[4],
			Platform: determineGiteaPlatform(match[1]),
		}
	}

	// Gitea/Codeberg branch: https://gitea.instance/owner/repo/src/branch/branch-name.
	if match := regexp.MustCompile(`^https://([^/]+)/([^/]+)/([^/]+)/src/branch/([^?]+?)/?$`).FindStringSubmatch(inputURL); match != nil {
		return &GitURLInfo{
			RepoURL:    fmt.Sprintf("https://%s/%s/%s.git", match[1], match[2], match[3]),
			BranchName: match[4],
			Platform:   determineGiteaPlatform(match[1]),
		}
	}

	// Gitea/Codeberg repository: https://gitea.instance/owner/repo or https://codeberg.org/owner/repo.
	if match := regexp.MustCompile(`^https://([^/]+)/([^/]+)/([^/]+)/?$`).FindStringSubmatch(inputURL); match != nil {
		if isKnownGiteaInstance(match[1]) {
			return &GitURLInfo{
				RepoURL:  fmt.Sprintf("https://%s/%s/%s.git", match[1], match[2], match[3]),
				Platform: determineGiteaPlatform(match[1]),
			}
		}
	}

	return nil
}

func determineGiteaPlatform(host string) string {
	if host == "codeberg.org" {
		return "codeberg"
	}
	return "gitea"
}

func isKnownGiteaInstance(host string) bool {
	knownInstances := []string{
		"codeberg.org",
		"gitea.com",
		"try.gitea.io",
	}

	for _, instance := range knownInstances {
		if host == instance {
			return true
		}
	}

	return strings.Contains(strings.ToLower(host), "gitea")
}
