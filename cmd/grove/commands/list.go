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
	var filter string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all worktrees with status",
		Long: `Show all worktrees with status and sync state.

Examples:
  grove list                  # Show all worktrees
  grove list --fast           # Skip remote sync checks
  grove list --filter dirty   # Show only dirty worktrees
  grove list --verbose        # Include paths and upstreams`,
		Args: cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runList(fast, jsonOutput, verbose, filter)
		},
	}

	cmd.Flags().BoolVar(&fast, "fast", false, "Skip sync status checks")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show paths and upstream names")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter by status: dirty,ahead,behind,gone,locked (comma-separated)")
	cmd.Flags().BoolP("help", "h", false, "Help for list")

	_ = cmd.RegisterFlagCompletionFunc("filter", completeFilterValues)

	return cmd
}

func runList(fast, jsonOutput, verbose bool, filter string) error {
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

	// Apply filter if specified
	infos = filterWorktrees(infos, filter)

	// Determine current worktree path (also works from subdirectories)
	currentPath := ""
	for _, info := range infos {
		if cwd == info.Path || strings.HasPrefix(cwd, info.Path+string(filepath.Separator)) {
			currentPath = info.Path
			break
		}
	}

	if jsonOutput {
		return outputJSON(infos, currentPath)
	}

	return outputTable(infos, currentPath, fast, verbose)
}

type worktreeJSON struct {
	Name       string `json:"name"`
	Branch     string `json:"branch,omitempty"`
	Path       string `json:"path"`
	Current    bool   `json:"current"`
	Detached   bool   `json:"detached,omitempty"`
	Upstream   string `json:"upstream,omitempty"`
	Dirty      bool   `json:"dirty,omitempty"`
	Ahead      int    `json:"ahead,omitempty"`
	Behind     int    `json:"behind,omitempty"`
	Gone       bool   `json:"gone,omitempty"`
	NoUpstream bool   `json:"no_upstream,omitempty"`
	Locked     bool   `json:"locked,omitempty"`
	LockReason string `json:"lock_reason,omitempty"`
}

func outputJSON(infos []*git.WorktreeInfo, currentPath string) error {
	output := []worktreeJSON{}
	for _, info := range infos {
		entry := worktreeJSON{
			Name:       filepath.Base(info.Path),
			Path:       info.Path,
			Current:    info.Path == currentPath,
			Detached:   info.Detached,
			Upstream:   info.Upstream,
			Dirty:      info.Dirty,
			Ahead:      info.Ahead,
			Behind:     info.Behind,
			Gone:       info.Gone,
			NoUpstream: info.NoUpstream,
			Locked:     info.Locked,
			LockReason: info.LockReason,
		}
		if !info.Detached {
			entry.Branch = info.Branch
		}
		output = append(output, entry)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(output)
}

func outputTable(infos []*git.WorktreeInfo, currentPath string, fast, verbose bool) error {
	// Sort: current worktree first, then alphabetically by worktree name
	sort.SliceStable(infos, func(i, j int) bool {
		iCurrent := infos[i].Path == currentPath
		jCurrent := infos[j].Path == currentPath
		if iCurrent != jCurrent {
			return iCurrent // Current worktree comes first
		}
		// Sort by worktree name (directory basename)
		return filepath.Base(infos[i].Path) < filepath.Base(infos[j].Path)
	})

	// Calculate max widths for padding
	maxNameLen := 0
	maxBranchLen := 0
	for _, info := range infos {
		nameLen := len(filepath.Base(info.Path))
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}

		branchLen := len(info.Branch) + 2 // brackets add 2 chars
		if info.Detached {
			branchLen = 10 // "(detached)" is 10 chars
		}
		if branchLen > maxBranchLen {
			maxBranchLen = branchLen
		}
	}

	for _, info := range infos {
		isCurrent := info.Path == currentPath

		// In fast mode, we don't have sync status - create a copy with zeroed sync info
		displayInfo := info
		if fast {
			displayInfo = &git.WorktreeInfo{
				Branch:     info.Branch,
				Path:       info.Path,
				Upstream:   info.Upstream,
				Locked:     info.Locked,
				LockReason: info.LockReason,
				Detached:   info.Detached,
				NoUpstream: true, // This prevents showing sync status
			}
		}

		// Print the worktree row using the formatter
		fmt.Println(formatter.WorktreeRow(displayInfo, isCurrent, maxNameLen, maxBranchLen))

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

func filterWorktrees(infos []*git.WorktreeInfo, filter string) []*git.WorktreeInfo {
	filters := parseFilters(filter)
	if len(filters) == 0 {
		return infos
	}

	var filtered []*git.WorktreeInfo
	for _, info := range infos {
		if matchesAnyFilter(info, filters) {
			filtered = append(filtered, info)
		}
	}
	return filtered
}

func parseFilters(filter string) []string {
	if filter == "" {
		return nil
	}

	parts := strings.Split(filter, ",")
	var filters []string
	for _, p := range parts {
		p = strings.TrimSpace(strings.ToLower(p))
		if p != "" {
			filters = append(filters, p)
		}
	}
	return filters
}

func matchesAnyFilter(info *git.WorktreeInfo, filters []string) bool {
	for _, f := range filters {
		switch f {
		case "dirty":
			if info.Dirty {
				return true
			}
		case "ahead":
			if info.Ahead > 0 {
				return true
			}
		case "behind":
			if info.Behind > 0 {
				return true
			}
		case "gone":
			if info.Gone {
				return true
			}
		case "locked":
			if info.Locked {
				return true
			}
		}
	}
	return false
}

func completeFilterValues(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	validFilters := []string{"dirty", "ahead", "behind", "gone", "locked"}

	parts := strings.Split(toComplete, ",")
	lastPart := parts[len(parts)-1]
	prefix := ""
	if len(parts) > 1 {
		prefix = strings.Join(parts[:len(parts)-1], ",") + ","
	}

	selected := make(map[string]bool)
	for _, p := range parts[:len(parts)-1] {
		selected[strings.TrimSpace(p)] = true
	}

	var completions []string
	for _, f := range validFilters {
		if !selected[f] && strings.HasPrefix(f, lastPart) {
			completions = append(completions, prefix+f)
		}
	}
	return completions, cobra.ShellCompDirectiveNoFileComp | cobra.ShellCompDirectiveNoSpace
}
