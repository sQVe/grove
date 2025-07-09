# Grove - Git worktree management CLI

## Project overview
CLI tool for Git worktree management with interactive TUI and subcommand scripting support.

## Core philosophy
- **Simplicity**: Make Git worktree operations intuitive
- **Speed**: Fast fuzzy search and keyboard shortcuts  
- **Flexibility**: TUI for interactive use, subcommands for scripting
- **Vim-like**: Familiar navigation patterns

## Usage patterns

### **IMPORTANT** Primary usage
```bash
grove                    # Interactive TUI (main usage)
grove init [path]        # Initialize bare repository
grove clone <branch>     # Clone worktree from branch
grove switch <worktree>  # Switch to worktree
grove list               # List all worktrees
```

## Technical architecture

### Decided stack
- **TypeScript/Node.js** with ES modules
- **yargs** for CLI parsing and auto-completion
- **Ink + React** for TUI framework
- **Biome** for linting/formatting, **Vitest** for testing
- **Direct git execution** via child_process (NOT simple-git)
- **fuse.js** for fuzzy search, **cosmiconfig + zod** for configuration

### Project structure (co-located pattern)
```
grove/
├── src/
│   ├── index.ts              # CLI entry point with yargs
│   ├── commands/             # Command implementations (with co-located types)
│   │   ├── init.ts, clone.ts, switch.ts, list.ts
│   ├── lib/                  # Core functionality
│   │   ├── git.ts           # Git operations via child_process
│   │   ├── config.ts        # Configuration with cosmiconfig/zod
│   │   └── fuzzy.ts         # Fuzzy search with fuse.js
│   └── components/          # React components for TUI (future)
├── package.json, tsconfig.json, README.md
└── ROADMAP.md              # This file
```

## **IMPORTANT** Implementation progress

### ✅ Phase 1: Foundation (COMPLETED)
- [x] Project structure with co-located types/tests
- [x] TypeScript configuration with Node.js types
- [x] All dependencies installed and configured
- [x] CLI structure with yargs subcommands
- [x] Fuzzy search implementation and tests
- [x] Validation workflow (format, lint, typecheck, test)
- [x] **Key Decision**: yargs over commander for auto-completion
- [x] **Key Decision**: Ink over @clack/prompts for rich TUI
- [x] **Key Decision**: Co-located types/tests vs separate folders
- [x] **Key Decision**: Subcommands (`grove init`) vs flags (`grove --init`)

### Phase 2: Git operations (NEXT)
- [ ] Complete git operations wrapper (`lib/git.ts`)
- [ ] Implement worktree creation, removal, switching
- [ ] Implement repository initialization
- [ ] Error handling and validation

### Phase 3: TUI development
- [ ] Basic Ink app with React components
- [ ] Worktree list with fuzzy search integration
- [ ] Vim-like navigation (j/k, enter, q)
- [ ] Multi-panel layout with status display
- [ ] Real-time git status updates

### Phase 4: Polish
- [ ] Environment file cloning between worktrees
- [ ] Configuration management
- [ ] Comprehensive testing
- [ ] Documentation and distribution

## TUI design (decided: vim-like modal interface)
```
┌─ Grove ──────────────────────────────────────────────────────────────────┐
│ ┌─ Search & Filter ──────────────────────────────────────────────────────┐ │
│ │ Query: [main_______]  [x] Show dirty  [x] Show ahead  [ ] Show locked │ │
│ └────────────────────────────────────────────────────────────────────────┘ │
│                                                                          │
│ ┌─ Worktrees ─────────────────────┐ ┌─ Details ─────────────────────────┐ │
│ │                                 │ │ Branch: main                      │ │
│ │ > main              *active     │ │ Path: /repo/main                  │ │
│ │   feature-auth      2 ahead     │ │ Status: ✓ Clean                   │ │
│ │   bugfix-login      dirty       │ │ Files: 12 total                   │ │
│ │   feature-ui        1 behind    │ │ Commits: 3 ahead, 1 behind       │ │
│ │   hotfix-security   locked      │ │                                   │ │
│ │                                 │ │ Recent commits:                   │ │
│ │                                 │ │ abc123f Fix authentication bug    │ │
│ │                                 │ │ def456a Update user interface     │ │
│ │                                 │ │ ghi789b Add comprehensive tests   │ │
│ │                                 │ │                                   │ │
│ │                                 │ │                                   │ │
│ └─────────────────────────────────┘ └───────────────────────────────────┘ │
│                                                                          │
│ -- NORMAL -- Type '?' for help                          4 worktrees       │
└──────────────────────────────────────────────────────────────────────────┘
```

### Modal interface (vim-like)
- **Normal mode**: `j/k` navigate, `enter` switch, `c` clone, `r` remove, `q` quit, `/` search
- **Insert mode**: Active when typing in search (shows `-- INSERT --`)
- **Command mode**: `:` for commands like `:help`, `:quit`
- **Help toggle**: `?` shows/hides help panel (optional, not always visible)

### Component structure
- **FilterBar**: Top search input + status filter checkboxes
- **WorktreeListPanel**: Left panel with fuzzy-filtered and status-filtered list
- **DetailsPanel**: Right panel showing selected worktree information  
- **App**: Main container with modal state management and keyboard handling
- **StatusLine**: Bottom status with mode indicator and worktree count
- **HelpPanel**: Optional overlay triggered by `?`

## Configuration
- **cosmiconfig** for flexible loading (JSON/JS/TOML)
- **Global**: `~/.config/grove/config.*`
- **Repository**: `.grove/config.*`
- **Validation**: zod schemas

## Key features (planned)
- Real-time worktree status monitoring
- Fuzzy search filtering
- Environment file cloning between worktrees
- Shell auto-completion
- Cross-platform compatibility

---
This roadmap reflects actual implementation progress and decisions made.
