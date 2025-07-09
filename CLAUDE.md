# Grove development guidelines

Grove-specific development guidelines for the Git worktree management CLI tool.

## Project overview
CLI tool for Git worktree management with TUI interface. Built with TypeScript, Ink (React), and yargs.

## **IMPORTANT** Development requirements
- **Always run validation**: `pnpm format && pnpm lint && pnpm tsc --noEmit && pnpm test:ci` after ALL code changes
- **Update ROADMAP.md** when completing implementation phases or making architectural decisions

## **IMPORTANT** Git workflow
- **Follow conventional commits**: Use format `type: description` (feat, fix, chore, docs, refactor, test)
- **Use imperative mood**: "add feature" not "added feature"
- **Limit first line to 72 characters**
- **Multiple commits**: Split logical changes into separate commits
- **Concise descriptions**: Keep commit messages brief but clear

## Technical stack (decided)

### Core dependencies
- **yargs**: CLI parsing with auto-completion (chosen over commander)
- **Ink + React**: TUI framework (chosen over @clack/prompts)
- **Biome**: Linting/formatting (chosen over ESLint/Prettier)
- **Vitest**: Testing (chosen over Jest)
- **Direct git execution**: child_process (NOT simple-git - lacks worktree support)

### Usage pattern
- **Primary**: `grove` launches interactive TUI
- **Scripting**: `grove init`, `grove clone <branch>`, `grove switch <name>`, `grove list`

## File organization (co-located pattern)
```
src/
├── index.ts              # CLI entry point
├── commands/             # Command implementations with co-located types
├── lib/                  # Core functionality (git, config, fuzzy search)
└── components/           # React components for TUI (future)
```
- **IMPORTANT**: Co-located types and tests (NOT separate folders)

## Code style

### Function parameters
- **IMPORTANT**: Destructure in function signature, NOT function body
	- ✅ `function example({ name, age = 18 }: Options) { ... }`
	- ❌ `function example(options: Options) { const { name, age = 18 } = options; ... }`
- **IMPORTANT**: Use `const` for function parameters.
- **IMPORTANT**: Comments should always end with a period.

### General rules
- TypeScript strict mode with Node.js types
- ES modules throughout
- No comments unless specifically requested
- Biome formatting
- **IMPORTANT**: All markdown headers use sentence case

## Git operations
- Use `child_process.exec()` for git commands
- Parse output manually for worktree operations
- Handle cross-platform differences
- Always validate git repository state

## TUI design (decided: vim-like modal interface)
- **Layout**: FilterBar + two-panel flexbox + StatusLine
- **Modal interface**: Normal/Insert/Command modes like vim
- **Components**: App → FilterBar + WorktreeListPanel + DetailsPanel + StatusLine + HelpPanel
- **Keybindings**: `j/k` navigate, `enter` switch, `c` clone, `r` remove, `/` search, `?` help, `q` quit
- **Filtering**: Fuzzy search + status filters (dirty, ahead, behind, locked)
- **Clean UI**: No always-visible hotkeys, optional help toggle

## Configuration
- cosmiconfig for flexible config loading (JSON/JS/TOML support)
- Global: `~/.config/grove/`
- Repository: `.grove/`
- Sensible defaults

## Testing
- Unit tests for core functionality
- Integration tests for git operations
- Use `pnpm test:ci` for CI environments
- Test TUI components with Ink testing utilities

## Performance
- Lazy loading of git information
- Debounced search filtering
- Efficient React re-renders
- Minimal git command execution
