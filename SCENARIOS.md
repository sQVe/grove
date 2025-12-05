# Grove Test Scenarios

This document captures test scenarios for Grove's CLI commands. Each scenario describes an input, the required preconditions (state), and the expected behavior.

## How This Document Was Created

### Methodology

For each command, we:

1. **Spawned 3 parallel exploration agents** with identical instructions to thoroughly analyze the command's implementation
2. **Consolidated findings** — each agent found overlapping but also unique scenarios
3. **Deduplicated and organized** by category (input parsing, state checks, error handling, etc.)

### Why 3 Agents?

Running multiple agents with the same prompt produces different exploration paths. One agent might focus on error handling, another on edge cases, a third on flag combinations. The union of their findings is more comprehensive than any single pass.

### Scenario Format

```markdown
- [ ] <input or action> → <expected behavior>
    - State: <preconditions required>
```

- **Checkbox**: For tracking manual testing progress
- **Input**: What triggers the scenario
- **Expected behavior**: What should happen
- **State**: Required repo/filesystem/git state before the test

### Adding New Scenarios

When adding scenarios for a new command:

1. Explore the command implementation thoroughly
2. Cover: input parsing, validation, git operations, filesystem operations, error handling, edge cases
3. Include both success and failure paths
4. Document the required state for each scenario

---

## grove add

### Input Parsing

- [x] PR URL with `/files` suffix (`/pull/123/files`) → should strip suffix, create worktree
    - State: valid workspace, gh authenticated
- [x] PR URL with `/commits` suffix → should strip suffix
    - State: valid workspace, gh authenticated
- [x] PR URL with query params (`?diff=split`) → should ignore params
    - State: valid workspace, gh authenticated
- [x] PR URL with trailing slash (`/pull/123/`) → should handle
    - State: valid workspace, gh authenticated
- [x] Branch name with leading/trailing whitespace → should trim
    - State: valid workspace
- [x] Branch name with slashes (`feature/auth/oauth`) → directory sanitized to `feature-auth-oauth`
    - State: valid workspace
- [x] Branch name with special chars (`<>|"?*:`) → sanitized to dashes
    - State: valid workspace
- [x] Empty argument after trimming → error
    - State: any
- [x] `--name` with empty string → falls back to default naming
    - State: valid workspace
- [x] `--name` with path separators → creates nested directories (not sanitized)
    - State: valid workspace
- [x] Very long branch name (200+ chars) → creates worktree with long dir name
    - State: valid workspace

### Flag Validation

- [x] `--detach` + `--base` together → error: "cannot be used together"
    - State: any
- [x] `--base` + PR reference (`#123`) → error: "cannot be used with PR references"
    - State: any
- [x] `--detach` + PR reference → error: "cannot be used with PR references"
    - State: any
- [x] `--base` + PR URL → error: "cannot be used with PR references"
    - State: any
- [x] `--base` with non-existent branch → error: "base branch does not exist"
    - State: valid workspace
- [x] `--base` with existing target branch → error: "cannot use --base with existing branch"
    - State: valid workspace, target branch exists

### Workspace Detection

- [x] Run from non-grove directory → error: "not in a grove workspace"
    - State: regular git repo or temp directory
- [x] Run from workspace root → works
    - State: CWD is workspace root
- [x] Run from inside worktree → works, uses as source for preservation
    - State: CWD inside worktree
- [x] Run from subdirectory inside worktree → finds worktree root
    - State: CWD is `workspace/main/src/pkg`
- [~] Workspace with corrupted `.bare` directory → graceful error (DEFERRED: hard to simulate safely)
    - State: `.bare` exists but invalid

### Git State

- [x] Branch already has worktree → error: "worktree already exists for branch"
    - State: branch checked out in another worktree
- [~] Existing worktree in detached HEAD state → skip with warning, operation succeeds (DEFERRED: complex setup)
    - State: another worktree is detached (like `review` example)
- [x] Merge in progress in source worktree → should not affect add
    - State: source worktree mid-merge
- [x] Rebase in progress → should not affect add
    - State: source worktree mid-rebase
- [x] Empty repository (no commits) → creates orphan worktree
    - State: newly initialized repo
- [~] Shallow clone → may have limitations (DEFERRED: platform-specific)
    - State: `git clone --depth 1`

### Branch Operations

- [x] Add existing local branch → creates worktree
    - State: branch exists locally
- [x] Add existing remote branch → creates worktree (tracking not auto-configured)
    - State: `origin/feature` exists
- [x] Add new branch (no `--base`) → creates branch from HEAD
    - State: branch doesn't exist
- [x] Add new branch with `--base main` → creates branch from main
    - State: main exists, new branch doesn't
- [x] `--detach` with valid tag → creates detached worktree
    - State: tag exists
- [x] `--detach` with commit SHA → creates detached worktree
    - State: commit exists
- [x] `--detach` with non-existent ref → error: "ref does not exist"
    - State: ref doesn't exist

### PR Operations

- [x] PR number (`#123`) with valid origin → creates worktree
    - State: gh authenticated, origin is GitHub repo
- [x] PR number without origin remote → error: "requires workspace context"
    - State: no origin configured (integration test: add_pr_integration.txt)
- [x] PR doesn't exist → error: "PR #N not found"
    - State: gh authenticated
- [x] Same-repo PR → fetches from origin
    - State: PR is not from fork
- [~] Fork PR → adds remote, fetches, creates worktree (DEFERRED: requires external fork repo)
    - State: PR is from fork
- [x] Fork PR, fetch fails → cleans up added remote
    - State: remote added but fetch fails (unit test: TestForkRemoteCleanup)
- [x] Fork PR, worktree creation fails → cleans up added remote
    - State: remote added, fetch succeeded, worktree fails (unit test: TestForkRemoteCleanup)
- [x] Fork remote already exists → reuses existing remote
    - State: `pr-123-forkuser` remote exists (unit test: TestForkRemoteCleanup)
- [x] gh CLI not installed → error: "gh CLI not found"
    - State: gh not in PATH (unit test: TestGhErrorMessages, skips in integration)
- [x] gh CLI not authenticated → error: "gh not authenticated"
    - State: gh installed but not logged in (unit test: TestGhErrorMessages, skips in integration)
- [x] PR branch already has worktree → error: "worktree already exists"
    - State: PR's branch is checked out elsewhere

### Directory Handling

- [x] Directory already exists (not a worktree) → error: "directory already exists"
    - State: `workspace/feature` exists as regular dir
- [x] Directory name collision after sanitization → error: "directory already exists"
    - State: `feat/test` and `feat\test` both → `feat-test`
- [x] Custom name via `--name` → uses custom name
    - State: valid workspace
- [~] Path too long for filesystem → graceful error (DEFERRED: platform-specific, Windows)
    - State: branch name creates 260+ char path on Windows

### Locking

- [x] Another `grove add` in progress → waits or fails with lock message
    - State: lock file exists, PID active
- [x] Stale lock file (crashed process) → detects and overrides
    - State: lock file exists, PID dead or old
- [x] Lock cleanup on success → removes lock file
    - State: operation completes (integration test: add_integration.txt)
- [x] Lock cleanup on failure → removes lock file
    - State: operation fails mid-way (integration test: add_integration.txt)

### Auto-Lock

- [x] Branch matches auto-lock pattern (`main`) → worktree locked
    - State: default auto-lock patterns
- [x] Branch matches custom pattern (`release/*`) → worktree locked
    - State: custom pattern in config
- [x] Branch doesn't match patterns → not locked
    - State: feature branch
- [x] Detached worktree → not auto-locked (no branch)
    - State: `--detach` used
- [x] Auto-lock fails → warning only, worktree still created
    - State: lock operation errors (implicit: uses Debug log, continues)

### File Preservation

- [x] Source worktree has `.env` → copied to new worktree
    - State: `.env` exists and ignored by git
- [x] Source worktree has multiple `.env.*` files → all matching preserved
    - State: `.env.local`, `.env.test` exist
- [x] File in `node_modules/.env` → NOT preserved (excluded path)
    - State: file matches pattern but in excluded dir
- [x] File already exists in destination → skipped, not overwritten
    - State: destination has conflicting file
- [x] No source worktree (run from root) → no preservation
    - State: CWD is workspace root
- [x] TOML config patterns override defaults → uses TOML patterns
    - State: `.grove.toml` has custom preserve patterns
- [x] Git config patterns used if no TOML → uses git config
    - State: `grove.preserve` set in git config

### Hooks

- [x] Hooks configured, all succeed → listed in output
    - State: `.grove.toml` has `hooks.add`
- [x] Hook fails (non-zero exit) → warning shown, worktree still created
    - State: hook exits with error
- [x] Multiple hooks, first fails → stops at first failure
    - State: multiple hooks configured
- [x] No hooks configured → no hook output
    - State: no `hooks.add` in config
- [x] Invalid TOML config → warning, hooks disabled
    - State: `.grove.toml` has syntax error

### Output Modes

- [x] Normal output → "Created worktree at <path>" + details
    - State: no `--switch`
- [x] `--switch` output → only path, no other output
    - State: `-s` or `--switch` flag
- [x] Detached output → "Created detached worktree at <path>"
    - State: `--detach` used
- [x] PR output → "Created worktree for PR #N at <path>"
    - State: PR reference used (integration test: add_pr_integration.txt)
- [x] Preserved files listed → shows copied files
    - State: files were preserved
- [x] Skipped files listed → shows already-existing files
    - State: some files not copied (integration test: add_integration.txt)

### Error Recovery

- [x] Git worktree add fails → error propagated with context
    - State: git command fails (e.g., non-existent ref with --detach)
- [~] Disk full → graceful error (DEFERRED: OS-level simulation)
    - State: no space left
- [~] Permission denied → clear error (DEFERRED: OS-level simulation)
    - State: can't write to workspace
- [~] Network timeout during fetch → error with context (DEFERRED: network-level simulation)
    - State: slow/offline network

---

## grove remove

### Input Parsing

- [x] No arguments → error (requires exactly 1)
    - State: any
- [x] Empty/whitespace argument → error: "worktree not found"
    - State: valid workspace
- [x] Match by directory name (exact) → removes worktree
    - State: worktree `feature-auth` exists
- [x] Match by branch name (fallback) → removes worktree
    - State: branch `feature/auth`, directory `feature-auth`
- [x] Directory name takes precedence over branch name → removes by dir
    - State: dir `develop` with branch `main`, dir `main` with branch `develop`
- [x] Worktree not found → error: "worktree not found: <name>"
    - State: no matching worktree
- [x] Detached HEAD worktree → removes normally
    - State: worktree in detached state

### Current Worktree Protection

- [x] CWD is target worktree root → error: "cannot delete current worktree"
    - State: CWD is `/workspace/main`
- [x] CWD is subdirectory of target → error: "cannot delete current worktree"
    - State: CWD is `/workspace/main/src/pkg`
- [x] CWD is different worktree → succeeds
    - State: CWD is `/workspace/feature`, removing `main`
- [x] CWD is workspace root → succeeds
    - State: CWD is `/workspace`

### Dirty State (without --force)

- [x] Untracked files only → error: "uncommitted changes"
    - State: worktree has new untracked files
- [x] Modified tracked files → error: "uncommitted changes"
    - State: worktree has modified files
- [x] Staged but uncommitted → error: "uncommitted changes"
    - State: worktree has staged changes
- [x] Clean worktree → succeeds
    - State: no uncommitted changes
- [~] git status fails → error propagated — DEFERRED: requires corrupted worktree
    - State: corrupted worktree

### Locked State (without --force)

- [x] Locked worktree (no reason) → error: "worktree is locked"
    - State: lock file exists, empty
- [x] Locked worktree (with reason) → error: "worktree is locked"
    - State: lock file has reason text (note: reason not shown in error)
- [x] Unlocked worktree → succeeds
    - State: no lock file
- [x] Dirty AND locked → dirty error shown first
    - State: both conditions true

### Force Flag

- [x] `--force` with dirty worktree → succeeds
    - State: uncommitted changes
- [x] `--force` with locked worktree → unlocks, then removes
    - State: worktree locked
- [x] `--force` with dirty AND locked → succeeds
    - State: both conditions
- [x] `-f` shorthand works → same as `--force`
    - State: dirty worktree
- [~] Unlock fails during force → logs debug, continues removal — DEFERRED: hard to simulate
    - State: unlock operation errors

### Branch Deletion (--branch)

- [x] `--branch` with clean worktree → deletes both
    - State: merged branch
- [x] `--branch` with unpushed commits → warns, branch delete fails without force
    - State: branch N commits ahead (use --force to delete anyway)
- [~] `--branch` with no upstream → no warning, deletes — DEFERRED: same behavior as merged
    - State: no upstream configured
- [x] `--branch` with upstream gone → no warning, deletes
    - State: upstream deleted
- [x] `--branch` unmerged without force → worktree removed, branch fails
    - State: unmerged commits, no `--force`
- [x] `--force --branch` unmerged → both deleted
    - State: unmerged commits
- [x] Worktree removed but branch delete fails → error with context
    - State: git branch -d fails (same as unmerged without force)

### Git Operations

- [x] Successful removal → directory deleted, git metadata cleaned
    - State: clean worktree
- [~] git worktree remove fails → error propagated — DEFERRED: hard to simulate
    - State: git error
- [x] Phantom worktree (dir manually deleted) → shows warning, not found error
    - State: worktree in list but no directory (note: Grove doesn't handle gracefully)
- [x] Worktree with submodules → git blocks removal (git limitation)
    - State: initialized submodules (workaround: rm -rf && git worktree prune)

### Output

- [x] Without `--branch` → "Deleted worktree <name>"
    - State: normal removal
- [x] With `--branch` → "Deleted worktree and branch <name>"
    - State: both deleted
- [~] Unpushed commits warning → "Branch has N unpushed commit(s)" — DEFERRED: requires remote tracking
    - State: `--branch` with ahead count > 0

### Edge Cases

- [x] Worktree path with spaces → handled correctly
    - State: directory name has spaces
- [~] Case-insensitive filesystem → matches correctly — DEFERRED: platform-specific (macOS/Windows)
    - State: macOS/Windows
- [x] Multiple worktrees with similar names → exact match only
    - State: `feat`, `feature`, `feat-new` exist
- [x] Worktree created outside grove → still removable
    - State: created with `git worktree add` directly
- [x] Ongoing merge/rebase → dirty check catches it
    - State: merge in progress
- [x] Remove then recreate → succeeds
    - State: remove, then `grove add` same name

### Shell Completion

- [x] Suggests removable worktrees → excludes current
    - State: CWD in one worktree
- [x] No suggestions for second argument → stops after first
    - State: first arg provided
- [~] Not in workspace → error directive — DEFERRED: unit test territory
    - State: random directory

---

## grove switch

### Input Parsing

- [x] No arguments → error: requires exactly 1
    - State: any
- [x] Too many arguments → error
    - State: any
- [x] Empty string after trimming → error: "worktree not found"
    - State: valid workspace
- [x] Leading/trailing whitespace → trimmed, matches worktree
    - State: `"  main  "` → finds `main`
- [x] Branch with slashes (`feature/auth`) → matches by branch name
    - State: worktree on branch `feature/auth`

### Worktree Matching

- [x] Match by directory name (first priority) → returns path
    - State: directory `feature-auth` exists
- [x] Match by branch name (fallback) → returns path
    - State: branch `feature/auth`, directory `feature-auth`
- [x] Directory name takes precedence over branch → uses dir match
    - State: dir `develop` with branch `main`, dir `main` with branch `develop`
- [x] No match found → error: "worktree not found: <name>"
    - State: neither dir nor branch matches
- [x] Case-sensitive matching → `Main` doesn't match `main`
    - State: worktree `main` exists
- [x] Switch to current worktree → succeeds, outputs same path
    - State: CWD in `main`, `grove switch main`
- [x] Detached HEAD worktree → matches by directory name
    - State: worktree in detached state

### Workspace Detection

- [x] Not in workspace → error: "not in a grove workspace"
    - State: CWD is `/tmp` or regular git repo
- [x] CWD is workspace root → works
    - State: CWD where `.bare` exists
- [x] CWD inside worktree → works
    - State: CWD is `/workspace/main`
- [x] CWD in worktree subdirectory → finds workspace
    - State: CWD is `/workspace/main/src/pkg`
- [~] Symlink loop in path → error about max depth
    - State: circular symlinks

### Output

- [x] Success → prints absolute path to stdout, no stderr
    - State: worktree found
- [x] Path with spaces → correctly output
    - State: worktree path has spaces
- [x] Exit code 0 on success
    - State: worktree found
- [x] Exit code non-zero on failure
    - State: worktree not found

### Worktree State (doesn't affect switch)

- [x] Locked worktree → switch succeeds
    - State: target is locked
- [x] Dirty worktree → switch succeeds
    - State: target has uncommitted changes
- [x] Merge conflict in target → switch succeeds
    - State: target has unresolved conflicts

### Shell Completion

- [x] Suggests all worktrees except current
    - State: CWD in `main`, suggests `develop`, `feature`
- [x] From worktree subdirectory → still excludes current
    - State: CWD is `/workspace/main/src`
- [x] Already has arg → no file completion
    - State: first arg provided
- [x] Not in workspace → error directive
    - State: outside workspace
- [~] git worktree list fails → error directive
    - State: corrupted `.bare`

### shell-init Subcommand

- [x] No arguments → detects shell from environment
    - State: `SHELL` env var set
- [x] `--shell bash` → outputs POSIX script
    - State: explicit flag
- [x] `--shell zsh` → outputs POSIX script
    - State: explicit flag
- [x] `--shell fish` → outputs fish script
    - State: explicit flag
- [x] `--shell powershell` → outputs PowerShell script
    - State: explicit flag
- [x] `--shell pwsh` → outputs PowerShell script (alias)
    - State: explicit flag
- [x] `--shell tcsh` → error: "unsupported shell"
    - State: unsupported shell
- [x] Arguments not allowed → error
    - State: `grove switch shell-init extra`

### Shell Detection

- [x] `SHELL=/bin/bash` → detects as `sh` (POSIX)
    - State: env var set
- [x] `SHELL=/bin/zsh` → detects as `sh`
    - State: env var set
- [x] `SHELL=/usr/bin/fish` → detects as `fish`
    - State: env var set
- [x] `SHELL=/bin/dash` → detects as `sh`
    - State: env var set
- [x] `PSModulePath` set, `SHELL` empty → detects as `powershell`
    - State: Windows environment
- [x] Neither set → defaults to `sh`
    - State: no shell indicators

### Shell Script Content

- [x] POSIX script contains `grove()` function
    - State: `--shell sh`
- [x] POSIX script uses `[ ]` not `[[ ]]` (portable)
    - State: `--shell sh`
- [x] POSIX script handles `grove switch` → calls `command grove switch`
    - State: `--shell sh`
- [x] POSIX script handles `grove add --switch` → detects `-s`/`--switch`
    - State: `--shell sh`
- [x] Fish script contains `function grove`
    - State: `--shell fish`
- [x] Fish script uses `set -l` for local vars
    - State: `--shell fish`
- [x] PowerShell script uses `grove.exe`
    - State: `--shell powershell`
- [x] PowerShell script uses `Set-Location`
    - State: `--shell powershell`

### Edge Cases

- [ ] Absolute path as input → no match (treats as name)
    - State: `grove switch /workspace/main`
- [ ] Relative path as input → no match
    - State: `grove switch ./main`
- [ ] Very long worktree name → works
    - State: no length limit
- [ ] Workspace with 100+ worktrees → linear search, finds target
    - State: many worktrees

---

## grove list

### Input Parsing

- [x] No arguments → lists all worktrees
    - State: valid workspace
- [x] Extra arguments → error (NoArgs validation)
    - State: `grove list foo`
- [x] `--fast` flag → skip sync status checks
    - State: valid workspace
- [x] `--json` flag → JSON output
    - State: valid workspace
- [x] `--verbose` / `-v` flag → show path, upstream, lock reason
    - State: valid workspace

### Filtering

- [x] `--filter dirty` → only dirty worktrees
    - State: mix of clean/dirty
- [x] `--filter locked` → only locked worktrees
    - State: mix of locked/unlocked
- [x] `--filter ahead` → only worktrees ahead of upstream
    - State: mix of sync states
- [x] `--filter behind` → only worktrees behind upstream
    - State: mix of sync states
- [x] `--filter gone` → only worktrees with deleted upstream
    - State: some upstreams deleted
- [x] `--filter dirty,locked` → OR logic, matches either
    - State: varied states
- [x] `--filter " dirty , locked "` → whitespace trimmed
    - State: valid workspace
- [x] `--filter DIRTY` → case-insensitive
    - State: dirty worktrees exist
- [x] `--filter invalid` → unknown filter, matches nothing
    - State: valid workspace
- [x] `--filter ""` → empty filter, returns all
    - State: valid workspace
- [x] Filter with no matches → empty output
    - State: `--filter dirty` on all-clean workspace

### Worktree States

- [x] Clean worktree → no dirty indicator
    - State: no uncommitted changes
- [x] Dirty worktree (modified files) → dirty indicator
    - State: tracked files modified
- [x] Dirty worktree (untracked files) → dirty indicator
    - State: new untracked files
- [x] Locked worktree → lock indicator
    - State: lock file exists
- [x] Locked with reason → reason shown in verbose
    - State: lock file has content
- [x] Detached HEAD → shows "(detached)" or short SHA
    - State: worktree detached
- [x] In sync with upstream → shows "=" indicator
    - State: ahead=0, behind=0
- [x] Ahead of upstream → shows "↑N" or "+N"
    - State: N commits ahead
- [x] Behind upstream → shows "↓N" or "-N"
    - State: N commits behind
- [x] Both ahead and behind → shows both indicators
    - State: diverged
- [x] Upstream gone → shows "×" or "gone"
    - State: upstream branch deleted
- [x] No upstream configured → no sync indicator
    - State: local branch, no tracking

### Fast Mode

- [x] `--fast` skips sync status → no ahead/behind/gone
    - State: valid workspace
- [x] `--fast` also skips dirty status (requires git status call)
    - State: dirty worktree
- [x] `--fast` still shows lock status
    - State: locked worktree
- [~] `--fast` with `--filter ahead` → no matches (sync not checked)
    - State: worktrees ahead

### Current Worktree

- [x] CWD in worktree → marked as current, sorted first
    - State: CWD is `/workspace/main`
- [x] CWD in worktree subdirectory → parent marked current
    - State: CWD is `/workspace/main/src/pkg`
- [x] CWD at workspace root → no current marker
    - State: CWD is `/workspace`
- [x] Alphabetical sort after current
    - State: multiple worktrees

### Table Output

- [~] Column alignment → padded to max widths
    - State: varying name/branch lengths
- [x] Current marker → `●` (color) or `*` (plain)
    - State: in a worktree
- [x] Dirty indicator → icon or `[dirty]`
    - State: dirty worktree
- [x] Lock indicator → icon or `[locked]`
    - State: locked worktree
- [x] Plain mode → ASCII indicators, no colors
    - State: `--plain` or config

### JSON Output

- [x] Valid JSON array → parseable
    - State: `--json` flag
- [x] All fields present → name, branch, path, current, etc.
    - State: normal worktree
- [x] Detached worktree → `detached: true`, no `branch` field
    - State: detached HEAD
- [x] Locked worktree → `locked: true`, `lock_reason` included
    - State: locked
- [x] Boolean fields omitted if false → `omitempty`
    - State: clean worktree
- [~] Empty workspace → `[]` (empty array)
    - State: no worktrees

### Verbose Output

- [x] Path sub-item → shows absolute path
    - State: `--verbose`
- [~] Upstream sub-item → shows remote/branch
    - State: `--verbose`, has upstream
- [x] Lock reason sub-item → shows reason text
    - State: `--verbose`, locked with reason
- [~] Sub-item prefix → `↳` (color) or `>` (plain)
    - State: `--verbose`

### Workspace Detection

- [x] Not in workspace → error
    - State: CWD is `/tmp`
- [x] From workspace root → works
    - State: CWD where `.bare` exists
- [x] From worktree subdirectory → works
    - State: CWD is nested in worktree

### Error Handling

- [~] git worktree list fails → error propagated (requires corrupting `.bare`)
    - State: corrupted `.bare`
- [x] Corrupted worktree → skipped with warning, others shown
    - State: one worktree invalid
- [~] GetSyncStatus fails → logged, continues without sync (requires git command failure)
    - State: git command fails

### Edge Cases

- [~] Zero worktrees → empty output (can't remove main worktree)
    - State: `.bare` exists, no worktrees
- [~] Many worktrees (100+) → all listed, acceptable performance (slow, implicit coverage)
    - State: large workspace
- [x] Very long branch name → displayed correctly
    - State: 200+ char branch
- [x] Branch with slashes → shows actual branch, dir is sanitized
    - State: branch `feature/auth`, dir `feature-auth`

### Shell Completion

- [x] `--filter` completion → suggests dirty, ahead, behind, gone, locked
    - State: tab after `--filter`
- [~] Completion excludes selected → no duplicates (complex multi-value edge case)
    - State: `--filter dirty,` + tab

---

## grove prune

### Input Parsing

- [x] No arguments → dry-run mode (no deletion)
    - State: valid workspace
- [x] `--commit` → actually removes worktrees
    - State: valid workspace
- [x] `--force` / `-f` → override dirty/locked/unpushed protections
    - State: valid workspace
- [x] `--stale 30d` → include worktrees older than 30 days
    - State: valid workspace
- [~] `--stale` (no value) → use config default (deferred: requires config file setup)
    - State: config has `grove.staleThreshold`
- [x] `--merged` → include merged worktrees
    - State: valid workspace
- [x] Combined flags → all work together
    - State: `--commit --force --stale 7d --merged`

### Duration Parsing

- [x] Valid days: `30d` → 30 days
    - State: any
- [x] Valid weeks: `2w` → 14 days
    - State: any
- [x] Valid months: `6m` → 180 days
    - State: any
- [x] Case insensitive: `30D`, `2W`, `6M` → works
    - State: any
- [~] Empty string → error: "duration cannot be empty" (deferred: edge case, unlikely user path)
    - State: any
- [x] Invalid unit: `30x` → error: "unknown duration unit"
    - State: any
- [x] Negative: `-5d` → error: "duration must be positive"
    - State: any
- [x] Zero: `0d` → error: "duration must be positive"
    - State: any

### Gone Branches (Primary Feature)

- [x] Upstream deleted → identified as gone candidate
    - State: worktree with `[gone]` upstream
- [x] Upstream exists → not gone
    - State: active upstream
- [x] No upstream configured → not gone
    - State: local branch, no tracking
- [~] Detached HEAD → not gone (no upstream) (implicit - detached has no upstream)
    - State: worktree detached
- [x] Multiple gone worktrees → all listed
    - State: 3+ worktrees with deleted upstreams

### Stale Worktrees

- [x] Over threshold → stale candidate
    - State: last commit 35 days ago, `--stale 30d`
- [x] Within threshold → not stale
    - State: last commit 25 days ago, `--stale 30d`
- [~] Exactly at threshold → not stale (deferred: requires precise timestamp, boundary edge case)
    - State: last commit exactly 30 days ago
- [~] No commits (LastCommitTime=0) → not stale (deferred: rare state)
    - State: empty worktree
- [x] `--stale` not provided → no stale detection
    - State: old worktrees exist

### Merged Branches

- [x] Regular merge (ancestry) → merged candidate
    - State: branch merged with `git merge`
- [x] Squash merge (patch-id) → merged candidate
    - State: branch squash-merged
- [x] Not merged → not candidate
    - State: unique commits not in default
- [~] Default branch itself → not checked (deferred: edge case, main worktree on main)
    - State: worktree on `main`, default is `main`
- [~] Can't determine default branch → warning, merged disabled (deferred: requires corrupted HEAD)
    - State: corrupted HEAD

### Candidate Priority (No Double-Counting)

- [x] Gone AND stale → counted as gone only
    - State: worktree matches both
- [~] Gone AND merged → counted as gone only (deferred: complex setup)
    - State: worktree matches both
- [x] Merged AND stale → counted as merged only
    - State: worktree matches both

### Skip Reasons

- [x] Current worktree → always skipped, even with `--force`
    - State: CWD in candidate worktree
- [x] Current from subdirectory → still protected
    - State: CWD is `worktree/src/pkg`
- [x] Dirty without `--force` → skipDirty
    - State: uncommitted changes
- [x] Dirty with `--force` → not skipped
    - State: `--force` flag
- [x] Locked without `--force` → skipLocked
    - State: worktree locked
- [~] Locked with `--force` → git needs --force --force, unlock first required
    - State: `--force` flag
- [~] Unpushed without `--force` → skipUnpushed (only works if upstream exists)
    - State: Ahead > 0
- [~] Unpushed with `--force` → not skipped (only works if upstream exists)
    - State: `--force` flag

### Skip Reason Priority

- [x] Current + dirty → shows "current worktree"
    - State: both conditions
- [~] Dirty + locked + unpushed → shows "dirty" (first check) (complex setup)
    - State: all three, not current
- [~] Locked + unpushed → shows "locked" (unpushed only works with active upstream)
    - State: both, not dirty

### Dry-Run Output

- [x] No candidates → "No worktrees to prune."
    - State: all healthy
- [x] Single prunable → "Would prune 1 worktree:"
    - State: 1 clean gone worktree
- [x] Multiple prunable → "Would prune N worktrees:"
    - State: 3 clean gone worktrees
- [x] Single skipped → "Would skip 1 worktree:" + reason
    - State: 1 dirty gone worktree
- [x] Mixed → both sections + hint about `--force`
    - State: some clean, some dirty
- [x] Stale shows age → "branch-name (2 months ago)"
    - State: stale worktree

### Commit Mode

- [x] No candidates → "No worktrees to remove."
    - State: nothing to prune
- [x] Single success → "Pruned 1 worktree:"
    - State: 1 clean candidate
- [x] Multiple success → "Pruned N worktrees:"
    - State: 3 clean candidates
- [~] Removal fails → "Failed to remove 1 worktree:" + error (hard to simulate)
    - State: git worktree remove fails
- [x] Mixed results → pruned + skipped sections
    - State: varied outcomes

### Git Operations

- [x] Fetch success → updates remote refs
    - State: valid remote
- [~] Fetch fails → warning, continues with existing state (requires network failure)
    - State: network error
- [~] git worktree list fails → error propagated (requires corrupted .bare)
    - State: corrupted `.bare`

### Edge Cases

- [x] Empty workspace → "No worktrees to prune."
    - State: `.bare` exists, no worktrees
- [~] All worktrees protected → all skipped (covered by individual skip tests)
    - State: all current/dirty/locked/unpushed
- [~] Age formatting boundaries (deferred: requires commits at exact day boundaries)
    - State: "today", "yesterday", "N days", "N weeks", "N months", "N years"
- [x] Repeated prune (idempotent) → second run shows nothing
    - State: prune again after `--commit`

---

## grove clone

### Argument Parsing

- [x] No arguments → error: "accepts between 1 and 2 arg(s)"
    - State: any
- [x] Too many arguments (3+) → error: "accepts between 1 and 2 arg(s)"
    - State: any
- [~] Valid HTTPS URL only → clone to current directory (requires network)
    - State: cwd empty
- [~] Valid SSH URL → clone to `.bare` as bare repo (requires network/SSH)
    - State: cwd empty
- [x] File protocol URL → clone local repo
    - State: local repo exists at path
- [x] Invalid URL (not a URL format) → git error propagated
    - State: any

### Directory Argument

- [x] URL only (no dir arg) → uses current working directory
    - State: `os.Getwd()` succeeds
- [x] URL + relative path (`./myrepo`) → converted to absolute
    - State: any
- [x] URL + absolute path (`/tmp/myrepo`) → used as-is
    - State: parent directory exists
- [x] Path with spaces → handled correctly
    - State: valid directory name
- [x] Non-existent parent directories → created recursively
    - State: `./nonexistent/parent/repo`

### Flag Validation

- [x] `--branches` without URL → error: "requires a repository URL"
    - State: `grove clone --branches main`
- [x] `--branches` with empty string → error: "no branches specified"
    - State: `--branches ""`
- [x] `--branches` with quoted empty string → error: "no branches specified"
    - State: `--branches '""'`
- [x] `--branches main` (single) → creates one worktree
    - State: branch exists
- [x] `--branches main,develop,feature` → creates multiple worktrees
    - State: all branches exist
- [x] `--branches " main , develop "` → whitespace trimmed
    - State: branches exist
- [x] `--branches "main,,develop"` → empty segments filtered
    - State: branches exist
- [x] `--branches ",main,develop,"` → leading/trailing commas handled
    - State: branches exist
- [x] `--verbose` / `-v` → shows git progress output
    - State: any
- [~] `--shallow` → adds `--depth 1` to clone (feature not implemented)
    - State: any
- [~] Combined flags (`-v --shallow --branches main`) → all apply (depends on --shallow)
    - State: any

### Directory Validation

- [x] Target directory is empty → succeeds
    - State: directory exists, empty
- [x] Target directory doesn't exist → created
    - State: path doesn't exist
- [x] Target directory not empty → error: "directory is not empty"
    - State: contains files
- [x] Target inside git repository → error: "cannot initialize inside existing git repository"
    - State: parent or self has `.git`
- [x] Target inside grove workspace → error: "cannot initialize inside existing grove workspace"
    - State: parent or self has `.bare`
- [x] Nested inside grove workspace (deep) → error
    - State: ancestor directory has `.bare`
- [~] Read-only parent directory → error: "failed to create directory" (platform-specific)
    - State: no write permission

### Git Operations

- [x] Clone creates `.bare` directory → bare git repo
    - State: clone succeeds
- [x] Clone creates `.git` file → contains `gitdir: .bare`
    - State: clone succeeds
- [ ] `.git` file permissions → 0o644
    - State: clone succeeds
- [~] Shallow clone (`--shallow`) → only latest commit (feature not implemented)
    - State: valid repo
- [x] Quiet mode (default) → git progress suppressed
    - State: no `-v` flag
- [x] Verbose mode → git progress on stderr
    - State: `-v` flag
- [x] Network error during clone → error with git stderr
    - State: unreachable URL
- [x] Repository not found (404) → error: "failed to clone repository"
    - State: non-existent repo URL

### Branch Handling

- [~] No `--branches` → uses default branch from remote (requires real remote)
    - State: remote has default branch
- [x] Branch doesn't exist → error: "failed to create worktree for branch 'X': invalid reference"
    - State: non-existent branch specified
- [x] Mix of valid/invalid branches → fails, cleans up created worktrees
    - State: `--branches main,nonexistent`
- [x] Branch with slashes (`feat/user-auth`) → directory sanitized to `feat-user-auth`
    - State: branch exists
- [x] Branch with multiple slashes → each replaced with dash
    - State: `release/v1/patch` → `release-v1-patch`
- [~] Branch with Windows-unsafe chars (`<>|"?*:`) → replaced with dash (git doesn't allow these chars)
    - State: branch with special chars
- [x] Duplicate branches (`main,main`) → error on second worktree creation
    - State: same branch twice

### Worktree Creation

- [x] Single branch → creates `workspace/main/`
    - State: `--branches main`
- [x] Multiple branches → creates multiple worktree directories
    - State: `--branches main,develop`
- [~] Default branch auto-detected → worktree created for it (requires real remote)
    - State: no `--branches` specified
- [~] Empty repository (no commits) → error: can't determine default branch (edge case)
    - State: repo has no commits

### Auto-Lock

- [x] Branch matches auto-lock pattern → worktree locked
    - State: `grove.autoLock` config matches branch
- [~] Auto-lock fails → warning logged, worktree still created (hard to simulate lock failure)
    - State: lock operation errors

### PR URL Detection

- [ ] `https://github.com/owner/repo/pull/123` → detected as PR URL (unit test: URL parsing)
    - State: any
- [ ] `http://github.com/owner/repo/pull/123` → detected as PR URL (HTTP) (unit test: URL parsing)
    - State: any
- [ ] `https://github.com/owner/repo/pull/123/` → trailing slash handled (unit test: URL parsing)
    - State: any
- [ ] `https://github.com/owner/repo` → NOT a PR URL, regular clone (unit test: URL parsing)
    - State: any
- [ ] `https://github.com/owner/repo/issues/123` → NOT a PR URL (unit test: URL parsing)
    - State: any
- [x] `#123` → NOT detected by clone (only `grove add` supports)
    - State: workspace context

### gh CLI Requirements

- [~] gh not installed → error: "gh CLI not found" (hard to simulate missing gh in test env)
    - State: `gh` not in PATH
- [x] gh not authenticated → error: "gh not authenticated" (tested in add_pr_integration.txt)
    - State: `gh auth status` fails
- [ ] gh installed and authenticated → proceeds (clone_pr_integration.txt)
    - State: `gh auth status` succeeds

### PR Clone (Same-Repo PR)

- [ ] Parse PR URL → extract owner, repo, number (clone_pr_integration.txt)
    - State: valid PR URL
- [ ] Clone via `gh repo clone` → respects user's protocol preference (clone_pr_integration.txt)
    - State: gh configured
- [ ] Fetch PR info via `gh pr view --json` → gets branch name (clone_pr_integration.txt)
    - State: PR exists
- [ ] Fetch branch from origin → `git fetch origin <branch>` (clone_pr_integration.txt)
    - State: branch in same repo
- [ ] Create worktree `pr-123/` → tracking origin branch (clone_pr_integration.txt)
    - State: fetch succeeds

### PR Clone (Fork PR)

- [~] Detect fork (`HeadOwner != BaseOwner`) → fork handling (needs fork PR for testing)
    - State: PR from fork
- [~] Get fork clone URL → `gh repo view contributor/repo --json url` (needs fork PR for testing)
    - State: fork accessible
- [~] Add remote `pr-123-contributor` → fork remote added (needs fork PR for testing)
    - State: URL retrieved
- [~] Fetch from fork remote → `git fetch pr-123-contributor <branch>` (needs fork PR for testing)
    - State: remote added
- [~] Create worktree tracking fork branch → `pr-123-contributor/<branch>` (needs fork PR for testing)
    - State: fetch succeeds
- [~] Fork URL retrieval fails → cleanup, error (needs fork PR for testing)
    - State: fork not accessible
- [~] Adding fork remote fails → cleanup, error (needs fork PR for testing)
    - State: git remote add fails
- [~] Fetching fork branch fails → cleanup, error (needs fork PR for testing)
    - State: branch doesn't exist

### PR Clone Errors

- [ ] PR doesn't exist → error: "PR #N not found in owner/repo" (clone_pr_integration.txt)
    - State: invalid PR number
- [~] Invalid JSON from gh → error: "failed to parse gh output" (hard to simulate malformed response)
    - State: gh returns malformed JSON
- [~] Missing headRefName in response → error: "missing headRefName" (hard to simulate incomplete response)
    - State: incomplete gh output

### Cleanup on Failure

- [x] Clone fails → nothing created
    - State: git clone fails
- [~] Clone succeeds, .git write fails → `.bare` removed (hard to simulate permission issues)
    - State: permission issue
- [x] First worktree fails → `.bare`, `.git` removed
    - State: git worktree add fails
- [x] Second worktree fails → all created worktrees, `.bare`, `.git` removed
    - State: partial success
- [ ] PR clone, fetch fails → `.bare`, `.git` removed (clone_pr_integration.txt)
    - State: network error after clone

### Output

- [x] Success → "Cloned repository to <path>"
    - State: clone completes
- [x] Worktrees listed → shows each created worktree
    - State: multiple branches
- [x] Quiet mode → "Repository cloned", "Creating worktrees:", "✓ main"
    - State: no `-v`
- [x] Verbose mode → git stderr visible
    - State: `-v` flag
- [x] Error messages include git stderr
    - State: git command fails

### Shell Completion

- [~] First arg (URL) → no file completion (hard to test shell completion behavior)
    - State: `grove clone <TAB>`
- [~] Second arg (directory) → directory completion only (hard to test shell completion behavior)
    - State: `grove clone https://... <TAB>`

### Edge Cases

- [x] Clone same repo twice (different dirs) → both succeed
    - State: independent directories
- [x] Clone same repo twice (same dir) → second fails: "grove workspace exists"
    - State: first clone completed
- [~] Very long branch name → may exceed filesystem path limits (platform-specific limits)
    - State: 255+ char branch
- [~] Large repository → takes time, spinner shown (or git progress with `-v`) (performance test)
    - State: big repo
- [~] Shallow clone with multiple branches → all worktrees from shallow (feature not implemented)
    - State: `--shallow --branches a,b,c`
- [ ] Unicode in branch name → passes through sanitization
    - State: branch with non-ASCII chars

---

## grove init

### Command Structure

- [x] `grove init` (no subcommand) → shows help with subcommands
    - State: any
- [x] `grove init --help` → shows "new" and "convert" subcommands
    - State: any
- [x] `grove init invalid` → error: "unknown command"
    - State: any

---

## grove init new

### Argument Parsing

- [~] No arguments → uses current working directory (deferred: test env complexity)
    - State: cwd accessible
- [~] Relative path (`./myproject`) → converted to absolute (deferred: implicitly tested via nested path)
    - State: any
- [~] Absolute path (`/tmp/myproject`) → used as-is (deferred: implicitly tested)
    - State: any
- [x] Too many arguments (`dir1 dir2`) → error: "accepts at most 1 arg"
    - State: any
- [x] Nested path (`parent/child/workspace`) → creates parent directories
    - State: parent exists or will be created
- [~] Current dir (`.`) → resolves correctly (deferred: test env complexity)
    - State: cwd accessible

### Directory Validation

- [x] Directory doesn't exist → created
    - State: path doesn't exist
- [~] Empty directory exists → succeeds (deferred: trivial case implicitly covered)
    - State: directory exists, empty
- [x] Non-empty directory → error: "directory is not empty"
    - State: directory has files
- [~] Inside existing git repository → error: "cannot initialize inside existing git repository" (deferred: clone tests cover similar validation)
    - State: parent has `.git`
- [x] Inside existing grove workspace → error: "cannot initialize inside existing grove workspace"
    - State: parent has `.bare`
- [x] Nested inside grove workspace (deep) → error
    - State: ancestor has `.bare`
- [~] Permission denied on parent → error: "failed to create directory" (deferred: requires permission manipulation)
    - State: no write permission

### Workspace Structure

- [x] Creates `.bare` directory → bare git repo
    - State: init succeeds
- [x] Creates `.git` file → contains `gitdir: .bare`
    - State: init succeeds
- [x] `.git` file permissions → 0o644
    - State: init succeeds
- [x] No worktrees created → empty workspace
    - State: `grove init new` (not convert)

### Error Recovery

- [~] `.bare` creation fails → error, nothing left behind (deferred: requires permission manipulation)
    - State: permission denied
- [~] `git init --bare` fails → `.bare` removed (deferred: hard to simulate git failure)
    - State: git error
- [~] `.git` file write fails → `.bare` removed (deferred: requires permission manipulation)
    - State: permission denied

### Output

- [x] Success → "Initialized grove workspace in: <path>"
    - State: init completes

---

## grove init convert

### Argument Parsing

- [x] No positional args → converts current directory
    - State: cwd is git repo
- [x] Positional arg provided → error: "unknown command"
    - State: `grove init convert somedir`
- [x] `--branches main,develop` → creates multiple worktrees
    - State: branches exist
- [~] `--branches` with spaces (`" main , develop "`) → whitespace trimmed (deferred: edge case)
    - State: branches exist
- [~] `--branches` with empty segments (`main,,develop`) → filtered (deferred: edge case)
    - State: branches exist
- [~] `--branches` with trailing comma → handled (deferred: edge case)
    - State: branches exist
- [~] `--verbose` / `-v` → shows git output (deferred: visual verification needed)
    - State: any

### Pre-Conversion Validation (Order Matters)

- [x] Already grove workspace → error: "already a grove workspace"
    - State: `.bare` exists
- [x] Not a git repository → error: "not a git repository"
    - State: no `.git`
- [x] Repository is a worktree → error: "repository is already a worktree"
    - State: created via `git worktree add`
- [x] Active lock files (`.git/index.lock`) → error: "has active lock files"
    - State: lock file exists
- [x] Has submodules → error: "has submodules"
    - State: `.gitmodules` exists
- [x] Unresolved merge conflicts → error: "has unresolved conflicts"
    - State: `MERGE_HEAD` exists with conflicts
- [x] Staged uncommitted changes → error: "has uncommitted changes"
    - State: `git add` without commit
- [x] Untracked files → error: "has uncommitted changes"
    - State: new file not staged
- [x] Modified unstaged files → error: "has uncommitted changes"
    - State: tracked file modified
- [x] Unpushed commits → error: "has unpushed commits"
    - State: local commits ahead of remote
- [x] Detached HEAD → error: "in detached HEAD state"
    - State: `git checkout HEAD~0`
- [x] Unborn HEAD (no commits) → error: "has no commits (unborn HEAD)"
    - State: `git init` without commits
- [x] Ongoing merge/rebase/cherry-pick → error: "has ongoing merge/rebase/cherry-pick"
    - State: operation in progress
- [x] Existing worktrees → error: "has existing worktrees"
    - State: `git worktree add` already run

### Branch Validation

- [x] `--branches nonexistent` → error: "branch does not exist"
    - State: branch not found
- [x] Branch validation runs BEFORE touching `.git` → no side effects
    - State: invalid branch specified
- [x] All branches validated before conversion starts
    - State: mixed valid/invalid branches

### Conversion Process

- [x] Lock file created (`.grove-convert.lock`) → prevents concurrent conversion
    - State: conversion in progress
- [x] `.git` moved to `.bare` → in-place rename
    - State: conversion starts
- [x] `.bare` configured as bare (`core.bare=true`)
    - State: after move
- [x] Current branch worktree created first
    - State: `--branches` specified
- [x] Files moved from root to first worktree
    - State: conversion succeeds
- [x] `.git` file created pointing to `.bare`
    - State: conversion completes
- [x] Lock file removed on success
    - State: conversion completes

### Worktree Creation

- [x] No `--branches` → creates worktree for current branch only
    - State: default mode
- [x] `--branches main,develop` → creates both worktrees
    - State: both branches exist
- [~] Current branch always first in worktree order (deferred: implicit via file move logic)
    - State: `--branches develop,main` from main
- [x] Branch with slashes sanitized → `feat/auth` → `feat-auth`
    - State: branch with `/`
- [~] Auto-lock applied if configured (deferred: tested in grove add/clone)
    - State: `grove.autoLock` matches branch

### File Preservation

- [x] `.env` preserved in all worktrees (default pattern)
    - State: `.env` exists, git-ignored
- [x] `.env.local` preserved → matches `*.env.*` pattern
    - State: file exists
- [x] Custom patterns from git config (`grove.preserve`)
    - State: config set
- [x] Custom patterns from `.grove.toml`
    - State: TOML has `[preserve]` section
- [x] Files copied to ALL worktrees, not just first
    - State: multiple branches
- [~] Missing source file for preserve → error (deferred: preserve patterns match existing files)
    - State: file in pattern list but doesn't exist

### Directory Structure After Conversion

- [x] Root only contains `.bare`, `.git`, worktree dirs
    - State: conversion complete
- [x] Nested directories preserved in worktree
    - State: `src/components/` structure
- [x] Deep nesting preserved → `deep/nested/structure/file.txt`
    - State: 4+ level nesting
- [x] Symlinks preserved with correct targets
    - State: symlink in repo

### Rollback on Failure

- [x] Worktree creation fails → full rollback triggered
    - State: blocking directory exists
- [x] Files moved back to root from worktree
    - State: rollback in progress
- [x] `.bare` renamed back to `.git`
    - State: rollback in progress
- [x] `core.bare` set back to false
    - State: rollback completes
- [x] Created worktrees removed
    - State: partial worktrees existed
- [x] Lock file removed even on failure
    - State: conversion fails
- [~] Recovery file written (`.grove-recovery.txt`) if rollback incomplete (deferred: requires cascading failures)
    - State: critical failure

### Rollback Edge Cases

- [x] Multiple files restored in reverse order
    - State: many files moved
- [x] Directory hierarchy restored
    - State: nested dirs moved
- [x] Hidden files restored (`.hidden-file`)
    - State: hidden files moved
- [~] Executable files preserve permissions (deferred: platform-specific, git handles this)
    - State: `script.sh` with +x
- [x] Symlink targets preserved during restore
    - State: symlink moved then restored
- [x] Multiple worktrees cleaned up on failure
    - State: 2+ worktrees created before failure

### Shell Completion

- [x] Empty input → all branches listed
    - State: tab completion
- [x] Partial match → filters branches
    - State: `fea` → `feature`, `feature-2`
- [x] Second branch partial (`main,fea`) → appends to existing
    - State: already selected `main`
- [x] Trailing comma → shows all non-selected
    - State: `main,` shows others
- [x] No duplicates in suggestions
    - State: `main,mai` doesn't suggest `main,main`
- [x] Whitespace handled → `main, fea` trims space
    - State: space after comma

### Output

- [x] Success → "Converted repository to grove workspace in: <path>"
    - State: conversion completes
- [x] Worktrees listed → "✓ main", "✓ develop"
    - State: multiple branches
- [x] Preserved files count shown
    - State: files preserved

### Edge Cases

- [~] Concurrent conversion blocked → error: "conversion already in progress" (deferred: requires race condition simulation)
    - State: lock file exists
- [~] Very deep nesting (50+ levels) → still works (deferred: filesystem limit stress test)
    - State: deeply nested files
- [~] Empty repository (no files except .git) → succeeds (deferred: uncommon scenario)
    - State: only commits, no tracked files
- [x] Remote-only branch in `--branches` → fetched during validation
    - State: branch only on origin

---

## grove move

### Argument Parsing

- [x] No arguments → error: "accepts 2 arg(s)"
    - State: any
- [x] One argument → error: "accepts 2 arg(s)"
    - State: any
- [x] Two arguments → accepted
    - State: valid workspace
- [x] Three+ arguments → error: "accepts 2 arg(s)"
    - State: any
- [~] Whitespace around arguments → trimmed before processing (deferred: implicit, code trims)
    - State: `"  old  "` and `"  new  "`

### Worktree Lookup

- [x] Match by directory name (basename) → finds worktree
    - State: worktree `feat-auth` exists
- [x] Match by branch name (fallback) → finds worktree
    - State: branch `feature/auth`, directory `feat-auth`
- [x] Directory name takes precedence over branch name
    - State: ambiguous match possible
- [x] Worktree not found → error: "worktree not found: <name>"
    - State: no matching worktree
- [~] Detached HEAD worktree → error (no branch to rename) (deferred: edge case)
    - State: worktree in detached state

### Validation - Same Branch

- [x] Target already has new branch name → error: "worktree already has branch"
    - State: `grove move feat feat`

### Current Worktree Protection

- [x] CWD is target worktree root → error: "cannot rename current worktree"
    - State: CWD is `/workspace/feat`
- [x] CWD is subdirectory of target → error: "cannot rename current worktree"
    - State: CWD is `/workspace/feat/src/pkg`
- [x] CWD is different worktree → allowed
    - State: CWD is `/workspace/main`, moving `feat`
- [x] CWD is workspace root → allowed
    - State: CWD is `/workspace`

### Branch Name Validation

- [x] New branch already exists locally → error: "branch already exists"
    - State: `new-branch` exists
- [x] New branch exists on remote → error: "branch already exists"
    - State: `origin/new-branch` exists
- [~] Invalid git branch name → git error propagated (deferred: git handles)
    - State: branch with `..`, `^`, `~`

### Dirty State Check

- [x] Uncommitted tracked changes → error: "has uncommitted changes"
    - State: modified tracked file
- [x] Staged but not committed → error: "has uncommitted changes"
    - State: `git add` without commit
- [x] Untracked files → error: "has uncommitted changes" (consistent with remove)
    - State: new untracked file
- [x] Clean worktree → allowed
    - State: no changes

### Locked Worktree Check

- [x] Worktree is locked → error: "worktree is locked"
    - State: lock file exists
- [x] Unlocked worktree → allowed
    - State: no lock file

### Directory Validation

- [x] New directory already exists → error: "directory already exists"
    - State: conflicting directory at destination
- [x] Sanitized name causes collision → error
    - State: `feat/new` → `feat-new` which exists

### Branch Sanitization

- [x] Branch with slashes → sanitized to dashes
    - State: `feat/auth` → directory `feat-auth`
- [x] Multiple slashes → each replaced
    - State: `release/v1/patch` → `release-v1-patch`
- [~] Windows-unsafe chars → replaced with dashes (deferred: git doesn't allow these in branch names)
    - State: `<>|"?*:` in branch name

### Git Operations

- [x] Branch rename succeeds → `git branch -m old new`
    - State: valid preconditions
- [~] Branch rename fails → error with git stderr (deferred: hard to simulate)
    - State: git error
- [x] Directory move succeeds → `os.Rename(old, new)`
    - State: valid paths
- [~] Directory move fails → error, triggers rollback (deferred: permission issues)
    - State: permission denied
- [x] Worktree repair called → `git worktree repair`
    - State: after directory move

### Rollback on Failure

- [~] Branch renamed, dir move fails → branch renamed back (deferred: hard to simulate)
    - State: step 2 fails
- [~] Both succeed, repair fails → error returned, partial state (deferred: hard to simulate)
    - State: step 3 fails
- [~] Rollback itself fails → error logged, continues (deferred: cascading failures)
    - State: cascading failures
- [~] Repair called on rollback → restores git registry (deferred: implicit in rollback)
    - State: during rollback

### Upstream Tracking

- [~] Upstream configured, new remote branch exists → upstream updated (deferred: requires push)
    - State: `origin/old` → `origin/new`
- [~] Upstream configured, new remote branch missing → warning, no update (deferred: complex setup)
    - State: remote branch doesn't exist
- [~] No upstream configured → no update attempted (deferred: implicit, no-op)
    - State: local-only branch
- [~] Non-origin remote → extracts remote name correctly (deferred: requires non-origin remote)
    - State: `upstream/old` → `upstream/new`
- [~] Malformed upstream (no slash) → skip update gracefully (deferred: edge case)
    - State: edge case
- [~] Upstream update fails → warning only, command succeeds (deferred: hard to simulate)
    - State: git error on set

### Workspace Locking

- [x] Lock acquired successfully → proceeds
    - State: no concurrent operation
- [~] Lock already held → error: "another operation in progress" (deferred: race condition)
    - State: concurrent grove command
- [x] Lock cleaned up on success → file removed
    - State: operation completes
- [~] Lock cleaned up on failure → file removed (deferred: implicit in rollback tests)
    - State: operation fails

### Output

- [x] Branch != directory name → "Renamed X to Y (dir: Z)"
    - State: sanitization applied
- [x] Branch == directory name → "Renamed X to Y"
    - State: no sanitization needed

### Shell Completion

- [~] First argument → suggests worktree names (deferred: unit test territory)
    - State: tab completion
- [~] Excludes current worktree from suggestions (deferred: unit test territory)
    - State: CWD in a worktree
- [x] Second argument → no file completion
    - State: first arg provided

### Edge Cases

- [~] Very long branch name → filesystem limits (deferred: platform-specific)
    - State: 255+ char branch
- [~] Branch with special chars (`#`, `@`) → sanitized (deferred: git may not allow)
    - State: branch with special chars
- [x] Rapid consecutive moves (A→B, B→C) → both succeed
    - State: sequential operations

---

## grove lock

### Argument Parsing

- [x] No arguments → error: "requires 1 arg"
    - State: any
- [x] Two+ arguments → error: "requires 1 arg"
    - State: any
- [~] Whitespace around argument → trimmed (deferred: unlikely user path)
    - State: `"  feature  "`
- [x] `--reason` flag → stores reason with lock
    - State: valid worktree
- [x] `--reason` with empty string → lock without reason
    - State: `--reason ""`
- [x] No shorthand for `--reason` → follows git conventions
    - State: any

### Workspace Detection

- [x] Not in workspace → error: "not in a grove workspace"
    - State: CWD outside workspace
- [x] From workspace root → works
    - State: CWD at workspace root
- [x] From worktree subdirectory → works
    - State: CWD is `/workspace/main/src`

### Worktree Lookup

- [x] Match by directory name → finds worktree
    - State: `grove lock feat-auth`
- [x] Match by branch name (fallback) → finds worktree
    - State: `grove lock feature/auth`
- [~] Directory name takes precedence → uses dir match (deferred: requires ambiguous setup, low priority)
    - State: ambiguous match possible
- [x] Worktree not found → error: "worktree not found: <name>"
    - State: no matching worktree

### Lock State Validation

- [x] Already locked (with reason) → error shows existing reason
    - State: worktree locked with "WIP"
- [x] Already locked (no reason) → error: "already locked"
    - State: worktree locked without reason
- [x] Unlocked worktree → succeeds
    - State: worktree not locked

### Git Operations

- [x] Lock without reason → `git worktree lock <path>`
    - State: unlocked worktree
- [x] Lock with reason → `git worktree lock --reason "..." <path>`
    - State: unlocked worktree
- [~] Git command fails → error propagated (deferred: hard to simulate git failure)
    - State: git error

### Output

- [x] Success without reason → "Locked worktree <name>"
    - State: lock completes
- [x] Success with reason → "Locked worktree <name> (<reason>)"
    - State: reason provided

### Shell Completion

- [x] Suggests only unlocked worktrees
    - State: mix of locked/unlocked
- [x] Excludes locked worktrees from suggestions
    - State: some worktrees locked
- [x] Returns `ShellCompDirectiveNoFileComp`
    - State: any

### Integration with Other Commands

- [x] `grove remove` blocked by lock (no `--force`) → error
    - State: locked worktree
- [x] `grove remove --force` → auto-unlocks, removes
    - State: locked worktree
- [x] `grove move` blocked by lock → error suggests unlock
    - State: locked worktree
- [x] `grove prune` skips locked (no `--force`) → skip reason shown
    - State: locked candidate
- [x] `grove list` shows lock indicator
    - State: locked worktree

---

## grove unlock

### Argument Parsing

- [x] No arguments → error: "requires 1 arg"
    - State: any
- [x] Two+ arguments → error: "requires 1 arg"
    - State: any
- [~] Whitespace around argument → trimmed (deferred: unlikely user path)
    - State: `"  feature  "`

### Workspace Detection

- [x] Not in workspace → error: "not in a grove workspace"
    - State: CWD outside workspace
- [x] From workspace root → works
    - State: CWD at workspace root
- [x] From worktree subdirectory → works
    - State: CWD is `/workspace/main/src`

### Worktree Lookup

- [x] Match by directory name → finds worktree
    - State: `grove unlock feat-auth`
- [x] Match by branch name (fallback) → finds worktree
    - State: `grove unlock feature/auth`
- [~] Directory name takes precedence → uses dir match (deferred: requires ambiguous setup)
    - State: ambiguous match
- [x] Worktree not found → error: "worktree not found: <name>"
    - State: no matching worktree

### Lock State Validation

- [x] Worktree not locked → error: "worktree is not locked"
    - State: unlocked worktree
- [x] Locked worktree → succeeds
    - State: worktree is locked

### Git Operations

- [x] Unlock → `git worktree unlock <path>`
    - State: locked worktree
- [~] Git command fails → error propagated (deferred: hard to simulate git failure)
    - State: git error
- [x] Lock reason cleared after unlock
    - State: was locked with reason

### Output

- [x] Success → "Unlocked worktree <name>"
    - State: unlock completes

### Shell Completion

- [x] Suggests only locked worktrees
    - State: mix of locked/unlocked
- [x] Excludes unlocked worktrees from suggestions
    - State: some unlocked
- [x] Returns `ShellCompDirectiveNoFileComp`
    - State: any

### Lock File Mechanics

- [~] Lock file stored at `<gitdir>/locked` (deferred: implementation detail, covered by git)
    - State: worktree locked
- [~] Reason stored as file content (deferred: implementation detail, covered by git)
    - State: locked with reason
- [~] Empty reason → file exists but empty (deferred: implementation detail)
    - State: locked without reason
- [~] Detached HEAD worktree → can be locked/unlocked (deferred: requires detached worktree setup)
    - State: detached worktree

### Edge Cases

- [~] Renamed worktree (after repair) → lock still works (deferred: complex setup)
    - State: worktree moved and repaired
- [~] Corrupted lock file → graceful handling (deferred: hard to simulate)
    - State: malformed lock file
- [~] `IsWorktreeLocked` returns false on error → safe default (deferred: unit test territory)
    - State: any error condition
- [~] `GetWorktreeLockReason` returns empty on error → safe default (deferred: unit test territory)
    - State: any error condition

---

## grove exec

### Argument Parsing

- [x] No command after `--` → error: "no command specified"
    - State: `grove exec --all --`
- [~] Missing `--` delimiter → all args treated as command (deferred: edge case behavior)
    - State: `grove exec echo hello`
- [x] Complex command with pipes → correctly parsed
    - State: `grove exec --all -- bash -c "npm install && npm test"`
- [x] Quoted arguments preserved
    - State: `grove exec --all -- echo "hello world"`

### Target Selection

- [x] `--all` flag → executes in all worktrees
    - State: multiple worktrees exist
- [x] Specific worktrees → executes in named only
    - State: `grove exec main feature -- echo test`
- [x] `--all` + specific worktrees → error: "cannot use --all with specific worktrees"
    - State: conflicting flags
- [x] No `--all` and no worktrees → error: "must specify --all or at least one worktree"
    - State: missing target
- [x] Invalid worktree name → error: "worktree not found: <name>"
    - State: non-existent worktree specified
- [x] Execution order → alphabetical by branch name
    - State: multiple worktrees

### Command Execution

- [x] Command runs in each worktree directory
    - State: `cmd.Dir` set to worktree path
- [x] Stdout/stderr passed through directly
    - State: no buffering
- [x] Header printed before each execution → branch name
    - State: multiple worktrees
- [x] Blank line separator between worktrees
    - State: multiple worktrees
- [x] Exit code 0 from command → success
    - State: command succeeds
- [x] Non-zero exit code → failure recorded
    - State: command fails

### Failure Handling

- [x] Default: continue despite failures
    - State: first worktree fails, others still execute
- [x] `--fail-fast` → stop on first failure
    - State: only first worktree executes
- [x] All succeeded → "Executed in N worktrees"
    - State: all commands succeed
- [x] All failed → "All N executions failed"
    - State: all commands fail
- [x] Partial failure → "N succeeded, M failed"
    - State: mixed results

### Shell Completion

- [x] Before `--` → suggests worktree names
    - State: tab completion
- [x] Already-used worktrees excluded
    - State: `main` already in args
- [~] After `--` → default file completion (deferred: shell completion hard to test in integration)
    - State: command position

### Workspace Detection

- [x] Not in workspace → error
    - State: CWD outside workspace

### Edge Cases

- [~] Command not found (exit 127) → treated as failure (deferred: edge case, implicit in failure handling)
    - State: invalid command
- [~] Detached HEAD worktrees → executed normally (deferred: requires detached setup)
    - State: detached worktree
- [x] Locked worktrees → executed (locks don't block exec)
    - State: locked worktree
- [~] Very large output → no buffering issues (deferred: performance test)
    - State: command with lots of output

---

## grove status

### Argument Parsing

- [x] No arguments → shows current worktree status
    - State: inside worktree
- [x] Extra arguments → error: "unknown command"
    - State: `grove status foo`
- [x] `--verbose` / `-v` → extended output
    - State: any
- [x] `--json` → JSON output
    - State: any
- [x] `--json` + `--verbose` → JSON output (verbose ignored)
    - State: both flags

### Worktree Detection

- [x] In worktree root → works
    - State: CWD is worktree
- [x] In worktree subdirectory → works (finds root)
    - State: CWD is `worktree/src/pkg`
- [x] At workspace root → error: "not inside a worktree"
    - State: CWD has `.bare` but not `.git` file
- [x] Not in workspace → error: "not in a grove workspace"
    - State: CWD outside workspace

### Branch Information

- [x] Normal branch → shows branch name
    - State: on `main`
- [x] Feature branch → shows full name (with slashes)
    - State: on `feature/auth`
- [x] Detached HEAD → shows "(detached)"
    - State: checked out commit, not branch
- [~] Detached shown in verbose → "detached HEAD" line (deferred: implicit in detached test)
    - State: `--verbose` with detached

### Sync Status

- [x] No upstream → `no_upstream: true` in JSON
    - State: local branch only
- [~] In sync → shows `=` (deferred: requires bare repo setup for push)
    - State: same as remote
- [~] Ahead → shows `↑N` (deferred: requires bare repo setup)
    - State: local commits not pushed
- [~] Behind → shows `↓N` (deferred: requires bare repo setup)
    - State: remote commits not pulled
- [~] Diverged → shows `↑M↓N` (deferred: requires bare repo setup)
    - State: both ahead and behind
- [~] Upstream gone → shows `×` (deferred: requires remote branch deletion)
    - State: remote branch deleted

### Dirty State

- [x] Clean → no `[dirty]` indicator
    - State: no uncommitted changes
- [x] Untracked files → `[dirty]`
    - State: new file not staged
- [x] Modified tracked files → `[dirty]`
    - State: existing file changed
- [x] Staged changes → `[dirty]`
    - State: changes in index
- [~] Deleted files → `[dirty]` (deferred: similar to modified, implicit)
    - State: tracked file removed

### Lock Status

- [x] Unlocked → no lock indicator
    - State: worktree not locked
- [x] Locked → `[locked]` indicator
    - State: worktree locked
- [x] Lock reason shown in verbose
    - State: `--verbose` with locked

### Stash Detection

- [~] No stashes → no stash line (deferred: implicit when stash count is 0)
    - State: stash list empty
- [x] Stashes present → "stashes: N" in verbose
    - State: `--verbose` with stashes

### Ongoing Operations

- [~] Clean → no operation line (deferred: implicit in all clean tests)
    - State: normal state
- [~] Merge in progress → "operation: merging" (deferred: requires complex merge setup)
    - State: MERGE_HEAD exists
- [~] Rebase in progress → "operation: rebasing" (deferred: requires interactive rebase)
    - State: rebase-merge exists
- [~] Cherry-pick → "operation: cherry-picking" (deferred: requires cherry-pick conflict)
    - State: CHERRY_PICK_HEAD exists
- [~] Revert → "operation: reverting" (deferred: requires revert conflict)
    - State: REVERT_HEAD exists

### Conflict Detection

- [~] No conflicts → no conflicts line (deferred: implicit in clean tests)
    - State: clean merge
- [~] Conflicts present → "conflicts: N" in verbose (deferred: requires merge conflict)
    - State: unresolved merge conflicts
- [~] Files with spaces handled → correct count (deferred: edge case)
    - State: `my file.txt` in conflict

### Output Formats

- [x] Default → single line with indicators
    - State: no flags
- [x] Verbose → main line + sub-items
    - State: `--verbose`
- [x] JSON → all fields included
    - State: `--json`
- [x] Plain mode → ASCII fallbacks
    - State: `--plain`

### Error Handling

- [~] Stash check fails → logged, continues (deferred: hard to simulate)
    - State: non-fatal
- [~] Operation check fails → logged, continues (deferred: hard to simulate)
    - State: non-fatal
- [~] Conflict check fails → logged, continues (deferred: hard to simulate)
    - State: non-fatal
- [~] Branch/HEAD read fails → fatal error (deferred: requires corrupted repo)
    - State: corrupted repo

---

## grove config

### Subcommand: list

- [x] `grove config list` → shows merged config
    - State: inside workspace
- [x] `grove config list --shared` → shows `.grove.toml` only
    - State: inside worktree
- [x] `grove config list --global` → shows git config only
    - State: anywhere
- [x] `--shared` + `--global` → error: "cannot be used together"
    - State: conflicting flags
- [~] Empty config → no output (deferred: implicit when no keys set)
    - State: nothing configured

### Subcommand: get

- [x] `grove config get <key>` → shows merged value
    - State: key exists
- [x] `grove config get <key> --shared` → from `.grove.toml`
    - State: inside worktree
- [x] `grove config get <key> --global` → from git config
    - State: anywhere
- [x] Key not found → error: "config key not found"
    - State: key doesn't exist
- [x] Non-grove._ key → error: "only grove._ settings supported"
    - State: `grove config get user.name`
- [~] Multi-value key → each value on separate line (deferred: preserve.patterns shows this works in list)
    - State: `grove.preserve` with multiple patterns
- [x] Missing key argument → error: "accepts 1 arg"
    - State: `grove config get`

### Subcommand: set

- [x] `grove config set <key> <value> --global` → updates git config
    - State: anywhere
- [x] `grove config set <key> <value> --shared` → updates `.grove.toml`
    - State: inside worktree
- [x] Missing `--shared` or `--global` → error: "must specify scope"
    - State: `grove config set grove.plain true`
- [x] Boolean values accepted → `true`, `false`, `yes`, `no`, `on`, `off`, `1`, `0`
    - State: boolean key
- [x] Invalid boolean → error: "invalid boolean value"
    - State: `grove config set --global grove.plain maybe`
- [x] Array keys via set rejected → error: "requires editing .grove.toml directly"
    - State: `grove config set --shared grove.preserve "*.log"`
- [x] Missing key or value → error: "accepts 2 args"
    - State: `grove config set --global key`

### Subcommand: unset

- [x] `grove config unset <key> --global` → removes from git config
    - State: anywhere
- [x] `grove config unset <key> --shared` → removes from `.grove.toml`
    - State: inside worktree
- [x] Non-existent key → silent success (idempotent)
    - State: key doesn't exist
- [~] `unset <key> <value>` → removes specific multi-value (deferred: complex TOML array handling)
    - State: `grove config unset --global grove.preserve "*.log"`
- [~] `unset <key>` (no value) → removes entire key (deferred: implicit in tested scenarios)
    - State: removes all values
- [x] Missing scope flag → error: "must specify scope"
    - State: no `--shared` or `--global`

### Subcommand: init

- [x] `grove config init` → creates `.grove.toml` template
    - State: inside worktree, file doesn't exist
- [x] File already exists → info: "already exists (use --force)"
    - State: `.grove.toml` present
- [x] `grove config init --force` → overwrites existing
    - State: file exists
- [x] Not in workspace → error: "not in a grove workspace"
    - State: outside workspace

### Configuration Keys

- [x] `grove.plain` → boolean
    - State: any
- [x] `grove.debug` → boolean
    - State: any
- [x] `grove.nerdFonts` → boolean
    - State: any
- [x] `grove.preserve` → multi-value string array
    - State: any
- [x] `preserve.patterns` → TOML array (shown in list --shared output)
    - State: `.grove.toml`
- [~] `hooks.add` → TOML array (deferred: CLI can't set, requires manual TOML editing)
    - State: `.grove.toml`

### Scope Handling

- [x] Global → `~/.gitconfig`
    - State: `--global` flag
- [x] Shared → `.grove.toml` in worktree
    - State: `--shared` flag
- [x] Effective → merged from all sources
    - State: no flag

### Precedence Rules

- [~] Boolean settings: git config > TOML > defaults (deferred: requires complex multi-source setup)
    - State: `grove.plain`
- [~] Pattern settings: TOML > git config > defaults (deferred: requires complex multi-source setup)
    - State: `grove.preserve`
- [~] Hooks: TOML only (deferred: hooks require shell integration)
    - State: `hooks.add`

### Error Handling

- [~] Invalid TOML syntax → parse error (deferred: requires malformed file)
    - State: malformed `.grove.toml`
- [~] Atomic writes → temp file + rename (deferred: implementation detail)
    - State: prevents corruption
- [~] Git command failures → stderr included in error (deferred: hard to simulate)
    - State: git error

### Shell Completion

- [x] Config key completion → suggests valid keys
    - State: `grove config get grove.`
- [x] Boolean value completion → suggests `true`, `false`
    - State: `grove config set grove.plain `
