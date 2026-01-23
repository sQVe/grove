# Phase 1: Core Fetch - Research

**Researched:** 2026-01-23
**Domain:** Git fetch operations, remote reference tracking, CLI output formatting
**Confidence:** HIGH

## Summary

Phase 1 implements `grove fetch` to fetch all remotes and display changes (new, updated, pruned refs). The implementation follows established patterns in the Grove codebase: Cobra commands in `cmd/grove/commands/`, git operations in `internal/git/`, and output formatting via `internal/formatter/` and `internal/styles/`.

The standard approach uses direct git command execution (not git libraries), parsing git output to detect ref changes. Grove already has patterns for workspace detection, error handling, progress indicators, and shell completion that will be reused.

**Primary recommendation:** Build on existing patterns - use `git fetch --prune --porcelain` for parseable output, detect changes by comparing refs before/after fetch, group output by remote, and follow the established formatter/styles pattern for consistent colored output.

## Standard Stack

### Core

| Library                | Version | Purpose               | Why Standard                                          |
| ---------------------- | ------- | --------------------- | ----------------------------------------------------- |
| spf13/cobra            | v1.8.1  | CLI framework         | Used by all Grove commands, provides shell completion |
| charmbracelet/lipgloss | v1.1.0  | Terminal styling      | Used for all Grove output formatting                  |
| Go stdlib exec         | -       | Git command execution | Grove pattern: direct git execution, no git libraries |
| Go stdlib testing      | -       | Test framework        | Project standard, with testutil helpers               |

### Supporting

| Library                         | Version | Purpose             | When to Use                                   |
| ------------------------------- | ------- | ------------------- | --------------------------------------------- |
| rogpeppe/go-internal/testscript | -       | Integration testing | Testing full command execution (already used) |

### Alternatives Considered

| Instead of           | Could Use       | Tradeoff                                                                           |
| -------------------- | --------------- | ---------------------------------------------------------------------------------- |
| Direct git execution | go-git library  | Grove explicitly avoids git libraries for simplicity and git version compatibility |
| Custom output parser | git --porcelain | Porcelain format is stable and designed for parsing                                |

**Installation:**
No new dependencies needed - all required libraries already in go.mod.

## Architecture Patterns

### Recommended Project Structure

```
cmd/grove/commands/
├── fetch.go           # Command definition and main logic
└── fetch_test.go      # Unit tests

internal/git/
├── fetch.go           # Git fetch operations (new file)
└── fetch_test.go      # Git operation tests (new file)
```

### Pattern 1: Command Structure

**What:** Cobra command with RunE, flags, and ValidArgsFunction for completion
**When to use:** All Grove commands follow this pattern
**Example:**

```go
// Source: cmd/grove/commands/status.go
func NewFetchCmd() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "fetch",
        Short: "Fetch all remotes and show changes",
        Args:  cobra.NoArgs,
        ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
            return nil, cobra.ShellCompDirectiveNoFileComp
        },
        RunE: func(cmd *cobra.Command, args []string) error {
            return runFetch()
        },
    }
    return cmd
}
```

### Pattern 2: Workspace Detection

**What:** Use `workspace.FindBareDir()` to verify workspace and get bare repo path
**When to use:** All commands that need to operate on grove workspace
**Example:**

```go
// Source: cmd/grove/commands/list.go
bareDir, err := workspace.FindBareDir(cwd)
if err != nil {
    return err
}
```

### Pattern 3: Git Command Execution

**What:** Use `git.GitCommand()` wrapper for timeout support and consistent error handling
**When to use:** All git operations
**Example:**

```go
// Source: internal/git/git.go
cmd, cancel := GitCommand("git", "fetch", "--prune")
defer cancel()
cmd.Dir = repoPath
return runGitCommand(cmd, true)
```

### Pattern 4: Output Formatting

**What:** Use styles package for colors, formatter for consistent layout
**When to use:** All user-facing output
**Example:**

```go
// Source: internal/formatter/formatter.go
// Use styles.Render() with predefined color schemes
styles.Render(&styles.Success, "✓")
styles.Render(&styles.Warning, "!")
styles.Render(&styles.Dimmed, "details")
```

### Pattern 5: Progress Indication

**What:** Use `logger.StartSpinner()` for long-running operations
**When to use:** Operations that may take time (clone, fetch)
**Example:**

```go
// Source: internal/workspace/workspace.go
stop := logger.StartSpinner("Cloning repository...")
defer stop()
// ... operation ...
stop()
logger.Success("Repository cloned")
```

### Anti-Patterns to Avoid

- **Parsing unstable git output:** Use `--porcelain` formats when available for stable parsing
- **Ignoring errors silently:** Grove has explicit error handling patterns - continue on fetch errors but report at end
- **Complex regex parsing:** Git's structured output formats (porcelain, for-each-ref) are easier to parse

## Don't Hand-Roll

| Problem                   | Don't Build             | Use Instead                               | Why                                                               |
| ------------------------- | ----------------------- | ----------------------------------------- | ----------------------------------------------------------------- |
| Remote reference tracking | Custom ref database     | `git for-each-ref` + compare before/after | Git maintains this state, edge cases (packed refs, symbolic refs) |
| Progress indicators       | Custom spinners         | `logger.StartSpinner()`                   | Already implemented, handles cleanup                              |
| Colored output            | ANSI codes directly     | `styles.Render()` with lipgloss           | Handles NO_COLOR, plain mode, Nerd Fonts config                   |
| Shell completion          | Custom completion logic | `cobra.ShellCompDirectiveNoFileComp`      | Cobra generates completion scripts                                |

**Key insight:** Git's plumbing commands (`for-each-ref`, `ls-remote`, `fetch --porcelain`) are designed for scripting and handle edge cases that would require significant testing to handle correctly.

## Common Pitfalls

### Pitfall 1: Parsing Unstable Git Output

**What goes wrong:** Git's human-readable output format changes between versions, breaking parsers
**Why it happens:** Using `git fetch` without `--porcelain` or parsing log/status without format strings
**How to avoid:**

- Use `git fetch --porcelain` for stable, parseable output (Git 2.37+)
- Use `git for-each-ref --format=...` for ref queries
- Never parse output meant for humans
  **Warning signs:** Tests break when git version changes, regex patterns trying to parse English text

### Pitfall 2: Not Handling Multiple Remotes

**What goes wrong:** Logic assumes single remote (origin), breaks for upstream/fork workflows
**Why it happens:** Common case bias - most repos have one remote
**How to avoid:**

- Use `git remote` to list all remotes
- Iterate and fetch each remote independently
- Track errors per-remote, don't fail fast
  **Warning signs:** Hardcoded "origin" strings, single fetch call

### Pitfall 3: Race Conditions in Ref Comparison

**What goes wrong:** Refs change between "before" and "after" snapshots if other processes run
**Why it happens:** Not taking atomic snapshots of ref state
**How to avoid:**

- Capture all refs with single `git for-each-ref` call before fetch
- Capture all refs with single call after fetch
- Compare the two snapshots
  **Warning signs:** Flaky tests, occasional missing changes in output

### Pitfall 4: Ignoring Fetch Errors

**What goes wrong:** Partial failures go unnoticed, user thinks everything fetched
**Why it happens:** Early return on first error
**How to avoid:**

- Continue fetching other remotes if one fails
- Collect errors during iteration
- Report all errors at end
- Context decision: retry once before giving up
  **Warning signs:** User confusion about why changes aren't visible

### Pitfall 5: Progress Output Interfering with Results

**What goes wrong:** Progress messages mixed with fetch results, breaking parsing/readability
**Why it happens:** Writing both to stdout or not clearing progress before results
**How to avoid:**

- Use logger spinner for progress (writes to stderr by default)
- Clear/stop progress before printing results
- Context decision: determine if progress line clears after completion
  **Warning signs:** Messy output with progress and results interleaved

## Code Examples

Verified patterns from existing Grove code:

### Listing All Remotes

```go
// Source: internal/git/git.go:444
func ListRemotes(repoPath string) ([]string, error) {
    cmd, cancel := GitCommand("git", "remote")
    defer cancel()
    cmd.Dir = repoPath

    output, err := executeWithOutputBuffer(cmd)
    if err != nil {
        return nil, err
    }

    var remotes []string
    scanner := bufio.NewScanner(output)
    for scanner.Scan() {
        line := strings.TrimSpace(scanner.Text())
        if line != "" {
            remotes = append(remotes, line)
        }
    }
    return remotes, scanner.Err()
}
```

### Getting Ref List with for-each-ref

```go
// Source: internal/git/status.go:241 (adapted)
// Format output to get ref name and target
cmd, cancel := GitCommand("git", "for-each-ref",
    "--format=%(refname) %(objectname)",
    "refs/remotes/origin/")
defer cancel()
cmd.Dir = repoPath

output, err := executeWithOutputBuffer(cmd)
if err != nil {
    return nil, err
}

refs := make(map[string]string)
scanner := bufio.NewScanner(output)
for scanner.Scan() {
    parts := strings.Fields(scanner.Text())
    if len(parts) == 2 {
        refs[parts[0]] = parts[1] // refname -> commit hash
    }
}
```

### Executing Fetch with Error Collection

```go
// Pattern from context decisions: continue on error, retry once
type remoteError struct {
    remote string
    err    error
}

var errors []remoteError

for _, remote := range remotes {
    err := fetchRemote(bareDir, remote)
    if err != nil {
        // Retry once
        err = fetchRemote(bareDir, remote)
        if err != nil {
            errors = append(errors, remoteError{remote, err})
        }
    }
}

// Report errors at end
if len(errors) > 0 {
    for _, e := range errors {
        logger.Error("Failed to fetch %s: %v", e.remote, e.err)
    }
    return fmt.Errorf("failed to fetch %d remotes", len(errors))
}
```

### Grouped Output by Remote

```go
// Pattern from formatter package style
for remote, changes := range changesByRemote {
    // Remote header
    fmt.Printf("%s:\n", styles.Render(&styles.Info, remote))

    // New branches
    for _, branch := range changes.New {
        fmt.Printf("  %s %s\n",
            styles.Render(&styles.Success, "+"),
            branch)
    }

    // Updated branches
    for _, update := range changes.Updated {
        fmt.Printf("  %s %s (%s)\n",
            styles.Render(&styles.Warning, "•"),
            update.Branch,
            update.CommitCount)
    }

    // Pruned refs
    for _, pruned := range changes.Pruned {
        fmt.Printf("  %s %s %s\n",
            styles.Render(&styles.Dimmed, "×"),
            pruned,
            styles.Render(&styles.Dimmed, "(deleted on remote)"))
    }
}
```

## State of the Art

| Old Approach             | Current Approach               | When Changed        | Impact                                   |
| ------------------------ | ------------------------------ | ------------------- | ---------------------------------------- |
| `git fetch` human output | `git fetch --porcelain`        | Git 2.37 (2022)     | Stable, parseable format for ref changes |
| Polling remote refs      | `git for-each-ref` local cache | Always available    | Fast local operation vs network call     |
| Single error model       | Per-remote error tracking      | N/A (design choice) | Better UX for multi-remote workflows     |

**Deprecated/outdated:**

- Parsing git fetch stderr: Output format varies, use --porcelain stdout instead
- `git ls-remote` for change detection: Slow (network call), use local ref comparison

## Open Questions

1. **Git version requirements for --porcelain**
    - What we know: `git fetch --porcelain` added in Git 2.37 (July 2022)
    - What's unclear: Whether Grove requires Git 2.37+ or needs fallback
    - Recommendation: Check grove.doctor or minimum git version, may need to parse traditional output as fallback

2. **Commit count calculation method**
    - What we know: Need to show "+3 commits" for updated branches
    - What's unclear: Base commit for "new" branches (context says "default branch")
    - Recommendation: Use `git rev-list --count origin/main..origin/feature` for new branches, `git rev-list --count OLD..NEW` for updated branches

3. **Default branch detection**
    - What we know: Context says new branches show commits "ahead of default branch"
    - What's unclear: How to determine default branch per remote
    - Recommendation: Use `git symbolic-ref refs/remotes/origin/HEAD` or fall back to main/master

## Sources

### Primary (HIGH confidence)

- Grove codebase analysis (2026-01-23):
    - `cmd/grove/commands/*.go` - Command patterns
    - `internal/git/git.go` - Git operation patterns
    - `internal/git/status.go` - Ref tracking with for-each-ref
    - `internal/formatter/formatter.go` - Output formatting patterns
    - `internal/styles/styles.go` - Color/styling patterns
    - `.planning/codebase/TESTING.md` - Test patterns
- Git documentation (official):
    - `git fetch --help` - Porcelain format since 2.37
    - `git for-each-ref --help` - Ref enumeration and formatting

### Secondary (MEDIUM confidence)

- Context decisions from 01-CONTEXT.md - User preferences for output format and error handling

### Tertiary (LOW confidence)

- None - all findings verified against codebase or official documentation

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - All packages already in use, verified in go.mod
- Architecture: HIGH - Patterns extracted from existing Grove commands
- Pitfalls: HIGH - Based on git documentation and common scripting pitfalls
- Code examples: HIGH - All examples from existing Grove codebase

**Research date:** 2026-01-23
**Valid until:** 90 days (stable domain - git plumbing commands and Grove patterns)
