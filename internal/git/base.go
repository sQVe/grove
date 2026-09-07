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
			// A repository without commits has no branch to base on yet, which is not a degraded base.
			if unborn, headErr := isHeadDangling(bareDir); headErr != nil || !unborn {
				warnWorktreeBase(bareDir, headRef, "default branch unavailable")
			}
			return headRef, nil
		}
	}
	var fetchErr error
	if hasOrigin, _ := RemoteExists(bareDir, "origin"); fetch && hasOrigin {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "git", "fetch", "--no-tags", "--refmap=", "origin", "+refs/heads/"+branch+":refs/remotes/origin/"+branch) //nolint:gosec // Branch resolved from git
		cmd.Dir = bareDir
		cmd.WaitDelay = time.Second
		fetchErr = runGitCommand(cmd, true)
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
