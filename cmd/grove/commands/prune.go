package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

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

// pruneType describes why a worktree is a prune candidate
type pruneType string

const (
	pruneGone  pruneType = "gone"
	pruneStale pruneType = "stale"
)

// pruneCandidate represents a worktree that could be pruned
type pruneCandidate struct {
	info      *git.WorktreeInfo
	reason    skipReason
	pruneType pruneType
	staleAge  string // Human-readable age for stale worktrees
}

// NewPruneCmd creates the prune command
func NewPruneCmd() *cobra.Command {
	var commit bool
	var force bool
	var stale string

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove worktrees with deleted upstream branches",
		Long:  `Remove worktrees whose upstream branches have been deleted (marked as "gone").`,
		Args:  cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If --stale was passed but no value given, use configured default
			if cmd.Flags().Changed("stale") && stale == "" {
				stale = config.GetStaleThreshold()
			}
			return runPrune(commit, force, stale)
		},
	}

	cmd.Flags().BoolVar(&commit, "commit", false, "Actually remove worktrees (default is dry-run)")
	cmd.Flags().BoolVar(&force, "force", false, "Remove even if dirty, locked, or has unpushed commits")
	cmd.Flags().StringVar(&stale, "stale", "", fmt.Sprintf("Also include worktrees with no commits in duration (e.g., 30d, 2w, 6m; default: %s)", config.GetStaleThreshold()))
	cmd.Flags().BoolP("help", "h", false, "Help for prune")

	return cmd
}

func runPrune(commit, force bool, stale string) error {
	// Parse stale threshold if provided
	var staleCutoff int64
	if stale != "" {
		duration, err := parseDuration(stale)
		if err != nil {
			return err
		}
		staleCutoff = time.Now().Add(-duration).Unix()
	}

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

	// Find prune candidates
	var candidates []pruneCandidate
	for _, info := range infos {
		// Check for gone upstream
		if info.Gone {
			reason := determineSkipReason(info, cwd, force)
			candidates = append(candidates, pruneCandidate{
				info:      info,
				reason:    reason,
				pruneType: pruneGone,
			})
			continue // Don't double-count as stale
		}

		// Check for stale (only if --stale flag was passed)
		if staleCutoff > 0 && info.LastCommitTime > 0 && info.LastCommitTime < staleCutoff {
			reason := determineSkipReason(info, cwd, force)
			candidates = append(candidates, pruneCandidate{
				info:      info,
				reason:    reason,
				pruneType: pruneStale,
				staleAge:  formatAge(info.LastCommitTime),
			})
		}
	}

	// Output results
	if commit {
		return executePrune(bareDir, candidates, force)
	}
	return displayDryRun(candidates, stale != "")
}

func determineSkipReason(info *git.WorktreeInfo, cwd string, force bool) skipReason {
	// Current worktree is always protected (also from subdirectories)
	if cwd == info.Path || strings.HasPrefix(cwd, info.Path+string(filepath.Separator)) {
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

func displayDryRun(candidates []pruneCandidate, includesStale bool) error {
	if len(candidates) == 0 {
		logger.Info("No worktrees to prune.")
		return nil
	}

	plain := config.IsPlain()

	fmt.Println()
	if includesStale {
		logger.Info("Worktrees to remove:")
	} else {
		logger.Info("Worktrees to remove (upstream deleted):")
	}
	fmt.Println()

	hasSkipped := false
	for _, candidate := range candidates {
		label := formatCandidateLabel(candidate, plain)
		switch candidate.reason {
		case skipCurrent:
			fmt.Printf("  %s  %s\n", styles.Render(&styles.Worktree, candidate.info.Branch), styles.Render(&styles.Dimmed, "(current)"))
		case skipNone:
			fmt.Printf("  %s  %s\n", styles.Render(&styles.Worktree, candidate.info.Branch), label)
		default:
			hasSkipped = true
			fmt.Printf("  %s  %s\n", styles.Render(&styles.Worktree, candidate.info.Branch), label)
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

func formatCandidateLabel(candidate pruneCandidate, plain bool) string {
	switch candidate.pruneType {
	case pruneGone:
		status := formatWorktreeStatus(candidate.info, plain)
		if plain {
			return fmt.Sprintf("[gone] %s", status)
		}
		return fmt.Sprintf("%s %s", styles.Render(&styles.Dimmed, "[gone]"), status)
	case pruneStale:
		status := formatWorktreeStatus(candidate.info, plain)
		if plain {
			return fmt.Sprintf("[stale] (%s) %s", candidate.staleAge, status)
		}
		return fmt.Sprintf("%s (%s) %s", styles.Render(&styles.Dimmed, "[stale]"), candidate.staleAge, status)
	default:
		return formatWorktreeStatus(candidate.info, plain)
	}
}

func executePrune(bareDir string, candidates []pruneCandidate, force bool) error {
	if len(candidates) == 0 {
		logger.Info("No worktrees to remove.")
		return nil
	}

	fmt.Println()
	logger.Info("Removing worktrees:")
	fmt.Println()

	removed := 0
	skipped := 0

	for _, candidate := range candidates {
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

// parseDuration parses human-friendly durations like "30d", "2w", "6m"
func parseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("duration cannot be empty")
	}

	s = strings.ToLower(s)
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration: %s", s)
	}

	unit := s[len(s)-1]
	numStr := s[:len(s)-1]

	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("invalid duration number: %s", s)
	}

	if num <= 0 {
		return 0, fmt.Errorf("duration must be positive: %s", s)
	}

	switch unit {
	case 'd':
		return time.Duration(num) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(num) * 30 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %c (use d, w, or m)", unit)
	}
}

// formatAge returns a human-readable string describing how long ago a timestamp was
func formatAge(timestamp int64) string {
	if timestamp == 0 {
		return ""
	}

	age := time.Since(time.Unix(timestamp, 0))
	days := int(age.Hours() / 24)

	switch {
	case days == 0:
		return "today"
	case days == 1:
		return "yesterday"
	case days < 7:
		return fmt.Sprintf("%d days ago", days)
	case days < 14:
		return "1 week ago"
	case days < 30:
		weeks := days / 7
		return fmt.Sprintf("%d weeks ago", weeks)
	case days < 60:
		return "1 month ago"
	case days < 365:
		months := days / 30
		return fmt.Sprintf("%d months ago", months)
	case days < 730:
		return "1 year ago"
	default:
		years := days / 365
		return fmt.Sprintf("%d years ago", years)
	}
}
