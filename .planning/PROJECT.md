# Grove

## What This Is

Grove is a CLI tool for managing git worktrees with a bare repository workflow. It simplifies creating, switching, and organizing worktrees while providing visibility into repository state across all branches.

## Core Value

Users get a clean, organized multi-branch workflow where each branch lives in its own directory with full IDE support.

## Current Milestone: v1.5 Output Polish

**Goal:** Consistent, polished output across all Grove commands with progress feedback and clear error messages.

**Target features:**

- Progress/streaming — Spinners for long operations, stream hook output
- Consistent messages — Unified success/error formats across all commands
- Better error UX — Input-to-output mapping, actionable errors, suppress noise

## Requirements

### Validated

**Fetch command (v1.4):**

- ✓ Command runs from anywhere in workspace
- ✓ Fetches all configured remotes with automatic pruning
- ✓ Shows new, updated, and pruned refs per remote
- ✓ Human-readable and JSON output modes
- ✓ `--verbose` flag shows commit hash details

**Core commands (pre-v1.4):**

- ✓ `grove new` creates worktrees from branches
- ✓ `grove list` shows all worktrees with status
- ✓ `grove remove` cleans up worktrees
- ✓ `grove doctor` diagnoses workspace issues
- ✓ `grove status` shows repository state

### Active

**Output polish (v1.5):**

- [ ] Spinners for long-running operations
- [ ] Stream hook output in real-time
- [ ] Consistent success/error message format
- [ ] Remove command maps input to output clearly
- [ ] Actionable error messages with suggestions
- [ ] Suppress noisy git output where appropriate

### Out of Scope

- GUI or TUI interface — CLI-first
- Git replacement — wraps git, doesn't replace it
- Remote hosting integration — pure local tool

## Context

Grove is a personal project for managing multiple worktrees efficiently. The codebase follows standard Go CLI patterns with Cobra for commands. Existing utilities include `logger.StartSpinner()`, `logger.Success/Error/Warning`, and a `formatter` package, but usage is inconsistent across commands.

**Known issues:**

- #68: `grove remove` output doesn't clearly show what was removed
- #44: Hook output isn't streamed, user sees nothing until completion

## Key Decisions

| Decision                 | Rationale                                        | Outcome   |
| ------------------------ | ------------------------------------------------ | --------- |
| Bare repository workflow | Clean separation of worktrees, no pollution      | ✓ Good    |
| Cobra CLI framework      | Standard Go CLI tooling, good completion support | ✓ Good    |
| Internal logger package  | Consistent formatting, spinner support           | — Pending |

## Constraints

- **Tech stack**: Go 1.21+, Cobra CLI
- **Compatibility**: git 2.48+
- **Pattern**: Follow existing command patterns in `internal/`

---

_Last updated: 2026-01-24 after v1.5 milestone start_
