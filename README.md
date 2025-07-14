# Grove

> Fast, intuitive Git worktree management CLI with optional TUI interface.

Grove transforms Git worktrees from a power-user feature into an essential productivity tool. Manage multiple working directories effortlessly with fuzzy search, smart cleanup, and seamless integration with GitHub and Linear.

**Status**: Currently being rewritten in Go for better performance and easier distribution.

## Installation

```bash
# Install from source (Go required)
go install github.com/sqve/grove@latest

# Or download binary from releases
# https://github.com/sqve/grove/releases
```

## Quick start

```bash
# List all worktrees (default command)
grove

# Initialize bare repository for worktree management
grove init

# Create worktree from existing branch
grove create feature-branch

# Switch to a worktree
grove switch main

# List with enhanced formatting
grove list
```

## Features

- **Instant worktree operations**: Create, switch, and manage worktrees with simple commands
- **Smart status display**: See git status, ahead/behind counts, and cleanliness at a glance
- **Cross-platform**: Works on macOS, Linux, and Windows
- **Enhanced listing**: Color-coded status indicators and structured output

**Coming soon**: GitHub PR integration, Linear issue support, smart cleanup, fuzzy search, and interactive TUI.

See [FEATURES.md](FEATURES.md) for complete feature details and roadmap.

## Documentation

- [Features and capabilities](FEATURES.md)
- [Technical architecture](ARCHITECTURE.md) 
- [Contributing guidelines](CONTRIBUTING.md)
- [Development roadmap](ROADMAP.md)

## Why Grove?

Git worktrees let you work on multiple branches simultaneously, but the commands are verbose and hard to remember. Grove makes worktree management as simple as switching branches.

Perfect for working on multiple features, code reviews without stashing, and testing different approaches in parallel.

## License

MIT License - see [LICENSE](LICENSE) file for details.