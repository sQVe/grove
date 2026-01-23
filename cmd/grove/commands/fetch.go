package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

type remoteResult struct {
	Remote  string
	Changes []git.RefChange
	Error   error
}

func NewFetchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch all remotes and show changes",
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch()
		},
	}

	return cmd
}

func runFetch() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	remotes, err := git.ListRemotes(bareDir)
	if err != nil {
		return fmt.Errorf("failed to list remotes: %w", err)
	}

	if len(remotes) == 0 {
		fmt.Println("No remotes configured")
		return nil
	}

	var results []remoteResult
	for _, remote := range remotes {
		result := fetchRemoteWithRetry(bareDir, remote)
		results = append(results, result)
	}

	return outputFetchResults(bareDir, results)
}

func fetchRemoteWithRetry(bareDir, remote string) remoteResult {
	result := remoteResult{Remote: remote}

	refsBefore, err := git.GetRemoteRefs(bareDir, remote)
	if err != nil {
		result.Error = err
		return result
	}

	stopSpinner := logger.StartSpinner(fmt.Sprintf("Fetching %s...", remote))

	err = git.FetchRemote(bareDir, remote)
	if err != nil {
		logger.Debug("Fetch failed for %s, retrying: %v", remote, err)
		err = git.FetchRemote(bareDir, remote)
	}
	stopSpinner()

	if err != nil {
		result.Error = err
		return result
	}

	refsAfter, err := git.GetRemoteRefs(bareDir, remote)
	if err != nil {
		result.Error = err
		return result
	}

	result.Changes = git.DetectRefChanges(refsBefore, refsAfter)
	return result
}

func outputFetchResults(bareDir string, results []remoteResult) error {
	var errors []error
	hasChanges := false

	for _, result := range results {
		if result.Error != nil {
			errors = append(errors, fmt.Errorf("%s: %w", result.Remote, result.Error))
			continue
		}

		if len(result.Changes) == 0 {
			continue
		}

		hasChanges = true
		fmt.Printf("%s:\n", result.Remote)
		for _, change := range result.Changes {
			printRefChange(bareDir, result.Remote, change)
		}
	}

	if !hasChanges && len(errors) == 0 {
		fmt.Println("All remotes up to date")
	}

	if len(errors) > 0 {
		for _, err := range errors {
			logger.Error("%v", err)
		}
		return fmt.Errorf("failed to fetch %d remote(s)", len(errors))
	}

	return nil
}

func printRefChange(bareDir, remote string, change git.RefChange) {
	shortName := stripRefPrefix(change.RefName, remote)

	switch change.Type {
	case git.New:
		symbol := styles.Render(&styles.Success, "+")
		fmt.Printf("  %s %s\n", symbol, shortName)

	case git.Updated:
		symbol := styles.Render(&styles.Warning, "*")
		count := getCommitCount(bareDir, change.OldHash, change.NewHash)
		switch {
		case count > 0:
			fmt.Printf("  %s %s (+%d commits)\n", symbol, shortName, count)
		case count < 0:
			fmt.Printf("  %s %s (%d commits)\n", symbol, shortName, count)
		default:
			fmt.Printf("  %s %s (force-pushed)\n", symbol, shortName)
		}

	case git.Pruned:
		symbol := styles.Render(&styles.Dimmed, "-")
		hint := styles.Render(&styles.Dimmed, "(deleted on remote)")
		fmt.Printf("  %s %s %s\n", symbol, shortName, hint)
	}
}

func stripRefPrefix(refName, remote string) string {
	prefix := fmt.Sprintf("refs/remotes/%s/", remote)
	return strings.TrimPrefix(refName, prefix)
}

func getCommitCount(bareDir, oldHash, newHash string) int {
	forwardCount := git.CountCommits(bareDir, oldHash, newHash)
	if forwardCount > 0 {
		return forwardCount
	}

	backwardCount := git.CountCommits(bareDir, newHash, oldHash)
	if backwardCount > 0 {
		return -backwardCount
	}

	return 0
}
