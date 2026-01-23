# Requirements: Grove Fetch

**Defined:** 2026-01-23
**Core Value:** Users can see exactly what changed on their remotes in a single command

## v1 Requirements

### Core

- [ ] **CORE-01**: Command runs from anywhere in grove workspace
- [ ] **CORE-02**: Fetches all configured remotes
- [ ] **CORE-03**: Prunes stale remote-tracking refs by default
- [ ] **CORE-04**: Shows new refs (branches created on remote)
- [ ] **CORE-05**: Shows updated refs (branches moved to different commit)
- [ ] **CORE-06**: Shows pruned refs (branches deleted on remote)
- [ ] **CORE-07**: Skips remotes with no changes in output

### Output

- [ ] **OUT-01**: Human-readable output with clear labeling per remote
- [ ] **OUT-02**: `--json` flag for machine-readable JSON output
- [ ] **OUT-03**: `--verbose` flag shows additional details (commit info)

### CLI

- [ ] **CLI-01**: Shell completion support (no file completions, flags complete)

## v2 Requirements

### Enhanced Output

- **ENH-01**: `--quiet` flag suppresses all output
- **ENH-02**: `--dry-run` shows what would be fetched without fetching
- **ENH-03**: Per-worktree behind counts after fetch

## Out of Scope

| Feature                       | Reason                                       |
| ----------------------------- | -------------------------------------------- |
| Fetch specific remote only    | Keep command simple, fetch all               |
| Auto-pull after fetch         | Separate concern, users should control pulls |
| Progress indicators           | Git handles this natively                    |
| Parallel fetch implementation | Use git's `--jobs` if needed later           |
| Tag handling                  | Focus on branches for v1                     |

## Traceability

| Requirement | Phase   | Status  |
| ----------- | ------- | ------- |
| CORE-01     | Phase 1 | Pending |
| CORE-02     | Phase 1 | Pending |
| CORE-03     | Phase 1 | Pending |
| CORE-04     | Phase 1 | Pending |
| CORE-05     | Phase 1 | Pending |
| CORE-06     | Phase 1 | Pending |
| CORE-07     | Phase 1 | Pending |
| OUT-01      | Phase 1 | Pending |
| OUT-02      | Phase 2 | Pending |
| OUT-03      | Phase 2 | Pending |
| CLI-01      | Phase 1 | Pending |

**Coverage:**

- v1 requirements: 11 total
- Mapped to phases: 11
- Unmapped: 0

---

_Requirements defined: 2026-01-23_
_Last updated: 2026-01-23 after roadmap creation_
