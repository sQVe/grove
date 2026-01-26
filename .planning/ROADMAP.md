# Roadmap: Grove v1.5 Output Polish

## Overview

This milestone delivers consistent, polished output across all Grove commands. Starting with spinner API enhancements and streaming patterns, we sweep all commands for consistency, then add contextual error hints. The result is a CLI that feels responsive during operations and provides clear, actionable feedback.

## Milestones

- v1.4 Grove Fetch - Phases 1-2 (shipped)
- v1.5 Output Polish - Phases 3-6 (in progress)

## Phases

<details>
<summary>v1.4 Grove Fetch (Phases 1-2) - SHIPPED</summary>

Completed phases from previous milestone. See git history for details.

</details>

### v1.5 Output Polish (In Progress)

**Milestone Goal:** Users get consistent, polished output with progress feedback and clear error messages.

- [x] **Phase 3: Foundation** - Spinner API enhancements and patterns
- [x] **Phase 4: Hook Streaming** - Real-time hook output during grove add
- [x] **Phase 5: Output Consistency** - Sweep all commands for unified output
- [x] **Phase 6: Error Formatting** - Actionable error hints

## Phase Details

### Phase 3: Foundation

**Goal**: Spinner API provides the building blocks for all progress feedback
**Depends on**: Nothing (first phase of v1.5)
**Requirements**: SPIN-01, SPIN-02, SPIN-03, SPIN-04
**Success Criteria** (what must be TRUE):

1. Spinner can stop with success checkmark or error X indicator
2. Spinner message can be updated mid-operation without flicker
3. Multi-step operations display "Step N/M: action" format
4. Batch operations conclude with summary count ("Removed 3 worktrees")

**Plans:** 4 plans

Plans:

- [x] 03-01-PLAN.md — TDD: Spinner type with Update/Stop/StopWithSuccess/StopWithError
- [x] 03-02-PLAN.md — Migrate callers from func() to \*Spinner API
- [x] 03-03-PLAN.md — TDD: StepFormat helper for multi-step progress (gap closure)
- [x] 03-04-PLAN.md — Batch summary pattern for remove command (gap closure)

### Phase 4: Hook Streaming

**Goal**: Users see hook output in real-time during grove add (fixes #44)
**Depends on**: Phase 3 (spinner patterns established)
**Requirements**: STRM-01, STRM-02
**Success Criteria** (what must be TRUE):

1. Hook stdout/stderr streams to terminal as hooks execute
2. Each line of hook output shows which hook is running (prefix)
3. Spinner pauses cleanly during streaming, resumes after

**Plans:** 2 plans

Plans:

- [x] 04-01-PLAN.md — TDD: PrefixWriter type for line-by-line prefixed output
- [x] 04-02-PLAN.md — RunAddHooksStreaming function and add.go integration

### Phase 5: Output Consistency

**Goal**: All commands use consistent output patterns
**Depends on**: Phase 4 (streaming patterns may inform output rules)
**Requirements**: SPIN-05, SPIN-06, SPIN-07, SPIN-08, CLRT-01, CLRT-02, CLRT-03, CLRT-04, CLRT-05
**Success Criteria** (what must be TRUE):

1. All long-running commands show spinner during wait
2. Success messages use consistent past-tense verbs (Created, Deleted, Updated)
3. All user-facing output goes through logger package (no bare fmt.Print)
4. Empty state messages are consistent ("No worktrees found" pattern)
5. grove remove shows full path of deleted worktree (fixes #68)

**Plans:** 3 plans

Plans:

- [x] 05-01-PLAN.md — Add spinners to list/clone/doctor commands (SPIN-05, SPIN-06, SPIN-07)
- [x] 05-02-PLAN.md — Remove command output: full path display + spinner (CLRT-01, SPIN-08)
- [x] 05-03-PLAN.md — Audit fmt.Print for user messages, standardize empty states (CLRT-04, CLRT-05)

### Phase 6: Error Formatting

**Goal**: Error messages include actionable hints for common issues
**Depends on**: Phase 5 (consistent output patterns required)
**Requirements**: HINT-01, HINT-02, HINT-03, HINT-04
**Success Criteria** (what must be TRUE):

1. "Worktree already exists" error suggests using existing or different name
2. "Cannot delete current worktree" error suggests switching first
3. "Already locked" error suggests unlock command
4. "Cannot rename current worktree" error suggests switch command

**Plans:** 1 plan

Plans:

- [x] 06-01-PLAN.md — Add actionable hints to error messages (HINT-01, HINT-02, HINT-03, HINT-04)

## Progress

**Execution Order:** 3 -> 4 -> 5 -> 6

| Phase                 | Milestone | Plans Complete | Status   | Completed  |
| --------------------- | --------- | -------------- | -------- | ---------- |
| 3. Foundation         | v1.5      | 4/4            | Complete | 2026-01-24 |
| 4. Hook Streaming     | v1.5      | 2/2            | Complete | 2026-01-26 |
| 5. Output Consistency | v1.5      | 3/3            | Complete | 2026-01-26 |
| 6. Error Formatting   | v1.5      | 1/1            | Complete | 2026-01-26 |

---

_Roadmap created: 2026-01-24_
_Last updated: 2026-01-26 — Milestone complete_
