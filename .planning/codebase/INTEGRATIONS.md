# External Integrations

**Analysis Date:** 2026-01-23

## APIs & External Services

**GitHub:**

- Used for: PR information retrieval, merged PR detection, repository cloning
- SDK/Client: `gh` CLI (shell out via `os/exec`)
- Implementation: `internal/github/github.go`
- Auth: `GH_TOKEN` env var (auto-detected via `gh auth token`)

**Key GitHub operations:**

- `gh pr view` - Fetch PR details (branch name, fork info)
- `gh pr list --state merged` - Get merged PRs for prune suggestions
- `gh repo view` - Get clone URL with user's protocol preference
- `gh auth status` - Verify authentication

## Data Storage

**Databases:**

- None (filesystem-only tool)

**File Storage:**

- Local filesystem only
- Lock files: `<workspace>/.grove.lock` (`internal/workspace/lock.go`)
- Config files: `.grove.toml` per workspace (`internal/config/file.go`)

**Caching:**

- None (stateless between invocations)

## Authentication & Identity

**Auth Provider:**

- Delegates to `gh` CLI for GitHub authentication
- No internal auth handling

**Implementation:**

```go
// internal/github/github.go
func CheckGhAvailable() error {
    if _, err := exec.LookPath("gh"); err != nil {
        return errors.New("gh CLI not found")
    }
    cmd := exec.Command("gh", "auth", "status")
    // ...
}
```

## Monitoring & Observability

**Error Tracking:**

- None (CLI tool, errors to stderr)

**Logs:**

- Debug logging via `internal/logger/logger.go`
- Enabled with `--debug` flag or `grove.debug` git config
- Output to stderr

## CI/CD & Deployment

**Hosting:**

- GitHub Releases (binaries)
- Self-hosted (users install via `install.sh` or package managers)

**CI Pipeline:**

- GitHub Actions (`.github/workflows/`)
- `ci.yml` - Lint, test (unit + integration), build on push/PR
- `release.yml` - goreleaser on tag push
- `audit.yml` - Security audit
- `codeql.yml` - Code scanning
- `release-pr.yml` - Release PR automation

**Test matrix:**

- OS: ubuntu-latest, windows-latest, macos-latest
- Unit tests: `make test-coverage`
- Integration tests: `make test-integration` (requires `GH_TOKEN`)

## Environment Configuration

**Required env vars:**

- None for basic operation

**Optional env vars:**

- `GH_TOKEN` - GitHub authentication for PR features
- `CI` - Detected for different lint/test behavior

**Secrets location:**

- GitHub Actions secrets (`secrets.GITHUB_TOKEN`)
- User's local `gh` CLI auth

## Webhooks & Callbacks

**Incoming:**

- None

**Outgoing:**

- None

## External CLI Dependencies

**Required:**

- `git` (>= 2.48) - Core functionality
    - All git operations shell out via `os/exec`
    - Implementation: `internal/git/git.go`

**Optional:**

- `gh` (GitHub CLI) - PR-related features
    - PR checkout, merged PR detection
    - Implementation: `internal/github/github.go`

## Git Operations

The tool wraps git commands extensively. Key operations:

| Operation       | Git Command                  | Location                           |
| --------------- | ---------------------------- | ---------------------------------- |
| Clone           | `git clone --bare`           | `internal/git/git.go:Clone()`      |
| Add worktree    | `git worktree add`           | `internal/git/worktree.go`         |
| List worktrees  | `git worktree list`          | `internal/git/worktree.go`         |
| Remove worktree | `git worktree remove`        | `internal/git/worktree.go`         |
| Fetch           | `git fetch --prune`          | `internal/git/git.go:FetchPrune()` |
| Branch ops      | `git branch`, `git checkout` | `internal/git/branch.go`           |
| Status          | `git status`                 | `internal/git/status.go`           |
| Config          | `git config`                 | `internal/config/config.go`        |

## User-Defined Hooks

**Add hooks:**

- Configured in `.grove.toml` under `[hooks]`
- Executed via `sh -c` on worktree creation
- Implementation: `internal/hooks/hooks.go`

```go
// internal/hooks/hooks.go
cmd := exec.Command("sh", "-c", cmdStr)
cmd.Dir = workDir
```

---

_Integration audit: 2026-01-23_
