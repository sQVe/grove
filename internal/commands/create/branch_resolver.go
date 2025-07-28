package create

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/utils"
)

type branchResolver struct {
	executor git.GitExecutor
	logger   *logger.Logger
}

func NewBranchResolver(executor git.GitExecutor) BranchResolver {
	return &branchResolver{
		executor: executor,
		logger:   logger.WithComponent("branch_resolver"),
	}
}

func (r *branchResolver) ResolveBranch(name, base string, createIfMissing bool) (*BranchInfo, error) {
	if err := validateBranchName(name); err != nil {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: fmt.Sprintf("invalid branch name: %s", err.Error()),
			Cause:   err,
			Context: map[string]interface{}{
				"branch": name,
			},
			Operation: "branch_validation",
		}
	}

	r.logger.DebugOperation("resolving branch", "name", name, "base", base, "create_if_missing", createIfMissing)

	localExists, err := r.branchExistsLocally(name)
	if err != nil {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: "failed to check local branch existence",
			Cause:   err,
			Context: map[string]interface{}{
				"branch": name,
			},
			Operation: "local_branch_check",
		}
	}

	if localExists {
		r.logger.Debug("branch exists locally", "branch", name)
		return &BranchInfo{
			Name:   name,
			Exists: true,
		}, nil
	}

	remoteInfo, err := r.checkRemoteBranch(name)
	if err != nil {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: "failed to check remote branch",
			Cause:   err,
			Context: map[string]interface{}{
				"branch": name,
			},
			Operation: "remote_branch_check",
		}
	}

	if remoteInfo != nil {
		r.logger.Debug("branch exists on remote", "branch", name, "remote", remoteInfo.RemoteName)
		return remoteInfo, nil
	}

	if !createIfMissing {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: fmt.Sprintf("branch '%s' does not exist", name),
			Context: map[string]interface{}{
				"branch":           name,
				"suggest_creation": true,
			},
			Operation: "branch_resolution",
		}
	}

	// For worktree creation, return BranchInfo with Exists=false to let
	// the worktree creator handle branch creation atomically via 'git worktree add -b'
	r.logger.Debug("branch will be created during worktree creation", "branch", name, "base", base)
	return &BranchInfo{
		Name:   name,
		Exists: false,
	}, nil
}

func (r *branchResolver) ResolveURL(url string) (*URLBranchInfo, error) {
	if err := validateURL(url); err != nil {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeUnsupportedURL,
			Message: fmt.Sprintf("invalid URL format: %s", err.Error()),
			Cause:   err,
			Context: map[string]interface{}{
				"url": url,
			},
			Operation: "url_validation",
		}
	}

	r.logger.DebugOperation("resolving URL", "url", url)

	gitURLInfo, err := utils.ParseGitPlatformURL(url)
	if err != nil {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeUnsupportedURL,
			Message: fmt.Sprintf("failed to parse URL: %s", url),
			Cause:   err,
			Context: map[string]interface{}{
				"url": url,
			},
			Operation: "url_parsing",
		}
	}

	urlBranchInfo := &URLBranchInfo{
		RepoURL:  gitURLInfo.RepoURL,
		Platform: gitURLInfo.Platform,
	}

	if gitURLInfo.BranchName != "" {
		urlBranchInfo.BranchName = gitURLInfo.BranchName
	}

	if gitURLInfo.PRNumber != "" {
		urlBranchInfo.PRNumber = gitURLInfo.PRNumber
		// For PRs from external repositories, we need to add the source repo as a remote
		// to enable fetching the PR branch for local checkout operations.
		urlBranchInfo.RequiresRemote = r.requiresRemoteSetup(gitURLInfo.RepoURL)
	}

	r.logger.Debug("resolved URL", "repo", urlBranchInfo.RepoURL, "branch", urlBranchInfo.BranchName, "platform", urlBranchInfo.Platform)
	return urlBranchInfo, nil
}

func (r *branchResolver) ResolveRemoteBranch(remoteBranch string) (*BranchInfo, error) {
	r.logger.DebugOperation("resolving remote branch", "remote_branch", remoteBranch)

	parts := strings.SplitN(remoteBranch, "/", 2)
	if len(parts) != 2 {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: fmt.Sprintf("invalid remote branch format: %s", remoteBranch),
			Context: map[string]interface{}{
				"remote_branch": remoteBranch,
				"expected":      "remote/branch",
			},
			Operation: "remote_branch_parsing",
		}
	}

	remoteName := parts[0]
	branchName := parts[1]

	if !r.remoteExists(remoteName) {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: fmt.Sprintf("remote '%s' not found", remoteName),
			Context: map[string]interface{}{
				"remote":        remoteName,
				"remote_branch": remoteBranch,
			},
			Operation: "remote_validation",
		}
	}

	// Fetch from remote to ensure we have latest branch information.
	err := r.fetchRemote(remoteName)
	if err != nil {
		// Log fetch failures at debug level to avoid cluttering user output
		r.logger.DebugOperation("failed to fetch from remote - branch info may be stale", "remote", remoteName, "error", err)
		// Continue anyway as branch might still exist locally, but warn user about potential staleness.
	}

	remoteExists, err := r.remoteBranchExists(remoteName, branchName)
	if err != nil {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: "failed to check remote branch existence",
			Cause:   err,
			Context: map[string]interface{}{
				"remote": remoteName,
				"branch": branchName,
			},
			Operation: "remote_branch_existence_check",
		}
	}

	if !remoteExists {
		return nil, &errors.GroveError{
			Code:    errors.ErrCodeGitOperation,
			Message: fmt.Sprintf("branch '%s' not found on remote '%s'", branchName, remoteName),
			Context: map[string]interface{}{
				"branch":        branchName,
				"remote":        remoteName,
				"remote_branch": remoteBranch,
			},
			Operation: "remote_branch_validation",
		}
	}

	r.logger.Debug("resolved remote branch", "branch", branchName, "remote", remoteName)
	return &BranchInfo{
		Name:           branchName,
		Exists:         false,
		IsRemote:       true,
		TrackingBranch: remoteBranch,
		RemoteName:     remoteName,
	}, nil
}

func (r *branchResolver) branchExistsLocally(branchName string) (bool, error) {
	output, err := r.executor.ExecuteQuiet("branch", "-a", "--list")
	if err != nil {
		return false, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		line = strings.TrimPrefix(line, "* ")
		line = strings.TrimPrefix(line, "+ ")
		line = strings.TrimSpace(line)
		if line == branchName && !strings.HasPrefix(line, "remotes/") {
			return true, nil
		}
	}
	return false, nil
}

func (r *branchResolver) checkRemoteBranch(branchName string) (*BranchInfo, error) {
	output, err := r.executor.ExecuteQuiet("branch", "-r", "--list", "*/"+branchName)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "/", 2)
		if len(parts) == 2 && parts[1] == branchName {
			return &BranchInfo{
				Name:           branchName,
				Exists:         false,
				IsRemote:       true,
				TrackingBranch: line,
				RemoteName:     parts[0],
			}, nil
		}
	}

	return nil, nil
}

func (r *branchResolver) requiresRemoteSetup(repoURL string) bool {
	output, err := r.executor.ExecuteQuiet("remote", "-v")
	if err != nil {
		return true // Assume we need setup if we can't check.
	}

	return !strings.Contains(output, repoURL)
}

func (r *branchResolver) RemoteExists(remoteName string) bool {
	return r.remoteExists(remoteName)
}

func (r *branchResolver) remoteExists(remoteName string) bool {
	output, err := r.executor.ExecuteQuiet("remote")
	if err != nil {
		return false
	}

	remotes := strings.Split(strings.TrimSpace(output), "\n")
	for _, remote := range remotes {
		if strings.TrimSpace(remote) == remoteName {
			return true
		}
	}
	return false
}

func (r *branchResolver) fetchRemote(remoteName string) error {
	_, err := r.executor.Execute("fetch", remoteName)
	return err
}

func (r *branchResolver) remoteBranchExists(remoteName, branchName string) (bool, error) {
	remoteBranch := remoteName + "/" + branchName
	output, err := r.executor.ExecuteQuiet("branch", "-r", "--list", remoteBranch)
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(output) != "", nil
}

func validateBranchName(name string) error {
	if name == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	if strings.ContainsAny(name, " \t\n\r\f") {
		return fmt.Errorf("branch name contains whitespace characters")
	}

	if strings.ContainsAny(name, "~^:\\") {
		return fmt.Errorf("branch name contains invalid characters (~^:\\)")
	}

	if strings.HasPrefix(name, "-") || strings.HasSuffix(name, "/") {
		return fmt.Errorf("branch name cannot start with '-' or end with '/'")
	}

	if strings.Contains(name, "..") || strings.Contains(name, "@{") {
		return fmt.Errorf("branch name contains invalid sequences (.., @{)")
	}

	return nil
}

func validateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	urlPattern := regexp.MustCompile(`^https?://[\w.-]+(/.*)?$`)
	if !urlPattern.MatchString(url) {
		return fmt.Errorf("URL must be a valid HTTP/HTTPS URL")
	}

	return nil
}
