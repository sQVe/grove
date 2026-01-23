# Phase 2: Output Modes - Research

**Researched:** 2026-01-23
**Domain:** CLI output formatting, JSON serialization, verbose output patterns
**Confidence:** HIGH

## Summary

Phase 2 extends `grove fetch` with two output modes: `--json` for machine-readable JSON output and `--verbose` for additional commit details. This builds directly on Phase 1's existing data structures (`RefChange`, `remoteResult`) and follows established patterns from `grove list` and `grove status` commands which already implement both flags.

The implementation is straightforward: Grove has consistent patterns for both flags across multiple commands. JSON output uses Go's `encoding/json` with struct tags for field naming and `omitempty` for optional fields. Verbose output uses the formatter package's `SubItemPrefix()` and follows the indented sub-item pattern.

**Primary recommendation:** Follow the exact patterns from `list.go` and `status.go` - add `--json` and `--verbose` flags via Cobra's `BoolVar`/`BoolVarP`, create JSON struct types with appropriate tags, and add conditional output functions that check the flag state before the existing output logic.

## Standard Stack

### Core

| Library       | Version | Purpose            | Why Standard                                            |
| ------------- | ------- | ------------------ | ------------------------------------------------------- |
| encoding/json | stdlib  | JSON serialization | Go standard library, already used in list.go, status.go |
| spf13/cobra   | v1.8.1  | CLI flags          | Already used, provides BoolVar/BoolVarP for flags       |

### Supporting

| Library            | Version | Purpose              | When to Use                        |
| ------------------ | ------- | -------------------- | ---------------------------------- |
| internal/formatter | -       | Output formatting    | SubItemPrefix() for verbose mode   |
| internal/styles    | -       | Colored output       | Render() for human-readable output |
| internal/config    | -       | Plain mode detection | IsPlain() for conditional styling  |

### Alternatives Considered

| Instead of    | Could Use | Tradeoff                                          |
| ------------- | --------- | ------------------------------------------------- |
| encoding/json | easyjson  | Performance not needed, stdlib is simpler         |
| struct tags   | manual    | Tags provide standard, maintainable field mapping |

**Installation:**
No new dependencies needed - all libraries already in use.

## Architecture Patterns

### Recommended Structure Changes

```
cmd/grove/commands/
├── fetch.go           # Add --json, --verbose flags
└── fetch_test.go      # Add flag tests
```

### Pattern 1: Flag Declaration (from list.go)

**What:** Declare boolean flags with short form for verbose
**When to use:** Adding output mode flags
**Example:**

```go
// Source: cmd/grove/commands/list.go:21-22,45-46
func NewFetchCmd() *cobra.Command {
    var jsonOutput bool
    var verbose bool

    cmd := &cobra.Command{
        // ...
        RunE: func(cmd *cobra.Command, args []string) error {
            return runFetch(jsonOutput, verbose)
        },
    }

    cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output as JSON")
    cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show additional commit details")

    return cmd
}
```

### Pattern 2: JSON Output (from list.go)

**What:** Define JSON struct with tags, use encoder with indentation
**When to use:** Machine-readable output
**Example:**

```go
// Source: cmd/grove/commands/list.go:91-133
type fetchChangeJSON struct {
    Remote   string `json:"remote"`
    RefName  string `json:"ref"`
    Type     string `json:"type"`
    OldHash  string `json:"old_hash,omitempty"`
    NewHash  string `json:"new_hash,omitempty"`
    Commits  int    `json:"commits,omitempty"`
}

type fetchResultJSON struct {
    Changes []fetchChangeJSON `json:"changes"`
    Errors  []fetchErrorJSON  `json:"errors,omitempty"`
}

func outputFetchJSON(results []remoteResult) error {
    output := fetchResultJSON{
        Changes: make([]fetchChangeJSON, 0),
    }
    // ... populate ...

    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(output)
}
```

### Pattern 3: Verbose Output (from status.go)

**What:** Add indented sub-items with prefix after main line
**When to use:** Showing additional details in human-readable mode
**Example:**

```go
// Source: cmd/grove/commands/status.go:195-228
func printRefChangeVerbose(bareDir, remote string, change git.RefChange) {
    // Print normal line first (existing printRefChange logic)
    printRefChange(bareDir, remote, change)

    // Add verbose details
    prefix := formatter.SubItemPrefix()

    // Show commit hash info
    if change.OldHash != "" {
        if config.IsPlain() {
            fmt.Printf("    %s old: %s\n", prefix, change.OldHash[:7])
        } else {
            fmt.Printf("    %s old: %s\n",
                styles.Render(&styles.Dimmed, prefix),
                styles.Render(&styles.Dimmed, change.OldHash[:7]))
        }
    }
    if change.NewHash != "" {
        if config.IsPlain() {
            fmt.Printf("    %s new: %s\n", prefix, change.NewHash[:7])
        } else {
            fmt.Printf("    %s new: %s\n",
                styles.Render(&styles.Dimmed, prefix),
                styles.Render(&styles.Dimmed, change.NewHash[:7]))
        }
    }
}
```

### Pattern 4: Output Mode Precedence (from status.go)

**What:** JSON takes precedence over verbose
**When to use:** When both flags could be set
**Example:**

```go
// Source: cmd/grove/commands/status.go:90-98
if jsonOutput {
    return outputFetchJSON(results)
}

if verbose {
    return outputFetchVerbose(bareDir, results)
}

return outputFetchResults(bareDir, results)
```

### Anti-Patterns to Avoid

- **Mixing output modes:** Don't output JSON fields in verbose mode or vice versa
- **Different data in JSON vs text:** JSON should contain same (or more) info as text output
- **Forgetting omitempty:** Empty strings and zero values clutter JSON output

## Don't Hand-Roll

| Problem             | Don't Build          | Use Instead                | Why                                            |
| ------------------- | -------------------- | -------------------------- | ---------------------------------------------- |
| JSON serialization  | Manual string concat | encoding/json with structs | Escaping, proper types, maintainable           |
| Indentation prefix  | Custom string        | formatter.SubItemPrefix()  | Handles plain mode, consistent across commands |
| Conditional styling | Inline ANSI codes    | styles.Render() + IsPlain  | Handles NO_COLOR env var, config settings      |

**Key insight:** Both list.go and status.go have working implementations. Copy their patterns exactly rather than inventing new ones.

## Common Pitfalls

### Pitfall 1: Inconsistent JSON Field Names

**What goes wrong:** Using Go struct field names instead of idiomatic JSON names
**Why it happens:** Forgetting to add json tags
**How to avoid:**

- Always add `json:"snake_case"` tags
- Use `omitempty` for optional fields
- Review existing JSON output in list.go, status.go for naming conventions
  **Warning signs:** CamelCase in JSON output, `null` instead of omitted fields

### Pitfall 2: Empty Results Handling

**What goes wrong:** JSON output differs when no changes (empty array vs null)
**Why it happens:** Uninitialized slice defaults to nil, encodes as `null`
**How to avoid:**

- Initialize slices with `make([]T, 0)` not `var x []T`
- Test with zero changes explicitly
  **Warning signs:** `"changes": null` instead of `"changes": []`

### Pitfall 3: Verbose Output Without Normal Output

**What goes wrong:** Verbose mode shows only extra details, loses main info
**Why it happens:** Replacing output instead of augmenting
**How to avoid:**

- Verbose should ADD to default output, not replace
- Call normal output first, then add verbose lines
  **Warning signs:** Users confused because verbose has less info than default

### Pitfall 4: Hash Truncation Inconsistency

**What goes wrong:** Full hashes in JSON, truncated in text (or vice versa)
**Why it happens:** Not deciding on consistent policy
**How to avoid:**

- JSON: full hashes (machine consumption, can truncate if needed)
- Text verbose: short hashes (7 chars, human readable)
- Document the policy
  **Warning signs:** Scripts parsing text output for hashes

### Pitfall 5: Plain Mode Forgotten in Verbose

**What goes wrong:** ANSI codes in output when --plain is set
**Why it happens:** Using styles.Render without checking IsPlain
**How to avoid:**

- Check `config.IsPlain()` for conditional styling
- Or use styles functions that handle it automatically
  **Warning signs:** Color codes visible when piping output

## Code Examples

### Complete JSON Output Structure

```go
// Based on patterns from list.go and doctor.go
type fetchChangeJSON struct {
    Remote      string `json:"remote"`
    RefName     string `json:"ref"`
    Type        string `json:"type"`
    OldHash     string `json:"old_hash,omitempty"`
    NewHash     string `json:"new_hash,omitempty"`
    CommitCount int    `json:"commit_count,omitempty"`
}

type fetchErrorJSON struct {
    Remote  string `json:"remote"`
    Message string `json:"message"`
}

type fetchResultJSON struct {
    Changes []fetchChangeJSON `json:"changes"`
    Errors  []fetchErrorJSON  `json:"errors,omitempty"`
}
```

### Empty Results JSON

```go
// Source pattern: list.go:107-108 initializes slice, not nil
func outputFetchJSON(results []remoteResult) error {
    output := fetchResultJSON{
        Changes: make([]fetchChangeJSON, 0),  // Empty array, not null
    }

    for _, result := range results {
        if result.Error != nil {
            output.Errors = append(output.Errors, fetchErrorJSON{
                Remote:  result.Remote,
                Message: result.Error.Error(),
            })
            continue
        }

        for _, change := range result.Changes {
            // ... add to Changes ...
        }
    }

    enc := json.NewEncoder(os.Stdout)
    enc.SetIndent("", "  ")
    return enc.Encode(output)
}
```

### Verbose Sub-Item Pattern

```go
// Source pattern: status.go:221-228
func printVerboseCommitInfo(change git.RefChange) {
    prefix := formatter.SubItemPrefix()

    if change.OldHash != "" {
        shortHash := change.OldHash
        if len(shortHash) > 7 {
            shortHash = shortHash[:7]
        }
        if config.IsPlain() {
            fmt.Printf("    %s from: %s\n", prefix, shortHash)
        } else {
            fmt.Printf("    %s from: %s\n",
                styles.Render(&styles.Dimmed, prefix),
                styles.Render(&styles.Dimmed, shortHash))
        }
    }

    if change.NewHash != "" {
        shortHash := change.NewHash
        if len(shortHash) > 7 {
            shortHash = shortHash[:7]
        }
        if config.IsPlain() {
            fmt.Printf("    %s to:   %s\n", prefix, shortHash)
        } else {
            fmt.Printf("    %s to:   %s\n",
                styles.Render(&styles.Dimmed, prefix),
                styles.Render(&styles.Dimmed, shortHash))
        }
    }
}
```

## State of the Art

| Old Approach        | Current Approach     | When Changed      | Impact                            |
| ------------------- | -------------------- | ----------------- | --------------------------------- |
| Custom text formats | Structured JSON      | Grove 1.0         | Machine parseable, scriptable     |
| Verbose as log      | Verbose as sub-items | Grove list/status | Consistent indentation, scannable |

**Deprecated/outdated:**

- None - JSON and verbose patterns are stable in Grove

## Open Questions

1. **Verbose commit details content**
    - What we know: Requirements say "additional commit details"
    - What's unclear: Exactly which details (hash, author, date, message?)
    - Recommendation: Start minimal - short hashes (from/to). Can expand later based on feedback.

2. **JSON error format**
    - What we know: Errors should be included in JSON
    - What's unclear: Whether to return non-zero exit code when JSON has errors
    - Recommendation: Follow doctor.go pattern - include errors in JSON AND return error for exit code.

## Sources

### Primary (HIGH confidence)

- Grove codebase analysis (2026-01-23):
    - `cmd/grove/commands/list.go` - JSON output pattern, verbose flag pattern
    - `cmd/grove/commands/status.go` - JSON output, verbose sub-items pattern
    - `cmd/grove/commands/doctor.go` - JSON structure with errors
    - `internal/formatter/formatter.go` - SubItemPrefix(), styling helpers
    - `internal/config/config.go` - IsPlain() for conditional styling
- Phase 1 implementation:
    - `cmd/grove/commands/fetch.go` - Current structure, remoteResult type
    - `internal/git/fetch.go` - RefChange type with OldHash, NewHash

### Secondary (MEDIUM confidence)

- None needed - all patterns verified in codebase

### Tertiary (LOW confidence)

- None - all findings from existing code

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - All packages already in use in similar commands
- Architecture: HIGH - Patterns copied directly from list.go and status.go
- Pitfalls: HIGH - Based on examining existing code edge cases

**Research date:** 2026-01-23
**Valid until:** 90 days (stable patterns, internal Grove code)
