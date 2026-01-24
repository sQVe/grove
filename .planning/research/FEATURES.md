# Feature Landscape: CLI Output Polish

**Domain:** CLI output UX for Grove worktree manager
**Researched:** 2026-01-24
**Confidence:** HIGH (based on clig.dev, GitHub CLI patterns, existing Grove codebase)

## Table Stakes

Features users expect from polished CLI output. Missing these makes Grove feel unfinished.

| Feature                              | Why Expected                                       | Complexity | Existing? | Notes                                             |
| ------------------------------------ | -------------------------------------------------- | ---------- | --------- | ------------------------------------------------- |
| Spinner for long operations          | Users need feedback during network/disk operations | Low        | Partial   | `logger.StartSpinner()` exists but only for fetch |
| Clear success/error indicators       | Visual distinction between outcomes                | Low        | Yes       | `logger.Success()`, `logger.Error()` with icons   |
| Plain mode for scripts               | Disable colors/spinners when piped                 | Low        | Yes       | `--plain` flag, `isPlain()` check                 |
| Show what was affected               | "Deleted worktree X" should show path              | Low        | No        | Issue #68 - remove output unclear                 |
| Stream subprocess output             | Hook output should appear in real-time             | Medium     | No        | Issue #44 - hooks buffer until complete           |
| Stderr for progress, stdout for data | Separation for piping                              | Low        | Yes       | Spinners use stderr                               |
| Error messages with context          | "Failed to X: because Y"                           | Low        | Partial   | Some errors lack actionable guidance              |
| Non-TTY graceful degradation         | Works in CI, scripts, cron                         | Low        | Yes       | Plain mode fallback                               |

## Differentiators

Features that elevate Grove above basic CLI tools. Not expected, but valuable.

| Feature                            | Value Proposition                                      | Complexity | Existing? | Notes                                        |
| ---------------------------------- | ------------------------------------------------------ | ---------- | --------- | -------------------------------------------- |
| Consistent message vocabulary      | Same verbs across commands (Created, Deleted, Updated) | Low        | Partial   | Some inconsistency exists                    |
| Structured output (--json)         | Machine-readable for tooling                           | Medium     | Partial   | Only on fetch command                        |
| Verbose mode (-v)                  | Show more details when requested                       | Low        | Partial   | Only on fetch command                        |
| Multi-step operation feedback      | "Step 1/3: Fetching..."                                | Medium     | No        | Would help grove add with hooks              |
| Summary after batch operations     | "Removed 3 worktrees"                                  | Low        | No        | Currently logs each individually             |
| Contextual hints                   | "Use --force to remove dirty worktree"                 | Low        | Partial   | Some commands have hints                     |
| Time estimates for long ops        | "Fetching... (~30s remaining)"                         | High       | No        | Rarely worth the complexity                  |
| Progress bars for known-length ops | File downloads, large operations                       | High       | No        | Grove operations are typically indeterminate |

## Anti-Features

Features to deliberately NOT build. Common mistakes in CLI design.

| Anti-Feature                      | Why Avoid                                                 | What to Do Instead                                 |
| --------------------------------- | --------------------------------------------------------- | -------------------------------------------------- |
| Interactive prompts by default    | Breaks scripting, CI, automation                          | Use flags; only prompt with explicit --interactive |
| Emoji everywhere                  | Not universal, accessibility issues, looks unprofessional | Use simple ASCII symbols with Unicode fallback     |
| Color-coding semantic information | Colorblind users miss meaning                             | Use icons + color, not color alone                 |
| Animations in non-TTY             | Pollutes logs with escape codes                           | Already handled: plain mode                        |
| Overly verbose default output     | Drowns important information                              | Default to concise; use -v for verbose             |
| Custom progress bar library       | Maintenance burden, edge cases                            | Use proven library or simple spinner               |
| Chatty "helpful" messages         | "Did you know you can..." clutters output                 | Keep output minimal and purposeful                 |

## Feature Dependencies

```
Existing Infrastructure
    |
    +-- logger.StartSpinner() -----> Enhanced spinner with message updates
    |
    +-- logger.Success/Error() ----> Consistent output vocabulary
    |
    +-- config.IsPlain() ----------> Already handles non-TTY
    |
    +-- styles (lipgloss) ---------> Already has color definitions
```

### Implementation Dependencies

```
Issue #68 (remove output)
    Requires: No new infrastructure
    Uses: logger.Success() with path info

Issue #44 (hook streaming)
    Requires: Refactor hooks.RunAddHooks()
    Uses: cmd.Stdout = os.Stdout pattern
    Impact: hooks.go, add.go output handling
```

## Existing Grove Patterns

### Current logger API

```go
logger.Success("Created worktree at %s", path)  // Green checkmark
logger.Error("failed to remove: %v", err)       // Red X
logger.Warning("branch has unpushed commits")   // Yellow triangle
logger.Info("Fetching PR #%d...", n)            // Blue arrow
logger.Dimmed("secondary information")          // Gray text
logger.StartSpinner("Fetching...")              // Animated spinner
```

### Current pain points from issues

**#68 - grove remove output unclear:**

```
Current:  "Deleted worktree feat-auth"
Expected: "Deleted worktree feat-auth at ~/code/project/feat-auth"
          OR show path, branch, and any deleted branch info
```

**#44 - Hook output not streamed:**

```
Current:  User sees nothing until hooks complete
Expected: Real-time output as hooks run (npm install, etc.)
```

## Implementation Recommendations

### Priority 1: Fix Known Issues (Low Effort, High Impact)

1. **Remove command output (#68)**
    - Add path to deletion message
    - Consistent format: "Deleted worktree {branch} at {path}"
    - If --branch: "Deleted worktree and branch {branch} at {path}"

2. **Hook streaming (#44)**
    - Replace buffered capture with streaming
    - Use `cmd.Stdout = os.Stdout`, `cmd.Stderr = os.Stderr`
    - Keep spinner until first output, then let output flow
    - Show "[hook: {command}]" prefix for clarity

### Priority 2: Consistency Pass (Low Effort)

3. **Audit all commands for output consistency**
    - Same verb tense (past: "Created", "Deleted", "Updated")
    - Same information density (always show path for create/delete)
    - Same error format ("failed to X: Y")

4. **Extend --json to more commands**
    - list command (already shows data, easy to JSONify)
    - status command
    - doctor command

### Priority 3: Enhanced Feedback (Medium Effort)

5. **Multi-step spinners for grove add**
    - "Fetching branch..." -> "Creating worktree..." -> "Running hooks..."
    - Each step with its own spinner/completion

6. **Summary for batch operations**
    - grove remove x y z: "Removed 3 worktrees"
    - grove prune: "Pruned 2 stale worktrees"

## Go Libraries for CLI Output

### Already in use

- **lipgloss** (charmbracelet) - Styling, colors
- **cobra** - Command framework

### Recommended additions

None required. Current implementation is sufficient with minor refactoring.

### If needing more features later

- **bubbletea** - Interactive TUIs (overkill for Grove)
- **briandowns/spinner** - More spinner options (current is fine)
- **schollz/progressbar** - Progress bars (Grove ops are indeterminate)

## Sources

- [Command Line Interface Guidelines](https://clig.dev/) - Comprehensive CLI UX guide
- [Heroku CLI Style Guide](https://devcenter.heroku.com/articles/cli-style-guide) - Action command patterns
- [GitHub CLI Accessibility](https://github.blog/engineering/user-experience/building-a-more-accessible-github-cli/) - Screen reader considerations
- [CLI UX Progress Patterns](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays) - Spinner vs progress bar guidance
- [briandowns/spinner](https://github.com/briandowns/spinner) - Go spinner library
- [go-cmd/cmd](https://github.com/go-cmd/cmd) - Streaming subprocess output
- [Go os/exec](https://pkg.go.dev/os/exec) - Subprocess handling patterns
