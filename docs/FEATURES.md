# Grove Development Roadmap

> **This is the unified development roadmap for Grove.** All implementation plans, feature development, and technical improvements are tracked here.

## 🎯 Project Vision

Grove is a fast, intuitive Git worktree management CLI that makes worktrees as simple as switching branches. The development follows a phased approach building from foundation to advanced features.

---

## 📋 Development Phases

### Phase 1: Foundation (v0.1.0) ✅ **COMPLETE**

**Objective**: Establish robust foundation with core infrastructure and basic commands.

#### ✅ Completed Features

**Repository Management**

```bash
grove init                   # Initialize bare repo in current directory
grove init <directory>       # Initialize in specified directory
grove init <remote-url>      # Clone remote with worktree structure
grove init <remote-url> --branches=main,develop,feature/auth  # Multi-branch setup
```

**Configuration System**

```bash
grove config list                  # Show all configuration
grove config get general.editor    # Get a specific value
grove config set git.max_retries 5 # Set a configuration value
grove config validate              # Validate current configuration
grove config path                  # Show config file paths
grove config init                  # Create default config file
grove config reset [key]           # Reset to defaults
```

**Infrastructure Completed**

- ✅ CLI version support (`grove --version`)
- ✅ Comprehensive error handling with standardized codes
- ✅ TOML/JSON configuration with environment variable overrides (`GROVE_*`)
- ✅ Cross-platform config directories and validation
- ✅ Retry mechanisms with exponential backoff for network operations
- ✅ Filesystem-safe worktree directory naming (handles `fix/123` → `fix-123`)
- ✅ Robust git command execution with context and error recovery
- ✅ 96.4% test coverage with comprehensive test infrastructure
- ✅ Cross-platform compatibility (Windows/macOS/Linux)
- ✅ golangci-lint setup with strict code quality standards
- ✅ Mage build system for development automation

**Success Criteria**: ✅ All foundation infrastructure complete, basic commands working

---

### Phase 2: Core Commands (v0.2.0) 🚧 **IN PROGRESS**

**Objective**: Implement essential worktree management commands for daily development workflow.

#### 🚧 Current Sprint

**Immediate Next Steps** (Priority Order):

1. **Complete Mock Consolidation** - Remove duplicate `MockGitExecutor` in `/internal/git/worktree_test.go`
2. **Command Registration Framework** - Systematic command handling and discovery
3. **Core Worktree Commands** - List, create, switch, remove functionality
4. **Progress Indicators** - User feedback for long-running operations

#### 📅 Planned Deliverables

**Core Worktree Commands**
| Command | Description | Status |
|---------|-------------|--------|
| `grove list` | List all worktrees with status | 📅 Planned |
| `grove create <branch> [path]` | Create worktree from branch | 📅 Planned |
| `grove switch <worktree>` | Switch to worktree directory | 📅 Planned |
| `grove remove <worktree>` | Remove worktree safely | 📅 Planned |

**Enhanced User Experience**

- **Progress Indicators**: Visual feedback for clone, worktree creation, and network operations
- **Command Registration Framework**: Systematic command handling with auto-discovery
- **Basic Cleanup**: Remove merged/stale worktrees with safety checks

**Technical Improvements**

- **Mock Consolidation Cleanup**: Remove remaining duplicate mocks
- **Enhanced Error Messages**: Context-aware error reporting with actionable suggestions
- **Command Auto-completion**: Basic shell completion for commands and flags

#### 🎯 Success Criteria

- [ ] All core worktree commands implemented and tested
- [ ] Progress indicators working for long operations
- [ ] Command registration framework in place
- [ ] Mock consolidation complete
- [ ] User can manage worktrees through complete CLI workflow
- [ ] 90%+ test coverage maintained

**Target**: **End of Q1 2025**

---

### Phase 3: Enhanced Features (v0.3.0) 📅 **PLANNED**

**Objective**: Add advanced features for power users and team workflows.

#### 📋 Feature Scope

**Enhanced Status & Information**

- Worktree age and activity indicators
- Disk usage per worktree
- Configurable stale detection (30 days default)
- Rich status display with git state information

**Advanced Shell Integration**

- Comprehensive shell completion (bash, zsh, fish, PowerShell)
- Dynamic branch name completion from repositories
- Context-aware suggestions based on repository state

**Smart Cleanup & Management**

```bash
grove clean --merged      # Remove merged worktrees
grove clean --stale       # Remove stale worktrees (configurable threshold)
grove clean --interactive # Interactive cleanup with confirmations
```

**Performance & Quality**

- Increase test coverage to 95%+
- Performance optimization for large repositories
- Memory usage optimization
- Parallel operations where safe

#### 🎯 Success Criteria

- [ ] Advanced status features provide actionable insights
- [ ] Shell completion works across all major shells
- [ ] Smart cleanup safely manages worktree lifecycle
- [ ] Performance acceptable for large repositories (100+ worktrees)
- [ ] 95%+ test coverage

**Target**: **Mid 2025**

---

### Phase 4: Integrations & Polish (v1.0.0) 🔮 **FUTURE**

**Objective**: Professional-grade tool with external integrations and polished UX.

#### 🌟 Advanced Integrations

**Platform Integrations**

```bash
grove pr 123                 # Create worktree from GitHub PR
grove linear PROJ-456        # Create worktree from Linear issue
grove branch --from-template # Create from branch templates
```

**Interactive TUI Interface**

```bash
grove tui # Interactive interface with vim-like navigation
```

- Multi-panel layout with real-time git status
- Fuzzy search and advanced filtering
- Visual git state display
- Keyboard-driven workflow

**Authentication & Security**

- System keychain credential storage
- GitHub/Linear OAuth with multi-account support
- Environment variable fallbacks
- Secure token management

**Enterprise Features**

- Team configuration templates
- Workflow automation hooks
- Integration with CI/CD pipelines
- Advanced logging and monitoring

#### 🎯 Success Criteria

- [ ] External platform integrations working reliably
- [ ] TUI provides efficient alternative to CLI commands
- [ ] Authentication handles enterprise requirements
- [ ] Tool suitable for professional development teams
- [ ] Comprehensive documentation and examples

**Target**: **End of 2025**

---

## 🚀 Current Focus: Phase 2 Implementation

### Immediate Action Items

1. **Complete Mock Consolidation** (30 minutes)
    - Remove duplicate `MockGitExecutor` in `/internal/git/worktree_test.go` lines 10-49
    - Update tests to use centralized mock from `/internal/testutils/mocks.go`

2. **Implement Command Registration Framework** (4 hours)
    - Create `/internal/commands/registry.go` for systematic command handling
    - Create `/internal/commands/base.go` with common command interface
    - Update `cmd/grove/main.go` to use registry-based command discovery

3. **Build Core Worktree Commands** (2-3 days)
    - `grove list` - Display worktrees with status, branch, and path information
    - `grove create` - Create new worktree with branch handling and validation
    - `grove switch` - Navigate to worktree directory with shell integration
    - `grove remove` - Safe removal with dependency checks and confirmations

4. **Add Progress Indicators** (4 hours)
    - Create `/internal/ui/progress.go` with spinner and progress bar support
    - Integrate with git operations for clone, fetch, and worktree creation
    - Add configuration options for progress display preferences

### Development Commands

```bash
# Fast development workflow
mage test:unit # Run unit tests (~2s)
mage lint      # Run golangci-lint with --fix
mage build:all # Build all targets

# Full validation (before commits)
mage ci # Complete CI pipeline
```

---

## 📊 Project Metrics

| Metric            | Current | Target (v0.2.0) | Target (v1.0.0) |
| ----------------- | ------- | --------------- | --------------- |
| **Test Coverage** | 96.4%   | 90%+            | 95%+            |
| **Commands**      | 2       | 6               | 15+             |
| **Platforms**     | 3       | 3               | 3               |
| **Integrations**  | 0       | 0               | 3+              |

---

## 🔗 Related Documentation

- **[CONTRIBUTING.md](CONTRIBUTING.md)** - Development setup and workflows
- **[README.md](../README.md)** - Project overview and quick start

---

_Last updated: 2024-07-19 - Phase 1 complete, Phase 2 in progress_
