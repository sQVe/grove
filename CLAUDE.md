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

## Project expertise

Run `mulch prime` at session start to load project knowledge.

- `mulch prime` — load all expertise domains
- `mulch prime --context` — load records for changed files
- `mulch search "query"` — find relevant records
- `mulch learn` — discover what to record from current session
- `mulch record <domain> --type <type> --description "..."` — record insight
