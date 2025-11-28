package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
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
		Long:  `Display detailed status information for the current worktree.`,
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runStatus(verbose, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&verbose, "verbose", false, "Show full sectioned diagnostic output")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	cmd.Flags().BoolP("help", "h", false, "Help for status")

	return cmd
}

func runStatus(verbose, jsonOutput bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Verify we're in a worktree, not the bare repo
	if !git.IsWorktree(cwd) {
		return fmt.Errorf("not inside a worktree (run from a worktree directory)")
	}

	info, err := gatherStatusInfo(cwd, bareDir)
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

func gatherStatusInfo(worktreePath, bareDir string) (*StatusInfo, error) {
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
	if err == nil {
		info.Stashes = stashes
	}

	// Get ongoing operation
	operation, err := git.GetOngoingOperation(worktreePath)
	if err == nil {
		info.Operation = operation
	}

	// Get conflict count
	conflicts, err := git.GetConflictCount(worktreePath)
	if err == nil {
		info.Conflicts = conflicts
	}

	// Check if locked
	worktreeName := filepath.Base(worktreePath)
	info.Locked = git.IsWorktreeLocked(bareDir, worktreeName)

	return info, nil
}

func outputStatusJSON(info *StatusInfo) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(info)
}

func outputStatusDefault(info *StatusInfo) error {
	plain := config.IsPlain()

	// Line 1: Branch and upstream
	line1 := formatBranchLine(info, plain)
	fmt.Println(line1)

	// Line 2: Status details
	line2 := formatStatusLine(info, plain)
	fmt.Println(line2)

	// Line 3: Issues (only if present)
	issues := formatIssuesLine(info, plain)
	if issues != "" {
		fmt.Println(issues)
	}

	return nil
}

func formatBranchLine(info *StatusInfo, plain bool) string {
	var marker, branch, arrow, upstream string

	if plain {
		marker = "*"
		branch = info.Branch
		arrow = "->"
	} else {
		marker = styles.Render(&styles.Success, "●")
		branch = styles.Render(&styles.Worktree, info.Branch)
		arrow = "→"
	}

	if info.NoUpstream {
		upstream = "(no upstream)"
		if !plain {
			upstream = styles.Render(&styles.Dimmed, upstream)
		}
	} else if info.Upstream != "" {
		upstream = info.Upstream
		if !plain {
			upstream = styles.Render(&styles.Dimmed, upstream)
		}
	}

	if upstream != "" {
		return fmt.Sprintf("%s %s %s %s", marker, branch, arrow, upstream)
	}
	return fmt.Sprintf("%s %s", marker, branch)
}

func formatStatusLine(info *StatusInfo, plain bool) string {
	var parts []string

	// Sync status
	syncPart := formatSyncPart(info, plain)
	if syncPart != "" {
		parts = append(parts, syncPart)
	}

	// Dirty state
	if info.Dirty {
		if plain {
			parts = append(parts, "dirty")
		} else {
			parts = append(parts, styles.Render(&styles.Warning, "dirty"))
		}
	} else {
		if plain {
			parts = append(parts, "clean")
		} else {
			parts = append(parts, styles.Render(&styles.Dimmed, "clean"))
		}
	}

	// Stashes
	if info.Stashes > 0 {
		stashText := fmt.Sprintf("%d stashed", info.Stashes)
		if plain {
			parts = append(parts, stashText)
		} else {
			parts = append(parts, styles.Render(&styles.Dimmed, stashText))
		}
	}

	// Ongoing operation
	if info.Operation != "" {
		if plain {
			parts = append(parts, info.Operation)
		} else {
			parts = append(parts, styles.Render(&styles.Warning, info.Operation))
		}
	}

	separator := " · "
	return "  " + strings.Join(parts, separator)
}

func formatSyncPart(info *StatusInfo, plain bool) string {
	if info.Gone {
		if plain {
			return "gone"
		}
		return styles.Render(&styles.Error, "gone")
	}

	if info.NoUpstream {
		return ""
	}

	if info.Ahead == 0 && info.Behind == 0 {
		return "="
	}

	var syncParts []string

	if info.Ahead > 0 {
		if plain {
			syncParts = append(syncParts, fmt.Sprintf("+%d", info.Ahead))
		} else {
			syncParts = append(syncParts, styles.Render(&styles.Success, fmt.Sprintf("↑%d", info.Ahead)))
		}
	}

	if info.Behind > 0 {
		if plain {
			syncParts = append(syncParts, fmt.Sprintf("-%d", info.Behind))
		} else {
			syncParts = append(syncParts, styles.Render(&styles.Warning, fmt.Sprintf("↓%d", info.Behind)))
		}
	}

	syncStatus := strings.Join(syncParts, "")

	// Add description
	switch {
	case info.Ahead > 0 && info.Behind > 0:
		syncStatus += " diverged"
	case info.Ahead > 0:
		syncStatus += " ahead"
	case info.Behind > 0:
		syncStatus += " behind"
	}

	return syncStatus
}

func formatIssuesLine(info *StatusInfo, plain bool) string {
	var issues []string

	if info.Conflicts > 0 {
		conflictText := fmt.Sprintf("%d conflicts", info.Conflicts)
		if plain {
			issues = append(issues, conflictText)
		} else {
			issues = append(issues, styles.Render(&styles.Error, conflictText))
		}
	}

	if info.Locked {
		if plain {
			issues = append(issues, "locked")
		} else {
			issues = append(issues, styles.Render(&styles.Warning, "locked"))
		}
	}

	if info.Detached {
		if plain {
			issues = append(issues, "detached HEAD")
		} else {
			issues = append(issues, styles.Render(&styles.Warning, "detached HEAD"))
		}
	}

	if len(issues) == 0 {
		return ""
	}

	prefix := "  "
	if !plain {
		prefix = "  " + styles.Render(&styles.Warning, "⚠") + " "
	}

	return prefix + strings.Join(issues, " · ")
}

func outputStatusVerbose(info *StatusInfo) error {
	plain := config.IsPlain()

	// Worktree section
	printSection("Worktree", plain)
	printField("Branch", info.Branch, plain)
	printField("Path", info.Path, plain)
	if info.Upstream != "" {
		printField("Upstream", info.Upstream, plain)
	} else if info.NoUpstream {
		printField("Upstream", "(not configured)", plain)
	}

	fmt.Println()

	// Sync Status section
	printSection("Sync Status", plain)
	switch {
	case info.Gone:
		printField("Status", "upstream deleted", plain)
	case info.NoUpstream:
		printField("Status", "no upstream configured", plain)
	default:
		if info.Ahead > 0 {
			printField("Ahead", fmt.Sprintf("%d commits", info.Ahead), plain)
		}
		if info.Behind > 0 {
			printField("Behind", fmt.Sprintf("%d commits", info.Behind), plain)
		}
		if info.Ahead == 0 && info.Behind == 0 {
			printField("Status", "in sync", plain)
		}
	}

	fmt.Println()

	// Working Tree section
	printSection("Working Tree", plain)
	if info.Dirty {
		printField("State", "dirty", plain)
	} else {
		printField("State", "clean", plain)
	}
	if info.Stashes > 0 {
		printField("Stashes", fmt.Sprintf("%d", info.Stashes), plain)
	}

	// Operations section (only if something active)
	if info.Operation != "" || info.Conflicts > 0 || info.Locked || info.Detached {
		fmt.Println()
		printSection("Operations", plain)
		if info.Operation != "" {
			printField("In Progress", info.Operation, plain)
		}
		if info.Conflicts > 0 {
			printField("Conflicts", fmt.Sprintf("%d unresolved", info.Conflicts), plain)
		}
		if info.Locked {
			printField("Locked", "yes", plain)
		}
		if info.Detached {
			printField("Detached", "yes", plain)
		}
	}

	return nil
}

func printSection(name string, plain bool) {
	if plain {
		fmt.Println(name)
	} else {
		fmt.Println(styles.Render(&styles.Info, name))
	}
}

func printField(key, value string, plain bool) {
	if plain {
		fmt.Printf("  %s: %s\n", key, value)
	} else {
		fmt.Printf("  %-10s %s\n", styles.Render(&styles.Dimmed, key+":"), value)
	}
}
