# Architecture Research

**Domain:** CLI fetch command integration
**Researched:** 2026-01-23
**Confidence:** HIGH (based on direct codebase analysis)

## Component Boundaries

Grove follows a clear separation between command orchestration and git operations.

### Command Layer (`cmd/grove/commands/`)

Commands handle:

- **User interaction**: Cobra command definition, flags, args, help text
- **Validation**: Input validation, flag conflicts, prerequisite checks
- **Orchestration**: Calling git package functions in sequence
- **Output**: User-facing messages via `logger` package
- **Error wrapping**: Adding context before returning to user

Commands do NOT:

- Execute raw git commands directly (use `internal/git/` helpers)
- Contain low-level git logic
- Handle retries or timeouts (git package handles this)

### Git Layer (`internal/git/`)

The git package handles:

- **Command execution**: `GitCommand()` factory with timeout support
- **Output parsing**: Converting git output to structured data
- **Error handling**: Stderr capture, exit code interpretation
- **Low-level operations**: Individual git operations as pure functions

Existing fetch-related functions in `internal/git/git.go`:

```go
FetchPrune(repoPath string) error           // git fetch --prune
FetchBranch(repoPath, remote, branch string) error  // git fetch <remote> <branch>
```

### Workspace Layer (`internal/workspace/`)

The workspace package handles:

- **Path resolution**: Finding `.bare` directory, workspace root
- **Workspace operations**: Clone, convert, initialize
- **Cross-worktree concerns**: Locking, file preservation

## Data Flow

### Existing Pattern (from `prune.go`)

```
User runs `grove prune`
    |
    v
runPrune() in commands/prune.go
    |
    +-- workspace.FindBareDir(cwd)     // Get workspace context
    |
    +-- git.FetchPrune(bareDir)        // Fetch updates from remote
    |
    +-- git.ListWorktreesWithInfo()    // Get current state
    |
    +-- [business logic in command]     // Determine what to prune
    |
    +-- git.RemoveWorktree()           // Execute changes
    |
    v
logger.Success() / logger.Error()      // Report to user
```

### Proposed Fetch Command Flow

```
User runs `grove fetch [options]`
    |
    v
runFetch() in commands/fetch.go
    |
    +-- workspace.FindBareDir(cwd)     // Establish workspace context
    |
    +-- [determine scope: all remotes vs specific]
    |
    +-- git.Fetch*() functions         // Delegate to git package
    |   +-- For --all: git.FetchPrune(bareDir)
    |   +-- For specific: git.FetchBranch(bareDir, remote, branch)
    |   +-- For new remote: git.FetchRemote(bareDir, remote)  // NEW
    |
    +-- [optional: update worktree tracking info]
    |
    v
logger.Info/Success()                  // Report results
```

## Integration Points

### 1. Workspace Context

All commands establish workspace context first:

```go
cwd, err := os.Getwd()
if err != nil {
    return fmt.Errorf("failed to get current directory: %w", err)
}

bareDir, err := workspace.FindBareDir(cwd)
if err != nil {
    return err
}
```

### 2. Git Operations

Route through existing git package functions. For fetch, existing functions:

| Function            | Purpose                      | Location                  |
| ------------------- | ---------------------------- | ------------------------- |
| `git.FetchPrune()`  | Fetch all + prune stale refs | `internal/git/git.go:211` |
| `git.FetchBranch()` | Fetch specific branch        | `internal/git/git.go:242` |
| `git.ListRemotes()` | List configured remotes      | `internal/git/git.go:444` |

Likely needed new function:

```go
// FetchRemote fetches all branches from a specific remote
func FetchRemote(repoPath, remote string) error
```

### 3. Error Handling Pattern

Commands wrap errors with user context:

```go
if err := git.FetchPrune(bareDir); err != nil {
    // Non-fatal in some contexts (like prune)
    logger.Warning("Failed to fetch: %v", err)
}

// Or fatal:
if err := git.FetchBranch(bareDir, remote, branch); err != nil {
    return fmt.Errorf("failed to fetch branch %s: %w", branch, err)
}
```

### 4. Progress Indication

For long operations, use the spinner pattern:

```go
stop := logger.StartSpinner("Fetching remote changes...")
defer stop()

if err := git.FetchPrune(bareDir); err != nil {
    return err
}

stop()
logger.Success("Fetched remote changes")
```

### 5. Verbose Output

Commands support verbose mode for git output passthrough:

```go
// In git package - runGitCommand handles quiet mode
func runGitCommand(cmd *exec.Cmd, quiet bool) error {
    if quiet {
        return executeWithStderr(cmd)  // Capture output
    }
    cmd.Stdout = os.Stdout  // Pass through to user
    cmd.Stderr = os.Stderr
    return cmd.Run()
}
```

Fetch command should accept `--verbose` flag and pass `!verbose` to quiet parameter.

## Command Structure Template

Based on existing commands, fetch should follow this structure:

```go
package commands

func NewFetchCmd() *cobra.Command {
    var verbose bool
    var all bool
    var prune bool

    cmd := &cobra.Command{
        Use:   "fetch [remote]",
        Short: "Fetch updates from remote repositories",
        Long:  `Fetch updates from remote repositories...`,
        Args:  cobra.MaximumNArgs(1),
        RunE: func(cmd *cobra.Command, args []string) error {
            return runFetch(args, verbose, all, prune)
        },
    }

    // Flags follow consistent naming conventions
    cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show git output")
    cmd.Flags().BoolVarP(&all, "all", "a", false, "Fetch from all remotes")
    cmd.Flags().BoolVar(&prune, "prune", false, "Prune stale remote-tracking refs")

    return cmd
}

func runFetch(args []string, verbose, all, prune bool) error {
    // 1. Get workspace context
    cwd, err := os.Getwd()
    if err != nil {
        return fmt.Errorf("failed to get current directory: %w", err)
    }

    bareDir, err := workspace.FindBareDir(cwd)
    if err != nil {
        return err
    }

    // 2. Execute git operations
    // ...

    // 3. Report results
    logger.Success("Fetched updates")
    return nil
}
```

## Package Dependencies

Fetch command will import:

- `github.com/spf13/cobra` - Command definition
- `github.com/sqve/grove/internal/git` - Git operations
- `github.com/sqve/grove/internal/logger` - User output
- `github.com/sqve/grove/internal/workspace` - Workspace context

## Sources

All findings based on direct analysis of Grove codebase:

- `/home/sqve/code/personal/grove/main/internal/git/git.go`
- `/home/sqve/code/personal/grove/main/cmd/grove/commands/prune.go`
- `/home/sqve/code/personal/grove/main/cmd/grove/commands/clone.go`
- `/home/sqve/code/personal/grove/main/cmd/grove/commands/status.go`
- `/home/sqve/code/personal/grove/main/internal/workspace/workspace.go`
