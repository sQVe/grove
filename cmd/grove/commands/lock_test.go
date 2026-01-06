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
	"github.com/sqve/grove/internal/workspace"
)

func TestNewLockCmd(t *testing.T) {
	cmd := NewLockCmd()

	if cmd.Use != "lock <worktree>..." {
		t.Errorf("expected Use 'lock <worktree>...', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	reasonFlag := cmd.Flags().Lookup("reason")
	if reasonFlag == nil {
		t.Fatal("expected --reason flag")
	}
	if reasonFlag.Shorthand != "" {
		t.Errorf("expected reason to have no shorthand (git convention), got %q", reasonFlag.Shorthand)
	}
}

func TestRunLock_NotInWorkspace(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	err := runLock([]string{"some-branch"}, "")
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunLock_BranchNotFound(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	// Setup a Grove workspace
	tempDir := t.TempDir()
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
	_ = os.Chdir(mainPath)

	err := runLock([]string{"nonexistent"}, "")
	if err == nil {
		t.Error("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "worktree not found") {
		t.Errorf("expected 'worktree not found' error, got: %v", err)
	}
}

func TestRunLock_AlreadyLocked(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	// Setup a Grove workspace
	tempDir := t.TempDir()
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
	cmd = exec.Command("git", "worktree", "lock", "--reason", "existing lock", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock worktree: %v", err)
	}

	// Change to main worktree
	_ = os.Chdir(mainPath)

	err := runLock([]string{"feature"}, "new reason")
	if err == nil {
		t.Error("expected error for already locked worktree")
	}
	// Error message is now "failed: feature" (per-item reason logged separately)
	if !strings.Contains(err.Error(), "feature") {
		t.Errorf("expected error to mention 'feature', got: %v", err)
	}
}

func TestRunLock_Success(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	// Setup a Grove workspace
	tempDir := t.TempDir()
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

	// Change to main worktree
	_ = os.Chdir(mainPath)

	err := runLock([]string{"feature"}, "")
	if err != nil {
		t.Fatalf("runLock failed: %v", err)
	}

	// Verify worktree is locked
	if !git.IsWorktreeLocked(featurePath) {
		t.Error("worktree should be locked after runLock")
	}
}

func TestRunLock_SuccessWithReason(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	// Setup a Grove workspace
	tempDir := t.TempDir()
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

	// Change to main worktree
	_ = os.Chdir(mainPath)

	reason := "WIP - do not remove"
	err := runLock([]string{"feature"}, reason)
	if err != nil {
		t.Fatalf("runLock with reason failed: %v", err)
	}

	// Verify worktree is locked with reason
	if !git.IsWorktreeLocked(featurePath) {
		t.Error("worktree should be locked after runLock")
	}
	gotReason := git.GetWorktreeLockReason(featurePath)
	if gotReason != reason {
		t.Errorf("expected lock reason %q, got %q", reason, gotReason)
	}
}

func TestRunLock_MultipleWorktrees(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	// Setup a Grove workspace
	tempDir := t.TempDir()
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

	// Create bugfix worktree
	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}

	// Change to main worktree
	_ = os.Chdir(mainPath)

	// Lock multiple worktrees at once
	err := runLock([]string{"feature", "bugfix"}, "")
	if err != nil {
		t.Fatalf("runLock failed: %v", err)
	}

	// Verify both are locked
	if !git.IsWorktreeLocked(featurePath) {
		t.Error("feature worktree should be locked")
	}
	if !git.IsWorktreeLocked(bugfixPath) {
		t.Error("bugfix worktree should be locked")
	}
}

func TestRunLock_MultipleWithReason(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}

	_ = os.Chdir(mainPath)

	// Lock multiple with same reason
	reason := "WIP - shared reason"
	err := runLock([]string{"feature", "bugfix"}, reason)
	if err != nil {
		t.Fatalf("runLock failed: %v", err)
	}

	// Verify both locked with same reason
	if git.GetWorktreeLockReason(featurePath) != reason {
		t.Errorf("feature should have reason %q", reason)
	}
	if git.GetWorktreeLockReason(bugfixPath) != reason {
		t.Errorf("bugfix should have reason %q", reason)
	}
}

func TestRunLock_MultipleDuplicateArgs(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	_ = os.Chdir(mainPath)

	// Pass same worktree twice - should deduplicate and succeed
	err := runLock([]string{"feature", "feature"}, "")
	if err != nil {
		t.Fatalf("expected success with duplicate args, got: %v", err)
	}

	if !git.IsWorktreeLocked(featurePath) {
		t.Error("feature should be locked")
	}
}

func TestRunLock_DuplicateViaBranchAndDirName(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create worktree where directory name differs from branch name
	// Branch: feature/auth, Directory: feature-auth (slashes become hyphens)
	featurePath := filepath.Join(tempDir, "feature-auth")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature/auth", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	_ = os.Chdir(mainPath)

	// Pass both directory name AND branch name - should deduplicate by path
	err := runLock([]string{"feature-auth", "feature/auth"}, "")
	if err != nil {
		t.Fatalf("expected success with branch/dir duplicate, got: %v", err)
	}

	if !git.IsWorktreeLocked(featurePath) {
		t.Error("feature-auth should be locked")
	}
}

func TestCompleteLockArgs_MultipleArgs(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}

	_ = os.Chdir(mainPath)

	lockCmd := NewLockCmd()

	// First completion - should show all unlocked worktrees
	completions, _ := completeLockArgs(lockCmd, nil, "")
	if len(completions) != 3 {
		t.Errorf("expected 3 completions, got %d: %v", len(completions), completions)
	}

	// Second completion (feature already typed) - should exclude feature
	completions, _ = completeLockArgs(lockCmd, []string{"feature"}, "")
	if len(completions) != 2 {
		t.Errorf("expected 2 completions after 'feature', got %d: %v", len(completions), completions)
	}
	for _, c := range completions {
		if c == "feature" {
			t.Error("completions should not include already-typed 'feature'")
		}
	}
}

func TestCompleteLockArgs(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	// Setup a Grove workspace
	tempDir := t.TempDir()
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
	_ = os.Chdir(mainPath)

	// Get completions
	lockCmd := NewLockCmd()
	completions, directive := completeLockArgs(lockCmd, nil, "")

	// Should include unlocked worktrees (main, feature)
	hasMain := false
	hasFeature := false
	for _, c := range completions {
		if c == "main" {
			hasMain = true
		}
		if c == "feature" {
			hasFeature = true
		}
	}
	if !hasMain {
		t.Error("completions should include unlocked 'main'")
	}
	if !hasFeature {
		t.Error("completions should include unlocked 'feature'")
	}

	// Should NOT include locked worktrees
	for _, c := range completions {
		if c == "bugfix" {
			t.Error("completions should not include locked 'bugfix'")
		}
	}

	// Should disable file completion
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}
