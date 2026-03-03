# CLAUDE.md

## Commands

- `make test` — run unit tests
- `make lint` — lint and fix violations
- `make build` — build binary
- `make ci` — run full CI pipeline

## Guidelines

- TDD always — write tests first, implementation second
- Follow existing patterns in `internal/`
- Add changelog entry via `make change` for significant changes

## Issue tracking

This project uses **br (beads_rust)** for issue tracking. Run `br prime` for workflow context.

**Note:** `br` is non-invasive and never executes git commands. After `br sync --flush-only`, you must manually run `git add .beads/ && git commit`.

- `br ready` - Find unblocked work
- `br list` - All open issues
- `br create "Title" --type task --priority 2` - Create issue
- `br close <id>` - Complete work
- `br sync --flush-only` - Flush to disk (then run `git add .beads/ && git commit`)

## Project expertise

Run `mulch prime` at session start to load project knowledge.

- `mulch prime` — load all expertise domains
- `mulch prime --context` — load records for changed files
- `mulch search "query"` — find relevant records
- `mulch learn` — discover what to record from current session
- `mulch record <domain> --type <type> --description "..."` — record insight
