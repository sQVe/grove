package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/formatter"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

// StatusInfo contains all status information for a worktree
type StatusInfo struct {
	Branch     string `json:"branch"`
	Path       string `json:"path"`
	Upstream   string `json:"upstream,omitempty"`
	Ahead      int    `json:"ahead"`
	Behind     int    `json:"behind"`
	Dirty      bool   `json:"dirty"`
	Staged     int    `json:"staged,omitempty"`
	Unstaged   int    `json:"unstaged,omitempty"`
	Stashes    int    `json:"stashes"`
	Unpushed   int    `json:"unpushed,omitempty"`
	Operation  string `json:"operation,omitempty"`
	Conflicts  int    `json:"conflicts,omitempty"`
	Locked     bool   `json:"locked"`
	LockReason string `json:"lock_reason,omitempty"`
	Detached   bool   `json:"detached"`
	Gone       bool   `json:"gone"`
	NoUpstream bool   `json:"no_upstream"`
}

// NewStatusCmd creates the status command
func NewStatusCmd() *cobra.Command {
	var verbose bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Show current worktree status",
		Long: `Show detailed status for the current worktree.

Examples:
  grove status            # Show status summary
  grove status --verbose  # Show all sections
  grove status --json     # Output as JSON`,
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(verbose, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show all diagnostic sections")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
	cmd.Flags().BoolP("help", "h", false, "Help for status")

	return cmd
}

func runStatus(verbose, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Verify we're in a grove workspace
	if _, err := workspace.FindBareDir(cwd); err != nil {
		return err
	}

	// Find worktree root (works from subdirectories)
	worktreeRoot, err := git.FindWorktreeRoot(cwd)
	if err != nil {
		return fmt.Errorf("not inside a worktree (run from a worktree directory)")
	}

	info, err := gatherStatusInfo(worktreeRoot)
	if err != nil {
		return err
	}

	if jsonOutput {
		return outputStatusJSON(info)
	}

	if verbose {
		return outputStatusVerbose(info)
	}

	return outputStatusDefault(info)
}

func gatherStatusInfo(worktreePath string) (*StatusInfo, error) {
	info := &StatusInfo{
		Path: worktreePath,
	}

	// Get branch (handles detached HEAD)
	branch, err := git.GetCurrentBranch(worktreePath)
	if err != nil {
		// Check if detached
		detached, detachErr := git.IsDetachedHead(worktreePath)
		if detachErr == nil && detached {
			info.Detached = true
			info.Branch = "(detached)"
		} else {
			return nil, fmt.Errorf("failed to get branch: %w", err)
		}
	} else {
		info.Branch = branch
	}

	// Get sync status
	syncStatus := git.GetSyncStatus(worktreePath)
	info.Upstream = syncStatus.Upstream
	info.Ahead = syncStatus.Ahead
	info.Behind = syncStatus.Behind
	info.Gone = syncStatus.Gone
	info.NoUpstream = syncStatus.NoUpstream

	// Check dirty state
	hasChanges, _, err := git.CheckGitChanges(worktreePath)
	if err != nil {
		return nil, fmt.Errorf("failed to check changes: %w", err)
	}
	info.Dirty = hasChanges

	// Get stash count
	stashes, err := git.GetStashCount(worktreePath)
	if err != nil {
		logger.Debug("Failed to get stash count: %v", err)
	} else {
		info.Stashes = stashes
	}

	// Get ongoing operation
	operation, err := git.GetOngoingOperation(worktreePath)
	if err != nil {
		logger.Debug("Failed to get ongoing operation: %v", err)
	} else {
		info.Operation = operation
	}

	// Get conflict count
	conflicts, err := git.GetConflictCount(worktreePath)
	if err != nil {
		logger.Debug("Failed to get conflict count: %v", err)
	} else {
		info.Conflicts = conflicts
	}

	// Check if locked
	info.Locked = git.IsWorktreeLocked(worktreePath)
	info.LockReason = git.GetWorktreeLockReason(worktreePath)

	return info, nil
}

func outputStatusJSON(info *StatusInfo) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(info)
}

func outputStatusDefault(info *StatusInfo) error {
	// Convert StatusInfo to git.WorktreeInfo for formatter
	wtInfo := &git.WorktreeInfo{
		Branch:     info.Branch,
		Path:       info.Path,
		Upstream:   info.Upstream,
		Ahead:      info.Ahead,
		Behind:     info.Behind,
		Dirty:      info.Dirty,
		Locked:     info.Locked,
		LockReason: info.LockReason,
		Gone:       info.Gone,
		NoUpstream: info.NoUpstream,
		Detached:   info.Detached,
	}

	// Use consistent single-line format (same as list)
	fmt.Println(formatter.WorktreeRow(wtInfo, true, 0, 0))

	return nil
}

func outputStatusVerbose(info *StatusInfo) error {
	// Convert StatusInfo to git.WorktreeInfo for formatter
	wtInfo := &git.WorktreeInfo{
		Branch:     info.Branch,
		Path:       info.Path,
		Upstream:   info.Upstream,
		Ahead:      info.Ahead,
		Behind:     info.Behind,
		Dirty:      info.Dirty,
		Locked:     info.Locked,
		LockReason: info.LockReason,
		Gone:       info.Gone,
		NoUpstream: info.NoUpstream,
		Detached:   info.Detached,
	}

	// Print the worktree row (same format as default)
	fmt.Println(formatter.WorktreeRow(wtInfo, true, 0, 0))

	// Print standard verbose sub-items (path, upstream, lock reason)
	subItems := formatter.VerboseSubItems(wtInfo)
	for _, item := range subItems {
		fmt.Println(item)
	}

	// Print additional status details as sub-items
	prefix := formatter.SubItemPrefix()

	// Stashes
	if info.Stashes > 0 {
		if config.IsPlain() {
			fmt.Printf("    %s stashes: %d\n", prefix, info.Stashes)
		} else {
			fmt.Printf("    %s stashes: %d\n", styles.Render(&styles.Dimmed, prefix), info.Stashes)
		}
	}

	// Ongoing operation
	if info.Operation != "" {
		if config.IsPlain() {
			fmt.Printf("    %s operation: %s\n", prefix, info.Operation)
		} else {
			fmt.Printf("    %s operation: %s\n", styles.Render(&styles.Dimmed, prefix), styles.Render(&styles.Warning, info.Operation))
		}
	}

	// Conflicts
	if info.Conflicts > 0 {
		if config.IsPlain() {
			fmt.Printf("    %s conflicts: %d\n", prefix, info.Conflicts)
		} else {
			fmt.Printf("    %s conflicts: %s\n", styles.Render(&styles.Dimmed, prefix), styles.Render(&styles.Error, fmt.Sprintf("%d", info.Conflicts)))
		}
	}

	// Detached HEAD warning
	if info.Detached {
		if config.IsPlain() {
			fmt.Printf("    %s detached HEAD\n", prefix)
		} else {
			fmt.Printf("    %s %s\n", styles.Render(&styles.Dimmed, prefix), styles.Render(&styles.Warning, "detached HEAD"))
		}
	}

	return nil
}
