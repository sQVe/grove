# Grove Fetch

## What This Is

A `grove fetch` command for Grove that fetches all remotes, prunes stale refs, and reports what changed. Users see exactly what happened on their remotes (new branches, updated branches, deleted branches) in a single command with optional JSON output for scripting.

## Core Value

Users can see exactly what changed on their remotes in a single command.

## Requirements

### Validated

- ✓ Command runs from anywhere in workspace — v1
- ✓ Fetches all configured remotes — v1
- ✓ Prunes stale remote-tracking refs by default — v1
- ✓ Shows new refs (branches that didn't exist before) — v1
- ✓ Shows updated refs (branches pointing to different commit) — v1
- ✓ Shows pruned refs (branches deleted on remote) — v1
- ✓ Skips remotes with no changes in output — v1
- ✓ Human-readable output with clear labeling per remote — v1
- ✓ `--json` flag for machine-readable output — v1
- ✓ `--verbose` flag shows commit hash details — v1
- ✓ Shell completion support — v1

### Active

- [ ] `--quiet` flag suppresses all output
- [ ] `--dry-run` shows what would be fetched without fetching
- [ ] Per-worktree behind counts after fetch

### Out of Scope

- Fetching specific remotes only — keep it simple, fetch all
- Progress indicators during fetch — git handles this
- Automatic worktree creation for new branches — separate concern
- Auto-pull after fetch — users should control pulls
- Tag handling — focus on branches

## Context

Shipped v1 with 1,189 LOC Go.
Tech stack: Go, Cobra CLI, existing internal packages.
Files: internal/git/fetch.go, cmd/grove/commands/fetch.go, integration tests.

## Key Decisions

| Decision                   | Rationale                                        | Outcome |
| -------------------------- | ------------------------------------------------ | ------- |
| "Updated" = commit changed | Simpler than tracking fast-forward vs force-push | ✓ Good  |
| Skip silent remotes        | Cleaner output, show only actionable info        | ✓ Good  |
| Fetch all remotes          | Keep command simple, single purpose              | ✓ Good  |
| Sorted output by ref name  | Deterministic results for testing                | ✓ Good  |
| Short hashes in verbose    | 7 chars balances uniqueness and readability      | ✓ Good  |
| omitempty JSON tags        | Cleaner machine output without nulls             | ✓ Good  |

## Constraints

- **Tech stack**: Go, Cobra CLI, existing internal packages
- **Compatibility**: Must work with git 2.48+
- **Pattern**: Follow existing command patterns (see `list.go`, `doctor.go`)

---

_Last updated: 2026-01-23 after v1 milestone_
