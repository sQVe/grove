# Coding Conventions

**Analysis Date:** 2026-01-23

## Naming Patterns

**Files:**

- Snake_case with platform suffix for OS-specific: `lock_unix.go`, `lock_windows.go`
- Test files: `*_test.go` co-located with implementation
- Single-purpose files: `git.go`, `workspace.go`, `config.go`

**Functions:**

- Exported: PascalCase (`FindBareDir`, `CreateWorktree`, `SanitizeBranchName`)
- Unexported: camelCase (`runAdd`, `executeWithStderr`, `matchGlobPattern`)
- Constructors: `New*Cmd()` for cobra commands (`NewAddCmd`, `NewCloneCmd`)
- Boolean checks: `Is*`, `Has*`, `Should*` (`IsPlain`, `HasSubmodules`, `ShouldAutoLock`)

**Variables:**

- Package-level errors: `Err*` prefix (`ErrNotInWorkspace`, `ErrGitTooOld`, `ErrNoUpstreamConfigured`)
- Constants: ALL_CAPS for public, camelCase for private
- Receivers: single letter matching type (`r *TestRepo`, `t *testing.T`)

**Types:**

- Domain types in `internal/git/types.go`: `BranchName`, `WorktreePath`, `RepoPath`, `RemoteName`
- Structs: PascalCase (`TestRepo`, `WorktreeInfo`, `PreserveResult`)

## Code Style

**Formatting:**

- gofumpt with extra rules enabled
- goimports for import organization
- Config: `.golangci.yml` lines 12-18

**Linting:**

- golangci-lint v2 with extensive linter set
- Key enabled linters: errcheck, staticcheck, govet, gosec, errorlint, testifylint
- goconst excluded from test files
- Run: `make lint` (auto-fixes locally, strict in CI)

**EditorConfig:**

- Tabs for Go files (indent_size: 4)
- Spaces for other files (indent_size: 2)
- UTF-8, trim trailing whitespace, final newline

## Import Organization

**Order:**

1. Standard library
2. External dependencies
3. Internal packages (project modules)

**Example from `cmd/grove/commands/add.go`:**

```go
import (
    "fmt"
    "os"
    "path/filepath"
    "strings"

    "github.com/spf13/cobra"
    "github.com/sqve/grove/internal/config"
    "github.com/sqve/grove/internal/fs"
    "github.com/sqve/grove/internal/git"
    "github.com/sqve/grove/internal/github"
    "github.com/sqve/grove/internal/hooks"
    "github.com/sqve/grove/internal/logger"
    "github.com/sqve/grove/internal/styles"
    "github.com/sqve/grove/internal/workspace"
)
```

**Path Aliases:**

- None used; full import paths throughout

## Error Handling

**Patterns:**

- Wrap errors with context: `fmt.Errorf("failed to X: %w", err)`
- Sentinel errors for expected conditions: `var ErrNotInWorkspace = errors.New("not in a grove workspace")`
- Check with `errors.Is()` and `errors.As()`
- Early validation with clear error messages

**Example from `internal/git/git.go`:**

```go
func AddRemote(repoPath, name, url string) error {
    if repoPath == "" {
        return errors.New("repository path cannot be empty")
    }
    if name == "" {
        return errors.New("remote name cannot be empty")
    }
    // ...
}
```

**Git command errors:**

- Capture stderr for context: `executeWithStderr(cmd)`
- Special handling for version-specific errors: `WrapGitTooOldError(err)`

## Logging

**Framework:** Custom logger in `internal/logger/logger.go`

**Patterns:**

- `logger.Debug()` - development info (controlled by --debug flag)
- `logger.Info()` - progress messages with arrow prefix
- `logger.Success()` - completion with checkmark
- `logger.Warning()` - non-fatal issues
- `logger.Error()` - failures

**Output destination:** All logging to stderr; stdout reserved for machine-readable output (e.g., `--switch` flag paths)

## Comments

**When to Comment:**

- Package-level doc comments on all exported symbols
- Complex logic explanations
- Platform-specific behavior notes
- Nolint directives with reason: `// nolint:gosec // Test helper with controlled input`

**JSDoc/TSDoc:**

- N/A (Go project)

**Go doc style:**

```go
// FindBareDir finds the .bare directory for a grove workspace
// by walking up the directory tree from the given path
func FindBareDir(startPath string) (string, error) {
```

## Function Design

**Size:**

- Keep functions focused on single responsibility
- Extract helpers for complex operations (e.g., `runAddFromBranch`, `runAddFromPR`, `runAddDetached`)

**Parameters:**

- Validate required params at function start
- Use named structs for 4+ parameters: `conversionOpts`
- Boolean flags grouped at end of parameter list

**Return Values:**

- `(value, error)` pattern consistently
- Named return values only when necessary for defer
- Multiple returns for complex state: `*conversionResult, error`

## Module Design

**Exports:**

- One package = one responsibility
- Minimize exported surface area
- Export types that need external use

**Barrel Files:**

- Not used; import specific packages directly

## Permission Constants

**Location:** `internal/fs/fs.go`

```go
const (
    DirStrict  = 0o750 // rwxr-x--- - gosec-compliant directory
    FileStrict = 0o600 // rw------- - gosec-compliant file
    DirGit     = 0o755 // rwxr-xr-x - git-compatible directory
    FileExec   = 0o755 // rwxr-xr-x - executable file
    FileGit    = 0o644 // rw-r--r-- - git-compatible file
)
```

**Usage:** Use `fs.DirGit`/`fs.FileGit` for git operations, `fs.DirStrict`/`fs.FileStrict` for sensitive files.

## Command Structure

**Pattern:** Each command in `cmd/grove/commands/` follows:

```go
func NewXxxCmd() *cobra.Command {
    // Local flag variables
    var flagName string

    cmd := &cobra.Command{
        Use:   "xxx [args]",
        Short: "Brief description",
        Long:  `Detailed description with examples.`,
        Args:  cobra.MaximumNArgs(1),
        ValidArgsFunction: completeXxxArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            return runXxx(args, flagName)
        },
    }

    // Flag definitions
    cmd.Flags().StringVar(&flagName, "flag", "", "Description")
    _ = cmd.RegisterFlagCompletionFunc("flag", completeFunc)

    return cmd
}

func runXxx(args []string, flag string) error {
    // Implementation
}

func completeXxxArgs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    // Tab completion
}
```

## Concurrency

**Global state:** Protected with sync.RWMutex in `internal/config/config.go`

```go
var globalMu sync.RWMutex

func IsPlain() bool {
    globalMu.RLock()
    defer globalMu.RUnlock()
    return Global.Plain
}
```

**Atomic operations:** Used for logger state (`atomic.Bool`)

---

_Convention analysis: 2026-01-23_
