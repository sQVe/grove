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
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/github"
	"github.com/sqve/grove/internal/logger"
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
	pruneGone     pruneType = "gone"
	pruneDetached pruneType = "detached"
	pruneStale    pruneType = "stale"
	pruneMerged   pruneType = "merged"
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
	var merged bool
	var detached bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove worktrees with deleted upstream branches",
		Long: `Remove worktrees with deleted upstream branches (marked "gone").

For gone branches, local branches are also deleted after removing the worktree.

Examples:
  grove prune                 # Dry-run: show what would be removed
  grove prune --commit        # Actually remove worktrees
  grove prune --stale 30d     # Include inactive worktrees
  grove prune --merged        # Include merged branches
  grove prune --detached      # Include detached worktrees
  grove prune --force         # Remove even if dirty or locked`,
		Args: cobra.NoArgs,
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// If --stale was passed but no value given, use configured default
			if cmd.Flags().Changed("stale") && stale == "" {
				stale = config.GetStaleThreshold()
			}
			return runPrune(commit, force, stale, merged, detached)
		},
	}

	cmd.Flags().BoolVar(&commit, "commit", false, "Remove worktrees (dry-run without this flag)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Remove even if dirty, locked, or unpushed")
	cmd.Flags().StringVar(&stale, "stale", "", fmt.Sprintf("Include inactive worktrees (e.g., 30d, 2w; default: %s)", config.GetStaleThreshold()))
	cmd.Flags().BoolVar(&merged, "merged", false, "Include worktrees merged into default branch")
	cmd.Flags().BoolVar(&detached, "detached", false, "Include detached worktrees")
	cmd.Flags().BoolP("help", "h", false, "Help for prune")

	_ = cmd.RegisterFlagCompletionFunc("stale", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"7d", "14d", "30d", "2w", "1m"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

func runPrune(commit, force bool, stale string, merged, detached bool) error {
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
	spin := logger.StartSpinner("Fetching remote changes...")
	if err := git.FetchPrune(bareDir); err != nil {
		spin.Stop()
		// Non-fatal: network issues shouldn't block prune of already-known gone branches
		logger.Warning("Failed to fetch: %v", err)
	} else {
		spin.Stop()
	}

	// Get default branch - needed for --merged check AND gone branch deletion
	defaultBranch, defaultBranchErr := git.GetDefaultBranch(bareDir)
	if defaultBranchErr != nil {
		logger.Debug("Could not determine default branch: %v", defaultBranchErr)
		if merged {
			logger.Warning("Could not determine default branch, skipping --merged check")
			merged = false // Disable merged check if we can't determine default branch
		}
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
			continue // Don't double-count as detached, stale, or merged
		}

		// Check for detached (only if --detached flag was passed)
		if detached && info.Detached {
			reason := determineSkipReason(info, cwd, force)
			candidates = append(candidates, pruneCandidate{
				info:      info,
				reason:    reason,
				pruneType: pruneDetached,
			})
			continue // Don't double-count as merged or stale
		}

		// Check for merged (only if --merged flag was passed)
		if merged && info.Branch != "" && info.Branch != defaultBranch {
			isMerged, mergeErr := git.IsBranchMerged(bareDir, info.Branch, defaultBranch)
			if mergeErr == nil && isMerged {
				reason := determineSkipReason(info, cwd, force)
				candidates = append(candidates, pruneCandidate{
					info:      info,
					reason:    reason,
					pruneType: pruneMerged,
				})
				continue // Don't double-count as stale
			}
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
		return executePrune(bareDir, candidates, force, defaultBranch)
	}
	return displayDryRun(candidates)
}

func determineSkipReason(info *git.WorktreeInfo, cwd string, force bool) skipReason {
	// Current worktree is always protected (also from subdirectories)
	if fs.PathsEqual(cwd, info.Path) || fs.PathHasPrefix(cwd, info.Path) {
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

func displayDryRun(candidates []pruneCandidate) error {
	if len(candidates) == 0 {
		logger.Info("No worktrees to prune.")
		return nil
	}

	// Group candidates by whether they can be pruned
	var toPrune []string
	var toSkip []string

	for _, candidate := range candidates {
		label := candidate.info.Branch
		if candidate.pruneType == pruneDetached {
			label = filepath.Base(candidate.info.Path)
		} else if candidate.pruneType == pruneStale && candidate.staleAge != "" {
			label = fmt.Sprintf("%s (%s)", candidate.info.Branch, candidate.staleAge)
		}

		if candidate.reason == skipNone {
			toPrune = append(toPrune, label)
		} else {
			toSkip = append(toSkip, fmt.Sprintf("%s (%s)", label, candidate.reason))
		}
	}

	// Display results
	if len(toPrune) > 0 {
		if len(toPrune) == 1 {
			logger.Info("Would prune 1 worktree:")
		} else {
			logger.Info("Would prune %d worktrees:", len(toPrune))
		}
		for _, item := range toPrune {
			logger.Dimmed("    %s", item)
		}
	}

	if len(toSkip) > 0 {
		if len(toSkip) == 1 {
			logger.Warning("Would skip 1 worktree:")
		} else {
			logger.Warning("Would skip %d worktrees:", len(toSkip))
		}
		for _, item := range toSkip {
			logger.Dimmed("    %s", item)
		}
	}

	if len(toPrune) > 0 {
		fmt.Println()
		if len(toSkip) > 0 {
			logger.Info("Run with --commit to remove. Use --force to include skipped.")
		} else {
			logger.Info("Run with --commit to remove.")
		}
	}

	return nil
}

func executePrune(bareDir string, candidates []pruneCandidate, force bool, defaultBranch string) error {
	if len(candidates) == 0 {
		logger.Info("No worktrees to remove.")
		return nil
	}

	// Fetch merged PR branches from GitHub (for multi-commit squash detection)
	var mergedViaPR map[string]bool
	if mergedPRs, err := github.GetMergedPRBranches(bareDir); err != nil {
		logger.Debug("GitHub PR check unavailable: %v", err)
	} else {
		mergedViaPR = mergedPRs
	}

	// Process all candidates
	var pruned []string
	var skipped []string
	var failed []string
	var deletedBranches int
	var keptBranches []string

	for _, candidate := range candidates {
		label := candidate.info.Branch
		if candidate.pruneType == pruneDetached {
			label = filepath.Base(candidate.info.Path)
		}

		if candidate.reason != skipNone {
			skipped = append(skipped, fmt.Sprintf("%s (%s)", label, candidate.reason))
			continue
		}

		// Actually remove the worktree
		if err := git.RemoveWorktree(bareDir, candidate.info.Path, force); err != nil {
			failed = append(failed, fmt.Sprintf("%s: %v", label, err))
			continue
		}

		pruned = append(pruned, label)

		// Delete local branch for gone worktrees (not detached)
		if candidate.pruneType == pruneGone && !candidate.info.Detached {
			forceDelete := false

			// Check if merged into default branch before deleting
			// (upstream is gone, so git -d can't verify merge status)
			if defaultBranch != "" {
				merged, mergeErr := git.IsBranchMerged(bareDir, candidate.info.Branch, defaultBranch)
				if mergeErr != nil {
					logger.Debug("Could not verify merge status for %s: %v", candidate.info.Branch, mergeErr)
				} else if merged {
					logger.Debug("Branch %s is squash-merged into %s, using force delete", candidate.info.Branch, defaultBranch)
					forceDelete = true
				}
			}

			// Fallback: check GitHub PR merge (for multi-commit squash merges)
			if !forceDelete && mergedViaPR[candidate.info.Branch] {
				logger.Debug("Branch %s detected as merged via GitHub PR", candidate.info.Branch)
				forceDelete = true
			}

			if err := git.DeleteBranch(bareDir, candidate.info.Branch, forceDelete); err != nil {
				if strings.Contains(err.Error(), "not fully merged") {
					keptBranches = append(keptBranches, fmt.Sprintf("%s (unmerged commits)", candidate.info.Branch))
				} else {
					keptBranches = append(keptBranches, fmt.Sprintf("%s (%v)", candidate.info.Branch, err))
				}
			} else {
				deletedBranches++
			}
		}
	}

	// Display results
	if len(pruned) > 0 {
		if len(pruned) == 1 {
			logger.Success("Pruned 1 worktree:")
		} else {
			logger.Success("Pruned %d worktrees:", len(pruned))
		}
		for _, item := range pruned {
			logger.Dimmed("    %s", item)
		}
		if deletedBranches > 0 {
			if deletedBranches == 1 {
				logger.Dimmed("    ↳ deleted 1 local branch")
			} else {
				logger.Dimmed("    ↳ deleted %d local branches", deletedBranches)
			}
		}
		if len(keptBranches) > 0 {
			if len(keptBranches) == 1 {
				logger.Dimmed("    ↳ kept 1 local branch: %s", keptBranches[0])
			} else {
				logger.Dimmed("    ↳ kept %d local branches:", len(keptBranches))
				for _, branch := range keptBranches {
					logger.Dimmed("        %s", branch)
				}
			}
		}
	}

	if len(skipped) > 0 {
		if len(skipped) == 1 {
			logger.Warning("Skipped 1 worktree:")
		} else {
			logger.Warning("Skipped %d worktrees:", len(skipped))
		}
		for _, item := range skipped {
			logger.Dimmed("    %s", item)
		}
	}

	if len(failed) > 0 {
		if len(failed) == 1 {
			logger.Error("Failed to remove 1 worktree:")
		} else {
			logger.Error("Failed to remove %d worktrees:", len(failed))
		}
		for _, item := range failed {
			logger.Dimmed("    %s", item)
		}
	}

	return nil
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
