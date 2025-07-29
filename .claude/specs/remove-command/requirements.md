# Requirements Document

## Introduction

The remove command enables developers to safely delete worktrees while maintaining data integrity and providing clear feedback about the removal process. This feature completes Grove's essential worktree lifecycle by providing reliable cleanup operations that follow the product's core principle of "zero accidental data loss during cleanup."

## Alignment with Product Vision

This feature directly supports Grove's mission to make worktree management "as simple as branch switching" by:

- **Reliable Automation**: Implements smart cleanup that prevents data loss through comprehensive safety checks
- **Speed First**: Provides fast removal operations with immediate feedback
- **Intuitive Interface**: Follows Git-familiar command patterns that feel natural to users
- **Developer Focused**: Addresses the core problem of "Cleanup Overhead" by automating safe worktree removal

The remove command transforms the cleanup experience from "I hope I don't lose anything" to "Grove will safely handle this for me."

## Requirements

### Requirement 1: Safe Worktree Removal

**User Story:** As a developer, I want to remove a worktree safely, so that I can clean up completed work without risking data loss.

#### Acceptance Criteria

1. WHEN user runs `grove remove <worktree-path>` THEN Grove SHALL check for uncommitted changes before removal
2. IF worktree has uncommitted changes THEN Grove SHALL block removal and display clear warning message
3. WHEN worktree has no uncommitted changes THEN Grove SHALL remove both the worktree directory and Git metadata
4. WHEN removal completes successfully THEN Grove SHALL display confirmation message with removed path
5. IF removal fails for any reason THEN Grove SHALL display clear error message and leave worktree intact

### Requirement 2: Interactive Removal with Safety Prompts

**User Story:** As a developer, I want to be prompted before removing worktrees with uncommitted changes, so that I can make informed decisions about data loss.

#### Acceptance Criteria

1. WHEN user runs `grove remove <worktree-path> --force` THEN Grove SHALL prompt for confirmation before removal
2. IF user confirms force removal THEN Grove SHALL remove worktree regardless of uncommitted changes
3. WHEN user runs `grove remove <worktree-path> --dry-run` THEN Grove SHALL show what would be removed without actual deletion
4. IF worktree contains untracked files THEN Grove SHALL list them and prompt for confirmation
5. WHEN user cancels confirmation prompt THEN Grove SHALL abort removal and leave worktree intact

### Requirement 3: Bulk Removal Operations

**User Story:** As a developer, I want to remove multiple worktrees at once, so that I can efficiently clean up completed features.

#### Acceptance Criteria

1. WHEN user runs `grove remove --merged` THEN Grove SHALL identify and remove all merged worktrees
2. IF any merged worktree has uncommitted changes THEN Grove SHALL skip it and report in summary
3. WHEN user runs `grove remove --stale --days=30` THEN Grove SHALL remove worktrees with no activity for 30+ days
4. WHEN bulk removal begins THEN Grove SHALL display summary of worktrees to be removed
5. IF user runs `grove remove --all` THEN Grove SHALL prompt for confirmation and remove all non-current worktrees

### Requirement 4: Branch Cleanup Integration

**User Story:** As a developer, I want worktree removal to optionally clean up associated branches, so that I can maintain a tidy Git repository.

#### Acceptance Criteria

1. WHEN user runs `grove remove <worktree-path> --delete-branch` THEN Grove SHALL remove both worktree and its associated branch
2. IF branch is merged into default branch OR branch is pushed to remote with no uncommitted changes THEN Grove SHALL safely delete local branch without confirmation
3. IF branch is not merged AND not pushed to remote THEN Grove SHALL warn user and require explicit confirmation
4. WHEN branch has remote tracking THEN Grove SHALL offer to delete remote branch as well
5. IF branch deletion fails THEN Grove SHALL complete worktree removal but report branch deletion failure
6. WHEN worktree removal includes branch deletion THEN Grove SHALL display summary of all cleanup actions

### Requirement 5: Error Recovery and Validation

**User Story:** As a developer, I want Grove to handle removal errors gracefully, so that I understand what went wrong and can take corrective action.

#### Acceptance Criteria

1. WHEN worktree path does not exist THEN Grove SHALL display clear "not found" message
2. IF worktree is currently active/checked out THEN Grove SHALL block removal and suggest switching first
3. WHEN filesystem permissions prevent removal THEN Grove SHALL display actionable error message
4. IF Git metadata removal fails THEN Grove SHALL attempt cleanup and report partial success status
5. WHEN removal is interrupted THEN Grove SHALL attempt to restore consistent state and report status

## Non-Functional Requirements

### Performance

- Removal operations must complete within 2 seconds for standard worktrees
- Bulk operations must provide progress feedback for operations taking >5 seconds
- Memory usage must remain under 50MB even when processing 100+ worktrees

### Security

- Must validate all file paths to prevent directory traversal attacks
- Must never remove files outside of the designated worktree path
- Must preserve file permissions and ownership during cleanup operations

### Reliability

- Must ensure atomic operations - either complete success or complete rollback
- Must maintain Git repository integrity even if worktree removal fails
- Must handle interrupted operations gracefully without corrupting repository state

### Usability

- Error messages must be actionable and include suggested solutions
- Command interface must be consistent with existing Grove commands
- Must provide clear feedback for all operations with appropriate detail level
