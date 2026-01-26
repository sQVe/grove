---
phase: 06-error-formatting
plan: 01
subsystem: cli
tags: [error-messages, user-experience, hints, golang]

# Dependency graph
requires:
    - phase: 05-output-consistency
      provides: logger API and consistent output patterns
provides:
    - Actionable hints for four common error scenarios
    - Consistent hint format across all commands
affects: [future error messages, documentation]

# Tech tracking
tech-stack:
    added: []
    patterns: ['Multiline error format with Hint section']

key-files:
    created: []
    modified:
        - cmd/grove/commands/add.go
        - cmd/grove/commands/remove.go
        - cmd/grove/commands/lock.go
        - cmd/grove/commands/move.go

key-decisions:
    - "Use multiline error format with \\n\\nHint: prefix for consistent presentation"
    - 'Include specific command examples in hints (grove list, grove unlock, grove switch)'

patterns-established:
    - "Error hints pattern: error message \\n\\nHint: suggested command to resolve"

# Metrics
duration: 3min
completed: 2026-01-26
---

# Phase 6 Plan 01: Error Formatting Summary

**Actionable hints added to four common error messages guiding users to grove list, grove unlock, and grove switch commands**

## Performance

- **Duration:** 3 min
- **Started:** 2026-01-26T13:17:03Z
- **Completed:** 2026-01-26T13:20:11Z
- **Tasks:** 3
- **Files modified:** 8

## Accomplishments

- Added hints to "worktree already exists" errors suggesting grove list and --name flag
- Added hint to "cannot delete current worktree" error suggesting grove switch
- Added hint to "already locked" error suggesting grove unlock with specific worktree name
- Added hint to "cannot rename current worktree" error suggesting grove switch

## Task Commits

Each task was committed atomically:

1. **Task 1: Add hints to add.go and remove.go errors** - `06c2544` (feat)
2. **Task 2: Add hints to lock.go and move.go errors** - `4855c21` (feat)
3. **Task 3: Add tests for error hints** - `8d168fd` (test)

## Files Created/Modified

- `cmd/grove/commands/add.go` - Added hint to worktree exists error (lines 207, 355)
- `cmd/grove/commands/remove.go` - Added hint to cannot delete current worktree error (line 115)
- `cmd/grove/commands/lock.go` - Added hints to already locked errors (lines 96-98)
- `cmd/grove/commands/move.go` - Added hint to cannot rename current worktree error (line 73)
- `cmd/grove/commands/add_test.go` - Added test verifying grove list and --name hints appear
- `cmd/grove/commands/remove_test.go` - Added test verifying grove switch hint appears
- `cmd/grove/commands/lock_test.go` - Added test verifying grove unlock hint appears
- `cmd/grove/commands/move_test.go` - Added test verifying grove switch hint appears

## Decisions Made

- Use multiline format with `\n\nHint:` separator for clear visual distinction between error and suggestion
- Include specific command examples rather than generic descriptions
- For lock.go, include the worktree name in the unlock hint for direct copy-paste usability

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

- Error formatting improvements complete
- All hints follow consistent pattern
- Tests verify hints appear in error output
- Ready for future error message enhancements

---

_Phase: 06-error-formatting_
_Completed: 2026-01-26_
