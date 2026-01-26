# Phase 5: Output Consistency - Research

**Researched:** 2026-01-26
**Domain:** CLI output patterns, Go logging practices, terminal detection
**Confidence:** HIGH

## Summary

Output consistency in CLI tools centers on three pillars: stream separation (stdout vs stderr), mode detection (TTY vs non-TTY), and message uniformity. The grove codebase already has strong foundations with its logger package and spinner API (from Phase 3), but inconsistencies remain in bare fmt.Print usage and verb tense patterns.

The standard approach is:

- **stderr for all user-facing messages** (success, errors, warnings, info) - keeps them visible during piping
- **stdout for machine-readable output** (JSON, list data) - enables clean piping to other commands
- **Plain mode detection** - automatic TTY detection or explicit --plain flag
- **Consistent verb tense** - past participles (Created, Deleted, Updated) for completed actions

**Primary recommendation:** Audit all commands for bare fmt.Print usage and standardize on logger package. Add spinners to the four identified long-running operations (list, clone, doctor, prune). Standardize empty state messages to "No X found" pattern. Fix remove command to show full path context (issue #68).

## Standard Stack

### Core

| Library           | Version | Purpose            | Why Standard                                        |
| ----------------- | ------- | ------------------ | --------------------------------------------------- |
| logger (internal) | current | Unified output API | Already handles plain mode, stderr routing, symbols |
| Spinner (logger)  | current | Progress feedback  | Already implemented in Phase 3 (SPIN-01-04)         |
| cobra             | current | CLI framework      | Already used, supports RunE for errors              |

### Supporting

| Library      | Version | Purpose          | When to Use                                                     |
| ------------ | ------- | ---------------- | --------------------------------------------------------------- |
| fmt (stdlib) | -       | Formatted output | **Only** for stdout data (JSON, lists). Never for user messages |
| os.Stderr    | -       | Direct stderr    | **Only** within logger package internals                        |

### Alternatives Considered

| Instead of         | Could Use        | Tradeoff                                                                                                         |
| ------------------ | ---------------- | ---------------------------------------------------------------------------------------------------------------- |
| Custom logger      | logrus/zap       | Over-engineered for CLI; adds dependency for features we don't need (log levels, structured logging for servers) |
| mattn/go-isatty    | Manual detection | Not needed - --plain flag + config provides sufficient control                                                   |
| briandowns/spinner | Custom spinner   | Already have custom implementation that integrates with logger                                                   |

**Installation:**
No new dependencies needed. All tools in place.

## Architecture Patterns

### Current State (Good Foundation)

```
internal/logger/
├── logger.go       # Success, Error, Info, Warning, Dimmed, Debug
├── spinner.go      # StartSpinner, Update, Stop, StopWithSuccess/Error
└── logger_test.go

internal/config/
└── config.go       # IsPlain(), SetPlain() - controls plain mode globally

cmd/grove/main.go   # Initializes logger.Init(config.IsPlain(), config.IsDebug())
```

### Pattern 1: User-Facing Output

**What:** All user messages go through logger package
**When to use:** Success confirmations, errors, warnings, info, debug
**Example:**

```go
// BAD - bypasses logger, doesn't respect plain mode, goes to wrong stream
fmt.Println("Created worktree")

// GOOD - respects plain mode, uses stderr, consistent formatting
logger.Success("Created worktree %s", styles.RenderPath(path))
```

### Pattern 2: Machine-Readable Output

**What:** Structured data goes to stdout via fmt
**When to use:** JSON output, list output (for piping)
**Example:**

```go
// GOOD - stdout for piping to jq or other tools
if jsonOutput {
    enc := json.NewEncoder(os.Stdout)
    return enc.Encode(data)
}

// Still use logger for the table display
fmt.Println(formatter.WorktreeRow(...)) // This is stdout, used by list
```

### Pattern 3: Long-Running Operations

**What:** Wrap operations in spinners with informative messages
**When to use:** Operations taking >500ms
**Example:**

```go
// Source: internal/logger/spinner.go (Phase 3)
spin := logger.StartSpinner("Fetching remote changes...")
if err := git.FetchPrune(bareDir); err != nil {
    spin.StopWithError("Failed to fetch")
    return err
}
spin.StopWithSuccess("Fetched remote changes")
```

### Pattern 4: Empty State Messages

**What:** Consistent wording for "no results" scenarios
**When to use:** List/query commands that return zero results
**Example:**

```go
// GOOD - consistent pattern "No X found" or "No X to Y"
if len(candidates) == 0 {
    logger.Info("No worktrees to prune.")
    return nil
}

// Also acceptable for actions
if len(items) == 0 {
    logger.Info("No changes detected.")
    return nil
}
```

### Pattern 5: Batch Operation Summaries

**What:** Success message states count of items affected
**When to use:** Commands operating on multiple items
**Example:**

```go
// From remove.go - good pattern
if len(removed) == 1 {
    logger.Success("Removed worktree %s", removed[0])
} else {
    logger.Success("Removed %d worktrees", len(removed))
}
```

### Anti-Patterns to Avoid

- **Bare fmt.Print for user messages:** Bypasses plain mode, goes to wrong stream
- **Mixed verb tenses:** "Removing worktree" (present continuous) mixed with "Deleted branch" (past)
- **Inconsistent empty states:** "not found" vs "No X found" vs "no results"
- **Missing context in success:** "Deleted worktree branch-name" when user passed directory name (issue #68)

## Don't Hand-Roll

| Problem              | Don't Build                               | Use Instead             | Why                                                                                |
| -------------------- | ----------------------------------------- | ----------------------- | ---------------------------------------------------------------------------------- |
| Plain mode detection | Custom TTY detection with mattn/go-isatty | --plain flag + config   | Explicit control > automatic detection. User can set git config grove.plain for CI |
| Structured logging   | Custom log levels, fields                 | logger package methods  | CLI tools need message formatting, not structured JSON logs                        |
| Progress spinners    | Custom animation loop                     | logger.StartSpinner     | Already handles plain mode, goroutine safety, cleanup                              |
| Output formatting    | String concatenation with ANSI codes      | styles package + logger | Centralized plain mode handling                                                    |

**Key insight:** CLI output differs from server logging. Users read messages, not machines. Focus on clarity and consistency over structured fields and log levels.

## Common Pitfalls

### Pitfall 1: Using fmt.Print for User Messages

**What goes wrong:** Messages bypass logger, ignore plain mode, go to stdout (get piped), lack consistent formatting
**Why it happens:** fmt.Print feels natural, developers forget about plain mode and stream separation
**How to avoid:**

- Audit with `grep -r "fmt\.Print" cmd/` and convert to logger calls
- In code review, reject any fmt.Print not for stdout data
  **Warning signs:** Tests failing in --plain mode, messages appearing in piped output, inconsistent symbols

### Pitfall 2: Inconsistent Verb Tenses

**What goes wrong:** Success messages mix "Creating", "Created", "Deleted", "Removing"
**Why it happens:** Natural language variety feels less repetitive, but confuses patterns
**How to avoid:**

- Use past participles consistently: Created, Deleted, Updated, Removed, Added, Moved
- Present perfect acceptable with "has been": "Worktree has been created"
- Never use present continuous (-ing forms) for completed actions
  **Warning signs:** User confusion about whether operation completed, messages feel inconsistent

### Pitfall 3: Empty States Without Guidance

**What goes wrong:** Command shows "No worktrees" but user doesn't know what to do next
**Why it happens:** Empty state is edge case, developer focuses on success path
**How to avoid:**

- Always show "No X found" or "No X to Y" format
- For commands expecting items, consider hint: "Run 'grove add' to create worktrees"
- Distinguish "none found" from "filtered to none": filter results should say how many total exist
  **Warning signs:** Users asking "did it work?" when command succeeds with no results

### Pitfall 4: Missing Context in Batch Operations

**What goes wrong:** "Deleted worktree feature-123" when user passed "feat-123" directory
**Why it happens:** Command stores branch name internally, displays that instead of user input
**How to avoid:**

- Show what user will recognize: directory name or both
- Pattern: "Deleted worktree dir-name (branch: branch-name)" when they differ
- Issue #68 specifically requests full path for remove command
  **Warning signs:** User can't map output to input, confusion with multiple items

### Pitfall 5: Forgetting Plain Mode in New Code

**What goes wrong:** New features work in terminal but break in CI or when output piped
**Why it happens:** Development happens in interactive terminal, plain mode isn't tested
**How to avoid:**

- Test with --plain flag during development
- Test piped output: `grove list | cat`
- From prior decision: Tests need both logger.Init and config.SetPlain
  **Warning signs:** CI failures, symbols appearing as `?` in plain environments

## Code Examples

Verified patterns from codebase:

### Spinner Usage (Long Operations)

```go
// Source: internal/logger/spinner.go
// Current usage in prune.go line 121
logger.Info("Fetching remote changes...")
if err := git.FetchPrune(bareDir); err != nil {
    logger.Warning("Failed to fetch: %v", err)
}

// Should become:
spin := logger.StartSpinner("Fetching remote changes...")
if err := git.FetchPrune(bareDir); err != nil {
    spin.StopWithError("Failed to fetch")
    return err // or continue with warning
}
spin.StopWithSuccess("Fetched remote changes")
```

### Success Messages (Consistent Verbs)

```go
// Source: cmd/grove/commands/remove.go lines 159-167
// GOOD - uses past participle consistently
logger.Success("Removed worktree %s", removed[0])
logger.Success("Removed %d worktrees", len(removed))

// Source: cmd/grove/commands/add.go line 267
logger.Success("Created worktree at %s", styles.RenderPath(worktreePath))

// Source: cmd/grove/commands/config.go line 651
logger.Success("Created .grove.toml")
```

### Empty State Messages

```go
// Source: cmd/grove/commands/prune.go line 224
// GOOD - consistent pattern
if len(candidates) == 0 {
    logger.Info("No worktrees to prune.")
    return nil
}

// Source: cmd/grove/commands/prune.go line 284
if len(candidates) == 0 {
    logger.Info("No worktrees to remove.")
    return nil
}
```

### Machine-Readable Output (stdout)

```go
// Source: cmd/grove/commands/list.go lines 130-132
// GOOD - JSON to stdout for piping
enc := json.NewEncoder(os.Stdout)
enc.SetIndent("", "  ")
return enc.Encode(output)

// Source: cmd/grove/commands/list.go line 183
// GOOD - table output to stdout for terminal display
fmt.Println(formatter.WorktreeRow(displayInfo, isCurrent, maxNameLen, maxBranchLen))
```

### Logger vs fmt.Print Audit

```go
// Source: cmd/grove/commands/doctor.go lines 577-579
// BAD - should use logger for consistency
if config.IsPlain() {
    fmt.Println("[ok] No issues found")
} else {
    fmt.Println("✓ No issues found")
}

// Should be:
logger.Success("No issues found")
// logger already handles plain mode symbol selection
```

## State of the Art

| Old Approach          | Current Approach                     | When Changed          | Impact                                        |
| --------------------- | ------------------------------------ | --------------------- | --------------------------------------------- |
| Direct fmt.Fprintf    | logger package                       | Phase 3 (2026-01-24)  | Centralized plain mode, consistent formatting |
| No progress feedback  | Spinner API                          | Phase 3 (2026-01-24)  | Users know operations are working             |
| stdout for all output | stderr for messages, stdout for data | Refactor (#14, prior) | Piping works correctly                        |
| Mixed output streams  | Consistent stderr for user messages  | Refactor (#14, prior) | Git-like behavior                             |

**Deprecated/outdated:**

- **Direct fmt.Fprintf for user messages:** Use logger package methods instead
- **Manual plain mode checks:** Let logger handle it (styles.Render already checks config.IsPlain)
- **Present tense success:** "Creating worktree" should be "Created worktree"

## Open Questions

1. **Should we auto-detect TTY or rely only on --plain flag?**
    - What we know: Current implementation uses --plain flag + git config grove.plain
    - What's unclear: Whether auto-detection would help users or cause confusion
    - Recommendation: Keep current approach. Explicit > implicit for CI reproducibility. Users who need it can set git config globally

2. **Empty state actionable hints: always or optional?**
    - What we know: Some commands show hints ("Run with --commit"), others don't
    - What's unclear: Whether every empty state needs a hint
    - Recommendation: Add hints when user likely doesn't know next step (e.g., "grove list" in new workspace). Skip obvious cases (e.g., "grove prune" finding nothing is success)

3. **Verb tense: "Created" vs "has been created"?**
    - What we know: Codebase uses "Created" (past participle, terse)
    - What's unclear: Whether full present perfect is ever better
    - Recommendation: Stick with terse past participles for consistency. Full sentences only in error messages where explanation needed

4. **Issue #68 full path format: always or conditionally?**
    - What we know: User wants mapping between input and output
    - What's unclear: Format when dir name matches branch name
    - Recommendation: Show "Deleted worktree dir-name (branch: branch-name)" only when they differ. Avoids redundancy like "Deleted worktree main (branch: main)"

## Sources

### Primary (HIGH confidence)

- Command Line Interface Guidelines (https://clig.dev) - authoritative CLI design guide
- Internal codebase: cmd/grove/commands/\*.go - current implementation patterns
- Internal codebase: internal/logger/\*.go - logger API and spinner implementation
- GitLab Pajamas Design System: Verb tenses (https://design.gitlab.com/content/verb-tenses/) - guidance on past participles for CLI output
- Issue #68: grove remove output clarity (https://github.com/sqve/grove/issues/68) - user feedback on path display

### Secondary (MEDIUM confidence)

- Go CLI output best practices (https://dev.to/wycliffealphus/leveraging-osstderr-in-go-best-practices-for-effective-error-handling-3iof) - stderr vs stdout patterns
- Cobra CLI framework (https://github.com/spf13/cobra) - standard Go CLI framework patterns
- Git behavior: stdout/stderr separation (https://github.com/cli/cli/issues/2984) - informational messages to stderr

### Tertiary (LOW confidence)

- Terminal spinner libraries comparison - briandowns/spinner most popular, but custom implementation already complete
- TTY detection: mattn/go-isatty (https://github.com/mattn/go-isatty) - not needed, flag-based approach sufficient

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - All tools already in place, no new dependencies needed
- Architecture: HIGH - Patterns verified in current codebase, Phase 3 foundation complete
- Pitfalls: HIGH - Identified from codebase audit and issue #68 feedback
- Code examples: HIGH - All sourced from actual codebase files
- Open questions: MEDIUM - Recommendations based on CLI best practices but need validation

**Research date:** 2026-01-26
**Valid until:** 30 days (stable domain - CLI output conventions don't change rapidly)

**Coverage:**

- ✅ Spinners (SPIN-05, SPIN-06, SPIN-07, SPIN-08)
- ✅ Output clarity (CLRT-01, CLRT-02, CLRT-03, CLRT-04, CLRT-05)
- ✅ Plain mode compliance (constraint verified)
- ✅ Issue #68 requirements analyzed
- ✅ Existing patterns documented
- ✅ fmt.Print audit scope identified
