# Grove Development Guide

## Project Context

Grove is a Git worktree management tool that makes worktrees as simple as switching branches.

**Essential reading:**

-   [Contributing Guide](CONTRIBUTING.md) - How to contribute
-   [Product Vision](PRODUCT.md) - Mission and user value
-   [Technical Standards](ARCHITECTURE.md) - Architecture and patterns
-   [Project Structure](STRUCTURE.md) - File organization

Follow steering documents for all architectural decisions.

## Development Commands

```bash
# Fast development loop
mage test  # Unit tests (~10s)
mage lint  # Auto-fix formatting
mage build # Build binaries

# Before commits
mage test:unit && mage test:integration
mage ci # Full pipeline
```
