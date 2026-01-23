# Pitfalls Research

**Domain:** CLI fetch command implementation
**Researched:** 2026-01-23
**Confidence:** HIGH (codebase analysis + git documentation)

## Critical Pitfalls

Mistakes that cause incorrect behavior or data loss.

### Pitfall 1: Confusing Pruned Remote-Tracking Refs with Local Branches

**What goes wrong:** `git fetch --prune` removes remote-tracking refs (e.g., `origin/feature-x`) but does NOT delete local branches. Users may expect local branches to be cleaned up too, or vice versa - fear that pruning will delete their local work.

**Why it happens:** The terminology "prune" suggests deletion broadly, but git prune only affects remote-tracking refs, not local branches.

**Warning signs:**

- User complains "I still see the branch after pruning"
- Code that attempts to use pruned refs as if they still exist
- Tests that check local branch state after expecting prune to delete it

**Prevention:**

- Document clearly: "Prunes stale remote-tracking refs, local branches unaffected"
- When showing pruned refs, use format `origin/branch-name` to clarify they are remote-tracking refs
- Consider offering a separate "clean local branches with gone upstream" feature, distinct from fetch

**Implementation step:** Output formatting - clearly label pruned refs as remote-tracking.

### Pitfall 2: Silent Failure When Remote is Unreachable

**What goes wrong:** Network timeouts or auth failures cause fetch to fail without clear indication of which remote failed or why. User doesn't know if their data is stale.

**Why it happens:** Error messages from git may be cryptic or buried in stderr. When fetching multiple remotes, one failure can be lost in output.

**Warning signs:**

- Fetch "succeeds" but refs aren't updated
- User is surprised when their ref is outdated
- Error appears only in debug logs

**Prevention:**

- Check remote reachability before fetch (Grove already has `git.IsRemoteReachable()`)
- Report per-remote fetch status explicitly
- Surface network/auth errors clearly, not just "fetch failed"
- Consider separate exit codes for partial success vs full failure

**Implementation step:** Error handling - explicit per-remote status reporting.

### Pitfall 3: Not Distinguishing New vs Updated vs Pruned Refs

**What goes wrong:** User cannot tell what actually changed. All refs appear the same in output. Critical distinction between "new branch appeared" vs "existing branch was force-pushed" is lost.

**Why it happens:** Default git fetch output is minimal. Porcelain output provides flags but requires parsing.

**Warning signs:**

- User asks "what actually changed?"
- Force-pushed branches go unnoticed (can cause rebase/merge conflicts later)
- New branches not noticed

**Prevention:**

- Parse git fetch porcelain output for machine-readable status flags:
    - `*` = new ref
    - `+` = forced update
    - `-` = pruned
    - `(space)` = fast-forward
- Present categorized output: "New branches:", "Updated branches:", "Pruned refs:"

**Implementation step:** Output parsing - use `--porcelain` for structured ref status.

## Moderate Pitfalls

Mistakes that cause confusion or suboptimal behavior.

### Pitfall 4: Forgetting Tags During Fetch

**What goes wrong:** Tags are not fetched by default, causing version references to be outdated or missing.

**Why it happens:** `git fetch` by default only fetches branches, not tags unless the refspec includes them or `--tags` is specified.

**Warning signs:**

- User expects to see latest release tag but it's missing
- `git describe` gives unexpected results
- Tag-based version detection fails

**Prevention:**

- Document tag behavior explicitly
- Consider `--tags` option for completeness
- Warn if tag fetch is specifically requested but fails

**Implementation step:** Flag definition - decide on tag fetch behavior (default: include tags).

### Pitfall 5: Progress Output Corruption When Piped

**What goes wrong:** Progress indicators (carriage returns, ANSI sequences) corrupt output when redirected to file or piped to another command.

**Why it happens:** Git's progress output uses terminal-specific features. Piping stdout doesn't automatically suppress progress.

**Warning signs:**

- JSON output contains garbage characters
- Log files have unreadable progress bars
- Downstream parsers fail on unexpected characters

**Prevention:**

- Detect if stdout is a TTY (use `os.IsTerminal(os.Stdout.Fd())`)
- Suppress progress indicators when not a TTY
- Use `--progress` / `--no-progress` git flags appropriately
- Grove's existing `logger.StartSpinner()` already handles this

**Implementation step:** TTY detection - check terminal before enabling progress output.

### Pitfall 6: Not Handling Shallow Clone Limitations

**What goes wrong:** Fetch operations behave differently or fail on shallow clones (created with `--depth`).

**Why it happens:** Shallow clones have incomplete history. Some operations like `git fetch --unshallow` or fetching older commits require different handling.

**Warning signs:**

- "fatal: refusing to fetch into shallow repository" errors
- Missing history after fetch
- Unexpected behavior with `--depth` flag

**Prevention:**

- Detect shallow clones: check for `.git/shallow` file or `git rev-parse --is-shallow-repository`
- Warn user if fetch may be limited due to shallow clone
- Consider `--unshallow` option for full history recovery

**Implementation step:** Clone detection - check for shallow state before fetch.

### Pitfall 7: Fetch Refspec Misconfiguration After Bare Clone

**What goes wrong:** Bare clones don't automatically configure fetch refspecs, causing `git fetch` to not update remote-tracking refs properly.

**Why it happens:** Grove uses bare clones for the `.bare` directory. Bare clones by default don't set up refspecs for remote-tracking branches.

**Warning signs:**

- `git fetch` runs but no remote-tracking branches update
- `origin/main` doesn't exist even after fetch
- Tests pass initially but fail after fresh clone

**Prevention:**

- Grove already calls `git.ConfigureFetchRefspec()` after clone
- Verify refspec exists before fetch; configure if missing
- Doctor command could check for missing refspecs

**Implementation step:** Prerequisite check - ensure refspec configured before fetch.

## Minor Pitfalls

Annoyances that are easily fixed.

### Pitfall 8: Timeout Too Short for Large Repositories

**What goes wrong:** Fetch times out on large repositories with slow connections.

**Why it happens:** Grove has a default 30s timeout (`config.DefaultConfig.Timeout`). Large repos or slow networks may exceed this.

**Warning signs:**

- "context deadline exceeded" errors
- Partial fetches that leave repository in inconsistent state
- Works locally but fails in CI with slow network

**Prevention:**

- Make timeout configurable (already is via `grove.timeout` git config)
- Document timeout configuration
- Consider longer default for fetch specifically (network operations)
- Show progress to indicate activity even if slow

**Implementation step:** Configuration - document timeout setting, consider fetch-specific default.

### Pitfall 9: Multiple Remotes Fetched Sequentially

**What goes wrong:** Fetching from many remotes takes a long time when done sequentially.

**Why it happens:** Default fetch processes remotes one at a time.

**Warning signs:**

- Long wait times with many remotes
- Users complain about slow performance

**Prevention:**

- Use `git fetch --all --jobs=N` for parallel fetching
- Grove could parallel fetch remotes using goroutines
- Report which remote is currently being fetched
- Grove's doctor command already parallelizes remote reachability checks (good pattern)

**Implementation step:** Performance - consider parallel fetch for `--all` case.

### Pitfall 10: Inconsistent Exit Codes

**What goes wrong:** Partial success (some remotes fetched, some failed) returns same exit code as full failure, making scripting unreliable.

**Why it happens:** Simple success/failure binary doesn't capture partial success.

**Warning signs:**

- Scripts can't distinguish "everything failed" from "one remote unreachable"
- Automation retries unnecessarily or not enough

**Prevention:**

- Document exit code semantics
- Consider: 0 = all success, 1 = partial success (some failed), 2 = total failure
- JSON output should include per-remote status regardless of exit code

**Implementation step:** Exit codes - define and document clearly.

## Edge Cases

Situations that need special handling.

### Edge Case 1: No Remotes Configured

**Scenario:** User runs `grove fetch` but no remotes are configured.

**Expected behavior:** Clear error message, not silent success or cryptic git error.

**Prevention:**

```go
remotes, err := git.ListRemotes(bareDir)
if err != nil {
    return fmt.Errorf("failed to list remotes: %w", err)
}
if len(remotes) == 0 {
    return fmt.Errorf("no remotes configured; add one with 'git remote add <name> <url>'")
}
```

### Edge Case 2: Force-Pushed Remote Branch

**Scenario:** Remote branch was force-pushed, causing divergence with local tracking branch.

**Expected behavior:** Warn user that fetch resulted in non-fast-forward update.

**Prevention:** Parse porcelain output for `+` flag indicating forced update.

### Edge Case 3: Remote Branch Renamed

**Scenario:** Team renamed `release/v1` to `release/2025-q4` on remote.

**Expected behavior:** After fetch with prune, old ref deleted, new ref appears. User should see both changes.

**Prevention:** Show both pruned and new refs in output so rename is apparent.

### Edge Case 4: Authentication Expired Mid-Fetch

**Scenario:** Token expires while fetching from multiple remotes.

**Expected behavior:** Partial results preserved, clear error about auth failure.

**Prevention:**

- Report which remote failed authentication
- Suggest credential refresh commands
- Don't fail silently or lose successful fetches

### Edge Case 5: Workspace Not a Grove Workspace

**Scenario:** User runs `grove fetch` in a regular git repo, not a Grove workspace.

**Expected behavior:** Clear error explaining this is a Grove command.

**Prevention:** Already handled by `workspace.FindBareDir()` check. Error message should be clear.

### Edge Case 6: Running from Subdirectory

**Scenario:** User is in `workspace/main/src/` and runs `grove fetch`.

**Expected behavior:** Should work - find workspace root and operate on `.bare`.

**Prevention:** `workspace.FindBareDir()` traverses upward. Ensure this works from any depth.

## Prevention Strategies Summary

| Pitfall                 | Strategy                    | Implementation Step |
| ----------------------- | --------------------------- | ------------------- |
| Prune confusion         | Clear labeling of ref types | Output formatting   |
| Silent network failure  | Per-remote status           | Error handling      |
| Indistinct ref changes  | Porcelain parsing           | Output parsing      |
| Missing tags            | `--tags` flag               | Flag definition     |
| Progress corruption     | TTY detection               | TTY detection       |
| Shallow clone issues    | Shallow detection           | Clone detection     |
| Missing refspec         | Verify before fetch         | Prerequisite check  |
| Timeout too short       | Configurable timeout        | Configuration       |
| Sequential remotes      | Parallel fetch              | Performance         |
| Inconsistent exit codes | Define semantics            | Exit codes          |

## Sources

- [Git Fetch Documentation](https://git-scm.com/docs/git-fetch)
- [Git Tower: Cleanup Remote Branches with Git Prune](https://www.git-tower.com/learn/git/faq/cleanup-remote-branches-with-git-prune)
- [Atlassian Git Tutorial: Git Prune](https://www.atlassian.com/git/tutorials/git-prune)
- [How-To Geek: Terminal Behavior When Piped](https://www.howtogeek.com/these-linux-tools-behave-very-differently-when-you-pipe-them/)
- Grove codebase analysis:
    - `/home/sqve/code/personal/grove/main/internal/git/git.go` - Existing fetch functions, timeout handling
    - `/home/sqve/code/personal/grove/main/cmd/grove/commands/doctor.go` - Parallel remote check pattern
    - `/home/sqve/code/personal/grove/main/internal/config/config.go` - Timeout configuration
