# Implementation Plan

## Task Overview

The implementation will follow the established service-oriented architecture pattern used throughout Grove, with clear separation of concerns between CLI handling, business logic, and Git operations. The tasks prioritize leveraging existing infrastructure and patterns to ensure consistency and maintainability.

## Steering Document Compliance

**Structure.md Compliance:**

- Command implementation in `/internal/commands/create.go`
- Unit tests in `create_test.go`, integration tests in `create_integration_test.go`
- Error handling using centralized patterns from `/internal/errors/`
- Follow configuration hierarchy: CLI flags → env vars → config files → defaults

**Tech.md Compliance:**

- Use Cobra + Viper following existing command patterns
- Maintain 90%+ test coverage with testify framework
- Leverage Charm Bracelet Lipgloss for consistent terminal styling
- Implement proper error handling and retry mechanisms for robustness

## Tasks

- [x]   1. Set up project structure and core interfaces
    - Create `/internal/commands/create.go` with Cobra command structure
    - Define core data structures: `CreateOptions`, `CreateResult`, `BranchInfo`, `WorktreeOptions`
    - Set up command registration in `cmd/grove/main.go`
    - _Leverage: internal/commands/init.go:17-40, cmd/grove/main.go_
    - _Requirements: 1.1, 4.3_

- [x]   2. Implement CLI command interface and argument parsing
    - Add Cobra command with flags: `--create/-c`, `--base`, `--force`
    - Implement argument validation for branch names and paths
    - Add shell completion for branch names and paths using existing completion system
    - _Leverage: internal/completion/branch.go:11, internal/commands/init.go patterns_
    - _Requirements: 1.1, 1.2, 4.1, 6.1, 6.2, 6.3_

- [x]   3. Create enhanced BranchResolver service component
    - Implement `ResolveBranch()` method to detect local/remote branches
    - Add `ResolveURL()` method using existing `ParseGitPlatformURL()` function
    - Add `ResolveRemoteBranch()` method for origin/branch-name patterns
    - Add logic for prompting user when branch doesn't exist
    - Handle remote branch tracking, fetching, and local branch creation
    - _Leverage: internal/utils/git.go:185 ParseGitPlatformURL(), internal/git/operations.go:18-22_
    - _Requirements: 2.1, 2.2, 2.3, 2.5, 7.1, 7.2, 7.3, 7.4, 7.5, 8.1, 8.2, 8.3_

- [x]   4. Create PathGenerator service component
    - Implement `GeneratePath()` method for filesystem-safe path generation
    - Add collision detection and unique path generation with suffixes
    - Integrate with Grove configuration for base path preferences
    - _Leverage: internal/git/naming.go BranchToDirectoryName(), internal/config/defaults.go_
    - _Requirements: 3.1, 3.2, 3.3, 3.5, 5.1, 5.2_

- [ ]   5. Create WorktreeCreator service component
    - Implement `CreateWorktree()` method for Git worktree operations
    - Add support for new branch creation and remote tracking
    - Implement atomic operations with proper cleanup on failure
    - _Leverage: internal/git/worktree.go:90 CreateWorktreeWithSafeNaming(), internal/git/worktree.go:136 CreateWorktreeFromExistingBranch()_
    - _Requirements: 1.1, 1.3, 2.4, 2.6_

- [ ]   6. Create FileManager service component
    - Implement `CopyFiles()` method for pattern-based file copying
    - Add `DiscoverSourceWorktree()` method to find main/source worktree
    - Add `ResolveConflicts()` method for handling file conflicts
    - Implement conflict resolution strategies (prompt, skip, overwrite, backup)
    - _Leverage: internal/config patterns, filesystem operations from existing codebase_
    - _Requirements: 9.1, 9.2, 9.3, 9.4, 9.5, 9.6, 9.7, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6_

- [ ]   7. Create CreateService orchestration layer
    - Implement main `Create()` method that coordinates all components including FileManager
    - Add comprehensive input validation and error handling for URLs and file copying
    - Integrate with Grove configuration system for user preferences and file copying settings
    - Add input classification logic to determine if input is branch, URL, or remote branch
    - _Leverage: internal/config patterns, internal/errors/wrap.go_
    - _Requirements: 4.2, 4.4, 5.3, 5.4, 5.5_

- [ ]   8. Add enhanced error handling and validation
    - Define standardized error types: `ErrNotGroveRepository`, `ErrBranchNotFound`, `ErrUnsupportedURL`, etc.
    - Implement validation for repository state, branch names, URLs, and paths
    - Add actionable error messages with troubleshooting context for URL parsing failures
    - Add validation for file copying patterns and source worktree existence
    - _Leverage: internal/errors/ patterns, existing error handling utilities_
    - _Requirements: 4.1, 4.2, 4.3, 4.4, 7.7, 8.4, 8.5_

- [ ]   9. Implement enhanced configuration integration
    - Add configuration options for worktree base path and naming preferences
    - Add `[worktree.copy_files]` configuration section with patterns and conflict resolution
    - Support for default base branch, prompting preferences, and remote tracking
    - Implement configuration validation with sensible fallbacks for file copying settings
    - _Leverage: internal/config/ patterns, Viper configuration system_
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 10.1, 10.2, 10.3, 10.4, 10.5, 10.6_

- [ ]   10. Add enhanced CLI interface with new flags
    - Add `--copy-env` flag for quick environment file copying
    - Add `--copy "pattern1,pattern2"` flag for specific file patterns
    - Add `--no-copy` flag to disable file copying
    - Update help text and command documentation with new URL and file copying features
    - _Leverage: Cobra flag patterns from existing commands_
    - _Requirements: 9.4, 9.5, 9.6_

- [ ]   11. Add comprehensive unit tests
    - Test CreateService business logic with mocked dependencies including FileManager
    - Test enhanced BranchResolver with URL parsing and remote branch resolution
    - Test FileManager file copying logic with various conflict scenarios
    - Test CLI command argument parsing and new flag handling
    - _Leverage: internal/testutils/mocks.go, testify framework patterns_
    - _Requirements: All functional requirements_

- [ ]   12. Add integration tests
    - Test end-to-end create workflow with URL inputs in temporary repositories
    - Test actual Git worktree creation and branch operations from URLs
    - Test file copying functionality with real filesystems
    - Test configuration system integration and error scenarios for new features
    - _Leverage: internal/testutils/fixtures.go, existing integration test patterns_
    - _Requirements: Performance, reliability, and security requirements_

- [ ]   13. Add progress indication and enhanced user feedback
    - Implement progress display for longer operations (remote branch checkout, file copying)
    - Add success messages with worktree path, copied files summary, and next steps
    - Style output using Charm Bracelet Lipgloss for consistency
    - Add informative messages during URL parsing and remote resolution
    - _Leverage: Charm Bracelet Lipgloss, existing command output patterns_
    - _Requirements: 1.5, 4.5_

- [ ]   14. Optimize performance and add validation
    - Ensure operations complete within performance requirements (< 5s local, < 15s remote)
    - Add filesystem permission validation and security checks for file copying
    - Implement proper cleanup and atomic operations for both worktree and file operations
    - Optimize file copying for large numbers of files
    - **PathGenerator Optimizations (from code review)**:
        - Optimize collision resolution algorithm to reduce filesystem operations (current O(n) up to 999)
        - Enhance path traversal detection logic for better precision vs false positives
        - Make collision iteration limit configurable instead of hardcoded 999
        - Cache home directory lookup to avoid repeated `os.UserHomeDir()` calls
        - Add performance benchmarks for collision resolution scenarios
    - _Requirements: Performance, security, and reliability requirements_
