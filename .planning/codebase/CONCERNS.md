# Codebase Concerns

**Analysis Date:** 2026-01-23

## Tech Debt

**Large Doctor Command File:**

- Issue: `cmd/grove/commands/doctor.go` at 1086 lines is significantly larger than other command files
- Files: `cmd/grove/commands/doctor.go`
- Impact: Harder to maintain, test, and understand; mixes version parsing, issue detection, output formatting, and fix logic
- Fix approach: Extract version parsing to `internal/version/`, issue detection to `internal/doctor/`, keep only cobra wiring in command file

**Duplicate Version Parsing Logic:**

- Issue: Version parsing functions (`parseVersion`, `parseGitVersionOutput`, `parseGhVersionOutput`, `compareVersions`) are defined in `doctor.go` but could be reused elsewhere
- Files: `cmd/grove/commands/doctor.go:40-132`
- Impact: Duplicated logic if version checking needed elsewhere; not unit testable in isolation
- Fix approach: Move to `internal/version/version.go` with proper unit tests

**Cross-Filesystem Directory Moves Not Supported:**

- Issue: `RenameWithFallback` explicitly does not support cross-filesystem directory moves
- Files: `internal/fs/fs.go:110`
- Impact: Conversion operations fail if workspace and target are on different filesystems
- Fix approach: Implement recursive copy for directories when `os.Rename` fails

**Hardcoded GitHub-Only Integration:**

- Issue: GitHub integration is tightly coupled; no abstraction for other git hosts (GitLab, Bitbucket)
- Files: `internal/github/github.go`, `cmd/grove/commands/add.go`
- Impact: Cannot support PRs/MRs from non-GitHub repos; limits adoption
- Fix approach: Create `internal/forge/` interface with GitHub as first implementation

## Known Bugs

**None detected during analysis.**

All tests pass. No explicit TODO/FIXME/BUG comments found in codebase.

## Security Considerations

**Git Command Injection Protection:**

- Risk: Branch names and paths could contain shell metacharacters
- Files: `internal/git/*.go` (all git command execution)
- Current mitigation: Using `exec.Command` (not shell) prevents injection; nolint comments document why branch names from git are trusted
- Recommendations: Current approach is correct; continue avoiding shell execution

**Lock File Race Conditions:**

- Risk: File lock acquisition could race with another process
- Files: `internal/workspace/lock.go`, `internal/workspace/workspace.go:810`
- Current mitigation: `O_EXCL` flag for atomic creation; platform-specific advisory locks
- Recommendations: Document that locks are advisory on Unix; consider flock integration

**Recovery File Contains Paths:**

- Risk: `.grove-recovery.txt` written on conversion failure may expose file paths
- Files: `internal/workspace/workspace.go:898-918`
- Current mitigation: File written to workspace directory, not world-readable
- Recommendations: Acceptable for debugging; consider adding to .gitignore template

## Performance Bottlenecks

**Sequential Worktree Info Gathering:**

- Problem: `ListWorktreesWithInfo` calls `GetWorktreeInfo` sequentially for each worktree
- Files: `internal/git/worktree.go:199-256`
- Cause: Each worktree requires multiple git commands (branch, status, sync status)
- Improvement path: Parallelize with goroutines; batch git commands where possible

**Prune Fetches Merged PRs One-by-One:**

- Problem: GitHub API call fetches up to 200 merged PRs in `GetMergedPRBranches`
- Files: `internal/github/github.go:222-240`
- Cause: Single API call is good, but called even when not needed (only used for squash merge detection)
- Improvement path: Lazy-load only when squash merge detection actually needed

**Remote Reachability Checks in Doctor:**

- Problem: `detectRemoteIssues` runs `git ls-remote` for each remote with 5s timeout
- Files: `cmd/grove/commands/doctor.go:522-558`
- Cause: Necessary for network diagnostics but slow on unreachable remotes
- Improvement path: Already parallelized with goroutines; consider caching results

## Fragile Areas

**Conversion Rollback Logic:**

- Files: `internal/workspace/workspace.go:843-929`
- Why fragile: Complex defer-based rollback with multiple failure modes; error accumulation during recovery
- Safe modification: Test all failure paths; ensure `conversionSucceeded` flag is set correctly before any early returns
- Test coverage: Unit tests exist but edge cases around partial failures are hard to simulate

**Git Porcelain Output Parsing:**

- Files: `internal/git/worktree.go:151-196`, `internal/git/status.go:217-286`
- Why fragile: Depends on specific git output formats; may break with future git versions
- Safe modification: Use `--porcelain` flag consistently (stable across versions); add version checks if format changes
- Test coverage: Good coverage via integration tests with real git

**PR URL Regex Patterns:**

- Files: `internal/github/github.go:22-25`
- Why fragile: Regex patterns assume specific GitHub URL formats; may break with URL structure changes
- Safe modification: Add comprehensive unit tests for URL variants; consider using GitHub's URL parsing library
- Test coverage: Basic tests exist in `github_test.go`

## Scaling Limits

**In-Memory Worktree List:**

- Current capacity: All worktrees loaded into memory as `[]*WorktreeInfo`
- Limit: Thousands of worktrees would cause memory pressure
- Scaling path: Stream processing for `list` command; pagination for large workspaces

**Merged PR Cache:**

- Current capacity: 200 PRs fetched per `gh pr list` call
- Limit: Repos with >200 recently merged PRs may miss some for squash detection
- Scaling path: Paginate through older PRs or use specific branch lookup

## Dependencies at Risk

**None identified.**

- Core dependencies (`spf13/cobra`, `BurntSushi/toml`) are stable and widely used
- `gh` CLI dependency is optional (only for PR features)
- Minimum Git version (2.48) is documented and checked at runtime

## Missing Critical Features

**No Parallel Worktree Operations:**

- Problem: Operations like `grove exec` run sequentially across worktrees
- Blocks: Fast bulk operations across many worktrees
- Note: `grove exec` exists but runs commands one at a time; adding `--parallel` flag would help

**No Branch Cleanup After PR Merge:**

- Problem: After PR is merged, local branch and worktree remain; `prune` must be run manually
- Blocks: Automatic cleanup workflows
- Note: Could add post-merge hook or integration with GitHub Actions

## Test Coverage Gaps

**No E2E Tests:**

- What's not tested: Full command-line invocations with real git repos
- Files: `cmd/grove/commands/*.go`
- Risk: Integration issues between components may go undetected
- Priority: Medium - script_test.go provides some coverage via shell scripts

**Conversion Edge Cases:**

- What's not tested: Cross-filesystem conversions, conversion with large files, conversion interruption
- Files: `internal/workspace/workspace.go` (Convert function)
- Risk: Rollback logic may fail in unusual scenarios
- Priority: High - conversion is destructive and hard to recover from

**Windows-Specific Code Paths:**

- What's not tested: Windows lock implementation, case-insensitive path handling
- Files: `internal/workspace/lock_windows.go`, `internal/fs/fs.go:188-209`
- Risk: Windows users may encounter untested bugs
- Priority: Low - CI includes Windows but coverage may be incomplete

**Error Messages for Invalid Configurations:**

- What's not tested: User-facing error messages for malformed `.grove.toml`
- Files: `internal/config/file.go`
- Risk: Cryptic errors when config is invalid
- Priority: Low - TOML library provides reasonable defaults

---

_Concerns audit: 2026-01-23_
