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

	if cmd.Use != "lock <worktree>" {
		t.Errorf("expected Use 'lock <worktree>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	if cmd.Flags().Lookup("reason") == nil {
		t.Error("expected --reason flag")
	}
}

func TestRunLock_NotInWorkspace(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	err := runLock("some-branch", "")
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

	err := runLock("nonexistent", "")
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

	err := runLock("feature", "new reason")
	if err == nil {
		t.Error("expected error for already locked worktree")
	}
	if !strings.Contains(err.Error(), "already locked") {
		t.Errorf("expected 'already locked' error, got: %v", err)
	}
	// Should include the existing reason
	if !strings.Contains(err.Error(), "existing lock") {
		t.Errorf("expected error to include existing reason, got: %v", err)
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

	err := runLock("feature", "")
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
	err := runLock("feature", reason)
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
