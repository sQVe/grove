# Contributing to Grove

Thanks for contributing! Grove makes Git worktrees simple, and we want contributing to Grove to be simple too.

## Quick Setup

| Step        | Command                                               |
| ----------- | ----------------------------------------------------- |
| **Clone**   | `git clone https://github.com/sQVe/grove && cd grove` |
| **Install** | `go mod download && make deps-tools`                  |
| **Verify**  | `make test-unit && make lint && make build-dev`       |

**Prerequisites:** Go 1.24+, Git 2.5+, golangci-lint

## Tech Stack

**Runtime:**

- **Go 1.24+** — Standard library preferred over external dependencies
- **spf13/cobra** — CLI framework
- **charmbracelet/lipgloss** — Terminal styling
- **muesli/termenv** — Terminal capability detection
- **rogpeppe/go-internal** — Testscript integration tests

**Development:**

- **make** — Build automation
- **golangci-lint** — Linting with gofumpt and goimports
- **gotestsum** — Test runner with better output

## Workspace Architecture

Grove stores Git data in a bare repository (`.bare`) with worktrees as sibling directories:

```
project/
├── .bare/           # Bare Git repository (objects, refs)
├── .git             # File: "gitdir: .bare"
├── main/            # Worktree for main branch
├── feature-auth/    # Worktree for feature/auth
└── bugfix-login/    # Worktree for bugfix/login
```

**Components:**

- `.bare` directory holds the complete Git repository without a working tree
- `.git` file redirects Git operations to `.bare`
- Worktree directories contain isolated working copies
- Branch names like `feature/auth` become `feature-auth` (slashes replaced with dashes)

**Benefits:**

- Work on multiple branches simultaneously without stashing
- Each worktree maintains independent working directory and index
- All worktrees share Git objects — no duplication

**Detection:** Grove finds workspaces by traversing parent directories for `.bare` or `.git` files containing `gitdir: .bare`.

## Changelog

Grove uses [changie](https://changie.dev) for changelog management. PRs that change code must include a changeset file.

### Adding a change

```bash
make change           # Interactive prompt
changie new           # Or directly
```

### When required

| Change type             | Changeset required? |
| ----------------------- | ------------------- |
| New feature             | Yes                 |
| Bug fix                 | Yes                 |
| Breaking change         | Yes                 |
| Performance improvement | Yes                 |
| Documentation only      | No                  |
| CI/workflow changes     | No                  |
| Test-only changes       | No                  |
| Dependency updates      | No (labeled PRs)    |

### Changeset types

| Type       | Use when                         | Version bump |
| ---------- | -------------------------------- | ------------ |
| Added      | New feature                      | Minor        |
| Changed    | Modification to existing feature | Minor        |
| Deprecated | Feature marked for removal       | Minor        |
| Removed    | Feature removed                  | Minor        |
| Fixed      | Bug fix                          | Patch        |
| Security   | Security fix                     | Patch        |

### Preview changes

```bash
make change-preview   # Show unreleased changes and next version
```

## Testing Strategy

Test the code that breaks, not the code that makes the metrics green.

**Unit tests** (`*_test.go`) — Internal functions. Use real Git.

- Single function behavior and error conditions
- Tests should be short and focused, and most importantly, fast

**Testscript tests** (`cmd/grove/testdata/script/*.txt`) — CLI commands and workflows.

- User-facing behavior and error messages
- Complex setups or multi-step flows
- Exit codes and command integration

**Decision:**

- Can a user type it? → Testscript
- Are we testing a flow? → Testscript
- Otherwise → Unit test

**Testscript organization:**

- `*_validation.txt` - Fast tests: arguments, help, preconditions
- `*_integration.txt` - Slower tests: actual Git operations, shared fixtures

## CLI Output Style Guide

Consistent output makes Grove predictable and professional. Follow these patterns for all commands.

### Output Symbols

| Symbol | Meaning  | When to use                                    |
| ------ | -------- | ---------------------------------------------- |
| `✓`    | Success  | Operation completed                            |
| `⚠`    | Warning  | Something the user should know, not an error   |
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
import "github.com/sqve/grove/internal/formatter"

// Format indicators
marker := formatter.CurrentMarker(isCurrent)  //  or * or space
dirty := formatter.Dirty(isDirty)             //  or [dirty] or empty
lock := formatter.Lock(isLocked)              //  or [locked] or empty
sync := formatter.Sync(ahead, behind)         // ↑N ↓M or +N -M
```
