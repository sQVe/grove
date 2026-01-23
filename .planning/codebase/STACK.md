# Technology Stack

**Analysis Date:** 2026-01-23

## Languages

**Primary:**

- Go 1.25.5 - All application code (`cmd/`, `internal/`)

**Secondary:**

- Shell (POSIX sh) - Install script (`install.sh`)
- YAML - CI/CD workflows, configuration

## Runtime

**Environment:**

- Go 1.25.5 (specified in `go.mod`, `.github/workflows/ci.yml`)
- CGO disabled for all release builds (`CGO_ENABLED=0`)

**Package Manager:**

- Go modules (`go.mod`, `go.sum`)
- Lockfile: present (`go.sum`)

## Frameworks

**Core:**

- [cobra](https://github.com/spf13/cobra) v1.10.2 - CLI framework (`cmd/grove/main.go`)

**Terminal UI:**

- [lipgloss](https://github.com/charmbracelet/lipgloss) v1.1.0 - Terminal styling (`internal/styles/`)
- [termenv](https://github.com/muesli/termenv) v0.16.0 - Terminal detection

**Configuration:**

- [BurntSushi/toml](https://github.com/BurntSushi/toml) v1.6.0 - TOML config parsing (`internal/config/file.go`)

**Testing:**

- Go standard `testing` package
- [rogpeppe/go-internal](https://github.com/rogpeppe/go-internal) v1.14.1 - Script-based testing (`cmd/grove/script_test.go`)
- gotestsum v1.13.0 - Test runner (dev tool)

**Build/Dev:**

- golangci-lint v2.8.0 - Linting (`.golangci.yml`)
- goreleaser v2 - Release automation (`.goreleaser.yml`)
- changie v1.24.0 - Changelog management (`.changie.yaml`)
- prettier v3.5.0 - Markdown/JSON/YAML formatting (`package.json`)

## Key Dependencies

**Critical:**

- `github.com/spf13/cobra` v1.10.2 - Entire CLI structure depends on this
- `github.com/BurntSushi/toml` v1.6.0 - Config file parsing
- `golang.org/x/sys` v0.40.0 - OS-specific functionality (process detection in `internal/workspace/lock_unix.go`, `lock_windows.go`)

**Infrastructure:**

- `github.com/charmbracelet/lipgloss` v1.1.0 - All terminal output formatting
- `github.com/muesli/termenv` v0.16.0 - Terminal capability detection

**Indirect (auto-managed):**

- `github.com/spf13/pflag` v1.0.9 - Flag parsing (via cobra)
- `github.com/mattn/go-isatty` v0.0.20 - TTY detection (via termenv)

## Configuration

**Environment:**

- `GH_TOKEN` - GitHub CLI authentication for PR operations
- `CI` - Detected in Makefile for lint/test behavior

**Git Config (runtime):**

- `grove.plain` - Disable colors/symbols
- `grove.debug` - Enable debug logging
- `grove.nerdFonts` - Use Nerd Font icons
- `grove.staleThreshold` - Stale worktree detection (e.g., "30d")
- `grove.timeout` - Command timeout duration
- `grove.preserve` - File patterns to preserve
- `grove.preserveExclude` - Directories to exclude from preservation
- `grove.autoLock` - Branch patterns to auto-lock

**TOML Config (`.grove.toml`):**

- Per-workspace config file supporting same options as git config
- Parsed via `internal/config/file.go`

**Build:**

- `Makefile` - Build, test, lint targets
- `.goreleaser.yml` - Release configuration
- `.golangci.yml` - Linter configuration

## Platform Requirements

**Development:**

- Go 1.25.5+
- Git 2.48+ (for `--relative-paths` worktree flag)
- Node.js (for prettier, dev only)
- golangci-lint v2.8.0
- gotestsum v1.13.0
- changie v1.24.0

**Runtime:**

- Git 2.48+ (required)
- `gh` CLI (optional, for PR-related features)

**Production:**

- Platforms: linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
- Package formats: tar.gz, deb, rpm
- No runtime dependencies beyond git

## Version Injection

Build-time variables injected via ldflags (`.goreleaser.yml`):

- `internal/version.Version` - Semantic version
- `internal/version.Commit` - Git short commit hash
- `internal/version.Date` - Build date

---

_Stack analysis: 2026-01-23_
