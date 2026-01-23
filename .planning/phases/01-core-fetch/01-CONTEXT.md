# Phase 1: Core Fetch - Context

**Gathered:** 2026-01-23
**Status:** Ready for planning

<domain>
## Phase Boundary

Fetch all remotes and show what changed (new, updated, pruned refs). Users can run `grove fetch` from any directory within a grove workspace. Pruning happens automatically. Remotes with no changes are omitted from output.

</domain>

<decisions>
## Implementation Decisions

### Output presentation

- Group changes by remote (origin: [changes], upstream: [changes])
- Follow existing codebase styling patterns for colors/symbols
- Display ref names as short names (strip refs/remotes/origin/)
- No summary line — just the list of changes

### Change summaries

- Updated branches: show commit count (+3 commits)
- New branches: show commit count ahead of default branch
- Pruned refs: show reason hint (deleted on remote)

### Progress feedback

- Show per-remote status during fetch ("Fetching origin...")
- Progress overwrites in place (single line that updates)
- Explicit message when no changes ("All remotes up to date")

### Error handling

- Continue fetching other remotes if one fails
- Report all failures at end
- Retry failed remotes once before giving up

### Claude's Discretion

- Whether progress line clears after fetch completes
- Exit code behavior on partial success
- Error message verbosity level

</decisions>

<specifics>
## Specific Ideas

- Comparison base for new branch commit counts is the default branch (origin/main or origin/master)
- Output should feel familiar to git users

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

_Phase: 01-core-fetch_
_Context gathered: 2026-01-23_
