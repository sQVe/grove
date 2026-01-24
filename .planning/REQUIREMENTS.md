# Requirements: Grove v1.5 Output Polish

**Defined:** 2026-01-24
**Core Value:** Users get consistent, polished output with progress feedback and clear error messages

## v1.5 Requirements

Requirements for this milestone. Each maps to roadmap phases.

### Spinners

- [ ] **SPIN-01**: Spinner API provides StopWithSuccess() and StopWithError() methods
- [ ] **SPIN-02**: Spinner API provides Update() method to change message mid-operation
- [ ] **SPIN-03**: Multi-step operations show "Step N/M: action" format
- [ ] **SPIN-04**: Batch operations show summary ("Removed 3 worktrees")
- [ ] **SPIN-05**: `grove list` shows spinner while gathering worktree status
- [ ] **SPIN-06**: `grove clone` shows spinner during clone operation
- [ ] **SPIN-07**: `grove doctor` shows spinner during remote reachability checks
- [ ] **SPIN-08**: `grove prune` shows spinner during fetch operation

### Streaming

- [ ] **STRM-01**: Hook output streams in real-time during `grove add`
- [ ] **STRM-02**: Hook output includes prefix identifying which hook is running

### Output Clarity

- [ ] **CLRT-01**: `grove remove` shows full path of deleted worktree
- [ ] **CLRT-02**: All commands use consistent past-tense verbs (Created, Deleted, Updated)
- [ ] **CLRT-03**: Error messages include actionable hints where applicable
- [ ] **CLRT-04**: Commands use logger package consistently (no bare fmt.Print for user output)
- [ ] **CLRT-05**: Commands show consistent empty state messages when no results

### Error Hints

- [ ] **HINT-01**: "worktree already exists" suggests using existing or different name
- [ ] **HINT-02**: "cannot delete current worktree" suggests switching first
- [ ] **HINT-03**: "already locked" suggests unlock command
- [ ] **HINT-04**: "cannot rename current worktree" suggests switch command

## Future Requirements

Deferred to later milestones. Tracked but not in current roadmap.

### Extended Machine Output

- **JSON-01**: `grove list` supports --json flag
- **JSON-02**: `grove status` supports --json flag
- **JSON-03**: `grove doctor` supports --json flag

### Advanced Progress

- **PROG-01**: Time estimates for long operations
- **PROG-02**: Progress bars for known-length operations

### Additional Polish

- **PLSH-01**: `--dry-run` flag for destructive commands (remove, exec)
- **PLSH-02**: Specialized logger.Hint() method with distinct styling
- **PLSH-03**: Batch operation intermediate success logging

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature                         | Reason                                      |
| ------------------------------- | ------------------------------------------- |
| Interactive TUI                 | Grove is CLI-first, TUI is separate concern |
| Custom spinner animations       | Single spinner style is sufficient          |
| Color-only semantic information | Accessibility concern — icons required      |
| Verbose "helpful" messages      | Keep output minimal and purposeful          |
| --json on all commands          | Focus on UX first, machine output later     |
| Concurrent spinner display      | Single spinner at a time is sufficient      |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status  |
| ----------- | ----- | ------- |
| SPIN-01     | 3     | Pending |
| SPIN-02     | 3     | Pending |
| SPIN-03     | 3     | Pending |
| SPIN-04     | 3     | Pending |
| SPIN-05     | 5     | Pending |
| SPIN-06     | 5     | Pending |
| SPIN-07     | 5     | Pending |
| SPIN-08     | 5     | Pending |
| STRM-01     | 4     | Pending |
| STRM-02     | 4     | Pending |
| CLRT-01     | 5     | Pending |
| CLRT-02     | 5     | Pending |
| CLRT-03     | 5     | Pending |
| CLRT-04     | 5     | Pending |
| CLRT-05     | 5     | Pending |
| HINT-01     | 6     | Pending |
| HINT-02     | 6     | Pending |
| HINT-03     | 6     | Pending |
| HINT-04     | 6     | Pending |

**Coverage:**

- v1.5 requirements: 19 total
- Mapped to phases: 19
- Unmapped: 0

---

_Requirements defined: 2026-01-24_
_Last updated: 2026-01-24 — Traceability updated after roadmap creation_
