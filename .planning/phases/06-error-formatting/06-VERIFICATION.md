---
phase: 06-error-formatting
verified: 2026-01-26T13:22:59Z
status: passed
score: 4/4 must-haves verified
---

# Phase 6: Error Formatting Verification Report

**Phase Goal:** Error messages include actionable hints for common issues
**Verified:** 2026-01-26T13:22:59Z
**Status:** passed
**Re-verification:** No — initial verification

## Goal Achievement

### Observable Truths

| #   | Truth                                                                     | Status     | Evidence                                                                                            |
| --- | ------------------------------------------------------------------------- | ---------- | --------------------------------------------------------------------------------------------------- |
| 1   | "Worktree already exists" error suggests using existing or different name | ✓ VERIFIED | Lines 207 and 355 in add.go contain multiline error with "grove list" and "--name" hints            |
| 2   | "Cannot delete current worktree" error suggests switching first           | ✓ VERIFIED | Line 115 in remove.go contains multiline error with "grove switch" hint                             |
| 3   | "Already locked" error suggests unlock command                            | ✓ VERIFIED | Lines 96 and 98 in lock.go contain multiline error with "grove unlock" hint including worktree name |
| 4   | "Cannot rename current worktree" error suggests switch command            | ✓ VERIFIED | Line 73 in move.go contains multiline error with "grove switch" hint                                |

**Score:** 4/4 truths verified

### Required Artifacts

| Artifact                            | Expected                            | Status     | Details                                                              |
| ----------------------------------- | ----------------------------------- | ---------- | -------------------------------------------------------------------- |
| `cmd/grove/commands/add.go`         | HINT-01: worktree exists hint       | ✓ VERIFIED | Lines 207 and 355 contain "grove list" and "--name" in error message |
| `cmd/grove/commands/remove.go`      | HINT-02: cannot delete current hint | ✓ VERIFIED | Line 115 contains "grove switch <worktree>" in error message         |
| `cmd/grove/commands/lock.go`        | HINT-03: already locked hint        | ✓ VERIFIED | Lines 96-98 contain "grove unlock %s" with specific worktree name    |
| `cmd/grove/commands/move.go`        | HINT-04: cannot rename current hint | ✓ VERIFIED | Line 73 contains "grove switch <worktree>" in error message          |
| `cmd/grove/commands/add_test.go`    | Test for HINT-01                    | ✓ VERIFIED | Lines 742-747 verify "grove list" and "--name" appear in error       |
| `cmd/grove/commands/remove_test.go` | Test for HINT-02                    | ✓ VERIFIED | Lines 120-149 test cannot delete current worktree                    |
| `cmd/grove/commands/lock_test.go`   | Test for HINT-03                    | ✓ VERIFIED | Lines 132-173 test already locked error                              |
| `cmd/grove/commands/move_test.go`   | Test for HINT-04                    | ✓ VERIFIED | Lines 82-99 verify hint for cannot rename current                    |

### Key Link Verification

| From                    | To                | Via                              | Status  | Details                                             |
| ----------------------- | ----------------- | -------------------------------- | ------- | --------------------------------------------------- |
| add.go error messages   | User output       | fmt.Errorf with multiline hint   | ✓ WIRED | Lines 207, 355: `\n\nHint:` format used             |
| remove.go error message | User output       | logger.Error with multiline hint | ✓ WIRED | Line 115: `\n\nHint:` format used                   |
| lock.go error messages  | User output       | logger.Error with multiline hint | ✓ WIRED | Lines 96, 98: `\n\nHint:` format with worktree name |
| move.go error message   | User output       | fmt.Errorf with multiline hint   | ✓ WIRED | Line 73: `\n\nHint:` format used                    |
| Test files              | Hint verification | String contains checks           | ✓ WIRED | Tests verify hint text appears in errors            |

### Requirements Coverage

| Requirement | Status      | Evidence                                  |
| ----------- | ----------- | ----------------------------------------- |
| HINT-01     | ✓ SATISFIED | add.go lines 207, 355 + test line 742     |
| HINT-02     | ✓ SATISFIED | remove.go line 115 + test lines 120-149   |
| HINT-03     | ✓ SATISFIED | lock.go lines 96, 98 + test lines 132-173 |
| HINT-04     | ✓ SATISFIED | move.go line 73 + test lines 82-99        |
| CLRT-03     | ✓ SATISFIED | All four hints implemented consistently   |

### Anti-Patterns Found

None detected. All error hints:

- Follow consistent `\n\nHint:` format
- Include specific command examples (grove list, grove unlock, grove switch)
- Are substantive and actionable
- Are tested

### Verification Details

**Truth 1: "Worktree already exists" error suggests using existing or different name**

- **File:** cmd/grove/commands/add.go
- **Lines 207, 355:** Error message includes:
    ```go
    "worktree already exists for branch %q at %s\n\nHint: Use 'grove list' to see existing worktrees, or use --name to choose a different directory"
    ```
- **Test:** add_test.go lines 742-747 verify both "grove list" and "--name" appear
- **Status:** ✓ Full implementation with test coverage

**Truth 2: "Cannot delete current worktree" error suggests switching first**

- **File:** cmd/grove/commands/remove.go
- **Line 115:** Error message includes:
    ```go
    "cannot delete current worktree\n\nHint: Switch to a different worktree first with 'grove switch <worktree>'"
    ```
- **Test:** remove_test.go lines 120-149 test the error scenario
- **Status:** ✓ Full implementation with test coverage

**Truth 3: "Already locked" error suggests unlock command**

- **File:** cmd/grove/commands/lock.go
- **Lines 96, 98:** Two variants (with/without reason), both include:
    ```go
    "already locked\n\nHint: Use 'grove unlock %s' to remove the lock"
    ```
- **Test:** lock_test.go lines 132-173 test already locked scenario
- **Status:** ✓ Full implementation with test coverage, includes worktree name for copy-paste

**Truth 4: "Cannot rename current worktree" error suggests switch command**

- **File:** cmd/grove/commands/move.go
- **Line 73:** Error message includes:
    ```go
    "cannot rename current worktree\n\nHint: Switch to a different worktree first with 'grove switch <worktree>'"
    ```
- **Test:** move_test.go lines 82-99 verify the hint exists
- **Status:** ✓ Full implementation with test coverage

### Pattern Consistency

All hints follow the established pattern:

- **Format:** `\n\nHint: [actionable suggestion]`
- **Content:** Specific command with example (`'grove list'`, `'grove unlock %s'`, `'grove switch <worktree>'`)
- **Placement:** Immediately after the error description
- **Testing:** Each hint has corresponding test verification

### Commits

Implementation delivered in three atomic commits:

1. **06c2544** - feat(06-01): add hints to add.go and remove.go errors
2. **4855c21** - feat(06-01): add hints to lock.go and move.go errors
3. **8d168fd** - test(06-01): add tests for error hints

---

_Verified: 2026-01-26T13:22:59Z_
_Verifier: Claude (gsd-verifier)_
