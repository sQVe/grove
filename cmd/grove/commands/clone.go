package commands

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/github"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

// resolveTargetDirectory resolves the target directory from command arguments
func resolveTargetDirectory(args []string, argIndex int) (string, error) {
	if len(args) <= argIndex {
		return os.Getwd()
	}
	return filepath.Abs(args[argIndex])
}

func NewCloneCmd() *cobra.Command {
	var branches string
	var verbose bool
	var shallow bool

	cloneCmd := &cobra.Command{
		Use:   "clone <url|PR-URL> [directory]",
		Short: "Clone a repository and create a grove workspace",
		Long: `Clone a repository into a grove workspace.

Clones from a repository URL or GitHub pull request URL.
From a PR URL, creates a worktree for the PR's branch.

Examples:
  grove clone https://github.com/owner/repo                  # Clone repo
  grove clone https://github.com/owner/repo my-project       # Clone to directory
  grove clone https://github.com/owner/repo/pull/123         # Clone and checkout PR`,
		Args: cobra.RangeArgs(1, 2),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("branches") && len(args) == 0 {
				return fmt.Errorf("--branches requires a repository URL to be specified")
			}
			return nil
		},
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			return nil, cobra.ShellCompDirectiveFilterDirs
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("branches") && (branches == "" || branches == `""`) {
				return fmt.Errorf("no branches specified")
			}

			urlOrPR := args[0]

			targetDir, err := resolveTargetDirectory(args, 1)
			if err != nil {
				return err
			}

			// Check if this is a PR URL (full URL only, not #N format)
			if github.IsPRURL(urlOrPR) {
				return runCloneFromPR(urlOrPR, targetDir, verbose)
			}

			// Check if this is a GitHub URL and gh is available - use gh for protocol preference
			if github.IsGitHubURL(urlOrPR) {
				if err := github.CheckGhAvailable(); err == nil {
					ref, err := github.ParseRepoURL(urlOrPR)
					if err != nil {
						return err
					}

					return runCloneFromGitHub(ref.Owner, ref.Repo, targetDir, branches, verbose, shallow)
				}

				logger.Debug("gh CLI not available, using direct clone (may not respect protocol preference)")
			}

			// Regular clone (non-GitHub URLs or GitHub without gh)
			if err := workspace.CloneAndInitialize(urlOrPR, targetDir, branches, verbose, shallow); err != nil {
				return err
			}

			logger.Success("Cloned repository to %s", styles.RenderPath(targetDir))
			return nil
		},
	}
	cloneCmd.Flags().StringVar(&branches, "branches", "", "Comma-separated list of branches to create worktrees for")
	cloneCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show git output")
	cloneCmd.Flags().BoolVar(&shallow, "shallow", false, "Create a shallow clone (depth=1)")
	cloneCmd.Flags().BoolP("help", "h", false, "Help for clone")

	_ = cloneCmd.RegisterFlagCompletionFunc("branches", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return nil, cobra.ShellCompDirectiveNoFileComp
	})

	return cloneCmd
}

func runCloneFromPR(prURL, targetDir string, verbose bool) error {
	// Check gh is available
	if err := github.CheckGhAvailable(); err != nil {
		return err
	}

	// Parse PR URL
	ref, err := github.ParsePRReference(prURL)
	if err != nil {
		return err
	}

	// PR URL always has owner/repo
	if ref.Owner == "" || ref.Repo == "" {
		return fmt.Errorf("PR URL must include owner and repo")
	}

	// Determine workspace directory
	workspaceDir := targetDir
	if workspaceDir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		workspaceDir = filepath.Join(cwd, ref.Repo)
	}

	// Validate and create workspace directory
	if err := workspace.ValidateAndPrepareDirectory(workspaceDir); err != nil {
		return err
	}

	bareDir := filepath.Join(workspaceDir, ".bare")
	gitFile := filepath.Join(workspaceDir, ".git")

	// Cleanup function for failure cases
	cleanup := func(worktreePath string) {
		if err := os.Remove(gitFile); err != nil && !os.IsNotExist(err) {
			logger.Warning("Failed to remove .git file during cleanup: %v", err)
		}
		if err := fs.RemoveAll(bareDir); err != nil {
			logger.Warning("Failed to remove .bare during cleanup: %v", err)
		}
		if worktreePath != "" {
			if err := fs.RemoveAll(worktreePath); err != nil {
				logger.Warning("Failed to remove worktree during cleanup: %v", err)
			}
		}
	}

	// Clone using gh repo clone (respects user's protocol preference)
	logger.Info("Cloning %s/%s...", ref.Owner, ref.Repo)
	repoSpec := fmt.Sprintf("%s/%s", ref.Owner, ref.Repo)

	args := []string{"repo", "clone", repoSpec, bareDir, "--", "--bare"}
	cmd := exec.Command("gh", args...) //nolint:gosec // Args are constructed from validated input
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		cleanup("")
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return fmt.Errorf("clone failed: %s", errStr)
		}
		return fmt.Errorf("clone failed: %w", err)
	}

	// Create .git file pointing to .bare
	if err := os.WriteFile(gitFile, []byte("gitdir: .bare\n"), 0o644); err != nil { //nolint:gosec // .git file needs standard permissions
		cleanup("")
		return fmt.Errorf("failed to create .git file: %w", err)
	}

	// Fetch PR info
	logger.Info("Fetching PR #%d...", ref.Number)
	prInfo, err := github.FetchPRInfo(ref.Owner, ref.Repo, ref.Number)
	if err != nil {
		cleanup("")
		return err
	}

	branch := prInfo.HeadRef
	dirName := fmt.Sprintf("pr-%d", ref.Number)
	worktreePath := filepath.Join(workspaceDir, dirName)

	// Handle fork PRs
	if prInfo.IsFork {
		remoteName := fmt.Sprintf("pr-%d-%s", ref.Number, prInfo.HeadOwner)
		remoteURL, err := github.GetRepoCloneURL(prInfo.HeadOwner, prInfo.HeadRepo)
		if err != nil {
			cleanup("")
			return fmt.Errorf("failed to get fork URL: %w", err)
		}

		logger.Info("Adding remote %s for fork...", remoteName)
		if err := git.AddRemote(bareDir, remoteName, remoteURL); err != nil {
			cleanup("")
			return fmt.Errorf("failed to add fork remote: %w", err)
		}

		logger.Info("Fetching branch %s from fork...", branch)
		if err := git.FetchBranch(bareDir, remoteName, branch); err != nil {
			cleanup("")
			return fmt.Errorf("failed to fetch fork branch: %w", err)
		}

		// Create worktree tracking the fork's branch
		trackingRef := fmt.Sprintf("%s/%s", remoteName, branch)
		if err := git.CreateWorktree(bareDir, worktreePath, trackingRef, !verbose); err != nil {
			cleanup(worktreePath)
			return git.HintGitTooOld(fmt.Errorf("failed to create worktree: %w", err))
		}
	} else {
		// Same-repo PR: fetch and create worktree
		logger.Info("Fetching branch %s...", branch)
		if err := git.FetchBranch(bareDir, "origin", branch); err != nil {
			cleanup("")
			return fmt.Errorf("failed to fetch branch: %w", err)
		}

		if err := git.CreateWorktree(bareDir, worktreePath, branch, !verbose); err != nil {
			cleanup(worktreePath)
			return git.HintGitTooOld(fmt.Errorf("failed to create worktree: %w", err))
		}
	}

	logger.Success("Cloned repository to %s", styles.RenderPath(workspaceDir))
	logger.ListSubItem("fetched PR #%d", ref.Number)
	return nil
}

func runCloneFromGitHub(owner, repo, targetDir, branches string, verbose, shallow bool) error {
	repoSpec := fmt.Sprintf("%s/%s", owner, repo)

	cloneFn := func(bareDir string) error {
		return cloneWithGh(repoSpec, bareDir, verbose, shallow)
	}

	if err := workspace.CloneAndInitializeWithCloner(cloneFn, targetDir, branches, verbose); err != nil {
		return err
	}

	logger.Success("Cloned repository to %s", styles.RenderPath(targetDir))
	return nil
}

// cloneWithGh clones a repository using the gh CLI, which respects the user's protocol preference.
func cloneWithGh(repoSpec, bareDir string, verbose, shallow bool) error {
	logger.Info("Cloning %s...", repoSpec)

	args := []string{"repo", "clone", repoSpec, bareDir, "--", "--bare"}
	if shallow {
		args = append(args, "--depth", "1")
	}

	cmd := exec.Command("gh", args...) //nolint:gosec // Args are constructed from validated input
	var stderr bytes.Buffer
	if verbose {
		cmd.Stdout = os.Stdout
		cmd.Stderr = io.MultiWriter(os.Stderr, &stderr)
	} else {
		cmd.Stderr = &stderr
	}

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return fmt.Errorf("clone failed: %s", errStr)
		}

		return fmt.Errorf("clone failed: %w", err)
	}

	return nil
}
