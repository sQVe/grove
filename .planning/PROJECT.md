# Grove Fetch

## What This Is

Adding a `grove fetch` command to Grove that fetches all remotes, prunes stale refs, and reports what changed. This gives users visibility into remote updates without manually running git commands.

## Core Value

Users can see exactly what changed on their remotes (new branches, updated branches, deleted branches) in a single command.

## Requirements

### Validated

(None yet — ship to validate)

### Active

- [ ] Command runs from anywhere in workspace
- [ ] Fetches all configured remotes
- [ ] Prunes stale remote-tracking refs by default
- [ ] Shows new refs (branches that didn't exist before)
- [ ] Shows updated refs (branches pointing to different commit)
- [ ] Shows pruned refs (branches deleted on remote)
- [ ] Supports `--json` flag for machine-readable output
- [ ] Skips remotes with no changes in output

### Out of Scope

- Fetching specific remotes only — keep it simple, fetch all
- Progress indicators during fetch — git handles this
- Automatic worktree creation for new branches — separate concern

## Context

Grove is an existing CLI tool for managing git worktrees. This adds a new command following established patterns:

- Commands in `cmd/grove/commands/`
- Git helpers in `internal/git/`
- Cobra-based CLI with `--json` flag pattern for machine output
- Script tests in `cmd/grove/testdata/script/`

## Constraints

- **Tech stack**: Go, Cobra CLI, existing internal packages
- **Compatibility**: Must work with git 2.48+
- **Pattern**: Follow existing command patterns (see `list.go`, `doctor.go`)

## Key Decisions

| Decision                   | Rationale                                        | Outcome   |
| -------------------------- | ------------------------------------------------ | --------- |
| "Updated" = commit changed | Simpler than tracking fast-forward vs force-push | — Pending |
| Skip silent remotes        | Cleaner output, show only actionable info        | — Pending |
| Fetch all remotes          | Keep command simple, single purpose              | — Pending |

---

_Last updated: 2026-01-23 after initialization_
