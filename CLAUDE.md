# Grove Development Guide

## Project Context

Grove is a Git worktree management tool that makes worktrees as simple as switching branches.

**Essential reading:**

- [Contributing Guide](CONTRIBUTING.md) - How to contribute
- [Product Vision](.spec-workflow/steering/product.md) - Mission and user value
- [Technical Standards](.spec-workflow/steering/tech.md) - Architecture and patterns
- [Project Structure](.spec-workflow/steering/structure.md) - File organization

Follow steering documents for all architectural decisions.

## Development Commands

```bash
# Fast development loop
mage test:unit # Unit tests (~10s)
mage lint      # Auto-fix formatting
mage build:all # Build binaries

# Before commits
mage test:unit && mage test:integration
mage ci # Full pipeline
```
