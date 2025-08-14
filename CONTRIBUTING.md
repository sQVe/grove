# Contributing to Grove

Thanks for contributing! Grove makes Git worktrees simple, and we want contributing to Grove to be simple too.

## Project Context

Before diving in, check out our steering documents to understand the project vision and standards:

- **[Product Vision](.spec-workflow/steering/product.md)** - Mission, target users, and value propositions
- **[Technical Standards](.spec-workflow/steering/tech.md)** - Architecture principles, tech stack, and patterns
- **[Project Structure](.spec-workflow/steering/structure.md)** - File organization and naming conventions

These documents guide all development decisions and ensure consistency across the project.

## Quick Setup

| Step        | Command                                               |
| ----------- | ----------------------------------------------------- |
| **Clone**   | `git clone https://github.com/sQVe/grove && cd grove` |
| **Install** | `go mod download`                                     |
| **Verify**  | `mage test:unit && mage lint && mage build:all`       |

**Prerequisites:** Go 1.21+, Git 2.5+, golangci-lint 1.50+, Mage build system

## Development Workflow

### Fast Development Loop

```bash
mage test:unit # Unit tests (~10s)
mage lint      # Auto-fix formatting
mage build:all # Verify builds
```

### Before Committing

```bash
mage test:unit && mage test:integration # Full test suite
mage ci                                 # Complete CI pipeline locally
```

### Pre-commit Hooks (Recommended)

```bash
# Install once per repository
brew install pre-commit # or pip install pre-commit
pre-commit install

# Now git commit automatically runs linting
```

## Code Standards

### Go Conventions

- Follow [Effective Go](https://go.dev/doc/effective_go.html)
- Use `gofmt`, meaningful names, explicit error handling
- Public functions need godoc comments ending with periods
- Error messages don't end with periods

### Git Workflow

- **Commits:** [Conventional format](https://conventionalcommits.org) (`feat:`, `fix:`, `docs:`)
- **Branches:** `feat/name`, `fix/name`, `docs/name`
- **PRs:** Clear description, focused scope

## Architecture Overview

```
grove/
├── cmd/grove/           # CLI entry point
├── internal/
│   ├── app/             # Root command setup
│   ├── commands/        # Command implementations (init, create, list)
│   ├── git/             # Git operations (Commander interface)
│   ├── testutils/       # Robust testing infrastructure
│   ├── config/          # Configuration management
│   ├── completion/      # Shell completion
│   ├── errors/          # Error handling
│   ├── logger/          # Structured logging
│   ├── utils/           # Cross-platform utilities
│   └── validation/      # Input validation
└── test/integration/    # End-to-end CLI tests
```

**Key Principles:**

- **Commander interface:** All git operations go through this for testability
- **Dependency injection:** Makes everything mockable for unit tests
- **Cross-platform:** Handle Windows/macOS/Linux gracefully
- **Binary caching:** Integration tests reuse built binaries (220,000x speedup!)

## Testing

Grove has 96.4% test coverage because we believe reliable Git tools matter. Here's how to keep it that way.

### When to Use Unit vs Integration Tests

**The Simple Rule:** Mock the git stuff for speed, use real git when it matters.

**Unit Tests** (`helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()`)

- Fast (~10s), use mocked Commander interface
- **Use when:**
    - Testing business logic without git operations
    - Verifying command construction: `git add file1 file2`
    - Parsing git output or error handling logic
    - Configuration parsing and validation
    - Error message formatting

**Integration Tests** (`helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()`)

- Slower (~90s), execute real grove binary with actual git
- **Use when:**
    - Testing complete CLI workflows end-to-end
    - Verifying git repository state changes
    - Testing cross-platform file operations
    - Validating user-facing error scenarios

### Decision Tree

Ask yourself: "Am I testing my logic or git's behavior?"

```
Testing command parsing logic? → Unit test (mock Commander)
Testing that a branch gets created? → Integration test (real git)
Testing error message formatting? → Unit test (mock the error)
Testing CLI shows correct error? → Integration test (real failure)
Testing path normalization? → Unit test (pure logic)
Testing file creation works? → Integration test (real filesystem)
```

### Examples

**Unit Test Pattern:**

```go
import (
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestCreateWorktree_ExistingBranch_Success(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	mockCommander := testutils.CreateMockGitCommander()
	creator := NewWorktreeCreator(mockCommander)

	// Mock branch existence check
	mockCommander.ExpectRunQuiet(".", []string{"show-ref", "--verify", "--quiet", "refs/heads/main"}).Return(nil)
	mockCommander.On("Run", ".", "worktree", "add", helper.CreateTempDir("test-worktree"), "main").
		Return([]byte(""), []byte(""), nil)

	err := creator.CreateWorktree("main", helper.CreateTempDir("test-worktree"), WorktreeOptions{})
	assert.NoError(t, err)
}
```

**Integration Test Pattern:**

```go
import (
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroveInitCommand(t *testing.T) {
	helper := testutils.NewIntegrationTestHelper(t).WithCleanFilesystem()

	// Test complete CLI behavior
	stdout, stderr, err := helper.ExecGrove("init", "test-repo")
	require.NoError(t, err)
	// Verify expected output (check actual grove init output)
	assert.NotEmpty(t, stdout)
}
```
