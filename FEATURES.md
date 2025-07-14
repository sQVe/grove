# Features

Current and planned capabilities for Grove Git worktree management.

## Current capabilities

### Core CLI commands

```bash
grove                        # List all worktrees
grove init [path]            # Initialize bare repository  
grove create <branch> [path] # Create worktree from branch
grove switch <worktree>      # Switch to worktree
grove list                   # Enhanced listing with status
```

### Enhanced worktree listing

- **Color-coded status**: Clean (green), dirty (yellow), ahead/behind (blue/red)
- **Structured output**: Name, branch, status, and path columns
- **Active worktree highlighting**: Shows current location
- **JSON output**: `--format=json` for scripting
- **Locked worktree detection**: Shows locked status

### Git integration

- **Repository validation**: Ensures valid git repository
- **Branch validation**: Checks branch existence before creation
- **Cross-platform execution**: Windows, macOS, Linux support
- **Error handling**: Clear messages with suggested solutions

## Planned features

### GitHub integration

```bash
grove pr 123 # Create worktree from PR
grove pr https://github.com/org/repo/pull/123
```

- Fetch PRs from origin and forks automatically
- Smart branch naming: `pr-123-feature-name`
- Display PR metadata (author, status, CI checks)

### Linear integration

```bash
grove linear PROJ-456 # Create worktree from issue
grove linear https://linear.app/team/issue/PROJ-456
```

- Auto-generate branch names from issue titles
- Display issue metadata (assignee, status, priority)
- Update issue status on worktree operations

### Smart cleanup

```bash
grove clean --merged      # Remove merged worktrees
grove clean --stale       # Remove unused worktrees
grove clean --interactive # Interactive cleanup
```

- Safe deletion with confirmation prompts
- Preserve worktrees with uncommitted changes
- Configurable stale detection (default: 30 days)

### Enhanced status

```bash
grove list --stale      # Show age indicators
grove list --disk-usage # Include disk space
grove list --all        # Show merged/stale status
```

- Worktree age and last activity timestamps
- Disk space usage per worktree  
- Activity scoring based on recent commits

### Interactive TUI

```bash
grove tui # Launch interactive interface
grove     # Default to TUI if configured
```

- Multi-panel layout with real-time git status
- Vim-like navigation (`j/k`, `gg/G`, `enter`, `c`, `d`, `/`, `?`, `q`)
- Fuzzy search with status filters (dirty, clean, ahead, behind)
- Visual git state display with rich metadata

## Authentication

- Secure credential storage using system keychain
- GitHub OAuth and Linear OAuth
- Environment variable fallbacks
- Multi-account support for organizations
