# Grove Technical Architecture & Standards

## Core Technology Stack

### Language & Runtime

- **Go 1.24.5**: Primary language for cross-platform CLI development
- **Minimum Go version**: 1.21+ for compatibility with CI/CD environments
- **Standard library first**: Prefer Go stdlib over external dependencies where possible

### CLI Framework & Dependencies

- **spf13/cobra**: CLI framework for command structure and argument parsing
- **spf13/viper**: Configuration management with TOML/JSON support and environment variables
- **pelletier/go-toml/v2**: TOML configuration file parsing
- **charmbracelet/lipgloss**: Terminal UI styling for consistent visual presentation
- **stretchr/testify**: Testing framework with rich assertions and mocking
- **magefile/mage**: Cross-platform build automation

### Development Tools

- **golangci-lint 1.50+**: Comprehensive linting with strict quality standards
- **gofumpt + goimports**: Automatic code formatting and import organization
- **pre-commit hooks**: Optional automated quality checks before commits

## System Architecture

Grove is organized into focused packages that do one thing well:

- **Commands** (`cmd/`): Cobra CLI command definitions and routing
- **Git operations** (`internal/git/`): Git command execution wrapper
- **Configuration** (`internal/config/`): TOML config and environment variables
- **Workspace management** (`internal/workspace/`): Create/clone grove workspaces
- **Validation** (`internal/validation/`): Input checking and sanitization  
- **Styling** (`internal/styles/`): Terminal output formatting
- **File operations** (`internal/fs/`): Filesystem permissions and paths

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
├── config/     // Configuration management
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

## Technology Choices & Justifications

### Go Language Selection

- **Cross-platform**: Single binary deployment across Windows, macOS, Linux
- **Performance**: Fast startup time critical for CLI responsiveness
- **Concurrency**: Goroutines for parallel git operations
- **Ecosystem**: Rich CLI tooling and git integration libraries

### Framework Decisions

#### Cobra CLI Framework

- **Industry standard**: Used by kubectl, docker, hugo
- **Rich features**: Subcommands, flags, help generation, shell completion
- **Extensibility**: Easy to add new commands and maintain consistency

#### Viper Configuration

- **Multiple formats**: TOML, JSON, YAML, environment variables
- **Hierarchical config**: Environment-specific overrides
- **Live reloading**: Config updates without restart

#### Lipgloss Styling

- **Consistent UI**: Unified color scheme and formatting
- **Accessibility**: Terminal compatibility and color adaptation
- **Professional appearance**: Enhanced user experience

### Testing Framework

- **Testify**: Rich assertions and mocking capabilities
- **Table-driven tests**: Comprehensive scenario coverage
- **Integration testing**: Real git repository testing

## Patterns & Best Practices

### Configuration Management

```go
// Configuration hierarchy (highest to lowest priority):
// 1. Command-line flags
// 2. Environment variables (GROVE_*)
// 3. Config file (~/.grove/config.toml)
// 4. Defaults
```

### Git Integration Patterns

```go
import (
	"context"
	"fmt"
	"os/exec"
)

// Good: Wrap git commands with context and error handling
func (g *GitClient) CreateBranch(ctx context.Context, name, from string) error {
	if err := g.validateBranchName(name); err != nil {
		return fmt.Errorf("invalid branch name %q: %w", name, err)
	}

	cmd := exec.CommandContext(ctx, "git", "checkout", "-b", name, from)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch %q from %q: %w", name, from, err)
	}
	return nil
}
```

### Progress Indication

```go
// Use lipgloss for consistent progress display
spinner := lipgloss.NewStyle().
    Foreground(lipgloss.Color("205")).
    Render("⠋ Creating worktree...")
```

### Validation Strategy

- **Early validation**: Validate inputs before any operations
- **Clear feedback**: Specific error messages with suggestions
- **Defensive programming**: Handle edge cases gracefully

## Security Guidelines

### Input Validation

- **Path traversal protection**: Validate all file paths
- **Command injection prevention**: Sanitize all external command arguments
- **Branch name validation**: Ensure git-safe branch names

### Secrets Management

- **No hardcoded secrets**: Use environment variables or config files
- **Config file permissions**: Restrict access to user-only (0600)
- **Credential helpers**: Integrate with system credential managers

## Performance Standards

### Command Response Times

- **Fast commands**: < 100ms (list, status, config)
- **Medium operations**: < 2s (create worktree, switch)
- **Long operations**: Progress indicators for > 2s operations

### Resource Efficiency

- **Memory usage**: Minimal heap allocation for simple commands
- **CPU usage**: Efficient git command execution
- **Disk usage**: Clean up temporary files promptly

### Scalability Considerations

- **Large repositories**: Handle repos with 1000+ worktrees
- **Concurrent operations**: Safe parallel worktree creation
- **Platform performance**: Consistent behavior across OS platforms

## Integration Standards

### External Tool Integration

- **Git compatibility**: Support git 2.20+ across all platforms
- **Shell integration**: Bash, zsh, fish completion scripts
- **Editor integration**: VSCode workspace file generation

### API Design (Future)

- **RESTful principles**: Clear resource-based endpoints
- **JSON responses**: Structured data for tooling integration
- **Versioning strategy**: Backward-compatible API evolution

## Monitoring & Observability

### Development Debugging

- **Structured logging**: JSON format for log aggregation
- **Debug mode**: Verbose output with --debug flag
- **Error tracking**: Context-rich error reporting

### Performance Monitoring

- **Command timing**: Optional performance metrics collection
- **Git operation profiling**: Identify slow git commands
- **Resource usage**: Memory and CPU utilization tracking

## Deployment & Distribution

### Build Process

- **Cross-compilation**: Automated builds for all target platforms
- **Static linking**: No external dependencies required
- **Reproducible builds**: Consistent binary generation

### Release Strategy

- **Semantic versioning**: Major.minor.patch version scheme
- **Release automation**: GitHub Actions for builds and publishing
- **Package managers**: Homebrew, apt, chocolatey distribution

This technical architecture provides the foundation for building a reliable, maintainable, and user-friendly Git worktree management tool that scales with user needs while maintaining simplicity and performance.
