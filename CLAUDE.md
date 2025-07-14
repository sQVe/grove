# Grove - AI assistant instructions

Grove-specific instructions for AI assistants working on this project.

## 🎯 Project context

Grove is a Git worktree management CLI tool being rewritten in Go for better performance and distribution.

### Key documentation
- **[README.md](README.md)**: Project overview and quick start
- **[FEATURES.md](FEATURES.md)**: Current and planned product capabilities  
- **[ARCHITECTURE.md](ARCHITECTURE.md)**: Technical design and implementation details
- **[CONTRIBUTING.md](CONTRIBUTING.md)**: Development guidelines and workflows
- **[ROADMAP.md](ROADMAP.md)**: Implementation status and milestones

### Current status
- **Rewriting from TypeScript to Go** for better performance
- **Focus on CLI-first approach** with optional TUI interface later
- **Direct git execution** for maximum compatibility

## 🚨 Critical requirements

### Development workflow
- **Always run validation**: See [CONTRIBUTING.md](CONTRIBUTING.md) for complete workflow
- **Update ROADMAP.md** when completing milestones or making architectural decisions
- **Follow Go conventions**: Use standard Go project layout and idioms

### Key implementation notes
- **Direct git execution**: Use `os/exec` to run git commands, parse output manually
- **Cross-platform support**: Handle Windows/macOS/Linux differences
- **Error handling**: Provide clear, actionable error messages
- **Configuration**: Support TOML/JSON config files with validation

