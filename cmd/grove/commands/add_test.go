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
	"github.com/sqve/grove/internal/testutil"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewAddCmd(t *testing.T) {
	cmd := NewAddCmd()

	// Verify command structure
	if cmd.Use != "add [branch|PR-URL|ref]" {
		t.Errorf("unexpected Use: %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}

	// Verify required flags exist with correct configuration
	flags := []struct {
		name      string
		shorthand string
	}{
		{"switch", "s"},
		{"base", ""},
		{"detach", "d"},
		{"name", ""},
		{"pr", ""},
	}

	for _, f := range flags {
		flag := cmd.Flags().Lookup(f.name)
		if flag == nil {
			t.Errorf("expected --%s flag to exist", f.name)
			continue
		}
		if f.shorthand != "" && flag.Shorthand != f.shorthand {
			t.Errorf("--%s: expected shorthand %q, got %q", f.name, f.shorthand, flag.Shorthand)
		}
	}

	// Verify ValidArgsFunction is set
	if cmd.ValidArgsFunction == nil {
		t.Error("expected ValidArgsFunction to be set")
	}
}

func TestNewAddCmd_HasSwitchFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("switch")
	if flag == nil {
		t.Fatal("expected --switch flag to exist")
	}
	if flag.Shorthand != "s" {
		t.Errorf("expected shorthand 's', got %q", flag.Shorthand)
	}
}

func TestNewAddCmd_HasBaseFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("base")
	if flag == nil {
		t.Fatal("expected --base flag to exist")
	}
	if flag.DefValue != "" {
		t.Errorf("expected default value '', got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %q", flag.Value.Type())
	}
}

func TestNewAddCmd_HasDetachFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("detach")
	if flag == nil {
		t.Fatal("expected --detach flag to exist")
	}
	if flag.Shorthand != "d" {
		t.Errorf("expected shorthand 'd', got %q", flag.Shorthand)
	}
	if flag.DefValue != "false" {
		t.Errorf("expected default value 'false', got %q", flag.DefValue)
	}
	if flag.Value.Type() != "bool" {
		t.Errorf("expected bool type, got %q", flag.Value.Type())
	}
}

func TestNewAddCmd_HasNameFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("name")
	if flag == nil {
		t.Fatal("expected --name flag to exist")
	}
	if flag.DefValue != "" {
		t.Errorf("expected default value '', got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %q", flag.Value.Type())
	}
}

func TestNewAddCmd_HasFromFlag(t *testing.T) {
	cmd := NewAddCmd()
	flag := cmd.Flags().Lookup("from")
	if flag == nil {
		t.Fatal("expected --from flag to exist")
	}
	if flag.DefValue != "" {
		t.Errorf("expected default value '', got %q", flag.DefValue)
	}
	if flag.Value.Type() != "string" {
		t.Errorf("expected string type, got %q", flag.Value.Type())
	}
}

func TestRunAdd_NotInWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := testutil.TempDir(t)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runAdd([]string{"feature-test"}, false, "", "", false, 0, false, "")
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestRunAdd_PRValidation(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := testutil.TempDir(t)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	t.Run("base flag cannot be used with --pr", func(t *testing.T) {
		err := runAdd(nil, false, "main", "", false, 123, false, "")
		if err == nil || !strings.Contains(err.Error(), "--base cannot be used with PR") {
			t.Errorf("expected base/PR error, got %v", err)
		}
	})

	t.Run("detach flag cannot be used with --pr", func(t *testing.T) {
		err := runAdd(nil, false, "", "", true, 123, false, "")
		if err == nil || !strings.Contains(err.Error(), "--detach cannot be used with PR") {
			t.Errorf("expected detach/PR error, got %v", err)
		}
	})

	t.Run("negative --pr gives clear error", func(t *testing.T) {
		err := runAdd(nil, false, "", "", false, -5, false, "")
		if err == nil || !strings.Contains(err.Error(), "--pr must be a positive number") {
			t.Errorf("expected positive number error, got %v", err)
		}
	})

	t.Run("--pr cannot be combined with positional argument", func(t *testing.T) {
		err := runAdd([]string{"feature"}, false, "", "", false, 123, false, "")
		if err == nil || !strings.Contains(err.Error(), "--pr flag cannot be combined with positional argument") {
			t.Errorf("expected --pr/positional conflict error, got %v", err)
		}
	})

	t.Run("old #N syntax gives helpful error", func(t *testing.T) {
		err := runAdd([]string{"#123"}, false, "", "", false, 0, false, "")
		if err == nil || !strings.Contains(err.Error(), "syntax no longer supported") {
			t.Errorf("expected helpful migration error, got %v", err)
		}
	})

	t.Run("base flag cannot be used with PR URL", func(t *testing.T) {
		err := runAdd([]string{"https://github.com/owner/repo/pull/456"}, false, "main", "", false, 0, false, "")
		if err == nil || !strings.Contains(err.Error(), "--base cannot be used with PR") {
			t.Errorf("expected base/PR error, got %v", err)
		}
	})

	t.Run("detach flag cannot be used with PR URL", func(t *testing.T) {
		err := runAdd([]string{"https://github.com/owner/repo/pull/456"}, false, "", "", true, 0, false, "")
		if err == nil || !strings.Contains(err.Error(), "--detach cannot be used with PR") {
			t.Errorf("expected detach/PR error, got %v", err)
		}
	})

	t.Run("reset flag can only be used with PR references", func(t *testing.T) {
		err := runAdd([]string{"feature-branch"}, false, "", "", false, 0, true, "")
		if err == nil || !strings.Contains(err.Error(), "--reset can only be used with PR references") {
			t.Errorf("expected --reset/PR error, got %v", err)
		}
	})
}

func TestRunAdd_DetachBaseValidation(t *testing.T) {
	t.Run("detach and base cannot be used together", func(t *testing.T) {
		err := runAdd([]string{"v1.0.0"}, false, "main", "", true, 0, false, "")
		if err == nil || err.Error() != "--detach and --base cannot be used together" {
			t.Errorf("expected detach/base error, got %v", err)
		}
	})
}

func TestRunAdd_InputValidation(t *testing.T) {
	// Save and restore cwd - we need to be outside a workspace
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := testutil.TempDir(t)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	t.Run("whitespace-only branch name", func(t *testing.T) {
		// Whitespace is trimmed, resulting in empty string
		// This should fail with "requires branch" error
		err := runAdd([]string{"   "}, false, "", "", false, 0, false, "")
		if err == nil || !strings.Contains(err.Error(), "requires branch") {
			t.Errorf("expected 'requires branch' error for whitespace-only branch name, got %v", err)
		}
	})

	t.Run("no args and no --pr flag", func(t *testing.T) {
		err := runAdd(nil, false, "", "", false, 0, false, "")
		if err == nil || !strings.Contains(err.Error(), "requires branch") {
			t.Errorf("expected 'requires branch' error, got %v", err)
		}
	})

	t.Run("leading and trailing whitespace is trimmed", func(t *testing.T) {
		// The trimming happens, then workspace detection runs
		// We're not in a workspace, so we'll get that error
		// But this verifies the trim doesn't crash
		err := runAdd([]string{"  feature-test  "}, false, "", "", false, 0, false, "")
		if !errors.Is(err, workspace.ErrNotInWorkspace) {
			t.Errorf("expected ErrNotInWorkspace after trimming, got %v", err)
		}
	})

	t.Run("PR URL with /files suffix works", func(t *testing.T) {
		// PR URLs with /files suffix should be detected as PR references
		// Flag validation happens before workspace detection
		err := runAdd([]string{"https://github.com/owner/repo/pull/123/files"}, false, "main", "", false, 0, false, "")
		if err == nil || !strings.Contains(err.Error(), "--base cannot be used with PR") {
			t.Errorf("expected base/PR error for URL with /files suffix, got %v", err)
		}
	})

	t.Run("PR URL with query params works", func(t *testing.T) {
		err := runAdd([]string{"https://github.com/owner/repo/pull/123?diff=split"}, false, "", "", true, 0, false, "")
		if err == nil || !strings.Contains(err.Error(), "--detach cannot be used with PR") {
			t.Errorf("expected detach/PR error for URL with query params, got %v", err)
		}
	})
}

func TestFindSourceWorktree(t *testing.T) {
	t.Run("returns empty at workspace root", func(t *testing.T) {
		workspaceRoot := testutil.TempDir(t)

		result := findSourceWorktree(workspaceRoot, workspaceRoot)
		if result != "" {
			t.Errorf("expected empty string at workspace root, got %q", result)
		}
	})

	t.Run("returns worktree path when in worktree", func(t *testing.T) {
		workspaceRoot := testutil.TempDir(t)

		// Create a fake worktree directory with .git file
		worktreeDir := filepath.Join(workspaceRoot, "main")
		if err := os.MkdirAll(worktreeDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		gitFile := filepath.Join(worktreeDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../.bare"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(worktreeDir, workspaceRoot)
		if result != worktreeDir {
			t.Errorf("expected %q, got %q", worktreeDir, result)
		}
	})

	t.Run("returns worktree from subdirectory", func(t *testing.T) {
		workspaceRoot := testutil.TempDir(t)

		// Create worktree with subdirectory
		worktreeDir := filepath.Join(workspaceRoot, "main")
		subDir := filepath.Join(worktreeDir, "src", "pkg")
		if err := os.MkdirAll(subDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		gitFile := filepath.Join(worktreeDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: ../.bare"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(subDir, workspaceRoot)
		if result != worktreeDir {
			t.Errorf("expected %q, got %q", worktreeDir, result)
		}
	})

	t.Run("returns empty when not in worktree", func(t *testing.T) {
		workspaceRoot := testutil.TempDir(t)

		// Create a directory that's not a worktree
		otherDir := filepath.Join(workspaceRoot, "other")
		if err := os.MkdirAll(otherDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		result := findSourceWorktree(otherDir, workspaceRoot)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestFindFallbackSourceWorktree(t *testing.T) {
	// Helper to create a bare repo with initial content
	setupBareRepo := func(t *testing.T, defaultBranch string) (tempDir, bareDir string) {
		t.Helper()
		tempDir = testutil.TempDir(t)
		bareDir = filepath.Join(tempDir, ".bare")

		// Create a regular repo first, then clone it as bare
		srcDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create src dir: %v", err)
		}

		// Init with specified default branch
		cmd := exec.Command("git", "init", "-b", defaultBranch) //nolint:gosec
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		// Configure git
		for _, cfg := range [][]string{
			{"user.email", "test@test.com"},
			{"user.name", "Test"},
			{"commit.gpgsign", "false"},
		} {
			cmd = exec.Command("git", "config", cfg[0], cfg[1]) //nolint:gosec
			cmd.Dir = srcDir
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to set config: %v", err)
			}
		}

		// Create initial commit
		if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "init")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Clone as bare repo
		cmd = exec.Command("git", "clone", "--bare", srcDir, bareDir) //nolint:gosec
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to clone bare: %v", err)
		}

		return tempDir, bareDir
	}

	t.Run("returns worktree for default branch", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create worktree for develop branch
		developDir := filepath.Join(tempDir, "develop")
		cmd := exec.Command("git", "worktree", "add", developDir, "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", developDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != developDir {
			t.Errorf("expected %q, got %q", developDir, result)
		}
	})

	t.Run("falls back to main when no default worktree", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create main branch from develop
		cmd := exec.Command("git", "branch", "main", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create main branch: %v", err)
		}

		// Only create worktree for main (not develop which is the default)
		mainDir := filepath.Join(tempDir, "main")
		cmd = exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != mainDir {
			t.Errorf("expected %q, got %q", mainDir, result)
		}
	})

	t.Run("falls back to master when no main", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create master branch
		cmd := exec.Command("git", "branch", "master", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create master branch: %v", err)
		}

		// Only create worktree for master
		masterDir := filepath.Join(tempDir, "master")
		cmd = exec.Command("git", "worktree", "add", masterDir, "master") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add master worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", masterDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != masterDir {
			t.Errorf("expected %q, got %q", masterDir, result)
		}
	})

	t.Run("returns empty when no worktrees exist", func(t *testing.T) {
		_, bareDir := setupBareRepo(t, "main")

		result := findFallbackSourceWorktree(bareDir)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("returns empty when no matching branch worktree", func(t *testing.T) {
		tempDir, bareDir := setupBareRepo(t, "develop")

		// Create feature branch
		cmd := exec.Command("git", "branch", "feature-x", "develop") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		// Only create worktree for feature-x (not develop/main/master)
		featureDir := filepath.Join(tempDir, "feature-x")
		cmd = exec.Command("git", "worktree", "add", featureDir, "feature-x") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add feature worktree: %v", err)
		}

		t.Cleanup(func() {
			cmd := exec.Command("git", "worktree", "remove", "--force", featureDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		result := findFallbackSourceWorktree(bareDir)
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})
}

func TestRunAdd_FromValidation(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Helper to create a grove workspace with worktrees
	setupWorkspace := func(t *testing.T) (tempDir, bareDir string) {
		t.Helper()
		tempDir = testutil.TempDir(t)
		bareDir = filepath.Join(tempDir, ".bare")

		// Create a regular repo first, then clone it as bare
		srcDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create src dir: %v", err)
		}

		// Init with main branch
		cmd := exec.Command("git", "init", "-b", "main")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		// Configure git
		for _, cfg := range [][]string{
			{"user.email", "test@test.com"},
			{"user.name", "Test"},
			{"commit.gpgsign", "false"},
		} {
			cmd = exec.Command("git", "config", cfg[0], cfg[1]) //nolint:gosec
			cmd.Dir = srcDir
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to set config: %v", err)
			}
		}

		// Create initial commit
		if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "init")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Clone as bare repo
		cmd = exec.Command("git", "clone", "--bare", srcDir, bareDir) //nolint:gosec
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to clone bare: %v", err)
		}

		// Clean up src directory
		if err := os.RemoveAll(srcDir); err != nil {
			t.Fatalf("failed to remove src: %v", err)
		}

		// Create main worktree
		mainDir := filepath.Join(tempDir, "main")
		cmd = exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main worktree: %v", err)
		}

		// Register cleanup to remove worktrees before temp dir cleanup (Windows file locks)
		t.Cleanup(func() {
			_ = os.Chdir(origDir)                                                // Exit temp dir entirely (Windows requirement)
			cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		return tempDir, bareDir
	}

	t.Run("--from with nonexistent worktree returns error", func(t *testing.T) {
		tempDir, _ := setupWorkspace(t)
		mainDir := filepath.Join(tempDir, "main")
		if err := os.Chdir(mainDir); err != nil {
			t.Fatal(err)
		}

		err := runAdd([]string{"feature-test"}, false, "", "", false, 0, false, "nonexistent")
		if err == nil {
			t.Fatal("expected error for nonexistent --from worktree")
		}
		if !strings.Contains(err.Error(), "worktree") || !strings.Contains(err.Error(), "not found") {
			t.Errorf("expected 'worktree not found' error, got %v", err)
		}
	})

	t.Run("--from with valid worktree by name succeeds", func(t *testing.T) {
		tempDir, bareDir := setupWorkspace(t)
		mainDir := filepath.Join(tempDir, "main")
		if err := os.Chdir(mainDir); err != nil {
			t.Fatal(err)
		}

		// Create another worktree to use as --from source
		sourceDir := filepath.Join(tempDir, "source")
		cmd := exec.Command("git", "worktree", "add", "-b", "source", sourceDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create source worktree: %v", err)
		}

		// Register cleanup for worktrees created in this subtest (Windows file locks)
		featureDir := filepath.Join(tempDir, "feature-from-test")
		t.Cleanup(func() {
			_ = os.Chdir(origDir)                                                   // Exit temp dir entirely (Windows requirement)
			cmd := exec.Command("git", "worktree", "remove", "--force", featureDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
			cmd = exec.Command("git", "worktree", "remove", "--force", sourceDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		// Create a new worktree with --from pointing to source
		err := runAdd([]string{"feature-from-test"}, false, "", "", false, 0, false, "source")
		if err != nil {
			t.Errorf("expected success with valid --from, got %v", err)
		}
	})
}

func TestRunAddFromBranch_WorktreeExistsHint(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tempDir := testutil.TempDir(t)
	bareDir := filepath.Join(tempDir, ".bare")

	srcDir := filepath.Join(tempDir, "src")
	if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
		t.Fatalf("failed to create src dir: %v", err)
	}

	cmd := exec.Command("git", "init", "-b", "main")
	cmd.Dir = srcDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init: %v", err)
	}

	for _, cfg := range [][]string{
		{"user.email", "test@test.com"},
		{"user.name", "Test"},
		{"commit.gpgsign", "false"},
	} {
		cmd = exec.Command("git", "config", cfg[0], cfg[1]) //nolint:gosec
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set config: %v", err)
		}
	}

	if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("test"), fs.FileStrict); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmd = exec.Command("git", "add", ".")
	cmd.Dir = srcDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add: %v", err)
	}
	cmd = exec.Command("git", "commit", "-m", "init")
	cmd.Dir = srcDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to commit: %v", err)
	}

	cmd = exec.Command("git", "clone", "--bare", srcDir, bareDir) //nolint:gosec
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to clone bare: %v", err)
	}

	if err := os.RemoveAll(srcDir); err != nil {
		t.Fatalf("failed to remove src: %v", err)
	}

	mainDir := filepath.Join(tempDir, "main")
	cmd = exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
	cmd.Dir = bareDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to add main worktree: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(origDir)
		cmd := exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
		cmd.Dir = bareDir
		_ = cmd.Run()
	})

	if err := os.Chdir(mainDir); err != nil {
		t.Fatal(err)
	}

	err = runAdd([]string{"main"}, false, "", "", false, 0, false, "")
	if err == nil {
		t.Fatal("expected error for existing worktree")
	}

	if !strings.Contains(err.Error(), "grove list") {
		t.Errorf("expected error to contain 'grove list' hint, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Errorf("expected error to contain '--name' hint, got: %v", err)
	}
}

func TestCompleteFromWorktree(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	// Helper to create a grove workspace with worktrees
	setupWorkspace := func(t *testing.T) (tempDir, bareDir string) {
		t.Helper()
		tempDir = testutil.TempDir(t)
		bareDir = filepath.Join(tempDir, ".bare")

		srcDir := filepath.Join(tempDir, "src")
		if err := os.MkdirAll(srcDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create src dir: %v", err)
		}

		cmd := exec.Command("git", "init", "-b", "main")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init: %v", err)
		}

		for _, cfg := range [][]string{
			{"user.email", "test@test.com"},
			{"user.name", "Test"},
			{"commit.gpgsign", "false"},
		} {
			cmd = exec.Command("git", "config", cfg[0], cfg[1]) //nolint:gosec
			cmd.Dir = srcDir
			if err := cmd.Run(); err != nil {
				t.Fatalf("failed to set config: %v", err)
			}
		}

		if err := os.WriteFile(filepath.Join(srcDir, "test.txt"), []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "init")
		cmd.Dir = srcDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		cmd = exec.Command("git", "clone", "--bare", srcDir, bareDir) //nolint:gosec
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to clone bare: %v", err)
		}

		if err := os.RemoveAll(srcDir); err != nil {
			t.Fatalf("failed to remove src: %v", err)
		}

		return tempDir, bareDir
	}

	t.Run("returns available worktree names", func(t *testing.T) {
		tempDir, bareDir := setupWorkspace(t)

		// Create worktrees
		mainDir := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", mainDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add main worktree: %v", err)
		}

		featureDir := filepath.Join(tempDir, "feature")
		cmd = exec.Command("git", "worktree", "add", "-b", "feature", featureDir, "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add feature worktree: %v", err)
		}

		// Register cleanup to remove worktrees before temp dir cleanup (Windows file locks)
		t.Cleanup(func() {
			_ = os.Chdir(origDir)                                                   // Exit temp dir entirely (Windows requirement)
			cmd := exec.Command("git", "worktree", "remove", "--force", featureDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
			cmd = exec.Command("git", "worktree", "remove", "--force", mainDir) //nolint:gosec
			cmd.Dir = bareDir
			_ = cmd.Run()
		})

		if err := os.Chdir(mainDir); err != nil {
			t.Fatal(err)
		}

		completions, directive := completeFromWorktree(nil, nil, "")

		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("expected ShellCompDirectiveNoFileComp, got %d", directive)
		}

		// Should include both worktrees
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
			t.Error("completions should include 'main'")
		}
		if !hasFeature {
			t.Error("completions should include 'feature'")
		}
	})
}
