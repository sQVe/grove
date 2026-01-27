# Phase 6: Error Formatting - Research

**Researched:** 2026-01-26
**Domain:** Error handling and user-facing error messages
**Confidence:** HIGH

## Summary

Error formatting with actionable hints is a well-established pattern in CLI applications. The research focused on three key areas: (1) how Go and popular CLIs structure helpful error messages, (2) patterns for error wrapping with hints in Cobra CLI applications, and (3) the codebase's existing error infrastructure.

The standard approach uses error wrapping with contextual hints appended to error messages, either through custom error types or multiline error strings. Git CLI's "did you mean" pattern and kubectl's actionable events demonstrate the value of suggesting next steps directly in error output.

The codebase already has infrastructure in place (`HintGitTooOld` function, `SilenceErrors: true` in main.go) and uses `logger.Error()` consistently for error display. The four required error cases are already detected and logged, but lack actionable hints.

**Primary recommendation:** Create an error wrapping utility that adds contextual hints to specific error patterns, similar to the existing `HintGitTooOld` pattern.

## Standard Stack

Error message enhancement in Go CLI applications uses built-in error handling with optional helper libraries.

### Core

| Library | Version | Purpose                          | Why Standard                                  |
| ------- | ------- | -------------------------------- | --------------------------------------------- |
| errors  | stdlib  | Error creation and wrapping      | Built-in, official Go error handling          |
| fmt     | stdlib  | Error formatting with fmt.Errorf | Standard for formatted errors                 |
| cobra   | 1.8+    | CLI framework with RunE pattern  | Already in use, provides error handling hooks |

### Supporting

| Library            | Version | Purpose                      | When to Use                         |
| ------------------ | ------- | ---------------------------- | ----------------------------------- |
| Custom error types | N/A     | Structured errors with hints | When hints need programmatic access |

### Alternatives Considered

| Instead of     | Could Use                     | Tradeoff                                                     |
| -------------- | ----------------------------- | ------------------------------------------------------------ |
| Error wrapping | pkg/errors                    | pkg/errors is legacy, stdlib errors package is now preferred |
| Inline hints   | Separate logger.Hint() method | Future enhancement (PLSH-02), but not needed for v1.5        |

**Installation:**

```bash
# No additional dependencies needed - uses stdlib
```

## Architecture Patterns

### Recommended Approach: Error Wrapping Pattern

The codebase already uses error wrapping with `fmt.Errorf`. Extend this with hint-aware wrappers:

```
commands/
├── add.go           # Return errors from RunE, hints added in command layer
├── remove.go        # Command layer has context for actionable hints
└── ...

internal/
├── errors/
│   └── errors.go    # Central hint utility (new)
└── git/
    └── git.go       # Low-level errors, wrapped by commands
```

### Pattern 1: Error Detection and Wrapping

**What:** Check for specific error patterns and wrap with hints
**When to use:** When commands can provide actionable next steps
**Example:**

```go
// Source: Codebase analysis + Go error handling patterns
func runAdd(args []string, ...) error {
    // ... existing logic ...
    for _, info := range infos {
        if info.Branch == branch {
            return errors.WithHint(
                fmt.Errorf("worktree already exists for branch %q at %s", branch, info.Path),
                "Use 'grove list' to see existing worktrees, or use --name to choose a different directory",
            )
        }
    }
}
```

### Pattern 2: Multiline Error Messages

**What:** Use multiline strings to append hints to errors
**When to use:** Simple cases where custom error types aren't needed
**Example:**

```go
// Source: Go fmt.Errorf patterns
if fs.PathsEqual(cwd, info.Path) {
    return fmt.Errorf(`cannot delete current worktree

Hint: Switch to a different worktree first with 'grove switch <worktree>'`)
}
```

### Pattern 3: Central Hint Registry

**What:** Map error patterns to hints in a central location
**When to use:** When multiple commands need the same hints, or for consistency
**Example:**

```go
// Source: Existing HintGitTooOld pattern
var errorHints = map[string]string{
    "worktree already exists": "Use 'grove list' to see existing worktrees",
    "cannot delete current": "Switch to a different worktree first",
    "already locked": "Use 'grove unlock' to remove the lock",
}

func WithHint(err error, hint string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%w\n\nHint: %s", err, hint)
}
```

### Anti-Patterns to Avoid

- **Verbose hints in low-level packages:** Git package shouldn't mention grove commands, only high-level commands know CLI context
- **Duplicating error logic:** Error detection already exists (lines 115, 207, 96, 73 in commands), don't re-detect for hints
- **Hints without context:** "Try unlocking" without specifying the command is less helpful than "Use 'grove unlock'"

## Don't Hand-Roll

| Problem              | Don't Build          | Use Instead                      | Why                                                       |
| -------------------- | -------------------- | -------------------------------- | --------------------------------------------------------- |
| Error wrapping       | Custom error chain   | stdlib errors + fmt.Errorf %w    | Native support since Go 1.13, errors.Is/As work correctly |
| Typo suggestions     | Levenshtein distance | cobra.SuggestionsMinimumDistance | Cobra has built-in command suggestions                    |
| Multiline formatting | String concatenation | Backtick strings or fmt.Sprintf  | Cleaner, handles newlines correctly                       |

**Key insight:** Go's stdlib error handling is sufficient for CLI error hints. Avoid introducing error handling libraries unless wrapping errors with context becomes complex enough to justify it.

## Common Pitfalls

### Pitfall 1: Adding Hints at Wrong Layer

**What goes wrong:** Low-level packages (internal/git) add CLI-specific hints like "use grove unlock"
**Why it happens:** Trying to centralize all error logic in one place
**How to avoid:**

- Git layer returns plain errors describing what failed
- Command layer adds hints with CLI context (command names, flags)
- Follow existing HintGitTooOld pattern: wrapper function in git package, called by commands
  **Warning signs:** Seeing "grove" command names in internal/git error messages

### Pitfall 2: Breaking Plain Mode

**What goes wrong:** Error hints use colors or unicode without checking plain mode
**Why it happens:** Forgetting that errors flow through logger.Error() which already handles plain mode
**How to avoid:**

- Return plain strings from error wrappers
- Let logger.Error() handle formatting and plain mode
- Test with --plain flag
  **Warning signs:** Error output differs between TTY and piped output

### Pitfall 3: Inconsistent Hint Format

**What goes wrong:** Some hints use "Try X", others "Use Y", others just describe what's wrong
**Why it happens:** No agreed-upon format for hint messages
**How to avoid:**

- Establish format convention: "Hint: Use 'grove <command>' to ..." or "Hint: Try ..."
- Keep hints actionable (specific commands/flags) not descriptive ("worktree is locked")
- Review existing hints (HintGitTooOld uses logger.Warning + logger.Info pattern)
  **Warning signs:** Hints that don't tell the user what to do next

### Pitfall 4: Losing Error Context

**What goes wrong:** Wrapping error replaces original message instead of preserving it
**Why it happens:** Using fmt.Errorf without %w, or string replacement
**How to avoid:**

- Always use %w when wrapping errors
- Append hints, don't replace error messages
- Test that errors.Is() still works on wrapped errors
  **Warning signs:** Original error details lost, can't check error type with errors.Is()

## Code Examples

Verified patterns from codebase analysis:

### Existing Error Detection (add.go:207)

```go
// Source: /home/sqve/code/personal/grove/main/cmd/grove/commands/add.go:207
for _, info := range infos {
    if info.Branch == branch {
        return fmt.Errorf("worktree already exists for branch %q at %s", branch, info.Path)
    }
}
```

### Existing Error Detection (remove.go:115)

```go
// Source: /home/sqve/code/personal/grove/main/cmd/grove/commands/remove.go:115
if fs.PathsEqual(cwd, info.Path) || fs.PathHasPrefix(cwd, info.Path) {
    logger.Error("%s: cannot delete current worktree", displayName)
    failed = append(failed, displayName)
    continue
}
```

### Existing Error Detection (lock.go:96)

```go
// Source: /home/sqve/code/personal/grove/main/cmd/grove/commands/lock.go:96
if git.IsWorktreeLocked(info.Path) {
    existingReason := git.GetWorktreeLockReason(info.Path)
    if existingReason != "" {
        logger.Error("%s: already locked (%q)", info.Branch, existingReason)
    } else {
        logger.Error("%s: already locked", info.Branch)
    }
    failed = append(failed, info.Branch)
    continue
}
```

### Existing Error Detection (move.go:73)

```go
// Source: /home/sqve/code/personal/grove/main/cmd/grove/commands/move.go:73
if fs.PathsEqual(cwd, worktreeInfo.Path) || fs.PathHasPrefix(cwd, worktreeInfo.Path) {
    return fmt.Errorf("cannot rename current worktree; switch to a different worktree first")
}
```

### Existing Hint Pattern (git.go:42)

```go
// Source: /home/sqve/code/personal/grove/main/internal/git/git.go:42
func HintGitTooOld(err error) error {
    if err != nil && IsGitTooOld(err) {
        logger.Warning("Grove requires Git %s+ for portable worktrees", MinGitVersion)
        logger.Info("Run 'grove doctor' to check your environment")
    }
    return err
}
```

### Error Flow in main.go

```go
// Source: /home/sqve/code/personal/grove/main/cmd/grove/main.go:67
if err := rootCmd.Execute(); err != nil {
    logger.Error("%s", err)
    logger.Dimmed("Run 'grove --help' for usage.")
    os.Exit(1)
}
```

### Recommended Hint Pattern

```go
// Similar to HintGitTooOld, but appends hint to error message
func WithHint(err error, hint string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%w\n\nHint: %s", err, hint)
}

// Usage in commands
return WithHint(
    fmt.Errorf("worktree already exists for branch %q at %s", branch, info.Path),
    "Use 'grove list' to see existing worktrees, or use --name to choose a different directory",
)
```

## State of the Art

| Old Approach        | Current Approach       | When Changed    | Impact                                                                   |
| ------------------- | ---------------------- | --------------- | ------------------------------------------------------------------------ |
| Basic error strings | Error wrapping with %w | Go 1.13 (2019)  | errors.Is/As enable type checking through wrapped errors                 |
| pkg/errors          | stdlib errors package  | Go 1.13+        | No external dependency needed for wrapping                               |
| Logger hint calls   | Inline error hints     | Not yet changed | Phase 6 will add hints to error returns instead of separate logger calls |

**Deprecated/outdated:**

- pkg/errors: Now redundant, stdlib errors provides wrapping
- Separate hint logging: HintGitTooOld uses logger after error check, but inline hints (appended to error message) are more portable

## Open Questions

None - the requirements and approach are well-defined.

## Sources

### Primary (HIGH confidence)

- Go stdlib documentation (errors, fmt packages) - Official error handling patterns
- Codebase analysis - Existing error detection at add.go:207, remove.go:115, lock.go:96, move.go:73
- Codebase analysis - Existing HintGitTooOld pattern at git.go:42
- Codebase analysis - Error flow through main.go:67-71

### Secondary (MEDIUM confidence)

- [Error handling and Go - The Go Programming Language](https://go.dev/blog/error-handling-and-go)
- [Working with Errors in Go 1.13 - The Go Programming Language](https://go.dev/blog/go1.13-errors)
- [Error Handling in Cobra - JetBrains Guide](https://www.jetbrains.com/guide/go/tutorials/cli-apps-go-cobra/error_handling/)
- [How to Build a CLI Tool in Go with Cobra](https://oneuptime.com/blog/post/2026-01-07-go-cobra-cli/view) - Recent 2026 guide
- [Make CLI Great Again: Crafting a User-Friendly Command Line](https://dev.to/realchakrawarti/make-cli-great-again-crafting-a-user-friendly-command-line-270k) - Git's "did you mean" pattern
- [Best Practices for Error Handling in Go - JetBrains Guide](https://www.jetbrains.com/guide/go/tutorials/handle_errors_in_go/best_practices/)
- [Go Style Best Practices](https://google.github.io/styleguide/go/best-practices.html)
- [Building Robust CLI Applications in Go: Best Practices and Patterns](https://jsschools.com/golang/building-robust-cli-applications-in-go-best-pract/)

### Tertiary (LOW confidence)

- None - all findings verified against official documentation or codebase

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Using stdlib only, verified in codebase
- Architecture: HIGH - Patterns exist in codebase (HintGitTooOld), locations identified
- Pitfalls: HIGH - Derived from codebase constraints (plain mode, logger usage)

**Research date:** 2026-01-26
**Valid until:** 2026-02-26 (30 days - stable domain, stdlib patterns don't change rapidly)
