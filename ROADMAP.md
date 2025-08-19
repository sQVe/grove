# Grove Roadmap

## Global Features Progress

| Command | Beautify | --plain | --debug | --help |
| ------- | -------- | ------- | ------- | ------ |
| clone   | [ ]      | [ ]     | [ ]     | [ ]    |
| create  | [ ]      | [ ]     | [ ]     | [ ]    |
| init    | [x]      | [x]     | [x]     | [x]    |
| list    | [ ]      | [ ]     | [ ]     | [ ]    |
| status  | [ ]      | [ ]     | [ ]     | [ ]    |
| switch  | [ ]      | [ ]     | [ ]     | [ ]    |

## Commands

### `init` (5/17 complete)

- [x] Should output help if no arguments are passed
- [x] Should fail if given a subcommand that does not exist

#### `init new` variants

| Command          | Features                                          | Status |
| ---------------- | ------------------------------------------------- | ------ |
| `init new`       | Initialize grove workspace in current directory   | [x]    |
| `init new <dir>` | Initialize grove workspace in specified directory | [x]    |
| `init new <dir>` | Provide completions for directory name            | [x]    |

**Failure conditions:**

- [x] Should accept at most 1 argument
- [x] Current/specified directory is not empty
- [x] Current/specified directory is a Git repository

#### `init clone` variants

| Command                       | Features                                          | Status |
| ----------------------------- | ------------------------------------------------- | ------ |
| `init clone`                  | Output help if no arguments                       | [x]    |
| `init clone <url>`            | Initialize grove workspace in current directory   | [ ]    |
| `init clone <url>`            | Clone specific URL into grove workspace           | [ ]    |
| `init clone <url>`            | Progress bar for cloning                          | [ ]    |
| `init clone <url> --branches` | Setup worktrees for each branch                   | [ ]    |
| `init clone <url> --branches` | Provide completions for branch name               | [ ]    |
| `init clone <url> <dir>`      | Initialize grove workspace in specified directory | [ ]    |
| `init clone <url> <dir>`      | Clone specific URL into grove workspace           | [ ]    |
| `init clone <url> <dir>`      | Progress bar for cloning                          | [ ]    |
| `init clone <url> <dir>`      | Provide completions for directory name            | [ ]    |

**Failure conditions:**

- [x] Should accept 1 or 2 arguments.
- [ ] Current/specified directory is not empty
- [ ] Current/specified directory is a Git repository
- [ ] Specified URL is not a valid Git repository URL

#### `init convert`

| Command        | Features                                     | Status |
| -------------- | -------------------------------------------- | ------ |
| `init convert` | Convert existing Git repo to grove workspace | [ ]    |

**Failure conditions:**

- [ ] Current directory is not a Git repository
- [ ] Repository has uncommitted changes

### `switch` (0/1 complete)

| Command           | Features                               | Status |
| ----------------- | -------------------------------------- | ------ |
| `switch <branch>` | Switch to existing worktree for branch | [ ]    |

### `list` (0/1 complete)

| Command | Features                            | Status |
| ------- | ----------------------------------- | ------ |
| `list`  | Show all worktrees and their status | [ ]    |

### `status` (0/1 complete)

| Command  | Features                                    | Status |
| -------- | ------------------------------------------- | ------ |
| `status` | Show current worktree and repository status | [ ]    |
