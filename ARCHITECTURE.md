# Architecture

Technical design and implementation details for Grove.

**Status**: This document describes the core Go architecture decisions. The TypeScript implementation is being rewritten.

## Project structure

```
grove/
├── cmd/grove/                # Main application entry point
├── internal/                 # Private application code
│   ├── git/                 # Git operations and worktree management
│   ├── config/              # Configuration loading and validation
│   └── util/                # Shared utilities
└── pkg/                      # Public packages (if needed)
```

## Core principles

### Direct git execution
- Execute git commands directly via `os/exec` for maximum compatibility
- Parse git output manually for worktree operations
- Avoid git libraries that lack comprehensive worktree support
- Handle git command failures with descriptive error messages

### Cross-platform compatibility
- Handle path separators and git installation differences
- Support Windows, macOS, and Linux environments
- Use Go's standard library for portable operations

### Robust error handling
- Provide clear error messages with suggested solutions
- Validate git repository state before operations
- Handle edge cases (no commits, detached HEAD, etc.)

## Git operations

### Core commands
- Parse `git worktree list --porcelain` for structured data
- Create worktrees with proper branch validation
- Remove worktrees with cleanup verification
- Parse `git status --porcelain` for file changes

### Repository validation
- Check if directory is a git repository
- Verify git is available in PATH
- Validate repository has commits before worktree operations

## Configuration system

### Configuration hierarchy
1. Repository-specific: `.grove/config.toml`
2. Global user: `~/.config/grove/config.toml`
3. Built-in defaults

### Basic configuration format
```toml
[grove]
default_branch = "main"
```

Future sections will be added as features are implemented.
