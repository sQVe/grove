# List Command Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Implement `grove list` command showing all worktrees with status, sync info, and optional JSON output.

**Architecture:** New command file with worktree info struct. Git package gets new functions for sync status (ahead/behind/gone). Logger gets worktree list output functions. TDD approach with unit tests first, integration tests last.

**Tech Stack:** Go, Cobra CLI, lipgloss styling, git commands (status, rev-list, rev-parse)

---

## Design Decisions (from Codex review)

1. **Upstream name in --verbose**: Add `Upstream` field to WorktreeInfo, show in verbose mode
2. **Workspace detection**: Add `FindBareDir` to workspace package (reuse walk-up logic from `IsInsideGroveWorkspace`)
3. **Output ordering**: Sort worktrees alphabetically, current branch first
4. **Skipped worktrees**: Log warning via `logger.Debug` when worktree is skipped (detached HEAD, errors)
5. **Sync status note**: Reports local knowledge only (no fetch). Document this in --help
6. **Integration tests**: Use repo-local git config, not --global. Add --verbose and error path tests
7. **JSON + fast interaction**: --fast with --json still returns all fields, but dirty/sync fields are zero-values

---

## Task 1: Add WorktreeInfo Type to Git Package

**Files:**

-   Modify: `internal/git/git.go`
-   Test: `internal/git/git_test.go`

**Step 1: Write failing test for GetWorktreeInfo**

```go
import "testing"

func TestGetWorktreeInfo(t *testing.T) {
	t.Parallel()

	t.Run("returns info for clean worktree in sync", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		info, err := git.GetWorktreeInfo(repo.Path)
		if err != nil {
			t.Fatalf("GetWorktreeInfo failed: %v", err)
		}

		if info.Branch != "main" && info.Branch != "master" {
			t.Errorf("expected branch main or master, got %s", info.Branch)
		}
		if info.Dirty {
			t.Error("expected clean worktree")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/git/... -run TestGetWorktreeInfo -v`
Expected: FAIL with "GetWorktreeInfo not defined"

**Step 3: Write WorktreeInfo type and GetWorktreeInfo function**

Add to `internal/git/git.go`:

```go
import (
	"errors"
	"fmt"
)

// WorktreeInfo contains status information about a worktree
type WorktreeInfo struct {
	Path       string // Absolute path to worktree
	Branch     string // Branch name
	Upstream   string // Upstream branch name (e.g., "origin/main")
	Dirty      bool   // Has uncommitted changes
	Ahead      int    // Commits ahead of upstream
	Behind     int    // Commits behind upstream
	Gone       bool   // Upstream branch deleted
	NoUpstream bool   // No upstream configured
}

// GetWorktreeInfo returns status information for a worktree
func GetWorktreeInfo(path string) (*WorktreeInfo, error) {
	if path == "" {
		return nil, errors.New("worktree path cannot be empty")
	}

	info := &WorktreeInfo{Path: path}

	// Get branch name
	branch, err := GetCurrentBranch(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get branch: %w", err)
	}
	info.Branch = branch

	// Check for dirty state
	hasChanges, _, err := CheckGitChanges(path)
	if err != nil {
		return nil, fmt.Errorf("failed to check changes: %w", err)
	}
	info.Dirty = hasChanges

	return info, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/git/... -run TestGetWorktreeInfo -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "$(
  cat << 'EOF'
feat(git): add WorktreeInfo type and GetWorktreeInfo function

Basic worktree info retrieval without sync status.
EOF
)"
```

---

## Task 2: Add Sync Status to WorktreeInfo

**Files:**

-   Modify: `internal/git/git.go`
-   Test: `internal/git/git_test.go`

**Step 1: Write failing test for ahead/behind detection**

```go
import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGetWorktreeInfo_SyncStatus(t *testing.T) {
	t.Parallel()

	t.Run("detects commits ahead of upstream", func(t *testing.T) {
		t.Parallel()
		// Create bare repo to act as remote
		remoteDir := t.TempDir()
		remoteRepo := filepath.Join(remoteDir, "remote.git")
		if err := os.MkdirAll(remoteRepo, 0755); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = remoteRepo
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create local repo and push
		repo := testgit.NewTestRepo(t)
		cmd = exec.Command("git", "remote", "add", "origin", remoteRepo)
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			// Try master if main fails
			cmd = exec.Command("git", "push", "-u", "origin", "master")
			cmd.Dir = repo.Path
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
		}

		// Create local commit
		testFile := filepath.Join(repo.Path, "new.txt")
		if err := os.WriteFile(testFile, []byte("new"), 0644); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = repo.Path
		_ = cmd.Run()
		cmd = exec.Command("git", "commit", "-m", "local commit")
		cmd.Dir = repo.Path
		_ = cmd.Run()

		info, err := git.GetWorktreeInfo(repo.Path)
		if err != nil {
			t.Fatalf("GetWorktreeInfo failed: %v", err)
		}

		if info.Ahead != 1 {
			t.Errorf("expected 1 commit ahead, got %d", info.Ahead)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/git/... -run TestGetWorktreeInfo_SyncStatus -v`
Expected: FAIL with "expected 1 commit ahead, got 0"

**Step 3: Add GetSyncStatus function and integrate**

Add to `internal/git/git.go`:

```go
import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// GetSyncStatus returns ahead/behind counts and upstream name relative to upstream
func GetSyncStatus(path string) (upstream string, ahead, behind int, gone, noUpstream bool, err error) {
	// Check if upstream is configured and get its name
	cmdUpstream := exec.Command("git", "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	cmdUpstream.Dir = path
	var upstreamOut, upstreamStderr bytes.Buffer
	cmdUpstream.Stdout = &upstreamOut
	cmdUpstream.Stderr = &upstreamStderr

	if err := cmdUpstream.Run(); err != nil {
		// No upstream configured
		return "", 0, 0, false, true, nil
	}
	upstream = strings.TrimSpace(upstreamOut.String())

	// Check if upstream still exists (gone detection)
	cmdCheck := exec.Command("git", "rev-parse", "@{u}")
	cmdCheck.Dir = path
	if err := cmdCheck.Run(); err != nil {
		return upstream, 0, 0, true, false, nil
	}

	// Get ahead count
	cmdAhead := exec.Command("git", "rev-list", "--count", "@{u}..HEAD")
	cmdAhead.Dir = path
	var aheadOut bytes.Buffer
	cmdAhead.Stdout = &aheadOut
	if err := cmdAhead.Run(); err == nil {
		fmt.Sscanf(strings.TrimSpace(aheadOut.String()), "%d", &ahead)
	}

	// Get behind count
	cmdBehind := exec.Command("git", "rev-list", "--count", "HEAD..@{u}")
	cmdBehind.Dir = path
	var behindOut bytes.Buffer
	cmdBehind.Stdout = &behindOut
	if err := cmdBehind.Run(); err == nil {
		fmt.Sscanf(strings.TrimSpace(behindOut.String()), "%d", &behind)
	}

	return upstream, ahead, behind, false, false, nil
}
```

Update `GetWorktreeInfo` to call `GetSyncStatus`:

```go
// In GetWorktreeInfo, after dirty check:
upstream, ahead, behind, gone, noUpstream, err := GetSyncStatus(path)
if err != nil {
	return nil, fmt.Errorf("failed to get sync status: %w", err)
}
info.Upstream = upstream
info.Ahead = ahead
info.Behind = behind
info.Gone = gone
info.NoUpstream = noUpstream
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/git/... -run TestGetWorktreeInfo_SyncStatus -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "$(
  cat << 'EOF'
feat(git): add sync status detection (ahead/behind/gone)

GetSyncStatus returns commit counts relative to upstream.
EOF
)"
```

---

## Task 3: Add ListWorktreesWithInfo Function

**Files:**

-   Modify: `internal/git/git.go`
-   Test: `internal/git/git_test.go`

**Step 1: Write failing test**

```go
import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestListWorktreesWithInfo(t *testing.T) {
	t.Parallel()

	t.Run("returns worktree info for grove workspace", func(t *testing.T) {
		t.Parallel()
		// Setup: create a grove-style workspace with bare repo and worktrees
		workspaceDir := t.TempDir()
		bareDir := filepath.Join(workspaceDir, ".bare")

		// Init bare repo
		cmd := exec.Command("git", "init", "--bare", bareDir)
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create initial commit in bare repo via temp clone
		tempDir := t.TempDir()
		cmd = exec.Command("git", "clone", bareDir, tempDir)
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "config", "user.email", "test@test.com")
		cmd.Dir = tempDir
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = tempDir
		_ = cmd.Run()
		testFile := filepath.Join(tempDir, "test.txt")
		_ = os.WriteFile(testFile, []byte("test"), 0644)
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		_ = cmd.Run()
		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = tempDir
		_ = cmd.Run()
		cmd = exec.Command("git", "push", "origin", "HEAD")
		cmd.Dir = tempDir
		_ = cmd.Run()

		// Create worktree
		worktreePath := filepath.Join(workspaceDir, "main")
		cmd = exec.Command("git", "worktree", "add", worktreePath, "main")
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			// Try master
			cmd = exec.Command("git", "worktree", "add", worktreePath, "master")
			cmd.Dir = bareDir
			if err := cmd.Run(); err != nil {
				t.Fatal(err)
			}
		}

		infos, err := git.ListWorktreesWithInfo(bareDir, false)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo failed: %v", err)
		}

		if len(infos) != 1 {
			t.Fatalf("expected 1 worktree, got %d", len(infos))
		}

		if infos[0].Path != worktreePath {
			t.Errorf("expected path %s, got %s", worktreePath, infos[0].Path)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/git/... -run TestListWorktreesWithInfo -v`
Expected: FAIL with "ListWorktreesWithInfo not defined"

**Step 3: Implement ListWorktreesWithInfo**

```go
import (
	"sort"

	"github.com/sqve/grove/internal/logger"
)

// ListWorktreesWithInfo returns info for all worktrees in a grove workspace
// Results are sorted alphabetically by branch name
func ListWorktreesWithInfo(bareDir string, fast bool) ([]*WorktreeInfo, error) {
	paths, err := ListWorktrees(bareDir)
	if err != nil {
		return nil, err
	}

	var infos []*WorktreeInfo
	for _, path := range paths {
		if fast {
			// Fast mode: only get branch name
			branch, err := GetCurrentBranch(path)
			if err != nil {
				logger.Debug("Skipping worktree %s: %v", path, err)
				continue
			}
			infos = append(infos, &WorktreeInfo{
				Path:   path,
				Branch: branch,
			})
		} else {
			info, err := GetWorktreeInfo(path)
			if err != nil {
				logger.Debug("Skipping worktree %s: %v", path, err)
				continue
			}
			infos = append(infos, info)
		}
	}

	// Sort alphabetically by branch name
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Branch < infos[j].Branch
	})

	return infos, nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/git/... -run TestListWorktreesWithInfo -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/git/git.go internal/git/git_test.go
git commit -m "$(
  cat << 'EOF'
feat(git): add ListWorktreesWithInfo for worktree enumeration

Supports fast mode to skip status checks.
EOF
)"
```

---

## Task 4: Add Logger Functions for List Output

**Files:**

-   Modify: `internal/logger/logger.go`
-   Test: `internal/logger/logger_test.go`

**Step 1: Write failing test for WorktreeListItem**

```go
import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/config"
)

func TestWorktreeListItem(t *testing.T) {
	t.Run("formats current worktree with bullet", func(t *testing.T) {
		config.Global.Plain = true
		defer func() { config.Global.Plain = false }()

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		WorktreeListItem("main", true, "[clean]", "=")

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()

		if !strings.Contains(output, "* main") {
			t.Errorf("expected '* main' in output, got: %s", output)
		}
		if !strings.Contains(output, "[clean]") {
			t.Errorf("expected '[clean]' in output, got: %s", output)
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/logger/... -run TestWorktreeListItem -v`
Expected: FAIL with "WorktreeListItem not defined"

**Step 3: Implement WorktreeListItem**

Add to `internal/logger/logger.go`:

```go
import (
	"fmt"
	"strings"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/styles"
)

// WorktreeListItem prints a worktree entry in list format
func WorktreeListItem(name string, current bool, status, sync string) {
	marker := " "
	if current {
		marker = "*"
	}

	if config.IsPlain() {
		fmt.Printf("%s %-20s %s %s\n", marker, name, status, sync)
	} else {
		markerStyled := marker
		if current {
			markerStyled = styles.Render(&styles.Success, "●")
		}
		nameStyled := styles.Render(&styles.Worktree, name)
		statusStyled := status
		if strings.Contains(status, "dirty") {
			statusStyled = styles.Render(&styles.Warning, status)
		} else {
			statusStyled = styles.Render(&styles.Dimmed, status)
		}
		fmt.Printf("%s %-20s %s %s\n", markerStyled, nameStyled, statusStyled, sync)
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/logger/... -run TestWorktreeListItem -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/logger/logger.go internal/logger/logger_test.go
git commit -m "$(
  cat << 'EOF'
feat(logger): add WorktreeListItem for list command output
EOF
)"
```

---

## Task 5: Create List Command Structure

**Files:**

-   Create: `cmd/grove/commands/list.go`
-   Test: `cmd/grove/commands/list_test.go`

**Step 1: Write failing test for command existence**

```go
package commands

import (
	"testing"
)

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()

	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("fast") == nil {
		t.Error("expected --fast flag")
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Error("expected --json flag")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./cmd/grove/commands/... -run TestNewListCmd -v`
Expected: FAIL with "NewListCmd not defined"

**Step 3: Implement basic command structure**

Create `cmd/grove/commands/list.go`:

```go
package commands

import (
	"github.com/spf13/cobra"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all worktrees with status",
		Long:  `Show all worktrees in the grove workspace with their status and sync information.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			fast, _ := cmd.Flags().GetBool("fast")
			jsonOutput, _ := cmd.Flags().GetBool("json")
			verbose, _ := cmd.Flags().GetBool("verbose")
			return runList(fast, jsonOutput, verbose)
		},
	}

	cmd.Flags().Bool("fast", false, "Skip sync status for faster output")
	cmd.Flags().Bool("json", false, "Output in JSON format")
	cmd.Flags().Bool("verbose", false, "Show extra details (paths, upstream names)")

	return cmd
}

func runList(fast, jsonOutput, verbose bool) error {
	// TODO: implement
	return nil
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./cmd/grove/commands/... -run TestNewListCmd -v`
Expected: PASS

**Step 5: Commit**

```bash
git add cmd/grove/commands/list.go cmd/grove/commands/list_test.go
git commit -m "$(
  cat << 'EOF'
feat(list): add list command structure with flags
EOF
)"
```

---

## Task 6: Wire List Command to Root

**Files:**

-   Modify: `cmd/grove/main.go`

**Step 1: Read current main.go to understand structure**

Run: Read `cmd/grove/main.go`

**Step 2: Add list command to root**

Add import and command registration similar to other commands.

**Step 3: Verify build succeeds**

Run: `mage build`
Expected: Build succeeds

**Step 4: Verify help shows list command**

Run: `./bin/grove --help`
Expected: Shows "list" in available commands

**Step 5: Commit**

```bash
git add cmd/grove/main.go
git commit -m "$(
  cat << 'EOF'
feat(list): wire list command to root
EOF
)"
```

---

## Task 7: Add FindBareDir to Workspace Package

**Files:**

-   Modify: `internal/workspace/workspace.go`
-   Test: `internal/workspace/workspace_test.go`

**Step 1: Write failing test**

```go
func TestFindBareDir(t *testing.T) {
	t.Parallel()

	t.Run("returns bare dir path from workspace root", func(t *testing.T) {
		t.Parallel()
		workspaceDir := t.TempDir()
		bareDir := filepath.Join(workspaceDir, ".bare")
		if err := os.MkdirAll(bareDir, 0755); err != nil {
			t.Fatal(err)
		}

		result, err := workspace.FindBareDir(workspaceDir)
		if err != nil {
			t.Fatalf("FindBareDir failed: %v", err)
		}
		if result != bareDir {
			t.Errorf("expected %s, got %s", bareDir, result)
		}
	})

	t.Run("returns bare dir from subdirectory", func(t *testing.T) {
		t.Parallel()
		workspaceDir := t.TempDir()
		bareDir := filepath.Join(workspaceDir, ".bare")
		subDir := filepath.Join(workspaceDir, "main", "src")
		if err := os.MkdirAll(bareDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(subDir, 0755); err != nil {
			t.Fatal(err)
		}

		result, err := workspace.FindBareDir(subDir)
		if err != nil {
			t.Fatalf("FindBareDir failed: %v", err)
		}
		if result != bareDir {
			t.Errorf("expected %s, got %s", bareDir, result)
		}
	})

	t.Run("returns error outside workspace", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()

		_, err := workspace.FindBareDir(dir)
		if err == nil {
			t.Error("expected error for non-workspace dir")
		}
	})
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/workspace/... -run TestFindBareDir -v`
Expected: FAIL with "FindBareDir not defined"

**Step 3: Implement FindBareDir**

Add to `internal/workspace/workspace.go`:

```go
// ErrNotInWorkspace is returned when not inside a grove workspace
var ErrNotInWorkspace = errors.New("not in a grove workspace")

// FindBareDir finds the .bare directory for a grove workspace
// by walking up the directory tree from the given path
func FindBareDir(startPath string) (string, error) {
	absPath, err := filepath.Abs(startPath)
	if err != nil {
		return "", err
	}

	dir := absPath
	for {
		bareDir := filepath.Join(dir, ".bare")
		if fs.DirectoryExists(bareDir) {
			return bareDir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", ErrNotInWorkspace
		}
		dir = parent
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/workspace/... -run TestFindBareDir -v`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/workspace/workspace.go internal/workspace/workspace_test.go
git commit -m "$(
  cat << 'EOF'
feat(workspace): add FindBareDir for workspace detection
EOF
)"
```

---

## Task 8: Implement List Core Logic

**Files:**

-   Modify: `cmd/grove/commands/list.go`
-   Test: `cmd/grove/commands/list_test.go`

**Step 1: Write failing test for list output**

```go
import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunList(t *testing.T) {
	t.Run("returns error when not in workspace", func(t *testing.T) {
		// Save and restore cwd
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)

		tmpDir := t.TempDir()
		os.Chdir(tmpDir)

		err := runList(false, false, false)
		if err == nil {
			t.Error("expected error for non-workspace directory")
		}
	})

	t.Run("lists worktrees in grove workspace", func(t *testing.T) {
		// Save and restore cwd
		origDir, _ := os.Getwd()
		defer os.Chdir(origDir)

		// Setup grove workspace
		workspaceDir := t.TempDir()
		bareDir := filepath.Join(workspaceDir, ".bare")

		cmd := exec.Command("git", "init", "--bare", bareDir)
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create initial commit via temp clone
		tempDir := t.TempDir()
		cmd = exec.Command("git", "clone", bareDir, tempDir)
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.email", "test@test.com")
		cmd.Dir = tempDir
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = tempDir
		_ = cmd.Run()
		_ = os.WriteFile(filepath.Join(tempDir, "test.txt"), []byte("test"), 0644)
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		_ = cmd.Run()
		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = tempDir
		_ = cmd.Run()
		cmd = exec.Command("git", "push", "origin", "HEAD")
		cmd.Dir = tempDir
		_ = cmd.Run()

		// Create worktree
		worktreePath := filepath.Join(workspaceDir, "main")
		cmd = exec.Command("git", "worktree", "add", worktreePath, "main")
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			cmd = exec.Command("git", "worktree", "add", worktreePath, "master")
			cmd.Dir = bareDir
			_ = cmd.Run()
		}

		os.Chdir(worktreePath)

		err := runList(false, false, false)
		if err != nil {
			t.Errorf("runList failed: %v", err)
		}
	})
}
```

**Step 2: Implement runList function**

```go
import (
	"fmt"
	"os"

	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/workspace"
)

func runList(fast, jsonOutput, verbose bool) error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	bareDir, err := workspace.FindBareDir(cwd)
	if err != nil {
		return err
	}

	// Get current worktree to mark it
	currentBranch := ""
	if git.IsWorktree(cwd) {
		currentBranch, _ = git.GetCurrentBranch(cwd)
	}

	// Get worktree info
	infos, err := git.ListWorktreesWithInfo(bareDir, fast)
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if jsonOutput {
		return outputJSON(infos, currentBranch)
	}

	return outputTable(infos, currentBranch, fast, verbose)
}
```

**Step 3: Run tests**

Run: `go test ./cmd/grove/commands/... -run TestRunList -v`
Expected: PASS

**Step 4: Commit**

```bash
git add cmd/grove/commands/list.go cmd/grove/commands/list_test.go
git commit -m "$(
  cat << 'EOF'
feat(list): implement core listing logic
EOF
)"
```

---

## Task 9: Implement Table Output

**Files:**

-   Modify: `cmd/grove/commands/list.go`

**Step 1: Implement formatSyncStatus helper**

```go
import (
	"fmt"
	"strings"

	"github.com/sqve/grove/internal/styles"
)

func formatSyncStatus(info *git.WorktreeInfo, plain bool) string {
	if info.Gone {
		if plain {
			return "gone"
		}
		return styles.Render(&styles.Error, "×")
	}
	if info.NoUpstream {
		return ""
	}
	if info.Ahead == 0 && info.Behind == 0 {
		return "="
	}

	var parts []string
	if info.Ahead > 0 {
		if plain {
			parts = append(parts, fmt.Sprintf("+%d", info.Ahead))
		} else {
			parts = append(parts, styles.Render(&styles.Success, fmt.Sprintf("↑%d", info.Ahead)))
		}
	}
	if info.Behind > 0 {
		if plain {
			parts = append(parts, fmt.Sprintf("-%d", info.Behind))
		} else {
			parts = append(parts, styles.Render(&styles.Warning, fmt.Sprintf("↓%d", info.Behind)))
		}
	}
	return strings.Join(parts, "")
}
```

**Step 2: Implement outputTable**

```go
import (
	"sort"

	"github.com/sqve/grove/internal/config"
	"github.com/sqve/grove/internal/logger"
)

func outputTable(infos []*git.WorktreeInfo, currentBranch string, fast, verbose bool) error {
	// Sort: current branch first, then alphabetically
	sort.SliceStable(infos, func(i, j int) bool {
		iCurrent := infos[i].Branch == currentBranch
		jCurrent := infos[j].Branch == currentBranch
		if iCurrent != jCurrent {
			return iCurrent // Current branch comes first
		}
		return false // Keep alphabetical order from ListWorktreesWithInfo
	})

	for _, info := range infos {
		isCurrent := info.Branch == currentBranch

		status := ""
		sync := ""
		if !fast {
			if info.Dirty {
				status = "[dirty]"
			} else {
				status = "[clean]"
			}
			sync = formatSyncStatus(info, config.IsPlain())
		}

		logger.WorktreeListItem(info.Branch, isCurrent, status, sync)

		if verbose {
			logger.ListSubItem(info.Path)
			if info.Upstream != "" {
				logger.ListSubItem("upstream: %s", info.Upstream)
			}
		}
	}
	return nil
}
```

**Step 3: Test manually**

Run: `mage build && ./bin/grove list`

**Step 4: Commit**

```bash
git add cmd/grove/commands/list.go
git commit -m "$(
  cat << 'EOF'
feat(list): implement table output with status formatting
EOF
)"
```

---

## Task 10: Implement JSON Output

**Files:**

-   Modify: `cmd/grove/commands/list.go`

**Step 1: Define JSON output structure**

```go
type worktreeJSON struct {
	Name       string `json:"name"`
	Path       string `json:"path"`
	Current    bool   `json:"current"`
	Upstream   string `json:"upstream,omitempty"`
	Dirty      bool   `json:"dirty,omitempty"`
	Ahead      int    `json:"ahead,omitempty"`
	Behind     int    `json:"behind,omitempty"`
	Gone       bool   `json:"gone,omitempty"`
	NoUpstream bool   `json:"no_upstream,omitempty"`
}
```

**Step 2: Implement outputJSON**

```go
import (
	"encoding/json"
	"os"
)

func outputJSON(infos []*git.WorktreeInfo, currentBranch string) error {
	var output []worktreeJSON
	for _, info := range infos {
		output = append(output, worktreeJSON{
			Name:       info.Branch,
			Path:       info.Path,
			Current:    info.Branch == currentBranch,
			Upstream:   info.Upstream,
			Dirty:      info.Dirty,
			Ahead:      info.Ahead,
			Behind:     info.Behind,
			Gone:       info.Gone,
			NoUpstream: info.NoUpstream,
		})
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	return encoder.Encode(output)
}
```

**Step 3: Test manually**

Run: `./bin/grove list --json | jq`

**Step 4: Commit**

```bash
git add cmd/grove/commands/list.go
git commit -m "$(
  cat << 'EOF'
feat(list): implement JSON output format
EOF
)"
```

---

## Task 11: Add Integration Tests

**Files:**

-   Create: `cmd/grove/testdata/script/list_integration.txt`

**Step 1: Write integration test**

```
# Create grove workspace via clone
mkdir testrepo
exec git init testrepo
cd testrepo
# Use repo-local config (not global) to avoid CI conflicts
exec git config user.name "Test"
exec git config user.email "test@example.com"
exec git config commit.gpgsign false
cp ../README.md .
exec git add .
exec git commit -m 'initial commit'
exec git checkout -b develop
cd ..

mkdir workspace
exec grove clone file://$WORK/testrepo workspace --branches main,develop
cd workspace/main
# Set local git config in worktree too
exec git config user.name "Test"
exec git config user.email "test@example.com"

## Basic list tests

# Test basic list shows branches and status
exec grove list
stdout 'main'
stdout 'develop'
stdout '\[clean\]'

# Test list with --fast skips status
exec grove list --fast
stdout 'main'
stdout 'develop'
! stdout '\[clean\]'
! stdout '\[dirty\]'

# Test list with --json
exec grove list --json
stdout '"name":'
stdout '"main"'
stdout '"develop"'
stdout '"path":'

# Test list with --verbose shows path and upstream
exec grove list --verbose
stdout 'main'
stdout 'develop'
stdout 'workspace/main'

## Dirty detection

# Create uncommitted file
cp ../../README.md dirty.txt
exec grove list
stdout '\[dirty\]'

# Clean up for next test
rm dirty.txt

## Error handling

# Test not in workspace error
cd /tmp
! exec grove list
stderr 'not in a grove workspace'

-- README.md --
# Test
```

**Step 2: Run integration tests**

Run: `mage test:integration`
Expected: PASS

**Step 3: Commit**

```bash
git add cmd/grove/testdata/script/list_integration.txt
git commit -m "$(
  cat << 'EOF'
test(list): add integration tests for list command
EOF
)"
```

---

## Task 12: Update ROADMAP

**Files:**

-   Modify: `ROADMAP.md`

**Step 1: Update list command status**

Mark all list features as complete in the roadmap table.

**Step 2: Update global features table**

Mark list row with [x] for Beautify, --plain, --debug, --help.

**Step 3: Commit**

```bash
git add ROADMAP.md
git commit -m "$(
  cat << 'EOF'
docs: mark list command as complete in roadmap
EOF
)"
```

---

## Critical Files Reference

**Must read before implementation:**

-   `internal/git/git.go` - Existing git helpers pattern
-   `internal/logger/logger.go` - Output formatting pattern
-   `internal/workspace/workspace.go` - Workspace detection pattern (IsInsideGroveWorkspace)
-   `cmd/grove/commands/config.go` - Command structure pattern
-   `cmd/grove/testdata/script/clone_integration.txt` - Integration test pattern

**Must modify:**

-   `internal/git/git.go` - Add WorktreeInfo, GetWorktreeInfo, GetSyncStatus, ListWorktreesWithInfo
-   `internal/workspace/workspace.go` - Add FindBareDir, ErrNotInWorkspace
-   `internal/logger/logger.go` - Add WorktreeListItem
-   `cmd/grove/commands/list.go` - New file
-   `cmd/grove/main.go` - Wire command
-   `ROADMAP.md` - Update status

## Task Summary

1. Add WorktreeInfo type to git package
2. Add sync status (ahead/behind/gone/upstream) to WorktreeInfo
3. Add ListWorktreesWithInfo function with sorting
4. Add logger functions for list output
5. Create list command structure
6. Wire list command to root
7. Add FindBareDir to workspace package
8. Implement list core logic
9. Implement table output with current-first sorting
10. Implement JSON output
11. Add integration tests
12. Update ROADMAP
