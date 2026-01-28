package commands

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutil"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewExecCmd(t *testing.T) {
	cmd := NewExecCmd()

	// Check command basics
	if cmd.Use != "exec [--all | <worktree>...] -- <command>" {
		t.Errorf("expected Use 'exec [--all | <worktree>...] -- <command>', got '%s'", cmd.Use)
	}

	// Check flags exist
	allFlag := cmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Fatal("expected --all flag")
	}
	if allFlag.Shorthand != "a" {
		t.Errorf("expected all shorthand 'a', got %q", allFlag.Shorthand)
	}
	if cmd.Flags().Lookup("fail-fast") == nil {
		t.Error("expected --fail-fast flag")
	}
}

func TestRunExec_NotInWorkspace(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := testutil.TempDir(t)
	_ = os.Chdir(tmpDir)

	err := runExec(true, false, nil, []string{"echo", "hello"})
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got: %v", err)
	}
}

func TestRunExec_NoTargets(t *testing.T) {
	// No --all and no worktree args
	err := runExec(false, false, nil, []string{"echo", "hello"})
	if err == nil {
		t.Error("expected error for no targets")
	}
	if err.Error() != "must specify --all or at least one worktree" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunExec_NoCommand(t *testing.T) {
	// No command after --
	err := runExec(true, false, nil, nil)
	if err == nil {
		t.Error("expected error for no command")
	}
	if err.Error() != "no command specified after --" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunExec_AllWithWorktrees(t *testing.T) {
	// Both --all and worktree args specified
	err := runExec(true, false, []string{"main"}, []string{"echo", "hello"})
	if err == nil {
		t.Error("expected error when both --all and worktrees specified")
	}
	if err.Error() != "cannot use --all with specific worktrees" {
		t.Errorf("unexpected error: %v", err)
	}
}

// setupExecTestWorkspace creates a grove workspace with multiple worktrees for testing
func setupExecTestWorkspace(t *testing.T) (tempDir, bareDir, mainPath string) {
	t.Helper()

	tempDir = testutil.TempDir(t)
	bareDir = filepath.Join(tempDir, ".bare")
	if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
		t.Fatal(err)
	}
	if err := git.InitBare(bareDir); err != nil {
		t.Fatal(err)
	}

	// Create main worktree
	mainPath = filepath.Join(tempDir, "main")
	cmd := exec.Command("git", "worktree", "add", mainPath, "-b", "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create worktree: %v", err)
	}

	// Configure git
	for _, args := range [][]string{
		{"config", "user.email", "test@test.com"},
		{"config", "user.name", "Test"},
		{"config", "commit.gpgsign", "false"},
	} {
		cmd = exec.Command("git", args...) //nolint:gosec
		cmd.Dir = mainPath
		_ = cmd.Run()
	}

	// Create initial commit
	testFile := filepath.Join(mainPath, "init.txt")
	if err := os.WriteFile(testFile, []byte("init"), fs.FileStrict); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "add", "init.txt") //nolint:gosec
	cmd.Dir = mainPath
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	cmd = exec.Command("git", "commit", "-m", "initial commit") //nolint:gosec
	cmd.Dir = mainPath
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}

	return tempDir, bareDir, mainPath
}

func TestRunExec_AllWorktrees(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir, bareDir, mainPath := setupExecTestWorkspace(t)

	// Create feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd := exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Change to main worktree
	_ = os.Chdir(mainPath)

	// Run command in all worktrees (creates a marker file)
	err := runExec(true, false, nil, []string{"touch", "exec-marker.txt"})
	if err != nil {
		t.Fatalf("runExec failed: %v", err)
	}

	// Verify marker files created in both worktrees
	if _, err := os.Stat(filepath.Join(mainPath, "exec-marker.txt")); os.IsNotExist(err) {
		t.Error("marker file should exist in main worktree")
	}
	if _, err := os.Stat(filepath.Join(featurePath, "exec-marker.txt")); os.IsNotExist(err) {
		t.Error("marker file should exist in feature worktree")
	}
}

func TestRunExec_SpecificWorktrees(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir, bareDir, mainPath := setupExecTestWorkspace(t)

	// Create feature and bugfix worktrees
	featurePath := filepath.Join(tempDir, "feature")
	cmd := exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
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

	// Change to main worktree
	_ = os.Chdir(mainPath)

	// Run command only in main and feature (not bugfix)
	err := runExec(false, false, []string{"main", "feature"}, []string{"touch", "specific-marker.txt"})
	if err != nil {
		t.Fatalf("runExec failed: %v", err)
	}

	// Verify marker files created in specified worktrees only
	if _, err := os.Stat(filepath.Join(mainPath, "specific-marker.txt")); os.IsNotExist(err) {
		t.Error("marker file should exist in main worktree")
	}
	if _, err := os.Stat(filepath.Join(featurePath, "specific-marker.txt")); os.IsNotExist(err) {
		t.Error("marker file should exist in feature worktree")
	}
	if _, err := os.Stat(filepath.Join(bugfixPath, "specific-marker.txt")); !os.IsNotExist(err) {
		t.Error("marker file should NOT exist in bugfix worktree")
	}
}

func TestRunExec_InvalidWorktree(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	_, _, mainPath := setupExecTestWorkspace(t)

	// Change to main worktree
	_ = os.Chdir(mainPath)

	// Try to run in non-existent worktree
	err := runExec(false, false, []string{"nonexistent"}, []string{"echo", "hello"})
	if err == nil {
		t.Error("expected error for non-existent worktree")
	}
	if !strings.Contains(err.Error(), "worktree not found: nonexistent") {
		t.Errorf("expected 'worktree not found' error, got: %v", err)
	}
}

func TestRunExec_CommandFails_ContinuesByDefault(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir, bareDir, mainPath := setupExecTestWorkspace(t)

	// Create feature worktree
	featurePath := filepath.Join(tempDir, "feature")
	cmd := exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature worktree: %v", err)
	}

	// Change to main worktree
	_ = os.Chdir(mainPath)

	// Run a command that creates a marker file then fails (exit 1).
	// Both worktrees will fail, but execution should continue to all worktrees.
	err := runExec(true, false, nil, []string{"sh", "-c", "touch marker.txt && exit 1"})

	// Should return error (all executions failed)
	if err == nil {
		t.Fatal("expected error when command fails")
	}

	// Both worktrees should have marker, proving execution continued to all worktrees
	if _, statErr := os.Stat(filepath.Join(mainPath, "marker.txt")); os.IsNotExist(statErr) {
		t.Error("marker should exist in main worktree")
	}
	if _, statErr := os.Stat(filepath.Join(featurePath, "marker.txt")); os.IsNotExist(statErr) {
		t.Error("marker should exist in feature worktree (execution should continue despite first failure)")
	}
}

func TestRunExec_FailFast(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir, bareDir, mainPath := setupExecTestWorkspace(t)

	// Create worktree with name that sorts before "main" alphabetically.
	// This ensures deterministic execution order for the test.
	aaaPath := filepath.Join(tempDir, "aaa")
	cmd := exec.Command("git", "worktree", "add", "-b", "aaa", aaaPath) //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create aaa worktree: %v", err)
	}

	// Change to main worktree
	_ = os.Chdir(mainPath)

	// Run a command that creates a marker then fails, with --fail-fast.
	// Worktrees are processed in alphabetical order by directory name.
	err := runExec(true, true, nil, []string{"sh", "-c", "touch failfast-marker.txt && exit 1"})

	// Should return error
	if err == nil {
		t.Fatal("expected error when command fails with --fail-fast")
	}

	// Only the first worktree (alphabetically: "aaa") should have marker.
	// "main" should NOT have marker because --fail-fast stops after first failure.
	if _, statErr := os.Stat(filepath.Join(aaaPath, "failfast-marker.txt")); os.IsNotExist(statErr) {
		t.Error("marker should exist in 'aaa' worktree (first alphabetically)")
	}
	if _, statErr := os.Stat(filepath.Join(mainPath, "failfast-marker.txt")); !os.IsNotExist(statErr) {
		t.Error("marker should NOT exist in 'main' worktree (fail-fast stops after first failure)")
	}
}

func TestCompleteExecArgs(t *testing.T) {
	origDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(origDir) }()

	tempDir, bareDir, mainPath := setupExecTestWorkspace(t)

	// Create feature and bugfix worktrees
	featurePath := filepath.Join(tempDir, "feature")
	cmd := exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
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

	// Change to main worktree
	_ = os.Chdir(mainPath)

	execCmd := NewExecCmd()

	t.Run("returns all worktrees when no args", func(t *testing.T) {
		completions, _ := completeExecArgs(execCmd, nil, "")
		if len(completions) != 3 {
			t.Errorf("expected 3 completions, got %d", len(completions))
		}
	})

	t.Run("excludes already specified worktrees", func(t *testing.T) {
		completions, _ := completeExecArgs(execCmd, []string{"main"}, "")
		// Should not include "main" since it's already specified
		for _, c := range completions {
			if c == "main" {
				t.Error("completions should not include already-specified worktree 'main'")
			}
		}
		if len(completions) != 2 {
			t.Errorf("expected 2 completions (feature, bugfix), got %d", len(completions))
		}
	})
}
