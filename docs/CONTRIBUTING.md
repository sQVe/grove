# Contributing

## Setup

| Step        | Command                                                      |
| ----------- | ------------------------------------------------------------ |
| **Clone**   | `git clone https://github.com/sqve/grove.git && cd grove`    |
| **Install** | `go mod download`                                            |
| **Verify**  | `go test ./... && golangci-lint run && go build ./cmd/grove` |

### Prerequisites

- Go 1.21+, Git 2.5+, golangci-lint v2.0+

**Install golangci-lint**: 
```bash
# macOS
brew install golangci-lint

# Linux/Windows
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Configuration**: Project uses `.golangci.yml` with recommended v2 format and essential linters for Go CLI development.

## Workflow

### Before committing

```bash
go fmt ./...                                   # Format
golangci-lint run                              # Lint
go test -race -coverprofile=coverage.out ./... # Unit tests
go test -tags=integration ./...                # Integration tests
go build ./cmd/grove                           # Build
```

### Git workflow

- **Commits**: [Conventional format](https://conventionalcommits.org) (`feat:`, `fix:`, `docs:`)
- **Branches**: `feature/name`, `fix/name`, `docs/name`
- **PRs**: Clear description, link issues, focused scope

## Code Standards

### Go conventions

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt`, meaningful names, explicit error handling
- Add godoc comments for public functions

### Testing

- **Coverage**: Aim for 90%+ (currently 85.6% overall: 94.3% utils, 86.4% commands, 85.0% git)
- **Types**: Unit tests (mocked), integration tests (real git operations)
- **Structure**: 
  - `file_test.go` - Unit tests with mocked dependencies
  - `file_integration_test.go` - Integration tests with real git operations
- **Build tags**: Integration tests use `//go:build integration` tag
- **Running tests**:
  - Unit tests: `go test ./...`
  - Integration tests: `go test -tags=integration ./...`
  - All tests: `go test -tags=integration ./...`

## Architecture

### Structure

```
grove/
├── cmd/grove/           # CLI entry point
├── internal/
│   ├── commands/        # Command implementations (init.go)
│   ├── git/            # Git operations (operations.go)
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

### Git Operations

| Command            | Purpose                                     |
| ------------------ | ------------------------------------------- |
| `git init --bare`  | Initialize repositories in `.bare/` subdirs |
| `git clone --bare` | Clone with worktree structure               |
| `git config`       | Configure remotes and fetch specs           |
| `git rev-parse`    | Repository validation                       |

### Build System: Mage

**Why**: Cross-platform, Go-native, modern best practice
**Tasks**: Build, Test, TestCoverage, Lint, Fmt, Clean, Install

## Current Priorities

1. **Core operations**: Implement worktree management
2. **Cross-platform**: Ensure Windows/macOS/Linux compatibility
3. **Configuration**: TOML-based config with validation
4. **Code quality**: Address remaining linting issues (3 current)

## Debug Logging

Grove provides structured debug logging to troubleshoot git operations and repository issues.

### Enable Debug Logging

```bash
# Command line flag
grove --debug init https://github.com/user/repo.git

# Environment variable  
GROVE_DEBUG=1 grove init https://github.com/user/repo.git

# JSON format for parsing
grove --log-format=json --debug init
```

### Key Components

- **`init_command`**: Repository initialization and cloning
- **`git_executor`**: All git command execution with timing
- **`default_branch`**: Multi-tier branch detection strategy
- **`git_utils`**: Repository validation and URL parsing
- **`system_utils`**: Git availability checks

### Common Issues

#### Git Not Found
```
level=ERROR component=system_utils msg="git not found in PATH"
```
**Solution**: Install git or add to PATH

#### Network Timeouts
```
level=DEBUG component=default_branch msg="context deadline exceeded"
```
**Solution**: Check network connectivity or use local repository

#### Repository Not Found
```
level=ERROR component=git_executor msg="repository not found"  
```
**Solution**: Verify URL is correct and accessible

### Debug Log Format

```
time=2024-01-01T12:00:00Z level=DEBUG msg="git command" component=git_executor git_command=clone duration=2.1s
```

Key attributes: `component` (source), `duration` (timing), `error` (details), `git_command`/`git_args` (exact commands)

### Style Guidelines

- **Component naming**: snake_case (`git_utils`, `init_command`)
- **Log messages**: Sentence case without periods (`"checking git availability"`)
- **Structured attributes**: Consistent naming (`duration`, `component`, `error`)
- **Log levels**: Debug (detailed flow), Info (major operations), Warn (fallbacks), Error (failures)

## Help

- **Features**: [FEATURES.md](FEATURES.md) for complete documentation
- **Issues**: Search existing before creating new
- **Questions**: Use GitHub Discussions

Grove aims to make Git worktrees accessible to all developers. Keep this in mind for features, errors, docs, and reviews.
