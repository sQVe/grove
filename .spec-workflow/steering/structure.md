# Grove Project Structure & Organization

## Directory Structure

### Root Level Organization
```
grove/
├── cmd/                    # CLI entry points and main applications
├── internal/              # Private application packages (not importable)
├── docs/                  # Project documentation  
├── test/                  # Test data, fixtures, and specialized test files
├── coverage/              # Test coverage reports and artifacts
├── bin/                   # Built binaries (generated)
├── .spec-workflow/        # Spec-driven development workflow files
├── go.mod                 # Go module definition
├── go.sum                 # Go module checksums
├── magefile.go           # Mage build system configuration
└── README.md             # Project overview and quick start
```

### Command Structure (`cmd/`)
```
cmd/
└── grove/                 # Main CLI application
    └── main.go           # Entry point with CLI initialization
```

**Conventions**:
- One subdirectory per CLI binary
- Keep main.go minimal - delegate to internal packages
- Binary name matches directory name

### Internal Package Organization (`internal/`)
```
internal/
├── app/                   # Application-level configuration and setup
│   └── root.go           # Root command and global app setup
├── commands/             # Command implementations
│   ├── commands.go       # Command registration and discovery
│   ├── config.go         # Configuration command implementation
│   ├── init/            # Init command module
│   ├── create/          # Create command module (complex commands get subdirs)
│   └── list/            # List command module
├── completion/           # Shell completion functionality
├── config/              # Configuration management
├── errors/              # Custom error types and handling
├── git/                 # Git operations and interfaces
├── logger/              # Structured logging
├── retry/               # Retry mechanisms and backoff strategies
├── testutils/           # Testing utilities and mocks
├── utils/               # General utilities and helpers
└── validation/          # Input validation and sanitization
```

### File Naming Conventions

#### Standard Files
- **Implementation**: `package.go` (main implementation)
- **Tests**: `package_test.go` (unit tests with mocks)
- **Integration tests**: `package_integration_test.go` (real dependencies)
- **Interfaces**: `interfaces.go` (when multiple interfaces per package)
- **Types**: `types.go` (when many types need organization)
- **Constants**: `constants.go` (package-level constants)

#### Specialized Files by Functionality
- **Options/Configuration**: `options.go`, `config.go`
- **Validation**: `validation.go`
- **Formatting/Presentation**: `formatter.go`, `presenter.go`
- **Service Layer**: `service.go` or `package_service.go`
- **Completion**: `completion.go`
- **Parsing**: `parsing.go`, `parser.go`

#### Complex Command Structure
For commands with multiple files, use subdirectories:
```
commands/
└── create/                # Complex command gets own directory
    ├── create.go          # Main command implementation
    ├── create_test.go     # Command-specific tests
    ├── options.go         # Command options and configuration
    ├── validation.go      # Input validation
    ├── completion.go      # Shell completion
    ├── service.go         # Business logic
    └── interfaces.go      # Command-specific interfaces
```

### Test Organization

#### Test File Structure
```
# Unit tests (with mocks)
package_test.go

# Integration tests (real dependencies) 
package_integration_test.go

# Specialized test files (when needed)
package_basic_test.go      # Basic functionality tests
package_error_test.go      # Error condition tests
package_validation_test.go # Input validation tests
package_benchmark_test.go  # Performance benchmarks
```

#### Test Data Organization
```
test/
├── integration/          # Integration test files
│   ├── integration_test.go
│   └── testdata/        # Test scripts and fixtures
│       ├── create_basic.txt
│       ├── create_error.txt
│       └── list_worktrees.txt
└── fixtures/            # Shared test data (if needed)
```

### Documentation Structure (`docs/`)
```
docs/
├── FEATURES.md           # Complete feature roadmap and status
├── CONTRIBUTING.md       # Development setup and guidelines  
├── testing.md           # Testing infrastructure and patterns
└── IMPROVEMENTS.md       # Technical improvements and optimizations
```

**Documentation Conventions**:
- **FEATURES.md**: Single source of truth for project roadmap
- **CONTRIBUTING.md**: Developer onboarding and workflow
- **testing.md**: Comprehensive testing documentation
- **README.md**: (Root level) User-facing quick start and overview

## Package Design Principles

### Package Responsibility
- **Single purpose**: Each package should have one clear responsibility
- **Clear boundaries**: Minimal coupling between packages
- **Interface-driven**: Define clear contracts between packages

### Import Organization
```go
import (
    // Standard library packages
    "context"
    "fmt"
    "os"
    
    // Third-party packages
    "github.com/spf13/cobra"
    "github.com/stretchr/testify/assert"
    
    // Local packages
    "github.com/sqve/grove/internal/config"
    "github.com/sqve/grove/internal/git"
)
```

### Package Naming
- **Lowercase**: All package names lowercase
- **Descriptive**: Package name should indicate its purpose
- **Singular**: Use singular nouns (`config`, not `configs`)
- **No underscores**: Avoid underscores in package names

## Configuration File Organization

### Spec Workflow Files (`.spec-workflow/`)
```
.spec-workflow/
├── steering/             # Project steering documents
│   ├── product.md       # Product vision and goals
│   ├── tech.md          # Technical standards
│   └── structure.md     # Project organization (this file)
├── specs/               # Feature specifications
│   └── feature-name/
│       ├── requirements.md
│       ├── design.md
│       └── tasks.md
├── commands/            # Generated task commands
├── templates/           # Document templates
└── spec-config.json     # Workflow configuration
```

### Development Configuration
```
.golangci.yml            # Linting configuration
.pre-commit-config.yaml  # Optional pre-commit hooks
magefile.go             # Build system configuration
CLAUDE.md               # Claude Code project instructions
```

## Code Organization Patterns

### Command Implementation Pattern
```go
// cmd/grove/main.go - Minimal main function
func main() {
    app.Execute()
}

// internal/app/root.go - Application setup
func Execute() {
    rootCmd := buildRootCommand()
    rootCmd.Execute()
}

// internal/commands/create/create.go - Command implementation
func NewCreateCommand() *cobra.Command {
    cmd := &cobra.Command{...}
    return cmd
}
```

### Interface Definition Pattern
```go
// internal/git/interfaces.go
type GitExecutor interface {
    Execute(ctx context.Context, args ...string) (string, error)
}

type GitCommander interface {
    CreateWorktree(path, branch string) error
    ListWorktrees() ([]Worktree, error)
}
```

### Service Layer Pattern
```go
// internal/commands/create/service.go
type CreateService struct {
    git    git.GitCommander
    config config.Manager
}

func (s *CreateService) CreateWorktree(opts Options) error {
    // Business logic implementation
}
```

### Error Handling Pattern
```go
// internal/errors/errors.go
type GitError struct {
    Operation string
    ExitCode  int
    Output    string
    Err       error
}

func (e GitError) Error() string {
    return fmt.Sprintf("git %s failed: %s", e.Operation, e.Err)
}
```

## File Size Guidelines

### Recommended File Sizes
- **Small files** (< 200 lines): Most implementation files
- **Medium files** (200-500 lines): Complex command implementations, service classes
- **Large files** (500+ lines): Only when cohesive functionality requires it

### When to Split Files
- **Multiple related types**: Create `types.go`
- **Many constants**: Create `constants.go`  
- **Complex validation**: Create `validation.go`
- **Multiple interfaces**: Create `interfaces.go`
- **Large command**: Create subdirectory with multiple files

## Workflow Integration

### Git Workflow
- **Branch naming**: `feature/name`, `fix/name`, `docs/name`
- **Commit messages**: Conventional Commits format (`feat:`, `fix:`, `docs:`)
- **File organization**: Maintain structure consistency across branches

### Development Workflow
- **Build commands**: Use Mage targets (`mage test:unit`, `mage lint`)
- **Code organization**: Follow established patterns for new features
- **Testing**: Maintain parallel unit/integration test structure

### Spec Workflow Integration
- **Feature development**: Use spec workflow for new features
- **File placement**: Follow structure.md guidelines during implementation
- **Documentation**: Update relevant docs/ files with changes

## Quality Standards

### Code Organization Quality
- **Consistent patterns**: Follow established patterns for similar functionality
- **Clear module boundaries**: Each package has well-defined responsibilities
- **Minimal coupling**: Dependencies flow in one direction where possible
- **Interface segregation**: Define focused, single-purpose interfaces

### File Organization Quality  
- **Logical grouping**: Related functionality grouped in same package/file
- **Clear naming**: File and package names indicate their purpose
- **Appropriate size**: Files are focused and not too large to understand
- **Consistent structure**: Similar files follow same organization patterns

### Documentation Organization
- **Up-to-date**: Documentation reflects current code organization
- **Complete coverage**: All packages and major functionality documented  
- **Clear navigation**: Easy to find relevant documentation
- **Examples provided**: Code examples follow documented patterns