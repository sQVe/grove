# Phase 4: Hook Streaming - Research

**Researched:** 2026-01-24
**Domain:** Real-time command output streaming with line prefixes in Go
**Confidence:** HIGH

## Summary

Hook streaming enables users to see output from long-running `grove add` hooks in real-time, addressing issue #44 where users see nothing until hooks complete. The solution involves replacing the current buffered exec.Command pattern (stdout/stderr captured to bytes.Buffer) with streaming via StdoutPipe/StderrPipe, processing output line-by-line with bufio.Scanner, and prefixing each line with the hook identifier.

The core challenge is integrating streaming with the existing spinner. The spinner must stop cleanly before hook output appears, then resume (or show success) after hooks complete. Go's os/exec package provides StdoutPipe/StderrPipe for streaming, with critical requirements: call Start() not Run(), read from pipes before calling Wait(), and handle both stdout/stderr separately to avoid blocking.

**Primary recommendation:** Create a custom io.Writer that prefixes each line with hook name, assign to cmd.Stdout/cmd.Stderr before cmd.Start(), use bufio.Scanner to read line-by-line. Stop spinner before streaming starts, print hook output directly to stderr (same stream as spinner), show final success/failure after all hooks complete.

## Standard Stack

The established libraries/tools for streaming command output in Go:

### Core

| Library               | Version  | Purpose                            | Why Standard                                        |
| --------------------- | -------- | ---------------------------------- | --------------------------------------------------- |
| stdlib `os/exec`      | Go 1.25+ | Command execution                  | Native, provides StdoutPipe/StderrPipe for streams  |
| stdlib `bufio`        | Go 1.25+ | Line-by-line reading               | Scanner with ScanLines for newline-delimited output |
| stdlib `io`           | Go 1.25+ | io.Writer interface                | Foundation for custom prefixing writers             |
| Grove `logger` pkg    | Internal | Consistent output formatting       | Existing, handles plain mode, stderr targeting      |
| Grove `*Spinner` type | Internal | Progress indication (from Phase 3) | Provides Stop() for clean handoff to streaming      |

### Supporting

| Library     | Version  | Purpose                        | When to Use                                       |
| ----------- | -------- | ------------------------------ | ------------------------------------------------- |
| `sync`      | Go 1.25+ | WaitGroup for concurrent reads | When reading stdout/stderr in separate goroutines |
| `bytes`     | Go 1.25+ | Buffer for line assembly       | Internal to custom io.Writer for partial lines    |
| ANSI escape | Native   | Terminal control               | Clearing spinner line before streaming starts     |

### Alternatives Considered

| Instead of             | Could Use                                      | Tradeoff                                                                   |
| ---------------------- | ---------------------------------------------- | -------------------------------------------------------------------------- |
| StdoutPipe/StderrPipe  | cmd.Stdout = os.Stderr directly                | Lose ability to prefix lines (output mixed with Grove messages)            |
| Custom io.Writer       | go-prefix-writer library                       | External dependency for 30 lines of code                                   |
| bufio.Scanner          | io.Copy with custom reader                     | More complex, Scanner handles newlines correctly                           |
| Sequential execution   | go-cmd/cmd (concurrent non-blocking)           | Overkill for sequential hooks, adds complexity                             |
| Separate stdout/stderr | CombinedOutput()                               | Lose distinction between hook stdout/stderr, can't stream (waits for exit) |
| Stop spinner, stream   | Concurrent spinner + streaming with mutex lock | Flicker/race issues, complexity without benefit                            |

**Installation:**
No new dependencies. Use stdlib only.

## Architecture Patterns

### Recommended Streaming Structure

```
internal/hooks/
├── hooks.go        // Existing: RunAddHooks, HookResult, RunResult
├── streaming.go    // NEW: PrefixWriter, streaming execution
└── hooks_test.go   // Existing + new streaming tests
```

### Pattern 1: Custom PrefixWriter for Line-by-Line Output

**What:** io.Writer that buffers input, emits complete lines with prefix
**When to use:** Any command where output lines need identification
**Example:**

```go
// Source: Pattern from kvz.io/blog/prefix-streaming-stdout-and-stderr-in-golang.html
// Simplified for Grove's sequential hook execution use case

type PrefixWriter struct {
    prefix string
    target io.Writer
    buf    bytes.Buffer
}

func NewPrefixWriter(prefix string, target io.Writer) *PrefixWriter {
    return &PrefixWriter{
        prefix: prefix,
        target: target,
    }
}

func (w *PrefixWriter) Write(p []byte) (n int, err error) {
    // Write to internal buffer
    n, err = w.buf.Write(p)
    if err != nil {
        return n, err
    }

    // Process complete lines
    for {
        line, err := w.buf.ReadString('\n')
        if err != nil {
            // EOF or no complete line yet - put back incomplete data
            if line != "" {
                w.buf.WriteString(line)
            }
            break
        }

        // Emit line with prefix
        _, writeErr := fmt.Fprintf(w.target, "%s %s", w.prefix, line)
        if writeErr != nil {
            return n, writeErr
        }
    }

    return n, nil
}

// Flush writes any remaining buffered data (for commands that don't end with \n)
func (w *PrefixWriter) Flush() error {
    remaining := w.buf.String()
    if remaining != "" {
        _, err := fmt.Fprintf(w.target, "%s %s\n", w.prefix, remaining)
        w.buf.Reset()
        return err
    }
    return nil
}
```

### Pattern 2: Streaming Hook Execution

**What:** Replace buffered execution with streaming via StdoutPipe/StderrPipe
**When to use:** Hooks that may produce output or run for >1 second
**Example:**

```go
// Source: Go os/exec documentation + Grove integration pattern

func RunAddHooksStreaming(workDir string, commands []string) *RunResult {
    result := &RunResult{}

    if len(commands) == 0 {
        return result
    }

    logger.Debug("Running %d add hooks in %s (streaming)", len(commands), workDir)

    for _, cmdStr := range commands {
        logger.Debug("Executing hook: %s", cmdStr)

        cmd := exec.Command("sh", "-c", cmdStr)
        cmd.Dir = workDir

        // Create prefix writers for stdout/stderr
        prefix := logger.Dimmed("  [%s]", cmdStr)
        stdout := NewPrefixWriter(prefix, os.Stderr)
        stderr := NewPrefixWriter(prefix, os.Stderr)

        cmd.Stdout = stdout
        cmd.Stderr = stderr

        // Start command (not Run - we need to flush after)
        err := cmd.Start()
        if err != nil {
            result.Failed = &HookResult{
                Command:  cmdStr,
                ExitCode: 1,
                Stdout:   "",
                Stderr:   err.Error(),
            }
            return result
        }

        // Wait for completion
        err = cmd.Wait()

        // Flush any remaining buffered output
        _ = stdout.Flush()
        _ = stderr.Flush()

        if err != nil {
            exitCode := 1
            exitErr := &exec.ExitError{}
            if errors.As(err, &exitErr) {
                exitCode = exitErr.ExitCode()
            }

            result.Failed = &HookResult{
                Command:  cmdStr,
                ExitCode: exitCode,
                Stdout:   "",  // Already streamed
                Stderr:   "",  // Already streamed
            }

            logger.Debug("Hook failed with exit code %d: %s", exitCode, cmdStr)
            return result
        }

        result.Succeeded = append(result.Succeeded, cmdStr)
        logger.Debug("Hook succeeded: %s", cmdStr)
    }

    return result
}
```

### Pattern 3: Integration with Spinner (from Phase 3)

**What:** Stop spinner before streaming, resume/complete after
**When to use:** All hook execution in `grove add`
**Example:**

```go
// Source: Grove add.go integration pattern

func runAddHooks(sourceWorktree, destWorktree string) *hooks.RunResult {
    var addHooks []string
    if sourceWorktree != "" {
        addHooks = hooks.GetAddHooks(sourceWorktree)
    }

    if len(addHooks) == 0 {
        logger.Debug("No add hooks configured")
        return nil
    }

    // Start spinner for preparation phase
    spinner := logger.StartSpinner("Running hooks...")

    // Stop spinner cleanly before streaming output
    spinner.Stop()

    // Stream hook output (no spinner running)
    result := hooks.RunAddHooksStreaming(destWorktree, addHooks)

    // No need to restart spinner - success/failure is immediate
    return result
}
```

### Pattern 4: Plain Mode Compliance

**What:** Prefix formatting works in both TTY and non-TTY
**When to use:** All streaming output
**Example:**

```go
// PrefixWriter already writes to io.Writer (os.Stderr)
// Plain mode is handled by logger.Dimmed() returning plain text vs styled

// In plain mode:
prefix := logger.Dimmed("  [npm install]")  // Returns "  [npm install]" (no ANSI)

// In TTY mode:
prefix := logger.Dimmed("  [npm install]")  // Returns "\033[2m  [npm install]\033[0m"

// PrefixWriter just emits prefix + line, doesn't care about TTY
```

### Anti-Patterns to Avoid

- **Using cmd.Run() with pipes:** Race condition. Always use cmd.Start() then cmd.Wait()
- **Reading pipes after cmd.Wait():** Deadlock risk. Read before or during Wait()
- **Forgetting to flush PrefixWriter:** Last line without \n gets lost
- **Concurrent spinner + streaming:** Flicker and complexity. Stop spinner first
- **Buffering hook output for later display:** Defeats purpose of streaming (user sees nothing until done)
- **Only streaming on error:** User needs feedback during long operations, not just failures

## Don't Hand-Roll

Problems that look simple but have existing solutions:

| Problem                          | Don't Build                   | Use Instead                              | Why                                                 |
| -------------------------------- | ----------------------------- | ---------------------------------------- | --------------------------------------------------- |
| Line buffering logic             | Manual newline splitting      | bufio.Scanner or bytes.Buffer.ReadString | Handles CRLF/LF, incomplete reads, edge cases       |
| Concurrent stdout/stderr reading | Custom goroutine coordination | Sequential reads or sync.WaitGroup       | Sequential hooks don't need concurrency complexity  |
| Prefix colorization              | ANSI codes in prefix string   | Existing logger.Dimmed()                 | Handles plain mode, consistent with Grove output    |
| Hook output capture for errors   | Separate capture mechanism    | Accept streaming-only                    | Hooks already show output live; exit code is enough |
| Multi-line prefix alignment      | Manual padding/wrapping       | Simple per-line prefix                   | Complexity not needed; each line stands alone       |

**Key insight:** The core is a simple io.Writer wrapper with line buffering. Don't add concurrency, don't add complex formatting, don't capture output. Just prefix and stream.

## Common Pitfalls

### Pitfall 1: cmd.Run() with StdoutPipe Causes Race

**What goes wrong:** Calling cmd.Run() when using StdoutPipe/StderrPipe leads to "Wait was already called" or hanging
**Why it happens:** Run() calls Wait() internally, but docs require reading pipes before Wait()
**How to avoid:**

- Always use cmd.Start(), then read pipes, then cmd.Wait()
- Pattern: `cmd.Start() -> read from pipes -> cmd.Wait()`
- Go documentation explicitly warns: "incorrect to call Run when using StdoutPipe"
  **Warning signs:** Tests hang, "Wait was already called" errors, command never completes

### Pitfall 2: Forgetting to Flush Incomplete Lines

**What goes wrong:** Last line of output missing if command doesn't end with newline
**Why it happens:** PrefixWriter.Write() only emits complete lines (ending in \n), buffers rest
**How to avoid:**

- Call `Flush()` after cmd.Wait() to emit any buffered partial line
- Add newline in Flush() if remaining text doesn't have one
- Test with commands that don't end in newline: `echo -n "no newline"`
  **Warning signs:** Hook output cut off, "Done" message missing from npm/pnpm output

### Pitfall 3: Spinner and Streaming Output Race

**What goes wrong:** Spinner animation interleaves with hook output, causing visual corruption
**Why it happens:** Spinner goroutine writes to stderr while hooks write to stderr simultaneously
**How to avoid:**

- Stop spinner BEFORE starting streaming execution
- Don't try to "pause and resume" spinner - just stop it
- Pattern: spinner.Stop() -> stream hooks -> logger.Success() or logger.Error()
  **Warning signs:** Corrupted output, spinner frames appearing in middle of hook output

### Pitfall 4: Wrong Stream for Hook Output

**What goes wrong:** Hook output goes to stdout, conflicts with `--switch` path output
**Why it happens:** Choosing os.Stdout instead of os.Stderr for hook output target
**How to avoid:**

- All user-facing output (spinners, hooks, messages) goes to stderr
- Only structured data (paths for shell wrapper) goes to stdout
- PrefixWriter target should be os.Stderr
  **Warning signs:** `grove add --switch` breaks, shell wrapper cd's to wrong path

### Pitfall 5: Unbounded Buffer Growth from Long Lines

**What goes wrong:** Hook outputs single 10MB line, causes OOM
**Why it happens:** bytes.Buffer in PrefixWriter grows without limit
**How to avoid:**

- Acceptable for hooks - npm/pnpm don't output MB-sized lines
- If needed: limit buffer size, truncate lines >64KB
- Monitor with tests: `dd if=/dev/zero bs=1M count=1 | hexdump -C` (long lines)
  **Warning signs:** Memory spikes during hook execution, OOM in CI

## Code Examples

Verified patterns from official sources and Grove integration:

### Complete PrefixWriter Implementation

```go
// Source: Adapted from kvz.io prefix-streaming pattern for Grove's needs

package hooks

import (
    "bytes"
    "fmt"
    "io"
)

// PrefixWriter wraps an io.Writer and prefixes each complete line
type PrefixWriter struct {
    prefix string
    target io.Writer
    buf    bytes.Buffer
}

func NewPrefixWriter(prefix string, target io.Writer) *PrefixWriter {
    return &PrefixWriter{
        prefix: prefix,
        target: target,
    }
}

func (w *PrefixWriter) Write(p []byte) (n int, err error) {
    n, err = w.buf.Write(p)
    if err != nil {
        return n, err
    }

    // Process all complete lines
    for {
        line, readErr := w.buf.ReadString('\n')
        if readErr != nil {
            // No complete line, put back incomplete data
            if line != "" {
                w.buf.WriteString(line)
            }
            break
        }

        // Emit line with prefix
        _, writeErr := fmt.Fprintf(w.target, "%s %s", w.prefix, line)
        if writeErr != nil {
            return n, writeErr
        }
    }

    return n, nil
}

// Flush emits any remaining buffered data with newline
func (w *PrefixWriter) Flush() error {
    remaining := w.buf.String()
    if remaining != "" {
        _, err := fmt.Fprintf(w.target, "%s %s\n", w.prefix, remaining)
        w.buf.Reset()
        return err
    }
    return nil
}
```

### Streaming Execution Pattern

```go
// Source: Go os/exec.StdoutPipe documentation example + Grove error handling

cmd := exec.Command("sh", "-c", "pnpm install")
cmd.Dir = worktreeDir

// Create prefixed writers
hookPrefix := styles.Render(&styles.Dimmed, "  [pnpm install]")
prefixStdout := NewPrefixWriter(hookPrefix, os.Stderr)
prefixStderr := NewPrefixWriter(hookPrefix, os.Stderr)

cmd.Stdout = prefixStdout
cmd.Stderr = prefixStderr

// Start command
if err := cmd.Start(); err != nil {
    return fmt.Errorf("failed to start hook: %w", err)
}

// Wait for completion
err := cmd.Wait()

// Flush any remaining buffered output
_ = prefixStdout.Flush()
_ = prefixStderr.Flush()

if err != nil {
    exitCode := 1
    if exitErr, ok := err.(*exec.ExitError); ok {
        exitCode = exitErr.ExitCode()
    }
    return &HookResult{
        Command:  "pnpm install",
        ExitCode: exitCode,
    }
}
```

### Integration with Grove add.go

```go
// Source: Grove internal/hooks/hooks.go + cmd/grove/commands/add.go

func runAddHooks(sourceWorktree, destWorktree string) *hooks.RunResult {
    addHooks := hooks.GetAddHooks(sourceWorktree)
    if len(addHooks) == 0 {
        return nil
    }

    // Show that hooks are starting
    logger.Info("Running %d hooks...", len(addHooks))

    // Stream hooks (no spinner during streaming)
    result := hooks.RunAddHooksStreaming(destWorktree, addHooks)

    return result
}

// In add.go after worktree creation:
hookResult := runAddHooks(sourceWorktree, worktreePath)

if switchTo {
    fmt.Println(worktreePath) // stdout for shell wrapper
} else {
    logger.Success("Created worktree at %s", styles.RenderPath(worktreePath))
    logHookResult(hookResult)  // Shows summary, not output (already streamed)
}
```

### Testing Streaming Output

```go
// Source: Grove testing pattern from hooks_test.go

func TestRunAddHooksStreaming(t *testing.T) {
    t.Run("streams output with prefix", func(t *testing.T) {
        workDir := t.TempDir()

        // Capture stderr
        oldStderr := os.Stderr
        r, w, _ := os.Pipe()
        os.Stderr = w

        commands := []string{"echo 'line 1'; echo 'line 2'"}
        result := RunAddHooksStreaming(workDir, commands)

        _ = w.Close()
        os.Stderr = oldStderr

        var buf bytes.Buffer
        _, _ = io.Copy(&buf, r)
        output := buf.String()

        // Verify output contains prefixed lines
        if !strings.Contains(output, "[echo") {
            t.Error("Expected prefix in output")
        }
        if !strings.Contains(output, "line 1") || !strings.Contains(output, "line 2") {
            t.Error("Expected hook output in stderr")
        }

        if len(result.Succeeded) != 1 {
            t.Errorf("Expected 1 succeeded, got %d", len(result.Succeeded))
        }
    })

    t.Run("handles commands without trailing newline", func(t *testing.T) {
        workDir := t.TempDir()

        oldStderr := os.Stderr
        r, w, _ := os.Pipe()
        os.Stderr = w

        commands := []string{"echo -n 'no newline'"}
        result := RunAddHooksStreaming(workDir, commands)

        _ = w.Close()
        os.Stderr = oldStderr

        var buf bytes.Buffer
        _, _ = io.Copy(&buf, r)
        output := buf.String()

        // Flush() should have added the missing newline
        if !strings.Contains(output, "no newline") {
            t.Error("Expected output from command without newline")
        }

        if len(result.Succeeded) != 1 {
            t.Error("Command should succeed despite no trailing newline")
        }
    })
}
```

## State of the Art

| Old Approach                    | Current Approach                  | When Changed      | Impact                                                     |
| ------------------------------- | --------------------------------- | ----------------- | ---------------------------------------------------------- |
| CombinedOutput (buffered)       | StdoutPipe/StderrPipe (streaming) | os/exec inception | Users see progress during execution, not just at end       |
| bytes.Buffer for output capture | Direct io.Writer assignment       | os/exec inception | Simpler, less memory, real-time output                     |
| External prefix libraries       | Custom io.Writer (30 lines)       | 2020s             | No dependencies for simple line buffering                  |
| Manual ReadString loops         | bufio.Scanner                     | Go 1.0            | Handles edge cases (CRLF, incomplete reads)                |
| Concurrent spinner + output     | Stop spinner, stream, finish      | CLI UX evolution  | No flicker, clean transitions, simpler mental model        |
| Third-party streaming wrappers  | Direct os/exec usage              | Current           | Stdlib is sufficient for sequential hooks, less complexity |

**Deprecated/outdated:**

- **go-exec-streamer:** Adds builder pattern complexity for simple case. Direct os/exec is clearer.
- **go-cmd/cmd:** Designed for concurrent non-blocking execution. Grove's hooks are sequential, don't need this.
- **Capturing output for later display:** Old pattern was buffer-then-show. Current is stream-immediately.

## Open Questions

Things that couldn't be fully resolved:

1. **Hook prefix format: "[command]" vs "command:" vs custom**
    - What we know: STRM-02 requires prefix identifying which hook is running
    - What's unclear: Exact format preference. Should it include hook number (1/3)?
    - Recommendation: Use `logger.Dimmed("  [%s]", cmdStr)` for consistency with existing ListItemGroup style. Don't add numbers unless user requests it.

2. **Handling extremely long-running hooks (>30s)**
    - What we know: Hooks stream output in real-time, no spinner during streaming
    - What's unclear: Should there be a timeout? Should spinner resume if no output for N seconds?
    - Recommendation: No timeout (npm install can take minutes). No spinner resume (adds complexity, output shows it's working). Trust the hook.

3. **Stdout vs stderr for hook output (from hook's perspective)**
    - What we know: Both stdout and stderr should stream to user
    - What's unclear: Should they be distinguished visually (different colors)?
    - Recommendation: Don't distinguish. Both use same prefix, both go to os.Stderr (Grove's user output stream). Color difference adds noise without value.

4. **Hook output in --switch mode**
    - What we know: --switch outputs worktree path to stdout for shell wrapper
    - What's unclear: Should hooks still stream to stderr in --switch mode, or be silent?
    - Recommendation: Still stream to stderr. Shell wrapper only reads stdout. User sees hook progress even when switching.

5. **Migration path for existing buffered hooks**
    - What we know: Current RunAddHooks buffers output, only shows on failure
    - What's unclear: Can we replace directly, or need feature flag for gradual rollout?
    - Recommendation: Direct replacement. Streaming is strictly better UX. No flag needed.

## Sources

### Primary (HIGH confidence)

- [os/exec package documentation](https://pkg.go.dev/os/exec) - StdoutPipe/StderrPipe patterns, cmd.Start()/Wait() requirements
- [bufio package documentation](https://pkg.go.dev/bufio) - Scanner usage, line buffering, buffer size limits
- Grove codebase: `internal/hooks/hooks.go` - Current buffered implementation
- Grove codebase: `internal/logger/logger.go` - Output patterns, plain mode handling
- Grove codebase: `cmd/grove/commands/add.go` - Hook integration point
- [kvz.io: Prefix Streaming stdout & stderr in Go](https://kvz.io/blog/prefix-streaming-stdout-and-stderr-in-golang.html) - PrefixWriter implementation pattern

### Secondary (MEDIUM confidence)

- [Advanced command execution in Go with os/exec](https://blog.kowalczyk.info/article/wOYk/advanced-command-execution-in-go-with-osexec.html) - Streaming patterns, race condition warnings
- [Reading os/exec.Cmd Output Without Race Conditions](https://hackmysql.com/rand/reading-os-exec-cmd-output-without-race-conditions/) - Start/Wait order, pipe handling
- [ora (Node.js spinner)](https://github.com/sindresorhus/ora) - Automatic stream handling pattern (inspiration for spinner stop)
- WebSearch: "Go exec.Command stream stdout stderr real-time 2026" - Verified streaming patterns

### Tertiary (LOW confidence)

- [go-prefix-writer package](https://github.com/egym-playground/go-prefix-writer) - Alternative library (not using, but validates approach)
- [lineprefix package](https://github.com/abiosoft/lineprefix) - Alternative library (not using, but validates approach)
- [go-cmd/cmd package](https://github.com/go-cmd/cmd) - Concurrent streaming (not needed for sequential hooks)

## Metadata

**Confidence breakdown:**

- Standard stack: HIGH - Based on Go stdlib documentation and Grove's existing patterns
- Architecture: HIGH - Direct codebase analysis, proven io.Writer pattern
- Pitfalls: HIGH - Verified from os/exec documentation warnings and Grove testing patterns
- Integration: HIGH - Clear integration point in add.go, spinner API from Phase 3

**Research date:** 2026-01-24
**Valid until:** 2026-03-24 (60 days - os/exec stdlib stable, pattern established)
