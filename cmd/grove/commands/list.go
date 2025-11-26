package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
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

	// Get current worktree to mark it
	currentBranch := ""
	if git.IsWorktree(cwd) {
		currentBranch, _ = git.GetCurrentBranch(cwd)
	}

	// Get worktree info
	infos, err := git.ListWorktreesWithInfo(bareDir, fast)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
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
}

func outputJSON(infos []*git.WorktreeInfo, currentBranch string) error {
	var output []worktreeJSON
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
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func formatSyncStatus(info *git.WorktreeInfo, plain bool) string {
	if info.Gone {
		if plain {
			return "gone"
		}
		return styles.Render(&styles.Error, "×")
	}
	if info.NoUpstream {
		return ""
	}
	if info.Ahead == 0 && info.Behind == 0 {
		return "="
	}

	var parts []string
	if info.Ahead > 0 {
		if plain {
			parts = append(parts, fmt.Sprintf("+%d", info.Ahead))
		} else {
			parts = append(parts, styles.Render(&styles.Success, fmt.Sprintf("↑%d", info.Ahead)))
		}
	}
	if info.Behind > 0 {
		if plain {
			parts = append(parts, fmt.Sprintf("-%d", info.Behind))
		} else {
			parts = append(parts, styles.Render(&styles.Warning, fmt.Sprintf("↓%d", info.Behind)))
		}
	}
	return strings.Join(parts, "")
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

	for _, info := range infos {
		isCurrent := info.Branch == currentBranch

		status := ""
		syncStatus := ""
		if !fast {
			if info.Dirty {
				status = "[dirty]"
			} else {
				status = "[clean]"
			}
			syncStatus = formatSyncStatus(info, config.IsPlain())
		}

		logger.WorktreeListItem(info.Branch, isCurrent, status, syncStatus)

		if verbose {
			logger.ListSubItem("%s", info.Path)
			if info.Upstream != "" {
				logger.ListSubItem("upstream: %s", info.Upstream)
			}
		}
	}
	return nil
}
