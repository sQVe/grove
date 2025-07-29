# Implementation Tasks

## Task Breakdown

- [x]   1. Create switch command package structure
    - Create `/internal/commands/switch/` directory
    - Create basic package files following Grove conventions
    - _Leverage: `/internal/commands/create/` structure, `/internal/commands/list/` patterns_
    - _Requirements: 1.1, 2.1_

- [x]   2. Implement core path resolution service
    - Create `SwitchService` interface and implementation
    - Implement worktree path resolution with exact and fuzzy matching
    - Add path validation and security checks
    - _Leverage: `internal/completion/worktree.go:161-207`, `internal/completion/worktree.go:33-68`_
    - _Requirements: 1.1, 1.2, 1.3_

- [x]   3. Create basic switch command with --get-path mode
    - Implement `switch.go` with Cobra command setup
    - Add `--get-path` flag for path-only output
    - Integrate with path resolution service
    - _Leverage: `/internal/commands/list/list.go`, `/internal/commands/create/create.go` command patterns_
    - _Requirements: 1.1, 4.1_

- [x]   4. Add multiple execution modes
    - Implement `--eval` mode for shell evaluation output
    - Implement `--subshell` mode for launching new shell
    - Add auto-detection of shell integration availability
    - _Leverage: existing shell detection patterns, configuration system_
    - _Requirements: 2.4, 4.2, 4.3, 4.4_

- [ ]   5. Implement shell integration generation
    - Create `shell_integration.go` with function generation logic
    - Support bash, zsh, fish, and PowerShell function generation
    - Add shell environment detection and validation
    - _Leverage: `/internal/completion/completion.go:120-165` completion patterns_
    - _Requirements: 2.1, 2.2, 5.2_

- [ ]   6. Create install-shell-integration command
    - Implement installation command with backup creation
    - Add shell configuration file modification logic
    - Provide manual setup instructions for fallback
    - _Leverage: existing configuration file handling patterns_
    - _Requirements: 5.1, 5.3, 5.4_

- [ ]   7. Integrate switch command completion
    - Extend existing completion system for switch command
    - Add worktree name completion using cached data
    - Register completion functions in main.go
    - _Leverage: `/internal/completion/worktree.go:11-31`, `/internal/completion/completion.go:167-189`_
    - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [ ]   8. Add switch command to main CLI
    - Register switch command in `cmd/grove/main.go`
    - Add completion registration for switch command
    - Update help text and command descriptions
    - _Leverage: existing command registration patterns in `main.go:38-41`_
    - _Requirements: 1.1, 2.1_

- [ ]   9. Implement comprehensive error handling
    - Add specific error types for worktree not found, permission denied
    - Create actionable error messages with suggestions
    - Add shell integration setup failure handling
    - _Leverage: `/internal/errors/` error handling patterns_
    - _Requirements: 1.2, 2.3, 5.4_

- [ ]   10. Create unit tests for core functionality
    - Test path resolution logic with various worktree configurations
    - Test shell integration generation for all supported shells
    - Test error handling and edge cases
    - _Leverage: `/internal/testutils/mocks.go`, existing test patterns_
    - _Requirements: All requirements validation_

- [ ]   11. Create integration tests
    - Test end-to-end switching with real git worktrees
    - Test cross-platform shell integration
    - Test performance with large numbers of worktrees
    - _Leverage: existing integration test patterns from create/list commands_
    - _Requirements: Performance and reliability requirements_

- [ ]   12. Add configuration options
    - Add switch-specific configuration options (default mode, shell preference)
    - Integrate with existing Viper/TOML configuration system
    - Add validation for configuration values
    - _Leverage: `/internal/config/` configuration patterns_
    - _Requirements: 2.2, 4.4_
