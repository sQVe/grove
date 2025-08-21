# Contributing to Grove

Thanks for contributing! Grove makes Git worktrees simple, and we want contributing to Grove to be simple too.

## Project Context

Before diving in, check out our steering documents to understand the project vision and standards:

- **[Product Vision](.spec-workflow/steering/product.md)** - Mission, target users, and value propositions
- **[Technical Standards](.spec-workflow/steering/tech.md)** - Architecture principles, tech stack, and patterns
- **[Project Structure](.spec-workflow/steering/structure.md)** - File organization and naming conventions

These documents guide all development decisions and ensure consistency across the project.

## Quick Setup

| Step        | Command                                               |
| ----------- | ----------------------------------------------------- |
| **Clone**   | `git clone https://github.com/sQVe/grove && cd grove` |
| **Install** | `go mod download`                                     |
| **Verify**  | `mage test:unit && mage lint && mage build:all`       |

**Prerequisites:** Go 1.21+, Git 2.5+, golangci-lint 1.50+, Mage build system

## Testing Strategy

**Unit tests** (`*_test.go`) - Test internal functions directly. Use real Git.

- Single function behavior and error conditions
- Tests should be short and focused, and most importantly, fast

**Testscript tests** (`testdata/script/*.txt`) - Test CLI commands and workflows.

- User-facing behavior and error messages
- Complex setups or multi-step flows
- Exit codes and command integration

**Decision:**

- Can a user type it? → Testscript
- Are we testing a flow? → Testscript
- Otherwise → Unit test

**Testscript organization:**

- `*_validation.txt` - Fast tests: arguments, help, preconditions
- `*_integration.txt` - Slower tests: actual Git operations, shared fixtures
