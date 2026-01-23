# Codebase Structure

**Analysis Date:** 2026-01-23

## Directory Layout

```
grove/
├── cmd/
│   └── grove/
│       ├── main.go              # Entry point, root command setup
│       └── commands/            # Subcommand implementations
├── internal/                    # Private packages
│   ├── config/                  # Configuration (git config + TOML)
│   ├── formatter/               # Output formatting for lists
│   ├── fs/                      # Filesystem utilities
│   ├── git/                     # Git CLI abstractions
│   ├── github/                  # GitHub/gh CLI integration
│   ├── hooks/                   # Shell hook execution
│   ├── logger/                  # Structured logging
│   ├── styles/                  # Terminal styling (lipgloss)
│   ├── testutil/                # Test helpers
│   │   └── git/                 # Git test fixtures
│   ├── version/                 # Build version info
│   └── workspace/               # Workspace management
├── bin/                         # Build output
├── coverage/                    # Test coverage reports
├── .changes/                    # Changie changelog entries
├── .github/                     # GitHub Actions workflows
├── go.mod                       # Go module definition
├── go.sum                       # Dependency checksums
├── Makefile                     # Build and dev commands
├── CHANGELOG.md                 # Release history
├── README.md                    # Project documentation
└── DESIGN.md                    # Workspace architecture docs
```

## Directory Purposes

**cmd/grove/commands/:**

- Purpose: Cobra command implementations
- Contains: One file per command (add.go, clone.go, list.go, etc.)
- Key files: `add.go` (most complex), `clone.go`, `list.go`, `status.go`

**internal/config/:**

- Purpose: Configuration loading and precedence
- Contains: Git config reading, TOML file parsing, global state
- Key files: `config.go` (global), `file.go` (TOML)

**internal/git/:**

- Purpose: Git CLI wrappers
- Contains: Worktree, branch, status, remote operations
- Key files: `git.go` (core), `worktree.go`, `branch.go`, `status.go`

**internal/workspace/:**

- Purpose: Grove workspace lifecycle
- Contains: Init, clone, convert, file preservation, locking
- Key files: `workspace.go` (main), `preserve.go`, `lock.go`

**internal/github/:**

- Purpose: GitHub integration via gh CLI
- Contains: PR fetching, URL parsing, auth checks
- Key files: `github.go`

**internal/fs/:**

- Purpose: Cross-platform filesystem helpers
- Contains: Path utilities, atomic writes, file copying
- Key files: `fs.go`

**internal/logger/:**

- Purpose: Colorized terminal output
- Contains: Log levels, spinner, formatted output
- Key files: `logger.go`

**internal/styles/:**

- Purpose: Lipgloss style definitions
- Contains: Color definitions, path rendering
- Key files: `styles.go`

**internal/formatter/:**

- Purpose: Worktree list formatting
- Contains: Row formatting, indicators, alignment
- Key files: `formatter.go`

**internal/hooks/:**

- Purpose: Execute user-defined shell hooks
- Contains: Hook runner, config loading
- Key files: `hooks.go`

**internal/testutil/:**

- Purpose: Shared test utilities
- Contains: Git repo scaffolding, test helpers
- Key files: `testutil.go`, `git/git.go`

## Key File Locations

**Entry Points:**

- `cmd/grove/main.go`: Application entry point

**Configuration:**

- `internal/config/config.go`: Global config struct and git config loading
- `internal/config/file.go`: TOML file loading (.grove.toml)
- `internal/config/grove.template.toml`: Template for `grove config init`

**Core Logic:**

- `internal/workspace/workspace.go`: Init, clone, convert operations
- `internal/git/worktree.go`: Worktree CRUD operations
- `internal/git/branch.go`: Branch operations and queries

**Testing:**

- `cmd/grove/commands/*_test.go`: Command integration tests
- `internal/*_test.go`: Unit tests alongside implementation
- `cmd/grove/script_test.go`: Testscript-based E2E tests

## Naming Conventions

**Files:**

- Lowercase with underscores: `worktree.go`, `file_test.go`
- Test files: `*_test.go`
- Platform-specific: `lock_unix.go`, `lock_windows.go`

**Directories:**

- Lowercase, single word: `commands`, `workspace`, `config`
- No pluralization inconsistency: `git` not `gits`

**Go Identifiers:**

- Public: PascalCase (`CreateWorktree`, `WorktreeInfo`)
- Private: camelCase (`runGitCommand`, `executeWithStderr`)
- Acronyms: preserve case (`PRRef`, `URL`, not `PrRef`)

## Where to Add New Code

**New Command:**

1. Create `cmd/grove/commands/<name>.go`
2. Implement `New<Name>Cmd() *cobra.Command`
3. Register in `cmd/grove/main.go`
4. Add test in `cmd/grove/commands/<name>_test.go`

**New Git Operation:**

1. Add to appropriate file in `internal/git/` (worktree.go, branch.go, etc.)
2. Follow pattern: validate inputs, build command, use `runGitCommand()`
3. Add unit test

**New Configuration Option:**

1. Add to `internal/config/config.go` Global struct and defaults
2. Add git config loading in `LoadFromGitConfig()`
3. Add TOML field to `internal/config/file.go` FileConfig struct
4. Update template in `internal/config/grove.template.toml`

**Utility Functions:**

- Filesystem: `internal/fs/fs.go`
- Git helpers: `internal/git/git.go`
- String/path manipulation: consider `internal/fs/` or inline

## Special Directories

**.bare/:**

- Purpose: Bare git repository in grove workspace
- Generated: Yes (by grove clone/init)
- Committed: No (gitignored in workspace)

**bin/:**

- Purpose: Compiled binary output
- Generated: Yes (by `make build`)
- Committed: No

**coverage/:**

- Purpose: Test coverage HTML reports
- Generated: Yes (by `make coverage`)
- Committed: No

**.gocache/:**

- Purpose: Go build cache (project-local)
- Generated: Yes
- Committed: No

**.changes/:**

- Purpose: Changie changelog entries
- Generated: Partially (by `make change`)
- Committed: Yes

---

_Structure analysis: 2026-01-23_
