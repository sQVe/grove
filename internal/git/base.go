package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/sqve/grove/internal/logger"
)

const headRef = "HEAD"

// ResolveWorktreeBase selects an explicit start point, optionally refreshing origin.
func ResolveWorktreeBase(bareDir, base string, fetch bool) (string, error) {
	branch := strings.TrimPrefix(base, "origin/")
	qualified := strings.HasPrefix(base, "origin/")
	explicit := base != ""
	if explicit {
		localOnly := false
		if !fetch && !qualified {
			remote, err := RemoteBranchExists(bareDir, "origin", branch)
			if err != nil {
				return "", fmt.Errorf("failed to check origin/%s: %w", branch, err)
			}
			localOnly = !remote
		}
		// Fetch an explicit base before calling it missing, unless fetching is disabled.
		if localOnly || base == headRef {
			// An empty repository has no ref to verify; CreateWorktree starts an orphan branch.
			if base == headRef {
				empty, emptyErr := hasNoBranches(bareDir)
				if emptyErr != nil {
					return "", emptyErr
				}
				if empty {
					return headRef, nil
				}
			}
			exists, err := BranchExists(bareDir, base)
			if err != nil {
				return "", err
			}
			if !exists {
				return "", fmt.Errorf("base branch %q does not exist", base)
			}
			return base, nil
		}
	} else {
		var err error
		branch, err = GetDefaultBranch(bareDir)
		if err != nil {
			return resolveHeadBase(bareDir)
		}
	}
	var fetchErr error
	if fetch {
		// An unreadable remote list is treated as a remote, so a real failure still warns.
		if hasOrigin, originErr := RemoteExists(bareDir, "origin"); originErr != nil || hasOrigin {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "git", "fetch", "--no-tags", "--refmap=", "origin", "+refs/heads/"+branch+":refs/remotes/origin/"+branch) //nolint:gosec // Fixed executable; branch is part of one refspec argument, not shell input.
			cmd.Dir = bareDir
			cmd.WaitDelay = time.Second
			fetchErr = runGitCommand(cmd, true)
		}
	}
	base = headRef
	remote, remoteErr := RemoteBranchExists(bareDir, "origin", branch)
	if remoteErr != nil {
		return "", fmt.Errorf("failed to check origin/%s: %w", branch, remoteErr)
	}
	local, localErr := LocalBranchExists(bareDir, branch)
	if localErr != nil {
		return "", fmt.Errorf("failed to check branch %s: %w", branch, localErr)
	}
	switch {
	case remote:
		base = "origin/" + branch
	case qualified:
		return "", fmt.Errorf("base branch \"origin/%s\" does not exist", branch)
	case local:
		base = branch
	case explicit:
		return "", fmt.Errorf("base branch %q does not exist", branch)
	}
	if fetchErr != nil {
		logger.Debug("Base fetch failed: %v", fetchErr)
		// An unpushed local branch is usable without a counterpart on origin.
		if !local || remote || !strings.HasSuffix(fetchErr.Error(), "fatal: couldn't find remote ref refs/heads/"+branch) {
			warnWorktreeBase(bareDir, base, "fetch failed")
		}
	}
	return base, nil
}

// resolveHeadBase falls back to the bare repository's HEAD once no default branch resolves.
func resolveHeadBase(bareDir string) (string, error) {
	dangling, err := isHeadDangling(bareDir)
	if err != nil {
		return "", err
	}
	if !dangling {
		warnWorktreeBase(bareDir, headRef, "default branch unavailable")
		return headRef, nil
	}
	// A repository without branches has nothing to base on yet, which is not a degraded base.
	empty, err := hasNoBranches(bareDir)
	if err != nil {
		return "", err
	}
	if empty {
		return headRef, nil
	}
	return "", errors.New("cannot resolve a base branch: HEAD points at a branch that no longer exists; pass --base <branch>")
}

// hasNoBranches reports whether the repository holds no local or remote-tracking branches.
func hasNoBranches(bareDir string) (bool, error) {
	cmd, cancel := GitCommand("git", "for-each-ref", "--count=1", "--format=%(refname)", "refs/heads", "refs/remotes")
	defer cancel()
	cmd.Dir = bareDir
	refs, err := executeWithOutput(cmd)
	if err != nil {
		return false, err
	}
	return refs == "", nil
}

func warnWorktreeBase(bareDir, base, reason string) {
	cmd, cancel := GitCommand("git", "log", "-1", "--format=%cr", base, "--")
	defer cancel()
	cmd.Dir = bareDir
	age, err := executeWithOutput(cmd)
	if err != nil || age == "" {
		age = "unknown time"
	}
	logger.Warning("basing on %s from %s - %s", base, age, reason)
}
