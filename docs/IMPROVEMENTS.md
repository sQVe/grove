# Grove Codebase Improvements

This document tracks the systematic implementation of code quality improvements identified during the comprehensive codebase review.

## Implementation Status

| #   | Improvement                                         | Status      | Priority | Est. Time |
| --- | --------------------------------------------------- | ----------- | -------- | --------- |
| 1   | Add CLI Version Flag                                | ✅ Complete | High     | 30 min    |
| 2   | Standardize Error Handling                          | ✅ Complete | High     | 2 hours   |
| 3   | Consolidate Mock Implementations                    | ✅ Complete | High     | 3 hours   |
| 4   | Implement Configuration System                      | ✅ Complete | High     | 1 day     |
| 5   | Implement Command Registration Framework            | ⏳ Pending  | High     | 4 hours   |
| 6   | Add Progress Indicators                             | ⏳ Pending  | High     | 4 hours   |
| 7   | Add Retry Mechanisms                                | ✅ Complete | High     | 3 hours   |
| 8   | Increase Test Coverage                              | ⏳ Pending  | High     | 1 day     |
| 9   | Implement Filesystem-Safe Worktree Directory Naming | ✅ Complete | Medium   | 2-3 hours |
| 10  | Implement CLI Completion Support                    | ⏳ Pending  | Medium   | 4-6 hours |

## Implementation Details

### 1. Add CLI Version Flag 🚀

**Issue**: `./grove --version` fails with "unknown flag" error

**Solution**: Add version support to cobra command in `cmd/grove/main.go`

**Files to modify**:

- `cmd/grove/main.go` - Add version flag support

**Implementation**:

```go
// Add to rootCmd definition
rootCmd.Version = "v1.0.0" // TODO: Read from build info
```

**Testing**: Verify `grove --version` works correctly

---

### 2. Standardize Error Handling 🔧

**Issue**: Good error handling but inconsistent patterns across codebase

**Solution**: Create centralized error handling with constants and templates

**Files to create/modify**:

- `internal/errors/errors.go` - Error constants and templates
- `internal/errors/codes.go` - Error codes for programmatic handling
- Update all packages to use standardized errors

**Implementation**:

```go
// Error codes for programmatic handling
const (
	ErrGitNotFound = "GIT_NOT_FOUND"
	ErrRepoExists  = "REPO_EXISTS"
	ErrInvalidURL  = "INVALID_URL"
)

// Error templates for consistent messaging
var ErrorTemplates = map[string]string{
	ErrGitNotFound: "Git is not available in PATH",
	ErrRepoExists:  "repository already exists at %s",
}
```

**Testing**: Verify error messages are consistent and helpful

---

### 3. Consolidate Mock Implementations 📋

**Issue**: Mock implementations duplicated across packages

**Solution**: Centralize all mocks in `internal/testutils/`

**Files to modify**:

- `internal/testutils/mocks.go` - Enhanced with all mock functionality
- `internal/git/mock_test.go` - Remove duplicate mock
- Update all test files to use centralized mocks

**Implementation**:

- Move all mock implementations to testutils
- Create comprehensive MockGitExecutor with all features
- Update imports across test files

**Testing**: Verify all existing tests pass with centralized mocks

---

### 4. Implement Configuration System ⚙️

**Issue**: No configuration system, hard-coded values

**Solution**: Add comprehensive configuration with TOML/JSON support

**Files to create**:

- `internal/config/config.go` - Configuration struct and loading
- `internal/config/defaults.go` - Default configuration values
- `internal/config/validation.go` - Configuration validation
- `internal/config/config_test.go` - Configuration tests

**Implementation**:

```go
import "time"

type Config struct {
	Git struct {
		Timeout    time.Duration `toml:"timeout" json:"timeout"`
		MaxRetries int           `toml:"max_retries" json:"max_retries"`
	} `toml:"git" json:"git"`

	Logging struct {
		Level  string `toml:"level" json:"level"`
		Format string `toml:"format" json:"format"`
	} `toml:"logging" json:"logging"`
}
```

**Testing**: Configuration loading, validation, and default handling

**✅ Status**: **Complete** - Implementation finished with comprehensive CLI commands

- Created `internal/config/config.go` with configuration struct and loading
- Created `internal/config/defaults.go` with default configuration values
- Created `internal/config/validation.go` with comprehensive validation
- Created `internal/config/paths.go` with cross-platform config directory support
- Created `internal/commands/config.go` with full CLI command interface
- Added comprehensive tests with 100% coverage of config commands
- All configuration commands working: `get`, `set`, `list`, `validate`, `reset`, `path`, `init`
- TOML configuration files with environment variable overrides (`GROVE_*`)
- Built-in validation and helpful error messages
- Configuration sections: general, git, retry, logging, worktree

---

### 5. Implement Command Registration Framework 📦

**Issue**: Manual command registration, no systematic command handling

**Solution**: Create command registration and discovery system

**Files to create/modify**:

- `internal/commands/registry.go` - Command registration system
- `internal/commands/base.go` - Base command interface
- `cmd/grove/main.go` - Use command registry
- Update existing commands to use new framework

**Implementation**:

```go
type Command interface {
	Name() string
	Description() string
	Execute(args []string) error
}

type Registry struct {
	commands map[string]Command
}
```

**Testing**: Command registration, discovery, and execution

---

### 6. Add Progress Indicators 📊

**Issue**: Long operations provide no user feedback

**Solution**: Add progress reporting for long-running operations

**Files to create/modify**:

- `internal/ui/progress.go` - Progress indicator implementation
- `internal/git/operations.go` - Add progress callbacks
- `internal/commands/init.go` - Integrate progress indicators

**Implementation**:

```go
type ProgressReporter interface {
	Start(message string)
	Update(message string)
	Complete(message string)
}
```

**Testing**: Progress indicators during clone, worktree creation

---

### 7. Add Retry Mechanisms 🔄

**Issue**: Network operations fail immediately without retry

**Solution**: Implement exponential backoff for network operations

**Files to create/modify**:

- `internal/retry/retry.go` - Retry mechanism implementation
- `internal/git/operations.go` - Add retry to network operations
- `internal/git/default_branch.go` - Add retry to branch detection

**Implementation**:

```go
import (
	"fmt"
	"time"
)

func WithRetry(fn func() error, maxRetries int) error {
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		}
		time.Sleep(time.Duration(i) * time.Second)
	}
	return fmt.Errorf("operation failed after %d retries", maxRetries)
}
```

**Testing**: Retry behavior under network failures

---

### 8. Increase Test Coverage 🧪

**Issue**: Current coverage 85.6%, gaps in critical paths

**Solution**: Add comprehensive tests for uncovered code paths

**Focus Areas**:

- Main execution paths in commands (currently 64.1%)
- Git operations (currently 62.1%)
- Error handling paths
- Edge cases and concurrent operations

**Files to create/modify**:

- Add tests for `runInit`, `runInitRemote`, `runInitConvert`
- Add tests for `Execute`, `ExecuteWithContext`
- Add comprehensive error condition tests
- Add integration tests for complex scenarios

**Target**: 90%+ test coverage across all packages

**Testing**: Comprehensive coverage of all code paths

---

### 9. Implement Filesystem-Safe Worktree Directory Naming 📁

**Issue**: Current worktree creation with `git worktree add fix/123` creates branch `123` instead of `fix/123`, and directory structure doesn't follow filesystem-safe naming conventions

**Solution**: Implement automatic branch name to directory name conversion that ensures filesystem-safe directory names while preserving intended branch names

**Files to create/modify**:

- `internal/git/worktree.go` - Add directory name transformation logic
- `internal/git/naming.go` - Branch name to filesystem-safe directory conversion
- `internal/commands/create.go` - Use filesystem-safe naming in worktree creation
- `internal/commands/init.go` - Apply naming conventions during initialization

**Implementation**:

```go
import (
	"os/exec"
	"path/filepath"
	"strings"
)

// Convert branch names like "fix/123" to filesystem-safe directory names like "fix-123"
func BranchToDirectoryName(branchName string) string {
	// Replace filesystem-unsafe characters with safe alternatives
	return strings.ReplaceAll(branchName, "/", "-")
}

// Create worktree with proper branch and directory naming
func CreateWorktreeWithSafeNaming(branchName string, basePath string) error {
	dirName := BranchToDirectoryName(branchName)
	dirPath := filepath.Join(basePath, dirName)

	// Use -b flag to ensure correct branch name
	return exec.Command("git", "worktree", "add", "-b", branchName, dirPath).Run()
}
```

**Testing**:

- Verify branch names with slashes create correct directory names
- Test edge cases like multiple slashes, dots, and other special characters
- Ensure proper branch creation with intended branch names
- Validate directory structure follows filesystem conventions

**✅ Status**: **Complete** - Implementation finished with comprehensive tests

- Created `internal/git/naming.go` with `BranchToDirectoryName()` function
- Created `internal/git/worktree.go` with enhanced worktree creation functions
- Updated `internal/commands/init.go` to use safe naming in `CreateAdditionalWorktrees`
- Updated `internal/git/operations.go` to use safe naming in `createProperWorktreeStructure`
- Added comprehensive tests covering all edge cases and error conditions
- All tests pass and functionality works as expected

---

### 10. Implement CLI Completion Support 🔧

**Issue**: No shell completion support for commands, flags, and dynamic values like branch names

**Solution**: Add comprehensive shell completion support for bash, zsh, fish, and PowerShell using Cobra's built-in completion features

**Files to create/modify**:

- `internal/completion/completion.go` - Completion logic and custom completers
- `internal/completion/branch.go` - Branch name completion
- `internal/completion/path.go` - Path completion for worktree directories
- `cmd/grove/main.go` - Register completion commands and custom completers
- `scripts/install-completion.sh` - Installation script for completion
- `docs/COMPLETION.md` - User documentation for completion setup

**Implementation**:

```go
import (
	"strings"

	"github.com/spf13/cobra"
)

// Custom completion for branch names
func BranchCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	branches, err := git.ListBranches()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}

	var suggestions []string
	for _, branch := range branches {
		if strings.HasPrefix(branch, toComplete) {
			suggestions = append(suggestions, branch)
		}
	}

	return suggestions, cobra.ShellCompDirectiveDefault
}

// Register completion for commands
func RegisterCompletion(rootCmd *cobra.Command) {
	// Command completion
	rootCmd.RegisterFlagCompletionFunc("branch", BranchCompletion)

	// Add completion subcommands
	rootCmd.AddCommand(genBashCompletionCmd)
	rootCmd.AddCommand(genZshCompletionCmd)
	rootCmd.AddCommand(genFishCompletionCmd)
	rootCmd.AddCommand(genPowerShellCompletionCmd)
}
```

**Features**:

- Command and subcommand completion
- Flag completion with validation
- Dynamic branch name completion from git repositories
- Worktree directory path completion
- Remote repository URL completion
- Configuration key completion
- Context-aware suggestions based on current repository state

**Testing**:

- Unit tests for completion functions
- Integration tests for shell completion scripts
- Manual testing across different shells (bash, zsh, fish, PowerShell)
- Test completion in various repository states (clean, dirty, detached HEAD)
- Verify completion works with remote repositories
- Test performance with large numbers of branches

---

## Implementation Order

### Phase 1: Quick Wins (Day 1)

1. Add CLI Version Flag
2. Standardize Error Handling

### Phase 2: Infrastructure (Days 2-3)

3. Consolidate Mock Implementations
4. Implement Configuration System
5. Implement Command Registration Framework

### Phase 3: User Experience (Days 4-5)

6. Add Progress Indicators
7. Add Retry Mechanisms
8. Implement CLI Completion Support

### Phase 4: Quality Assurance (Days 6-7)

8. Increase Test Coverage

## Success Criteria

- [ ] All improvements implemented with comprehensive tests
- [ ] No regression in existing functionality
- [ ] Test coverage increased to 90%+
- [ ] All linting and formatting checks pass
- [ ] Documentation updated for new features
- [ ] Integration tests pass consistently

## Notes

- Each improvement should be implemented in a separate commit/PR
- All changes must maintain backward compatibility
- Comprehensive testing required for each improvement
- Update documentation as implementations are completed
