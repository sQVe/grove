package commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutil"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewUnlockCmd(t *testing.T) {
	cmd := NewUnlockCmd()

	if cmd.Use != "unlock <worktree>..." {
		t.Errorf("expected Use 'unlock <worktree>...', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
}

func TestRunUnlock_NotInWorkspace(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tmpDir := testutil.TempDir(t)
	testutil.Chdir(t, tmpDir)

	err := runUnlock([]string{"some-branch"})
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunUnlock_BranchNotFound(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Change to workspace
	testutil.Chdir(t, mainPath)

	err := runUnlock([]string{"nonexistent"})
	if err == nil {
		t.Error("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "worktree not found") {
		t.Errorf("expected 'worktree not found' error, got: %v", err)
	}
}

func TestRunUnlock_NotLocked(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create feature worktree (not locked)
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Change to main worktree
	testutil.Chdir(t, mainPath)

	err := runUnlock([]string{"feature"})
	if err == nil {
		t.Error("expected error for unlocked worktree")
	}
	// Error message is now "failed: feature" (per-item reason logged separately)
	if !strings.Contains(err.Error(), "feature") {
		t.Errorf("expected error to mention 'feature', got: %v", err)
	}
}

func TestRunUnlock_Success(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Lock the feature worktree
	cmd = exec.Command("git", "worktree", "lock", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock worktree: %v", err)
	}

	// Verify it's locked
	if !git.IsWorktreeLocked(featurePath) {
		t.Fatal("worktree should be locked before test")
	}

	// Change to main worktree
	testutil.Chdir(t, mainPath)

	err := runUnlock([]string{"feature"})
	if err != nil {
		t.Fatalf("runUnlock failed: %v", err)
	}

	// Verify worktree is unlocked
	if git.IsWorktreeLocked(featurePath) {
		t.Error("worktree should be unlocked after runUnlock")
	}
}

func TestRunUnlock_MultipleWorktrees(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create and lock feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock feature worktree: %v", err)
	}

	// Create and lock bugfix worktree
	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock bugfix worktree: %v", err)
	}

	// Verify both are locked
	if !git.IsWorktreeLocked(featurePath) || !git.IsWorktreeLocked(bugfixPath) {
		t.Fatal("both worktrees should be locked before test")
	}

	// Change to main worktree
	testutil.Chdir(t, mainPath)

	// Unlock multiple worktrees at once
	err := runUnlock([]string{"feature", "bugfix"})
	if err != nil {
		t.Fatalf("runUnlock failed: %v", err)
	}

	// Verify both are unlocked
	if git.IsWorktreeLocked(featurePath) {
		t.Error("feature worktree should be unlocked")
	}
	if git.IsWorktreeLocked(bugfixPath) {
		t.Error("bugfix worktree should be unlocked")
	}
}

func TestRunUnlock_MultipleWithOneInvalid(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create and lock feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock feature worktree: %v", err)
	}

	testutil.Chdir(t, mainPath)

	// Include a nonexistent worktree - should fail immediately during validation
	err := runUnlock([]string{"feature", "nonexistent"})
	if err == nil {
		t.Error("expected error for invalid worktree")
	}
	if !strings.Contains(err.Error(), "worktree not found") {
		t.Errorf("expected 'worktree not found' error, got: %v", err)
	}

	// Feature should still be locked (validation failed before processing)
	if !git.IsWorktreeLocked(featurePath) {
		t.Error("feature should still be locked after validation failure")
	}
}

func TestRunUnlock_MultipleWithOneNotLocked(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create and lock feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock feature worktree: %v", err)
	}

	// Create bugfix worktree (NOT locked)
	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}

	testutil.Chdir(t, mainPath)

	// Try to unlock both - feature should succeed, bugfix should fail
	err := runUnlock([]string{"feature", "bugfix"})
	if err == nil {
		t.Error("expected error because bugfix is not locked")
	}
	if !strings.Contains(err.Error(), "bugfix") {
		t.Errorf("expected error to mention 'bugfix', got: %v", err)
	}

	// Feature should be unlocked (processed before bugfix failed)
	if git.IsWorktreeLocked(featurePath) {
		t.Error("feature should be unlocked")
	}
}

func TestRunUnlock_MultipleDuplicateArgs(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create and lock feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock feature worktree: %v", err)
	}

	testutil.Chdir(t, mainPath)

	// Pass the same worktree twice - should deduplicate and succeed
	err := runUnlock([]string{"feature", "feature"})
	if err != nil {
		t.Fatalf("expected success with duplicate args, got: %v", err)
	}

	// Feature should be unlocked
	if git.IsWorktreeLocked(featurePath) {
		t.Error("feature should be unlocked")
	}
}

func TestCompleteUnlockArgs_MultipleArgs(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree (unlocked)
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create and lock feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock feature worktree: %v", err)
	}

	// Create and lock bugfix worktree
	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock bugfix worktree: %v", err)
	}

	testutil.Chdir(t, mainPath)

	unlockCmd := NewUnlockCmd()

	// First completion (no args yet) - should show both locked worktrees
	completions, directive := completeUnlockArgs(unlockCmd, nil, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
	if len(completions) != 2 {
		t.Errorf("expected 2 completions, got %d: %v", len(completions), completions)
	}

	// Second completion (feature already typed) - should only show bugfix
	completions, _ = completeUnlockArgs(unlockCmd, []string{"feature"}, "")
	if len(completions) != 1 {
		t.Errorf("expected 1 completion after 'feature', got %d: %v", len(completions), completions)
	}
	if len(completions) > 0 && completions[0] != "bugfix" {
		t.Errorf("expected 'bugfix' completion, got %q", completions[0])
	}
}

func TestCompleteUnlockArgs(t *testing.T) {
	defer testutil.SaveCwd(t)()

	// Setup a Grove workspace
	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree (unlocked)
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create feature worktree (unlocked)
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Create bugfix worktree and lock it
	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock bugfix worktree: %v", err)
	}

	// Change to main worktree
	testutil.Chdir(t, mainPath)

	// Get completions
	unlockCmd := NewUnlockCmd()
	completions, directive := completeUnlockArgs(unlockCmd, nil, "")

	// Should include locked worktrees only (bugfix)
	hasBugfix := false
	for _, c := range completions {
		if c == "bugfix" { //nolint:goconst // test string
			hasBugfix = true
		}
	}
	if !hasBugfix {
		t.Error("completions should include locked 'bugfix'")
	}

	// Should NOT include unlocked worktrees
	for _, c := range completions {
		if c == "main" || c == "feature" { //nolint:goconst // test strings
			t.Errorf("completions should not include unlocked %q", c)
		}
	}

	// Should disable file completion
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}
