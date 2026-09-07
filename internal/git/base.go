package git

import (
	"context"
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
	if base != "" {
		if exists, _ := RemoteBranchExists(bareDir, "origin", branch); !exists || base == headRef {
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
			// A repository without branches has nothing to base on yet, which is not a degraded base.
			if empty, emptyErr := hasNoBranches(bareDir); emptyErr != nil || !empty {
				warnWorktreeBase(bareDir, headRef, "default branch unavailable")
			}
			return headRef, nil
		}
	}
	var fetchErr error
	if fetch {
		// An unreadable remote list is treated as a remote, so a real failure still warns.
		if hasOrigin, originErr := RemoteExists(bareDir, "origin"); originErr != nil || hasOrigin {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, "git", "fetch", "--no-tags", "--refmap=", "origin", "+refs/heads/"+branch+":refs/remotes/origin/"+branch) //nolint:gosec // Branch resolved from git
			cmd.Dir = bareDir
			cmd.WaitDelay = time.Second
			fetchErr = runGitCommand(cmd, true)
		}
	}
	base = headRef
	if exists, _ := RemoteBranchExists(bareDir, "origin", branch); exists {
		base = "origin/" + branch
	} else if exists, _ := LocalBranchExists(bareDir, branch); exists {
		base = branch
	}
	if fetchErr != nil {
		logger.Debug("Base fetch failed: %v", fetchErr)
		warnWorktreeBase(bareDir, base, "fetch failed")
	}
	return base, nil
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
