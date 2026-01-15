# Design

## Workspace architecture

Grove stores Git data in a bare repository (`.bare`) with worktrees as sibling directories:

```
project/
├── .bare/           # Bare Git repository (objects, refs)
├── .git             # File: "gitdir: .bare"
├── main/            # Worktree for main branch
├── feature-auth/    # Worktree for feature/auth
└── bugfix-login/    # Worktree for bugfix/login
```

- `.bare` holds the complete Git repository without a working tree
- `.git` file redirects Git operations to `.bare`
- Worktree directories contain isolated working copies
- Branch names like `feature/auth` become `feature-auth`

Grove finds workspaces by traversing parent directories for `.bare` or `.git` files containing `gitdir: .bare`.
