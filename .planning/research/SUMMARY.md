# Project Research Summary

**Project:** Grove fetch command
**Domain:** CLI command for git worktree management
**Researched:** 2026-01-23
**Confidence:** HIGH

## Executive Summary

The fetch command is a straightforward addition to Grove that fetches remote changes and reports what changed. The recommended implementation uses a **snapshot-based approach**: capture refs before fetch, run `git fetch --all --prune`, capture refs after, then diff the two snapshots. This approach is simpler and more reliable than parsing git's porcelain output, and aligns with Grove's existing exec-based git patterns.

The key value-add over raw `git fetch` is structured output showing what actually changed: new branches, updated branches, and pruned refs. Table stakes features include progress indication, prune support, and clear error handling. Differentiators include per-worktree behind status and JSON output for scripting.

Primary risks are network failures going unnoticed and prune terminology confusing users about what gets deleted. Mitigate by checking remote reachability upfront (Grove already has this), providing per-remote status, and clearly labeling pruned items as remote-tracking refs (not local branches).

## Key Findings

### Recommended Stack

Use `git for-each-ref` for snapshotting refs before/after fetch. This is explicitly recommended over `git show-ref` in git documentation for scripts. Stick with Grove's exec-based approach rather than adding go-git as a dependency.

**Core technologies:**

- `git for-each-ref`: Ref listing (preferred over show-ref for packed-refs handling)
- `git fetch --all --prune`: Multi-remote fetch with cleanup
- Snapshot diffing: Compare before/after ref maps (simpler than parsing porcelain output)

### Expected Features

**Must have (table stakes):**

- Fetch all remotes with prune
- Progress spinner during network operation
- Summary of changes (count of new/updated/pruned refs)
- `--quiet` flag for scripting

**Should have (competitive):**

- Show incoming changes per worktree (`--status`)
- Parallel fetch with `--jobs`
- `--json` output for tooling
- `--dry-run` preview

**Defer (v2+):**

- Submodule support
- Interactive mode
- Auto-pull

### Architecture Approach

Follow Grove's existing command structure: command layer handles user interaction and orchestration, git layer handles low-level operations. Add new `git.GetRefSnapshot()` and `git.DiffRefs()` functions. The fetch command will use `workspace.FindBareDir()` for context, then call git functions.

**Major components:**

1. `commands/fetch.go`: Cobra command, flags, orchestration, output formatting
2. `git/refs.go`: `GetRefSnapshot()`, `DiffRefs()` functions (new file)
3. `git/git.go`: Extend with `FetchAllPrune()` (adds `--all` to existing pattern)

### Critical Pitfalls

1. **Prune confusion**: Users may think `--prune` deletes local branches. Label pruned refs clearly as `origin/branch` format and document that local branches are unaffected.

2. **Silent network failures**: Fetch may "succeed" but refs don't update due to network issues. Check remote reachability first (use existing `git.IsRemoteReachable()`), report per-remote status.

3. **No ref change visibility**: Users can't tell what changed after fetch. Categorize output: "New branches:", "Updated:", "Pruned:" based on snapshot diff.

4. **Bare clone refspec issues**: Grove's bare clones need proper fetch refspecs. Verify refspec configured before fetch; Grove already handles this during clone.

5. **Timeout on large repos**: Default 30s may not be enough. Document timeout configuration, consider longer default for fetch.

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Core fetch with snapshot tracking

**Rationale:** Foundation must exist before enhancements. Snapshot approach establishes data model for all subsequent features.

**Delivers:** `grove fetch` command that fetches all remotes, prunes stale refs, and reports changes (new/updated/pruned).

**Addresses:** Table stakes features (fetch all, prune, progress, summary output).

**Avoids:** Silent failure pitfall by adding basic error reporting. Prune confusion by labeling refs clearly.

### Phase 2: Output modes and flags

**Rationale:** Once core works, add flexibility. JSON output is essential for scripting; verbose mode passes through git's native output.

**Delivers:** `--json`, `--quiet`, `--verbose`, `--dry-run` flags.

**Uses:** Existing Grove JSON patterns from `list.go` and `status.go`.

**Implements:** Consistent flag handling across command.

### Phase 3: Worktree integration

**Rationale:** Key differentiator, but requires working fetch first. Shows which worktrees are behind their upstream after fetch.

**Delivers:** `--status` flag showing per-worktree behind/ahead counts.

**Uses:** Existing `git.ListWorktreesWithInfo()` infrastructure.

**Avoids:** Performance issues by making this opt-in.

### Phase Ordering Rationale

- Phase 1 first because all other features depend on working fetch with change tracking
- Phase 2 before Phase 3 because JSON output is simpler and more universally useful
- Worktree integration last because it's a differentiator, not table stakes

### Research Flags

Phases with standard patterns (no additional research needed):

- **Phase 1:** Snapshot approach is well-documented, git commands are standard
- **Phase 2:** Flag patterns already exist in Grove codebase
- **Phase 3:** Reuses existing worktree listing infrastructure

No phases require additional research. Implementation can proceed directly.

## Confidence Assessment

| Area         | Confidence | Notes                                                       |
| ------------ | ---------- | ----------------------------------------------------------- |
| Stack        | HIGH       | Git documentation verified, approach matches Grove patterns |
| Features     | HIGH       | Based on git-fetch docs and existing CLI patterns           |
| Architecture | HIGH       | Direct codebase analysis, clear component boundaries        |
| Pitfalls     | HIGH       | Combination of git docs and Grove-specific concerns         |

**Overall confidence:** HIGH

### Gaps to Address

- **Parallel fetch performance:** Whether `--jobs` provides meaningful speedup for typical Grove users (most have 1-2 remotes). Can defer to user feedback.
- **Tag handling:** Whether to include tags by default. Recommend yes for completeness, but monitor for edge cases.

## Sources

### Primary (HIGH confidence)

- [Git fetch documentation](https://git-scm.com/docs/git-fetch) - porcelain format, flags
- [Git for-each-ref documentation](https://git-scm.com/docs/git-for-each-ref) - ref listing
- Grove codebase (`internal/git/git.go`, `cmd/grove/commands/`) - existing patterns

### Secondary (MEDIUM confidence)

- [Atlassian Git fetch tutorial](https://www.atlassian.com/git/tutorials/syncing/git-fetch) - usage patterns
- [Git Tower prune guide](https://www.git-tower.com/learn/git/faq/cleanup-remote-branches-with-git-prune) - prune behavior

---

_Research completed: 2026-01-23_
_Ready for roadmap: yes_
