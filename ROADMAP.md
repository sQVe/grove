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
grove pr <number>        # Create worktree from GitHub PR
grove linear <issue-id>  # Create worktree from Linear issue
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

### ✅ Phase 2: Git operations (COMPLETED)

- [x] Complete git operations wrapper (`lib/git.ts`)
- [x] Implement worktree creation, removal, switching
- [x] Implement repository initialization
- [x] Error handling and validation
- [x] All CLI commands functional with proper git integration
- [x] Comprehensive error handling and user-friendly messages
- [x] Tests for git parsing and error handling

### Phase 3: TUI development (NEXT)

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

### Phase 5: External integrations

- [ ] GitHub PR support
  - [ ] `grove pr <number>` command to create worktree from PR
  - [ ] `grove pr <number> --review` for read-only PR review worktrees
  - [ ] PR metadata display in TUI (title, author, status, CI checks)
  - [ ] Integration with GitHub CLI (`gh`) for authentication
- [ ] Linear issue support
  - [ ] `grove linear <issue-id>` command to create worktree from Linear issue
  - [ ] `grove linear <issue-id> --feature` for feature branch creation
  - [ ] Automatic branch naming from issue title (kebab-case)
  - [ ] Issue metadata display in TUI (title, status, assignee, priority)
  - [ ] Integration with Linear API for authentication
- [ ] Enhanced TUI for external metadata
- [ ] Configuration for API tokens and repository settings
- [ ] Error handling for API failures and authentication

## TUI design

```
┌─ Grove ─ /repo/my-project ─ main* ──────────────────────── 4 worktrees ──┐
│                                                                          │
│ ┌─ Worktrees ─────────────────────┐ ┌─ Details ─────────────────────────┐ │
│ │                                 │ │ Branch: main                      │ │
│ │ > main              *active     │ │ Path: /repo/main                  │ │
│ │   feature-auth      2 ahead     │ │ Status: ✓ Clean                   │ │
│ │   bugfix-login      dirty       │ │ Files: 12 total                   │ │
│ │   feature-ui        1 behind    │ │ Commits: 3 ahead, 1 behind       │ │
│ │   hotfix-security   locked      │ │                                   │ │
│ │   pr-123           PR #123      │ │ Recent commits:                   │ │
│ │   linear-ABC       ABC-123      │ │ abc123f Fix authentication bug    │ │
│ │                                 │ │ def456a Update user interface     │ │
│ │                                 │ │ ghi789b Add comprehensive tests   │ │
│ │                                 │ │                                   │ │
│ │                                 │ │ PR #123: Add user authentication  │ │
│ │                                 │ │ Status: ✓ Checks passed           │ │
│ │                                 │ │ Author: john.doe                  │ │
│ └─────────────────────────────────┘ └───────────────────────────────────┘ │
│                                                                          │
│ [c]reate [d]elete [r]ename [p]ull [P]ush [/]filter [?]help [q]uit       │
└──────────────────────────────────────────────────────────────────────────┘
```

### Modal interface (lazygit-inspired)

- **Normal mode**: Default mode for navigation and actions
  - `j/k` or `↑/↓` navigate worktree list
  - `h/l` or `←/→` switch between panels
  - `Space` or `Enter` switch to selected worktree
  - `c` create/clone new worktree
  - `d` delete worktree
  - `D` force delete worktree (shift+d)
  - `r` rename worktree
  - `p` pull in current worktree
  - `P` push from current worktree
  - `/` open filter mode
  - `?` toggle help overlay
  - `q` quit application
  - `Esc` cancel modal actions
- **Filter mode**: Active when typing search query (triggered by `/`)
  - Type to filter worktrees
  - `Enter` apply filter and return to normal mode
  - `Esc` cancel filter and return to normal mode
- **Help mode**: Contextual help overlay (triggered by `?`)
  - Shows available keybindings for current context
  - `?` or `Esc` to close help

### Component structure

- **HeaderBar**: Top bar with repository info and current worktree
- **WorktreeListPanel**: Left panel with worktree list and navigation
  - Clean, minimal list display
  - Visual indicators for status (active, dirty, ahead/behind)
  - PR/Linear metadata integration
- **DetailsPanel**: Right panel with contextual information
  - Git status and branch information
  - Recent commits for selected worktree
  - External metadata (PR/Linear details when available)
  - Contextual actions and hints
- **StatusLine**: Bottom bar with contextual action hints
  - Shows available keybindings based on current selection
  - Mode indicator (Normal, Filter, Help)
  - Statistics (total worktrees, active worktree)
- **FilterModal**: Modal overlay for search/filtering (triggered by `/`)
  - Fuzzy search input
  - Status filtering options
- **HelpModal**: Contextual help overlay (triggered by `?`)
  - Shows keybindings relevant to current context
- **App**: Main container with modal state management and keyboard handling

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
- GitHub PR integration with worktree creation
- Linear issue integration with automatic branch naming
- External metadata display in TUI

---

This roadmap reflects actual implementation progress and decisions made.
