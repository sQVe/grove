# Stack Research: Fetch Command Ref Tracking

**Project:** Grove fetch command
**Researched:** 2026-01-23
**Overall confidence:** HIGH

## Executive Summary

The fetch command needs to track ref changes (new, updated, pruned) across a fetch operation. Two viable approaches exist: parsing `git fetch --porcelain` output, or snapshotting refs before/after with `git for-each-ref`. The **snapshot approach is recommended** because it provides more reliable data, works with all git versions, and aligns with Grove's existing patterns.

## Ref Tracking Approaches

### Approach 1: Parse `git fetch --porcelain` (Not Recommended)

Git's `--porcelain` flag produces machine-parseable output:

```
<flag> <old-oid> <new-oid> <local-reference>
```

Flag values:
| Flag | Meaning |
|------|---------|
| ` ` | Fast-forward |
| `+` | Forced update |
| `-` | Pruned ref |
| `t` | Tag update |
| `*` | New ref |
| `!` | Rejected/failed |
| `=` | Up-to-date (requires `--verbose`) |

**Pros:**

- Single command captures changes
- Efficient for large repos

**Cons:**

- Requires `--verbose` to see up-to-date refs
- Output goes to stdout instead of stderr (different from normal fetch)
- Incompatible with `--recurse-submodules`
- Parsing complexity for multi-remote scenarios
- Doesn't capture current worktree branch positions

**Confidence:** HIGH (verified via [git-fetch documentation](https://git-scm.com/docs/git-fetch))

### Approach 2: Snapshot Before/After (Recommended)

Capture ref state before fetch, run fetch, capture after, diff the maps.

```go
// Before fetch
beforeRefs := getRefSnapshot(repoPath)

// Run fetch
git.FetchPrune(repoPath)

// After fetch
afterRefs := getRefSnapshot(repoPath)

// Diff
changes := diffRefs(beforeRefs, afterRefs)
```

**Pros:**

- Simple, reliable logic
- Works with any git version
- Captures exact state (not parsing fetch output)
- Handles edge cases (interrupted fetches, concurrent changes)
- Aligns with Grove's existing `git.RevParse` and `git.ListBranches` patterns

**Cons:**

- Two `for-each-ref` calls (negligible overhead)
- Doesn't distinguish fast-forward vs forced update (rarely needed)

**Confidence:** HIGH (verified via testing and [git-for-each-ref documentation](https://git-scm.com/docs/git-for-each-ref))

## Git Commands

### Listing Refs: `git for-each-ref`

Preferred over `git show-ref` for scripting. Handles packed refs efficiently, supports flexible formatting.

```bash
git for-each-ref --format='%(objectname) %(refname)' refs/remotes/
```

Output:

```
dc4b48a49559285268e73bc30d0cd03fb614aeb5 refs/remotes/origin/main
1ca63b779dab9b36a7c8f97437a2cda6dabdd601 refs/remotes/origin/feature
```

**Why not `git show-ref`:** Documentation explicitly recommends `for-each-ref` for scripts due to better packed-refs handling and format flexibility.

**Confidence:** HIGH (verified via [git-for-each-ref docs](https://git-scm.com/docs/git-for-each-ref))

### Fetching: `git fetch --all --prune`

For multi-remote scenarios, `--all` fetches from all remotes. Add `--prune` to remove stale refs.

```bash
git fetch --all --prune
```

For parallel fetching (faster with multiple remotes):

```bash
git fetch --all --prune --jobs=4
```

**Note:** Grove already has `git.FetchPrune()` that runs `git fetch --prune`. Extend to support `--all` for multi-remote.

**Confidence:** HIGH (verified via [git-fetch documentation](https://git-scm.com/docs/git-fetch))

## Implementation Recommendation

### Data Structures

```go
type RefSnapshot map[string]string // refname -> commit hash

type RefChange struct {
    Ref     string
    OldHash string // empty for new refs
    NewHash string // empty for pruned refs
    Type    RefChangeType
}

type RefChangeType int

const (
    RefNew RefChangeType = iota
    RefUpdated
    RefPruned
)
```

### Core Functions

```go
// GetRefSnapshot returns current ref state for given patterns
func GetRefSnapshot(repoPath string, patterns ...string) (RefSnapshot, error) {
    args := []string{"for-each-ref", "--format=%(objectname) %(refname)"}
    args = append(args, patterns...)
    // Execute and parse
}

// DiffRefs compares two snapshots and returns changes
func DiffRefs(before, after RefSnapshot) []RefChange {
    var changes []RefChange

    // Find new and updated refs
    for ref, newHash := range after {
        if oldHash, exists := before[ref]; !exists {
            changes = append(changes, RefChange{Ref: ref, NewHash: newHash, Type: RefNew})
        } else if oldHash != newHash {
            changes = append(changes, RefChange{Ref: ref, OldHash: oldHash, NewHash: newHash, Type: RefUpdated})
        }
    }

    // Find pruned refs
    for ref, oldHash := range before {
        if _, exists := after[ref]; !exists {
            changes = append(changes, RefChange{Ref: ref, OldHash: oldHash, Type: RefPruned})
        }
    }

    return changes
}
```

### Fetch Flow

```go
func FetchWithChanges(repoPath string, allRemotes bool) ([]RefChange, error) {
    // Snapshot before
    before, err := GetRefSnapshot(repoPath, "refs/remotes/", "refs/tags/")
    if err != nil {
        return nil, err
    }

    // Execute fetch
    if allRemotes {
        err = FetchAllPrune(repoPath)
    } else {
        err = FetchPrune(repoPath)
    }
    if err != nil {
        return nil, err
    }

    // Snapshot after
    after, err := GetRefSnapshot(repoPath, "refs/remotes/", "refs/tags/")
    if err != nil {
        return nil, err
    }

    return DiffRefs(before, after), nil
}
```

## Alternatives Considered

### go-git Library

Pure Go git implementation. Would eliminate exec overhead.

**Why not:**

- Grove already uses exec-based git commands throughout
- go-git doesn't support all git features
- Adds significant dependency for minimal benefit
- exec approach is consistent with existing codebase

**Confidence:** MEDIUM (based on [go-git documentation](https://github.com/go-git/go-git))

### Parse Human-Readable Fetch Output

Parse the default `git fetch` stderr output.

**Why not:**

- Output format varies by git version
- Not designed for machine parsing
- Localization can affect output

**Confidence:** HIGH (this is a known anti-pattern)

## Consistency with Existing Codebase

Grove's `internal/git` package already uses:

- `executeWithOutput()` for capturing command output
- `bufio.Scanner` for line-by-line parsing
- `GitCommand()` for command construction with timeout
- Similar ref operations in `git.RevParse()`, `git.ListBranches()`

The snapshot approach fits these patterns exactly.

## JSON Output Structure

Following Grove's existing JSON patterns (see `list.go`, `status.go`):

```go
type FetchResult struct {
    New     []RefInfo `json:"new,omitempty"`
    Updated []RefInfo `json:"updated,omitempty"`
    Pruned  []RefInfo `json:"pruned,omitempty"`
}

type RefInfo struct {
    Ref     string `json:"ref"`
    OldHash string `json:"old_hash,omitempty"`
    NewHash string `json:"new_hash,omitempty"`
}
```

## Summary

| Aspect         | Recommendation              | Confidence |
| -------------- | --------------------------- | ---------- |
| Ref tracking   | Snapshot before/after       | HIGH       |
| Ref listing    | `git for-each-ref`          | HIGH       |
| Fetch command  | `git fetch --all --prune`   | HIGH       |
| Library        | Stick with exec (no go-git) | HIGH       |
| Output parsing | Line-by-line with Scanner   | HIGH       |

## Sources

- [Git fetch documentation](https://git-scm.com/docs/git-fetch) - porcelain output format
- [Git for-each-ref documentation](https://git-scm.com/docs/git-for-each-ref) - ref listing
- [Git show-ref documentation](https://git-scm.com/docs/git-show-ref) - comparison with for-each-ref
- [go-git repository](https://github.com/go-git/go-git) - pure Go alternative
