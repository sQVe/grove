package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// PRRef represents a parsed GitHub PR reference.
type PRRef struct {
	Owner  string // Empty if parsed from #N format
	Repo   string // Empty if parsed from #N format
	Number int
}

var (
	prNumberRegex = regexp.MustCompile(`^#(\d+)$`)
	prURLRegex    = regexp.MustCompile(`^https?://github\.com/([^/]+)/([^/]+)/pull/(\d+)/?$`)
)

// IsPRReference returns true if the input looks like a PR reference (#N or URL).
func IsPRReference(s string) bool {
	if prNumberRegex.MatchString(s) {
		return true
	}
	if prURLRegex.MatchString(s) {
		return true
	}
	return false
}

// IsPRURL returns true if the input is a full GitHub PR URL.
// Unlike IsPRReference, this does not match #N format.
func IsPRURL(s string) bool {
	return prURLRegex.MatchString(s)
}

// ParsePRReference parses a PR reference (#N or URL) into its components.
func ParsePRReference(s string) (*PRRef, error) {
	// Try #N format
	if matches := prNumberRegex.FindStringSubmatch(s); matches != nil {
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			return nil, err
		}
		return &PRRef{Number: num}, nil
	}

	// Try URL format
	if matches := prURLRegex.FindStringSubmatch(s); matches != nil {
		num, err := strconv.Atoi(matches[3])
		if err != nil {
			return nil, err
		}
		return &PRRef{
			Owner:  matches[1],
			Repo:   matches[2],
			Number: num,
		}, nil
	}

	return nil, errors.New("invalid PR reference: expected #N or GitHub PR URL")
}

// RepoRef holds owner and repo parsed from a remote URL.
type RepoRef struct {
	Owner string
	Repo  string
}

var (
	sshURLRegex   = regexp.MustCompile(`^git@github\.com:([^/]+)/(.+?)(?:\.git)?$`)
	httpsURLRegex = regexp.MustCompile(`^https://github\.com/([^/]+)/(.+?)(?:\.git)?$`)
)

// ParseRepoURL extracts owner/repo from a git remote URL.
func ParseRepoURL(url string) (*RepoRef, error) {
	url = strings.TrimSpace(url)

	// Try SSH format: git@github.com:owner/repo.git
	if matches := sshURLRegex.FindStringSubmatch(url); matches != nil {
		return &RepoRef{Owner: matches[1], Repo: matches[2]}, nil
	}

	// Try HTTPS format: https://github.com/owner/repo.git
	if matches := httpsURLRegex.FindStringSubmatch(url); matches != nil {
		return &RepoRef{Owner: matches[1], Repo: matches[2]}, nil
	}

	return nil, errors.New("invalid GitHub remote URL")
}

// PRInfo contains information about a GitHub pull request.
type PRInfo struct {
	HeadRef   string // Branch name
	HeadOwner string // Owner of the repository containing the branch
	HeadRepo  string // Repository name containing the branch
	IsFork    bool   // True if PR is from a fork
}

// ghPRResponse represents the JSON response from `gh pr view --json`.
type ghPRResponse struct {
	HeadRefName         string `json:"headRefName"`
	HeadRepository      ghRepo `json:"headRepository"`
	HeadRepositoryOwner ghUser `json:"headRepositoryOwner"`
}

type ghRepo struct {
	Name string `json:"name"`
}

type ghUser struct {
	Login string `json:"login"`
}

// parsePRInfoJSON parses the JSON output from `gh pr view --json`.
func parsePRInfoJSON(data []byte, baseOwner string) (*PRInfo, error) {
	var resp ghPRResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse gh output: %w", err)
	}

	if resp.HeadRefName == "" {
		return nil, errors.New("missing headRefName in gh output")
	}

	info := &PRInfo{
		HeadRef:   resp.HeadRefName,
		HeadOwner: resp.HeadRepositoryOwner.Login,
		HeadRepo:  resp.HeadRepository.Name,
	}

	// Determine if this is a fork PR by comparing head owner to base owner
	info.IsFork = info.HeadOwner != baseOwner

	return info, nil
}

// FetchPRInfo fetches PR information using the gh CLI.
func FetchPRInfo(owner, repo string, number int) (*PRInfo, error) {
	args := []string{
		"pr", "view", strconv.Itoa(number),
		"--repo", fmt.Sprintf("%s/%s", owner, repo),
		"--json", "headRefName,headRepository,headRepositoryOwner",
	}

	cmd := exec.Command("gh", args...) //nolint:gosec // Args are constructed from validated input
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if strings.Contains(stderrStr, "Could not resolve") {
			return nil, fmt.Errorf("PR #%d not found in %s/%s", number, owner, repo)
		}
		if stderrStr != "" {
			return nil, fmt.Errorf("gh failed: %s", stderrStr)
		}
		return nil, fmt.Errorf("gh failed: %w", err)
	}

	return parsePRInfoJSON(stdout.Bytes(), owner)
}

// CheckGhAvailable checks if gh CLI is installed and authenticated.
func CheckGhAvailable() error {
	// Check if gh is installed
	if _, err := exec.LookPath("gh"); err != nil {
		return errors.New("gh CLI not found. Install from https://cli.github.com")
	}

	// Check if gh is authenticated
	cmd := exec.Command("gh", "auth", "status")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return errors.New("gh not authenticated. Run 'gh auth login' first")
	}

	return nil
}

// GetRepoCloneURL returns the clone URL for a repository, respecting user's protocol preference.
// Uses gh CLI to get the URL with the user's configured protocol (SSH or HTTPS).
func GetRepoCloneURL(owner, repo string) (string, error) {
	repoSpec := fmt.Sprintf("%s/%s", owner, repo)
	cmd := exec.Command("gh", "repo", "view", repoSpec, "--json", "url", "-q", ".url") //nolint:gosec // Args are constructed from validated input

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if stderrStr != "" {
			return "", fmt.Errorf("failed to get repo URL: %s", stderrStr)
		}
		return "", fmt.Errorf("failed to get repo URL: %w", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}
