# Roadmap: Grove Fetch

## Overview

Deliver `grove fetch` in two phases: first a working command with human-readable output showing what changed on remotes, then output mode enhancements for scripting and debugging. TDD is required throughout - tests first, implementation second.

## Phases

**Phase Numbering:**

- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [x] **Phase 1: Core Fetch** - Fetch all remotes and show what changed
- [ ] **Phase 2: Output Modes** - Add JSON and verbose output options

## Phase Details

### Phase 1: Core Fetch

**Goal**: Users can fetch all remotes and see what changed (new, updated, pruned refs)
**Depends on**: Nothing (first phase)
**Requirements**: CORE-01, CORE-02, CORE-03, CORE-04, CORE-05, CORE-06, CORE-07, OUT-01, CLI-01
**Success Criteria** (what must be TRUE):

1. User can run `grove fetch` from any directory within a grove workspace
2. Command fetches all configured remotes and prunes stale refs automatically
3. Output clearly shows new branches, updated branches, and pruned refs per remote
4. Remotes with no changes are omitted from output
5. Shell completion works (no file completions, flags complete)
6. All tests pass (TDD: tests written before implementation)

**Plans:** 3 plans in 3 waves

Plans:

- [x] 01-01-PLAN.md - Git fetch operations with TDD (ref snapshot, change detection)
- [x] 01-02-PLAN.md - Fetch command implementation (CLI, output formatting)
- [x] 01-03-PLAN.md - Integration tests (testscript for end-to-end verification)

### Phase 2: Output Modes

**Goal**: Users can get machine-readable output and additional details
**Depends on**: Phase 1
**Requirements**: OUT-02, OUT-03
**Success Criteria** (what must be TRUE):

1. User can run `grove fetch --json` to get structured JSON output
2. User can run `grove fetch --verbose` to see additional commit details
3. Both flags work correctly with empty results (no changes)
4. All tests pass (TDD: tests written before implementation)

**Plans:** 1 plan in 1 wave

Plans:

- [ ] 02-01-PLAN.md - Add --json and --verbose output modes

## Progress

**Execution Order:**
Phases execute in numeric order: 1 -> 2

| Phase           | Plans Complete | Status      | Completed  |
| --------------- | -------------- | ----------- | ---------- |
| 1. Core Fetch   | 3/3            | Complete    | 2026-01-23 |
| 2. Output Modes | 0/1            | Not started | -          |
