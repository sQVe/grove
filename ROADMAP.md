# Grove Roadmap

## Global Features Progress

| Command | Beautify | --plain | --debug | --help |
| ------- | :------: | :-----: | :-----: | :----: |
| clone   |   [x]    |   [x]   |   [x]   |  [x]   |
| config  |   [x]    |   [x]   |   [x]   |  [x]   |
| create  |   [x]    |   [x]   |   [x]   |  [x]   |
| delete  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| doctor  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| exec    |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| init    |   [x]    |   [x]   |   [x]   |  [x]   |
| list    |   [x]    |   [x]   |   [x]   |  [x]   |
| lock    |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| prune   |   [x]    |   [x]   |   [x]   |  [x]   |
| rename  |   [x]    |   [x]   |   [x]   |  [x]   |
| status  |   [x]    |   [x]   |   [x]   |  [x]   |
| switch  |   [x]    |   [x]   |   [x]   |  [x]   |
| unlock  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |

## Commands

### `init`

-   [x] Should output help if no arguments are passed
-   [x] Should fail if given a sub-command that does not exist

#### `init new`

| Command          | Features                                          | Status |
| ---------------- | ------------------------------------------------- | :----: |
| `init new`       | Initialize grove workspace in current directory   |  [x]   |
| `init new <dir>` | Initialize grove workspace in specified directory |  [x]   |
| `init new <dir>` | Provide completions for directory name            |  [x]   |

**Notes:**

-   When given a directory name, the output should always output an absolute path.

**Failure conditions:**

-   [x] Should accept at most 1 argument
-   [x] Not inside a grove workspace already
-   [x] Current/specified directory is not empty
-   [x] Current/specified directory is a Git repository

#### `init convert`

| Command                   | Features                                            | Status |
| ------------------------- | --------------------------------------------------- | :----: |
| `init convert`            | Convert existing Git repo to Grove workspace        |  [x]   |
| `init convert`            | Move .git to .bare                                  |  [x]   |
| `init convert`            | Configure repository as bare                        |  [x]   |
| `init convert`            | Create worktree for current branch                  |  [x]   |
| `init convert`            | Move all files to worktree directory                |  [x]   |
| `init convert`            | Create .git file pointing to .bare                  |  [x]   |
| `init convert --branches` | Setup worktrees for local and remote branches       |  [x]   |
| `init convert --branches` | Preserve git-ignored files in all created worktrees |  [x]   |
| `init convert --branches` | Use grove.preserve config patterns                  |  [x]   |
| `init convert --branches` | Provide completions for branch names                |  [x]   |

**Notes:**

-   Preserve common git-ignored files in created worktrees to match normal git behavior:
    -   `.env` and `.env.local` files
    -   Local config overrides (`*.local.json`, `*.local.yaml`, etc.)
    -   Credential files require explicit opt-in for security

**Failure conditions:**

-   [x] Should not accept any arguments
-   [x] Current directory is not a Git repository
-   [x] Current directory is already a Grove workspace
-   [x] Repository is in detached HEAD state
-   [x] Repository has ongoing merge/rebase
-   [x] Should not convert when in a dirty state
-   [x] Should not output double error message:
-   [x] Should revert all changes on failure
-   [x] Convert branch names to safe directory names

### `clone`

| Command                  | Features                                          | Status |
| ------------------------ | ------------------------------------------------- | :----: |
| `clone`                  | Output help if no arguments                       |  [x]   |
| `clone <url>`            | Initialize grove workspace in current directory   |  [x]   |
| `clone <url>`            | Clone specific URL into grove workspace           |  [x]   |
| `clone <url>`            | Progress bar for cloning                          |  [x]   |
| `clone <url> --branches` | Setup worktrees for each branch                   |  [x]   |
| `clone <url> --branches` | Provide completions for branch name               |  [x]   |
| `clone <url> <dir>`      | Initialize grove workspace in specified directory |  [x]   |
| `clone <url> <dir>`      | Clone specific URL into grove workspace           |  [x]   |
| `clone <url> <dir>`      | Provide completions for directory name            |  [x]   |

**Failure conditions:**

-   [x] Should accept 1 or 2 arguments.
-   [x] Not inside a grove workspace already
-   [x] Current/specified directory is not empty
-   [x] Current/specified directory is a Git repository
-   [x] Convert branch name to safe directory name

### `config`

| Command                 | Features                                   | Status |
| ----------------------- | ------------------------------------------ | :----: |
| `config`                | Output help if no arguments                |  [x]   |
| `config init`           | Create .grove.toml in current worktree     |  [x]   |
| `config list`           | Show all grove.\* settings from git config |  [x]   |
| `config list --shared`  | Show .grove.toml settings                  |  [x]   |
| `config list --global`  | Show only global grove.\* settings         |  [x]   |
| `config get <key>`      | Get effective config value (merged)        |  [x]   |
| `config get --shared`   | Get value from .grove.toml                 |  [x]   |
| `config get --global`   | Get value from git config                  |  [x]   |
| `config set --shared`   | Set value in .grove.toml                   |  [x]   |
| `config set --global`   | Set value in git config                    |  [x]   |
| `config unset --shared` | Remove setting from .grove.toml            |  [x]   |
| `config unset --global` | Remove setting from git config             |  [x]   |

**Notes:**

-   Two config layers: `.grove.toml` (team-shareable) and git config (personal)
-   Config precedence varies by setting type:
    -   Team settings (preserve patterns): TOML > git config > defaults
    -   Personal settings (plain, debug): git config > TOML > defaults
-   `--shared` and `--global` flags required for set/unset operations

**Config keys:**

-   `grove.plain` - Disable colors/symbols (boolean, default: false)
-   `grove.debug` - Enable debug output (boolean, default: false)
-   `grove.preserve` - Patterns for ignored files to preserve in new worktrees (multi-value)
    -   Default patterns: `.env`, `.env.local`, `.env.development.local`, `*.local.json`, `*.local.yaml`, `*.local.yml`, `*.local.toml`
    -   Note: Credential files (`*.key`, `*.pem`) not included by default for security
-   `hooks.create` - Commands to run after creating worktrees (TOML only, array)

### `switch`

| Command             | Features                                    | Status |
| ------------------- | ------------------------------------------- | :----: |
| `switch <branch>`   | Output path to worktree for branch          |  [x]   |
| `switch <branch>`   | Provide completions for worktree names      |  [x]   |
| `switch shell-init` | Output shell function for directory changes |  [x]   |

### `create`

| Command                 | Features                                      | Status |
| ----------------------- | --------------------------------------------- | :----: |
| `create <branch>`       | Create worktree for existing branch           |  [x]   |
| `create <branch>`       | Create worktree with new branch if not exists |  [x]   |
| `create <branch>`       | Provide completions for branch names          |  [x]   |
| `create <branch>`       | Sanitize branch name for directory name       |  [x]   |
| `create <branch>`       | Preserve configured files from source         |  [x]   |
| `create <branch>`       | Run configured hooks after creation           |  [x]   |
| `create -s <branch>`    | Switch to worktree after creation             |  [x]   |
| `create --detach <ref>` | Create worktree at commit/tag without branch  |  [ ]   |

**Notes:**

-   `--detach` useful for inspecting releases, hotfixes on tags

### `status`

| Command            | Features                                         | Status |
| ------------------ | ------------------------------------------------ | :----: |
| `status`           | Show current worktree status (branch, sync, etc) |  [x]   |
| `status`           | Show dirty state and stash count                 |  [x]   |
| `status`           | Show ongoing operations (merge/rebase)           |  [x]   |
| `status`           | Show conflicts and lock status                   |  [x]   |
| `status --verbose` | Full sectioned diagnostic output                 |  [x]   |
| `status --json`    | Machine-readable JSON output                     |  [x]   |

### `prune`

| Command          | Features                                                   | Status |
| ---------------- | ---------------------------------------------------------- | :----: |
| `prune`          | Output help if no arguments                                |  [x]   |
| `prune`          | Show worktrees linked to deleted remote branches (dry-run) |  [x]   |
| `prune --commit` | Remove worktrees for branches with `[gone]` upstream       |  [x]   |
| `prune --force`  | Remove worktrees even if dirty or locked                   |  [x]   |
| `prune --stale`  | Also prune worktrees with no commits in specified duration |  [x]   |

**Notes:**

-   Uses `git branch -vv` to detect `[gone]` remote tracking branches
-   Dry-run by default for safety
-   Skips dirty worktrees and locked worktrees unless `--force`
-   Most critical feature - solves biggest daily pain point

**Failure conditions:**

-   [x] Should not remove worktrees with uncommitted changes without `--force`
-   [x] Should not remove locked worktrees without `--force`
-   [x] Should require confirmation before destructive operations

### `list`

| Command          | Features                                        | Status |
| ---------------- | ----------------------------------------------- | :----: |
| `list`           | Show all worktrees with status by default       |  [x]   |
| `list --fast`    | Skip sync status for faster output              |  [x]   |
| `list --json`    | Machine-readable output for tooling integration |  [x]   |
| `list --verbose` | Show extra details (paths, upstream names)      |  [x]   |

**Default output format:**

```
● main              [clean]    ↑2
  feature-auth      [dirty]    ↓5
  bugfix-timeout    [clean]    ↑1↓3
  old-experiment    [dirty]    ×
  release-2.0       [clean]    =
```

**ASCII fallback (when unicode not supported):**

```
* main              [clean]    +2
  feature-auth      [dirty]    -5
  bugfix-timeout    [clean]    +1-3
  old-experiment    [dirty]    gone
  release-2.0       [clean]    =
```

**Status symbols:**

-   `●` / `*` Current worktree
-   `[dirty]` Uncommitted changes (blocks switching)
-   `[clean]` No uncommitted changes
-   `↑N` / `+N` N commits ahead of upstream
-   `↓N` / `-N` N commits behind upstream
-   `↑N↓M` / `+N-M` N commits ahead, M commits behind (diverged)
-   `×` / `gone` Upstream branch deleted
-   `=` In sync with upstream

**Notes:**

-   Single command answers "where is my work?" and "what needs attention?"
-   Rich default shows everything useful at a glance
-   Performance-conscious with --fast escape hatch
-   JSON output enables editor/IDE integration

### `exec`

| Command                   | Features                                  | Status |
| ------------------------- | ----------------------------------------- | :----: |
| `exec --all -- <command>` | Run command in all worktree directories   |  [ ]   |
| `exec --all -- <command>` | Interactive confirmation before execution |  [ ]   |
| `exec --all -- <command>` | Sequential execution with prefixed output |  [ ]   |
| `exec --worktree <name>`  | Run command in specific worktree          |  [ ]   |

**Notes:**

-   Uses `--` separator to clearly delineate Grove args from command
-   Always asks for confirmation to prevent accidental damage
-   Perfect for `npm install`, dependency updates across branches
-   Sequential execution keeps output readable

**Failure conditions:**

-   Should require explicit confirmation for all executions
-   Should handle command failures gracefully without stopping
-   Should clearly identify which worktree each output comes from

### `rename`

| Command              | Features                                        | Status |
| -------------------- | ----------------------------------------------- | :----: |
| `rename <old> <new>` | Rename branch and associated worktree directory |  [x]   |
| `rename <old> <new>` | Update upstream tracking references             |  [x]   |
| `rename <old> <new>` | Provide completions for existing branch names   |  [x]   |

**Notes:**

-   Atomically handles git branch -m, directory rename, and upstream updates
-   Eliminates the painful 4-step manual process
-   Maintains worktree functionality throughout rename

**Failure conditions:**

-   [x] Should not rename if worktree has uncommitted changes
-   [x] Should not rename if target branch name already exists
-   [x] Should revert all changes if any step fails

### `delete`

| Command                    | Features                               | Status |
| -------------------------- | -------------------------------------- | :----: |
| `delete <branch>`          | Remove worktree and optionally branch  |  [ ]   |
| `delete <branch>`          | Provide completions for worktree names |  [ ]   |
| `delete --force <branch>`  | Force delete even if dirty/locked      |  [ ]   |
| `delete --branch <branch>` | Also delete the branch after removal   |  [ ]   |

**Notes:**

-   Wraps `git worktree remove` with Grove conventions
-   Safe by default: refuses to delete dirty or locked worktrees
-   `--branch` flag provides convenient cleanup of merged branches

**Failure conditions:**

-   [ ] Should not delete worktree with uncommitted changes without `--force`
-   [ ] Should not delete locked worktrees without `--force`
-   [ ] Should not delete the current worktree
-   [ ] Should warn if branch has unpushed commits when using `--branch`

### `lock`

| Command                        | Features                               | Status |
| ------------------------------ | -------------------------------------- | :----: |
| `lock <branch>`                | Lock worktree to prevent removal       |  [ ]   |
| `lock <branch>`                | Provide completions for worktree names |  [ ]   |
| `lock --reason <msg> <branch>` | Add reason for locking                 |  [ ]   |

**Notes:**

-   Locked worktrees are protected from `prune` and `delete`
-   Lock reason displayed in `list --verbose` and `status`

### `unlock`

| Command           | Features                                 | Status |
| ----------------- | ---------------------------------------- | :----: |
| `unlock <branch>` | Unlock worktree to allow removal         |  [ ]   |
| `unlock <branch>` | Provide completions for locked worktrees |  [ ]   |

### `doctor`

| Command        | Features                                      | Status |
| -------------- | --------------------------------------------- | :----: |
| `doctor`       | Check git safe.directory entries              |  [ ]   |
| `doctor`       | Detect detached HEAD worktrees                |  [ ]   |
| `doctor`       | Find missing upstream tracking                |  [ ]   |
| `doctor`       | Identify worktrees pointing at gone upstreams |  [ ]   |
| `doctor`       | Run `git worktree repair` to fix broken links |  [ ]   |
| `doctor --fix` | Automatically fix common issues               |  [ ]   |

**Notes:**

-   Diagnoses common worktree setup problems
-   Safe.directory issues prevent Git operations
-   Detached HEADs indicate incomplete setup
-   Missing upstreams break sync operations
