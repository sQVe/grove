# Requirements Document

## Introduction

The switch command enables users to easily change their current working directory to any existing worktree. This feature eliminates the friction of manually navigating between worktrees and provides the seamless context switching that Grove's product vision promises. The command offers multiple execution modes to accommodate different user preferences and shell environments.

## Alignment with Product Vision

This feature directly addresses the "Navigation Difficulty" core problem by providing seamless movement between different worktrees. It supports the product principle of "Intuitive Interface" by making worktree switching feel natural to developers, and enables the "Seamless context switching between features" success metric outlined in product.md.

## Requirements

### Requirement 1

**User Story:** As a developer, I want to quickly switch to any existing worktree, so that I can seamlessly transition between different features and branches without manual navigation.

#### Acceptance Criteria

1. WHEN I run `grove switch <worktree-name>` THEN the system SHALL change my current working directory to the specified worktree path
2. IF the worktree name does not exist THEN the system SHALL display an error message listing available worktrees
3. WHEN I provide a partial worktree name THEN the system SHALL complete the name if there is a unique match or show suggestions for ambiguous matches
4. WHEN the switch operation succeeds THEN the system SHALL update my shell environment to the target directory

### Requirement 2

**User Story:** As a developer using different shells, I want grove switch to work naturally in my preferred shell environment, so that I don't need to learn different workflows for different shells.

#### Acceptance Criteria

1. WHEN I install Grove shell integration THEN `grove switch` SHALL work seamlessly in bash, zsh, fish, and PowerShell
2. IF shell integration is not installed THEN the system SHALL provide clear setup instructions with examples
3. WHEN I run `grove switch <worktree>` without shell integration THEN the system SHALL clearly explain that shell integration is required for directory switching and show setup instructions
4. WHEN shell integration fails THEN the system SHALL fallback to alternative methods (--eval, --subshell) gracefully
5. WHEN I run the command without shell integration THEN the system SHALL offer multiple output modes (--eval, --subshell)

### Requirement 3

**User Story:** As a developer, I want tab completion for worktree names, so that I can quickly identify and switch to the correct worktree without typing full names.

#### Acceptance Criteria

1. WHEN I type `grove switch <tab>` THEN the system SHALL show all available worktree names
2. WHEN I type `grove switch <partial-name><tab>` THEN the system SHALL complete matching worktree names
3. IF multiple worktrees match the partial name THEN the system SHALL show all matching options
4. WHEN worktree names are cached THEN completions SHALL remain fast (under 100ms) even with many worktrees

### Requirement 4

**User Story:** As a developer who prefers different interaction methods, I want multiple ways to use the switch command, so that I can choose the approach that best fits my workflow.

#### Acceptance Criteria

1. WHEN I run `grove switch --get-path <worktree>` THEN the system SHALL output only the absolute path to the worktree
2. WHEN I run `grove switch --eval <worktree>` THEN the system SHALL output `cd /path/to/worktree` for shell evaluation
3. WHEN I run `grove switch --subshell <worktree>` THEN the system SHALL launch a new shell in the target directory
4. IF no flags are provided THEN the system SHALL use the default behavior based on detected shell integration

### Requirement 5

**User Story:** As a developer, I want helpful setup guidance for shell integration, so that I can easily configure grove switch to work optimally in my environment.

#### Acceptance Criteria

1. WHEN I run `grove switch` without shell integration THEN the system SHALL display setup instructions for my current shell
2. WHEN I run `grove install-shell-integration` THEN the system SHALL offer both automatic installation (with confirmation) and manual output options
3. WHEN I choose automatic installation THEN the system SHALL create backup files before modifying shell configuration files (e.g., ~/.zshrc, ~/.bashrc)
4. WHEN I choose manual installation THEN the system SHALL output the exact shell functions and instructions for me to copy-paste into my shell configuration
5. IF automatic installation fails THEN the system SHALL fallback to manual setup instructions with exact commands

## Non-Functional Requirements

### Performance

- Switch operations must complete in under 100ms for local worktrees
- Tab completion must respond in under 100ms even with 100+ worktrees
- Worktree name caching should improve completion performance

### Security

- Path resolution must validate worktree paths to prevent directory traversal
- Shell command generation must escape special characters properly
- Generated shell functions must not execute arbitrary code

### Reliability

- Must handle non-existent worktree names gracefully with helpful error messages
- Must detect and recover from shell integration setup failures
- Must work across all supported platforms (Windows, macOS, Linux)

### Usability

- Error messages must be actionable and include suggestions for resolution
- Setup instructions must be copy-pasteable for each supported shell
- Command must follow existing Grove CLI patterns and conventions
