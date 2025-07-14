# Grove roadmap

Development roadmap and implementation tracking for Grove.

## Current status

Grove is being rewritten in Go for better performance and easier distribution. The TypeScript implementation is being replaced with a focused CLI-first approach.

## Implementation phases

### Phase 1: Core CLI Foundation ⏳ *In Progress*

**Goal**: Essential worktree management commands with Go foundation

- [x] Project structure setup
- [x] Basic CLI architecture planning
- [ ] Core git operations (`git worktree list`, `git worktree add`, `git worktree remove`)
- [ ] Configuration system (TOML support)
- [ ] Cross-platform compatibility
- [ ] Error handling and validation
- [ ] Basic commands: `grove`, `grove init`, `grove create`, `grove switch`, `grove list`

### Phase 2: Enhanced Features 📅 *Planned*

**Goal**: Improved user experience and productivity features

- [ ] Smart cleanup commands (`grove clean --merged`, `grove clean --stale`)
- [ ] Enhanced status display (age indicators, disk usage)
- [ ] JSON output format for scripting
- [ ] Fuzzy search for worktree selection
- [ ] Configuration file management
- [ ] Performance optimizations

### Phase 3: Service Integrations 🔮 *Future*

**Goal**: Seamless integration with development workflows

- [ ] GitHub PR integration (`grove pr 123`)
- [ ] Linear issue integration (`grove linear PROJ-456`)
- [ ] Authentication system (OAuth, token management)
- [ ] Branch name generation from PR/issue metadata
- [ ] Status updates and workflow automation

### Phase 4: Interactive TUI 🔮 *Future*

**Goal**: Optional interactive interface for power users

- [ ] Multi-panel TUI layout
- [ ] Vim-like navigation and keybindings
- [ ] Real-time git status updates
- [ ] Interactive cleanup and management
- [ ] Mouse support and contextual actions

## Milestones

### v0.1.0 - Core Foundation
- [ ] Basic CLI commands working
- [ ] Cross-platform compatibility
- [ ] Configuration system
- [ ] Test suite and CI/CD

### v0.2.0 - Enhanced Experience
- [ ] Smart cleanup features
- [ ] Enhanced status display
- [ ] Fuzzy search integration
- [ ] Performance optimizations

### v0.3.0 - Service Integrations
- [ ] GitHub PR support
- [ ] Linear issue support
- [ ] Authentication system
- [ ] Workflow automation

### v1.0.0 - Complete CLI
- [ ] All core features stable
- [ ] Comprehensive documentation
- [ ] Cross-platform distribution
- [ ] Community feedback integration

## Current priorities

1. **Core git operations**: Implement reliable worktree management
2. **Cross-platform testing**: Ensure Windows/macOS/Linux compatibility
3. **Configuration system**: TOML-based configuration with validation
4. **Error handling**: Clear, actionable error messages
5. **Testing**: Comprehensive test suite for git operations

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development guidelines and [FEATURES.md](FEATURES.md) for detailed feature descriptions.
