# Implementation Plan

## Task Overview

Implementation follows the vertical slice approach, starting with the GitCommander interface foundation and building up through unit testing infrastructure, then completing with integration testing framework. Each task is designed to be atomic, completable in 15-30 minutes, and focused on specific file modifications to enable optimal execution by development agents.

The implementation prioritizes establishing the core testing abstractions first (GitCommander interface), then applies these patterns to one representative command (CreateService), before expanding to the full testing infrastructure and remaining services.

## Steering Document Compliance

Tasks follow established patterns from structure.md and tech.md:

- **Package Organization**: New components placed in `/internal/testutils/` and `/test/integration/` as specified
- **File Naming**: Uses snake_case conventions and standard Go patterns (`*_test.go`, `*_integration_test.go`)
- **Testing Standards**: Maintains 90%+ coverage requirement and integrates with existing Mage build system
- **Architecture Patterns**: Follows existing dependency injection and service layer patterns
- **Code Quality**: Leverages existing error handling, logging, and retry mechanisms

## Atomic Task Requirements

**Each task meets these criteria for optimal agent execution:**

- **File Scope**: Touches 1-3 related files maximum
- **Time Boxing**: Completable in 15-30 minutes
- **Single Purpose**: One testable outcome per task
- **Specific Files**: Must specify exact files to create/modify
- **Agent-Friendly**: Clear input/output with minimal context switching

## Task Format Guidelines

- Use checkbox format: `- [ ] Task number. Task description`
- **Specify files**: Always include exact file paths to create/modify
- **Include implementation details** as bullet points
- Reference requirements using: `_Requirements: X.Y, Z.A_`
- Reference existing code to leverage using: `_Leverage: path/to/file.go_`
- Focus only on coding tasks (no deployment, user testing, etc.)
- **Avoid broad terms**: No "system", "integration", "complete" in task titles

## Comprehensive Test Specifications

### Critical Test Categories Overview

Based on consensus analysis, this section provides detailed specifications for all tests that must be implemented to ensure robust testing infrastructure. Each test category includes specific test cases, expected behaviors, and validation criteria.

### Unit Test Specifications

#### GitCommander Interface Tests (internal/git/commander_test.go)

**Core Functionality Tests:**

- `TestGitCommander_Run_Success` - Verify successful command execution with valid inputs
- `TestGitCommander_Run_WithArguments` - Test command execution with various argument combinations
- `TestGitCommander_Run_EmptyCommand` - Validate handling of empty command strings
- `TestGitCommander_RunQuiet_NoOutput` - Ensure quiet mode suppresses output appropriately
- `TestGitCommander_Run_LongRunningCommand` - Test behavior with commands that take time
- `TestGitCommander_Run_CommandWithSpaces` - Validate proper argument parsing with spaces

**Error Handling Tests:**

- `TestGitCommander_Run_InvalidCommand` - Test response to non-existent Git commands
- `TestGitCommander_Run_PermissionDenied` - Validate handling of permission errors
- `TestGitCommander_Run_NetworkTimeout` - Test behavior when Git operations timeout
- `TestGitCommander_Run_CorruptedRepository` - Handle corrupted Git repository scenarios
- `TestGitCommander_Run_DiskSpaceError` - Test behavior when disk space is insufficient
- `TestGitCommander_Run_InterruptedOperation` - Validate handling of interrupted Git commands

**Context and Cancellation Tests:**

- `TestGitCommander_Run_WithContext` - Verify context propagation through Git operations
- `TestGitCommander_Run_ContextCancellation` - Test graceful handling of context cancellation
- `TestGitCommander_Run_ContextTimeout` - Validate timeout behavior with context deadlines
- `TestGitCommander_Run_ContextValues` - Ensure context values are preserved across calls

**Logging and Observability Tests:**

- `TestGitCommander_Run_LogsCommands` - Verify all Git commands are properly logged
- `TestGitCommander_Run_LogsExecutionTime` - Validate performance metrics logging
- `TestGitCommander_Run_LogsErrorDetails` - Ensure error details are captured in logs
- `TestGitCommander_Run_StructuredLogging` - Test structured log format compliance

#### CreateService Tests (internal/commands/create/create_service_test.go)

**Branch Creation Tests:**

- `TestCreateService_CreateBranch_Success` - Verify successful branch creation flow
- `TestCreateService_CreateBranch_ExistingBranch` - Handle existing branch scenarios
- `TestCreateService_CreateBranch_InvalidName` - Validate branch name validation rules
- `TestCreateService_CreateBranch_ReservedNames` - Test handling of Git reserved names
- `TestCreateService_CreateBranch_UnicodeNames` - Validate Unicode branch name support
- `TestCreateService_CreateBranch_LongNames` - Test branch name length limits

**Remote Repository Tests:**

- `TestCreateService_CloneRepository_Success` - Verify successful repository cloning
- `TestCreateService_CloneRepository_InvalidURL` - Handle malformed repository URLs
- `TestCreateService_CloneRepository_AuthenticationFailed` - Test authentication error handling
- `TestCreateService_CloneRepository_NetworkError` - Validate network failure recovery
- `TestCreateService_CloneRepository_LargeRepository` - Test handling of large repositories
- `TestCreateService_CloneRepository_ShallowClone` - Verify shallow clone optimization

**Rollback and Cleanup Tests:**

- `TestCreateService_Rollback_PartialFailure` - Test rollback on partial operation failure
- `TestCreateService_Rollback_FileSystemError` - Handle file system errors during rollback
- `TestCreateService_Rollback_ConcurrentAccess` - Test rollback with concurrent operations
- `TestCreateService_Cleanup_TemporaryFiles` - Verify temporary file cleanup
- `TestCreateService_Cleanup_PartialDirectories` - Handle cleanup of partial directory structures
- `TestCreateService_Cleanup_PermissionErrors` - Test cleanup with permission issues

**Validation Tests:**

- `TestCreateService_ValidateInput_RequiredFields` - Verify required field validation
- `TestCreateService_ValidateInput_PathTraversal` - Prevent path traversal attacks
- `TestCreateService_ValidateInput_SpecialCharacters` - Handle special characters in inputs
- `TestCreateService_ValidateInput_MaxLengths` - Test input length limitations
- `TestCreateService_ValidateInput_Encoding` - Validate character encoding handling

#### BranchResolver Tests (internal/commands/create/branch_resolver_test.go)

**Remote Detection Tests:**

- `TestBranchResolver_DetectRemote_GitHubURL` - Parse GitHub repository URLs correctly
- `TestBranchResolver_DetectRemote_GitLabURL` - Handle GitLab URL variations
- `TestBranchResolver_DetectRemote_BitbucketURL` - Support Bitbucket URL formats
- `TestBranchResolver_DetectRemote_SSHFormat` - Parse SSH-format URLs properly
- `TestBranchResolver_DetectRemote_CustomGitServer` - Handle custom Git server URLs
- `TestBranchResolver_DetectRemote_IPv6Addresses` - Support IPv6 in Git URLs

**URL Parsing Tests:**

- `TestBranchResolver_ParseURL_HTTPSUrls` - Parse HTTPS Git URLs correctly
- `TestBranchResolver_ParseURL_SSHUrls` - Handle SSH protocol URLs
- `TestBranchResolver_ParseURL_FileUrls` - Support local file:// URLs
- `TestBranchResolver_ParseURL_RelativePaths` - Handle relative path URLs
- `TestBranchResolver_ParseURL_SpecialCharacters` - Parse URLs with special characters
- `TestBranchResolver_ParseURL_URLEncoding` - Handle URL-encoded characters

**Network Error Handling Tests:**

- `TestBranchResolver_Resolve_ConnectionTimeout` - Handle connection timeouts gracefully
- `TestBranchResolver_Resolve_DNSFailure` - Test DNS resolution failure handling
- `TestBranchResolver_Resolve_ProxyErrors` - Validate proxy configuration error handling
- `TestBranchResolver_Resolve_CertificateErrors` - Handle SSL certificate validation errors
- `TestBranchResolver_Resolve_RateLimiting` - Test API rate limiting scenarios
- `TestBranchResolver_Resolve_TemporaryFailures` - Handle temporary network failures

#### WorktreeCreator Tests (internal/commands/create/worktree_creator_test.go)

**Atomic Operation Tests:**

- `TestWorktreeCreator_Create_AtomicSuccess` - Verify atomic worktree creation
- `TestWorktreeCreator_Create_AtomicRollback` - Test rollback on creation failure
- `TestWorktreeCreator_Create_ConcurrentCreation` - Handle concurrent worktree creation
- `TestWorktreeCreator_Create_FileSystemConsistency` - Verify file system consistency
- `TestWorktreeCreator_Create_SymlinkHandling` - Test symlink creation and validation
- `TestWorktreeCreator_Create_PermissionValidation` - Validate file permissions

**Cleanup and Recovery Tests:**

- `TestWorktreeCreator_Cleanup_PartialWorktree` - Clean up partially created worktrees
- `TestWorktreeCreator_Cleanup_LockedFiles` - Handle locked files during cleanup
- `TestWorktreeCreator_Cleanup_CrossPlatformPaths` - Test cleanup across platforms
- `TestWorktreeCreator_Recovery_CorruptedState` - Recover from corrupted worktree state
- `TestWorktreeCreator_Recovery_InterruptedCreation` - Handle interrupted creation process
- `TestWorktreeCreator_Recovery_DiskFullDuringCreation` - Recover from disk space errors

### Integration Test Specifications

#### CLI Command Tests (test/integration/testdata/)

**Create Command Integration Tests:**

- `create_basic.txt` - Basic worktree creation with default settings
- `create_with_remote.txt` - Create worktree from remote repository
- `create_with_branch.txt` - Create worktree with specific branch
- `create_existing_directory.txt` - Handle existing directory conflicts
- `create_invalid_repository.txt` - Handle invalid repository URLs
- `create_permission_denied.txt` - Test permission error scenarios
- `create_network_failure.txt` - Handle network connectivity issues
- `create_large_repository.txt` - Test performance with large repositories

**List Command Integration Tests:**

- `list_empty_workspace.txt` - List worktrees in empty workspace
- `list_multiple_worktrees.txt` - Display multiple worktree entries
- `list_with_formatting.txt` - Test output formatting options
- `list_with_filters.txt` - Apply filtering criteria to worktree list
- `list_corrupted_worktree.txt` - Handle corrupted worktree entries
- `list_permission_errors.txt` - Handle permission denied scenarios

**Config Command Integration Tests:**

- `config_set_values.txt` - Set configuration values successfully
- `config_get_values.txt` - Retrieve configuration values correctly
- `config_invalid_keys.txt` - Handle invalid configuration keys
- `config_file_permissions.txt` - Test configuration file permission handling
- `config_environment_override.txt` - Test environment variable overrides
- `config_migration_scenarios.txt` - Handle configuration migration

#### Cross-Platform Integration Tests:\*\*

- `cross_platform_paths.txt` - Test path handling across Windows/Linux/macOS
- `cross_platform_permissions.txt` - Validate permission models across platforms
- `cross_platform_symlinks.txt` - Test symlink behavior across platforms
- `cross_platform_line_endings.txt` - Handle line ending differences
- `cross_platform_case_sensitivity.txt` - Test case sensitivity differences
- `cross_platform_unicode.txt` - Validate Unicode filename support

#### Environment Isolation Tests:\*\*

- `environment_home_directory.txt` - Test HOME directory isolation
- `environment_config_paths.txt` - Validate XDG config directory handling
- `environment_git_config.txt` - Test Git configuration isolation
- `environment_ssh_keys.txt` - Handle SSH key isolation in tests
- `environment_proxy_settings.txt` - Test proxy configuration isolation
- `environment_cleanup.txt` - Verify complete environment cleanup

### Performance and Load Tests

#### Performance Benchmarks:

- `BenchmarkGitCommander_Run` - Benchmark basic Git command execution
- `BenchmarkCreateService_CreateWorktree` - Benchmark worktree creation performance
- `BenchmarkBranchResolver_ParseURL` - Benchmark URL parsing performance
- `BenchmarkIntegration_LargeRepository` - Test with repositories >1GB
- `BenchmarkIntegration_ManyWorktrees` - Test with 100+ worktrees
- `BenchmarkIntegration_ConcurrentOperations` - Test concurrent command execution

#### Load Testing Scenarios:

- `TestLoad_ConcurrentCreateOperations` - 10+ simultaneous worktree creations
- `TestLoad_HighVolumeCommands` - 1000+ sequential Git commands
- `TestLoad_MemoryPressure` - Test under memory-constrained conditions
- `TestLoad_DiskPressure` - Test with limited disk space
- `TestLoad_NetworkLatency` - Test with high network latency simulation
- `TestLoad_ResourceExhaustion` - Test resource limit handling

### Error Scenario and Edge Case Tests

#### File System Edge Cases:

- `TestEdgeCase_ReadOnlyFileSystem` - Handle read-only file system
- `TestEdgeCase_SymlinkCycles` - Detect and handle symlink cycles
- `TestEdgeCase_VeryLongPaths` - Test with maximum path length
- `TestEdgeCase_SpecialFileNames` - Handle special characters in filenames
- `TestEdgeCase_CaseSensitivity` - Test case sensitivity edge cases
- `TestEdgeCase_UnicodeNormalization` - Handle Unicode normalization issues

#### Git Repository Edge Cases:

- `TestEdgeCase_EmptyRepository` - Handle completely empty repositories
- `TestEdgeCase_BareRepository` - Work with bare Git repositories
- `TestEdgeCase_ShallowRepository` - Handle shallow clone repositories
- `TestEdgeCase_LargeFiles` - Test with Git LFS and large files
- `TestEdgeCase_ManyBranches` - Handle repositories with 1000+ branches
- `TestEdgeCase_DeepHistory` - Test with very deep commit history

#### Network Edge Cases:

- `TestEdgeCase_SlowNetwork` - Handle very slow network connections
- `TestEdgeCase_IntermittentConnectivity` - Deal with connection drops
- `TestEdgeCase_ProxyAuthentication` - Handle proxy authentication
- `TestEdgeCase_IPv6OnlyNetwork` - Test IPv6-only network environments
- `TestEdgeCase_FirewallBlocking` - Handle blocked ports/protocols
- `TestEdgeCase_CertificateExpiry` - Handle expired SSL certificates

### Infrastructure Validation Tests

#### TestUtils Validation:

- `TestTestUtils_MockSetup` - Verify mock setup and teardown
- `TestTestUtils_AssertionHelpers` - Validate custom assertion functions
- `TestTestUtils_GitRepoHelpers` - Test Git repository helper functions
- `TestTestUtils_CleanupHandling` - Verify proper test cleanup
- `TestTestUtils_CrossPlatformHelpers` - Test cross-platform utilities
- `TestTestUtils_FixtureManagement` - Validate test fixture handling

#### Build System Integration:

- `TestBuild_UnitTestSeparation` - Verify unit tests run independently
- `TestBuild_IntegrationTestSeparation` - Confirm integration test isolation
- `TestBuild_CoverageAccuracy` - Validate coverage reporting accuracy
- `TestBuild_ParallelExecution` - Test parallel test execution
- `TestBuild_TaggedExecution` - Verify build tag functionality
- `TestBuild_ContinuousIntegration` - Test CI pipeline integration

#### Documentation Accuracy Tests:

- `TestDocs_CodeExamples` - Verify all code examples compile and run
- `TestDocs_LinkValidation` - Check all documentation links are valid
- `TestDocs_APIDocumentation` - Ensure API docs match implementation
- `TestDocs_SetupInstructions` - Validate setup instructions work correctly
- `TestDocs_TroubleshootingGuides` - Test troubleshooting scenarios
- `TestDocs_VersionCompatibility` - Verify version compatibility information

## Tasks

### Phase 1: Core Testing Infrastructure

- [x]   1. Create GitCommander interface in internal/git/commander.go
    - File: internal/git/commander.go
    - Define Commander interface with Run() and RunQuiet() methods
    - Include proper error handling and context support
    - Add comprehensive docstrings for interface methods
    - Purpose: Establish Git operation abstraction for dependency injection
    - _Leverage: internal/git/operations.go_
    - _Requirements: 2.1, 2.2_

- [x]   2. Implement LiveGitCommander in internal/git/commander.go
    - File: internal/git/commander.go (continue from task 1)
    - Create LiveGitCommander struct implementing Commander interface
    - Use os/exec for real Git command execution
    - Add structured logging for Git operations
    - Purpose: Provide production Git command execution
    - _Leverage: internal/git/operations.go, internal/logger/logger.go_
    - _Requirements: 2.2, 2.5_

- [x]   3. Create testutils package structure in internal/testutils/
    - Files: internal/testutils/git.go, internal/testutils/mocks.go
    - Create Git repository helper functions (NewTestRepo, NewTestRepoWithCommit)
    - Add MockGitCommander using testify/mock
    - Include cleanup and assertion utilities
    - Purpose: Provide centralized testing utilities for Git operations
    - _Leverage: existing testify usage_
    - _Requirements: 8.1, 8.2_

- [x]   4. Add testutils convenience functions in internal/testutils/helpers.go
    - File: internal/testutils/helpers.go
    - Implement AssertGitState function for repository state validation
    - Add CreateMockGitCommander factory function
    - Include cross-platform path handling utilities
    - Purpose: Provide standardized test assertion and setup helpers
    - _Leverage: internal/testutils/git.go, internal/testutils/mocks.go_
    - _Requirements: 8.3, 6.1_

### Phase 2: Vertical Slice Implementation

- [x]   5. Refactor CreateService for dependency injection in internal/commands/create/create_service.go
    - File: internal/commands/create/create_service.go
    - Modify CreateServiceImpl to accept GitCommander interface
    - Update NewCreateService constructor to accept GitCommander parameter
    - Replace direct git.DefaultExecutor usage with injected GitCommander
    - Purpose: Enable GitCommander injection for testing
    - _Leverage: internal/commands/create/create_service.go, internal/git/commander.go_
    - _Requirements: 2.2, 2.3_

- [x]   6. Create command factory pattern in internal/commands/create/create.go
    - File: internal/commands/create/create.go (modify existing NewCreateCmd)
    - Create App struct for dependency injection
    - Refactor NewCreateCmd to accept App parameter with GitCommander
    - Update command construction to use injected dependencies
    - Purpose: Enable clean dependency injection for testing
    - _Leverage: internal/commands/create/create.go, internal/git/commander.go_
    - _Requirements: 2.2, 7.1_

- [x]   7. Create CreateService unit tests in internal/commands/create/create_service_test.go
    - File: internal/commands/create/create_service_test.go
    - Write unit tests using MockGitCommander for CreateService methods
    - Test both success and error scenarios with mocked Git responses
    - Follow Arrange-Act-Assert pattern with clear section comments
    - Purpose: Validate CreateService business logic without Git dependencies
    - _Leverage: internal/testutils/mocks.go, internal/testutils/helpers.go_
    - _Requirements: 1.1, 1.4, 5.1_

- [x]   8. Create CreateCmd unit tests in internal/commands/create/create_test.go
    - File: internal/commands/create/create_test.go
    - Write unit tests for command parsing and validation using mocked dependencies
    - Test error handling and user message formatting
    - Include tests for flag parsing and argument validation
    - Purpose: Validate command-level logic and user interface
    - _Leverage: internal/testutils/mocks.go, internal/commands/create/create_service_test.go_
    - _Requirements: 1.1, 1.3, 5.2_

### Phase 3: Integration Testing Framework

- [x]   9. Create testscript infrastructure in test/integration/integration_test.go
    - File: test/integration/integration_test.go
    - Add //go:build integration build tag
    - Implement TestMain with testscript.RunMain for Grove binary
    - Create TestCLI function using testscript.Run with testdata directory
    - Purpose: Establish testscript-based integration testing framework
    - _Leverage: cmd/grove/main.go_
    - _Requirements: 3.1, 3.2, 7.2_

- [x]   10. Add testscript dependency and go.mod updates
    - Files: go.mod, go.sum
    - Add github.com/rogpeppe/go-internal/testscript dependency
    - Update go.mod with required testscript version
    - Run go mod tidy to update dependencies
    - Purpose: Enable testscript framework usage in integration tests
    - _Leverage: existing go.mod_
    - _Requirements: 3.1_

- [x]   11. Create basic testscript scenarios in test/integration/testdata/
    - Files: test/integration/testdata/create_basic.txt, test/integration/testdata/create_error.txt
    - Create txtar test scripts for grove create command scenarios
    - Include both success and error cases with Git repository setup
    - Test command output and resulting Git repository state
    - Purpose: Validate end-to-end CLI behavior with real Git operations
    - _Leverage: internal/commands/create/ patterns_
    - _Requirements: 3.2, 3.3, 6.2_

- [x]   12. Create environment isolation helpers in test/integration/testdata/helpers.txt
    - File: test/integration/testdata/helpers.txt
    - Add testscript helper functions for Git repository setup
    - Include environment variable isolation (HOME, XDG_CONFIG_HOME)
    - Create reusable Git state setup and validation functions
    - Purpose: Provide consistent test environment setup across integration tests
    - _Leverage: internal/testutils/git.go patterns_
    - _Requirements: 4.4, 6.1_

### Phase 4: Build System Integration

- [x]   13. Update Mage build system in magefile.go
    - File: magefile.go (modify existing Test methods)
    - Add build tag support to separate unit and integration tests
    - Update test:unit to exclude integration tests
    - Update test:integration to run only integration tests with build tags
    - Purpose: Enable separate execution of unit and integration test suites
    - _Leverage: existing magefile.go Test namespace_
    - _Requirements: 7.1, 7.2_

- [x]   14. Add coverage configuration in magefile.go
    - File: magefile.go (modify existing Coverage method)
    - Configure coverage collection for unit tests only
    - Add coverage exclusions for test files and generated code
    - Update coverage reporting to maintain 90%+ target
    - Purpose: Provide accurate coverage metrics for unit tests
    - _Leverage: existing magefile.go Coverage method_
    - _Requirements: 5.1, 7.4_

### Phase 5: Service Layer Expansion

- [x]   15. Refactor BranchResolver for GitCommander in internal/commands/create/branch_resolver.go
    - File: internal/commands/create/branch_resolver.go
    - Update BranchResolver to accept GitCommander interface
    - Replace direct git executor usage with injected GitCommander
    - Update NewBranchResolver constructor for dependency injection
    - Purpose: Enable GitCommander injection for BranchResolver testing
    - _Leverage: internal/commands/create/branch_resolver.go, internal/git/commander.go_
    - _Requirements: 2.2, 2.3_

- [x]   16. Create BranchResolver unit tests in internal/commands/create/branch_resolver_test.go
    - File: internal/commands/create/branch_resolver_test.go
    - Write unit tests using MockGitCommander for branch resolution methods
    - Test remote branch detection and URL parsing with mocked Git responses
    - Include error handling tests for network and Git failures
    - Purpose: Validate BranchResolver logic without Git dependencies
    - _Leverage: internal/testutils/mocks.go, internal/testutils/helpers.go_
    - _Requirements: 1.1, 1.4, 5.2_

- [x]   17. Refactor WorktreeCreator for GitCommander in internal/commands/create/worktree_creator.go
    - File: internal/commands/create/worktree_creator.go
    - Update WorktreeCreatorImpl to accept GitCommander interface
    - Replace direct git executor usage with injected GitCommander
    - Update NewWorktreeCreator constructor for dependency injection
    - Purpose: Enable GitCommander injection for WorktreeCreator testing
    - _Leverage: internal/commands/create/worktree_creator.go, internal/git/commander.go_
    - _Requirements: 2.2, 2.3_

- [x]   18. Create WorktreeCreator unit tests in internal/commands/create/worktree_creator_test.go
    - File: internal/commands/create/worktree_creator_test.go
    - Write unit tests using MockGitCommander for worktree creation methods
    - Test rollback scenarios and error handling with mocked Git failures
    - Include atomic operation testing and cleanup validation
    - Purpose: Validate WorktreeCreator complex logic without filesystem dependencies
    - _Leverage: internal/testutils/mocks.go, internal/testutils/helpers.go_
    - _Requirements: 1.1, 1.4, 5.3_

### Phase 6: Comprehensive Test Coverage

- [x]   19. Create additional integration test scenarios in test/integration/testdata/
    - Files: test/integration/testdata/list_worktrees.txt, test/integration/testdata/config_commands.txt
    - Add testscript scenarios for list and config commands
    - Include cross-platform compatibility tests
    - Test configuration file handling and environment variable isolation
    - Purpose: Expand integration test coverage to additional CLI commands
    - _Leverage: test/integration/testdata/create_basic.txt patterns_
    - _Requirements: 3.4, 6.1, 6.2_

- [x]   20. Add performance and error scenario tests in test/integration/testdata/
    - Files: test/integration/testdata/large_repo.txt, test/integration/testdata/network_errors.txt
    - Create tests for large repository performance
    - Add network failure simulation and recovery testing
    - Include Git operation timeout and retry scenario tests
    - Purpose: Validate Grove behavior under stress and error conditions
    - _Leverage: internal/retry/retry.go patterns_
    - _Requirements: 4.1, 4.2, 6.3_

### Phase 7: Documentation and Finalization

- [x]   21. Create testing documentation in docs/testing.md
    - File: docs/testing.md
    - Document unit test patterns and GitCommander usage
    - Include testscript integration test examples and debugging guide
    - Add developer workflow documentation for running different test types
    - Purpose: Provide comprehensive testing documentation for developers
    - _Leverage: existing documentation patterns_
    - _Requirements: 7.4, 7.5_

- [x]   22. Update README with testing information
    - File: README.md (modify existing testing section)
    - Add testing infrastructure overview and quick start guide
    - Include examples of running unit and integration tests
    - Document coverage requirements and contribution guidelines
    - Purpose: Provide immediate testing information for new contributors
    - _Leverage: existing README.md structure_
    - _Requirements: 7.5_

### Phase 8: Comprehensive Test Implementation

#### Unit Test Implementation Tasks

- [x]   23. Implement GitCommander interface tests in internal/git/commander_test.go
    - File: internal/git/commander_test.go
    - Implement all 20 GitCommander test cases from specifications
    - Focus on core functionality, error handling, context, and logging tests
    - Use testify/mock for clean mock implementations (avoid previous manual mocking patterns)
    - Purpose: Validate GitCommander interface with comprehensive coverage
    - _Leverage: internal/git/commander.go interface design_
    - _Requirements: 1.1, 2.2, 5.1_

- [x]   24. Implement CreateService tests in internal/commands/create/create_service_test.go
    - File: internal/commands/create/create_service_test.go
    - Implement all 23 CreateService test cases from specifications
    - Cover branch creation, remote repository, rollback, and validation scenarios
    - Use MockGitCommander for clean dependency injection (avoid tight coupling)
    - Purpose: Validate CreateService business logic without external dependencies
    - _Leverage: internal/testutils/mocks.go when available_
    - _Requirements: 1.1, 1.4, 5.1_

- [x]   25. Implement BranchResolver tests in internal/commands/create/branch_resolver_test.go
    - File: internal/commands/create/branch_resolver_test.go
    - Implement all 18 BranchResolver test cases from specifications
    - Cover remote detection, URL parsing, and network error handling
    - Mock network operations to avoid flaky external dependencies
    - Purpose: Validate BranchResolver logic with comprehensive URL and network scenarios
    - _Leverage: internal/testutils/mocks.go when available_
    - _Requirements: 1.1, 1.4, 5.2_

- [x]   26. Implement WorktreeCreator tests in internal/commands/create/worktree_creator_test.go (FIXED: Now properly tests functionality with state-based assertions)
    - File: internal/commands/create/worktree_creator_test.go
    - Implement all 12 WorktreeCreator test cases from specifications
    - Cover atomic operations, cleanup, and recovery scenarios
    - Use filesystem mocking to avoid test interference (learn from previous race conditions)
    - Purpose: Validate WorktreeCreator complex operations with reliable isolation
    - _Leverage: internal/testutils/mocks.go when available_
    - _Requirements: 1.1, 1.4, 5.3_

#### Integration Test Implementation Tasks

- [x]   27. Implement CLI command integration tests in test/integration/testdata/
    - Files: create_basic.txt, create_with_remote.txt, list_empty_workspace.txt, etc.
    - Implement all 20 CLI integration test scenarios from specifications
    - Use testscript framework for reliable test execution (avoid previous complex setup)
    - Cover create, list, and config commands with proper isolation
    - Purpose: Validate end-to-end CLI behavior with real but isolated environments
    - _Leverage: test/integration/integration_test.go framework_
    - _Requirements: 3.2, 3.3, 6.2_

- [x]   28. Implement cross-platform integration tests
    - Files: cross_platform_paths.txt, cross_platform_permissions.txt, cross_platform_symlinks.txt, cross_platform_line_endings.txt, cross_platform_case_sensitivity.txt, cross_platform_unicode.txt
    - Implement all 6 cross-platform test scenarios from specifications
    - Use CI matrix testing for Windows/Linux/macOS validation
    - Test path handling, permissions, symlinks, and unicode support
    - Purpose: Ensure Grove works consistently across all supported platforms
    - _Leverage: existing CI configuration patterns_
    - _Requirements: 6.1, 6.2_

- [x]   29. Implement environment isolation tests
    - Files: environment_home_directory.txt, environment_config_paths.txt, etc.
    - Implement all 6 environment isolation test scenarios from specifications
    - Ensure complete environment cleanup (learn from previous cleanup failures)
    - Test HOME, XDG, Git config, SSH, and proxy isolation
    - Purpose: Validate tests don't interfere with each other or host environment
    - _Leverage: test/integration/testdata/helpers.txt_
    - _Requirements: 4.4, 6.1_

#### Performance and Load Test Implementation Tasks

- [ ]   30. Implement performance benchmark tests
    - Files: internal/git/commander_bench_test.go, internal/commands/create/create_service_bench_test.go
    - Implement all 6 performance benchmark scenarios from specifications
    - Benchmark GitCommander, CreateService, BranchResolver operations
    - Test with large repositories (>1GB) and many worktrees (100+)
    - Purpose: Establish performance baselines and detect regressions
    - _Leverage: Go testing.B benchmark patterns_
    - _Requirements: 4.1, 4.2_

- [ ]   31. Implement load testing scenarios
    - Files: test/integration/load/concurrent_test.go, test/integration/load/stress_test.go
    - Implement all 6 load testing scenarios from specifications
    - Test concurrent operations, high volume commands, resource pressure
    - Use controlled resource constraints to validate graceful degradation
    - Purpose: Validate Grove behavior under stress and resource constraints
    - _Leverage: Go testing patterns for concurrent operations_
    - _Requirements: 4.1, 4.2, 6.3_

#### Edge Case and Infrastructure Test Implementation Tasks

- [x]   32. Implement edge case tests
    - Files: test/integration/edge_cases/filesystem_test.go, test/integration/edge_cases/git_test.go, test/integration/edge_cases/network_test.go
    - Implement all 18 edge case test scenarios from specifications
    - Cover filesystem, Git repository, and network edge cases
    - Use controlled failure injection to test error handling
    - Purpose: Validate Grove handles extreme and unusual scenarios gracefully
    - _Leverage: existing error handling patterns_
    - _Requirements: 4.1, 4.2, 6.3_

- [ ]   33. Implement infrastructure validation tests
    - Files: internal/testutils/testutils_test.go, test/build/build_test.go, docs/docs_test.go
    - Implement all 18 infrastructure validation test scenarios from specifications
    - Test TestUtils, build system integration, documentation accuracy
    - Validate the testing infrastructure itself is reliable
    - Purpose: Ensure testing infrastructure is robust and maintainable
    - _Leverage: existing build and documentation patterns_
    - _Requirements: 7.1, 7.4, 7.5_

### Phase 9: Test Quality Assurance

- [ ]   34. Validate comprehensive test coverage
    - Run all implemented tests and validate coverage metrics
    - Ensure 90%+ coverage target is met with meaningful tests
    - Identify and fill any coverage gaps discovered
    - Purpose: Confirm comprehensive test coverage meets quality standards
    - _Leverage: magefile.go coverage configuration_
    - _Requirements: 5.1, 7.4_

- [ ]   35. Performance baseline establishment
    - Run all performance benchmarks and establish baseline metrics
    - Document expected performance characteristics and thresholds
    - Set up performance regression detection in CI
    - Purpose: Establish performance monitoring and regression detection
    - _Leverage: existing CI/CD patterns_
    - _Requirements: 4.1, 4.2, 7.2_
