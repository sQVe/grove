# Grove Roadmap

## Global Features Progress

| Command | Beautify | --plain | --debug | --help |
| ------- | :------: | :-----: | :-----: | :----: |
| clone   |   [x]    |   [x]   |   [x]   |  [x]   |
| config  |   [x]    |   [x]   |   [x]   |  [x]   |
| create  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| doctor  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| exec    |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| init    |   [x]    |   [x]   |   [x]   |  [x]   |
| list    |   [x]    |   [x]   |   [x]   |  [x]   |
| prune   |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| rename  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| status  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| switch  |   [x]    |   [x]   |   [x]   |  [x]   |

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
| `init convert --branches` | Use grove.convert.preserve config patterns          |  [x]   |
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

| Command                    | Features                                   | Status |
| -------------------------- | ------------------------------------------ | :----: |
| `config`                   | Output help if no arguments                |  [x]   |
| `config list`              | Show all grove.\* settings from git config |  [x]   |
| `config list --global`     | Show only global grove.\* settings         |  [x]   |
| `config get <key>`         | Get specific config value                  |  [x]   |
| `config get --global`      | Get specific global config value           |  [x]   |
| `config set <key> <value>` | Set config value (defaults to local)       |  [x]   |
| `config set <key> <value>` | Provide completions for boolean values     |  [x]   |
| `config set --global`      | Set global config value                    |  [x]   |
| `config add <key> <value>` | Add to multi-value key (defaults to local) |  [x]   |
| `config add <key> <value>` | Provide completions for boolean values     |  [x]   |
| `config add --global`      | Add to global multi-value key              |  [x]   |
| `config unset <key>`       | Remove config setting (defaults to local)  |  [x]   |
| `config unset <key>`       | Provide completions for existing keys      |  [x]   |
| `config unset <key>`       | Unset specific multi-value key             |  [x]   |
| `config unset <key> <val>` | Provide completions for existing values    |  [x]   |
| `config unset --global`    | Remove global config setting               |  [x]   |

**Notes:**

-   Ensure that it is possible to override the default config values when you're setting them in your config.
-   Uses Git's existing config system (no new dependencies)
-   Multi-value support for patterns (e.g., grove.convert.preserve)

**Implementation approach:**

-   Shell out to `git config` commands
-   Read with `git config --get` and `git config --get-all`
-   Same precedence as delta: CLI > ENV > git config > defaults

**Config keys:**

-   `grove.plain` - Disable colors/symbols (boolean, default: false)
-   `grove.debug` - Enable debug output (boolean, default: false)
-   `grove.convert.preserve` - Patterns for ignored files to preserve in new worktrees (multi-value)
    -   Default patterns: `.env`, `.env.local`, `.env.development.local`, `*.local.json`, `*.local.yaml`, `*.local.yml`, `*.local.toml`
    -   Note: Credential files (`*.key`, `*.pem`) not included by default for security

### `switch`

| Command             | Features                                    | Status |
| ------------------- | ------------------------------------------- | :----: |
| `switch <branch>`   | Output path to worktree for branch          |  [x]   |
| `switch <branch>`   | Provide completions for worktree names      |  [x]   |
| `switch shell-init` | Output shell function for directory changes |  [x]   |

### `status`

| Command  | Features                                    | Status |
| -------- | ------------------------------------------- | :----: |
| `status` | Show current worktree and repository status |  [ ]   |

### `prune`

| Command          | Features                                                   | Status |
| ---------------- | ---------------------------------------------------------- | :----: |
| `prune`          | Output help if no arguments                                |  [ ]   |
| `prune`          | Show worktrees linked to deleted remote branches (dry-run) |  [ ]   |
| `prune --commit` | Remove worktrees for branches with `[gone]` upstream       |  [ ]   |
| `prune --force`  | Remove worktrees even if dirty or locked                   |  [ ]   |

**Notes:**

-   Uses `git branch -vv` to detect `[gone]` remote tracking branches
-   Dry-run by default for safety
-   Skips dirty worktrees and locked worktrees unless `--force`
-   Most critical feature - solves biggest daily pain point

**Failure conditions:**

-   Should not remove worktrees with uncommitted changes without `--force`
-   Should not remove locked worktrees without `--force`
-   Should require confirmation before destructive operations

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

| Command                  | Features                                        | Status |
| ------------------------ | ----------------------------------------------- | :----: |
| `rename <old> <new>`     | Rename branch and associated worktree directory |  [ ]   |
| `rename <old> <new>`     | Update upstream tracking references             |  [ ]   |
| `rename <old> <new>`     | Provide completions for existing branch names   |  [ ]   |
| `rename --follow-remote` | Detect and migrate from remote branch renames   |  [ ]   |

**Notes:**

-   Atomically handles git branch -m, directory rename, and upstream updates
-   Eliminates the painful 4-step manual process
-   Maintains worktree functionality throughout rename

**Failure conditions:**

-   Should not rename if worktree has uncommitted changes
-   Should not rename if target branch name already exists
-   Should revert all changes if any step fails

### `doctor`

| Command        | Features                                      | Status |
| -------------- | --------------------------------------------- | :----: |
| `doctor`       | Check git safe.directory entries              |  [ ]   |
| `doctor`       | Detect detached HEAD worktrees                |  [ ]   |
| `doctor`       | Find missing upstream tracking                |  [ ]   |
| `doctor`       | Identify worktrees pointing at gone upstreams |  [ ]   |
| `doctor --fix` | Automatically fix common issues               |  [ ]   |

**Notes:**

-   Diagnoses common worktree setup problems
-   Safe.directory issues prevent Git operations
-   Detached HEADs indicate incomplete setup
-   Missing upstreams break sync operations
