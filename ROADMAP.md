# Grove Roadmap

## Global Features Progress

| Command | Beautify | --plain | --debug | --help |
| ------- | :------: | :-----: | :-----: | :----: |
| add     |   [x]    |   [x]   |   [x]   |  [x]   |
| clone   |   [x]    |   [x]   |   [x]   |  [x]   |
| config  |   [x]    |   [x]   |   [x]   |  [x]   |
| doctor  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| exec    |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| init    |   [x]    |   [x]   |   [x]   |  [x]   |
| list    |   [x]    |   [x]   |   [x]   |  [x]   |
| lock    |   [x]    |   [x]   |   [x]   |  [x]   |
| move    |   [x]    |   [x]   |   [x]   |  [x]   |
| prune   |   [x]    |   [x]   |   [x]   |  [x]   |
| remove  |   [x]    |   [x]   |   [x]   |  [x]   |
| status  |   [x]    |   [x]   |   [x]   |  [x]   |
| switch  |   [x]    |   [x]   |   [x]   |  [x]   |
| unlock  |   [x]    |   [x]   |   [x]   |  [x]   |

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
| `clone <pr-url>`         | Clone repo and create worktree for PR             |  [x]   |
| `clone <pr-url>`         | Support fork PRs with automatic remote setup      |  [x]   |
| `clone <pr-url> <dir>`   | Clone PR to specified directory                   |  [x]   |

**Notes:**

-   PR cloning requires `gh` CLI to be installed and authenticated
-   Uses `gh repo clone` to respect user's protocol preference (SSH/HTTPS)
-   Fork PRs automatically add a remote named `pr-{number}-{owner}`

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
    -   Default patterns: `.env`, `.env.keys`, `.env.local`, `.env.*.local`, `.envrc`, `docker-compose.override.yml`, `*.local.json`, `*.local.toml`, `*.local.yaml`, `*.local.yml`
    -   Note: Credential files (`*.key`, `*.pem`) not included by default for security
-   `grove.autoLock` - Branch patterns to auto-lock when creating worktrees (multi-value)
    -   Default patterns: `main`, `master`, `develop`
    -   Supports glob patterns (e.g., `release/*`)
    -   Locked with reason: "Auto-locked (grove.autoLock)"
-   `hooks.add` - Commands to run after adding worktrees (TOML only, array)

### `switch`

| Command             | Features                                    | Status |
| ------------------- | ------------------------------------------- | :----: |
| `switch <branch>`   | Output path to worktree for branch          |  [x]   |
| `switch <branch>`   | Provide completions for worktree names      |  [x]   |
| `switch shell-init` | Output shell function for directory changes |  [x]   |

### `add`

| Command              | Features                                     | Status |
| -------------------- | -------------------------------------------- | :----: |
| `add <branch>`       | Add worktree for existing branch             |  [x]   |
| `add <branch>`       | Add worktree with new branch if not exists   |  [x]   |
| `add <branch>`       | Provide completions for branch names         |  [x]   |
| `add <branch>`       | Sanitize branch name for directory name      |  [x]   |
| `add <branch>`       | Preserve configured files from source        |  [x]   |
| `add <branch>`       | Run configured hooks after creation          |  [x]   |
| `add -s <branch>`    | Switch to worktree after creation            |  [x]   |
| `add #<number>`      | Add worktree from GitHub PR number           |  [x]   |
| `add <pr-url>`       | Add worktree from GitHub PR URL              |  [x]   |
| `add <pr-ref>`       | Support fork PRs with automatic remote setup |  [x]   |
| `add --detach <ref>` | Add worktree at commit/tag without branch    |  [ ]   |

**Notes:**

-   `--detach` useful for inspecting releases, hotfixes on tags
-   PR support requires `gh` CLI to be installed and authenticated
-   PR format: `#123` (requires being in a grove workspace) or full GitHub PR URL
-   Fork PRs automatically add a remote named `pr-{number}-{owner}`

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

**Default output format (color mode with Nerd Fonts):**

```
 main  ↑2
  feature-auth  ↓5
  bugfix-timeout ↑1↓3
  old-experiment  ×
  release-2.0 =
```

**Plain mode fallback:**

```
* main [dirty] +2 [locked]
  feature-auth [dirty] -5
  bugfix-timeout +1-3
  old-experiment [dirty] gone
  release-2.0 =
```

**Status symbols:**

| Element        | Color Mode              | Plain Mode           |
| -------------- | ----------------------- | -------------------- |
| Current marker | (U+E0A0 nf-pl-branch)   | `*`                  |
| Dirty state    | (nf-md-diff) when dirty | `[dirty]` when dirty |
| Clean state    | nothing                 | nothing              |
| Lock           | (U+F033E nf-md-lock)    | `[locked]`           |
| Ahead          | `↑N` green              | `+N`                 |
| Behind         | `↓N` yellow             | `-N`                 |
| Gone           | `×`                     | `gone`               |
| Synced         | `=` dimmed              | `=`                  |

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

### `move`

| Command            | Features                                      | Status |
| ------------------ | --------------------------------------------- | :----: |
| `move <old> <new>` | Move branch and associated worktree directory |  [x]   |
| `move <old> <new>` | Update upstream tracking references           |  [x]   |
| `move <old> <new>` | Provide completions for existing branch names |  [x]   |

**Notes:**

-   Atomically handles git branch -m, directory rename, and upstream updates
-   Eliminates the painful 4-step manual process
-   Maintains worktree functionality throughout move

**Failure conditions:**

-   [x] Should not move if worktree has uncommitted changes
-   [x] Should not move if target branch name already exists
-   [x] Should revert all changes if any step fails

### `remove`

| Command                    | Features                               | Status |
| -------------------------- | -------------------------------------- | :----: |
| `remove <branch>`          | Remove worktree and optionally branch  |  [x]   |
| `remove <branch>`          | Provide completions for worktree names |  [x]   |
| `remove --force <branch>`  | Force remove even if dirty/locked      |  [x]   |
| `remove --branch <branch>` | Also delete the branch after removal   |  [x]   |

**Notes:**

-   Wraps `git worktree remove` with Grove conventions
-   Safe by default: refuses to remove dirty or locked worktrees
-   `--branch` flag provides convenient cleanup of merged branches

**Failure conditions:**

-   [x] Should not remove worktree with uncommitted changes without `--force`
-   [x] Should not remove locked worktrees without `--force`
-   [x] Should not remove the current worktree
-   [x] Should warn if branch has unpushed commits when using `--branch`

### `lock`

| Command                  | Features                                          | Status |
| ------------------------ | ------------------------------------------------- | :----: |
| `lock <branch>`          | Lock worktree to prevent removal                  |  [x]   |
| `lock <branch>`          | Provide completions for non-locked worktree names |  [x]   |
| `lock -r <msg> <branch>` | Add reason for locking                            |  [x]   |

**Notes:**

-   Locked worktrees are protected from `prune` and `remove`
-   Lock reason displayed in `list --verbose` and `status`
-   Auto-lock: configurable branch patterns are locked on creation (see `grove.autoLock` config)

### `unlock`

| Command           | Features                                 | Status |
| ----------------- | ---------------------------------------- | :----: |
| `unlock <branch>` | Unlock worktree to allow removal         |  [x]   |
| `unlock <branch>` | Provide completions for locked worktrees |  [x]   |

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
