# Design Document

## Overview

The switch command will be implemented following Grove's established patterns, leveraging existing worktree parsing, completion infrastructure, and command structure. The design supports multiple execution modes through a unified interface with intelligent fallback behavior.

## Code Reuse Analysis

**Extensive reuse opportunities identified:**

- **Command Structure**: Follow patterns from `/internal/commands/create/`, `/internal/commands/list/`
- **Worktree Parsing**: Reuse `GetWorktreeInfo()` from `/internal/completion/worktree.go:161`
- **Shell Completion**: Extend existing `WorktreeCompletion()` from `/internal/completion/worktree.go:11`
- **Git Execution**: Use existing `git.GitExecutor` interface and implementation
- **Configuration**: Leverage Viper/TOML configuration system
- **Cross-platform Support**: Reuse existing platform detection and shell handling

## Architecture

### Command Registration

Following existing patterns in `cmd/grove/main.go:38-41`, the switch command will be registered alongside init, config, list, and create commands.

### Package Structure

```
internal/commands/switch/
├── switch.go              # Main command implementation
├── switch_service.go      # Core switching logic
├── shell_integration.go   # Shell function generation
├── path_resolver.go       # Worktree path resolution
├── switch_test.go         # Unit tests
└── switch_integration_test.go  # Integration tests
```

## Components and Interfaces

### SwitchService Interface

```go
type SwitchService interface {
    Switch(ctx context.Context, worktreeName string, options SwitchOptions) (*SwitchResult, error)
    GetWorktreePath(worktreeName string) (string, error)
    ListWorktrees() ([]WorktreeInfo, error)
    GenerateShellIntegration(shell string) (string, error)
}
```

### SwitchOptions

```go
type SwitchOptions struct {
    Mode         SwitchMode  // Direct, Eval, Subshell, GetPath
    Shell        string      // Target shell for integration
    ForceInstall bool        // Force shell integration installation
}

type SwitchMode int
const (
    ModeAuto SwitchMode = iota  // Auto-detect based on shell integration
    ModeEval                    // Output cd command for eval
    ModeSubshell               // Launch subshell in directory
    ModeGetPath                // Output path only
)
```

### Path Resolution Strategy

Reuse existing worktree parsing logic from `internal/completion/worktree.go:161-207` with enhancements:

1. **Exact Match**: Direct worktree name lookup
2. **Fuzzy Match**: Partial name completion using existing `FilterCompletions()`
3. **Path Validation**: Security checks for directory traversal
4. **Error Handling**: Actionable error messages with suggestions

## Data Models

### WorktreeInfo Extension

Leverage existing `WorktreeInfo` struct from `/internal/completion/worktree.go:209-224` with additional methods:

```go
// Extend existing WorktreeInfo with convenience methods
func (w WorktreeInfo) AbsolutePath() (string, error)
func (w WorktreeInfo) IsAccessible() bool
func (w WorktreeInfo) DisplayName() string
```

## Error Handling

Following Grove's established error handling patterns in `/internal/errors/`:

- **NotFound**: Worktree does not exist with suggestions for similar names
- **Permission**: Directory access denied with troubleshooting steps
- **ShellIntegration**: Shell setup failures with manual instructions
- **InvalidPath**: Path validation failures with security context

## Testing Strategy

### Unit Tests (switch_test.go)

- Path resolution logic with various worktree configurations
- Shell integration generation for all supported shells
- Error handling for edge cases (missing worktrees, permission issues)
- Mode selection logic and fallback behavior

### Integration Tests (switch_integration_test.go)

- End-to-end switching with real git worktrees
- Cross-platform shell integration testing
- Performance testing with large numbers of worktrees
- Shell completion integration testing

### Leveraging Existing Test Infrastructure

- Reuse `MockGitExecutor` from `/internal/testutils/mocks.go`
- Use existing test fixture patterns from create/list commands
- Follow established test organization (basic, errors, validation, benchmarks)

## Shell Integration Design

### Function Override Approach (Primary)

Generate shell functions that override the `grove` command for the `switch` subcommand:

```bash
# Generated function for bash/zsh
grove() {
    if [[ "$1" == "switch" ]]; then
        local target=$(command grove switch --get-path "$2")
        if [[ $? -eq 0 ]] && [[ -n "$target" ]]; then
            cd "$target"
        else
            command grove switch "$@"  # Show error message
        fi
    else
        command grove "$@"
    fi
}
```

### Installation Command

New `grove install-shell-integration` command that:

1. Detects current shell environment
2. Generates appropriate shell functions
3. Offers to modify shell configuration files
4. Creates backup files before modification
5. Provides manual setup instructions as fallback

### Completion Integration

Extend existing completion registration in `cmd/grove/main.go:45` to include switch command:

```go
// Add to RegisterCompletionFunctions
if cmd.Name() == "switch" {
    registerSwitchCompletions(cmd, ctx)
}
```

## Implementation Plan

### Phase 1: Core Switching Logic

- Implement `SwitchService` with worktree path resolution
- Add basic switch command with `--get-path` mode
- Integrate with existing worktree parsing logic

### Phase 2: Multiple Execution Modes

- Add `--eval` and `--subshell` modes
- Implement auto-detection of shell integration
- Add comprehensive error handling and user guidance

### Phase 3: Shell Integration

- Implement shell function generation for bash, zsh, fish, PowerShell
- Add `install-shell-integration` command
- Integrate with existing completion system

### Phase 4: Testing and Polish

- Comprehensive test coverage following existing patterns
- Performance optimization using existing caching strategies
- Documentation and help text following Grove conventions

## Security Considerations

- **Path Validation**: Use `filepath.Clean()` and validate resolved paths are within expected directories
- **Shell Injection**: Properly escape all path components in generated shell commands
- **File Permissions**: Validate directory accessibility before attempting to switch
- **Configuration Files**: Create backups before modifying shell configuration files

## Performance Considerations

- **Caching**: Leverage existing worktree caching from completion system
- **Lazy Loading**: Only resolve paths when needed for actual switches
- **Completion Speed**: Reuse cached worktree lists for tab completion
- **Shell Detection**: Cache shell environment detection results

## Compatibility

### Cross-Platform Support

- Reuse existing platform detection patterns
- Handle Windows path separators and drive letters
- Support both PowerShell and cmd.exe on Windows

### Shell Support Matrix

- **bash**: Function override with completion
- **zsh**: Function override with enhanced completion
- **fish**: Function wrapper with fish-specific completion
- **PowerShell**: Function with PowerShell completion
- **Fallback**: Eval and subshell modes for unsupported shells
