# Grove

A fast, intuitive Git worktree management tool that makes Git worktrees as simple as switching branches.

## Quick Start

```bash
# Initialize workspace from repository
grove clone https://github.com/example/project

# Or create empty workspace
grove init new my-project
```

## Installation

```bash
git clone https://github.com/sqve/grove && cd grove
go build -o bin/grove ./cmd/grove
./bin/grove --help
```

## Development

```bash
# Run tests
mage test

# Format and lint
mage format && mage lint

# Build
mage build
```

## What is Grove?

Grove makes Git worktree management accessible to any developer. Work on multiple features simultaneously without stashing or branch switching.

**Instead of:**

-   Stashing/unstashing when switching branches
-   Complex `git worktree` commands
-   Multiple repository clones

**Grove provides:**

-   Simple workspace initialization
-   Branch-like worktree management
-   Cross-platform compatibility

MIT License
