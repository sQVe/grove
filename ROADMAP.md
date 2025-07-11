# Grove - Git worktree management CLI

> CLI tool for Git worktree management with interactive TUI and subcommand scripting support.

## 🚀 Usage

```bash
grove                    # Interactive TUI (main usage)
grove init [path]        # Initialize bare repository
grove clone <branch>     # Clone worktree from branch
grove switch <worktree>  # Switch to worktree
grove list               # List all worktrees
grove pr <number>        # Create worktree from GitHub PR
grove linear <issue-id>  # Create worktree from Linear issue
```

## 📈 Implementation progress

### ✅ Phase 1: Foundation (COMPLETED)

- [x] Project structure with co-located types/tests
- [x] TypeScript configuration with Node.js types
- [x] CLI structure with yargs subcommands
- [x] Fuzzy search implementation and tests
- [x] Validation workflow (format, lint, typecheck, test)

### ✅ Phase 2: Git operations (COMPLETED)

- [x] Complete git operations wrapper (`lib/git.ts`)
- [x] Implement worktree creation, removal, switching
- [x] Implement repository initialization
- [x] Error handling and validation
- [x] All CLI commands functional with proper git integration

### 🔄 Phase 3: TUI development (IN PROGRESS)

- [ ] Basic Ink app with React components
- [ ] Worktree list with fuzzy search integration
- [ ] Vim-like navigation (j/k, enter, q)
- [ ] Multi-panel layout with status display
- [ ] Real-time git status updates

### 📦 Phase 4: Polish and core features

- [ ] Environment file cloning between worktrees
- [ ] Configuration management with cosmiconfig + zod
- [ ] Comprehensive testing suite
- [ ] Documentation and distribution setup

### 🔗 Phase 5: External integrations

- [ ] **GitHub PR support**
  - [ ] `grove pr <number>` command to create worktree from PR
  - [ ] `grove pr <number> --review` for read-only PR review worktrees
  - [ ] PR metadata display in TUI (title, author, status, CI checks)
  - [ ] Integration with GitHub CLI (`gh`) for authentication
- [ ] **Linear issue support**
  - [ ] `grove linear <issue-id>` command to create worktree from Linear issue
  - [ ] `grove linear <issue-id> --feature` for feature branch creation
  - [ ] Automatic branch naming from issue title (kebab-case)
  - [ ] Issue metadata display in TUI (title, status, assignee, priority)
  - [ ] Integration with Linear API for authentication
- [ ] Enhanced TUI for external metadata
- [ ] Configuration for API tokens and repository settings
- [ ] Error handling for API failures and authentication

---

## 🎨 TUI design

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
│ │ └─────────────────────────────────┘ └───────────────────────────────────┘ │
│                                                                          │
│ [c]reate [d]elete [r]ename [p]ull [P]ush [/]filter [?]help [q]uit       │
└──────────────────────────────────────────────────────────────────────────┘
```

### Interaction model (vim-like)

**Normal mode** (default):

- `j/k` or `↑/↓` navigate worktree list
- `h/l` or `←/→` switch between panels
- `Space/Enter` switch to selected worktree
- `c` create/clone new worktree
- `d` delete worktree, `D` force delete
- `r` rename worktree
- `p` pull, `P` push
- `/` open filter mode
- `?` toggle help overlay
- `q` quit application

**Filter mode** (triggered by `/`):

- Type to filter worktrees with fuzzy search
- `Enter` apply filter, `Esc` cancel

**Help mode** (triggered by `?`):

- Shows contextual keybindings
- `?` or `Esc` to close

### Component structure

- **HeaderBar**: Repository info and current worktree
- **WorktreeListPanel**: Worktree list with status indicators
- **DetailsPanel**: Git status, commits, and metadata
- **StatusLine**: Contextual action hints and mode indicator
- **FilterModal**: Fuzzy search overlay
- **HelpModal**: Contextual help overlay
- **App**: Main container with state and keyboard handling

---

## 🏗️ Technical infrastructure

### Technology stack

- **TypeScript/Node.js** with ES modules
- **yargs** for CLI parsing and auto-completion
- **Ink + React** for TUI framework
- **Biome** for linting/formatting, **Vitest** for testing
- **Direct git execution** via child_process (NOT simple-git)
- **fuse.js** for fuzzy search, **cosmiconfig + zod** for configuration

### Project structure

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
│   └── components/          # React components for TUI
├── package.json, tsconfig.json, README.md
└── ROADMAP.md              # This file
```

### Configuration

- **cosmiconfig** for flexible loading (JSON/JS/TOML)
- **Global**: `~/.config/grove/config.*`
- **Repository**: `.grove/config.*`
- **Validation**: zod schemas

### Core philosophy

- **Simplicity**: Make Git worktree operations intuitive
- **Speed**: Fast fuzzy search and keyboard shortcuts
- **Flexibility**: TUI for interactive use, subcommands for scripting
- **Vim-like**: Familiar navigation patterns

---

## 🔮 Future enhancements

### Tool integration and workflow

- **Editor launching**: Direct integration with Claude Code, VS Code, Cursor, and configurable editors
- **Shell access**: Open terminal/shell in selected worktree directory
- **Project awareness**: Auto-detect project type (package.json, Cargo.toml, etc.) and offer relevant actions
- **Development workflow**: Quick access to common commands (npm install, dev server, tests)

### Enhanced context display

- **Git context**: Stash count, untracked files count, ahead/behind with numbers
- **Activity tracking**: Last modified timestamps for worktrees
- **Status enrichment**: Lock status, remote tracking status, conflict indicators
- **Commit context**: Author, timestamp, and extended commit information

### Performance and UX

- **Background updates**: Async git status refresh without blocking UI
- **Lazy loading**: Only fetch detailed status for visible/selected worktrees
- **Caching**: Smart caching with file system watching for invalidation
- **Scalability**: Handle repositories with 10+ worktrees efficiently

### Configuration and customization

- **Tool preferences**: Configurable default editor, terminal, shell commands
- **Display options**: Customizable information density and column visibility
- **Workflow templates**: Project-specific worktree naming and setup patterns
- **Team settings**: Shared configuration for team workflows

---

## 💡 Value proposition

Grove transforms Git worktrees from a power-user feature into an essential productivity tool by providing instant visibility, zero-friction switching, intelligent tool integration, and context preservation for development workflows.

### Key benefits

- Real-time worktree status monitoring
- Fuzzy search filtering
- Environment file cloning between worktrees
- Shell auto-completion
- Cross-platform compatibility
- GitHub PR integration with worktree creation
- Linear issue integration with automatic branch naming
- External metadata display in TUI

---

_This roadmap reflects actual implementation progress and decisions made._
