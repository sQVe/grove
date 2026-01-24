# Requirements Archive: v1 Grove Fetch

**Archived:** 2026-01-23
**Status:** ✅ SHIPPED

This is the archived requirements specification for v1.
For current requirements, see `.planning/REQUIREMENTS.md` (created for next milestone).

---

# Requirements: Grove Fetch

**Defined:** 2026-01-23
**Core Value:** Users can see exactly what changed on their remotes in a single command

## v1 Requirements

### Core

- [x] **CORE-01**: Command runs from anywhere in grove workspace
- [x] **CORE-02**: Fetches all configured remotes
- [x] **CORE-03**: Prunes stale remote-tracking refs by default
- [x] **CORE-04**: Shows new refs (branches created on remote)
- [x] **CORE-05**: Shows updated refs (branches moved to different commit)
- [x] **CORE-06**: Shows pruned refs (branches deleted on remote)
- [x] **CORE-07**: Skips remotes with no changes in output

### Output

- [x] **OUT-01**: Human-readable output with clear labeling per remote
- [x] **OUT-02**: `--json` flag for machine-readable JSON output
- [x] **OUT-03**: `--verbose` flag shows additional details (commit info)

### CLI

- [x] **CLI-01**: Shell completion support (no file completions, flags complete)

## Out of Scope

| Feature                       | Reason                                       |
| ----------------------------- | -------------------------------------------- |
| Fetch specific remote only    | Keep command simple, fetch all               |
| Auto-pull after fetch         | Separate concern, users should control pulls |
| Progress indicators           | Git handles this natively                    |
| Parallel fetch implementation | Use git's `--jobs` if needed later           |
| Tag handling                  | Focus on branches for v1                     |

## Traceability

| Requirement | Phase   | Status   |
| ----------- | ------- | -------- |
| CORE-01     | Phase 1 | Complete |
| CORE-02     | Phase 1 | Complete |
| CORE-03     | Phase 1 | Complete |
| CORE-04     | Phase 1 | Complete |
| CORE-05     | Phase 1 | Complete |
| CORE-06     | Phase 1 | Complete |
| CORE-07     | Phase 1 | Complete |
| OUT-01      | Phase 1 | Complete |
| OUT-02      | Phase 2 | Complete |
| OUT-03      | Phase 2 | Complete |
| CLI-01      | Phase 1 | Complete |

**Coverage:**

- v1 requirements: 11 total
- Shipped: 11
- Adjusted: 0
- Dropped: 0

---

## Milestone Summary

**Shipped:** 11 of 11 v1 requirements
**Adjusted:** None
**Dropped:** None

---

_Archived: 2026-01-23 as part of v1 milestone completion_
