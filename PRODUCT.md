# Grove Product Vision

**"Make Git worktrees as simple as switching branches"**

Grove makes Git worktree management accessible to any developer. Work on multiple features simultaneously without stashing or branch switching.

## Target Users

-   Developers working on multiple features/branches
-   Teams doing hotfixes while developing features
-   Code reviewers needing quick branch access
-   Open source contributors with multiple PRs

## Core Value

Replace complex Git worktree commands with simple Grove commands that handle the complexity automatically.

**Instead of:**

-   Stashing/unstashing when switching branches
-   Complex `git worktree` commands
-   Multiple repository clones

**Grove provides:**

-   `grove init clone <url>` - Setup workspace
-   `grove create feature/auth` - New worktree
-   `grove switch main` - Switch to branch

## Key Benefits

-   Work on multiple branches simultaneously
-   Each worktree maintains independent state
-   Shared Git repository (no duplication)
-   Cross-platform compatibility

## Current Status

Grove is in early development:

-   ✅ Initialize workspaces (`grove init`)
-   ✅ Clone repositories with branch setup
-   ⏳ Core worktree commands (create, list, switch, remove)
-   ⏳ Enhanced features and integrations

Grove succeeds when developers find worktree management easy instead of expert-level.
