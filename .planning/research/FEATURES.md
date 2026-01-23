# Features Research

**Domain:** CLI fetch command for git worktree management
**Researched:** 2026-01-23
**Confidence:** HIGH (based on git documentation and existing CLI patterns)

## Table Stakes

Features users expect from a fetch command. Missing any of these makes the command feel incomplete.

| Feature             | Why Expected                                    | Complexity | Notes                                     |
| ------------------- | ----------------------------------------------- | ---------- | ----------------------------------------- |
| Fetch all remotes   | Standard behavior for `git fetch --all`         | Low        | Grove already has `FetchPrune()`          |
| Progress indication | Users expect feedback during network operations | Low        | Spinner or progress bar                   |
| Prune stale refs    | `--prune` is standard practice                  | Low        | Already implemented in `git.FetchPrune()` |
| Error handling      | Clear messages when fetch fails (network, auth) | Low        | Map git errors to user-friendly messages  |
| Quiet mode          | `-q/--quiet` for scripting                      | Low        | Suppress progress output                  |
| Summary output      | Show what changed after fetch                   | Medium     | Count of updated refs                     |

## Differentiators

Features that would distinguish Grove fetch from raw `git fetch`. Not expected, but valuable for worktree workflows.

| Feature               | Value Proposition                                           | Complexity | Notes                                    |
| --------------------- | ----------------------------------------------------------- | ---------- | ---------------------------------------- |
| Show incoming changes | `git log HEAD..@{u}` after fetch shows what would be pulled | Medium     | Key value-add for workflow               |
| Per-worktree status   | Show which worktrees are behind after fetch                 | Medium     | Leverages existing `list` infrastructure |
| Parallel fetch        | `git fetch --jobs=N` for multiple remotes                   | Low        | Just pass `--jobs` flag                  |
| Dry run               | `--dry-run` shows what would be fetched                     | Low        | Native git support                       |
| JSON output           | `--json` for scripting/tooling                              | Low        | Follow existing Grove patterns           |
| Prune tags            | `--prune-tags` removes stale local tags                     | Low        | Native git support, use carefully        |
| Verbose mode          | `-v/--verbose` shows detailed ref updates                   | Low        | Pass through to git                      |

## Anti-Features

Features to explicitly avoid. Common mistakes or scope creep.

| Anti-Feature        | Why Avoid                                               | What to Do Instead                     |
| ------------------- | ------------------------------------------------------- | -------------------------------------- |
| Auto-pull/merge     | Fetch should be read-only, never modify working tree    | Keep fetch and pull separate           |
| Interactive mode    | Asking "do you want to pull?" breaks scripting          | Let user decide next action            |
| Branch creation     | Don't auto-create local branches from remote            | User creates via `grove add`           |
| Submodule recursion | Adds complexity, most worktree users don't need it      | Add only if explicitly requested       |
| Per-worktree fetch  | Worktrees share refs, fetching per-worktree is wasteful | Fetch once at bare repo level          |
| Complex filtering   | Filtering which refs to fetch                           | Keep it simple: all or specific remote |

## Reference Commands

### git fetch

```bash
git fetch --all           # Fetch all remotes
git fetch --prune         # Remove stale remote-tracking refs
git fetch --jobs=N        # Parallel fetch for multiple remotes
git fetch --dry-run       # Show what would be fetched
git fetch -v              # Verbose output
git fetch -q              # Quiet mode
```

Key flags from [git-fetch documentation](https://git-scm.com/docs/git-fetch):

- `--all` fetches all remotes
- `--prune` removes refs that no longer exist on remote
- `--prune-tags` removes local tags deleted on remote
- `-j/--jobs=N` parallel fetching (default sequential)
- `--dry-run` simulate without changes

### gh repo sync

```bash
gh repo sync              # Sync local from remote parent
gh repo sync --branch v1  # Specific branch
gh repo sync --force      # Hard reset to match
```

From [gh repo sync](https://cli.github.com/manual/gh_repo_sync):

- Focused on fork synchronization
- Uses default branch by default
- Force flag for hard reset

### Viewing incoming changes

```bash
git fetch origin
git log HEAD..@{u}        # Commits on remote not in local
git diff HEAD..@{u}       # Full diff of changes
git diff --stat @{u}      # Summary of changed files
```

From [git workflow guides](https://safjan.com/git-workflow-reviewing-changes-before-pulling-remote-branch/):

- Fetch is read-only, pull is destructive
- Preview before integrating

### Git Repo Manager (grm)

```bash
grm fetch                 # Fetch all remotes in worktree setup
grm wt pull               # Update all worktrees
```

From [Git Repo Manager docs](https://hakoerber.github.io/git-repo-manager/worktree_remotes.html):

- Equivalent to `git fetch --all`
- Designed for worktree workflows

## Feature Dependencies

```
grove fetch (core)
    |
    +-- Fetch all remotes
    |       |
    |       +-- Progress indication
    |       +-- Error handling
    |       +-- Prune stale refs
    |
    +-- Show changes (differentiator)
            |
            +-- Per-worktree behind status
            +-- Incoming commit summary
```

## Recommended MVP

**Phase 1 (MVP):**

1. Fetch all remotes with prune (table stakes)
2. Progress spinner during fetch (table stakes)
3. Summary of what changed (table stakes)
4. `--quiet` flag (table stakes)

**Phase 2 (Enhancement):**

1. Show incoming changes per worktree (`--status`)
2. Parallel fetch with `--jobs`
3. `--json` output
4. `--dry-run`

**Defer indefinitely:**

- Submodule support
- Interactive mode
- Auto-pull

## Sources

- [Git fetch documentation](https://git-scm.com/docs/git-fetch)
- [Git fetch options](https://git-scm.com/docs/fetch-options)
- [gh repo sync](https://cli.github.com/manual/gh_repo_sync)
- [Git Repo Manager worktrees](https://hakoerber.github.io/git-repo-manager/worktree_remotes.html)
- [Atlassian Git fetch tutorial](https://www.atlassian.com/git/tutorials/syncing/git-fetch)
- [CLI UX progress patterns](https://evilmartians.com/chronicles/cli-ux-best-practices-3-patterns-for-improving-progress-displays)
- [Git prune best practices](https://www.git-tower.com/learn/git/faq/cleanup-remote-branches-with-git-prune)
