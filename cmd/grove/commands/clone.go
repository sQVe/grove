package commands

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/github"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

// ErrCloneRepositoryName is returned when a repository name cannot be derived from a clone source.
var ErrCloneRepositoryName = errors.New("cannot derive a repository name from URL; pass a directory argument")

// resolveTargetDirectory resolves the target directory from command arguments
func resolveTargetDirectory(args []string, argIndex int) (string, error) {
	if len(args) <= argIndex {
		return os.Getwd()
	}
	return filepath.Abs(args[argIndex])
}

func repositoryName(source string) (string, error) {
	name := filepath.Base(strings.TrimSuffix(strings.TrimRight(source, "/"), ".git"))
	if name == "" || name == "." || name == ".." {
		return "", ErrCloneRepositoryName
	}
	return name, nil
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
			if cmd.Flags().Changed("branches") && github.IsPRURL(args[0]) {
				return errors.New("--branches cannot be combined with a PR URL")
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
			isPRURL := github.IsPRURL(urlOrPR)

			targetDir, err := resolveTargetDirectory(args, 1)
			if err != nil {
				return err
			}
			if len(args) == 1 {
				if isPRURL {
					targetDir = ""
				} else {
					name, err := repositoryName(urlOrPR)
					if err != nil {
						return err
					}
					targetDir = filepath.Join(targetDir, name)
				}
			}

			// Check if this is a PR URL (full URL only, not #N format)
			if isPRURL {
				return runCloneFromPR(urlOrPR, targetDir, verbose, shallow)
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

func runCloneFromPR(prURL, targetDir string, verbose, shallow bool) error {
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

	repoSpec := fmt.Sprintf("%s/%s", ref.Owner, ref.Repo)
	cloneFn := func(bareDir string) error {
		return cloneWithGh(repoSpec, bareDir, verbose, shallow)
	}
	createWorktree := func(bareDir string) ([]string, error) {
		spin := logger.StartSpinner(fmt.Sprintf("Fetching PR #%d...", ref.Number))
		prInfo, err := github.FetchPRInfo(ref.Owner, ref.Repo, ref.Number)
		if err != nil {
			spin.StopWithError("Failed to fetch PR info")
			return nil, err
		}
		spin.Stop()

		worktreePath := filepath.Join(workspaceDir, fmt.Sprintf("pr-%d", ref.Number))
		err = checkoutPR(bareDir, worktreePath, ref, prInfo, !verbose, false, false)
		return []string{worktreePath}, err
	}
	if err := workspace.CloneAndInitializeWithCloner(cloneFn, workspaceDir, "", verbose, shallow, createWorktree); err != nil {
		return err
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

	if err := workspace.CloneAndInitializeWithCloner(cloneFn, targetDir, branches, verbose, shallow, nil); err != nil {
		return err
	}

	logger.Success("Cloned repository to %s", styles.RenderPath(targetDir))
	return nil
}

// cloneWithGh clones a repository using the gh CLI, which respects the user's protocol preference.
func cloneWithGh(repoSpec, bareDir string, verbose, shallow bool) error {
	spin := logger.StartSpinner(fmt.Sprintf("Cloning %s...", repoSpec))

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
		spin.StopWithError("Clone failed")
		errStr := strings.TrimSpace(stderr.String())
		if errStr != "" {
			return fmt.Errorf("clone failed: %s", errStr)
		}

		return fmt.Errorf("clone failed: %w", err)
	}
	spin.Stop()

	return nil
}
