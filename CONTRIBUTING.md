# Contributing to Grove

Thanks for contributing! Grove makes Git worktrees simple, and we want contributing to Grove to be simple too.

## Project Context

Before diving in, check out our steering documents to understand the project vision and standards:

-   **[Product Vision](PRODUCT.md)** - Mission, target users, and value propositions
-   **[Technical Standards](ARCHITECTURE.md)** - Architecture principles, tech stack, and patterns
-   **[Project Structure](STRUCTURE.md)** - File organization and naming conventions

These documents guide all development decisions and ensure consistency across the project.

## Quick Setup

| Step        | Command                                               |
| ----------- | ----------------------------------------------------- |
| **Clone**   | `git clone https://github.com/sQVe/grove && cd grove` |
| **Install** | `go mod download`                                     |
| **Verify**  | `mage test:unit && mage lint && mage build:dev`       |

**Prerequisites:** Go 1.21+, Git 2.5+, golangci-lint 1.50+, Mage build system

## Testing Strategy

Test the code that breaks, not the code that makes the metrics green.

**Unit tests** (`*_test.go`) - Test internal functions directly. Use real Git.

-   Single function behavior and error conditions
-   Tests should be short and focused, and most importantly, fast

**Testscript tests** (`testdata/script/*.txt`) - Test CLI commands and workflows.

-   User-facing behavior and error messages
-   Complex setups or multi-step flows
-   Exit codes and command integration

**Decision:**

-   Can a user type it? → Testscript
-   Are we testing a flow? → Testscript
-   Otherwise → Unit test

**Testscript organization:**

-   `*_validation.txt` - Fast tests: arguments, help, preconditions
-   `*_integration.txt` - Slower tests: actual Git operations, shared fixtures

## CLI Output Style Guide

Consistent output makes Grove predictable and professional. Follow these patterns for all commands.

### Output Symbols

| Symbol | Meaning  | When to use                                    |
| ------ | -------- | ---------------------------------------------- |
| `✓`    | Success  | Operation completed                            |
| `⚠`   | Warning  | Something the user should know, not an error   |
| `✗`    | Error    | Operation failed                               |
| `→`    | Info     | Purely informational (config location, etc.)   |
| `↳`    | Sub-item | Noteworthy details about a completed operation |

### Principles

1. **Keep it simple.** Most operations just need `✓`. Don't over-explain.
2. **One success per operation.** Never nest `✓` markers.
3. **Sub-items for noteworthy details only.** Use `↳` when the user needs to know something extra.
4. **Silence is golden for normal state.** Don't show `[clean]` - absence of `[dirty]` implies clean.

### Worktree Display Format

Both `status` and `list` use identical row format:

```
 main  ↑2      ← current, dirty, ahead 2, locked
  feature            ← clean
```

**Element order:** marker → branch → dirty → sync → lock

| Element | Color Mode     | Plain Mode |
| ------- | -------------- | ---------- |
| Current | (nf-pl-branch) | `*`        |
| Dirty   | (nf-md-diff)   | `[dirty]`  |
| Lock    | (nf-md-lock)   | `[locked]` |
| Ahead   | `↑N` green     | `+N`       |
| Behind  | `↓N` yellow    | `-N`       |

### Operation Output Pattern

```go
// Simple success
logger.Success("Created worktree at %s", path)

// Success with sub-items (noteworthy details)
logger.Success("Created worktree at %s", path)
logger.ListSubItem("preserved 2 files:")
for _, f := range files {
    logger.Dimmed("        %s", f)
}

// Warning before success
logger.Warning("Worktree has unpushed commits")
logger.Success("Removed worktree '%s'", name)
```

### Using the Formatter Package

Use `internal/formatter` for consistent worktree formatting:

```go
import "grove/internal/formatter"

// Format indicators
marker := formatter.CurrentMarker(isCurrent)  //  or * or space
dirty := formatter.Dirty(isDirty)             //  or [dirty] or empty
lock := formatter.Lock(isLocked)              //  or [locked] or empty
sync := formatter.Sync(ahead, behind)         // ↑N ↓M or +N -M
```
