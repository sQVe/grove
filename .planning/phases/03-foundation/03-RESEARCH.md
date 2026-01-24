# Phase 3: Foundation - Research

**Researched:** 2026-01-24
**Domain:** Terminal spinner API design and state management
**Confidence:** HIGH

## Summary

The Spinner API is Grove's foundation for all progress feedback. This phase extracts and enhances the existing `logger.StartSpinner()` implementation to provide a robust, well-tested API with three core capabilities: stopping with success/error indicators, updating messages mid-operation, and properly managing goroutine lifecycle.

The existing 50-line spinner implementation in `internal/logger/logger.go` is sound but limited. It uses Go's `sync.Once` and channel-based goroutine management correctly, avoiding common pitfalls like goroutine leaks. The enhancement path is clear: return a `*Spinner` type instead of a function, add state management with `atomic.Value` for message updates, and expose methods for success/error termination with visual indicators.

**Primary recommendation:** Extract spinner to `internal/logger/spinner.go`, return `*Spinner` with `Update()`, `Stop()`, `StopWithSuccess()`, and `StopWithError()` methods. Use `atomic.Value` for lock-free message updates. Maintain existing plain mode compliance and ANSI escape code patterns.

## Standard Stack

The established libraries/tools for terminal spinners in Go:

### Core

| Library              | Version  | Purpose                        | Why Standard                                             |
| -------------------- | -------- | ------------------------------ | -------------------------------------------------------- |
| stdlib `sync/atomic` | Go 1.23+ | Lock-free state updates        | Native, zero-dependency, hardware-backed atomicity       |
| stdlib `sync`        | Go 1.23+ | `sync.Once` for cleanup safety | Prevents double-close panics, standard goroutine pattern |
| stdlib `time`        | Go 1.23+ | Ticker for animation frames    | Animation timing, 80ms interval is industry standard     |

### Supporting

| Library           | Version           | Purpose                             | When to Use                                      |
| ----------------- | ----------------- | ----------------------------------- | ------------------------------------------------ |
| `lipgloss`        | v1.1.0 (existing) | Color/styling via `styles.Render()` | Already integrated, handles plain mode           |
| ANSI escape codes | Native            | `\r\033[K` pattern                  | Direct terminal control for flicker-free updates |

### Alternatives Considered

| Instead of     | Could Use                  | Tradeoff                                                                            |
| -------------- | -------------------------- | ----------------------------------------------------------------------------------- |
| Custom spinner | briandowns/spinner v1.23.2 | 90+ spinner styles vs single needed style; external dependency for 50 lines of code |
| Custom spinner | yacspin                    | Better concurrency handling but Grove's single-goroutine pattern doesn't need it    |
| `atomic.Value` | `sync.Mutex`               | Locks block readers; atomic reads scale across goroutines without contention        |

**Installation:**
No new dependencies. Use stdlib only.

## Architecture Patterns

### Recommended Spinner Structure

```go
internal/logger/
├── logger.go        // Existing: Success/Error/Info/Warning/Debug functions
├── spinner.go       // NEW: Extracted spinner implementation
└── logger_test.go   // Existing + new spinner state tests
```

### Pattern 1: Spinner Type with State Management

**What:** Spinner struct with atomic message updates and channel-based termination
**When to use:** Any operation >500ms (network, git operations on large repos)
**Example:**

```go
// Source: Grove enhancement pattern + sync/atomic best practices
type Spinner struct {
    message atomic.Value // string, updated lock-free
    done    chan bool
    once    sync.Once
}

func StartSpinner(message string) *Spinner {
    if isPlain() {
        // Plain mode: print message, return no-op spinner
        fmt.Fprintf(os.Stderr, "%s %s\n",
            styles.Render(&styles.Info, "→"), message)
        return &Spinner{done: make(chan bool)} // Already closed
    }

    s := &Spinner{done: make(chan bool)}
    s.message.Store(message)

    go s.animate()
    return s
}

func (s *Spinner) animate() {
    frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
    ticker := time.NewTicker(80 * time.Millisecond)
    defer ticker.Stop()

    i := 0
    for {
        select {
        case <-s.done:
            fmt.Fprint(os.Stderr, "\r\033[K") // Clear line
            return
        case <-ticker.C:
            msg := s.message.Load().(string)
            fmt.Fprintf(os.Stderr, "\r%s %s",
                styles.Render(&styles.Info, frames[i]),
                msg)
            i = (i + 1) % len(frames)
        }
    }
}

func (s *Spinner) Update(message string) {
    s.message.Store(message)
}

func (s *Spinner) Stop() {
    s.once.Do(func() {
        close(s.done)
        time.Sleep(10 * time.Millisecond) // Let goroutine clear line
    })
}

func (s *Spinner) StopWithSuccess(message string) {
    s.Stop()
    logger.Success(message)
}

func (s *Spinner) StopWithError(message string) {
    s.Stop()
    logger.Error(message)
}
```

### Pattern 2: Multi-Step Progress Updates

**What:** Update spinner message for each step in a sequence
**When to use:** Multi-step operations like clone (fetch + checkout), add with PR (fetch + create)
**Example:**

```go
// Source: CLI UX best practices (Evil Martians, 2024)
spinner := logger.StartSpinner("Step 1/3: Fetching remote...")
// ... fetch operation ...
spinner.Update("Step 2/3: Creating worktree...")
// ... create operation ...
spinner.Update("Step 3/3: Running hooks...")
// ... hook execution ...
spinner.StopWithSuccess("Created worktree with 3 preserved files")
```

### Pattern 3: Batch Operation Summary

**What:** Accumulate count during loop, show summary at end
**When to use:** Commands accepting multiple arguments (remove, lock, unlock)
**Example:**

```go
// Source: Existing remove.go pattern + SPIN-04 requirement
var removed []string
for _, target := range targets {
    if err := removeWorktree(target); err != nil {
        logger.Error("%s: %v", target, err)
        continue
    }
    removed = append(removed, target)
}

if len(removed) == 0 {
    return fmt.Errorf("no worktrees removed")
}

// Summary format
if len(removed) == 1 {
    logger.Success("Removed worktree %s", removed[0])
} else {
    logger.Success("Removed %d worktrees", len(removed))
}
```

### Pattern 4: Plain Mode Compliance

**What:** Detect TTY, degrade gracefully to text output
**When to use:** All spinner usage, all color/symbol output
**Example:**

```go
// Source: Existing logger.isPlain() pattern
func isPlain() bool {
    return plainMode.Load() // atomic.Bool
}

// In spinner:
if isPlain() {
    // Just print the message once, return no-op spinner
    fmt.Fprintf(os.Stderr, "%s %s\n",
        styles.Render(&styles.Info, "→"), message)
    // Return spinner with done channel already closed
}
```

### Anti-Patterns to Avoid

- **Mixing stdout/stderr for spinners:** Spinners MUST go to stderr. Data output goes to stdout. Mixing breaks piping.
- **Forgetting to call Stop():** Causes goroutine leak. Always use `defer spinner.Stop()` or explicit Stop in error paths.
- **Updating message with concatenation:** Use `Update()`, don't restart spinner. Restarting causes flicker.
- **Testing spinner animation in CI:** Check behavior (Stop called, message shown in plain mode), not animation frames.

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem                    | Don't Build             | Use Instead                                      | Why                                                         |
| -------------------------- | ----------------------- | ------------------------------------------------ | ----------------------------------------------------------- |
| TTY detection              | Custom fd checks        | Existing `config.IsPlain()` / `logger.isPlain()` | Already integrated, respects --plain flag override          |
| ANSI escape sequences      | Manual byte arrays      | ANSI standard `\r\033[K` pattern                 | Well-documented, cross-platform (Windows Terminal supports) |
| Goroutine cleanup          | Custom channel patterns | `sync.Once` + close(done)                        | Prevents double-close panic, stdlib pattern                 |
| Message update concurrency | Mutexes                 | `atomic.Value`                                   | Lock-free reads, copy-on-write pattern, no contention       |
| Plain mode detection       | ENV var checks          | `lipgloss` color profile detection               | Handles CI detection, terminal capability probing           |

**Key insight:** Grove already has correct patterns. The risk is over-engineering (adding yacspin when custom works) or under-engineering (not using atomic for message updates).

## Common Pitfalls

### Pitfall 1: Goroutine Leaks from Unclosed Spinners

**What goes wrong:** Spinner goroutine runs forever if Stop() never called. Memory leak, especially in tests.
**Why it happens:** Error paths return early without cleanup. Defer forgotten.
**How to avoid:**

- Use `defer spinner.Stop()` immediately after `StartSpinner()`
- In tests, verify goroutine count doesn't grow
- `sync.Once` ensures Stop() is idempotent (safe to call multiple times)
  **Warning signs:** Tests slow down over time, `runtime.NumGoroutine()` increases

### Pitfall 2: Message Update Race Conditions

**What goes wrong:** Concurrent reads of spinner message while goroutine updates it causes data race.
**Why it happens:** Using plain string field instead of `atomic.Value`. Go race detector catches this.
**How to avoid:**

- Store message in `atomic.Value`, not plain field
- Use `message.Load().(string)` in animate loop
- Use `message.Store(newMsg)` in Update()
  **Warning signs:** `go test -race` reports data race, occasional garbled spinner text

### Pitfall 3: Flicker from Line Clearing

**What goes wrong:** Spinner flickers or leaves artifacts when updating message.
**Why it happens:** Wrong ANSI sequence (`\r` alone vs `\r\033[K`), or clearing after printing.
**How to avoid:**

- Use `\r\033[K` (carriage return + clear to end of line) BEFORE printing new content
- Pattern: `fmt.Fprint(os.Stderr, "\r\033[K")` then print new line
- On Stop(), clear line before printing final message
  **Warning signs:** Visible cursor jump, leftover characters from longer messages

### Pitfall 4: Plain Mode Spinners in CI

**What goes wrong:** Spinner animations fill CI logs with partial frames, or fail because no TTY.
**Why it happens:** Not checking `isPlain()` before starting goroutine.
**How to avoid:**

- Check `isPlain()` in `StartSpinner()`, return no-op spinner
- No-op spinner: print message once, return spinner with closed channel
- Existing pattern: `logger.isPlain()` checks `plainMode.Load()` atomic bool
  **Warning signs:** CI logs show spinner frames, tests fail in CI but pass locally

### Pitfall 5: Testing Spinner State Instead of Behavior

**What goes wrong:** Tests assert "spinner animated 10 times" instead of "operation showed progress".
**Why it happens:** Testing implementation (animation loop) instead of interface (Stop called, message shown).
**How to avoid:**

- Test public API: `Update()` doesn't panic, `Stop()` clears line, `StopWithSuccess()` shows checkmark
- Test plain mode: message appears in stderr, no ANSI codes
- Don't test: animation frame count, exact timing, goroutine internals
  **Warning signs:** Flaky tests due to timing, tests break when animation speed changes

## Code Examples

Verified patterns from official sources and Grove codebase:

### Atomic Value for Spinner Messages

```go
// Source: https://pkg.go.dev/sync/atomic (atomic.Value documentation)
type Spinner struct {
    message atomic.Value // stores string
    done    chan bool
    once    sync.Once
}

// Writer goroutine (frequent reads)
func (s *Spinner) animate() {
    for {
        select {
        case <-s.done:
            return
        case <-ticker.C:
            msg := s.message.Load().(string) // Atomic read
            fmt.Fprintf(os.Stderr, "\r%s %s", frame, msg)
        }
    }
}

// Updater (infrequent writes)
func (s *Spinner) Update(message string) {
    s.message.Store(message) // Atomic write
}
```

### ANSI Escape Codes for Flicker-Free Updates

```go
// Source: https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797 (ANSI codes reference)
// \r      = Carriage return (move to column 0)
// \033[K  = Clear from cursor to end of line (CSI K)

// Clear current line before redrawing
fmt.Fprint(os.Stderr, "\r\033[K")

// Update spinner frame on same line
fmt.Fprintf(os.Stderr, "\r%s %s", frame, message)
```

### sync.Once for Safe Cleanup

```go
// Source: Grove's existing logger.go StartSpinner implementation
func (s *Spinner) Stop() {
    s.once.Do(func() {
        close(s.done)
        time.Sleep(10 * time.Millisecond) // Let animate() clear line
    })
}

// Safe to call multiple times:
spinner.Stop()
spinner.Stop() // No-op, doesn't panic
```

### Plain Mode Detection

```go
// Source: Grove internal/config/config.go and internal/logger/logger.go
var plainMode atomic.Bool

func Init(plain, debug bool) {
    plainMode.Store(plain)
}

func isPlain() bool {
    return plainMode.Load()
}

// In StartSpinner:
if isPlain() {
    fmt.Fprintf(os.Stderr, "%s %s\n", styles.Render(&styles.Info, "→"), message)
    s := &Spinner{done: make(chan bool)}
    close(s.done) // Already done, all methods are no-ops
    return s
}
```

### Testing Plain Mode

```go
// Source: Grove internal/logger/logger_test.go pattern
func TestSpinnerPlainMode(t *testing.T) {
    oldStderr := os.Stderr
    r, w, _ := os.Pipe()
    os.Stderr = w

    Init(true, false) // plain=true
    spinner := StartSpinner("Loading...")
    spinner.Stop()

    _ = w.Close()
    os.Stderr = oldStderr

    var buf bytes.Buffer
    _, _ = io.Copy(&buf, r)
    output := buf.String()

    if !strings.Contains(output, "Loading...") {
        t.Error("Plain mode should print message")
    }
    if strings.Contains(output, "⠋") {
        t.Error("Plain mode should not show spinner frames")
    }
}
```

## State of the Art

| Old Approach                | Current Approach              | When Changed         | Impact                                                   |
| --------------------------- | ----------------------------- | -------------------- | -------------------------------------------------------- |
| `sync.Mutex` for message    | `atomic.Value`                | Go 1.4 (2014)        | Lock-free reads scale to thousands of goroutines         |
| Return `func()` closer      | Return `*Spinner` type        | Library evolution    | Enables Update() and contextual Stop methods             |
| Separate success/error logs | `StopWithSuccess/Error()`     | Modern CLI UX (2024) | Single line of output instead of two (spinner + success) |
| TTY detection via syscall   | `lipgloss` color profile      | 2020s                | Handles Windows Terminal, CI detection, user overrides   |
| Braille patterns (U+28xx)   | Industry standard since 2010s | Established          | Smooth animation, low visual noise                       |

**Deprecated/outdated:**

- Plain `func() Stop()` return value: Still valid, but `*Spinner` enables richer API
- Manual TTY detection: Use existing `config.IsPlain()` which respects user flags

## Open Questions

Things that couldn't be fully resolved:

1. **Sleep duration after channel close (10ms)**
    - What we know: Existing implementation uses 10ms sleep after closing channel to let goroutine clear line
    - What's unclear: Is this enough for all systems? Too much? Could use sync.WaitGroup instead?
    - Recommendation: Keep 10ms. It works. Optimize only if tests show timing issues. WaitGroup adds complexity for marginal gain.

2. **Multi-step format: "Step N/M: action" vs "action (N/M)"**
    - What we know: SPIN-03 requires "Step N/M: action" format
    - What's unclear: Should parentheses variant be supported? Command preference?
    - Recommendation: Use "Step N/M: action" consistently per requirement. Don't overthink.

3. **Batch summary: singular/plural handling**
    - What we know: Need "Removed 3 worktrees" but also "Removed worktree main" for single
    - What's unclear: Generic pluralization helper vs per-command logic?
    - Recommendation: Per-command conditional. English plurals are irregular, generic helper would need locale support (out of scope).

## Sources

### Primary (HIGH confidence)

- [sync/atomic package](https://pkg.go.dev/sync/atomic) - atomic.Value, atomic.Bool API and usage patterns
- Grove codebase: `internal/logger/logger.go` (existing spinner implementation)
- Grove codebase: `internal/logger/logger_test.go` (testing patterns)
- Grove codebase: `internal/styles/styles.go` (plain mode integration)
- [ANSI Escape Codes reference](https://gist.github.com/fnky/458719343aabd01cfb17a3a4f7296797) - Terminal control sequences

### Secondary (MEDIUM confidence)

- [CLI UX best practices: 3 patterns for improving progress displays](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays) - Step N/M pattern, spinner best practices (2024)
- [yacspin package](https://pkg.go.dev/github.com/theckman/yacspin) - Modern spinner API design patterns (Stop/StopFail methods)
- [Two concurrency patterns which avoid goroutine leaks](https://nsrip.com/posts/goroutineleak.html) - Channel-based cleanup patterns
- [Atomic Value in Go Concurrency](https://medium.com/@AlexanderObregon/atomic-value-in-go-concurrency-d82dd187e73b) - atomic.Value use cases

### Tertiary (LOW confidence)

- [briandowns/spinner](https://github.com/briandowns/spinner) - Feature comparison only (not using library)
- WebSearch: "Go TTY detection isatty non-TTY output best practices 2026" - Confirmed stdlib patterns

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Based on stdlib documentation and existing Grove patterns
- Architecture: HIGH - Direct codebase analysis, proven patterns in logger.go
- Pitfalls: HIGH - Verified with race detector warnings, goroutine leak patterns from official Go sources

**Research date:** 2026-01-24
**Valid until:** 2026-02-24 (30 days - Go stdlib stable, spinner patterns established)
