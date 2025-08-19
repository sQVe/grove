# Grove Project Structure & Development Workflow

## Project Organization

### Directory Structure

```
grove/
├── cmd/                   # Application entry point
│   └── grove/             # Main application
│       └── main.go        # Application bootstrap
├── internal/              # Private application code
│   ├── git/               # Git operations wrapper
│   ├── config/            # Configuration management
│   ├── validation/        # Input validation and sanitization
│   ├── workspace/         # Grove workspace operations
│   ├── styles/            # Terminal styling with lipgloss
│   ├── logger/            # Logging utilities
│   └── fs/                # File system constants
├── testdata/              # Integration test repositories and fixtures
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

### Package Organization Principles

#### Internal Package Structure

```go
internal/
├── git/                 # Git operations wrapper  
├── config/              # Configuration management
├── validation/          # Input validation and sanitization
├── workspace/           # Grove workspace operations
├── styles/              # Terminal styling with lipgloss
├── logger/              # Logging utilities
└── fs/                  # File system constants
```

### Package Principles

- **Focused packages**: Each package does one thing well
- **Clear names**: Package names say what they do
- **Minimal dependencies**: Keep imports simple and direct

#### Command Structure

```go
cmd/grove/
├── main.go              # Application entry point
└── commands/            # Cobra command definitions
    ├── root.go          # Root command (launches TUI when no args)
    ├── init.go          # Initialize grove in repository
    ├── create.go        # Create new worktree
    ├── list.go          # List worktrees
    ├── switch.go        # Switch to worktree
    ├── remove.go        # Remove worktree
    └── config.go        # Configuration commands
```

Usage patterns:

- `grove` → Launch TUI (future)
- `grove create <branch>` → CLI command
- `grove list` → CLI command

### File Naming Conventions

#### Go Files

- **Core code**: Descriptive nouns (`manager.go`, `repository.go`)
- **Tests**: Parallel structure (`manager_test.go`, `repository_test.go`)
- **Interfaces**: Suffix with behavior (`reader.go`, `validator.go`)
- **Implementation**: Match interface name (`file_reader.go`, `name_validator.go`)

#### Test Files

- **Unit tests**: `*_test.go` alongside implementation, use mocks/stubs
- **Integration tests**: Centralized in `testdata/` directory with complete test scenarios
- **Test data**: Root-level `testdata/` contains all test git repositories and fixtures
- **Table tests**: Use descriptive test case names

#### Configuration Files

- **Development**: `.golangci.yml`, `magefile.go`, `.gitignore`
- **Runtime**: `~/.grove/config.toml`, `.grove.toml` (project-specific)
- **CI/CD**: `.github/workflows/` (future)

## Development Workflow

### Git Workflow

#### Branch Strategy

```
main                     # Production-ready code
├── feat/auth-system     # Feature development
├── fix/worktree-cleanup # Bug fixes
└── chore/update-deps    # Maintenance work
```

#### Branch Naming Convention

- **Features**: `feat/short-description`
- **Bug fixes**: `fix/short-description`
- **Chores**: `chore/short-description`
- **Experiments**: `experiment/short-description`

#### Commit Message Format

```
type(scope): brief description

Longer explanation if needed, including:
- What was changed and why
- Any breaking changes
- References to issues or specs

Closes #123
```

**Commit Types**:

- `feat`: New feature
- `fix`: Bug fix
- `refactor`: Code restructuring without behavior change
- `test`: Adding or modifying tests
- `docs`: Documentation changes
- `chore`: Build process, dependency updates

### Code Review Process

#### Pull Request Requirements

- **All tests pass**: Unit and integration tests must pass
- **Coverage maintained**: 90% minimum coverage enforced
- **Linting clean**: No golangci-lint violations
- **Clear description**: What, why, and how of the changes
- **Small scope**: Single responsibility per PR

#### Review Checklist

- [ ] Code follows architecture patterns from tech.md
- [ ] Error handling is consistent and user-friendly
- [ ] Tests cover edge cases and error scenarios
- [ ] Performance impact considered for large repositories
- [ ] Security implications reviewed (input validation, command injection)
- [ ] Documentation updated if needed

### Testing Workflow

#### Test Organization

```
# Unit Tests (fast, isolated)
go test -tags='!integration' -short ./...

# Integration Tests (slower, requires git)
go test -tags=integration ./...

# Coverage Report
go test -coverprofile=coverage/coverage.out ./...
go tool cover -html=coverage/coverage.out
```

#### Test Categories

- **Unit tests**: Core code, validation, configuration using mocks/stubs
- **Integration tests**: End-to-end workflows run from testdata/ with real git repositories
- **Table tests**: Multiple scenarios with data-driven approach
- **Error tests**: Validation of error conditions and messages

#### Mage Commands for Development

```bash
mage test:unit        # Fast unit tests (~10s)
mage test:integration # Full integration tests (~60s)
mage test:coverage    # Unit tests with coverage report
mage lint             # Run golangci-lint with --fix
mage build:dev        # Build development binary
mage ci               # Full CI pipeline locally
```

### Release Workflow

#### Semantic Versioning

- **Major** (v2.0.0): Breaking changes, major architectural changes
- **Minor** (v1.1.0): New features, backward-compatible
- **Patch** (v1.0.1): Bug fixes, security patches

#### Release Process

1. **Create release branch**: `release/v1.2.0`
2. **Update version**: Version constants and documentation
3. **Build and test**: Full CI pipeline on all platforms
4. **Create release**: GitHub release with changelog
5. **Merge to main**: Fast-forward merge
6. **Tag release**: `git tag v1.2.0`

#### Platform Builds

```bash
mage build:release # Cross-platform binaries:
# - Linux: amd64, arm64
# - macOS: amd64, arm64 (Apple Silicon)
# - Windows: amd64
```

## Documentation Structure

### Documentation Hierarchy

All documentation files are located at the project root for easy discovery:

```
grove/
├── README.md            # Project overview and quick start
└── CONTRIBUTING.md      # Contribution guidelines and setup
```

### Specification Management

```
.spec-workflow/
├── steering/            # Project direction documents
│   ├── product.md       # Product vision and requirements
│   ├── tech.md          # Technical architecture
│   └── structure.md     # Project organization (this doc)
└── specs/               # Feature specifications
    └── feature-name/    # Individual feature specs
        ├── requirements.md
        ├── design.md
        └── tasks.md
```

### Code Documentation Standards

- **Package documentation**: Clear purpose and usage examples
- **Public functions**: Comprehensive docstrings with examples
- **Complex logic**: Inline comments for non-obvious code
- **Error messages**: User-friendly with suggested actions

## Team Conventions

### Communication Guidelines

#### Issue Management

- **Bug reports**: Use issue templates with reproduction steps
- **Feature requests**: Link to specifications when available
- **Questions**: Use discussions for general questions
- **Documentation**: Update docs with code changes

#### Decision Making

- **Architecture decisions**: Document in steering documents
- **Feature scope**: Define in specifications before coding
- **Bug priorities**: Triage based on user impact
- **Technical debt**: Track and prioritize in backlog

### Code Standards Enforcement

#### Pre-commit Hooks (Optional)

```bash
# Install pre-commit hooks
pre-commit install

# Hooks run automatically on commit:
# - go fmt
# - golangci-lint
# - go mod tidy
# - Basic test validation
```

#### Continuous Integration

```yaml
# CI Pipeline (future .github/workflows/ci.yml)
- Checkout code
- Setup Go environment
- Run mage ci:
      - Clean artifacts
      - Lint with golangci-lint
      - Unit tests with 90% coverage
      - Integration tests
      - Build binaries
- Report results
```

### Knowledge Management

#### Onboarding Process

1. **Read steering documents**: Understand project vision and architecture
2. **Setup development environment**: Go, golangci-lint, mage
3. **Run tests**: Verify local development setup
4. **Build binary**: Create and test grove binary
5. **First contribution**: Start with documentation or simple fixes

#### Code Ownership

- **Maintainers**: Full repository access, release authority
- **Contributors**: Submit PRs, participate in reviews
- **Users**: Report issues, suggest features

#### Learning Resources

- **Go documentation**: https://golang.org/doc/
- **Git worktree docs**: https://git-scm.com/docs/git-worktree
- **Cobra CLI framework**: https://cobra.dev/
- **Testing patterns**: Table-driven tests, mocking with testify

## Configuration Management

### Configuration Files

#### User Configuration (`~/.grove/config.toml`)

```toml
[default]
worktree_root = "~/worktrees"
auto_cleanup = true
max_worktrees = 50

[ui]
color_scheme = "auto"
progress_indicators = true

[git]
default_branch = "main"
auto_fetch = false
```

#### Project Configuration (`.grove.toml`)

```toml
[project]
name = "grove"
worktree_root = "./worktrees"

[workflows]
pre_create = ["git fetch origin"]
post_create = ["code $GROVE_WORKTREE_PATH"]
```

### Environment Variables

```bash
# Grove-specific environment variables
GROVE_CONFIG_DIR=~/.grove       # Config directory
GROVE_WORKTREE_ROOT=~/worktrees # Default worktree location
GROVE_DEBUG=true                # Enable debug output
GROVE_NO_COLOR=true             # Disable colored output

# Development environment
GO_VERSION=1.24.5            # Required Go version
GOLANGCI_LINT_VERSION=1.50.0 # Required linter version
```

### Security Considerations

- **Config file permissions**: User-only access (0600)
- **Sensitive data**: Use environment variables, not config files
- **Path validation**: Prevent directory traversal attacks
- **Command injection**: Sanitize all external command arguments

This structure provides a solid foundation for collaborative development while maintaining code quality and consistency across the Grove project. The workflow supports both individual contributors and team-based development with clear processes and automated quality checks.
