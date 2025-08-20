# Grove Technical Architecture & Standards

## Core Technology Stack

### Language & Runtime

- **Go 1.24.5**: Primary language for cross-platform CLI development
- **Minimum Go version**: 1.21+ for compatibility with CI/CD environments
- **Standard library first**: Prefer Go stdlib over external dependencies where possible

### CLI Framework & Dependencies

- **spf13/cobra**: CLI framework for command structure and argument parsing
- **charmbracelet/lipgloss**: Terminal UI styling for consistent visual presentation
- **magefile/mage**: Cross-platform build automation

### Development Tools

- **golangci-lint 1.50+**: Comprehensive linting with strict quality standards
- **gofumpt + goimports**: Automatic code formatting and import organization
- **pre-commit hooks**: Optional automated quality checks before commits

## System Architecture

Grove is organized into focused packages that do one thing well:

- **Commands** (`cmd/`): Cobra CLI command definitions and routing
- **Git operations** (`internal/git/`): Git command execution wrapper
- **Configuration** (`internal/config/`): Environment variables
- **Workspace management** (`internal/workspace/`): Create/clone grove workspaces
- **Validation** (`internal/validation/`): Input checking and sanitization
- **Styling** (`internal/styles/`): Terminal output formatting
- **File operations** (`internal/fs/`): Filesystem permissions and paths

### Grove Workspace Architecture

Grove workspaces are the core architectural pattern that enables efficient Git worktree management. This structure separates Git repository data from working directories, allowing multiple isolated development environments.

#### Directory Structure

```
project-name/           # Grove workspace root
├── .bare/             # Bare Git repository (all Git objects & refs)
├── .git               # File containing "gitdir: .bare"
├── main/              # Worktree for main branch
├── feature-auth/      # Worktree for feature/auth branch
└── bugfix-login/      # Worktree for bugfix/login branch
```

#### Core Components

- **`.bare` directory**: Contains the complete Git repository as a bare repo (no working tree)
- **`.git` file**: Plain text file with content `gitdir: .bare` that redirects Git operations to the bare repository
- **Worktree directories**: Sibling directories containing working copies of different branches
- **Branch name sanitization**: Unsafe filesystem characters (`/`, `<`, `>`, `|`, `"`) are replaced with dashes

#### Key Benefits

- **Parallel development**: Work on multiple features simultaneously without stashing or branch switching
- **Isolated environments**: Each worktree maintains its own working directory and index state
- **Shared repository**: All worktrees share the same Git objects, avoiding duplication
- **Cross-platform compatibility**: Works consistently across Windows, macOS, and Linux filesystems

#### Implementation Details

- Workspace detection traverses parent directories looking for `.bare` directories or `.git` files with `gitdir: .bare` content
- Branch names are sanitized using `sanitizeBranchName()` to ensure filesystem compatibility
- Validation prevents initialization inside existing Git repositories or Grove workspaces
- Empty directory validation ensures clean workspace creation

This architecture provides the foundation for Grove's simple, branch-like worktree management while maintaining the performance and reliability of Git's underlying worktree implementation.

## Development Standards

### Code Quality Requirements

#### Test Coverage

- **Unit tests**: 90% minimum coverage requirement (enforced in CI)
- **Integration tests**: End-to-end workflow validation
- **Test organization**: `_test.go` files alongside implementation
- **Test data**: Use table-driven tests for multiple scenarios

#### Linting Standards

- **golangci-lint**: Comprehensive linting with --fix in development
- **gofumpt**: Stricter formatting than go fmt
- **goimports**: Automatic import organization
- **Security scanning**: gosec for security vulnerability detection

#### Error Handling

- **Fail fast**: Return errors immediately, don't silently ignore
- **Context preservation**: Wrap errors with additional context
- **User-friendly messages**: Clear, actionable error messages for CLI users
- **Logging**: Structured logging for debugging (development only)

### Coding Conventions

#### Package Organization

```go
// Good: Clear, focused packages
internal/
├── git/        // Git command wrapper
├── config/     // Environment variables
├── validation/ // Input validation
├── workspace/  // Workspace operations
├── styles/     // Terminal styling
├── logger/     // Logging
└── fs/         // File system constants

// Avoid: Vague or generic packages
internal/
├── app/        // What does this do?
├── domain/     // Meaningless prefix
├── utils/      // Too generic
└── service/    // Says nothing
└── common/     // Garbage dump
```

#### Function Design

```go
import "context"

// Good: Single responsibility, clear naming
func CreateWorktree(ctx context.Context, name string, branch string) error
func ListActiveWorktrees() ([]Worktree, error)
func ValidateWorktreeName(name string) error

// Avoid: Multiple responsibilities, unclear purpose
func DoWorkTreeStuff(args ...interface{}) interface{}
func Handle(req interface{}) interface{}
```

#### Error Patterns

```go
// Good: Descriptive, actionable errors
return fmt.Errorf("failed to create worktree %q: branch %q does not exist", name, branch)
return fmt.Errorf("worktree %q already exists at %s", name, path)

// Avoid: Generic, unhelpful errors
return errors.New("operation failed")
return err // without context
```
