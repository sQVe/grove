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

func TestNewDeleteCmd(t *testing.T) {
	cmd := NewDeleteCmd()

	if cmd.Use != "delete <branch>" {
		t.Errorf("expected Use 'delete <branch>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	if cmd.Flags().Lookup("force") == nil {
		t.Error("expected --force flag")
	}
	if cmd.Flags().Lookup("branch") == nil {
		t.Error("expected --branch flag")
	}
}

func TestRunDelete_NotInWorkspace(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	err := runDelete("some-branch", false, false)
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunDelete_BranchNotFound(t *testing.T) {
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

	err := runDelete("nonexistent", false, false)
	if err == nil {
		t.Error("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "no worktree found") {
		t.Errorf("expected 'no worktree found' error, got: %v", err)
	}
}

func TestRunDelete_CurrentWorktree(t *testing.T) {
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

	// Change to workspace (the worktree we'll try to delete)
	_ = os.Chdir(mainPath)

	err := runDelete("main", false, false)
	if err == nil {
		t.Error("expected error when deleting current worktree")
	}
	if !strings.Contains(err.Error(), "current worktree") {
		t.Errorf("expected 'current worktree' error, got: %v", err)
	}
}

func TestRunDelete_DirtyWorktree(t *testing.T) {
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

	// Make feature worktree dirty
	dirtyFile := filepath.Join(featurePath, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("dirty"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}

	// Change to main worktree (not the one we're deleting)
	_ = os.Chdir(mainPath)

	err := runDelete("feature", false, false)
	if err == nil {
		t.Error("expected error for dirty worktree")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") && !strings.Contains(err.Error(), "dirty") {
		t.Errorf("expected error about uncommitted changes, got: %v", err)
	}
}

func TestRunDelete_LockedWorktree(t *testing.T) {
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

	// Change to main worktree
	_ = os.Chdir(mainPath)

	err := runDelete("feature", false, false)
	if err == nil {
		t.Error("expected error for locked worktree")
	}
	if !strings.Contains(err.Error(), "locked") {
		t.Errorf("expected error about locked worktree, got: %v", err)
	}
}

func TestRunDelete_Success(t *testing.T) {
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

	// Create initial commit (needed for branch refs to work correctly)
	testFile := filepath.Join(mainPath, "init.txt")
	if err := os.WriteFile(testFile, []byte("init"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "init.txt") //nolint:gosec
	cmd.Dir = mainPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "initial commit") //nolint:gosec
	cmd.Dir = mainPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
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

	// Verify worktree exists
	if _, err := os.Stat(featurePath); os.IsNotExist(err) {
		t.Fatal("feature worktree should exist before deletion")
	}

	err := runDelete("feature", false, false)
	if err != nil {
		t.Fatalf("runDelete failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after deletion")
	}

	// Verify branch still exists (--branch not used)
	exists, err := git.BranchExists(bareDir, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		t.Error("feature branch should still exist when --branch not used")
	}
}

func TestRunDelete_ForceDirty(t *testing.T) {
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

	// Make feature worktree dirty
	dirtyFile := filepath.Join(featurePath, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("dirty"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}

	// Change to main worktree
	_ = os.Chdir(mainPath)

	// Force delete dirty worktree
	err := runDelete("feature", true, false)
	if err != nil {
		t.Fatalf("runDelete with force failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after forced deletion")
	}
}

func TestRunDelete_ForceLocked(t *testing.T) {
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

	// Change to main worktree
	_ = os.Chdir(mainPath)

	// Force delete locked worktree
	err := runDelete("feature", true, false)
	if err != nil {
		t.Fatalf("runDelete with force failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after forced deletion")
	}
}

func TestRunDelete_WithBranchFlag(t *testing.T) {
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

	// Create main worktree with initial commit
	mainPath := filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Create initial commit so branches can be created
	testFile := filepath.Join(mainPath, "init.txt")
	if err := os.WriteFile(testFile, []byte("init"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "init.txt") //nolint:gosec
	cmd.Dir = mainPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add file: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "initial commit") //nolint:gosec
	cmd.Dir = mainPath
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
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

	// Delete with --branch flag
	err := runDelete("feature", false, true)
	if err != nil {
		t.Fatalf("runDelete with --branch failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after deletion")
	}

	// Verify branch is also deleted
	exists, err := git.BranchExists(bareDir, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("feature branch should not exist when --branch was used")
	}
}

func TestCompleteDeleteArgs(t *testing.T) {
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

	// Get completions
	deleteCmd := NewDeleteCmd()
	completions, directive := completeDeleteArgs(deleteCmd, nil, "")

	// Should not include current worktree (main)
	for _, c := range completions {
		if c == "main" {
			t.Error("completions should not include current worktree")
		}
	}

	// Should include feature and bugfix
	hasFeature := false
	hasBugfix := false
	for _, c := range completions {
		if c == "feature" {
			hasFeature = true
		}
		if c == "bugfix" {
			hasBugfix = true
		}
	}
	if !hasFeature {
		t.Error("completions should include 'feature'")
	}
	if !hasBugfix {
		t.Error("completions should include 'bugfix'")
	}

	// Should disable file completion
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
	}
}
