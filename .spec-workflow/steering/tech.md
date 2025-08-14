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

## Architectural Principles

### 1. Direct Git Execution
- **Pattern**: Use `os/exec` to execute git commands directly, parse output manually
- **Rationale**: Maintains compatibility across Git versions, avoids dependency on Git libraries
- **Implementation**: Structured command execution through `GitExecutor` interface

### 2. Dependency Injection
- **Pattern**: Define interfaces for external dependencies (Git operations, file system, etc.)
- **Rationale**: Enables comprehensive testing with mocked dependencies
- **Key Interfaces**: 
  - `GitExecutor`: Git command execution
  - `GitCommander`: High-level Git operations
  - `FileManager`: File system operations

### 3. Cross-Platform Design
- **Pattern**: Use Go standard library for platform abstraction
- **Implementation**: Handle Windows/macOS/Linux differences through utility functions
- **Testing**: Validate behavior on all target platforms

### 4. Error-First Design
- **Pattern**: Custom error types with context and structured information
- **Implementation**: `GitError` type with exit codes, command context, and user-friendly messages
- **User Experience**: Clear, actionable error messages without technical jargon

### 5. Interface-Based Architecture
- **Pattern**: Define clear interfaces between modules
- **Benefits**: Testability, modularity, clear separation of concerns
- **Examples**: Command interfaces, Git operation interfaces, configuration interfaces

## Code Standards

### Go Conventions
- **Follow Effective Go**: Adhere to official Go style guidelines
- **Meaningful names**: Variable and function names should be self-documenting
- **Explicit error handling**: All errors must be handled explicitly, no silent failures
- **Godoc comments**: All public functions and types require comprehensive documentation

### Comment Standards
- **Top-level declarations**: Must end with periods (enforced by `godot` linter)
- **Error messages**: No periods at end (following Go convention)
- **Code clarity priority**: Write obvious code that doesn't require comments to understand
- **Log messages**: Sentence case without periods (`"checking git availability"`)

### Testing Standards
- **Coverage target**: Maintain 90%+ test coverage (currently 96.4%)
- **Test types**:
  - **Unit tests**: `file_test.go` with mocked dependencies
  - **Integration tests**: `file_integration_test.go` with real Git operations
- **Build tags**: Integration tests use `//go:build integration`
- **Test structure**: Clear arrange-act-assert pattern with descriptive test names

### Code Organization
- **Package structure**: Clear separation between command logic, Git operations, and utilities
- **Internal packages**: Use `internal/` to prevent external dependency on implementation details
- **Single responsibility**: Each package and function should have one clear purpose
- **Import organization**: Standard library, third-party, local packages (enforced by goimports)

## Build & Development Standards

### Build System
- **Mage**: Cross-platform build automation with Go-based task definitions
- **Key targets**:
  - `mage test:unit`: Fast unit tests (~2s feedback)
  - `mage test:integration`: Comprehensive integration tests
  - `mage lint`: Code quality checks with auto-fix
  - `mage ci`: Full CI pipeline validation

### Quality Gates
- **Pre-commit validation**: `mage test:unit && mage lint && mage build:all`
- **CI validation**: Complete pipeline including integration tests
- **Coverage enforcement**: Fail builds if coverage drops below 90%
- **Cross-platform testing**: Validate on Windows, macOS, and Linux

### Performance Standards
- **Command responsiveness**: Core commands should complete within 1-2 seconds for typical repositories
- **Memory efficiency**: Minimal memory footprint for CLI operations
- **Parallel safety**: All operations must be safe for concurrent execution

## Configuration Architecture

### Configuration Sources (Priority Order)
1. **Command-line flags**: Highest priority for immediate overrides
2. **Environment variables**: `GROVE_*` prefixed variables
3. **Configuration files**: TOML/JSON in standard config directories
4. **Built-in defaults**: Sensible defaults for all configuration options

### Configuration Structure
- **Hierarchical**: Nested configuration sections (general, git, worktree, etc.)
- **Validation**: All configuration values validated on load with clear error messages
- **Cross-platform**: Platform-specific config directory handling
- **Environment integration**: Support for development, staging, production configurations

## Security & Reliability Standards

### Security Practices
- **Input validation**: All user inputs validated before processing
- **Path safety**: Prevent directory traversal and unsafe path operations
- **Credential handling**: Secure handling of Git credentials and authentication
- **Minimal permissions**: Request only necessary file system permissions

### Reliability Patterns
- **Retry mechanisms**: Exponential backoff for network operations
- **Graceful degradation**: Fallback behaviors for non-critical feature failures
- **Resource cleanup**: Proper cleanup of temporary files and resources
- **Atomic operations**: Git operations should be atomic where possible

### Logging Standards
- **Structured logging**: Use consistent log format with levels and context
- **User-friendly output**: Separate technical logging from user-facing messages
- **Debug information**: Comprehensive debug logging for troubleshooting
- **No sensitive data**: Never log credentials, tokens, or personal information

## Integration Guidelines

### Git Integration
- **Version compatibility**: Support Git 2.5+ (minimum for worktree support)
- **Command consistency**: Use git porcelain commands where stable, plumbing where necessary
- **State validation**: Verify Git repository state before operations
- **Error parsing**: Parse Git error messages to provide user-friendly feedback

### Shell Integration
- **Cross-shell support**: bash, zsh, fish, PowerShell completion support
- **Environment respect**: Honor user's shell configuration and preferences
- **Path handling**: Proper handling of paths with spaces and special characters

### Future Integration Readiness
- **API design**: Internal APIs designed for future external integrations
- **Plugin architecture**: Prepare for future plugin system
- **Configuration extensibility**: Config system ready for integration-specific settings

## Performance & Scalability

### Performance Targets
- **Small repositories** (<100 MB): Sub-second response times
- **Medium repositories** (100MB-1GB): 1-3 second response times  
- **Large repositories** (>1GB): Acceptable performance with progress indicators
- **Many worktrees** (10+): Efficient listing and management operations

### Scalability Considerations
- **Memory usage**: Bounded memory consumption regardless of repository size
- **Disk space**: Smart cleanup to prevent excessive disk usage
- **Network operations**: Efficient clone and fetch operations with progress feedback
- **Concurrent operations**: Safe handling of multiple concurrent Grove instances