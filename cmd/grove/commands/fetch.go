package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/formatter"
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

type fetchChangeJSON struct {
	Remote      string `json:"remote"`
	RefName     string `json:"ref"`
	Type        string `json:"type"`
	OldHash     string `json:"old_hash,omitempty"`
	NewHash     string `json:"new_hash,omitempty"`
	CommitCount int    `json:"commit_count,omitempty"`
}

type fetchErrorJSON struct {
	Remote  string `json:"remote"`
	Message string `json:"message"`
}

type fetchResultJSON struct {
	Changes []fetchChangeJSON `json:"changes"`
	Errors  []fetchErrorJSON  `json:"errors,omitempty"`
}

func NewFetchCmd() *cobra.Command {
	var jsonOutput bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "fetch",
		Short: "Fetch all remotes and show changes",
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runFetch(jsonOutput, verbose)
		},
	}

	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show commit hash details")

	return cmd
}

func runFetch(jsonOutput, verbose bool) error {
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
		logger.Info("No remotes configured")
		return nil
	}

	var results []remoteResult
	for _, remote := range remotes {
		result := fetchRemoteWithRetry(bareDir, remote)
		results = append(results, result)
	}

	return outputFetchResults(bareDir, results, jsonOutput, verbose)
}

func fetchRemoteWithRetry(bareDir, remote string) remoteResult {
	result := remoteResult{Remote: remote}

	refsBefore, err := git.GetRemoteRefs(bareDir, remote)
	if err != nil {
		result.Error = err
		return result
	}

	spinner := logger.StartSpinner(fmt.Sprintf("Fetching %s...", remote))

	err = git.FetchRemote(bareDir, remote)
	if err != nil {
		logger.Debug("Fetch failed for %s, retrying: %v", remote, err)
		err = git.FetchRemote(bareDir, remote)
	}
	spinner.Stop()

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

func outputFetchJSON(bareDir string, results []remoteResult) error {
	output := fetchResultJSON{
		Changes: make([]fetchChangeJSON, 0),
	}

	for _, result := range results {
		if result.Error != nil {
			output.Errors = append(output.Errors, fetchErrorJSON{
				Remote:  result.Remote,
				Message: result.Error.Error(),
			})
			continue
		}

		for _, change := range result.Changes {
			jsonChange := fetchChangeJSON{
				Remote:  result.Remote,
				RefName: stripRefPrefix(change.RefName, result.Remote),
				Type:    change.Type.String(),
				OldHash: change.OldHash,
				NewHash: change.NewHash,
			}

			if change.Type == git.Updated {
				jsonChange.CommitCount = getCommitCount(bareDir, change.OldHash, change.NewHash)
			}

			output.Changes = append(output.Changes, jsonChange)
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(output); err != nil {
		return err
	}

	if len(output.Errors) > 0 {
		return fmt.Errorf("failed to fetch %d remote(s)", len(output.Errors))
	}

	return nil
}

func outputFetchResults(bareDir string, results []remoteResult, jsonOutput, verbose bool) error {
	if jsonOutput {
		return outputFetchJSON(bareDir, results)
	}

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
			if verbose {
				printRefChangeVerbose(bareDir, result.Remote, change)
			} else {
				printRefChange(bareDir, result.Remote, change)
			}
		}
	}

	if !hasChanges && len(errors) == 0 {
		logger.Success("All remotes up to date")
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

func printRefChangeVerbose(bareDir, remote string, change git.RefChange) {
	printRefChange(bareDir, remote, change)

	prefix := formatter.SubItemPrefix()

	switch change.Type {
	case git.New:
		if change.NewHash != "" {
			shortHash := change.NewHash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			if config.IsPlain() {
				fmt.Printf("    %s at: %s\n", prefix, shortHash)
			} else {
				fmt.Printf("    %s at: %s\n",
					styles.Render(&styles.Dimmed, prefix),
					styles.Render(&styles.Dimmed, shortHash))
			}
		}

	case git.Updated:
		if change.OldHash != "" {
			shortHash := change.OldHash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			if config.IsPlain() {
				fmt.Printf("    %s from: %s\n", prefix, shortHash)
			} else {
				fmt.Printf("    %s from: %s\n",
					styles.Render(&styles.Dimmed, prefix),
					styles.Render(&styles.Dimmed, shortHash))
			}
		}
		if change.NewHash != "" {
			shortHash := change.NewHash
			if len(shortHash) > 7 {
				shortHash = shortHash[:7]
			}
			if config.IsPlain() {
				fmt.Printf("    %s to:   %s\n", prefix, shortHash)
			} else {
				fmt.Printf("    %s to:   %s\n",
					styles.Render(&styles.Dimmed, prefix),
					styles.Render(&styles.Dimmed, shortHash))
			}
		}
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
