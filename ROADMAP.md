# Grove - Git Worktree Management CLI

## Project Overview

Grove is a CLI tool that simplifies Git worktree management through both command-line interface and an interactive TUI. It provides easy initialization of bare repositories, cloning of worktrees, switching between worktrees, and environment cloning between worktrees.

## Core Philosophy

- **Simplicity**: Make complex Git worktree operations simple and intuitive
- **Speed**: Fast operations with fuzzy search and keyboard shortcuts
- **Flexibility**: Both CLI commands and interactive TUI for different workflows
- **Vim-like**: Familiar navigation patterns for developers

## Features

### Primary Usage Pattern
**Grove is designed as a single interactive command with optional CLI flags for scripting/CI:**

```bash
grove                    # Launch interactive TUI (main usage)
grove --init             # Initialize bare repository (scripting)
grove --clone <branch>   # Clone worktree (scripting)
grove --switch <name>    # Switch worktree (scripting)
grove --list             # List worktrees (scripting)
```

### Interactive TUI (Primary Feature)
- **Single command**: `grove` launches full interactive interface
- **Fuzzy search**: Real-time filtering of worktrees
- **Vim-like navigation**: j/k movement, enter to select, familiar keybindings
- **Rich interface**: Multi-panel layout with worktree list, status, and help
- **Real-time updates**: Live git status, branch changes, file modifications
- **Quick actions**: Single-key shortcuts for common operations
- **Visual indicators**: Clear status for each worktree (active, clean, dirty, etc.)

### Scripting/CI Support
- **Flag-based**: Optional command-line flags for automation
- **Auto-completion**: Built-in shell completion for branches and worktrees
- **Exit codes**: Proper exit codes for scripting
- **Non-interactive**: Skip TUI when flags are provided

## Technical Architecture

### Language & Runtime
- **TypeScript/Node.js** - Familiar development environment
- **ES Modules** - Modern JavaScript module system
- **Node.js 18+** - Modern runtime features

### CLI Framework
- **yargs** - Command parsing with built-in auto-completion support
- **chalk** - Terminal colors and styling
- **figures** - Cross-platform Unicode symbols

### TUI Framework
- **Ink** - React-based terminal UI framework
- **React** - Component-based architecture for complex interfaces
- **Built-in fuzzy search** - JavaScript-based fuzzy filtering
- **Real-time state management** - React hooks for live updates

### Git Operations
- **Direct git execution** - Using Node.js child_process for full worktree support

### Configuration
- **TOML/JSON** - Configuration file format
- **XDG Base Directory** - Standard config location
- **Per-repository** - Repository-specific settings

### Auto-Completion
- **Yargs built-in completion** - Native auto-completion support
- **Context-aware suggestions** - Branch names, worktree names, file paths
- **Shell support** - Bash, Zsh, Fish completion scripts
- **Dynamic completion** - Real-time suggestions based on git state

## Project Structure

```
grove/
├── src/
│   ├── index.ts              # Entry point with yargs CLI parsing
│   ├── components/           # React components for TUI
│   │   ├── App.tsx          # Main TUI application
│   │   ├── WorktreeList.tsx # Worktree list with fuzzy search
│   │   ├── StatusPanel.tsx  # Status and preview panel
│   │   ├── HelpPanel.tsx    # Help and keybindings
│   │   └── Header.tsx       # Application header
│   ├── commands/             # Command implementations
│   │   ├── init.ts          # Initialize bare repository
│   │   ├── clone.ts         # Clone worktrees
│   │   ├── switch.ts        # Switch between worktrees
│   │   ├── env.ts           # Environment cloning
│   │   ├── list.ts          # List worktrees
│   │   └── remove.ts        # Remove worktrees
│   ├── lib/                 # Library functions
│   │   ├── git.ts           # Git operations using child_process
│   │   ├── config.ts        # Configuration management
│   │   ├── fuzzy.ts         # JavaScript fuzzy search
│   │   ├── paths.ts         # Path utilities
│   │   └── utils.ts         # General utilities
│   ├── hooks/               # React hooks
│   │   ├── useWorktrees.ts  # Worktree state management
│   │   ├── useKeyboard.ts   # Keyboard input handling
│   │   └── useGitStatus.ts  # Live git status updates
│   └── types/               # TypeScript definitions
│       ├── git.ts           # Git-related types
│       ├── config.ts        # Configuration types
│       └── tui.ts           # TUI-related types
├── tests/                   # Vitest test files
├── docs/                    # Documentation
├── package.json
├── tsconfig.json
├── README.md
└── PLAN.md                  # This file
```

## Implementation Phases

### Phase 1: Project Foundation
- [x] Create project structure
- [x] Set up package.json with basic dependencies
- [x] Configure TypeScript (tsconfig.json) using Total TypeScript best practices
- [x] Add development dependencies (TypeScript, tsx, @biomejs/biome, vitest)
- [x] Add runtime dependencies (yargs, chalk, figures, ink, react)
- [x] Configure scripts for Biome (linting/formatting) and Vitest (testing)
- [x] **Key Decision**: Use yargs over commander for built-in auto-completion
- [x] **Key Decision**: Use Ink over @clack/prompts for rich TUI capabilities
- [x] **Key Decision**: Use direct git execution over simple-git for full worktree support
- [x] **Key Decision**: Use Biome over ESLint/Prettier for faster tooling
- [x] **Key Decision**: Use Vitest over Jest for better ES modules support
- [ ] Set up basic CLI structure with yargs
- [ ] Create git operations wrapper using child_process
- [ ] Basic configuration management

### Phase 2: Core Commands
- [ ] Implement `grove init`
- [ ] Implement `grove clone`
- [ ] Implement `grove list`
- [ ] Implement `grove switch`
- [ ] Basic worktree management functionality

### Phase 3: TUI Development (Ink-based)
- [ ] Create basic Ink app structure with React components
- [ ] Implement WorktreeList component with fuzzy search
- [ ] Add keyboard input handling with useKeyboard hook
- [ ] Implement Vim-like navigation (j/k, enter, q, etc.)
- [ ] Create StatusPanel component for worktree details
- [ ] Add real-time git status updates with useGitStatus hook
- [ ] Implement multi-panel layout with proper focus management
- [ ] Add visual indicators and theming

### Phase 4: Advanced Features
- [ ] Environment cloning between worktrees
- [ ] Preview pane with branch info
- [ ] Advanced TUI features (help, configuration)
- [ ] Performance optimizations
- [ ] Error handling and user feedback

### Phase 5: Polish & Distribution
- [ ] Comprehensive testing
- [ ] Documentation
- [ ] Binary distribution
- [ ] npm package publishing
- [ ] Installation guides

## TUI Design (Ink-based)

### Main Interface Layout
```
┌─ Grove - Git Worktree Manager ──────────────────────────────────────┐
│                                                                      │
│  Search: [main_________]                     │  Branch: main         │
│                                              │  Status: ✓ Clean      │
│  > main              * active    main        │  Files:  12 modified  │
│    feature/auth               feature/auth   │  Commits: 3 ahead     │
│    bugfix/issue-123          fix: login      │                       │
│    feature/ui                new: redesign   │  Recent commits:      │
│                                              │  abc123f Fix auth     │
│  ┌─ Help ─────────────────────────────────┐  │  def456a Update UI    │
│  │ [j/k] navigate  [enter] switch         │  │  ghi789b Add tests    │
│  │ [c] clone  [r] remove  [e] env         │  │                       │
│  │ [/] search  [?] help  [q] quit         │  │                       │
│  └─────────────────────────────────────────┘  │                       │
│                                                                      │
└─ 4 worktrees ─────────────────────────────────────────────────────┘
```

### Component Structure
```typescript
<App>
  <Header title="Grove - Git Worktree Manager" />
  <Box flexDirection="row">
    <Box flexGrow={1}>
      <SearchInput />
      <WorktreeList />
    </Box>
    <Box width={30}>
      <StatusPanel />
    </Box>
  </Box>
  <HelpPanel />
</App>
```

### Keybindings
- `j/k` - Navigate up/down
- `enter` - Switch to selected worktree
- `c` - Clone new worktree
- `r` - Remove selected worktree
- `e` - Environment operations
- `l` - List all worktrees
- `?` - Show help
- `q` - Quit
- `/` - Search/filter

## Configuration

### Global Configuration (`~/.config/grove/config.toml`)
```toml
[general]
default_branch = "main"
auto_fetch = true
confirm_destructive = true

[tui]
theme = "default"
vim_bindings = true
preview_enabled = true

[env]
files = [".env", ".env.local", "package.json"]
ignore_patterns = ["node_modules", ".git", "dist"]
```

### Repository Configuration (`.grove/config.toml`)
```toml
[repository]
bare_path = "/path/to/bare/repo"
worktree_prefix = "grove-"

[env]
sync_files = [".env.local", "config.json"]
```

## Environment Cloning

### Supported Files
- Environment files (`.env`, `.env.local`, etc.)
- Configuration files (`config.json`, `settings.toml`)
- Package manager files (`package.json`, `yarn.lock`)
- Custom patterns defined in configuration

### Operations
- **Copy**: Copy files from one worktree to another
- **Sync**: Bidirectional synchronization
- **Diff**: Show differences between worktree environments
- **Merge**: Intelligently merge environment changes

## Error Handling

### Graceful Degradation
- **Built-in fuzzy search**: No external dependencies required
- **Fallback modes**: Continue operation if non-critical features fail
- **Clear error messages**: Helpful suggestions for common issues
- **Git availability**: Graceful handling when git is not installed

### User Feedback
- Progress indicators for long operations
- Clear success/failure messages
- Helpful suggestions for common errors

## Performance Considerations

### Optimization Strategies
- **Lazy loading**: Git information loaded on-demand
- **Caching**: Frequently accessed data cached in memory
- **Efficient fuzzy search**: JavaScript-based fuzzy matching
- **Minimal git operations**: Direct child_process execution
- **React optimizations**: Proper memoization and state management
- **Debounced updates**: Prevent excessive re-renders during typing

### Benchmarks
- **Startup time**: < 100ms for TUI launch
- **Fuzzy search**: < 50ms response time
- **Git operations**: Optimized child_process calls
- **Memory usage**: Efficient React component lifecycle

## Testing Strategy

### Unit Tests
- Individual command functionality
- Git operations wrapper
- Configuration management
- Utility functions

### Integration Tests
- **Full CLI command execution**: Test yargs argument parsing
- **TUI interaction simulation**: Test Ink component rendering
- **Git repository operations**: Test child_process git commands
- **Configuration loading**: Test config file parsing
- **Keyboard input handling**: Test key press simulation

### Manual Testing
- Real-world workflow scenarios
- Performance testing
- Cross-platform compatibility

## Distribution

### Installation Methods
- npm global install: `npm install -g grove-cli`
- Binary releases for major platforms
- Package managers (brew, apt, etc.)

### Binary Distribution
- Single executable with Node.js bundled
- Platform-specific optimizations
- Minimal dependencies

## Future Enhancements

### Potential Features
- Integration with GitHub/GitLab
- Custom worktree templates
- Team collaboration features
- Plugin system for extensions
- IDE integrations

### Community
- Contributing guidelines
- Issue templates
- Feature request process
- Documentation improvements

---

This plan serves as a living document that will be updated as the project evolves.
