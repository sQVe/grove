# Grove Project Structure

## Directory Structure

```
grove/
├── cmd/                   # Application entry point
│   └── grove/             # Main application
│       ├── main.go        # Application bootstrap
│       ├── commands/      # Command implementations (init.go, clone.go, config.go)
│       └── testdata/      # CLI test scripts
├── internal/              # Private application code
│   ├── git/               # Git operations wrapper
│   ├── config/            # Environment variable configuration
│   ├── workspace/         # Grove workspace operations
│   ├── styles/            # Terminal styling with lipgloss
│   ├── logger/            # Logging utilities
│   └── fs/                # File system constants
├── .spec-workflow/        # Specification-driven development
│   ├── steering/          # Project steering documents
│   └── specs/             # Feature specifications
├── bin/                   # Built binaries (gitignored)
├── coverage/              # Test coverage reports (gitignored)
├── magefile.go            # Build automation
├── go.mod                 # Go module definition
├── go.sum                 # Dependency checksums
├── .golangci.yml          # Linter configuration
├── .gitignore             # Version control exclusions
├── README.md              # Project overview and quick start
└── CONTRIBUTING.md        # Contribution guidelines and setup
```

## Package Organization

### Commands

Currently implemented:

-   `grove init new [dir]` - Initialize empty workspace
-   `grove clone <url> [dir]` - Clone repository into workspace

### Internal Packages

-   **git/**: Wrapper around Git CLI commands
-   **config/**: Environment variables (`GROVE_PLAIN`, `GROVE_DEBUG`)
-   **workspace/**: Core workspace creation and management
-   **styles/**: Terminal output formatting with lipgloss
-   **logger/**: Debug and spinner output
-   **fs/**: File system constants and validation
-   **testutil/**: Testing utilities and helpers

## Development Workflow

### Testing

```bash
mage test:unit        # Fast unit tests
mage test:integration # Full integration tests
mage test:coverage    # Unit tests with coverage report
mage lint             # Run golangci-lint with --fix
mage format           # Format code and documentation files
mage build:dev        # Build development binary
mage ci               # Full CI pipeline locally
```

### Requirements

-   Go 1.21+
-   golangci-lint 1.50+
-   mage
-   gotestsum

### Standards

-   90% test coverage requirement
-   Table-driven tests for multiple scenarios
-   Error messages must be user-friendly
-   Fail fast on invalid input

## File Naming Conventions

-   **Core code**: Descriptive nouns (`workspace.go`, `git.go`)
-   **Tests**: Parallel structure (`workspace_test.go`, `git_test.go`)
-   **Test data**: Root-level `testdata/` for integration tests

## Git Workflow

### Branch Naming

-   **Features**: `feat/short-description`
-   **Bug fixes**: `fix/short-description`
-   **Chores**: `chore/short-description`

### Commit Format

```
type(scope): brief description
```

Types: `feat`, `fix`, `refactor`, `test`, `docs`, `chore`
