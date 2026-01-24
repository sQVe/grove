# Architecture Patterns: CLI Output Polish

**Domain:** CLI output unification for Go CLI
**Researched:** 2026-01-24
**Focus:** Integration with existing Grove architecture

## Current Architecture Analysis

### Existing Components

| Component             | Location                 | Purpose                                               | Current State                   |
| --------------------- | ------------------------ | ----------------------------------------------------- | ------------------------------- |
| `internal/logger/`    | logger.go                | Spinner, Success/Error/Warning/Info/Debug, list items | Used inconsistently             |
| `internal/formatter/` | formatter.go             | Worktree row formatting, indicators                   | Used by list/status only        |
| `internal/styles/`    | styles.go                | Lipgloss color definitions, Render helper             | Works well, respects plain mode |
| `internal/config/`    | config.go                | Global.Plain, Global.Debug, IsPlain()                 | Foundation is solid             |
| Commands              | cmd/grove/commands/\*.go | Cobra command handlers                                | Mixed output patterns           |

### Output Pattern Inventory

Analyzed all 15 commands. Current patterns fall into categories:

**Uses logger correctly:**

- `add.go` — logger.Success/Info/Warning/ListItemWithNote/ListSubItem/ListItemGroup
- `clone.go` — logger.Success/Info/Warning
- `fetch.go` — logger.StartSpinner, logger.Error, but also direct fmt.Printf
- `prune.go` — logger.Success/Info/Warning/Error/Dimmed

**Mixed logger + fmt.Print:**

- `doctor.go` — fmt.Printf for issue output, logger only for debug
- `list.go` — fmt.Println only (delegates to formatter)
- `status.go` — fmt.Println only (delegates to formatter)
- `exec.go` — logger.Info/Success/Error/Warning, fmt.Println for spacing

**Mostly fmt.Print (no logger):**

- `config.go` — pure fmt.Println
- `fetch.go` (output functions) — printRefChange uses fmt.Printf with styles.Render

### Current Flow

```
main.go
  |-- config.LoadFromGitConfig()    # Sets Global.Plain, Global.Debug
  |-- logger.Init(plain, debug)     # Stores in atomic vars
  +-- rootCmd.PersistentPreRun      # Handles --plain/--debug overrides
        +-- logger.Init() again     # Re-init after flag parsing

Commands call:
  |-- logger.* functions            # Go to stderr, respect plain mode
  |-- fmt.Print*                    # Go to stdout, no plain awareness
  +-- styles.Render()               # Plain-aware but manual
```

### Key Observations

1. **Spinner exists but underused** — Only `fetch.go` uses `logger.StartSpinner()`. Other long operations (clone, add with PR fetch, prune with fetch) don't show progress.

2. **Output goes to wrong streams** — Some commands mix stdout/stderr inconsistently. List/status use stdout (correct for data), but doctor uses both.

3. **Plain mode partially respected** — `styles.Render()` checks `config.IsPlain()`, logger functions check it, but direct `fmt.Printf` calls with unicode symbols don't.

4. **No streaming for hooks** — `hooks.RunAddHooks()` collects all output and shows at end. No real-time feedback.

5. **Formatter is domain-specific** — `internal/formatter/` only handles worktree rows. Not a general output abstraction.

## Recommended Architecture

### Integration Strategy: Extend, Don't Replace

The existing `internal/logger/` package is the right foundation. Extend it rather than creating parallel abstractions.

### Component Boundaries

```
internal/
|-- logger/
|   |-- logger.go        # Existing: Success/Error/Warning/Info/Debug/Dimmed
|   |-- spinner.go       # NEW: Extract StartSpinner, add progress variants
|   |-- stream.go        # NEW: Real-time output streaming
|   +-- logger_test.go   # Existing + new tests
|-- styles/
|   +-- styles.go        # Keep as-is, already works well
|-- formatter/
|   +-- formatter.go     # Keep as-is, domain-specific is fine
+-- config/
    +-- config.go        # Keep as-is, plain mode works
```

### New Components Needed

#### 1. `internal/logger/spinner.go` (Extract + Extend)

Extract spinner from logger.go and add variants:

```go
type Spinner struct {
    message string
    done    chan bool
    once    sync.Once
}

func StartSpinner(message string) *Spinner
func (s *Spinner) Stop()
func (s *Spinner) StopWithSuccess(message string)  // NEW
func (s *Spinner) StopWithError(message string)    // NEW
func (s *Spinner) Update(message string)           // NEW: Change spinner text
```

Rationale: Commands like fetch currently stop spinner then print success separately. This creates a cleaner pattern.

#### 2. `internal/logger/stream.go` (New)

For hook output streaming:

```go
type StreamWriter struct {
    prefix string
    indent int
}

func NewStreamWriter(prefix string) *StreamWriter
func (w *StreamWriter) Write(p []byte) (n int, err error)
func StreamCommand(cmd *exec.Cmd, prefix string) error
```

Rationale: Hook output currently buffered. Users see nothing during npm install. Stream with optional prefix for multi-command context.

### Modified Components

#### Commands: Adopt Consistent Patterns

Each command should follow this structure:

```go
func runXxx(...) error {
    // 1. Setup/validation (no output)

    // 2. Long operation with spinner
    spinner := logger.StartSpinner("Fetching...")
    err := doThing()
    if err != nil {
        spinner.StopWithError("Fetch failed")
        return err
    }
    spinner.StopWithSuccess("Fetched 5 refs")

    // 3. Result output via logger
    logger.Success("Created worktree at %s", path)

    // 4. Detail output via logger.ListSubItem etc

    return nil
}
```

**No changes needed to:**

- `internal/config/` — Already works
- `internal/styles/` — Already works
- `internal/formatter/` — Domain-specific, fine as-is

### Data Flow: Before and After

**Before (current):**

```
Command --> (mixed) --> stdout/stderr
                \--> logger.Success --> stderr
                \--> fmt.Printf --> stdout
                \--> spinner (fetch only)
```

**After (proposed):**

```
Command --> logger.* --> stderr (status/feedback)
       \--> fmt.* --> stdout (data: list, status, --json)
       \--> StreamWriter --> stderr (hook output, prefixed)
```

Rule: Data goes to stdout. Feedback goes to stderr.

## Integration Points

### Where to Hook In

| Integration Point  | Change Required                 | Files Affected             |
| ------------------ | ------------------------------- | -------------------------- |
| Spinner extraction | Move to spinner.go, add methods | logger.go, new spinner.go  |
| Spinner adoption   | Add spinners to long ops        | add.go, clone.go, prune.go |
| Stream writer      | New file + hooks integration    | new stream.go, hooks.go    |
| Output consistency | Replace fmt.Printf with logger  | doctor.go, fetch.go        |
| Error messages     | Add context/suggestions         | All command files          |

### Dependency Direction

```
commands/*.go
    | imports
internal/logger/
    | imports
internal/config/     (for IsPlain)
internal/styles/     (for styling)
```

No circular dependencies. Logger can import config and styles safely.

## Patterns to Follow

### Pattern 1: Spinner for Long Operations

When to use: Any operation taking >500ms (network, git operations on large repos).

```go
stop := logger.StartSpinner("Cloning repository...")
if err := git.Clone(...); err != nil {
    stop()
    return err
}
stop()
logger.Success("Cloned to %s", path)
```

With new methods (preferred):

```go
spinner := logger.StartSpinner("Cloning repository...")
if err := git.Clone(...); err != nil {
    spinner.StopWithError("Clone failed")
    return err
}
spinner.StopWithSuccess(fmt.Sprintf("Cloned to %s", path))
```

### Pattern 2: Streaming Hook Output

Replace buffered execution:

```go
// Before
output, err := cmd.Output()  // Blocks, user sees nothing

// After
err := logger.StreamCommand(cmd, "npm install")  // Real-time with prefix
```

### Pattern 3: Actionable Errors

```go
// Before
return fmt.Errorf("branch %q not found", name)

// After
return fmt.Errorf("branch %q not found\n  hint: run 'grove list' to see available branches", name)
```

Hints go on separate line, indented. Use sparingly.

## Anti-Patterns to Avoid

### Anti-Pattern 1: Output in Business Logic

Don't put logger calls in `internal/git/` or `internal/workspace/`. Keep output in commands only.

```go
// BAD: internal/git/worktree.go
func CreateWorktree(...) error {
    logger.Info("Creating worktree...")  // NO
}

// GOOD: cmd/grove/commands/add.go
func runAdd(...) error {
    spinner := logger.StartSpinner("Creating worktree...")
    err := git.CreateWorktree(...)
    spinner.Stop()
}
```

### Anti-Pattern 2: Mixed stdout/stderr for Same Content

Pick one stream per content type:

- Data (list output, JSON) goes to stdout
- Feedback (spinners, success, errors) goes to stderr

```go
// BAD
fmt.Println("Created worktree")      // stdout
logger.Success("Created worktree")   // stderr

// GOOD: pick one
logger.Success("Created worktree")   // stderr for feedback
```

### Anti-Pattern 3: Inconsistent Plain Mode

Always use logger or styles.Render for anything with color/symbols.

```go
// BAD
fmt.Printf("\u2713 Done\n")  // Won't respect --plain

// GOOD
logger.Success("Done")   // Respects --plain
```

## Build Order

Suggested phase structure based on dependencies:

### Phase 1: Logger Extraction + Spinner Enhancement

1. Extract spinner to `internal/logger/spinner.go`
2. Add `StopWithSuccess`, `StopWithError`, `Update` methods
3. Update fetch.go to use new methods (prove it works)
4. No breaking changes to existing code

### Phase 2: Spinner Adoption

1. Add spinners to clone.go (multiple network operations)
2. Add spinners to add.go (PR fetch, git operations)
3. Add spinners to prune.go (fetch operation)
4. Each command independently

### Phase 3: Stream Writer + Hooks

1. Create `internal/logger/stream.go`
2. Modify `internal/hooks/hooks.go` to use streaming
3. Update add.go to stream hook output
4. Resolves issue #44

### Phase 4: Output Consistency Sweep

1. Audit doctor.go — move to logger functions
2. Audit fetch.go output functions — use logger
3. Check all commands for plain mode compliance
4. Resolves inconsistent output patterns

### Phase 5: Error Message Enhancement

1. Add hints to common errors
2. Input-to-output mapping for remove command (issue #68)
3. Suppress noisy git output where appropriate

## Testing Strategy

### Existing Test Patterns

Commands have `*_test.go` files using testscript for integration tests. Unit tests sparse.

### Recommended Additions

1. **Logger unit tests** — Test spinner state machine, stream writer output
2. **Plain mode tests** — Verify output differs with `--plain`
3. **Testscript updates** — Update expected output after changes

Test plain mode by setting environment:

```go
func TestPlainOutput(t *testing.T) {
    config.SetPlain(true)
    defer config.SetPlain(false)
    // Test output doesn't contain ANSI codes
}
```

## Sources

- Codebase analysis: `/home/sqve/code/personal/grove/main/internal/logger/logger.go`
- Codebase analysis: `/home/sqve/code/personal/grove/main/internal/styles/styles.go`
- Codebase analysis: `/home/sqve/code/personal/grove/main/cmd/grove/commands/*.go`
- Project context: `.planning/PROJECT.md`

**Confidence: HIGH** — Based entirely on codebase analysis, no external research needed.
