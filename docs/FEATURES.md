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

- Worktree-optimized structure with `.bare/` subdirectory
- Smart URL parsing for GitHub, GitLab, Bitbucket, Azure DevOps, Gitea, Codeberg
- Multi-branch worktree creation with the `--branches` flag
- Automatic branch detection from URLs (e.g., github.com/repo/tree/branch)
- Repository conversion from traditional Git structure
- Cross-platform compatibility and robust error handling

### Configuration System

```bash
grove config list                  # Show all configuration
grove config get general.editor    # Get a specific value
grove config set git.max_retries 5 # Set a configuration value
grove config validate              # Validate current configuration
grove config path                  # Show config file paths
grove config init                  # Create default config file
grove config reset [key]           # Reset to defaults
```

**Features**:

- TOML configuration files with YAML/JSON support
- Environment variable overrides (using `GROVE_*` prefix)
- Cross-platform config directories
- Built-in validation and helpful error messages
- Configuration sections: general, git, retry, logging, worktree

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
grove tui # Interactive interface with vim-like navigation
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
- [x] Configuration system with CLI commands
- [x] Comprehensive testing (96.4% coverage)
- [x] golangci-lint setup
- [x] Mage build system
- [ ] Core worktree commands
