# Grove Development Guidelines

This file contains Grove-specific development guidelines that complement the global CLAUDE.md.

## Project Overview
Grove is a Git worktree management CLI tool with a rich TUI interface built using TypeScript, Ink (React), and yargs.

## Progress Tracking
- **Always update ROADMAP.md** when making architectural decisions or completing implementation phases
- Mark completed phases with [x] and update progress notes
- Document any changes to the technical stack or approach
- Keep the implementation phases section current with actual progress

## Technical Stack Decisions

### CLI Framework
- **Use yargs** for CLI parsing and auto-completion support
- **Primary pattern**: Single `grove` command launches TUI, optional flags for scripting
- **Auto-completion**: Leverage yargs built-in completion for branches and worktree names

### TUI Framework
- **Use Ink + React** for the interactive interface
- **Component structure**: Organize into logical React components (WorktreeList, StatusPanel, etc.)
- **State management**: Use React hooks for worktree state, keyboard input, and git status
- **Real-time updates**: Implement live git status monitoring with hooks

### Git Operations
- **Use direct git execution** via Node.js child_process, NOT simple-git
- **Reason**: simple-git lacks proper worktree support, direct execution gives full control
- **Parse output manually** for worktree list, status, and branch information
- **Handle cross-platform differences** in git command output

### Development Tools
- **Use Biome** for linting and formatting (configured in package.json scripts)
- **Use Vitest** for testing with `pnpm test:ci` command
- **Use tsx** for development execution with watch mode

## Development Patterns

### File Organization
```
src/
├── index.ts              # Entry point with yargs CLI parsing
├── components/           # React components for TUI
├── commands/             # Command implementations for flags
├── lib/                  # Core functionality (git, config, utils)
├── hooks/                # React hooks for state management
└── types/                # TypeScript type definitions
```

### Component Guidelines
- **Single responsibility**: Each component should have one clear purpose
- **Props interface**: Define clear TypeScript interfaces for all props
- **Keyboard handling**: Use custom hooks for keyboard input management
- **State updates**: Minimize re-renders with proper memoization

### Git Integration
- **Command wrapper**: Create a centralized git operations module in `lib/git.ts`
- **Error handling**: Gracefully handle git command failures
- **Path handling**: Use absolute paths for worktree operations
- **Status parsing**: Parse git output into structured TypeScript types

## Testing Strategy

### Unit Tests
- Test git command parsing logic extensively
- Test fuzzy search functionality
- Test configuration loading and validation
- Use Vitest with `pnpm test:ci` for CI environments

### Integration Tests
- Test actual git worktree operations in temporary repositories
- Test CLI argument parsing with yargs
- Test component rendering with Ink testing utilities

### Manual Testing
- Test TUI navigation and keyboard shortcuts
- Test real-world worktree workflows
- Test error scenarios (invalid repos, missing git, etc.)

## Grove-Specific Guidelines

### Worktree Operations
- **Always validate** git repository state before operations
- **Handle bare repositories** properly for initialization
- **Support relative and absolute paths** for worktree locations
- **Graceful cleanup** when operations fail

### TUI Design
- **Vim-like navigation**: j/k for movement, enter for selection, q for quit
- **Visual feedback**: Clear indicators for active worktree, dirty state, etc.
- **Search integration**: Real-time fuzzy filtering of worktree list
- **Help accessibility**: Easy access to keybinding help

### CLI + TUI Hybrid
- **Default behavior**: `grove` with no args launches TUI
- **Flag behavior**: `grove --init`, `grove --clone`, etc. for scripting
- **Consistent output**: Same underlying operations for both modes
- **Exit codes**: Proper exit codes for scripting scenarios

## Performance Considerations
- **Lazy loading**: Load git information only when needed
- **Debounced search**: Prevent excessive filtering during typing
- **Efficient updates**: Minimize git command execution
- **Memory management**: Proper cleanup of child processes

## Error Handling
- **User-friendly messages**: Clear error descriptions with suggested actions
- **Graceful degradation**: Continue operation when non-critical features fail
- **Git validation**: Check for git availability and valid repository state
- **Path validation**: Ensure target paths are valid and accessible

## Configuration Management
- **Global config**: Store in `~/.config/grove/config.toml`
- **Repository config**: Store in `.grove/config.toml`
- **Sensible defaults**: Work well out of the box without configuration
- **Validation**: Validate configuration files on load

## Implementation Notes
- **TypeScript strict mode**: Use strict TypeScript configuration
- **ES modules**: Use import/export syntax throughout
- **No comments in code** unless specifically requested
- **Consistent formatting**: Use Biome for all formatting

## Development Workflow
1. **Plan first**: Update ROADMAP.md before implementing features
2. **Test-driven**: Write tests for core functionality
3. **Component-driven**: Build and test components in isolation
4. **Integration last**: Test full workflows after components work

## Distribution
- **Binary compilation**: Prepare for single-executable distribution
- **NPM packaging**: Ensure proper package.json configuration
- **Cross-platform**: Test on Windows, macOS, and Linux
- **Documentation**: Maintain README and help documentation