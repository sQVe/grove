# Testing Patterns

**Analysis Date:** 2026-01-23

## Test Framework

**Runner:**

- gotestsum v1.13.0 (wraps `go test`)
- Config: None (uses defaults)

**Assertion Library:**

- Standard library `testing` package (no external assertion library)
- Direct comparisons with `t.Errorf()` / `t.Fatalf()`

**Run Commands:**

```bash
make test              # Run unit tests (default target)
make test-unit         # Run unit tests explicitly
make test-integration  # Run integration tests (requires git)
make test-coverage     # Run with coverage report
```

## Test File Organization

**Location:**

- Co-located with implementation (`*_test.go` next to `*.go`)

**Naming:**

- `{filename}_test.go` matches source file
- Example: `git.go` → `git_test.go`

**Structure:**

```
internal/
├── git/
│   ├── git.go
│   ├── git_test.go
│   ├── branch.go
│   ├── branch_test.go
│   └── ...
├── workspace/
│   ├── workspace.go
│   └── workspace_test.go
└── testutil/
    ├── testutil.go      # Shared test helpers
    └── git/
        └── git.go       # Git-specific test utilities
```

## Test Structure

**Suite Organization:**

```go
func TestFunctionName(t *testing.T) {
    t.Run("returns X when Y", func(t *testing.T) {
        // Arrange
        input := "test"

        // Act
        result := FunctionName(input)

        // Assert
        if result != expected {
            t.Errorf("expected %v, got %v", expected, result)
        }
    })

    t.Run("returns error for invalid input", func(t *testing.T) {
        err := FunctionName("")
        if err == nil {
            t.Error("expected error for empty input")
        }
    })
}
```

**Patterns:**

- Subtests with `t.Run()` for grouping related cases
- Table-driven tests for multiple input/output combinations
- Parallel tests with `t.Parallel()` where safe

**Table-driven example from `internal/workspace/workspace_test.go`:**

```go
func TestSanitizeBranchName(t *testing.T) {
    tests := []struct {
        branch   string
        expected string
    }{
        {"feature/add-button", "feature-add-button"},
        {"feat/user-auth", "feat-user-auth"},
        // ...
    }

    for _, tt := range tests {
        t.Run(tt.branch, func(t *testing.T) {
            result := SanitizeBranchName(tt.branch)
            if result != tt.expected {
                t.Errorf("expected '%s', got '%s'", tt.expected, result)
            }
        })
    }
}
```

## Test Utilities

**Location:** `internal/testutil/`

**TempDir helper** (`internal/testutil/testutil.go`):

```go
// TempDir returns a temp directory with symlinks resolved.
// Use instead of t.TempDir() when tests compare paths with git output.
func TempDir(t *testing.T) string {
    t.Helper()
    dir := t.TempDir()
    resolved, err := filepath.EvalSymlinks(dir)
    if err != nil {
        t.Fatalf("failed to resolve symlinks: %v", err)
    }
    return resolved
}
```

**TestRepo helper** (`internal/testutil/git/git.go`):

```go
type TestRepo struct {
    t    *testing.T
    Dir  string
    Path string
}

func NewTestRepo(t *testing.T, branchName ...string) *TestRepo
func (r *TestRepo) AddRemote(name, url string)
func (r *TestRepo) CreateBranch(name string)
func (r *TestRepo) Checkout(name string)
func (r *TestRepo) WriteFile(name, content string)
func (r *TestRepo) Add(name string)
func (r *TestRepo) Commit(message string)
func (r *TestRepo) Merge(branch string)
func (r *TestRepo) SquashMerge(branch string)
```

**Worktree cleanup** (Windows compatibility):

```go
func CleanupWorktree(t *testing.T, bareDir, worktreePath string) {
    t.Helper()
    t.Cleanup(func() {
        cmd := exec.Command("git", "worktree", "remove", "--force", worktreePath)
        cmd.Dir = bareDir
        _ = cmd.Run()
    })
}
```

## Mocking

**Framework:** None (real git operations)

**Patterns:**

- Use real git repositories in temp directories
- TestRepo helper provides controlled git state
- No interface-based mocking

**What to Mock:**

- Nothing - tests use real git commands

**What NOT to Mock:**

- Git operations (use TestRepo helper instead)
- Filesystem operations (use t.TempDir())

## Integration Tests

**Build tag:** `//go:build integration`

**Location:** `cmd/grove/testdata/script/*.txt`

**Framework:** `github.com/rogpeppe/go-internal/testscript`

**Script format example** (`cmd/grove/testdata/script/add_integration.txt`):

```txt
# grove add integration tests
# Tests worktree creation with real git state

# Setup: Create workspace via clone
mkdir testrepo
exec git init testrepo
cd testrepo
exec git config user.name "Test"
exec git config user.email "test@example.com"
exec git config commit.gpgsign false
cp ../README.md .
exec git add .
exec git commit -m 'initial commit'
cd ..

mkdir workspace
exec grove clone file://$WORK/testrepo workspace
cd workspace/main

## Success: Add existing branch

exec grove add existing-branch
stderr 'Created worktree at .*[/\\\\]existing-branch'
exists ../existing-branch

## Error: Branch already has worktree

! exec grove add existing-branch
stderr 'worktree already exists for branch'

-- README.md --
# Test
```

**Testscript setup** (`cmd/grove/script_test.go`):

```go
func TestScript(t *testing.T) {
    testscript.Run(t, testscript.Params{
        Dir: "testdata/script",
        Setup: func(env *testscript.Env) error {
            // Set up HOME, git config, GH_TOKEN
        },
        Condition: func(cond string) (bool, error) {
            switch cond {
            case "ghauth":
                return os.Getenv("GH_TOKEN") != "", nil
            }
            return false, nil
        },
    })
}
```

**Conditional tests:**

- `[ghauth]` - requires GitHub authentication for PR tests
- `[!windows]` - skip on Windows (path length limits)

## Coverage

**Requirements:** None enforced

**View Coverage:**

```bash
make test-coverage
# Output: coverage/coverage.out
# Summary printed to stdout
```

**CI behavior:**

- Coverage collected with `-covermode=atomic`
- Race detector enabled in CI (`-race` flag)

## Test Types

**Unit Tests:**

- Scope: Single function/package
- Location: `*_test.go` co-located with source
- Run: `make test` or `make test-unit`
- Characteristics: Fast, isolated, use TestRepo for git state

**Integration Tests:**

- Scope: Full command execution with real git
- Location: `cmd/grove/testdata/script/*.txt`
- Run: `make test-integration`
- Build tag: `//go:build integration`
- Characteristics: Uses testscript framework, creates real repositories

**E2E Tests:**

- Not used (integration tests serve this purpose)

## Common Patterns

**Async Testing:**

```go
// Not commonly used - most tests are synchronous
// Parallel subtests where independent:
t.Run("case", func(t *testing.T) {
    t.Parallel()
    // ...
})
```

**Error Testing:**

```go
t.Run("returns error for empty path", func(t *testing.T) {
    err := SomeFunction("")
    if err == nil {
        t.Fatal("expected error for empty path")
    }
})

t.Run("returns specific error type", func(t *testing.T) {
    _, err := FindBareDir(tempDir)
    if !errors.Is(err, ErrNotInWorkspace) {
        t.Errorf("expected ErrNotInWorkspace, got %v", err)
    }
})
```

**Directory/File Setup:**

```go
func TestSomething(t *testing.T) {
    // Use t.TempDir() for automatic cleanup
    tempDir := t.TempDir()

    // Or testutil.TempDir() when comparing with git paths
    tempDir := testutil.TempDir(t)

    // Create test files
    if err := os.WriteFile(
        filepath.Join(tempDir, "test.txt"),
        []byte("content"),
        fs.FileStrict,
    ); err != nil {
        t.Fatal(err)
    }
}
```

**Git repository setup:**

```go
func TestGitOperation(t *testing.T) {
    repo := testgit.NewTestRepo(t)

    // repo.Path is the repository directory
    // repo.Dir is the parent temp directory

    repo.CreateBranch("feature")
    repo.Checkout("feature")
    repo.WriteFile("new.txt", "content")
    repo.Add("new.txt")
    repo.Commit("add new file")
}
```

**Cleanup patterns:**

```go
// Automatic via t.TempDir()
tempDir := t.TempDir()

// Manual cleanup with t.Cleanup()
t.Cleanup(func() {
    _ = os.Remove(lockFile)
})

// Worktree cleanup (Windows file locks)
testgit.CleanupWorktree(t, bareDir, worktreePath)
```

## Test Naming

**Function names:**

- `TestFunctionName` - basic test
- `TestFunctionName_SpecificCase` - variant
- `TestNewXxxCmd` - command constructor tests

**Subtest names (t.Run):**

- Descriptive lowercase: `"returns error for empty path"`
- Use present tense: `"creates directory when missing"`
- Describe behavior, not implementation

---

_Testing analysis: 2026-01-23_
