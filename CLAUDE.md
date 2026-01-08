# Grove Development Guide

## Project Context

Grove is a Git worktree management tool that makes worktrees as simple as switching branches.

**Essential reading:**

- [Contributing Guide](CONTRIBUTING.md) — Setup, architecture, and standards
- [Roadmap](ROADMAP.md) — Feature status and priorities

## Development principles

**TDD always** — Write tests first, implementation second.

## Development Commands

```bash
# Fast development loop
make test  # Unit tests (~10s)
make lint  # Auto-fix formatting
make build # Build binaries

# Before commits
make test-unit && make test-integration
make ci # Full pipeline
```
