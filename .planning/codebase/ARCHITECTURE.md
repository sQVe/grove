# Architecture

**Analysis Date:** 2026-01-23

## Pattern Overview

**Overall:** Command-Line Application with Layered Architecture

**Key Characteristics:**

- Cobra-based CLI with subcommands (add, clone, list, etc.)
- Clear separation between command handlers and business logic
- Domain packages in `internal/` encapsulate git operations, filesystem, and configuration
- No database; operates directly on filesystem and git repositories

## Layers

**Commands Layer:**

- Purpose: Parse CLI arguments, validate flags, orchestrate business logic
- Location: `cmd/grove/commands/`
- Contains: Cobra command definitions, flag handling, user-facing output
- Depends on: All internal packages
- Used by: `cmd/grove/main.go` (root command)

**Workspace Layer:**

- Purpose: Manage grove workspace lifecycle (init, clone, convert)
- Location: `internal/workspace/`
- Contains: Workspace validation, worktree creation orchestration, file preservation
- Depends on: `internal/git`, `internal/fs`, `internal/config`, `internal/logger`
- Used by: Commands layer

**Git Layer:**

- Purpose: Abstract git CLI operations
- Location: `internal/git/`
- Contains: Branch operations, worktree management, status queries, remote handling
- Depends on: `internal/fs`, `internal/logger`, `internal/config`
- Used by: Commands layer, Workspace layer

**GitHub Layer:**

- Purpose: Integrate with GitHub via gh CLI
- Location: `internal/github/`
- Contains: PR fetching, repo URL parsing, authentication checks
- Depends on: None (pure functions and exec calls)
- Used by: Commands layer (add, clone)

**Config Layer:**

- Purpose: Manage configuration from git config and TOML files
- Location: `internal/config/`
- Contains: Global config state, TOML parsing, config precedence logic
- Depends on: None
- Used by: All layers

**Infrastructure Packages:**

- `internal/fs/`: Cross-platform filesystem utilities
- `internal/logger/`: Structured logging with color/plain modes
- `internal/styles/`: Lipgloss-based terminal styling
- `internal/formatter/`: Output formatting for worktree lists
- `internal/hooks/`: Shell hook execution
- `internal/version/`: Build version info

## Data Flow

**Add Worktree Flow:**

1. `commands/add.go` parses flags and validates input
2. `workspace.FindBareDir()` locates `.bare` directory
3. `workspace.AcquireWorkspaceLock()` prevents concurrent modifications
4. `git.CreateWorktree()` executes `git worktree add`
5. `workspace.PreserveFilesToWorktree()` copies ignored files
6. `hooks.RunAddHooks()` executes configured hooks
7. `logger.Success()` reports result to user

**Clone Repository Flow:**

1. `commands/clone.go` determines clone type (URL, PR, GitHub)
2. For GitHub: `github.CheckGhAvailable()` verifies gh CLI
3. `workspace.ValidateAndPrepareDirectory()` ensures target is empty
4. `git.Clone()` or `gh repo clone` creates bare clone in `.bare`
5. `git.ConfigureFetchRefspec()` sets up remote tracking
6. `workspace.CreateWorktreesFromBranches()` creates initial worktrees
7. `.git` file created pointing to `.bare`

**State Management:**

- Global config loaded once at startup via `config.LoadFromGitConfig()`
- Per-workspace config loaded on demand from `.grove.toml` in worktree
- Config precedence: CLI flags > git config > TOML file > defaults
- No persistent state beyond git repository and config files

## Key Abstractions

**WorktreeInfo:**

- Purpose: Represent worktree state for display/operations
- Examples: `internal/git/worktree.go`
- Pattern: Data transfer object with computed properties

**CloneFunc:**

- Purpose: Abstract different clone mechanisms (git, gh CLI)
- Examples: `internal/workspace/workspace.go`
- Pattern: Function type for dependency injection

**FileConfig:**

- Purpose: Represent `.grove.toml` configuration
- Examples: `internal/config/file.go`
- Pattern: Struct with TOML tags for deserialization

## Entry Points

**Main Entry:**

- Location: `cmd/grove/main.go`
- Triggers: User invokes `grove` command
- Responsibilities: Load config, init logger, register commands, execute root

**Command Entry Points:**

- Location: `cmd/grove/commands/*.go`
- Triggers: Subcommand invocation (e.g., `grove add`)
- Responsibilities: Validate input, call business logic, format output

## Error Handling

**Strategy:** Return errors up the call stack; commands display via logger

**Patterns:**

- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Sentinel errors for expected conditions: `workspace.ErrNotInWorkspace`, `git.ErrDetachedHead`
- Cleanup on failure: deferred rollback functions in `workspace.Convert()`
- User-friendly hints: `git.HintGitTooOld()` suggests solutions

## Cross-Cutting Concerns

**Logging:**

- `internal/logger/` provides Debug, Info, Success, Warning, Error
- Output goes to stderr; only machine-readable output to stdout
- Plain mode strips colors/symbols for scripting

**Validation:**

- Commands validate flags before business logic
- `workspace.ValidateAndPrepareDirectory()` checks filesystem state
- Git operations validate paths/refs before executing

**Authentication:**

- GitHub operations require `gh auth status`
- No credentials stored by grove; delegates to gh CLI
- Remote operations respect git's credential helpers

---

_Architecture analysis: 2026-01-23_
