# Implementation Plan

## Task Overview

The implementation leverages Grove's existing infrastructure extensively, building the remove command by extending established patterns and reusing ~90% of existing functionality. Tasks focus on creating the CLI interface, safety validation logic, and comprehensive testing while maintaining Grove's high code quality standards (90%+ test coverage).

## Steering Document Compliance

All tasks follow established conventions:

- **structure.md**: Commands in `/internal/commands/remove/`, consistent file naming, centralized test utilities
- **tech.md**: Go best practices, Cobra CLI framework, testify testing, comprehensive error handling
- **product.md**: Safety-first approach with "zero accidental data loss" principle

## Task Format Guidelines

- Use checkbox format: `- [ ] Task number. Task description`
- Include implementation details as bullet points
- Reference requirements using: `_Requirements: X.Y, Z.A_`
- Reference existing code to leverage using: `_Leverage: path/to/file.go, path/to/component.go_`
- Focus only on coding tasks (no deployment, user testing, etc.)

## Tasks

- [x]   1. Create remove command structure and interfaces
    - Create `/internal/commands/remove/` directory
    - Define core interfaces for RemoveService, SafetyChecker, BranchManager
    - Create remove command entry point with basic Cobra structure
    - Add command registration to main.go
    - _Leverage: internal/commands/create/create.go, internal/commands/list/list.go, cmd/grove/main.go_
    - _Requirements: 1.1, 5.1_

- [x]   2. Implement data models and options structures
    - Define RemoveOptions, BulkCriteria, SafetyReport data structures
    - Create BranchSafetyStatus, RemoveResults, RemoveSkip, RemoveFailure structs
    - Add validation methods for all option structures
    - Write unit tests for data models and validation
    - _Leverage: internal/commands/create/options.go, internal/git/worktree.go (WorktreeInfo struct)_
    - _Requirements: 1.1, 2.1, 3.1, 4.1, 5.1_

- [x]   3. Create safety validation system
    - Build SafetyChecker component with uncommitted change detection
    - Implement current worktree validation (prevent removing active worktree)
    - Add path validation and existence checking
    - Create comprehensive safety reporting with actionable warnings
    - Write extensive unit tests for all safety scenarios
    - _Leverage: internal/git/worktree.go (getWorktreeStatus, getCurrentWorktreePath), internal/validation/filesystem.go_
    - _Requirements: 1.1, 1.2, 1.4, 5.1, 5.2_

- [x]   4. Implement intelligent branch safety logic
    - Create BranchManager with smart branch deletion logic
    - Implement automatic deletion for merged branches and pushed branches
    - Add confirmation prompts for unmerged, unpushed branches
    - Build remote branch deletion capabilities
    - Write comprehensive unit tests for branch safety determination
    - _Leverage: internal/git/worktree.go (getRemoteStatus, RemoteStatus struct), internal/git/operations.go_
    - _Requirements: 4.2, 4.3, 4.4, 4.5_

- [x]   5. Build core remove service implementation
    - Create RemoveService with single worktree removal logic
    - Implement dry-run functionality with detailed previews
    - Add force removal with comprehensive safety bypasses
    - Integrate with existing RemoveWorktree function from git package
    - Create result reporting and summary generation
    - Write unit tests for core removal logic
    - _Leverage: internal/git/worktree.go (RemoveWorktree function), internal/commands/shared/executor.go_
    - _Requirements: 1.1, 1.3, 1.5, 2.1, 2.3_

- [x]   6. Implement bulk removal operations
    - Add merged worktree identification and removal
    - Create stale worktree detection based on last activity timestamps
    - Implement bulk removal with progress reporting and summaries
    - Add "remove all" functionality with confirmation prompts
    - Write integration tests for bulk operations
    - _Leverage: internal/git/worktree.go (ListWorktrees, WorktreeInfo.LastActivity, RemoteStatus.IsMerged)_
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x]   7. Create CLI command interface and flag handling
    - Implement main remove command with comprehensive flag support
    - Add --force, --dry-run, --delete-branch, --merged, --stale, --all flags
    - Create input validation and path resolution
    - Add shell completion for worktree paths
    - Write CLI interaction unit tests
    - _Leverage: internal/commands/create/create.go (flag patterns), internal/completion/worktree.go_
    - _Requirements: 1.1, 2.1, 3.1, 4.1_

- [x]   8. Implement comprehensive error handling
    - Create user-friendly error messages for all failure scenarios
    - Add specific error handling for permissions, not found, current worktree
    - Implement partial failure reporting for bulk operations
    - Add error recovery and cleanup for interrupted operations
    - Write error scenario unit tests
    - _Leverage: internal/errors/errors.go, internal/validation/error_enhancer.go_
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [x]   9. Add result presentation and user feedback
    - Create clear success/failure messaging with operation summaries
    - Implement progress indicators for bulk operations
    - Add detailed dry-run output with action previews
    - Build confirmation prompts for destructive operations
    - Write presentation logic unit tests
    - _Leverage: internal/commands/list/list_presenter.go, internal/commands/shared/worktree_formatter.go_
    - _Requirements: 1.4, 2.2, 3.4_

- [ ]   10. Write comprehensive integration tests
    - Create end-to-end removal flow tests with real Git repositories
    - Test all flag combinations and safety scenarios
    - Add bulk operation integration tests with multiple worktrees
    - Test error scenarios and recovery behavior
    - Verify cross-platform compatibility
    - _Leverage: internal/commands/create/create_integration_test.go, internal/testutils/fixtures.go_
    - _Requirements: All requirements_

- [ ]   11. Add performance optimizations and validation
    - Implement parallel safety checks for bulk operations
    - Add performance benchmarks for large repository scenarios
    - Optimize memory usage for processing many worktrees
    - Validate <2s response time requirements
    - Write performance benchmark tests
    - _Leverage: internal/commands/create/path_generator_bench_test.go, internal/git/operations_bench_test.go_
    - _Requirements: Non-functional performance requirements_

- [ ]   12. Final integration and testing cleanup
    - Integrate remove command with main CLI application
    - Run comprehensive test suite and ensure 90%+ coverage
    - Fix any integration issues with existing commands
    - Update completion system registration
    - Verify all requirements are met through testing
    - _Leverage: cmd/grove/main.go, internal/completion/cache.go, internal/testutils/test_helpers.go_
    - _Requirements: All requirements_
