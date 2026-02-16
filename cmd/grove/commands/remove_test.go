package commands

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/testutil"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewRemoveCmd(t *testing.T) {
	cmd := NewRemoveCmd()

	if cmd.Use != "remove <worktree>..." {
		t.Errorf("expected Use 'remove <worktree>...', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description")
	}
	forceFlag := cmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatal("expected --force flag")
	}
	if forceFlag.Shorthand != "f" {
		t.Errorf("expected force shorthand 'f', got %q", forceFlag.Shorthand)
	}
	if cmd.Flags().Lookup("branch") == nil {
		t.Error("expected --branch flag")
	}
}

func TestRunRemove_NotInWorkspace(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tmpDir := testutil.TempDir(t)
	testutil.Chdir(t, tmpDir)

	err := runRemove([]string{"some-branch"}, false, false)
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunRemove_WorktreeNotFound(t *testing.T) {
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

	err := runRemove([]string{"nonexistent"}, false, false)
	if err == nil {
		t.Error("expected error for non-existent branch")
	}
	if !strings.Contains(err.Error(), "worktree not found") {
		t.Errorf("expected 'worktree not found' error, got: %v", err)
	}
}

func TestRunRemove_CurrentWorktree(t *testing.T) {
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

	// Change to workspace (the worktree we'll try to remove)
	testutil.Chdir(t, mainPath)

	err := runRemove([]string{"main"}, false, false)
	if err == nil {
		t.Error("expected error when removing current worktree")
	}
	// Error message is now "failed: main" (per-item reason logged separately)
	if !strings.Contains(err.Error(), "main") {
		t.Errorf("expected error to mention 'main', got: %v", err)
	}
}

func TestRunRemove_CurrentWorktreeHint(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	testutil.Chdir(t, mainPath)

	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(nil)
	logger.Init(true, false)

	err := runRemove([]string{"main"}, false, false)
	if err == nil {
		t.Error("expected error when removing current worktree")
	}

	output := buf.String()
	if !strings.Contains(output, "grove switch") {
		t.Errorf("expected output to contain 'grove switch' hint, got: %s", output)
	}
}

func TestRunRemove_DirtyWorktree(t *testing.T) {
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

	// Make feature worktree dirty
	dirtyFile := filepath.Join(featurePath, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("dirty"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}

	// Change to main worktree (not the one we're removing)
	testutil.Chdir(t, mainPath)

	err := runRemove([]string{"feature"}, false, false)
	if err == nil {
		t.Error("expected error for dirty worktree")
	}
	// Error message is now "failed: feature" (per-item reason logged separately)
	if !strings.Contains(err.Error(), "feature") {
		t.Errorf("expected error to mention 'feature', got: %v", err)
	}
}

func TestRunRemove_LockedWorktree(t *testing.T) {
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

	// Change to main worktree
	testutil.Chdir(t, mainPath)

	err := runRemove([]string{"feature"}, false, false)
	if err == nil {
		t.Error("expected error for locked worktree")
	}
	// Error message is now "failed: feature" (per-item reason logged separately)
	if !strings.Contains(err.Error(), "feature") {
		t.Errorf("expected error to mention 'feature', got: %v", err)
	}
}

func TestRunRemove_Success(t *testing.T) {
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
	testutil.Chdir(t, mainPath)

	// Verify worktree exists
	if _, err := os.Stat(featurePath); os.IsNotExist(err) {
		t.Fatal("feature worktree should exist before deletion")
	}

	err := runRemove([]string{"feature"}, false, false)
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

	// Make feature worktree dirty
	dirtyFile := filepath.Join(featurePath, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("dirty"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}

	// Change to main worktree
	testutil.Chdir(t, mainPath)

	// Force remove dirty worktree
	err := runRemove([]string{"feature"}, true, false)
	if err != nil {
		t.Fatalf("runRemove with force failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after forced removal")
	}
}

func TestRunRemove_ForceLocked(t *testing.T) {
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

	// Change to main worktree
	testutil.Chdir(t, mainPath)

	// Force remove locked worktree
	err := runRemove([]string{"feature"}, true, false)
	if err != nil {
		t.Fatalf("runRemove with force failed: %v", err)
	}

	// Verify worktree is removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after forced removal")
	}
}

func TestRunRemove_WithBranchFlag(t *testing.T) {
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
	testutil.Chdir(t, mainPath)

	// Remove with --branch flag
	err := runRemove([]string{"feature"}, false, true)
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

func TestRunRemove_MultipleWorktrees(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	testutil.Chdir(t, mainPath)

	// Remove multiple worktrees at once
	err := runRemove([]string{"feature", "bugfix"}, false, false)
	if err != nil {
		t.Fatalf("runRemove failed: %v", err)
	}

	// Verify both are removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after removal")
	}
	if _, err := os.Stat(bugfixPath); !os.IsNotExist(err) {
		t.Error("bugfix worktree should not exist after removal")
	}
}

func TestRunRemove_MultipleWithForce(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	// Create feature worktree and make it dirty
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(featurePath, "dirty.txt"), []byte("dirty"), fs.FileStrict); err != nil {
		t.Fatal(err)
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
		t.Fatalf("failed to lock worktree: %v", err)
	}

	testutil.Chdir(t, mainPath)

	// Force remove both dirty and locked worktrees
	err := runRemove([]string{"feature", "bugfix"}, true, false)
	if err != nil {
		t.Fatalf("runRemove with force failed: %v", err)
	}

	// Verify both are removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should not exist after forced removal")
	}
	if _, err := os.Stat(bugfixPath); !os.IsNotExist(err) {
		t.Error("bugfix worktree should not exist after forced removal")
	}
}

func TestRunRemove_MultipleWithOneDirty(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	// Create clean feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Create dirty bugfix worktree
	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bugfixPath, "dirty.txt"), []byte("dirty"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}

	testutil.Chdir(t, mainPath)

	// Remove both without force - bugfix should fail, feature should succeed
	err := runRemove([]string{"feature", "bugfix"}, false, false)
	if err == nil {
		t.Fatal("expected error for dirty worktree")
	}
	if !strings.Contains(err.Error(), "bugfix") {
		t.Errorf("expected error to mention 'bugfix', got: %v", err)
	}

	// feature should be removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should be removed")
	}
	// bugfix should still exist (dirty)
	if _, err := os.Stat(bugfixPath); os.IsNotExist(err) {
		t.Error("bugfix worktree should still exist (dirty)")
	}
}

func TestRunRemove_MultipleWithOneLocked(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	// Create clean feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Create locked bugfix worktree
	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}
	cmd = exec.Command("git", "worktree", "lock", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to lock worktree: %v", err)
	}

	testutil.Chdir(t, mainPath)

	// Remove both without force - bugfix should fail, feature should succeed
	err := runRemove([]string{"feature", "bugfix"}, false, false)
	if err == nil {
		t.Fatal("expected error for locked worktree")
	}
	if !strings.Contains(err.Error(), "bugfix") {
		t.Errorf("expected error to mention 'bugfix', got: %v", err)
	}

	// feature should be removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should be removed")
	}
	// bugfix should still exist (locked)
	if _, err := os.Stat(bugfixPath); os.IsNotExist(err) {
		t.Error("bugfix worktree should still exist (locked)")
	}
}

func TestRunRemove_MultipleWithOneCurrent(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	// Change to feature worktree (current)
	testutil.Chdir(t, featurePath)

	// Try to remove both current (feature) and main
	err := runRemove([]string{"feature", "main"}, false, false)
	if err == nil {
		t.Fatal("expected error for current worktree")
	}
	if !strings.Contains(err.Error(), "feature") {
		t.Errorf("expected error to mention 'feature', got: %v", err)
	}

	// feature should still exist (current)
	if _, err := os.Stat(featurePath); os.IsNotExist(err) {
		t.Error("feature worktree should still exist (current)")
	}
	// main should be removed
	if _, err := os.Stat(mainPath); !os.IsNotExist(err) {
		t.Error("main worktree should be removed")
	}
}

func TestRunRemove_MultipleWithDeleteBranch(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	// Create initial commit
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

	testutil.Chdir(t, mainPath)

	// Remove with --branch flag
	err := runRemove([]string{"feature", "bugfix"}, false, true)
	if err != nil {
		t.Fatalf("runRemove failed: %v", err)
	}

	// Verify worktrees are removed
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should be removed")
	}
	if _, err := os.Stat(bugfixPath); !os.IsNotExist(err) {
		t.Error("bugfix worktree should be removed")
	}

	// Verify branches are also deleted
	exists, err := git.BranchExists(bareDir, "feature")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("feature branch should be deleted")
	}

	exists, err = git.BranchExists(bareDir, "bugfix")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Error("bugfix branch should be deleted")
	}
}

func TestRunRemove_DuplicateArgs(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	testutil.Chdir(t, mainPath)

	// Remove with duplicate args (same worktree specified twice)
	err := runRemove([]string{"feature", "feature"}, false, false)
	if err != nil {
		t.Fatalf("runRemove with duplicates failed: %v", err)
	}

	// Verify worktree is removed (only processed once)
	if _, err := os.Stat(featurePath); !os.IsNotExist(err) {
		t.Error("feature worktree should be removed")
	}
}

func TestRunRemove_ErrorShowsWorktreeLabel(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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
	featurePath := filepath.Join(tempDir, "feat-auth")
	cmd = exec.Command("git", "worktree", "add", "-b", "feature/auth", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Make feature worktree dirty
	dirtyFile := filepath.Join(featurePath, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("dirty"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}

	testutil.Chdir(t, mainPath)

	var buf bytes.Buffer
	logger.SetOutput(&buf)
	defer logger.SetOutput(nil)
	logger.Init(true, false)

	_ = runRemove([]string{"feat-auth"}, false, false)

	output := buf.String()
	// Error should show directory name as primary identifier with branch in brackets
	if !strings.Contains(output, "feat-auth [feature/auth]") {
		t.Errorf("expected error to show worktree label 'feat-auth [feature/auth]', got: %s", output)
	}
}

func TestCompleteRemoveArgs_MultipleArgs(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	testutil.Chdir(t, mainPath)

	removeCmd := NewRemoveCmd()

	// First arg already typed: "feature"
	completions, _ := completeRemoveArgs(removeCmd, []string{"feature"}, "")

	// Should not include feature (already used) or main (current)
	for _, c := range completions {
		if c == "feature" {
			t.Error("completions should not include already-used 'feature'")
		}
		if c == "main" {
			t.Error("completions should not include current worktree 'main'")
		}
	}

	// Should include bugfix
	hasBugfix := false
	for _, c := range completions {
		if c == "bugfix" {
			hasBugfix = true
		}
	}
	if !hasBugfix {
		t.Error("completions should include 'bugfix'")
	}
}

func TestCompleteRemoveArgs(t *testing.T) {
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

	// Create bugfix worktree
	bugfixPath := filepath.Join(tempDir, "bugfix")
	cmd = exec.Command("git", "worktree", "add", "-b", "bugfix", bugfixPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create bugfix worktree: %v", err)
	}

	// Change to main worktree
	testutil.Chdir(t, mainPath)

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

func TestRunRemove_CurrentWorktreeSubdirectory(t *testing.T) {
	defer testutil.SaveCwd(t)()

	tempDir := testutil.TempDir(t)
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

	// Create a subdirectory within the feature worktree and cd into it
	subDir := filepath.Join(featurePath, "src", "pkg")
	if err := os.MkdirAll(subDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	testutil.Chdir(t, subDir)

	err := runRemove([]string{"feature"}, false, false)
	if err == nil {
		t.Error("expected error when removing worktree from subdirectory within it")
	}
	if !strings.Contains(err.Error(), "feature") {
		t.Errorf("expected error to mention 'feature', got: %v", err)
	}

	// Worktree should still exist
	if _, statErr := os.Stat(featurePath); os.IsNotExist(statErr) {
		t.Error("feature worktree should still exist (protected from subdirectory)")
	}
}
