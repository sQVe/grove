# Grove

A fast, intuitive Git worktree management CLI that makes Git worktrees as simple as switching branches.

## Installation

```bash
git clone https://github.com/sqve/grove && cd grove
go build -o grove ./cmd/grove && ./grove --help
```

## Usage

```bash
grove init                              # Initialize bare repo for worktrees
grove init https://github.com/user/repo # Clone remote with worktree structure
```

## Features

- ✅ Repository initialization and remote cloning
- ✅ Cross-platform (Windows/macOS/Linux)
- ✅ 96.4% test coverage
- 🚧 Worktree management (list, create, switch, remove)
- 📅 GitHub/Linear integration, smart cleanup, TUI

## Documentation

- **[FEATURES.md](docs/FEATURES.md)** - Complete features and roadmap
- **[CONTRIBUTING.md](docs/CONTRIBUTING.md)** - Development and architecture

MIT License
