# 🌳 Grove

[![GitHub release](https://img.shields.io/github/v/release/sQVe/grove?style=flat-square)](https://github.com/sQVe/grove/releases/latest)
[![License](https://img.shields.io/badge/license-MIT-blue.svg?style=flat-square)](LICENSE)

> [!WARNING]
> Grove is under active development. APIs may change.

**Grove** is a fast, intuitive Git worktree management tool that makes worktrees as simple as switching branches.

## ❓ Why

Git worktrees let you work on multiple branches simultaneously in separate directories.
No stashing. No "wrong branch" mistakes. Your work stays exactly where you left it.

**The problem:** `git worktree` is powerful but clunky. Setting up the recommended
`.bare` structure is manual. Switching means `cd`-ing to paths. New worktrees don't
have your `.env` files.

**Grove fixes this:**

- **Best-practice setup** — Uses the `.bare` repo structure automatically
- **Switch like branches** — `grove switch main` just works
- **File preservation** — `.env`, `.envrc` copied to new worktrees
- **PR checkout** — `grove add #123` creates a worktree for any PR
- **Post-create hooks** — Run `npm install` automatically after creating worktrees

## 📦 Installation

Build from source (package managers coming soon):

```bash
git clone https://github.com/sQVe/grove && cd grove
go build -o grove ./cmd/grove
sudo mv grove /usr/local/bin/ # or add to PATH
```

## 🔧 Setup

### Shell Integration

Enables `grove switch` to change your shell's directory — making worktree switching as seamless as `cd`.

```bash
# Without shell integration
cd $(grove switch main)

# With shell integration
grove switch main
```

Add to your shell config:

```bash
# bash (~/.bashrc) or zsh (~/.zshrc)
eval "$(grove switch shell-init)"

# fish (~/.config/fish/config.fish)
grove switch shell-init | source

# powershell ($PROFILE)
grove switch shell-init | Invoke-Expression
```

### Shell Completion

Tab completion for commands, flags, and worktree names.

```bash
# bash
eval "$(grove completion bash)"

# zsh
eval "$(grove completion zsh)"

# fish
grove completion fish | source

# powershell
grove completion powershell | Invoke-Expression
```

## 🚀 Quick Start

```bash
# Clone a repository into a Grove workspace
grove clone https://github.com/owner/repo

# See all worktrees with status
grove list

# Create a new worktree and switch to it
grove add feat/auth --switch

# Work on your feature...

# Switch back to main
grove switch main

# Clean up when done
grove remove feat/auth --branch
```

## 📗 Commands

| Command  | Description                                    |
| -------- | ---------------------------------------------- |
| `clone`  | Clone repository into Grove workspace          |
| `init`   | Initialize workspace (new or convert existing) |
| `add`    | Add worktree for branch, PR, or ref            |
| `switch` | Switch to a worktree                           |
| `list`   | List worktrees with status                     |
| `status` | Show current worktree status                   |
| `remove` | Remove a worktree                              |
| `move`   | Rename branch and its worktree                 |
| `lock`   | Lock worktree to prevent removal               |
| `unlock` | Unlock a worktree                              |
| `prune`  | Remove stale worktrees                         |
| `exec`   | Run command across worktrees                   |
| `config` | Manage configuration                           |
| `doctor` | Diagnose workspace issues                      |

<details>
<summary><code>grove clone &lt;url&gt; [directory]</code></summary>

Clone a repository into a Grove workspace.

**Flags:**

- `--branches <list>` — Comma-separated branches to create worktrees for
- `--shallow` — Shallow clone (depth=1)
- `-v, --verbose` — Show git output

**Examples:**

```bash
grove clone https://github.com/owner/repo
grove clone https://github.com/owner/repo my-project
grove clone https://github.com/owner/repo --branches main,develop
grove clone https://github.com/owner/repo/pull/123 # Clone and checkout PR
```

</details>

<details>
<summary><code>grove init new [directory]</code> / <code>grove init convert</code></summary>

Initialize a Grove workspace.

**Subcommands:**

- `new [directory]` — Create new workspace
- `convert` — Convert existing repository

**Flags (convert):**

- `--branches <list>` — Additional branches to create worktrees for
- `-v, --verbose` — Show git output

**Examples:**

```bash
grove init new my-project
grove init convert
grove init convert --branches develop,staging
```

</details>

<details>
<summary><code>grove add &lt;branch|#PR|ref&gt;</code></summary>

Add a worktree for a branch, pull request, or ref.

**Flags:**

- `-s, --switch` — Switch to worktree after creating
- `--base <branch>` — Create new branch from base instead of HEAD
- `--name <name>` — Custom directory name
- `-d, --detach` — Detached HEAD state

**Examples:**

```bash
grove add feat/auth
grove add feat/auth --switch
grove add --base main feat/auth
grove add                 #123                    # PR by number
grove add --detach v1.0.0 # Tag in detached HEAD
```

</details>

<details>
<summary><code>grove switch &lt;worktree&gt;</code></summary>

Switch to a worktree by directory or branch name.

Requires shell integration (see Setup section).

**Examples:**

```bash
grove switch main
grove switch feat-auth
grove switch feat/auth
```

</details>

<details>
<summary><code>grove list</code></summary>

List all worktrees with status.

**Flags:**

- `--fast` — Skip remote sync checks
- `--filter <status>` — Filter by: `dirty`, `ahead`, `behind`, `gone`, `locked`
- `--json` — JSON output
- `-v, --verbose` — Show paths and upstreams

**Examples:**

```bash
grove list
grove list --fast
grove list --filter dirty
grove list --filter ahead,behind
grove list --json
```

</details>

<details>
<summary><code>grove status</code></summary>

Show current worktree status.

**Flags:**

- `-v, --verbose` — Show all diagnostic sections
- `--json` — JSON output

**Examples:**

```bash
grove status
grove status --verbose
```

</details>

<details>
<summary><code>grove remove &lt;worktree&gt;</code></summary>

Remove a worktree.

**Flags:**

- `-f, --force` — Remove even if dirty or locked
- `--branch` — Also delete the branch

**Examples:**

```bash
grove remove feat-auth
grove remove feat-auth --branch
grove remove --force wip
```

</details>

<details>
<summary><code>grove move &lt;worktree&gt; &lt;new-branch&gt;</code></summary>

Rename a branch and its worktree.

**Examples:**

```bash
grove move feat/old feat/new
```

</details>

<details>
<summary><code>grove lock &lt;worktree&gt;</code></summary>

Lock a worktree to prevent removal.

**Flags:**

- `--reason <text>` — Reason for locking

**Examples:**

```bash
grove lock main
grove lock release --reason "Production release"
```

</details>

<details>
<summary><code>grove unlock &lt;worktree&gt;</code></summary>

Unlock a worktree.

**Examples:**

```bash
grove unlock feat-auth
```

</details>

<details>
<summary><code>grove prune</code></summary>

Remove worktrees with deleted upstream branches. Dry-run by default.

**Flags:**

- `--commit` — Actually remove (default is dry-run)
- `-f, --force` — Remove even if dirty, locked, or unpushed
- `--stale <duration>` — Include inactive worktrees (e.g., `30d`, `2w`)
- `--merged` — Include branches merged into default branch

**Examples:**

```bash
grove prune          # Dry-run
grove prune --commit # Actually remove
grove prune --stale 30d --commit
grove prune --merged --commit
```

</details>

<details>
<summary><code>grove exec [worktrees...] -- &lt;command&gt;</code></summary>

Execute a command in worktrees.

**Flags:**

- `-a, --all` — Execute in all worktrees
- `--fail-fast` — Stop on first failure

**Examples:**

```bash
grove exec --all -- npm install
grove exec main feat-auth -- git pull
grove exec --all --fail-fast -- go build
grove exec --all -- bash -c "npm install && npm test"
```

</details>

<details>
<summary><code>grove config &lt;subcommand&gt;</code></summary>

Manage configuration.

**Subcommands:**

- `list` — Show all settings
- `get <key>` — Get value
- `set <key> <value>` — Set value (requires `--shared` or `--global`)
- `unset <key>` — Remove setting (requires `--shared` or `--global`)
- `init` — Create `.grove.toml` template

**Flags:**

- `--shared` — Target `.grove.toml`
- `--global` — Target git config

**Examples:**

```bash
grove config list
grove config get preserve.patterns
grove config set --global plain true
grove config set --shared autolock.patterns "main,release/*"
grove config init
```

</details>

<details>
<summary><code>grove doctor</code></summary>

Diagnose workspace issues.

**Flags:**

- `--fix` — Auto-fix safe issues
- `--json` — JSON output
- `--perf` — Disk space analysis

**Detects:**

- Broken `.git` pointers
- Stale worktree entries
- Invalid `.grove.toml` syntax
- Stale lock files

**Examples:**

```bash
grove doctor
grove doctor --fix
grove doctor --perf
```

</details>

## ⚙ Configuration

Grove uses a two-layer configuration system:

- **`.grove.toml`** — Team settings (checked into repository)
- **Git config** (`~/.gitconfig`) — Personal settings

Run `grove config init` to create a `.grove.toml` template.

<details>
<summary>Default configuration</summary>

```toml
# Grove - Git worktree management
# https://github.com/sqve/grove

[preserve]
# Files to copy from the current worktree when creating a new one.
# Useful for environment files and local configuration that shouldn't be in git.
# Supports glob patterns. These patterns override git config grove.preserve.
patterns = [
  ".env",
  ".env.keys",
  ".env.local",
  ".env.*.local",
  ".envrc",
  ".grove.toml",
  "docker-compose.override.yml",
]

# Path segments to exclude from preservation.
# Files containing any of these path segments will be skipped.
exclude = [
  # Caches
  ".cache",
  "__pycache__",

  # Build output
  "build",
  "coverage",
  "dist",
  "out",
  "target",

  # Dependencies
  ".venv",
  "node_modules",
  "vendor",
  "venv",
]

[hooks]
# Shell commands to run after creating a worktree.
# Runs sequentially, stops on first failure.
# Examples: ["npm install"], ["go mod download", "make setup"]
add = []

[autolock]
# Branch patterns to automatically lock when creating worktrees.
# Locked worktrees are protected from accidental deletion.
# Supports exact matches and trailing /* wildcards (e.g., "release/*").
patterns = ["develop", "main", "master"]

# Use Nerd Font icons in output (when not in plain mode).
nerd_fonts = true

# Threshold for marking worktrees as stale (no commits within this period).
# Format: number followed by d (days), w (weeks), or m (months).
stale_threshold = "30d"

# Disable colors and symbols in output.
plain = false

# Enable debug logging.
debug = false
```

</details>

## 🤝 Contributing

Contributions welcome! See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.
