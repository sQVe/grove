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

- [x] Should output help if no arguments are passed
- [x] Should fail if given a sub-command that does not exist

#### `init new` variants

| Command          | Features                                          | Status |
| ---------------- | ------------------------------------------------- | :----: |
| `init new`       | Initialize grove workspace in current directory   |  [x]   |
| `init new <dir>` | Initialize grove workspace in specified directory |  [x]   |
| `init new <dir>` | Provide completions for directory name            |  [x]   |

**Notes:**

- When given a directory name, the output should always output an absolute path.

**Failure conditions:**

- [x] Should accept at most 1 argument
- [x] Not inside a grove workspace already
- [x] Current/specified directory is not empty
- [x] Current/specified directory is a Git repository

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

- [x] Should accept 1 or 2 arguments.
- [x] Not inside a grove workspace already
- [x] Current/specified directory is not empty
- [x] Current/specified directory is a Git repository
- [x] Convert branch name to safe directory name

#### `init convert`

| Command                   | Features                                      | Status |
| ------------------------- | --------------------------------------------- | :----: |
| `init convert`            | Convert existing Git repo to Grove workspace  |  [x]   |
| `init convert`            | Move .git to .bare                            |  [x]   |
| `init convert`            | Configure repository as bare                  |  [x]   |
| `init convert`            | Create worktree for current branch            |  [x]   |
| `init convert`            | Move all files to worktree directory          |  [x]   |
| `init convert`            | Create .git file pointing to .bare            |  [x]   |
| `init convert --branches` | Setup worktrees for local branches            |  [ ]   |
| `init convert --branches` | Copy untracked files to all created worktrees |  [ ]   |
| `init convert --branches` | Provide completions for branch names          |  [ ]   |

**Notes:**

- Flow should be:

    > `init convert --branches main,develop,feature-x`:
    >
    > 1. Move everything to main/ (current branch)
    > 2. Create develop/ worktree
    > 3. Copy untracked files from main/ to develop/
    > 4. Create feature-x/ worktree
    > 5. Copy untracked files from main/ to feature-x/

- Configuration-based file preservation via `.groveconfig`:
    - Include-only patterns (no exclusions needed)
    - Defaults: `.env*`, `*.local.*`, `*.key`, `*.pem`
    - Post-operation hooks: `post_convert = ["pnpm install"]`

    User experience:
    1. Show what will be preserved (matches patterns)
    2. Show what will be lost (everything else)
    3. Group large directories (node_modules, .cache) separately in warnings
    4. Require confirmation if many files at risk

    No full backups - only preserve what's explicitly configured.

    File deletion preview:
    - Use `git clean -ndX` to show exactly what Git would delete
    - Group by leaf name (e.g., node_modules/, dist/, \*.json) to avoid language-specific hardcoding
    - Always highlight sensitive files (.env, .key, etc.) separately
    - Show counts for repeated patterns across monorepo packages
    - Simple presentation: "node_modules/ (4 locations)" vs 50+ individual file lines

    ```go
    	import (
    		"fmt"
    		"path/filepath"
    		"strings"
    	)

    	func GroupIgnored(output string) {
    		byLeaf := make(map[string][]string)

    		for _, line := range strings.Split(output, "\n") {
    			path := strings.TrimPrefix(line, "Would remove ")

    			// Get the last path component
    			leaf := filepath.Base(path)
    			if strings.HasSuffix(path, "/") {
    				leaf = path[strings.LastIndex(path[:len(path)-1], "/")+1:]
    			}

    			byLeaf[leaf] = append(byLeaf[leaf], path)
    		}

    		// Present grouped by pattern
    		for pattern, paths := range byLeaf {
    			if len(paths) > 1 {
    				fmt.Printf("%s (%d locations)\n", pattern, len(paths))
    			} else {
    				fmt.Printf("%s\n", paths[0])
    			}
    		}
    	}
    ```

**Failure conditions:**

- [x] Should not accept any arguments
- [x] Current directory is not a Git repository
- [x] Current directory is already a Grove workspace
- [x] Repository is in detached HEAD state
- [x] Repository has ongoing merge/rebase
- [x] Should not convert when in a dirty state
- [ ] Should revert all changes on failure
- [x] Convert branch names to safe directory names

**Safety Strategy:**

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
