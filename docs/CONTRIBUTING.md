# Contributing

## Setup

| Step        | Command                                                   |
| ----------- | --------------------------------------------------------- |
| **Clone**   | `git clone https://github.com/sqve/grove.git && cd grove` |
| **Install** | `go mod download`                                         |
| **Verify**  | `mage test:unit && mage lint && mage build:all`           |

### Prerequisites

- Go 1.21+, Git 2.5+, golangci-lint 1.50+
- Mage build system (installed automatically via `go run github.com/magefile/mage@latest`)
- `entr` for watch mode functionality (optional)

## Workflow

### Before committing

```bash
# Fast development workflow
mage test:unit                                 # Fast unit tests (~2s)
mage lint                                      # Run golangci-lint (with --fix)
mage build:all                                 # Build

# Full validation (CI-like)
mage ci                                        # Complete pipeline
```

### Testing workflow

```bash
# Development (fast feedback)
mage test:unit                                 # Unit tests (~2s)
mage test:coverage                             # Unit tests with coverage report

# Full validation
mage test:integration                          # Integration tests (~35s)
mage test:all                                  # All tests

# Debugging
mage test:watch                                # Watch for changes and run unit tests
```

### Git workflow

- **Commits**: [Conventional format](https://conventionalcommits.org) (`feat:`, `fix:`, `docs:`)
- **Branches**: `feature/name`, `fix/name`, `docs/name`
- **PRs**: Clear description, link issues, focused scope

## Code Standards

### Go conventions

- Follow [Effective Go](https://go.dev/doc/effective_go.html)
- Use `gofmt`, meaningful names, explicit error handling
- Add godoc comments for public functions

### Testing

- **Coverage**: Aim for 90%+ (currently 96.4% overall)
- **Types**: Unit tests (mocked), integration tests (real git operations)
- **Structure**:
    - `file_test.go` - Unit tests with mocked dependencies
    - `file_integration_test.go` - Integration tests with real git operations
- **Build tags**: Integration tests use the `//go:build integration` tag

## Architecture

### Structure

```
grove/
├── cmd/grove/           # CLI entry point
├── internal/
│   ├── commands/        # Command implementations (init.go)
│   ├── completion/      # Shell completion functionality
│   ├── config/          # Configuration management
│   ├── errors/          # Error handling and custom error types
│   ├── git/            # Git operations (operations.go)
│   ├── logger/          # Structured logging
│   ├── retry/           # Retry mechanisms for network operations
│   ├── testutils/       # Testing utilities and mocks
│   └── utils/          # Cross-platform utilities
│       ├── files.go     # File system operations
│       ├── git.go       # Git URL validation & repo checks
│       └── system.go    # System utilities
└── go.mod
```

### Principles

- **Direct git execution**: Use `os/exec`, parse output manually
- **Dependency injection**: `GitExecutor` interface for testable operations
- **Cross-platform**: Handle Windows/macOS/Linux with Go stdlib
- **Error handling**: Custom `GitError` with context and exit codes

### Build System

Grove uses [Mage](https://magefile.org/) for cross-platform build automation. Run `mage help` to see all available targets.

## Style Guidelines

### Automated Checks

- **Format**: `gofumpt` and `goimports` for consistent formatting
- **Lint**: `golangci-lint` with enabled checks including top-level comment punctuation
- **Tests**: Unit and integration tests with good coverage

### Code Standards

- **Comments**: End with periods for top-level declarations (enforced by `godot` linter)
- **Error messages**: No periods at end (following Go convention)
- **Naming**: Clear, descriptive names that don't require comments to understand
- **Code clarity**: No "clever tricks" - prioritize obvious, maintainable code
- **Log messages**: Sentence case without periods (`"checking git availability"`)
- **Documentation**: Public types and functions need comprehensive godoc comments

## Help

- **Features**: [FEATURES.md](FEATURES.md) for complete documentation
- **Issues**: Search existing before creating new
- **Questions**: Use GitHub Discussions

Grove aims to make Git worktrees accessible to all developers. Keep this in mind for features, errors, docs, and reviews.
