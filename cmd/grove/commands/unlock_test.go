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

func TestNewUnlockCmd(t *testing.T) {
	cmd := NewUnlockCmd()

	if cmd.Use != "unlock <branch>" {
		t.Errorf("expected Use 'unlock <branch>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
}

func TestRunUnlock_NotInWorkspace(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	err := runUnlock("some-branch")
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunUnlock_BranchNotFound(t *testing.T) {
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

	err := runUnlock("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "no worktree found") {
		t.Errorf("expected 'no worktree found' error, got: %v", err)
	}
}

func TestRunUnlock_NotLocked(t *testing.T) {
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

	// Create feature worktree (not locked)
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Change to main worktree
	_ = os.Chdir(mainPath)

	err := runUnlock("feature")
	if err == nil {
		t.Error("expected error for unlocked worktree")
	}
	if !strings.Contains(err.Error(), "not locked") {
		t.Errorf("expected 'not locked' error, got: %v", err)
	}
}

func TestRunUnlock_Success(t *testing.T) {
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
	_ = os.Chdir(mainPath)

	err := runUnlock("feature")
	if err != nil {
		t.Fatalf("runUnlock failed: %v", err)
	}

	// Verify worktree is unlocked
	if git.IsWorktreeLocked(featurePath) {
		t.Error("worktree should be unlocked after runUnlock")
	}
}

func TestCompleteUnlockArgs(t *testing.T) {
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
	_ = os.Chdir(mainPath)

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
