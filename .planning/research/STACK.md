# Technology Stack: CLI Output Polish

**Project:** Grove v1.5 Output Polish
**Researched:** 2026-01-24
**Confidence:** HIGH (verified against official sources and existing codebase)

## Executive Summary

Grove already has the right foundation. The existing stack (lipgloss + termenv + custom spinner) is sufficient for v1.5 output polish. **No new dependencies needed.** Focus effort on consistent usage of existing tools, not adding libraries.

## Current Stack (Keep)

| Technology                                            | Version | Purpose              | Status           |
| ----------------------------------------------------- | ------- | -------------------- | ---------------- |
| [lipgloss](https://github.com/charmbracelet/lipgloss) | v1.1.0  | Styling, colors      | Keep at v1       |
| [termenv](https://github.com/muesli/termenv)          | v0.16.0 | Terminal detection   | Keep             |
| Custom spinner                                        | N/A     | Progress indication  | Enhance in-place |
| `os/exec`                                             | stdlib  | Subprocess execution | Keep             |

### Why Not Upgrade lipgloss to v2

lipgloss v2 (currently v2.0.0-beta.3 as of July 2025) is in beta. Changes include:

- New layers/canvas API for compositing
- Breaking API changes from v1

**Recommendation:** Stay on v1.1.0. The v2 features (layers, canvas) are for complex TUI layouts, not CLI output. Upgrading during an output polish milestone adds risk without value.

## New Capabilities: Implementation Approach

### Spinners for Long Operations

**Current state:** `logger.StartSpinner()` exists and works. Used in `fetch.go` and `workspace.go`.

**Problem:** Limited to single message, no update capability, no success/fail indication.

**Solution:** Enhance existing `StartSpinner()` in `internal/logger/logger.go`:

```go
type Spinner struct {
    done    chan bool
    message atomic.Value
    once    sync.Once
}

func StartSpinner(message string) *Spinner
func (s *Spinner) Update(message string)
func (s *Spinner) Stop()
func (s *Spinner) StopWithSuccess(message string)
func (s *Spinner) StopWithError(message string)
```

**Why not briandowns/spinner or yacspin:**

- briandowns/spinner (v1.23.2): 90+ spinner styles, but Grove only needs one. Adds dependency for unused features.
- yacspin: Better concurrency handling, but Grove's spinner runs in a single goroutine pattern. Not needed.
- Custom implementation is 50 lines. External library overhead isn't justified.

### Streaming Hook Output

**Problem:** Hook output (post-checkout, pre-commit) isn't visible until completion. Users see nothing during long hooks.

**Solution:** Use stdlib `os/exec` with direct stdout/stderr passthrough:

```go
func RunWithStreaming(cmd *exec.Cmd) error {
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

For line-by-line capture with processing:

```go
func RunWithStreamingCallback(cmd *exec.Cmd, onLine func(string)) error {
    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()

    cmd.Start()

    var wg sync.WaitGroup
    wg.Add(2)

    go func() {
        defer wg.Done()
        scanner := bufio.NewScanner(stdout)
        for scanner.Scan() {
            onLine(scanner.Text())
        }
    }()

    go func() {
        defer wg.Done()
        scanner := bufio.NewScanner(stderr)
        for scanner.Scan() {
            onLine(scanner.Text())
        }
    }()

    wg.Wait()
    return cmd.Wait()
}
```

**Why not go-cmd/cmd:**

- Adds dependency for a pattern that's ~20 lines of stdlib code
- go-cmd's streaming channels add complexity not needed for simple line-by-line output
- Grove's git operations are sequential, not concurrent. No race condition risk.

### Consistent Message Formats

**Current state:** `logger.Success/Error/Warning/Info` exist but usage varies:

- Some commands use `logger.Success()`, others use `fmt.Println()`
- Error messages sometimes include "Error:" prefix, sometimes not
- Dimmed text used inconsistently

**Solution:** Audit and standardize. No new libraries needed.

Establish patterns:

- All user-facing messages through `logger.*` functions
- Success messages: past tense verb ("Created worktree", "Fetched updates")
- Error messages: present tense problem ("Cannot remove locked worktree")
- Dimmed text: supplementary details only

## What NOT to Add

| Library                                                     | Why Not                                                                |
| ----------------------------------------------------------- | ---------------------------------------------------------------------- |
| [bubbletea](https://github.com/charmbracelet/bubbletea)     | Full TUI framework. Grove is a CLI, not a TUI. Massive overkill.       |
| [bubbles](https://github.com/charmbracelet/bubbles)         | TUI components. Same reason as bubbletea.                              |
| [briandowns/spinner](https://github.com/briandowns/spinner) | 90+ spinner styles when Grove needs one. Custom impl is simpler.       |
| [go-cmd/cmd](https://github.com/go-cmd/cmd)                 | Streaming wrapper for os/exec. Stdlib is sufficient for Grove's needs. |
| [huh](https://github.com/charmbracelet/huh)                 | Form/prompt library. Grove doesn't do interactive prompts.             |

## Integration Points

### With Existing Logger Package

The `internal/logger` package already:

- Respects `plainMode` for CI/non-TTY environments
- Uses `internal/styles` for consistent colors
- Writes to stderr (keeps stdout clean for JSON output)

Enhancements should maintain these properties.

### With Existing Formatter Package

The `internal/formatter` package handles:

- Worktree row formatting
- Status indicators (dirty, locked, sync)
- Plain mode fallbacks

No changes needed for output polish. Already well-structured.

### With JSON Output Mode

Some commands support `--json` flag. Spinner behavior:

- In JSON mode: Skip spinner entirely (no terminal animation)
- Logger functions already check `plainMode`

## File Changes Summary

| File                              | Change                                                                           |
| --------------------------------- | -------------------------------------------------------------------------------- |
| `internal/logger/logger.go`       | Enhance `StartSpinner()` return type, add `Update/StopWithSuccess/StopWithError` |
| `internal/git/git.go`             | Add `RunWithStreaming()` helper for hook execution                               |
| Various `cmd/grove/commands/*.go` | Audit and standardize logger usage                                               |

## Versions to Pin

No new dependencies. Existing go.mod versions are current:

```
github.com/charmbracelet/lipgloss v1.1.0  // Latest stable v1, published Mar 2025
github.com/muesli/termenv v0.16.0         // Latest, published Feb 2024
github.com/spf13/cobra v1.10.2            // Latest
```

## Sources

- [lipgloss releases](https://github.com/charmbracelet/lipgloss/releases) - v1.1.0 (Mar 2025), v2.0.0-beta.3 (Jul 2025)
- [termenv releases](https://github.com/muesli/termenv/releases) - v0.16.0 (Feb 2024)
- [briandowns/spinner](https://pkg.go.dev/github.com/briandowns/spinner) - v1.23.2 (Jan 2025)
- [go-cmd/cmd](https://pkg.go.dev/github.com/go-cmd/cmd) - os/exec wrapper
- [os/exec advanced usage](https://blog.kowalczyk.info/article/wOYk/advanced-command-execution-in-go-with-osexec.html) - streaming patterns
- Direct analysis of Grove codebase: `internal/logger/logger.go`, `internal/styles/styles.go`, `internal/formatter/formatter.go`
