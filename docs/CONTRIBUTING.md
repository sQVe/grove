# Contributing

## Setup

| Step        | Command                                                      |
| ----------- | ------------------------------------------------------------ |
| **Clone**   | `git clone https://github.com/sqve/grove.git && cd grove`    |
| **Install** | `go mod download`                                            |
| **Verify**  | `mage test:unit && mage lint:all && mage build:all`         |

### Prerequisites

- Go 1.21+, Git 2.5+, golangci-lint v2.0+
- Mage build system (installed automatically via `go run github.com/magefile/mage@latest`)
- `entr` for watch mode functionality (optional)

**Install golangci-lint**: Required for code linting and maintaining code quality standards. See [golangci-lint installation guide](https://golangci-lint.run/usage/install/) for platform-specific instructions.

**Install Mage**: Grove's build system for cross-platform development tasks. See [Mage installation guide](https://magefile.org/) for setup instructions, or use `go run github.com/magefile/mage@latest` to run without installation.

**Install entr**: Optional dependency for watch mode functionality during development. See [entr documentation](http://eradman.com/entrproject/) for installation instructions.

**Configuration**: Project uses `.golangci.yml` with recommended v2 format and essential linters for Go CLI development.

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
    - Unit tests: `mage test:unit`
    - Integration tests: `mage test:integration`
    - All tests: `mage test:all`

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

**Available targets**:
- **Test targets**: `unit`, `integration`, `all`, `coverage`, `watch`, `clean`
- **Build targets**: `all`, `release`, `clean`
- **Lint targets**: `lint`
- **Other targets**: `ci`, `dev`, `clean`, `info`, `help`

**Usage**: `mage <namespace>:<target>` (e.g., `mage test:unit`, `mage build:release`)

### Configuration

**Watch mode features**:
- Simple file discovery using `find` command
- Real-time test execution on file changes
- Requires `entr` tool for file watching
- Works across platforms with shell compatibility

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

#### Automated Checks
- **Format**: `gofmt` and `goimports` for consistent formatting
- **Lint**: `golangci-lint` with enabled checks including comment punctuation
- **Tests**: Unit and integration tests with good coverage

#### Manual Style Standards
- **Comments**: End with periods for top-level declarations (enforced by `godot` linter)
- **Error messages**: No periods at end (following Go convention)
- **Naming**: Clear, descriptive names that don't require comments to understand
- **Code clarity**: No "clever tricks" - prioritize obvious, maintainable code
- **Redundant comments**: Avoid comments that just restate what the code does

#### Grove-Specific Guidelines
- **Component naming**: snake_case for logging components (`git_utils`, `init_command`)
- **Log messages**: Sentence case without periods (`"checking git availability"`)
- **Structured attributes**: Consistent naming (`duration`, `component`, `error`)
- **Log levels**: Debug (detailed flow), Info (major operations), Warn (fallbacks), Error (failures)
- **Documentation**: Public types and functions need comprehensive godoc comments

#### Manual Validation Checklist
Before submitting changes, review for:
- [ ] Comments on public declarations end with periods
- [ ] Error messages don't end with periods
- [ ] No redundant comments (explaining what the code obviously does)
- [ ] Complex patterns extracted to named constants with explanatory comments
- [ ] Function/variable names are self-explanatory
- [ ] Public APIs have complete godoc documentation

## Help

- **Features**: [FEATURES.md](FEATURES.md) for complete documentation
- **Issues**: Search existing before creating new
- **Questions**: Use GitHub Discussions

Grove aims to make Git worktrees accessible to all developers. Keep this in mind for features, errors, docs, and reviews.
