package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

// Error message constants for test assertions.
const (
	errBareRepoPathEmpty  = "bare repository path cannot be empty"
	errWorktreePathEmpty  = "worktree path cannot be empty"
	errBranchNameEmpty    = "branch name cannot be empty"
	errBaseReferenceEmpty = "base reference cannot be empty"
	errRefEmpty           = "ref cannot be empty"
)

func TestCreateWorktree(t *testing.T) {
	t.Run("fails with non-existent branch in empty repo", func(t *testing.T) {
		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, "test.bare")
		worktreeDir := filepath.Join(tempDir, "main")

		if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create bare directory: %v", err)
		}

		if err := InitBare(bareDir); err != nil {
			t.Fatalf("failed to create bare repo: %v", err)
		}

		err := CreateWorktree(bareDir, worktreeDir, "main", false)
		if err == nil {
			t.Fatal("expected error as main branch doesn't exist in empty repo")
		}
	})

	t.Run("succeeds with existing branch", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		worktreeDir := filepath.Join(repo.Dir, "feature-worktree")

		// Create a branch first
		cmd := exec.Command("git", "branch", "feature") //nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		err := CreateWorktree(repo.Path, worktreeDir, "feature", true)
		if err != nil {
			t.Fatalf("CreateWorktree failed: %v", err)
		}

		// Verify worktree was created
		if _, err := os.Stat(worktreeDir); os.IsNotExist(err) {
			t.Error("worktree directory was not created")
		}

		// Verify it's recognized as a worktree
		if !IsWorktree(worktreeDir) {
			t.Error("created directory is not recognized as a worktree")
		}
	})
}

func TestIsWorktree(t *testing.T) {
	t.Run("returns false for regular git repo", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		if IsWorktree(tempDir) {
			t.Error("expected regular git repo not to be detected as worktree")
		}
	})

	t.Run("returns true for git worktree", func(t *testing.T) {
		tempDir := t.TempDir()
		gitFile := filepath.Join(tempDir, ".git")

		if err := os.WriteFile(gitFile, []byte("gitdir: /path/to/repo"), fs.FileGit); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		if !IsWorktree(tempDir) {
			t.Error("expected git worktree to be detected")
		}
	})

	t.Run("returns false for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		if IsWorktree(tempDir) {
			t.Error("expected non-git directory not to be detected as worktree")
		}
	})

	t.Run("returns false for nonexistent path", func(t *testing.T) {
		nonexistentPath := "/nonexistent/path"

		if IsWorktree(nonexistentPath) {
			t.Error("expected nonexistent path not to be detected as worktree")
		}
	})
}

func TestListWorktrees(t *testing.T) {
	t.Run("returns empty slice when no worktrees", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		worktrees, err := ListWorktrees(repo.Path)
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		if len(worktrees) != 0 {
			t.Errorf("expected empty slice, got %v", worktrees)
		}
	})

	t.Run("returns worktree paths when worktrees exist", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		worktreeDir := filepath.Join(repo.Dir, "branch-worktree")

		cmd := exec.Command("git", "worktree", "add", worktreeDir, "-b", "feature") // nolint:gosec // Test uses controlled temp directory
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		worktrees, err := ListWorktrees(repo.Path)
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		if len(worktrees) != 1 {
			t.Errorf("expected 1 worktree, got %d: %v", len(worktrees), worktrees)
		}

		if worktrees[0] != worktreeDir {
			t.Errorf("expected worktree path %s, got %s", worktreeDir, worktrees[0])
		}
	})

	t.Run("handles worktree paths with spaces", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Create a worktree with spaces in the path
		worktreeDir := filepath.Join(repo.Dir, "branch with spaces")

		cmd := exec.Command("git", "worktree", "add", worktreeDir, "-b", "feature-spaces") // nolint:gosec // Test uses controlled temp directory
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		worktrees, err := ListWorktrees(repo.Path)
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		if len(worktrees) != 1 {
			t.Fatalf("expected 1 worktree, got %d: %v", len(worktrees), worktrees)
		}

		if worktrees[0] != worktreeDir {
			t.Errorf("expected worktree path %q, got %q", worktreeDir, worktrees[0])
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := ListWorktrees(tempDir)
		if err == nil {
			t.Fatal("ListWorktrees should fail for non-git directory")
		}
	})
}

func TestGetWorktreeInfo(t *testing.T) {
	t.Parallel()

	t.Run("returns info for clean worktree", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		info, err := GetWorktreeInfo(repo.Path)
		if err != nil {
			t.Fatalf("GetWorktreeInfo failed: %v", err)
		}

		if info.Branch != testDefaultBranch {
			t.Errorf("expected branch %s, got %s", testDefaultBranch, info.Branch)
		}
		if info.Dirty {
			t.Error("expected clean worktree")
		}
		if info.Path != repo.Path {
			t.Errorf("expected path %s, got %s", repo.Path, info.Path)
		}
	})

	t.Run("detects dirty worktree", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		// Create uncommitted file
		testFile := filepath.Join(repo.Path, "dirty.txt")
		if err := os.WriteFile(testFile, []byte("dirty"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		info, err := GetWorktreeInfo(repo.Path)
		if err != nil {
			t.Fatalf("GetWorktreeInfo failed: %v", err)
		}

		if !info.Dirty {
			t.Error("expected dirty worktree")
		}
	})

	t.Run("fails with empty path", func(t *testing.T) {
		t.Parallel()
		_, err := GetWorktreeInfo("")
		if err == nil {
			t.Fatal("expected error for empty path")
		}
	})

	t.Run("fails for incomplete git directory", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		// Create a fake .git directory with only a HEAD file (not a complete git repo)
		// This tests that GetWorktreeInfo fails gracefully on malformed worktrees
		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		headFile := filepath.Join(gitDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("abc1234567890\n"), fs.FileGit); err != nil {
			t.Fatalf("failed to create HEAD file: %v", err)
		}

		_, err := GetWorktreeInfo(tempDir)
		if err == nil {
			t.Fatal("expected error for incomplete git directory")
		}
	})
}

func TestListWorktreesWithInfo(t *testing.T) {
	t.Run("returns worktree info for repo with worktrees", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Create a worktree
		worktreeDir := filepath.Join(repo.Dir, "feature-worktree")
		cmd := exec.Command("git", "worktree", "add", worktreeDir, "-b", "feature") // nolint:gosec // Test uses controlled temp directory
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		infos, err := ListWorktreesWithInfo(repo.Path, false)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo failed: %v", err)
		}

		if len(infos) != 1 {
			t.Fatalf("expected 1 worktree, got %d", len(infos))
		}

		if infos[0].Path != worktreeDir {
			t.Errorf("expected path %s, got %s", worktreeDir, infos[0].Path)
		}
		if infos[0].Branch != "feature" {
			t.Errorf("expected branch feature, got %s", infos[0].Branch)
		}
	})

	t.Run("fast mode skips status checks", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		worktreeDir := filepath.Join(repo.Dir, "feature-worktree")
		cmd := exec.Command("git", "worktree", "add", worktreeDir, "-b", "feature") // nolint:gosec // Test uses controlled temp directory
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		// Make worktree dirty
		_ = os.WriteFile(filepath.Join(worktreeDir, "dirty.txt"), []byte("dirty"), fs.FileStrict)

		infos, err := ListWorktreesWithInfo(repo.Path, true)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo failed: %v", err)
		}

		if len(infos) != 1 {
			t.Fatalf("expected 1 worktree, got %d", len(infos))
		}

		// In fast mode, dirty should not be checked (stays false)
		if infos[0].Dirty {
			t.Error("in fast mode, Dirty should be false (not checked)")
		}
	})

	t.Run("results are sorted alphabetically", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Create worktrees in unsorted order
		cmd := exec.Command("git", "worktree", "add", filepath.Join(repo.Dir, "zebra-wt"), "-b", "zebra") // nolint:gosec // Test
		cmd.Dir = repo.Path
		_ = cmd.Run()
		cmd = exec.Command("git", "worktree", "add", filepath.Join(repo.Dir, "alpha-wt"), "-b", "alpha") // nolint:gosec // Test
		cmd.Dir = repo.Path
		_ = cmd.Run()
		cmd = exec.Command("git", "worktree", "add", filepath.Join(repo.Dir, "mid-wt"), "-b", "mid") // nolint:gosec // Test
		cmd.Dir = repo.Path
		_ = cmd.Run()

		infos, _ := ListWorktreesWithInfo(repo.Path, true)

		if len(infos) != 3 {
			t.Fatalf("expected 3 worktrees, got %d", len(infos))
		}

		expected := []string{"alpha", "mid", "zebra"}
		for i, exp := range expected {
			if infos[i].Branch != exp {
				t.Errorf("at index %d: expected branch %s, got %s", i, exp, infos[i].Branch)
			}
		}
	})

	t.Run("populates Locked field for locked worktrees", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		worktreeDir := filepath.Join(repo.Dir, "feature-worktree")
		cmd := exec.Command("git", "worktree", "add", worktreeDir, "-b", "feature") // nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		infos, err := ListWorktreesWithInfo(repo.Path, false)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo failed: %v", err)
		}
		if len(infos) != 1 {
			t.Fatalf("expected 1 worktree, got %d", len(infos))
		}
		if infos[0].Locked {
			t.Error("expected Locked to be false for unlocked worktree")
		}

		cmd = exec.Command("git", "worktree", "lock", worktreeDir) // nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to lock worktree: %v", err)
		}

		infos, err = ListWorktreesWithInfo(repo.Path, false)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo failed: %v", err)
		}
		if len(infos) != 1 {
			t.Fatalf("expected 1 worktree, got %d", len(infos))
		}
		if !infos[0].Locked {
			t.Error("expected Locked to be true for locked worktree")
		}
	})

	t.Run("populates LockReason field for locked worktrees", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		worktreeDir := filepath.Join(repo.Dir, "feature-worktree")
		cmd := exec.Command("git", "worktree", "add", worktreeDir, "-b", "feature") // nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		// Unlocked worktree should have empty LockReason
		infos, err := ListWorktreesWithInfo(repo.Path, false)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo failed: %v", err)
		}
		if infos[0].LockReason != "" {
			t.Errorf("expected empty LockReason for unlocked worktree, got %q", infos[0].LockReason)
		}

		// Lock with reason
		expectedReason := "important work"
		cmd = exec.Command("git", "worktree", "lock", "--reason", expectedReason, worktreeDir) // nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to lock worktree: %v", err)
		}

		infos, err = ListWorktreesWithInfo(repo.Path, false)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo failed: %v", err)
		}
		if infos[0].LockReason != expectedReason {
			t.Errorf("expected LockReason %q, got %q", expectedReason, infos[0].LockReason)
		}
	})

	t.Run("includes detached HEAD worktrees", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Create a detached worktree
		worktreeDir := filepath.Join(repo.Dir, "detached-worktree")
		cmd := exec.Command("git", "worktree", "add", "--detach", worktreeDir, "HEAD") // nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create detached worktree: %v", err)
		}

		infos, err := ListWorktreesWithInfo(repo.Path, false)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo failed: %v", err)
		}

		if len(infos) != 1 {
			t.Fatalf("expected 1 worktree (detached should be included), got %d", len(infos))
		}

		if !infos[0].Detached {
			t.Error("expected Detached to be true for detached worktree")
		}

		if infos[0].Branch == "" {
			t.Error("expected Branch to contain commit hash, got empty string")
		}
	})

	t.Run("includes detached HEAD worktrees in fast mode", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Create a detached worktree
		worktreeDir := filepath.Join(repo.Dir, "detached-worktree")
		cmd := exec.Command("git", "worktree", "add", "--detach", worktreeDir, "HEAD") // nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create detached worktree: %v", err)
		}

		infos, err := ListWorktreesWithInfo(repo.Path, true)
		if err != nil {
			t.Fatalf("ListWorktreesWithInfo (fast) failed: %v", err)
		}

		if len(infos) != 1 {
			t.Fatalf("expected 1 worktree (detached should be included), got %d", len(infos))
		}

		if !infos[0].Detached {
			t.Error("expected Detached to be true for detached worktree")
		}

		if infos[0].Branch == "" {
			t.Error("expected Branch to contain commit hash, got empty string")
		}
	})
}

func TestCreateWorktreeWithNewBranch(t *testing.T) {
	t.Run("creates worktree with new branch", func(t *testing.T) {
		// Create a regular repo first to have something to clone
		sourceRepo := testgit.NewTestRepo(t)

		// Clone it as bare to simulate grove workspace structure
		bareDir := filepath.Join(t.TempDir(), ".bare")
		cmd := exec.Command("git", "clone", "--bare", sourceRepo.Path, bareDir) // nolint:gosec // Test
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create bare clone: %v", err)
		}

		worktreePath := filepath.Join(filepath.Dir(bareDir), "feature-test")
		err := CreateWorktreeWithNewBranch(bareDir, worktreePath, "feature-test", true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Verify worktree exists
		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Error("worktree directory was not created")
		}

		// Verify branch was created
		branch, err := GetCurrentBranch(worktreePath)
		if err != nil {
			t.Fatalf("failed to get branch: %v", err)
		}
		if branch != "feature-test" {
			t.Errorf("expected branch 'feature-test', got '%s'", branch)
		}
	})

	t.Run("fails with empty bare repo path", func(t *testing.T) {
		err := CreateWorktreeWithNewBranch("", "/tmp/wt", "branch", true)
		if err == nil {
			t.Fatal("expected error for empty bare repo path")
		}
		if err.Error() != errBareRepoPathEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty worktree path", func(t *testing.T) {
		err := CreateWorktreeWithNewBranch("/some/repo", "", "branch", true)
		if err == nil {
			t.Fatal("expected error for empty worktree path")
		}
		if err.Error() != errWorktreePathEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty branch name", func(t *testing.T) {
		err := CreateWorktreeWithNewBranch("/some/repo", "/tmp/wt", "", true)
		if err == nil {
			t.Fatal("expected error for empty branch name")
		}
		if err.Error() != errBranchNameEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCreateWorktreeWithNewBranchFrom(t *testing.T) {
	t.Run("fails with empty base", func(t *testing.T) {
		err := CreateWorktreeWithNewBranchFrom("/repo", "/wt", "branch", "", true)
		if err == nil {
			t.Fatal("expected error for empty base")
		}
		if err.Error() != errBaseReferenceEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty bare repo path", func(t *testing.T) {
		err := CreateWorktreeWithNewBranchFrom("", "/wt", "branch", "main", true)
		if err == nil {
			t.Fatal("expected error for empty bare repo path")
		}
		if err.Error() != errBareRepoPathEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty worktree path", func(t *testing.T) {
		err := CreateWorktreeWithNewBranchFrom("/repo", "", "branch", "main", true)
		if err == nil {
			t.Fatal("expected error for empty worktree path")
		}
		if err.Error() != errWorktreePathEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty branch name", func(t *testing.T) {
		err := CreateWorktreeWithNewBranchFrom("/repo", "/wt", "", "main", true)
		if err == nil {
			t.Fatal("expected error for empty branch name")
		}
		if err.Error() != errBranchNameEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestCreateWorktreeDetached(t *testing.T) {
	t.Run("fails with empty bare repo path", func(t *testing.T) {
		err := CreateWorktreeDetached("", "/wt", "v1.0.0", true)
		if err == nil {
			t.Fatal("expected error for empty bare repo path")
		}
		if err.Error() != errBareRepoPathEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty worktree path", func(t *testing.T) {
		err := CreateWorktreeDetached("/repo", "", "v1.0.0", true)
		if err == nil {
			t.Fatal("expected error for empty worktree path")
		}
		if err.Error() != errWorktreePathEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty ref", func(t *testing.T) {
		err := CreateWorktreeDetached("/repo", "/wt", "", true)
		if err == nil {
			t.Fatal("expected error for empty ref")
		}
		if err.Error() != errRefEmpty {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestIsWorktreeLocked(t *testing.T) {
	t.Run("returns false for unlocked worktree", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)
		worktreePath := filepath.Join(repo.Dir, "feature")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") //nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		if IsWorktreeLocked(worktreePath) {
			t.Error("expected unlocked worktree to return false")
		}
	})

	t.Run("returns true for locked worktree", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)
		worktreePath := filepath.Join(repo.Dir, "feature")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") //nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		if err := LockWorktree(repo.Path, worktreePath, "test reason"); err != nil {
			t.Fatalf("failed to lock worktree: %v", err)
		}

		if !IsWorktreeLocked(worktreePath) {
			t.Error("expected locked worktree to return true")
		}
	})

	t.Run("returns false for non-existent worktree", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		nonexistentPath := filepath.Join(tempDir, "nonexistent")

		if IsWorktreeLocked(nonexistentPath) {
			t.Error("expected non-existent worktree to return false")
		}
	})

	t.Run("works when directory name differs from git worktree name", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		// Create worktree with original name
		originalPath := filepath.Join(repo.Dir, "original")
		cmd := exec.Command("git", "worktree", "add", originalPath, "-b", "feature") //nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		// Rename the worktree directory (simulating user moving it)
		renamedPath := filepath.Join(repo.Dir, "renamed")
		if err := os.Rename(originalPath, renamedPath); err != nil {
			t.Fatalf("failed to rename worktree: %v", err)
		}

		// Repair the worktree so git knows about the new location
		repairCmd := exec.Command("git", "worktree", "repair", renamedPath) //nolint:gosec
		repairCmd.Dir = repo.Path
		if err := repairCmd.Run(); err != nil {
			t.Fatalf("failed to repair worktree: %v", err)
		}

		// Lock using git directly (which uses the path)
		lockCmd := exec.Command("git", "worktree", "lock", "--reason", "test lock", renamedPath) //nolint:gosec
		lockCmd.Dir = repo.Path
		if err := lockCmd.Run(); err != nil {
			t.Fatalf("failed to lock worktree: %v", err)
		}

		// IsWorktreeLocked should return true even though directory is "renamed" but git knows it as "original"
		if !IsWorktreeLocked(renamedPath) {
			t.Error("expected renamed worktree to show as locked")
		}

		// GetWorktreeLockReason should also work
		reason := GetWorktreeLockReason(renamedPath)
		if reason != "test lock" {
			t.Errorf("expected lock reason 'test lock', got %q", reason)
		}
	})
}

func TestRemoveWorktree(t *testing.T) {
	t.Run("removes clean worktree", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		worktreePath := filepath.Join(tempDir, "feature")
		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") // nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Fatal("worktree should exist before removal")
		}

		if err := RemoveWorktree(bareDir, worktreePath, false); err != nil {
			t.Fatalf("RemoveWorktree failed: %v", err)
		}

		if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
			t.Error("worktree should not exist after removal")
		}
	})

	t.Run("fails for dirty worktree without force", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		worktreePath := filepath.Join(tempDir, "feature")
		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") // nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		dirtyFile := filepath.Join(worktreePath, "untracked.txt")
		if err := os.WriteFile(dirtyFile, []byte("dirty"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		err := RemoveWorktree(bareDir, worktreePath, false)
		if err == nil {
			t.Fatal("expected error when removing dirty worktree without force")
		}

		if _, err := os.Stat(worktreePath); os.IsNotExist(err) {
			t.Error("worktree should still exist after failed removal")
		}
	})

	t.Run("removes dirty worktree with force", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		worktreePath := filepath.Join(tempDir, "feature")
		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") // nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		dirtyFile := filepath.Join(worktreePath, "untracked.txt")
		if err := os.WriteFile(dirtyFile, []byte("dirty"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		if err := RemoveWorktree(bareDir, worktreePath, true); err != nil {
			t.Fatalf("RemoveWorktree with force failed: %v", err)
		}

		if _, err := os.Stat(worktreePath); !os.IsNotExist(err) {
			t.Error("worktree should not exist after forced removal")
		}
	})

	t.Run("returns error for non-existent worktree", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		err := RemoveWorktree(bareDir, "/nonexistent/path", false)
		if err == nil {
			t.Fatal("expected error for non-existent worktree")
		}
	})
}

func TestFindWorktreeRoot(t *testing.T) {
	t.Run("returns root when called from worktree root", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		// Create a .git file (simulating a worktree)
		gitFile := filepath.Join(tempDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		root, err := FindWorktreeRoot(tempDir)
		if err != nil {
			t.Fatalf("FindWorktreeRoot failed: %v", err)
		}
		if root != tempDir {
			t.Errorf("expected %s, got %s", tempDir, root)
		}
	})

	t.Run("returns root when called from subdirectory", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		// Create a .git file at root
		gitFile := filepath.Join(tempDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		// Create subdirectory
		subDir := filepath.Join(tempDir, "src")
		if err := os.Mkdir(subDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		root, err := FindWorktreeRoot(subDir)
		if err != nil {
			t.Fatalf("FindWorktreeRoot failed: %v", err)
		}
		if root != tempDir {
			t.Errorf("expected %s, got %s", tempDir, root)
		}
	})

	t.Run("returns root when called from deeply nested subdirectory", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		// Create a .git file at root
		gitFile := filepath.Join(tempDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		// Create deeply nested subdirectory
		deepDir := filepath.Join(tempDir, "src", "components", "auth")
		if err := os.MkdirAll(deepDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create nested directories: %v", err)
		}

		root, err := FindWorktreeRoot(deepDir)
		if err != nil {
			t.Fatalf("FindWorktreeRoot failed: %v", err)
		}
		if root != tempDir {
			t.Errorf("expected %s, got %s", tempDir, root)
		}
	})

	t.Run("returns error when not in a worktree", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		_, err := FindWorktreeRoot(tempDir)
		if err == nil {
			t.Fatal("expected error when not in a worktree")
		}
	})

	t.Run("returns root from deeply nested subdirectory (50 levels)", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		// Create a .git file at root
		gitFile := filepath.Join(tempDir, ".git")
		if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		// Create 50-level deep directory structure
		deepDir := tempDir
		for i := 0; i < 50; i++ {
			deepDir = filepath.Join(deepDir, "level")
		}
		if err := os.MkdirAll(deepDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create deep directory: %v", err)
		}

		root, err := FindWorktreeRoot(deepDir)
		if err != nil {
			t.Fatalf("FindWorktreeRoot failed for deep path: %v", err)
		}
		if root != tempDir {
			t.Errorf("expected %s, got %s", tempDir, root)
		}
	})
}

func TestLockWorktree(t *testing.T) {
	t.Run("locks worktree without reason", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		bareDir := repo.Path
		worktreePath := filepath.Join(repo.Dir, "feature")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		if err := LockWorktree(bareDir, worktreePath, ""); err != nil {
			t.Fatalf("LockWorktree failed: %v", err)
		}

		if !IsWorktreeLocked(worktreePath) {
			t.Error("worktree should be locked after LockWorktree")
		}
	})

	t.Run("locks worktree with reason", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		bareDir := repo.Path
		worktreePath := filepath.Join(repo.Dir, "feature")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		reason := "WIP - do not remove"
		if err := LockWorktree(bareDir, worktreePath, reason); err != nil {
			t.Fatalf("LockWorktree with reason failed: %v", err)
		}

		if !IsWorktreeLocked(worktreePath) {
			t.Error("worktree should be locked after LockWorktree")
		}

		gotReason := GetWorktreeLockReason(worktreePath)
		if gotReason != reason {
			t.Errorf("expected lock reason %q, got %q", reason, gotReason)
		}
	})

	t.Run("fails for nonexistent worktree", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		err := LockWorktree(repo.Path, "/nonexistent/worktree", "test reason")
		if err == nil {
			t.Fatal("LockWorktree should fail for nonexistent worktree")
		}
	})

	t.Run("fails for already locked worktree", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		worktreePath := filepath.Join(repo.Dir, "feature")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") //nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		// Lock the worktree first
		if err := LockWorktree(repo.Path, worktreePath, "first lock"); err != nil {
			t.Fatalf("first LockWorktree failed: %v", err)
		}

		// Try to lock again - should fail
		err := LockWorktree(repo.Path, worktreePath, "second lock")
		if err == nil {
			t.Fatal("LockWorktree should fail for already locked worktree")
		}
	})
}

func TestGetWorktreeLockReason(t *testing.T) {
	t.Run("returns empty string for unlocked worktree", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		worktreePath := filepath.Join(repo.Dir, "feature")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") //nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		reason := GetWorktreeLockReason(worktreePath)
		if reason != "" {
			t.Errorf("expected empty reason for unlocked worktree, got %q", reason)
		}
	})

	t.Run("returns reason for locked worktree", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		worktreePath := filepath.Join(repo.Dir, "feature")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") //nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		expectedReason := "important work in progress"
		lockCmd := exec.Command("git", "worktree", "lock", "--reason", expectedReason, worktreePath) //nolint:gosec
		lockCmd.Dir = repo.Path
		if err := lockCmd.Run(); err != nil {
			t.Fatalf("failed to lock worktree: %v", err)
		}

		reason := GetWorktreeLockReason(worktreePath)
		if reason != expectedReason {
			t.Errorf("expected reason %q, got %q", expectedReason, reason)
		}
	})

	t.Run("returns empty string for nonexistent worktree", func(t *testing.T) {
		tempDir := t.TempDir()
		nonexistentPath := filepath.Join(tempDir, "nonexistent")

		reason := GetWorktreeLockReason(nonexistentPath)
		if reason != "" {
			t.Errorf("expected empty reason for nonexistent worktree, got %q", reason)
		}
	})
}

func TestFindWorktree(t *testing.T) {
	infos := []*WorktreeInfo{
		{Path: "/workspace/main", Branch: "main"},
		{Path: "/workspace/feature-auth", Branch: "feature/auth"},
		{Path: "/workspace/bugfix", Branch: "bugfix/issue-123"},
	}

	t.Run("finds by worktree name", func(t *testing.T) {
		result := FindWorktree(infos, "feature-auth")
		if result == nil {
			t.Fatal("expected to find worktree")
		}
		if result.Branch != "feature/auth" {
			t.Errorf("expected branch feature/auth, got %s", result.Branch)
		}
	})

	t.Run("finds by branch name as fallback", func(t *testing.T) {
		result := FindWorktree(infos, "feature/auth")
		if result == nil {
			t.Fatal("expected to find worktree")
		}
		if result.Path != "/workspace/feature-auth" {
			t.Errorf("expected path /workspace/feature-auth, got %s", result.Path)
		}
	})

	t.Run("worktree name takes priority over branch name", func(t *testing.T) {
		// Create a scenario where worktree name matches one entry
		// but branch name would match another
		testInfos := []*WorktreeInfo{
			{Path: "/workspace/main", Branch: "main"},
			{Path: "/workspace/develop", Branch: "main"}, // Different worktree with same branch
		}
		result := FindWorktree(testInfos, "develop")
		if result == nil {
			t.Fatal("expected to find worktree")
		}
		if result.Path != "/workspace/develop" {
			t.Errorf("expected path /workspace/develop, got %s", result.Path)
		}
	})

	t.Run("returns nil when not found", func(t *testing.T) {
		result := FindWorktree(infos, "nonexistent")
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
}
