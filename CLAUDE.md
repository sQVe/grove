# Grove development guidelines

Grove-specific development guidelines for the Git worktree management CLI tool.

## 🚨 **CRITICAL REQUIREMENTS**

### Development workflow

- **Always run validation**: `pnpm format && pnpm format:md && pnpm lint && pnpm tsc --noEmit && pnpm test:ci` after ALL code changes.
- **Pre-commit hooks**: Automatically run format, lint, typecheck, and tests on staged files before each commit.
- **Commit message validation**: All commits must follow [Conventional Commits](https://www.conventionalcommits.org/) format.
- **Double-check before committing**: Review all changes, verify tests pass, ensure code follows guidelines.
- **Update ROADMAP.md** when completing implementation phases or making architectural decisions.

### Git workflow

- **Follow conventional commits**: Use format `type: description` (feat, fix, chore, docs, refactor, test).
- **Use imperative mood**: "add feature" not "added feature".
- **Limit first line to 72 characters**.
- **Multiple commits**: Split logical changes into separate commits.
- **Concise descriptions**: Keep commit messages brief but clear.

### Code style requirements

- **Function destructuring**: In function signature, NOT function body
  - ✅ `function example({ name, age = 18 }: Options) { ... }`
  - ❌ `function example(options: Options) { const { name, age = 18 } = options; ... }`
- **Comments**: Must end with a period.
- **Headers**: All markdown headers use sentence case.
- **Lists**: All markdown list items must end with a period.
- **Organization**: Co-located types and tests (NOT separate folders).

---

## 📋 **PROJECT OVERVIEW**

CLI tool for Git worktree management with TUI interface. Built with TypeScript, Ink (React), and yargs.

### Usage patterns

- **Primary**: `grove` launches interactive TUI.
- **Scripting**: `grove init`, `grove clone <branch>`, `grove switch <name>`, `grove list`.

### Project structure

```
src/
├── index.ts              # CLI entry point
├── commands/             # Command implementations with co-located types
├── lib/                  # Core functionality (git, config, fuzzy search)
└── components/           # React components for TUI (future)
```

---

## 🔧 **TECHNICAL STACK**

### Core dependencies

- **yargs**: CLI parsing with auto-completion (chosen over commander).
- **Ink + React**: TUI framework (chosen over @clack/prompts).
- **Biome**: Linting/formatting (chosen over ESLint/Prettier).
- **Vitest**: Testing (chosen over Jest).
- **Direct git execution**: child_process (NOT simple-git - lacks worktree support).

### Code standards

- TypeScript strict mode with Node.js types.
- ES modules throughout.
- Use comments only when necessary.
- Biome formatting.

---

## 🎯 **IMPLEMENTATION DETAILS**

### Git operations

- Use `child_process.exec()` for git commands.
- Parse output manually for worktree operations.
- Handle cross-platform differences.
- Always validate git repository state.

### TUI design (decided: vim-like modal interface)

- **Layout**: FilterBar + two-panel flexbox + StatusLine.
- **Modal interface**: Normal/Insert/Command modes like vim.
- **Components**: App → FilterBar + WorktreeListPanel + DetailsPanel + StatusLine + HelpPanel.
- **Keybindings**: `j/k` navigate, `enter` switch, `c` clone, `d` delete, `/` search, `?` help, `q` quit.
- **Filtering**: Fuzzy search + status filters (dirty, ahead, behind, locked).
- **Clean UI**: No always-visible hotkeys, optional help toggle.

### Configuration

- cosmiconfig for flexible config loading (JSON/JS/TOML support).
- Global: `~/.config/grove/`.
- Repository: `.grove/`.
- Sensible defaults.

### Testing

- Unit tests for core functionality.
- Integration tests for git operations.
- Use `pnpm test:ci` for CI environments.
- Test TUI components with Ink testing utilities.

### Performance

- Lazy loading of git information.
- Debounced search filtering.
- Efficient React re-renders.
- Minimal git command execution.

### Pre-commit hooks

Automated quality enforcement using **Husky** and **lint-staged**:

#### Pre-commit hook

- **Format**: Auto-format TypeScript files with Biome, markdown files with Prettier.
- **Lint**: Auto-fix linting issues with Biome.
- **Type check**: Verify TypeScript compilation without emitting files.
- **Test**: Run full test suite to ensure no regressions.

#### Commit message hook

- **Validation**: Enforce [Conventional Commits](https://www.conventionalcommits.org/) format.
- **Allowed types**: `feat`, `fix`, `docs`, `style`, `refactor`, `perf`, `test`, `chore`, `ci`, `build`, `revert`.
- **Rules**: Subject must be lowercase, max 72 characters, not empty.

#### Setup

- Hooks are automatically installed via `pnpm install` (prepare script).
- Configuration in `package.json` (lint-staged) and `commitlint.config.js`.
- Manual hook testing: `.husky/pre-commit` and `echo "message" | npx commitlint`.
