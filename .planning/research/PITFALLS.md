# CLI output polish pitfalls

**Domain:** Adding output polish to existing Go CLI
**Researched:** 2026-01-24
**Context:** Grove CLI with existing logger/styles/formatter packages

## Critical pitfalls

Mistakes that cause rewrites or break existing functionality.

### Pitfall 1: Breaking scripted usage when adding spinners

**What goes wrong:** Adding spinners or progress indicators to commands that users pipe or parse breaks their automation. The escape sequences for clearing lines corrupt output when redirected to files or piped to other tools.

**Why it happens:** Developers test interactively but forget that CLI output is consumed by scripts, CI pipelines, and other tools. The [Salesforce CLI issue](https://github.com/forcedotcom/cli/issues/327) documents how progress bars in non-TTY environments produce garbage output.

**Consequences:**

- Scripts that parse output start failing silently
- CI pipelines interpret any stderr as errors
- Users redirect to files and get ANSI escape sequences instead of text

**Warning signs:**

- New spinners appear in commands that previously had parseable output
- Tests run in non-TTY environments start producing different output
- Users report issues in CI/CD contexts

**Prevention:**

1. Always check `isatty()` before displaying spinners. Grove's `StartSpinner()` does this correctly via `isPlain()`, but any new output polish must follow the same pattern
2. Test commands with pipes: `grove list | head -1` should produce clean output
3. Ensure `--plain` flag disables all motion/animation, not just colors
4. Use `--json` output for truly machine-parseable cases (already present in list, fetch, doctor)

**Which phase should address:** Phase 1 (Foundation) - establish TTY detection as a hard requirement for any animated output.

**Grove-specific notes:** The existing `config.IsPlain()` and `logger.isPlain()` checks are the right pattern. New output features must use this check. The `StartSpinner()` implementation already handles this correctly.

---

### Pitfall 2: Stdout/stderr confusion during migration

**What goes wrong:** Mixing primary output with status messages on the same stream, or changing which stream a command uses mid-migration, breaks downstream tooling.

**Why it happens:** The [golang-migrate issue](https://github.com/golang-migrate/migrate/issues/363) shows how putting all output to stderr causes tools like Helm to treat successful runs as failures.

**Consequences:**

- Scripts fail because they can't distinguish success output from error output
- Piping breaks because decorative output pollutes the data stream
- Users can't redirect errors separately from results

**Warning signs:**

- Commands use `fmt.Println()` for both data and status messages
- Inconsistent usage across commands (some use logger, some print directly)
- `--json` output mixed with status messages

**Prevention:**

1. Primary data goes to stdout (worktree paths, JSON output, things users pipe)
2. Status/progress/error messages go to stderr (via logger package)
3. Document the contract: what goes where and why
4. Never change an existing stream destination without a major version bump

**Which phase should address:** Phase 1 - audit existing commands, document current behavior, establish consistent rules.

**Grove-specific notes:** Current inconsistency examples:

- `list.go:183` uses `fmt.Println()` to stdout (correct for data)
- `add.go:265` uses `fmt.Println(worktreePath)` for `--switch` (correct)
- `remove.go` uses `logger.Success/Error` to stderr (correct for status)
- `fetch.go:191` mixes `fmt.Printf` to stdout with `logger.Error` to stderr

The existing pattern is mostly correct but should be explicitly documented.

---

### Pitfall 3: Blocking output during long operations

**What goes wrong:** Users see no feedback during long-running operations, assume the tool is frozen, and ctrl+C mid-operation causing partial state.

**Why it happens:** Output is buffered until operation completes. The [hook streaming issue #44](https://github.com/sqve/grove/issues/44) describes exactly this: `pnpm install` runs for 30+ seconds with no visible progress.

**Consequences:**

- Users interrupt operations, leaving worktrees in broken states
- Support burden increases ("is it frozen?")
- Poor perception of tool quality

**Warning signs:**

- `cmd.Run()` with captured stdout/stderr buffers
- No spinner or progress indicator for operations > 2 seconds
- Tests mock away the slow parts, hiding the UX problem

**Prevention:**

1. Any operation that can exceed 2 seconds needs visible feedback
2. For subprocess output: stream in real-time when possible
3. For opaque operations: use spinner with descriptive message
4. Capture output for error diagnostics while also streaming

**Which phase should address:** Phase 2 (Streaming) - implement hybrid streaming for hooks.

**Grove-specific notes:** The `hooks.go` implementation captures to buffers:

```go
var stdout, stderr bytes.Buffer
cmd.Stdout = &stdout
cmd.Stderr = &stderr
```

Fix requires streaming to stderr while also capturing for failure diagnostics. The `exec` command already does this correctly with `cmd.Stdout = os.Stdout`.

---

## Moderate pitfalls

Mistakes that cause confusion or technical debt.

### Pitfall 4: Input-output mapping confusion

**What goes wrong:** When processing multiple items, output doesn't clearly map back to input, leaving users confused about what happened.

**Why it happens:** Developers use internal identifiers (branch names) instead of user-provided identifiers (directory names) in output messages.

**Consequences:**

- Users can't tell which of their inputs succeeded or failed
- Error recovery requires manual investigation
- Trust in the tool decreases

**Warning signs:**

- Output mentions identifiers the user never typed
- Batch operations report counts without item-level detail
- Errors don't reference the original input

**Prevention:**

1. Echo user input in output: "Deleted worktree pr-1641 (branch: d9f2fba7)"
2. When names differ, show both
3. Report on each input, not just successful operations
4. Group related output by input item

**Which phase should address:** Phase 3 (Error Formatting) - fix remove command, apply pattern to other batch operations.

**Grove-specific notes:** [Issue #68](https://github.com/sqve/grove/issues/68) describes exactly this. The remove command accepts `pr-1641` but outputs `d9f2fba7` because it always uses `info.Branch`. Fix:

```go
// When directory name differs from branch, show both
name := filepath.Base(info.Path)
if name != info.Branch {
    logger.Success("Deleted worktree %s (branch: %s)", name, info.Branch)
} else {
    logger.Success("Deleted worktree %s", info.Branch)
}
```

---

### Pitfall 5: Inconsistent output formatting across commands

**What goes wrong:** Each command uses different styles for success/error/warning messages, different indentation levels, different icon sets. Users can't build mental models.

**Why it happens:** Commands added at different times by different contributors without a style guide. No shared components for common output patterns.

**Consequences:**

- Cognitive load increases for users
- Documentation becomes verbose trying to explain variations
- Maintenance burden as fixes must be applied inconsistently

**Warning signs:**

- Multiple ways to show a list (some use `logger.ListItemWithNote`, some use `fmt.Printf`)
- Inconsistent indentation (2 spaces vs 4 spaces vs tabs)
- Different success indicators ("Created" vs "Added" vs just showing the result)

**Prevention:**

1. Create shared formatter functions for common patterns
2. Document output style guidelines
3. Audit existing commands for consistency before adding polish

**Which phase should address:** Phase 1 (Foundation) - audit and document, then fix before adding new features.

**Grove-specific notes:** The existing `formatter` package is good but underused. Current patterns:

- `logger.Success/Error/Warning/Info` - consistent prefixes
- `logger.ListItemWithNote` - for list items with metadata
- `logger.ListSubItem` - for nested details
- `formatter.WorktreeRow` - specialized for worktree lists

Commands should use these consistently. The `doctor.go` implements its own `getIssueSymbol()` rather than reusing logger patterns.

---

### Pitfall 6: Spinner lifecycle bugs

**What goes wrong:** Spinner doesn't stop on error, multiple spinners overlap, or spinner cleanup races with program exit.

**Why it happens:** Error paths don't call the stop function. Goroutine timing issues cause visual artifacts.

**Consequences:**

- Terminal left in broken state
- Multiple spinners fight for the same line
- Last spinner frame remains visible after command exits

**Warning signs:**

- Deferred spinner stop without considering early returns
- No mutex protection for spinner state
- Sleep-based synchronization

**Prevention:**

1. Use `defer stopSpinner()` immediately after starting
2. Single active spinner at a time (mutex or state check)
3. Clear line on stop before any output
4. Add small delay after stop to let goroutine clean up

**Which phase should address:** Phase 2 (Streaming) - when adding more spinners, establish patterns.

**Grove-specific notes:** The existing `StartSpinner()` handles this reasonably:

- Uses `sync.Once` for safe shutdown
- Adds 10ms delay for cleanup
- Single spinner design (no queue/stack)

New usages should follow this pattern. The fetch command's spinner usage is correct:

```go
stopSpinner := logger.StartSpinner(fmt.Sprintf("Fetching %s...", remote))
// ... work ...
stopSpinner()
```

---

### Pitfall 7: Breaking --plain mode

**What goes wrong:** New features add visual elements that don't degrade gracefully in plain mode, making output unreadable.

**Why it happens:** Plain mode is tested as an afterthought. ANSI codes or Unicode characters slip through.

**Consequences:**

- CI logs become unreadable
- Users with accessibility needs can't use the tool
- Piped output contains garbage characters

**Warning signs:**

- Tests only run in non-plain mode
- New output paths don't check `config.IsPlain()`
- Unicode characters used without ASCII fallback

**Prevention:**

1. Test every output change in both modes
2. Plain mode should produce pure ASCII
3. Use formatter functions that handle both modes
4. Add CI step that runs with `--plain`

**Which phase should address:** Phase 1 (Foundation) - add plain mode testing before adding new features.

**Grove-specific notes:** The existing `formatter` package handles this well:

```go
func useAsciiIcons() bool {
    return config.IsPlain() || !config.IsNerdFonts()
}
```

New output must follow this pattern. The logger already has plain mode support. Test coverage should include `GROVE_TEST_COLORS=false` cases.

---

## Minor pitfalls

Annoyances that are easily fixed.

### Pitfall 8: Verbose error paths in normal output

**What goes wrong:** Warnings about unrelated issues appear during normal operations, cluttering output.

**Why it happens:** Defensive code logs warnings about edge cases even when they don't affect the current operation.

**Consequences:**

- Users worry about warnings they can't act on
- Important messages get lost in noise
- Output becomes harder to scan

**Warning signs:**

- Full file paths in warning messages
- Technical error details exposed to end users
- Warnings about items not involved in the operation

**Prevention:**

1. Only show warnings relevant to current operation
2. Use `--verbose` for detailed diagnostics
3. Shorten paths using `styles.PrettyPath()`
4. Make error messages actionable

**Which phase should address:** Phase 3 (Error Formatting) - clean up error messaging.

**Grove-specific notes:** [Issue #68](https://github.com/sqve/grove/issues/68) mentions:

```
Warning: Skipping worktree /home/sqve/code/work/platform/revert-1678-... (may be corrupted)
```

This appears during `grove remove` for an unrelated worktree. Fix: suppress or move to `--verbose`.

---

### Pitfall 9: Help/version output to wrong stream

**What goes wrong:** `--help` or `--version` output goes to stderr instead of stdout.

**Why it happens:** Default behavior in some CLI frameworks. [Common issue](https://news.ycombinator.com/item?id=37682859) that breaks `grove --help | less`.

**Consequences:**

- Can't pipe help to pager
- Scripts that capture help text fail

**Prevention:**

1. Ensure help goes to stdout (cobra handles this correctly)
2. Test: `grove --help | head -1` should work

**Which phase should address:** Verify early, fix if needed.

**Grove-specific notes:** Cobra sends help to stdout by default, but verify. Also ensure custom error messages still go to stderr.

---

### Pitfall 10: Color scheme conflicts with terminal themes

**What goes wrong:** Colors look unreadable on certain terminal backgrounds (yellow on white, dark blue on black).

**Why it happens:** Hardcoded color choices without testing both light and dark terminal themes.

**Consequences:**

- Text invisible for subset of users
- Users forced to use `--plain` mode

**Prevention:**

1. Use ANSI 16-color palette that terminal themes can override
2. Test on both light and dark terminals
3. Consider adaptive colors based on `COLORFGBG` or lipgloss detection

**Which phase should address:** Phase 1 (Foundation) - audit existing colors.

**Grove-specific notes:** Current `styles.go` uses ANSI 256 colors (0-15 range) which are themeable:

```go
Success = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
Warning = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
```

This is correct. Avoid adding hardcoded RGB values.

---

## Phase-specific warnings

| Phase            | Likely Pitfall                            | Mitigation                           |
| ---------------- | ----------------------------------------- | ------------------------------------ |
| Foundation       | Breaking plain mode with new abstractions | Test both modes for every change     |
| Foundation       | Inconsistent stdout/stderr usage          | Document rules, audit existing code  |
| Streaming        | Spinner lifecycle bugs when adding more   | Follow existing StartSpinner pattern |
| Streaming        | Buffering issues with subprocess output   | Use real-time streaming with capture |
| Error Formatting | Input-output mapping confusion            | Echo user input in output messages   |
| Error Formatting | Verbose irrelevant warnings               | Gate behind --verbose flag           |

## Integration risks

### Risk: Global state mutation during output

The logger and config packages use global state:

- `logger.plainMode` - atomic bool
- `config.Global` - struct with mutex

Adding new output features that modify these during command execution could cause race conditions in concurrent scenarios (like `exec --all`).

**Mitigation:** Treat config as read-only after initialization. If output behavior must vary per-operation, pass context through function parameters rather than mutating globals.

### Risk: Breaking test assertions

Many tests assert on exact output strings. Polishing output will break these tests.

**Mitigation:**

1. Accept that tests will break
2. Update assertions to match new format
3. Consider testing structure rather than exact strings where appropriate
4. Use golden files for complex output

### Risk: Backward compatibility for scripted users

Users may have scripts that parse current output format.

**Mitigation:**

1. Document that stderr is for humans, stdout data format is stable
2. Prefer `--json` for machine-readable output (already available)
3. Consider `--output=legacy` if migration path needed
4. Announce changes in changelog

## Sources

- [Command Line Interface Guidelines](https://clig.dev/) - Comprehensive CLI design guide
- [CLI UX Best Practices: Progress Displays](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays) - Spinner and progress bar patterns
- [Salesforce CLI TTY Detection Issue](https://github.com/forcedotcom/cli/issues/327) - Progress bar in non-TTY environments
- [golang-migrate stderr Issue](https://github.com/golang-migrate/migrate/issues/363) - stdout/stderr confusion consequences
- [CLI Input/Output Mapping Case Study](https://www.tweag.io/blog/2023-10-05-cli-ux-in-topiary/) - Topiary CLI UX improvements
- [Heroku CLI Style Guide](https://devcenter.heroku.com/articles/cli-style-guide) - Practical CLI style guidance
- [Subprocess Buffering](https://lucadrf.dev/blog/python-subprocess-buffers/) - Real-time output streaming challenges
- [Grove Issue #68](https://github.com/sqve/grove/issues/68) - Remove command output clarity
- [Grove Issue #44](https://github.com/sqve/grove/issues/44) - Hook output streaming
