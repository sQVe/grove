# Project Structure

## Directory Layout

### Standard Go Project Structure

```
grove/
├── cmd/                    # Application entry points
│   └── grove/             # Main CLI application
├── internal/              # Private application code
│   ├── commands/          # CLI command implementations
│   ├── config/           # Configuration management
│   ├── git/              # Git operations and worktree logic
│   ├── utils/            # Shared utilities
│   ├── completion/       # Shell completion logic
│   ├── errors/           # Error handling and definitions
│   ├── logger/           # Logging infrastructure
│   ├── retry/            # Retry mechanisms
│   └── testutils/        # Testing utilities and mocks
├── docs/                 # Documentation
└── test/                 # Test files and fixtures
    ├── integration/      # Integration test suite
    └── unit/            # Unit test helpers
```

## File Organization Conventions

### Package Structure

- **One concept per package**: Each package should have a single, clear responsibility
- **Internal packages**: Use `/internal` for all application-specific code
- **Shared utilities**: Place reusable code in `/internal/utils`
- **Test organization**: Keep tests close to implementation (`*_test.go`)

### Command Implementation

- **Location**: All CLI commands in `/internal/commands/`
- **Pattern**: One file per command (`init.go`, `list.go`, `config.go`)
- **Tests**: Command tests alongside implementation
- **Integration tests**: Separate integration test files with `_integration_test.go` suffix

### Naming Conventions

- **Files**: Use snake_case for multi-word file names
- **Packages**: Use lowercase, single words when possible
- **Types**: Use PascalCase for exported types, camelCase for unexported
- **Functions**: Follow Go conventions (PascalCase for exported, camelCase for unexported)

## Testing Structure

### Test Organization

- **Unit tests**: `*_test.go` files alongside implementation
- **Integration tests**: `*_integration_test.go` for cross-component testing
- **Mock tests**: `*_mock_test.go` for tests requiring mocks
- **Test utilities**: Centralized in `/internal/testutils/`

### Test File Patterns

- **Test coverage**: Maintain 90%+ coverage across all packages
- **Mock consolidation**: Use centralized mocks from `/internal/testutils/mocks.go`
- **Fixtures**: Test data and setup helpers in `/internal/testutils/fixtures.go`

## Development Workflow Files

### Build and Development

- **Mage targets**: Defined in `magefile.go` at project root
- **Configuration**: Go modules (`go.mod`, `go.sum`)
- **Documentation**: Comprehensive docs in `/docs/` directory

### Code Quality

- **Linting**: golangci-lint configuration (existing setup)
- **Testing**: Use Mage targets (`mage test:unit`, `mage test:integration`)
- **Coverage**: HTML coverage reports generated to `coverage.html`

## Configuration Structure

### File Placement

- **Global config**: System-appropriate config directories (OS-specific)
- **Project config**: `.grove/` directory in project root
- **Environment**: Support `GROVE_*` environment variables
- **Defaults**: Defined in `/internal/config/defaults.go`

### Configuration Hierarchy

1. Command-line flags (highest priority)
2. Environment variables (`GROVE_*`)
3. Configuration files (TOML/JSON)
4. Default values (lowest priority)

## Error Handling Patterns

### Error Organization

- **Error definitions**: Centralized in `/internal/errors/`
- **Error codes**: Standardized error codes with context
- **Error wrapping**: Use `/internal/errors/wrap.go` utilities
- **User-facing errors**: Clear, actionable error messages

### Logging Patterns

- **Global logger**: Available via `/internal/logger/global.go`
- **Structured logging**: Use structured format for debugging
- **Log levels**: Support configurable log levels
- **Context**: Include operation context in log messages
