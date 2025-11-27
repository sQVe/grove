package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/styles"
	"github.com/sqve/grove/internal/workspace"
)

// skipReason describes why a worktree would be skipped during prune
type skipReason string

const (
	skipNone     skipReason = ""
	skipCurrent  skipReason = "current worktree"
	skipDirty    skipReason = "dirty, use --force"
	skipLocked   skipReason = "locked, use --force"
	skipUnpushed skipReason = "unpushed commits, use --force"
)

// pruneCandidate represents a worktree that could be pruned
type pruneCandidate struct {
	info   *git.WorktreeInfo
	reason skipReason
}

// NewPruneCmd creates the prune command
func NewPruneCmd() *cobra.Command {
	var commit bool
	var force bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove worktrees with deleted upstream branches",
		Long:  `Remove worktrees whose upstream branches have been deleted (marked as "gone").`,
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(commit, force)
		},
	}

	cmd.Flags().BoolVar(&commit, "commit", false, "Actually remove worktrees (default is dry-run)")
	cmd.Flags().BoolVar(&force, "force", false, "Remove even if dirty, locked, or has unpushed commits")
	cmd.Flags().BoolP("help", "h", false, "Help for prune")

	return cmd
}

func runPrune(commit, force bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Fetch and prune remote refs
	logger.Info("Fetching remote changes...")
	if err := git.FetchPrune(bareDir); err != nil {
		// Non-fatal: network issues shouldn't block prune of already-known gone branches
		logger.Warning("Failed to fetch: %v", err)
	}

	// Get all worktrees with info
	infos, err := git.ListWorktreesWithInfo(bareDir, false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Find worktrees with deleted upstream
	var goneWorktrees []pruneCandidate
	for _, info := range infos {
		if info.Gone {
			reason := determineSkipReason(info, cwd, force)
			goneWorktrees = append(goneWorktrees, pruneCandidate{info: info, reason: reason})
		}
	}

	// Output results
	if commit {
		return executePrune(bareDir, goneWorktrees, force)
	}
	return displayDryRun(goneWorktrees)
}

func determineSkipReason(info *git.WorktreeInfo, cwd string, force bool) skipReason {
	// Current worktree is always protected
	if info.Path == cwd {
		return skipCurrent
	}

	// Skip reasons that can be overridden with --force
	if !force {
		if info.Dirty {
			return skipDirty
		}
		if info.Locked {
			return skipLocked
		}
		if info.Ahead > 0 {
			return skipUnpushed
		}
	}

	return skipNone
}

func displayDryRun(goneWorktrees []pruneCandidate) error {
	if len(goneWorktrees) == 0 {
		logger.Info("No stale worktrees found.")
		return nil
	}

	plain := config.IsPlain()

	fmt.Println()
	logger.Info("Stale worktrees (upstream deleted):")
	fmt.Println()

	hasSkipped := false
	for _, candidate := range goneWorktrees {
		status := formatWorktreeStatus(candidate.info, plain)
		switch candidate.reason {
		case skipCurrent:
			fmt.Printf("  %s  %s\n", styles.Render(&styles.Worktree, candidate.info.Branch), styles.Render(&styles.Dimmed, "(current)"))
		case skipNone:
			fmt.Printf("  %s  %s\n", styles.Render(&styles.Worktree, candidate.info.Branch), status)
		default:
			hasSkipped = true
			fmt.Printf("  %s  %s\n", styles.Render(&styles.Worktree, candidate.info.Branch), status)
		}
	}

	fmt.Println()
	if hasSkipped {
		logger.Info("Run with --commit to remove. Use --force to include dirty/locked/ahead.")
	} else {
		logger.Info("Run with --commit to remove.")
	}

	return nil
}

func executePrune(bareDir string, goneWorktrees []pruneCandidate, force bool) error {
	if len(goneWorktrees) == 0 {
		logger.Info("No stale worktrees to remove.")
		return nil
	}

	fmt.Println()
	logger.Info("Removing stale worktrees:")
	fmt.Println()

	removed := 0
	skipped := 0

	for _, candidate := range goneWorktrees {
		if candidate.reason != skipNone {
			skipped++
			if config.IsPlain() {
				fmt.Printf("  - Skipped %s (%s)\n", candidate.info.Branch, candidate.reason)
			} else {
				fmt.Printf("  %s Skipped %s (%s)\n",
					styles.Render(&styles.Warning, "⊘"),
					styles.Render(&styles.Worktree, candidate.info.Branch),
					candidate.reason)
			}
			continue
		}

		// Actually remove the worktree
		if err := git.RemoveWorktree(bareDir, candidate.info.Path, force); err != nil {
			skipped++
			if config.IsPlain() {
				fmt.Printf("  - Failed %s: %v\n", candidate.info.Branch, err)
			} else {
				fmt.Printf("  %s Failed %s: %v\n",
					styles.Render(&styles.Error, "✗"),
					styles.Render(&styles.Worktree, candidate.info.Branch),
					err)
			}
			continue
		}

		removed++
		if config.IsPlain() {
			fmt.Printf("  + Removed %s\n", candidate.info.Branch)
		} else {
			fmt.Printf("  %s Removed %s\n",
				styles.Render(&styles.Success, "✓"),
				styles.Render(&styles.Worktree, candidate.info.Branch))
		}
	}

	fmt.Println()
	if skipped > 0 {
		logger.Info("Removed %d worktree(s), skipped %d.", removed, skipped)
	} else {
		logger.Success("Removed %d worktree(s).", removed)
	}

	return nil
}

func formatWorktreeStatus(info *git.WorktreeInfo, plain bool) string {
	if info.Locked {
		if plain {
			return "[locked]"
		}
		return styles.Render(&styles.Warning, "[locked]")
	}
	if info.Dirty {
		if plain {
			return "[dirty]"
		}
		return styles.Render(&styles.Warning, "[dirty]")
	}
	if info.Ahead > 0 {
		if plain {
			return fmt.Sprintf("[ahead %d]", info.Ahead)
		}
		return styles.Render(&styles.Warning, fmt.Sprintf("[ahead %d]", info.Ahead))
	}
	if plain {
		return "[clean]"
	}
	return styles.Render(&styles.Success, "[clean]")
}
