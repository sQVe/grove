package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/formatter"
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
	prunePrunable pruneType = "prunable"
)

// pruneCandidate represents a worktree that could be pruned
type pruneCandidate struct {
	info      *git.WorktreeInfo
	reason    skipReason
	pruneType pruneType
	staleAge  string // Human-readable age for stale worktrees
}

// mergedDefaultTarget is the --merged value Cobra substitutes when the flag is
// passed without one. The spaces make it impossible to collide with a branch name.
const mergedDefaultTarget = "<default branch>"

// NewPruneCmd creates the prune command
func NewPruneCmd() *cobra.Command {
	var commit bool
	var force bool
	var stale string
	var merged string
	var detached bool
	var jsonOutput bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove worktrees with deleted upstream branches",
		Long: `Remove worktrees with deleted upstream branches (marked "gone").

For gone branches, local branches are also deleted after removing the worktree.

Examples:
  grove prune                 # Dry-run: show what would be removed
  grove prune --commit        # Actually remove worktrees
  grove prune --stale 30d     # Include inactive worktrees
  grove prune --merged        # Include branches merged into the default branch
  grove prune --merged=dev    # Include branches merged into dev (the = is required)
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

			// Bare --merged becomes the sentinel, so an empty value here means
			// the user passed --merged= and a script expanded nothing into it.
			if cmd.Flags().Changed("merged") && merged == "" {
				return fmt.Errorf("--merged requires a branch name")
			}
			return runPrune(commit, force, stale, merged, detached, jsonOutput)
		},
	}

	cmd.Flags().BoolVar(&commit, "commit", false, "Remove worktrees (dry-run without this flag)")
	cmd.Flags().BoolVarP(&force, "force", "f", false, "Remove even if dirty, locked, or unpushed")
	cmd.Flags().StringVar(&stale, "stale", "", fmt.Sprintf("Include inactive worktrees (e.g., 30d, 2w; default: %s)", config.GetStaleThreshold()))
	cmd.Flags().StringVar(&merged, "merged", "", "Include worktrees merged into a branch (default: the default branch; use --merged=<branch>)")
	cmd.Flags().Lookup("merged").NoOptDefVal = mergedDefaultTarget
	cmd.Flags().BoolVar(&detached, "detached", false, "Include detached worktrees")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output dry run as JSON")
	cmd.Flags().BoolP("help", "h", false, "Help for prune")

	_ = cmd.RegisterFlagCompletionFunc("stale", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"7d", "14d", "30d", "2w", "1m"}, cobra.ShellCompDirectiveNoFileComp
	})

	return cmd
}

// mergedTarget names the branch that --merged selects against. ref is what git
// resolves for the merge check; branch is the plain name a worktree is checked
// out on, which is what the candidate exclusion compares.
type mergedTarget struct {
	ref    string
	branch string
}

// resolveMergedTarget maps the --merged flag value to its target, preferring a
// local branch and falling back to a remote one so a branch that exists only on
// the remote still resolves. The ref is fully qualified, since a short name like
// origin/develop is ambiguous when a local branch of that name also exists.
// An empty result means the flag was not passed.
func resolveMergedTarget(bareDir, merged, defaultBranch string) (mergedTarget, error) {
	if merged == "" {
		return mergedTarget{}, nil
	}

	// The default branch is derived rather than typed, so it keeps its old
	// behavior: an unresolvable one yields no candidates instead of an error.
	if merged == mergedDefaultTarget {
		return mergedTarget{ref: defaultBranch, branch: defaultBranch}, nil
	}

	// A local branch wins, the way git prefers refs/heads over refs/remotes. A
	// branch named feature/foo must not be read as remote "feature", branch "foo".
	local, err := git.LocalBranchExists(bareDir, merged)
	if err != nil {
		return mergedTarget{}, fmt.Errorf("failed to check branch %q: %w", merged, err)
	}
	if local {
		return mergedTarget{ref: "refs/heads/" + merged, branch: merged}, nil
	}

	// Accept the remote-qualified form users reach for, e.g. origin/develop.
	if remote, branch, found := strings.Cut(merged, "/"); found {
		exists, cutErr := git.RemoteBranchExists(bareDir, remote, branch)
		if cutErr != nil {
			return mergedTarget{}, fmt.Errorf("failed to check branch %q: %w", merged, cutErr)
		}
		if exists {
			return mergedTarget{ref: "refs/remotes/" + merged, branch: branch}, nil
		}
	}

	remotes, err := git.ListRemotes(bareDir)
	if err != nil {
		return mergedTarget{}, fmt.Errorf("failed to list remotes: %w", err)
	}
	for _, remote := range remotes {
		exists, remoteErr := git.RemoteBranchExists(bareDir, remote, merged)
		if remoteErr != nil {
			return mergedTarget{}, fmt.Errorf("failed to check branch %q: %w", merged, remoteErr)
		}
		if exists {
			return mergedTarget{ref: "refs/remotes/" + remote + "/" + merged, branch: merged}, nil
		}
	}

	return mergedTarget{}, fmt.Errorf("branch not found: %s", merged)
}

func runPrune(commit, force bool, stale, merged string, detached, jsonOutput bool) error {
	if jsonOutput && commit {
		return fmt.Errorf("--json applies to dry run only")
	}

	// Parse stale threshold if provided
	var staleCutoff int64
	if stale != "" {
		duration, err := config.ParseDuration(stale)
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
		if merged == mergedDefaultTarget {
			logger.Warning("Could not determine default branch, skipping --merged check")
			merged = "" // Disable merged check if we can't determine default branch
		}
	}

	target, err := resolveMergedTarget(bareDir, merged, defaultBranch)
	if err != nil {
		return err
	}

	// Get all worktrees with info
	infos, err := git.ListWorktreesWithInfo(bareDir, false)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	// Find prune candidates. git-prunable (path-gone) worktrees are listed
	// separately: they are excluded from ListWorktreesWithInfo so they never
	// leak into other commands, and are reaped by default here.
	var candidates []pruneCandidate

	prunables, err := git.ListPrunableWorktrees(bareDir)
	if err != nil {
		return fmt.Errorf("failed to list prunable worktrees: %w", err)
	}
	for _, info := range prunables {
		candidates = append(candidates, pruneCandidate{
			info:      info,
			reason:    determineSkipReason(info, cwd, force),
			pruneType: prunePrunable,
		})
	}

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

		// Check for merged (only if --merged flag was passed). The default-branch
		// and target-branch worktrees are never candidates for their own merge.
		if target.ref != "" && info.Branch != "" && info.Branch != defaultBranch && info.Branch != target.branch {
			isMerged, mergeErr := git.IsBranchMerged(bareDir, info.Branch, target.ref)
			if mergeErr != nil {
				// Say so rather than silently reporting no candidates.
				logger.Warning("Could not check whether %s is merged into %s: %v", info.Branch, target.ref, mergeErr)
			} else if isMerged {
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
	if jsonOutput {
		return outputPruneJSON(candidates)
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
		if info.Locked {
			return skipLocked
		}
		if info.Prunable {
			return skipNone
		}
		if info.Dirty {
			return skipDirty
		}
		if info.Ahead > 0 {
			return skipUnpushed
		}
	}

	return skipNone
}

type pruneJSON struct {
	Name       string     `json:"name"`
	Path       string     `json:"path"`
	Branch     string     `json:"branch"`
	Reason     pruneType  `json:"reason"`
	SkipReason skipReason `json:"skip_reason"`
	StaleAge   string     `json:"stale_age"`
}

func outputPruneJSON(candidates []pruneCandidate) error {
	output := make([]pruneJSON, 0, len(candidates))
	for _, candidate := range candidates {
		output = append(output, pruneJSON{
			Name:       filepath.Base(candidate.info.Path),
			Path:       candidate.info.Path,
			Branch:     candidate.info.Branch,
			Reason:     candidate.pruneType,
			SkipReason: candidate.reason,
			StaleAge:   candidate.staleAge,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")

	return encoder.Encode(output)
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
		label := formatter.WorktreeLabel(candidate.info)
		if candidate.pruneType == prunePrunable {
			label = fmt.Sprintf("%s (%s)", label, candidate.pruneType)
		}
		if candidate.pruneType == pruneStale && candidate.staleAge != "" {
			label = fmt.Sprintf("%s (%s)", label, candidate.staleAge)
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

	// git worktree prune is a single repository-wide operation that reaps every
	// path-gone entry at once, so run it lazily and share its result across all
	// prunable candidates instead of invoking it per candidate. It exits 0 even
	// when it fails to delete an individual entry, so verify removal per
	// candidate against the post-prune registry rather than trusting the exit.
	prunedRegistry := false
	var pruneRegistryErr error
	var registeredAfterPrune []string
	pruneRegistry := func() error {
		if prunedRegistry {
			return pruneRegistryErr
		}
		prunedRegistry = true
		pruneRegistryErr = git.PruneWorktrees(bareDir)
		if pruneRegistryErr != nil {
			return pruneRegistryErr
		}
		registeredAfterPrune, pruneRegistryErr = git.ListWorktrees(bareDir)
		return pruneRegistryErr
	}
	stillRegistered := func(path string) bool {
		for _, p := range registeredAfterPrune {
			if fs.PathsEqual(p, path) {
				return true
			}
		}
		return false
	}

	for _, candidate := range candidates {
		label := formatter.WorktreeLabel(candidate.info)
		if candidate.pruneType == prunePrunable {
			label = fmt.Sprintf("%s (%s)", label, candidate.pruneType)
		}

		if candidate.reason != skipNone {
			skipped = append(skipped, fmt.Sprintf("%s (%s)", label, candidate.reason))
			continue
		}

		// git-prunable entries are reaped by the shared prune call; confirm the
		// entry is actually gone before reporting it pruned.
		if candidate.pruneType == prunePrunable {
			if err := pruneRegistry(); err != nil {
				failed = append(failed, fmt.Sprintf("%s: %v", label, err))
				continue
			}
			if stillRegistered(candidate.info.Path) {
				failed = append(failed, fmt.Sprintf("%s: git worktree prune did not remove it", label))
				continue
			}
			pruned = append(pruned, label)
			continue
		}

		// Actually remove the worktree
		if force && candidate.info.Locked {
			if err := git.UnlockWorktree(bareDir, candidate.info.Path); err != nil {
				logger.Debug("Failed to unlock worktree: %v", err)
			}
		}
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

	// Branch deletion can leave the bare repo's HEAD pointing at a deleted
	// ref, which breaks future `git worktree add` invocations.
	var restoredHead string
	if deletedBranches > 0 {
		newHead, err := git.RestoreBareHeadIfDangling(bareDir)
		if err != nil {
			logger.Warning("Could not restore bare HEAD: %v", err)
		} else {
			restoredHead = newHead
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
		if restoredHead != "" {
			logger.Dimmed("    ↳ updated bare HEAD to %s", restoredHead)
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

		return fmt.Errorf("failed to prune %d worktree(s)", len(failed))
	}

	return nil
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
