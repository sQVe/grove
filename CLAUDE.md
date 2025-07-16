# Grove - AI assistant instructions

Grove-specific instructions for AI assistants working on this project.

## 🎯 Project context

Grove is a Git worktree management CLI tool being rewritten in Go for better performance and distribution.

### Key documentation

- **[README.md](README.md)**: Project overview and quick start
- **[FEATURES.md](FEATURES.md)**: Complete feature documentation and development roadmap
- **[CONTRIBUTING.md](CONTRIBUTING.md)**: Development guidelines, architecture, and workflows

## 🚨 Critical requirements

### Development workflow

- **Always run validation**: See [CONTRIBUTING.md](CONTRIBUTING.md) for complete workflow
- **Update FEATURES.md** when completing milestones or making architectural decisions
- **Follow Go conventions**: Use standard Go project layout and idioms

### Key implementation notes

- **Direct git execution**: Use `os/exec` to run git commands, parse output manually
- **Cross-platform support**: Handle Windows/macOS/Linux differences
- **Error handling**: Provide clear, actionable error messages
- **Configuration**: Support TOML/JSON config files with validation
