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

func TestNewRemoveCmd(t *testing.T) {
	cmd := NewRemoveCmd()

	if cmd.Use != "remove <branch>" {
		t.Errorf("expected Use 'remove <branch>', got %q", cmd.Use)
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

func TestRunRemove_NotInWorkspace(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	_ = os.Chdir(tmpDir)

	err := runRemove("some-branch", false, false)
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunRemove_BranchNotFound(t *testing.T) {
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

	err := runRemove("nonexistent", false, false)
	if err == nil {
		t.Error("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "no worktree found") {
		t.Errorf("expected 'no worktree found' error, got: %v", err)
	}
}

func TestRunRemove_CurrentWorktree(t *testing.T) {
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

	// Change to workspace (the worktree we'll try to remove)
	_ = os.Chdir(mainPath)

	err := runRemove("main", false, false)
	if err == nil {
		t.Error("expected error when removing current worktree")
	}
	if !strings.Contains(err.Error(), "current worktree") {
		t.Errorf("expected 'current worktree' error, got: %v", err)
	}
}

func TestRunRemove_DirtyWorktree(t *testing.T) {
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

	// Change to main worktree (not the one we're removing)
	_ = os.Chdir(mainPath)

	err := runRemove("feature", false, false)
	if err == nil {
		t.Error("expected error for dirty worktree")
	}
	if !strings.Contains(err.Error(), "uncommitted changes") && !strings.Contains(err.Error(), "dirty") {
		t.Errorf("expected error about uncommitted changes, got: %v", err)
	}
}

func TestRunRemove_LockedWorktree(t *testing.T) {
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

	err := runRemove("feature", false, false)
	if err == nil {
		t.Error("expected error for locked worktree")
	}
	if !strings.Contains(err.Error(), "locked") {
		t.Errorf("expected error about locked worktree, got: %v", err)
	}
}

func TestRunRemove_Success(t *testing.T) {
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

	// Configure git for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com") //nolint:gosec
	cmd.Dir = mainPath
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test") //nolint:gosec
	cmd.Dir = mainPath
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "commit.gpgsign", "false") //nolint:gosec
	cmd.Dir = mainPath
	_ = cmd.Run()

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

	err := runRemove("feature", false, false)
	if err != nil {
		t.Fatalf("runRemove failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after removal")
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

func TestRunRemove_ForceDirty(t *testing.T) {
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

	// Force remove dirty worktree
	err := runRemove("feature", true, false)
	if err != nil {
		t.Fatalf("runRemove with force failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after forced removal")
	}
}

func TestRunRemove_ForceLocked(t *testing.T) {
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

	// Force remove locked worktree
	err := runRemove("feature", true, false)
	if err != nil {
		t.Fatalf("runRemove with force failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after forced removal")
	}
}

func TestRunRemove_WithBranchFlag(t *testing.T) {
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

	// Configure git for commits
	cmd = exec.Command("git", "config", "user.email", "test@test.com") //nolint:gosec
	cmd.Dir = mainPath
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "user.name", "Test") //nolint:gosec
	cmd.Dir = mainPath
	_ = cmd.Run()
	cmd = exec.Command("git", "config", "commit.gpgsign", "false") //nolint:gosec
	cmd.Dir = mainPath
	_ = cmd.Run()

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

	// Remove with --branch flag
	err := runRemove("feature", false, true)
	if err != nil {
		t.Fatalf("runRemove with --branch failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after removal")
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

func TestCompleteRemoveArgs(t *testing.T) {
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
	removeCmd := NewRemoveCmd()
	completions, directive := completeRemoveArgs(removeCmd, nil, "")

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
