# Grove Roadmap

## Global Features Progress

| Command | Beautify | --plain | --debug | --help |
| ------- | :------: | :-----: | :-----: | :----: |
| clone   |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| create  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| init    |   [x]    |   [x]   |   [x]   |  [x]   |
| list    |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| status  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |
| switch  |   [ ]    |   [ ]   |   [ ]   |  [ ]   |

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

#### `init clone` variants

| Command                       | Features                                          | Status |
| ----------------------------- | ------------------------------------------------- | :----: |
| `init clone`                  | Output help if no arguments                       |  [x]   |
| `init clone <url>`            | Initialize grove workspace in current directory   |  [x]   |
| `init clone <url>`            | Clone specific URL into grove workspace           |  [x]   |
| `init clone <url>`            | Progress bar for cloning                          |  [x]   |
| `init clone <url> --branches` | Setup worktrees for each branch                   |  [x]   |
| `init clone <url> --branches` | Provide completions for branch name               |  [x]   |
| `init clone <url> <dir>`      | Initialize grove workspace in specified directory |  [x]   |
| `init clone <url> <dir>`      | Clone specific URL into grove workspace           |  [x]   |
| `init clone <url> <dir>`      | Provide completions for directory name            |  [x]   |

**Failure conditions:**

-   [x] Should accept 1 or 2 arguments.
-   [x] Not inside a grove workspace already
-   [x] Current/specified directory is not empty
-   [x] Current/specified directory is a Git repository
-   [x] Convert branch name to safe directory name

#### `init convert`

| Command                   | Features                                        | Status |
| ------------------------- | ----------------------------------------------- | :----: |
| `init convert`            | Convert existing Git repo to Grove workspace    |  [x]   |
| `init convert`            | Move .git to .bare                              |  [x]   |
| `init convert`            | Configure repository as bare                    |  [x]   |
| `init convert`            | Create worktree for current branch              |  [x]   |
| `init convert`            | Move all files to worktree directory            |  [x]   |
| `init convert`            | Create .git file pointing to .bare              |  [x]   |
| `init convert --branches` | Setup worktrees for local and remote branches   |  [x]   |
| `init convert --branches` | Copy git-ignored files to all created worktrees |  [ ]   |
| `init convert --branches` | Provide completions for branch names            |  [x]   |

**Notes:**

-   Copy common git-ignored files to created worktrees to match normal git behavior:
    -   `.env*` files
    -   `*.local.*` files
    -   `*.key`, `*.pem` credential files

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
