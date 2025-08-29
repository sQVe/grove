# Grove Roadmap

## Global Features Progress

| Command | Beautify | --plain | --debug | --help |
| ------- | :------: | :-----: | :-----: | :----: |
| clone   |   [x]    |   [x]   |   [x]   |  [x]   |
| create  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| init    |   [x]    |   [x]   |   [x]   |  [x]   |
| list    |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| status  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| switch  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| config  |   [x]    |   [x]   |   [x]   |  [x]   |

## Commands

### `init`

-   [x] Should output help if no arguments are passed
-   [x] Should fail if given a sub-command that does not exist

#### `init new` variants

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
| `init convert --branches` | Preserve git-ignored files in all created worktrees |  [ ]   |
| `init convert --branches` | Use grove.convert.preserve config patterns          |  [ ]   |
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
-   [ ] Should revert all changes on failure
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
| `config set --global`      | Set global config value                    |  [x]   |
| `config add <key> <value>` | Add to multi-value key (defaults to local) |  [x]   |
| `config add --global`      | Add to global multi-value key              |  [x]   |
| `config unset <key>`       | Remove config setting (defaults to local)  |  [x]   |
| `config unset --global`    | Remove global config setting               |  [x]   |

**Notes:**

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

| Command           | Features                               | Status |
| ----------------- | -------------------------------------- | :----: |
| `switch <branch>` | Switch to existing worktree for branch |  [ ]   |

### `list`

| Command | Features                            | Status |
| ------- | ----------------------------------- | :----: |
| `list`  | Show all worktrees and their status |  [ ]   |

### `status`

| Command  | Features                                    | Status |
| -------- | ------------------------------------------- | :----: |
| `status` | Show current worktree and repository status |  [ ]   |
