# Project Research Summary

**Project:** Grove v1.5 Output Polish
**Domain:** CLI output UX for Go-based worktree manager
**Researched:** 2026-01-24
**Confidence:** HIGH

## Executive Summary

Grove v1.5 output polish is a low-risk, high-impact milestone that requires no new dependencies. The existing stack (lipgloss v1.1.0, termenv, custom spinner, stdlib os/exec) is sufficient. Research confirms the codebase already has the right foundations: logger package with plain mode support, styles with themeable colors, formatter for domain-specific output. The work is refactoring and consistency, not greenfield development.

The recommended approach is extend-not-replace. Extract the spinner to its own file with new methods (StopWithSuccess, StopWithError, Update), add a stream writer for hook output, then sweep all commands for output consistency. The two open issues (#68 remove output clarity, #44 hook streaming) fit naturally into this plan and serve as validation targets.

Key risks are breaking scripted usage and CI pipelines. The existing `config.IsPlain()` pattern prevents most issues, but any new output must use it. Testing in non-TTY environments before each phase completes is essential. The phased approach (foundation, streaming, consistency, errors) isolates risk and allows incremental delivery.

## Key Findings

### Recommended Stack

No new dependencies needed. The existing stack is current and sufficient.

**Core technologies (keep as-is):**

- **lipgloss v1.1.0:** Styling and colors — stay on v1, v2 beta offers no value for CLI output
- **termenv v0.16.0:** Terminal detection — already handles TTY checks
- **Custom spinner:** Progress indication — enhance in-place, 50 lines vs external dependency
- **os/exec stdlib:** Subprocess execution — streaming patterns are 20 lines of stdlib

**Explicit non-recommendations:**

- bubbletea/bubbles: TUI frameworks, overkill for CLI
- briandowns/spinner: 90+ styles when Grove needs one
- go-cmd/cmd: Streaming wrapper for something stdlib handles

### Expected Features

**Must have (table stakes):**

- Spinner for long operations — exists but underused, needs extension
- Clear success/error indicators — exists, needs consistent usage
- Show what was affected — missing, causes #68
- Stream subprocess output — missing, causes #44
- Plain mode for scripts — exists, works correctly

**Should have (differentiators):**

- Consistent message vocabulary — partial, needs audit
- Multi-step operation feedback — would help grove add with hooks
- Summary after batch operations — "Removed 3 worktrees"
- Contextual hints — "Use --force to remove dirty worktree"

**Defer (v2+):**

- Time estimates for long ops — rarely worth complexity
- Progress bars for known-length ops — Grove operations are indeterminate
- Extended --json to all commands — lower priority than UX fixes

### Architecture Approach

Extend internal/logger/, don't create parallel abstractions. Keep output in command layer, not business logic. Use stderr for feedback (spinners, success, errors) and stdout for data (list output, JSON). The existing formatter package is domain-specific and should stay that way.

**Component changes:**

1. **internal/logger/spinner.go** (extract) — add StopWithSuccess/StopWithError/Update methods
2. **internal/logger/stream.go** (new) — StreamCommand for real-time hook output with prefix
3. **cmd/grove/commands/\*.go** (audit) — standardize logger usage, eliminate bare fmt.Print

### Critical Pitfalls

1. **Breaking scripted usage with spinners** — Always check isatty(), test with pipes, ensure --plain disables all animation
2. **Stdout/stderr confusion during migration** — Data to stdout, feedback to stderr, never change existing stream destinations
3. **Blocking output during long operations** — Stream hook output in real-time, use spinner for opaque operations
4. **Input-output mapping confusion (#68)** — Echo user input in output, show both directory name and branch when they differ
5. **Breaking --plain mode** — Test every change in both modes, plain mode must produce pure ASCII

## Implications for Roadmap

Based on research, suggested phase structure:

### Phase 1: Foundation

**Rationale:** Establish patterns before adding features. The spinner extraction and output audit de-risk later phases.
**Delivers:** Enhanced spinner API, documented output rules, consistency audit
**Addresses:** Consistent message vocabulary, plain mode compliance
**Avoids:** Breaking existing functionality, inconsistent patterns proliferating

### Phase 2: Hook Streaming

**Rationale:** Fixes the most painful UX issue (#44). Requires spinner to be stable first.
**Delivers:** Real-time hook output via StreamWriter, better grove add experience
**Addresses:** Stream subprocess output feature, blocking output pitfall
**Avoids:** Spinner lifecycle bugs by building on Phase 1 patterns

### Phase 3: Output Consistency

**Rationale:** Apply consistent patterns across all commands. Depends on Phase 1 patterns being established.
**Delivers:** Unified output format, all commands using logger properly
**Addresses:** Consistent message vocabulary, doctor.go and fetch.go standardization
**Avoids:** Piecemeal fixes that create new inconsistencies

### Phase 4: Error Formatting

**Rationale:** Polish layer on top of consistent output. Includes #68 fix.
**Delivers:** Input-to-output mapping, actionable errors, suppressed noise
**Addresses:** Remove command clarity (#68), contextual hints, verbose path warnings
**Avoids:** Input-output mapping confusion, verbose error paths

### Phase Ordering Rationale

- Phase 1 before Phase 2: Spinner extraction must stabilize before hook streaming adds complexity
- Phase 2 before Phase 3: Streaming implementation may reveal new output patterns to codify
- Phase 3 before Phase 4: Consistent output format required before adding hints and enhanced errors
- All phases test plain mode: Each phase must pass non-TTY tests before completion

### Research Flags

**Phases with standard patterns (skip research-phase):**

- **Phase 1:** Logger enhancement is internal refactoring, patterns well understood
- **Phase 3:** Consistency sweep is audit-driven, no new research needed
- **Phase 4:** Error formatting follows established patterns from Phase 1

**Phases potentially needing deeper research:**

- **Phase 2:** Hook streaming may need investigation if subprocess buffering causes issues (scanner line limits, stderr/stdout ordering). Monitor during implementation.

## Confidence Assessment

| Area         | Confidence | Notes                                                    |
| ------------ | ---------- | -------------------------------------------------------- |
| Stack        | HIGH       | Verified against codebase and official releases          |
| Features     | HIGH       | Based on clig.dev, GitHub CLI patterns, existing issues  |
| Architecture | HIGH       | Based entirely on codebase analysis                      |
| Pitfalls     | HIGH       | Real issues from community and Grove's own issue tracker |

**Overall confidence:** HIGH

### Gaps to Address

- **Test coverage for plain mode:** Current tests may not cover --plain adequately. Add plain mode assertions during Phase 1.
- **Spinner goroutine cleanup:** Existing implementation uses 10ms sleep for cleanup. Monitor for race conditions when adding more spinners in Phase 2.
- **Backward compatibility for parsed output:** No formal contract exists for stdout format. Document during Phase 1 that stderr is for humans, --json is for machines.

## Sources

### Primary (HIGH confidence)

- Grove codebase: internal/logger/logger.go, internal/styles/styles.go, cmd/grove/commands/\*.go
- [Command Line Interface Guidelines](https://clig.dev/)
- [GitHub CLI Accessibility](https://github.blog/engineering/user-experience/building-a-more-accessible-github-cli/)

### Secondary (MEDIUM confidence)

- [Heroku CLI Style Guide](https://devcenter.heroku.com/articles/cli-style-guide)
- [CLI UX Progress Patterns](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays)
- [lipgloss releases](https://github.com/charmbracelet/lipgloss/releases)

### Issue-specific

- [Grove Issue #68](https://github.com/sqve/grove/issues/68) — Remove command output clarity
- [Grove Issue #44](https://github.com/sqve/grove/issues/44) — Hook output streaming
- [Salesforce CLI TTY Issue](https://github.com/forcedotcom/cli/issues/327) — Progress bar in non-TTY
- [golang-migrate stderr Issue](https://github.com/golang-migrate/migrate/issues/363) — Stream confusion

---

_Research completed: 2026-01-24_
_Ready for roadmap: yes_
