# Features & Roadmap

## Current ✅

### Repository Setup

```bash
grove init                   # Initialize bare repo in current directory
grove init <directory>       # Initialize in specified directory
grove init <remote-url>      # Clone remote with worktree structure
grove init <remote-url> --branches=main,develop,feature/auth  # Multi-branch setup
```

**Features**: 
- Worktree-optimized structure with .bare/ subdirectory
- Smart URL parsing for GitHub, GitLab, Bitbucket, Azure DevOps, Gitea, Codeberg
- Multi-branch worktree creation with --branches flag
- Automatic branch detection from URLs (e.g., github.com/repo/tree/branch)
- Repository conversion from traditional Git structure
- Cross-platform compatibility and robust error handling

### Infrastructure

- Robust git command execution with error handling
- Repository validation and comprehensive URL detection  
- 96.4% test coverage with comprehensive test infrastructure
- Cross-platform compatibility (Windows/macOS/Linux)

## Planned 📅

### Core Commands

| Command                        | Description                    |
| ------------------------------ | ------------------------------ |
| `grove list`                   | List all worktrees with status |
| `grove create <branch> [path]` | Create worktree from branch    |
| `grove switch <worktree>`      | Switch to worktree directory   |
| `grove remove <worktree>`      | Remove worktree safely         |

### Configuration

- TOML configuration files
- Environment variable overrides
- Cross-platform config directories

### Integrations

| Feature     | Commands                       | Description                |
| ----------- | ------------------------------ | -------------------------- |
| **GitHub**  | `grove pr 123`                 | Create worktree from PR    |
| **Linear**  | `grove linear PROJ-456`        | Create worktree from issue |
| **Cleanup** | `grove clean --merged/--stale` | Smart worktree cleanup     |

### Enhanced Status

- Worktree age and activity indicators
- Disk usage per worktree
- Configurable stale detection (30 days default)

### TUI Interface

```bash
grove tui  # Interactive interface with vim-like navigation
```

- Multi-panel layout with real-time git status
- Fuzzy search and status filters
- Visual git state display

### Authentication

- System keychain credential storage
- GitHub/Linear OAuth with multi-account support
- Environment variable fallbacks

## Roadmap

| Version    | Status         | Features                                           |
| ---------- | -------------- | -------------------------------------------------- |
| **v0.1.0** | ⏳ In Progress | Core foundation, init command, testing             |
| **v0.2.0** | 📅 Planned     | Worktree management, cleanup, enhanced status      |
| **v0.3.0** | 🔮 Future      | GitHub/Linear integration, authentication          |
| **v1.0.0** | 🔮 Future      | Complete CLI with TUI, cross-platform distribution |

### Current Phase: Core Foundation

- [x] Project structure and CLI architecture
- [x] Git operations infrastructure
- [x] `grove init` command
- [x] Comprehensive testing (85.6% coverage)
- [x] golangci-lint setup
- [ ] Mage build system
- [ ] Core worktree commands
- [ ] Configuration system
