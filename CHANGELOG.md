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

