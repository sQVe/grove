package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/formatter"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/workspace"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	var fast bool
	var jsonOutput bool
	var verbose bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all worktrees with status",
		Long:  `Show all worktrees in the grove workspace with their status and sync information.`,
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(fast, jsonOutput, verbose)
		},
	}

	cmd.Flags().BoolVar(&fast, "fast", false, "Skip sync status for faster output")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show extra details (paths, upstream names)")
	cmd.Flags().BoolP("help", "h", false, "Help for list")

	return cmd
}

func runList(fast, jsonOutput, verbose bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Get worktree info
	infos, err := git.ListWorktreesWithInfo(bareDir, fast)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Determine current branch (also works from subdirectories)
	currentBranch := ""
	for _, info := range infos {
		if cwd == info.Path || strings.HasPrefix(cwd, info.Path+string(filepath.Separator)) {
			currentBranch = info.Branch
			break
		}
	}

	if jsonOutput {
		return outputJSON(infos, currentBranch)
	}

	return outputTable(infos, currentBranch, fast, verbose)
}

type worktreeJSON struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Current    bool   `json:"current"`
	Upstream   string `json:"upstream,omitempty"`
	Dirty      bool   `json:"dirty,omitempty"`
	Ahead      int    `json:"ahead,omitempty"`
	Behind     int    `json:"behind,omitempty"`
	Gone       bool   `json:"gone,omitempty"`
	NoUpstream bool   `json:"no_upstream,omitempty"`
	Locked     bool   `json:"locked,omitempty"`
	LockReason string `json:"lock_reason,omitempty"`
}

func outputJSON(infos []*git.WorktreeInfo, currentBranch string) error {
	output := []worktreeJSON{}
	for _, info := range infos {
		output = append(output, worktreeJSON{
			Name:       info.Branch,
			Path:       info.Path,
			Current:    info.Branch == currentBranch,
			Upstream:   info.Upstream,
			Dirty:      info.Dirty,
			Ahead:      info.Ahead,
			Behind:     info.Behind,
			Gone:       info.Gone,
			NoUpstream: info.NoUpstream,
			Locked:     info.Locked,
			LockReason: info.LockReason,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputTable(infos []*git.WorktreeInfo, currentBranch string, fast, verbose bool) error {
	// Sort: current branch first, then alphabetically
	sort.SliceStable(infos, func(i, j int) bool {
		iCurrent := infos[i].Branch == currentBranch
		jCurrent := infos[j].Branch == currentBranch
		if iCurrent != jCurrent {
			return iCurrent // Current branch comes first
		}
		return false // Keep alphabetical order from ListWorktreesWithInfo
	})

	// Calculate max branch name length for padding
	maxNameLen := 0
	for _, info := range infos {
		if len(info.Branch) > maxNameLen {
			maxNameLen = len(info.Branch)
		}
	}

	for _, info := range infos {
		isCurrent := info.Branch == currentBranch

		// In fast mode, we don't have sync status - create a copy with zeroed sync info
		displayInfo := info
		if fast {
			displayInfo = &git.WorktreeInfo{
				Branch:     info.Branch,
				Path:       info.Path,
				Upstream:   info.Upstream,
				Locked:     info.Locked,
				LockReason: info.LockReason,
				NoUpstream: true, // This prevents showing sync status
			}
		}

		// Print the worktree row using the formatter
		fmt.Println(formatter.WorktreeRow(displayInfo, isCurrent, maxNameLen))

		// Print verbose sub-items
		if verbose {
			subItems := formatter.VerboseSubItems(displayInfo)
			for _, item := range subItems {
				fmt.Println(item)
			}
		}
	}
	return nil
}
