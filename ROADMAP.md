# Grove roadmap

> **Git worktree management CLI** - Fast, intuitive worktree operations with optional TUI interface.

## Overview

Grove transforms Git worktrees from a power-user feature into an essential productivity tool. Currently focused on delivering exceptional CLI experience with plans for interactive TUI.

### Current capabilities

```bash
grove                    # List all worktrees (default)
grove init [path]        # Initialize bare repository
grove create <branch>     # Create worktree from branch
grove switch <worktree>  # Switch to worktree
grove list               # List with enhanced formatting
```

### Planned capabilities

```bash
grove tui                # Interactive TUI interface
grove pr <number>        # Create worktree from GitHub PR
grove linear <issue-id>  # Create worktree from Linear issue
```

## Implementation status

### ✅ Foundation (complete)

**Core infrastructure established**

- [x] TypeScript project with ES modules and Node.js types
- [x] CLI structure with yargs subcommands and auto-completion
- [x] Validation workflow (format, lint, typecheck, test)
- [x] Co-located types and tests for maintainability
- [x] Fuzzy search implementation with comprehensive tests

### ✅ Git operations (complete)

**Full worktree management functionality**

- [x] Complete git operations wrapper in `lib/git.ts`
- [x] Worktree creation, removal, and switching
- [x] Repository initialization with bare repo support
- [x] Robust error handling and git validation
- [x] All CLI commands functional with proper integration

### 🚧 CLI polish (in progress)

**Enhanced command-line experience**

- [x] Enhanced list output with colors and status indicators
- [x] Improved table formatting with proper spacing
- [ ] Environment file cloning between worktrees
- [ ] Configuration management with cosmiconfig + zod
- [ ] Shell auto-completion for commands and worktree names
- [ ] Comprehensive documentation and distribution setup

### 📋 Planned features

**External integrations**

- [ ] GitHub PR support (`grove pr <number>`)
- [ ] Linear issue support (`grove linear <issue-id>`)
- [ ] API authentication and metadata display
- [ ] Enhanced CLI output for external data

**Interactive TUI** _(future)_

- [ ] React/Ink-based interface with vim-like navigation
- [ ] Multi-panel layout with real-time git status
- [ ] Fuzzy search filtering and contextual actions
- [ ] Visual git state display with rich metadata

## Technical architecture

### Technology stack

**Core technologies**

- TypeScript/Node.js with ES modules
- yargs for CLI parsing and auto-completion
- Biome for linting/formatting, Vitest for testing
- Direct git execution via child_process
- fuse.js for fuzzy search, cosmiconfig + zod for configuration

**Future TUI stack**

- Ink + React for terminal interface
- Vim-like navigation patterns
- Multi-panel layout with real-time updates

### Project structure

```
grove/
├── src/
│   ├── index.ts              # CLI entry point
│   ├── commands/             # Command implementations
│   │   ├── init.ts, clone.ts, switch.ts, list.ts
│   └── lib/                  # Core functionality
│       ├── git.ts           # Git operations
│       ├── config.ts        # Configuration
│       └── fuzzy.ts         # Fuzzy search
├── package.json, tsconfig.json
└── ROADMAP.md
```

### Configuration approach

**Flexible configuration loading**

- cosmiconfig for JSON/JS/TOML support
- Global: `~/.config/grove/config.*`
- Repository: `.grove/config.*`
- Validation with zod schemas

### Design principles

**Core philosophy**

- **Simplicity**: Make Git worktree operations intuitive
- **Speed**: Fast fuzzy search and keyboard shortcuts
- **CLI-first**: Excellent command-line experience with optional TUI
- **Vim-like**: Familiar navigation patterns for power users

## Future vision

### Interactive TUI interface

**When CLI foundation is solid**

- Multi-panel layout with vim-like navigation
- Real-time fuzzy search with status filters
- Visual git status with rich metadata display
- Contextual keyboard shortcuts for common actions

**Planned TUI design**

```
┌─ Grove ─ /repo/project ─ main* ──────────── 4 worktrees ──┐
│ ┌─ Worktrees ─────────────┐ ┌─ Details ─────────────────┐ │
│ │ > main        *active   │ │ Branch: main              │ │
│ │   feature     2 ahead   │ │ Status: ✓ Clean           │ │
│ │   bugfix      dirty     │ │ Files: 12 total           │ │
│ │   pr-123     PR #123    │ │ Commits: 3↑ 1↓            │ │
│ └─────────────────────────┘ └───────────────────────────┘ │
│ [c]reate [d]elete [/]filter [?]help [q]uit               │
└──────────────────────────────────────────────────────────┘
```

### Advanced integrations

**Developer workflow**

- Editor launching (VS Code, Cursor, configurable)
- Shell access in worktree directories
- Project-aware actions (npm install, tests)
- Environment file cloning between worktrees

**External services**

- GitHub PR worktree creation
- Linear issue integration with automatic branch naming
- API authentication and metadata display
- Team configuration sharing

### Performance enhancements

**Scalability and speed**

- Background git status updates
- Lazy loading for large repositories
- Smart caching with file system watching
- Handle 10+ worktrees efficiently

**User experience**

- Customizable information density
- Workflow templates for teams
- Enhanced git context (stashes, untracked files)
- Activity tracking and timestamps

---

## Value proposition

**Transform Git worktrees** from a power-user feature into an essential productivity tool.

### Key benefits

- **Instant visibility**: Real-time worktree status monitoring
- **Zero friction**: Fast switching with fuzzy search
- **Smart integration**: GitHub/Linear workflow support
- **Cross-platform**: Works everywhere Git works
- **Extensible**: Configuration for team workflows

---

_This roadmap reflects current implementation status and future vision._
