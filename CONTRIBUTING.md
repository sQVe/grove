# Contributing to Grove

Quick guide for contributing to Grove Git worktree management CLI.

## Getting started

### Prerequisites
- Go 1.21 or later
- Git 2.5 or later (for worktree support)

### Setup

1. **Clone and setup**:
   ```bash
   git clone https://github.com/sqve/grove.git
   cd grove
   go mod download
   ```

2. **Verify setup**:
   ```bash
   go test ./...
   go build -o grove ./cmd/grove
   ./grove --help
   ```

3. **Set up pre-commit hooks** (optional but recommended):
   ```bash
   pip install pre-commit
   pre-commit install
   ```

## Development workflow

### Before committing
Always run these commands after making changes:

```bash
go fmt ./...                                    # Format code
golangci-lint run                               # Run linter
go test -race -coverprofile=coverage.out ./... # Run tests
go build ./cmd/grove                            # Verify compilation
```

### Git workflow

**Commit format**: Use [Conventional Commits](https://www.conventionalcommits.org/)
```
type: description

Examples:
feat: add GitHub PR integration
fix: handle detached HEAD state
docs: update installation instructions
```

**Branch naming**:
```
feature/github-pr-integration
fix/worktree-cleanup-error
docs/contributing-guide
```

**Allowed commit types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`, `build`, `revert`

## Code standards

### Go conventions
- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` for formatting
- Write meaningful variable and function names
- Handle errors explicitly with context
- Add godoc comments for public functions

### Project structure
```go
// Package organization
package git

import (
    // Standard library first
    "fmt"
    "os"
    
    // Third-party packages
    "github.com/pkg/errors"
    
    // Local packages
    "github.com/sqve/grove/internal/config"
)
```

### Testing
- **Unit tests**: Test individual functions in isolation
- **Integration tests**: Test complete workflows with real git repositories
- **Table-driven tests**: Use for testing multiple scenarios
- **Co-locate tests**: `file.go` → `file_test.go`

## Pull requests

### Before submitting
- [ ] Tests pass locally
- [ ] Code follows style guidelines
- [ ] Documentation updated if needed
- [ ] Commit messages follow conventional format

### PR checklist
- **Clear description**: Explain what and why, not just how
- **Focused scope**: One feature or fix per PR
- **Link issues**: Reference related issues with `fixes #123`
- **Update docs**: Include relevant documentation changes

## Key implementation notes

- **Direct git execution**: Use `os/exec` to run git commands, parse output manually
- **Cross-platform support**: Handle Windows/macOS/Linux differences
- **Error handling**: Provide clear, actionable error messages
- **Configuration**: Use TOML format with validation
- **Testing**: Focus on git operations and CLI functionality

## Getting help

- **Documentation**: Check [ARCHITECTURE.md](ARCHITECTURE.md) for technical details
- **Issues**: Search existing issues before creating new ones
- **Questions**: Use GitHub Discussions for general questions
- **Review**: Tag maintainers if your PR needs attention

## Project goals

Grove aims to make Git worktrees accessible to all developers, not just power users. Keep this in mind when:
- Designing new features
- Writing error messages
- Creating documentation
- Reviewing code

Thank you for contributing to Grove!