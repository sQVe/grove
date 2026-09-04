## [v1.10.0](https://github.com/sQVe/grove/releases/tag/v1.10.0) - 2026-09-04

### Added
- `grove list --json` now includes each worktree's last commit time, and `--filter` rejects unknown filter names.
- `grove list --sort recent` now orders worktrees by their latest commit in table and JSON output.
- Make grove exec print failed worktree names, propagate single-target exit codes, and support JSON output.

### Changed
- Internal worktree creation now uses one Git helper, one branch loop, and one auto-lock path. Dead types and duplicate helpers were removed, and status now reuses the shared worktree record. Commands now share workspace loading, target resolution, worktree completion, add finalization, and PR checkout. Remaining Git, filesystem, config, traversal, output, styling, and test-repository helpers now share their common implementations.

### Fixed
- `.grove.toml` now applies `autolock.patterns`, `nerd_fonts`, `stale_threshold`, `plain`, and `debug` at startup.
- `grove add` now rejects invalid names and detached refs, reports failed hooks, and `grove exec` requires `--` while passing stdin to commands.
- Worktree and ignored-file paths preserve newlines, detached worktrees remove safely and cannot move, Git errors propagate, and Windows device names gain a safe suffix.
- `grove clone` now defaults to `<cwd>/<repo-name>` when no directory is given and rejects `--branches` with a PR URL.
- README configuration examples now match the CLI, and `grove config list` includes every `.grove.toml` setting.
- `grove convert` now restores files before removing failed worktrees, and workspace locks remain active while their process is running.
- `grove doctor` now reads worktree config, keeps locked stale entries, and marks repaired issues as fixed in JSON output.
- `grove prune --force` now removes locked worktrees, and `grove remove --force --branch` warns before deleting commits not on another branch.
- Spinners now print a single plain `→ message` line instead of animating when stderr is not a terminal, so piped and CI output no longer contains cursor-control sequences. `--plain` still takes precedence.
- `grove add` now releases the workspace lock before running hooks, and lock contention waits up to 60 seconds for the other operation instead of failing immediately.

## [v1.9.0](https://github.com/sQVe/grove/releases/tag/v1.9.0) - 2026-07-01

### Changed
- Updated Go dependencies to their latest releases.

### Fixed
- Prune now re-points the bare repo's HEAD when it would otherwise dangle on a deleted branch, keeping subsequent `grove add` invocations working.
- `grove prune` now reaps git-prunable (path-gone) worktrees by default instead of skipping them with a "may be corrupted" warning. Live and locked worktrees are left untouched.

## [v1.8.1](https://github.com/sQVe/grove/releases/tag/v1.8.1) - 2026-04-13

### Fixed
- Show preserve, link, and hook output when using `grove add --switch` ([#91](https://github.com/sQVe/grove/issues/91))
- Apply `[link]` and `[preserve]` patterns when `grove add` runs outside the main worktree, by loading `.grove.toml` from any worktree containing it (preferring the default branch)
- Warn when a `[link]` target conflicts with git-tracked content in the destination worktree instead of silently skipping

## [v1.8.0](https://github.com/sQVe/grove/releases/tag/v1.8.0) - 2026-03-04

### Added
- Recursive directory preservation via `preserve.directories` config for copying entire directory trees to new worktrees

### Fixed
- Show preserve, link, and hook output when using `grove add --switch` ([#91](https://github.com/sQVe/grove/issues/91))

## [v1.7.0](https://github.com/sQVe/grove/releases/tag/v1.7.0) - 2026-02-25

### Added
- Symlink directories from source worktree on grove add via new [link] config section ([#87](https://github.com/sQVe/grove/issues/87))

### Fixed
- Use directory as worktree identifier in exec, lock, remove, and unlock commands

## [v1.6.0](https://github.com/sQVe/grove/releases/tag/v1.6.0) - 2026-02-03

### Added
- Fallback worktree selection by directory name when branch doesn't match, enabling file preservation when primary worktree is on a feature branch ([#79](https://github.com/sQVe/grove/issues/79))
- Show worktree directory alongside branch name in prune output for easier identification ([#81](https://github.com/sQVe/grove/issues/81))

## [v1.5.0](https://github.com/sQVe/grove/releases/tag/v1.5.0) - 2026-01-27

### Added
- `--json` and `--verbose` flags to `grove fetch` for structured output and commit hash details ([#61](https://github.com/sQVe/grove/issues/61))
- `grove fetch` command to sync all remotes and display new, updated, and pruned refs ([#71](https://github.com/sQVe/grove/issues/71))
- Spinner API with `Update`, `StopWithSuccess`, `StopWithError` methods and `StepFormat` helper for step/total formatting ([#73](https://github.com/sQVe/grove/issues/73))
- `grove doctor` checks remote accessibility and warns about unreachable remotes
- Actionable hints to error messages showing recovery commands
- Real-time hook output streaming during `grove add` with prefixed lines identifying each hook

### Changed
- `grove remove` shows summary counts instead of per-item messages ([#73](https://github.com/sQVe/grove/issues/73))
- Add spinners to show progress during network operations and improve output consistency across commands

## [v1.4.0](https://github.com/sQVe/grove/releases/tag/v1.4.0) - 2026-01-19

### Added
- Configure fetch refspec automatically during clone to enable remote branch tracking
- `grove doctor` checks Git and gh versions with upgrade guidance
- Add `--from` flag to `grove add` for specifying source worktree during file preservation
- Detect old Git versions and hint users to run `grove doctor`

### Fixed
- Configure upstream tracking automatically when creating worktrees for existing remote branches

## [v1.3.0](https://github.com/sQVe/grove/releases/tag/v1.3.0) - 2026-01-15

### Added
- Use relative paths for portable worktrees (requires Git 2.48+)

## [v1.2.1](https://github.com/sQVe/grove/releases/tag/v1.2.1) - 2026-01-15

### Fixed
- Normalize path separators for Windows compatibility

## [v1.2.0](https://github.com/sQVe/grove/releases/tag/v1.2.0) - 2026-01-08

### Added
- Add `--detached` flag to `grove prune` for detecting and removing worktrees with detached HEAD state ([#19](https://github.com/sQVe/grove/issues/19))
- Delete local branches when pruning worktrees with gone upstreams ([#20](https://github.com/sQVe/grove/issues/20))
- Add multi-worktree support to remove, lock, and unlock commands ([#21](https://github.com/sQVe/grove/issues/21))

### Changed
- Migrate build system from Mage to Make for simpler, faster builds ([#32](https://github.com/sQVe/grove/issues/32))

### Fixed
- Detect multi-commit squash merges via GitHub CLI when pruning
- Detect diverged PR branches and add --reset flag to sync with remote
- Preserve files when running `grove add` from workspace root by falling back to default branch worktree
- Handle squash-merged branches when pruning gone worktrees

## [v1.1.0](https://github.com/sQVe/grove/releases/tag/v1.1.0) - 2026-01-04

### Added
- Changelog management with changie and automated release workflow ([#11](https://github.com/sQVe/grove/issues/11))

### Fixed
- Output logger messages to stderr instead of stdout, fixing --switch flag functionality ([#14](https://github.com/sQVe/grove/issues/14))

## [v1.0.0](https://github.com/sQVe/grove/releases/tag/v1.0.0) - 2026-01-01

Initial release. See [GitHub release](https://github.com/sQVe/grove/releases/tag/v1.0.0) for full changelog.

