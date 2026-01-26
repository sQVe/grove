# Grove

## What This Is

Grove is a CLI tool for managing git worktrees with a bare repository workflow. It simplifies creating, switching, and organizing worktrees while providing visibility into repository state across all branches. The CLI provides consistent, responsive feedback with progress spinners and actionable error messages.

## Core Value

Users get a clean, organized multi-branch workflow where each branch lives in its own directory with full IDE support.

## Requirements

### Validated

**Output polish (v1.5):**

- ✓ Spinner API with StopWithSuccess/StopWithError — v1.5
- ✓ Spinners for long-running operations (list, clone, doctor, prune) — v1.5
- ✓ Real-time hook output streaming during grove add — v1.5
- ✓ Consistent success/error message format — v1.5
- ✓ Remove command shows full path clearly — v1.5
- ✓ Actionable error messages with suggestions — v1.5

**Fetch command (v1.4):**

- ✓ Command runs from anywhere in workspace — v1.4
- ✓ Fetches all configured remotes with automatic pruning — v1.4
- ✓ Shows new, updated, and pruned refs per remote — v1.4
- ✓ Human-readable and JSON output modes — v1.4
- ✓ `--verbose` flag shows commit hash details — v1.4

**Core commands (pre-v1.4):**

- ✓ `grove new` creates worktrees from branches
- ✓ `grove list` shows all worktrees with status
- ✓ `grove remove` cleans up worktrees
- ✓ `grove doctor` diagnoses workspace issues
- ✓ `grove status` shows repository state

### Active

(None — ready for next milestone planning)

### Out of Scope

- GUI or TUI interface — CLI-first
- Git replacement — wraps git, doesn't replace it
- Remote hosting integration — pure local tool
- Interactive TUI — Grove is CLI-first
- Custom spinner animations — single style sufficient
- Color-only semantic information — accessibility concern

## Context

Grove is at v1.5 with 25,245 lines of Go. The codebase follows standard Go CLI patterns with Cobra for commands. The logger package provides consistent output with spinner support, plain mode compliance, and styled messages.

**Tech stack:** Go 1.21+, Cobra CLI, git 2.48+

**Recent issues resolved:**

- #44: Hook output now streams in real-time
- #68: grove remove shows full path of deleted worktree

## Key Decisions

| Decision                  | Rationale                                        | Outcome |
| ------------------------- | ------------------------------------------------ | ------- |
| Bare repository workflow  | Clean separation of worktrees, no pollution      | ✓ Good  |
| Cobra CLI framework       | Standard Go CLI tooling, good completion support | ✓ Good  |
| Internal logger package   | Consistent formatting, spinner support           | ✓ Good  |
| Spinner returns \*Spinner | Enables contextual stop methods (success/error)  | ✓ Good  |
| Hook output to stderr     | Keeps stdout clean for shell wrapper cd paths    | ✓ Good  |
| Multiline error hints     | Clear separation between error and suggestion    | ✓ Good  |

## Constraints

- **Tech stack**: Go 1.21+, Cobra CLI
- **Compatibility**: git 2.48+
- **Pattern**: Follow existing command patterns in `internal/`
- **Plain mode**: All output must work in non-TTY environments

---

_Last updated: 2026-01-26 after v1.5 milestone_
