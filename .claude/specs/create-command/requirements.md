# Requirements Document

## Introduction

The create command enables Grove users to create new Git worktrees from branches quickly and intuitively. This feature is a core part of Grove's mission to make worktree management as simple as branch switching, providing the essential functionality for users to create isolated working directories for different branches.

## Alignment with Product Vision

This feature directly supports Grove's core mission of transforming Git worktrees from "a complex power-user feature into an essential productivity tool." It addresses the "Complex Setup" problem by eliminating the friction of Git worktree initialization and aligns with the "Speed First" principle by making worktree creation snappy and quick.

## Codebase Analysis Summary

**Existing Components to Leverage:**

- Worktree creation functions: `CreateWorktreeWithSafeNaming` and `CreateWorktreeFromExistingBranch` (internal/git/worktree.go:90, 136)
- Git operations infrastructure (internal/git/operations.go)
- Command structure patterns from existing commands (internal/commands/init.go, config.go)
- Error handling patterns from internal/errors
- Git executor interface for testability (internal/git/operations.go:18-22)
- Naming utilities for safe worktree names (internal/git/naming.go)

**Integration Points:**

- Main command registration in cmd/grove/main.go
- Existing Git operation patterns and retry mechanisms
- Configuration system for worktree naming and path preferences
- Completion system for branch name suggestions

## Requirements

### Requirement 1

**User Story:** As a Grove user, I want to create a new worktree from an existing branch, so that I can work on multiple features simultaneously without conflicts.

#### Acceptance Criteria

1. WHEN user runs `grove create <branch-name>` AND branch exists THEN Grove SHALL create a new worktree for the existing branch
2. WHEN user runs `grove create <branch-name> <path>` THEN Grove SHALL create the worktree at the specified path
3. WHEN branch exists locally or remotely THEN Grove SHALL checkout the branch in the new worktree
4. WHEN worktree is created THEN Grove SHALL use safe naming conventions to avoid path conflicts
5. WHEN worktree creation succeeds THEN Grove SHALL display success message with worktree path

### Requirement 2

**User Story:** As a Grove user, I want Grove to automatically detect if a branch exists and handle new branch creation intelligently, so that I have a seamless experience without remembering flags.

#### Acceptance Criteria

1. WHEN user runs `grove create <branch-name>` AND branch does not exist THEN Grove SHALL prompt "Branch '<branch-name>' does not exist. Create it? (Y/n)"
2. WHEN user provides `--create` or `-c` flag THEN Grove SHALL create new branch without prompting
3. WHEN user confirms branch creation OR provides `--create` flag THEN Grove SHALL create new branch based off current branch or specified base branch
4. WHEN user provides `--base <branch>` THEN Grove SHALL create the new branch from the specified base branch
5. IF base branch does not exist THEN Grove SHALL display error and list available branches
6. WHEN new branch is created THEN Grove SHALL track it against the default remote

### Requirement 3

**User Story:** As a Grove user, I want the create command to automatically handle path generation, so that I don't have to manually specify paths for each worktree.

#### Acceptance Criteria

1. WHEN user does not specify path THEN Grove SHALL generate safe path based on branch name
2. WHEN generated path conflicts with existing directory THEN Grove SHALL append suffix to make it unique
3. WHEN branch name contains special characters THEN Grove SHALL sanitize them for filesystem safety
4. IF path generation fails THEN Grove SHALL provide alternative suggestions
5. WHEN path is generated THEN Grove SHALL create parent directories as needed

### Requirement 4

**User Story:** As a Grove user, I want the create command to validate inputs and provide helpful feedback, so that I can quickly resolve any issues.

#### Acceptance Criteria

1. WHEN user provides invalid branch name THEN Grove SHALL display validation error with correct format
2. WHEN specified path already exists THEN Grove SHALL prompt for confirmation or suggest alternatives
3. WHEN not in a Grove repository THEN Grove SHALL display clear error message with initialization instructions
4. IF git operations fail THEN Grove SHALL provide actionable error messages with troubleshooting steps
5. WHEN command succeeds THEN Grove SHALL display success message with worktree path and next steps

### Requirement 5

**User Story:** As a Grove user, I want the create command to integrate with Grove's configuration system, so that worktree creation follows my preferences.

#### Acceptance Criteria

1. WHEN user has configured default worktree base path THEN Grove SHALL use it for path generation
2. WHEN user has naming preferences configured THEN Grove SHALL apply them to worktree directory names
3. WHEN user has remote tracking preferences THEN Grove SHALL apply them to new branches
4. IF configuration is invalid THEN Grove SHALL use sensible defaults and warn user
5. WHEN creating worktree THEN Grove SHALL respect configured git timeout and retry settings

### Requirement 6

**User Story:** As a Grove user, I want shell completion for the create command, so that I can quickly select branch names without typing them fully.

#### Acceptance Criteria

1. WHEN user types `grove create <TAB>` THEN Grove SHALL provide completion suggestions for local and remote branch names
2. WHEN user types `grove create --base <TAB>` THEN Grove SHALL provide completion suggestions for available base branches
3. WHEN user types `grove create branch-name <TAB>` THEN Grove SHALL provide completion suggestions for directory paths
4. WHEN completion is triggered THEN Grove SHALL filter suggestions based on current input
5. WHEN no branches exist THEN Grove SHALL provide helpful completion message indicating repository status

### Requirement 7

**User Story:** As a Grove user, I want to create worktrees from Git platform URLs (GitHub, GitLab, etc.), so that I can easily work on pull requests and branches from web interfaces.

#### Acceptance Criteria

1. WHEN user runs `grove create https://github.com/owner/repo/pull/123` THEN Grove SHALL create worktree from the PR branch
2. WHEN user runs `grove create https://gitlab.com/owner/repo/-/merge_requests/456` THEN Grove SHALL create worktree from the merge request branch
3. WHEN user provides GitHub branch URL `https://github.com/owner/repo/tree/feature-branch` THEN Grove SHALL create worktree from that specific branch
4. WHEN user provides Bitbucket PR URL THEN Grove SHALL parse and create worktree from the pull request branch
5. WHEN user provides Azure DevOps or Gitea URLs THEN Grove SHALL support those platforms using existing ParseGitPlatformURL function
6. WHEN URL points to external repository THEN Grove SHALL add remote and fetch branch appropriately
7. IF URL parsing fails THEN Grove SHALL display clear error message with supported URL formats

### Requirement 8

**User Story:** As a Grove user, I want to create worktrees from remote branch references, so that I can quickly work on branches from different remotes.

#### Acceptance Criteria

1. WHEN user runs `grove create origin/feature-branch` THEN Grove SHALL create worktree from the remote branch
2. WHEN user runs `grove create upstream/hotfix` THEN Grove SHALL create worktree from the upstream remote branch
3. WHEN remote branch doesn't exist locally THEN Grove SHALL fetch from remote before creating worktree
4. WHEN remote doesn't exist THEN Grove SHALL display error with available remotes
5. WHEN user provides remote branch that conflicts with local branch THEN Grove SHALL prompt for resolution strategy

### Requirement 9

**User Story:** As a Grove user, I want Grove to automatically copy configuration files from my main worktree, so that I don't have to manually set up my development environment each time.

#### Acceptance Criteria

1. WHEN user creates new worktree THEN Grove SHALL automatically copy files matching configured patterns from source worktree
2. WHEN configuration specifies patterns like `.env*` and `.vscode/` THEN Grove SHALL copy all matching files and directories
3. WHEN copied file already exists in target THEN Grove SHALL prompt user for conflict resolution (skip, overwrite, backup)
4. WHEN user provides `--copy-env` flag THEN Grove SHALL copy common environment files regardless of configuration
5. WHEN user provides `--copy "pattern1,pattern2"` THEN Grove SHALL copy only those specific patterns
6. WHEN user provides `--no-copy` flag THEN Grove SHALL skip all file copying
7. IF source worktree cannot be determined THEN Grove SHALL use sensible defaults or prompt user

### Requirement 10

**User Story:** As a Grove user, I want to configure which files are automatically copied to new worktrees, so that my team has consistent development environments.

#### Acceptance Criteria

1. WHEN user configures `[worktree.copy_files]` section THEN Grove SHALL use those patterns for automatic copying
2. WHEN user specifies `source_worktree = "main"` THEN Grove SHALL copy files from the main worktree
3. WHEN user sets `on_conflict = "prompt"` THEN Grove SHALL ask before overwriting existing files
4. WHEN user sets `on_conflict = "skip"` THEN Grove SHALL not overwrite existing files
5. WHEN user sets `on_conflict = "overwrite"` THEN Grove SHALL replace existing files without prompting
6. WHEN configuration is invalid THEN Grove SHALL use sensible defaults and warn user

## Non-Functional Requirements

### Performance

- Worktree creation must complete within 5 seconds for local branches
- Remote branch checkout should complete within 15 seconds with progress indication
- Path validation and generation must be instantaneous (< 100ms)

### Security

- Generated paths must be validated to prevent directory traversal attacks
- Branch names must be sanitized to prevent command injection
- File operations must respect filesystem permissions and fail safely

### Reliability

- Command must handle network interruptions gracefully with retry mechanisms
- Partial worktree creation must be cleaned up on failure
- Operations must be atomic where possible (complete success or complete rollback)

### Usability

- Error messages must be specific and actionable
- Success messages must include next steps (e.g., "cd into directory")
- Command must provide helpful suggestions for common mistakes
- Integration with shell completion for branch names
