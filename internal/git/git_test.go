package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/sqve/grove/internal/fs"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

const testDefaultBranch = "main"

func TestInitBare(t *testing.T) {
	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, "test.bare")

	if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
		t.Fatalf("failed to create bare directory: %v", err)
	}
	if err := InitBare(bareDir); err != nil {
		t.Fatalf("InitBare should succeed: %v", err)
	}

	if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); os.IsNotExist(err) {
		t.Error("HEAD file should be created in bare repository")
	}
	if _, err := os.Stat(filepath.Join(bareDir, "config")); os.IsNotExist(err) {
		t.Error("config file should be created in bare repository")
	}
}

func TestInitBareGitNotAvailable(t *testing.T) {
	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, "test.bare")

	if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
		t.Fatalf("failed to create bare directory: %v", err)
	}

	t.Setenv("PATH", "")

	err := InitBare(bareDir)
	if err == nil {
		t.Fatal("InitBare should fail when git is not available")
	}

	if err.Error() != `exec: "git": executable file not found in $PATH` {
		t.Errorf("expected git not found error, got: %v", err)
	}
}

func TestConfigureBare(t *testing.T) {
	t.Run("configures repository as bare", func(t *testing.T) {
		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, "test.bare")

		if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create bare directory: %v", err)
		}

		if err := InitBare(bareDir); err != nil {
			t.Fatalf("failed to create test repo: %v", err)
		}

		if err := ConfigureBare(bareDir); err != nil {
			t.Fatalf("ConfigureBare should succeed: %v", err)
		}

		cmd := exec.Command("git", "config", "--bool", "core.bare")
		cmd.Dir = bareDir
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to check core.bare config: %v", err)
		}

		if strings.TrimSpace(out.String()) != "true" {
			t.Errorf("expected core.bare=true, got: %s", out.String())
		}
	})

	t.Run("fails with empty path", func(t *testing.T) {
		err := ConfigureBare("")
		if err == nil {
			t.Fatal("ConfigureBare should fail with empty path")
		}

		if err.Error() != "repository path cannot be empty" {
			t.Errorf("expected empty path error, got: %v", err)
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		err := ConfigureBare(tempDir)
		if err == nil {
			t.Fatal("ConfigureBare should fail for non-git directory")
		}
	})
}

func TestListBranches(t *testing.T) {
	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, "test.bare")

	if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
		t.Fatalf("failed to create bare directory: %v", err)
	}

	if err := InitBare(bareDir); err != nil {
		t.Fatalf("failed to create test repo: %v", err)
	}

	branches, err := ListBranches(bareDir)
	if err != nil {
		t.Fatalf("ListBranches failed: %v", err)
	}

	if len(branches) != 0 {
		t.Errorf("Expected no branches in empty repo, got: %v", branches)
	}
}

func TestCreateWorktree(t *testing.T) {
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
		t.Fatal("Expected error as main branch doesn't exist in empty repo")
	}
}

func TestCloneQuietMode(t *testing.T) {
	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, "test.bare")

	err := Clone("file:///nonexistent/repo.git", bareDir, true, false)
	if err == nil {
		t.Fatal("Expected error for non-existent repo")
	}

	if err.Error() == "" {
		t.Error("Error message should not be empty in quiet mode")
	}
}

func TestCloneVerboseMode(t *testing.T) {
	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, "test.bare")

	err := Clone("file:///nonexistent/repo.git", bareDir, false, false)
	if err == nil {
		t.Fatal("Expected error for non-existent repo")
	}

	if err.Error() == "" {
		t.Error("Error message should not be empty in verbose mode")
	}
}

func TestIsInsideGitRepo_NotGitRepo(t *testing.T) {
	tempDir := t.TempDir()

	if IsInsideGitRepo(tempDir) {
		t.Error("Expected IsInsideGitRepo to return false for non-git directory")
	}
}

func TestIsInsideGitRepo_NonexistentPath(t *testing.T) {
	nonexistentPath := "/nonexistent/path"

	if IsInsideGitRepo(nonexistentPath) {
		t.Error("Expected IsInsideGitRepo to return false for nonexistent path")
	}
}

func TestGetCurrentBranch(t *testing.T) {
	t.Run("returns branch name from HEAD file", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		// Write HEAD file pointing to main branch
		headFile := filepath.Join(gitDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), fs.FileGit); err != nil {
			t.Fatalf("failed to create HEAD file: %v", err)
		}

		branch, err := GetCurrentBranch(tempDir)
		if err != nil {
			t.Fatalf("GetCurrentBranch failed: %v", err)
		}

		if branch != testDefaultBranch {
			t.Errorf("expected branch '%s', got '%s'", testDefaultBranch, branch)
		}
	})

	t.Run("fails for detached HEAD", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		// Write HEAD file with a commit hash (detached)
		headFile := filepath.Join(gitDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("abc1234567890\n"), fs.FileGit); err != nil {
			t.Fatalf("failed to create HEAD file: %v", err)
		}

		_, err := GetCurrentBranch(tempDir)
		if err == nil {
			t.Fatal("GetCurrentBranch should fail for detached HEAD")
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := GetCurrentBranch(tempDir)
		if err == nil {
			t.Fatal("GetCurrentBranch should fail for non-git directory")
		}
	})
}

func TestIsDetachedHead(t *testing.T) {
	t.Run("detects detached HEAD with commit hash", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		// Test detached HEAD (commit hash)
		headFile := filepath.Join(gitDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("abc1234567890abcdef1234567890abcdef123456\n"), fs.FileGit); err != nil {
			t.Fatalf("failed to create HEAD file: %v", err)
		}

		isDetached, err := IsDetachedHead(tempDir)
		if err != nil {
			t.Fatalf("IsDetachedHead failed: %v", err)
		}
		if !isDetached {
			t.Error("expected detached HEAD to be detected")
		}
	})

	t.Run("does not detect normal branch as detached", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		headFile := filepath.Join(gitDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), fs.FileGit); err != nil {
			t.Fatalf("failed to update HEAD file: %v", err)
		}

		isDetached, err := IsDetachedHead(tempDir)
		if err != nil {
			t.Fatalf("IsDetachedHead failed: %v", err)
		}
		if isDetached {
			t.Error("expected branch HEAD not to be detected as detached")
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := IsDetachedHead(tempDir)
		if err == nil {
			t.Fatal("IsDetachedHead should fail for non-git directory")
		}
	})

	t.Run("works for git worktrees", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		worktreePath := filepath.Join(repo.Dir, "wt-detached")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") // nolint:gosec // test-controlled path
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		isDetached, err := IsDetachedHead(worktreePath)
		if err != nil {
			t.Fatalf("IsDetachedHead failed for worktree: %v", err)
		}
		if isDetached {
			t.Fatal("expected worktree to be attached initially")
		}

		cmd = exec.Command("git", "-C", worktreePath, "checkout", "--detach") // nolint:gosec // test-controlled path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to detach HEAD in worktree: %v", err)
		}

		isDetached, err = IsDetachedHead(worktreePath)
		if err != nil {
			t.Fatalf("IsDetachedHead failed for detached worktree: %v", err)
		}
		if !isDetached {
			t.Fatal("expected detached HEAD to be detected in worktree")
		}
	})
}

func TestHasOngoingOperation(t *testing.T) {
	t.Run("returns false for clean repo", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		hasOperation, err := HasOngoingOperation(tempDir)
		if err != nil {
			t.Fatalf("HasOngoingOperation failed: %v", err)
		}
		if hasOperation {
			t.Error("expected no ongoing operation in clean repo")
		}
	})

	t.Run("detects merge operation", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
		if err := os.WriteFile(mergeHead, []byte("commit-hash"), fs.FileGit); err != nil {
			t.Fatalf("failed to create MERGE_HEAD: %v", err)
		}

		hasOperation, err := HasOngoingOperation(tempDir)
		if err != nil {
			t.Fatalf("HasOngoingOperation failed: %v", err)
		}
		if !hasOperation {
			t.Error("expected merge operation to be detected")
		}
	})

	t.Run("detects rebase operation", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		rebaseDir := filepath.Join(gitDir, "rebase-merge")
		if err := os.Mkdir(rebaseDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create rebase-merge: %v", err)
		}

		hasOperation, err := HasOngoingOperation(tempDir)
		if err != nil {
			t.Fatalf("HasOngoingOperation failed: %v", err)
		}
		if !hasOperation {
			t.Error("expected rebase operation to be detected")
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := HasOngoingOperation(tempDir)
		if err == nil {
			t.Fatal("HasOngoingOperation should fail for non-git directory")
		}
	})

	t.Run("detects operations in worktrees", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		worktreePath := filepath.Join(repo.Dir, "wt-ongoing")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") // nolint:gosec // test-controlled path
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		worktreeGitDir, err := GetGitDir(worktreePath)
		if err != nil {
			t.Fatalf("failed to resolve worktree git dir: %v", err)
		}

		mergeHead := filepath.Join(worktreeGitDir, "MERGE_HEAD")
		if err := os.WriteFile(mergeHead, []byte("commit-hash"), fs.FileGit); err != nil {
			t.Fatalf("failed to create MERGE_HEAD in worktree: %v", err)
		}

		hasOperation, err := HasOngoingOperation(worktreePath)
		if err != nil {
			t.Fatalf("HasOngoingOperation failed for worktree: %v", err)
		}
		if !hasOperation {
			t.Error("expected merge operation to be detected in worktree")
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

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := ListWorktrees(tempDir)
		if err == nil {
			t.Fatal("ListWorktrees should fail for non-git directory")
		}
	})
}

func TestHasLockFiles(t *testing.T) {
	t.Run("returns false for clean repo", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		hasLocks, err := HasLockFiles(tempDir)
		if err != nil {
			t.Fatalf("HasLockFiles failed: %v", err)
		}
		if hasLocks {
			t.Error("expected clean repo to have no lock files")
		}
	})

	t.Run("detects index.lock file", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		gitDir := filepath.Join(tempDir, ".git")
		lockFile := filepath.Join(gitDir, "index.lock")
		if err := os.WriteFile(lockFile, []byte("lock content"), fs.FileGit); err != nil {
			t.Fatalf("failed to create index.lock: %v", err)
		}

		hasLocks, err := HasLockFiles(tempDir)
		if err != nil {
			t.Fatalf("HasLockFiles failed: %v", err)
		}
		if !hasLocks {
			t.Error("expected index.lock to be detected")
		}
	})

	t.Run("detects HEAD.lock file", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		gitDir := filepath.Join(tempDir, ".git")
		lockFile := filepath.Join(gitDir, "HEAD.lock")
		if err := os.WriteFile(lockFile, []byte("lock content"), fs.FileGit); err != nil {
			t.Fatalf("failed to create HEAD.lock: %v", err)
		}

		hasLocks, err := HasLockFiles(tempDir)
		if err != nil {
			t.Fatalf("HasLockFiles failed: %v", err)
		}
		if !hasLocks {
			t.Error("expected HEAD.lock to be detected")
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := HasLockFiles(tempDir)
		if err == nil {
			t.Fatal("HasLockFiles should fail for non-git directory")
		}

		expected := "not a git repository"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("handles git worktrees", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		worktreePath := filepath.Join(repo.Dir, "wt-locks")

		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "feature") // nolint:gosec // test-controlled path
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		hasLocks, err := HasLockFiles(worktreePath)
		if err != nil {
			t.Fatalf("HasLockFiles failed for worktree: %v", err)
		}
		if hasLocks {
			t.Fatal("expected worktree to be reported clean")
		}

		worktreeGitDir, err := GetGitDir(worktreePath)
		if err != nil {
			t.Fatalf("failed to resolve worktree git dir: %v", err)
		}

		lockFile := filepath.Join(worktreeGitDir, "index.lock")
		if err := os.WriteFile(lockFile, []byte("lock content"), fs.FileGit); err != nil {
			t.Fatalf("failed to create index.lock in worktree: %v", err)
		}

		hasLocks, err = HasLockFiles(worktreePath)
		if err != nil {
			t.Fatalf("HasLockFiles failed for worktree with lock: %v", err)
		}
		if !hasLocks {
			t.Error("expected lock file in worktree to be detected")
		}
	})
}

func TestHasUnresolvedConflicts(t *testing.T) {
	t.Run("returns false for clean repo", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		hasConflicts, err := HasUnresolvedConflicts(tempDir)
		if err != nil {
			t.Fatalf("HasUnresolvedConflicts failed: %v", err)
		}
		if hasConflicts {
			t.Error("expected clean repo to have no unresolved conflicts")
		}
	})

	t.Run("detects merge conflicts", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		testFile := filepath.Join(repo.Path, "conflict.txt")
		if err := os.WriteFile(testFile, []byte("initial"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit files: %v", err)
		}

		cmd = exec.Command("git", "checkout", "-b", "branch1")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch1: %v", err)
		}

		if err := os.WriteFile(testFile, []byte("branch1"), fs.FileStrict); err != nil {
			t.Fatalf("failed to modify test file on branch1: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files on branch1: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "branch1 changes")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit branch1 changes: %v", err)
		}

		cmd = exec.Command("git", "checkout", "main")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout main: %v", err)
		}

		if err := os.WriteFile(testFile, []byte("main"), fs.FileStrict); err != nil {
			t.Fatalf("failed to modify test file on main: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files on main: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "main changes")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit main changes: %v", err)
		}

		cmd = exec.Command("git", "merge", "branch1")
		cmd.Dir = repo.Path
		_ = cmd.Run() // Expected to fail with conflict

		hasConflicts, err := HasUnresolvedConflicts(repo.Path)
		if err != nil {
			t.Fatalf("HasUnresolvedConflicts failed: %v", err)
		}
		if !hasConflicts {
			t.Error("expected merge conflicts to be detected")
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := HasUnresolvedConflicts(tempDir)
		if err == nil {
			t.Fatal("HasUnresolvedConflicts should fail for non-git directory")
		}
	})
}

func TestHasSubmodules(t *testing.T) {
	t.Run("returns false for repo without submodules", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		hasSubmodules, err := HasSubmodules(tempDir)
		if err != nil {
			t.Fatalf("HasSubmodules failed: %v", err)
		}
		if hasSubmodules {
			t.Error("expected repo without submodules to return false")
		}
	})

	t.Run("detects submodules when present", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		gitmodulesPath := filepath.Join(repo.Path, ".gitmodules")
		gitmodulesContent := `[submodule "test"]
	path = test
	url = https://example.com/test.git
`
		if err := os.WriteFile(gitmodulesPath, []byte(gitmodulesContent), fs.FileGit); err != nil {
			t.Fatalf("failed to create .gitmodules: %v", err)
		}

		hasSubmodules, err := HasSubmodules(repo.Path)
		if err != nil {
			t.Fatalf("HasSubmodules failed: %v", err)
		}
		if !hasSubmodules {
			t.Error("expected submodules to be detected")
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := HasSubmodules(tempDir)
		if err == nil {
			t.Fatal("HasSubmodules should fail for non-git directory")
		}
	})
}

func TestHasUnpushedCommits(t *testing.T) {
	t.Run("returns error for repo with no upstream", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		hasUnpushed, err := HasUnpushedCommits(repo.Path)
		if err == nil {
			t.Fatal("Expected error for repo with no upstream")
		}
		if hasUnpushed {
			t.Error("expected no unpushed commits for repo with no upstream")
		}
		if !errors.Is(err, ErrNoUpstreamConfigured) {
			t.Errorf("Expected ErrNoUpstreamConfigured, got: %v", err)
		}
	})

	t.Run("returns true for repo with unpushed commits", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		remoteRepo := testgit.NewTestRepo(t)

		cmd := exec.Command("git", "remote", "add", "origin", remoteRepo.Path) // nolint:gosec // Test uses controlled temp directory
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		cmd = exec.Command("git", "fetch", "origin")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to fetch from origin: %v", err)
		}

		cmd = exec.Command("git", "branch", "--set-upstream-to=origin/main", "main")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set upstream: %v", err)
		}

		testFile := filepath.Join(repo.Path, "new.txt")
		if err := os.WriteFile(testFile, []byte("new content"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "new commit")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		hasUnpushed, err := HasUnpushedCommits(repo.Path)
		if err != nil {
			t.Fatalf("HasUnpushedCommits failed: %v", err)
		}
		if !hasUnpushed {
			t.Error("expected unpushed commits to be detected")
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := HasUnpushedCommits(tempDir)
		if err == nil {
			t.Fatal("HasUnpushedCommits should fail for non-git directory")
		}
	})
}

func TestListLocalBranches(t *testing.T) {
	t.Run("returns single branch for new repo", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		branches, err := ListLocalBranches(repo.Path)
		if err != nil {
			t.Fatalf("ListLocalBranches failed: %v", err)
		}

		if len(branches) != 1 {
			t.Errorf("expected 1 branch, got %d: %v", len(branches), branches)
		}
		if branches[0] != "main" {
			t.Errorf("expected 'main', got '%s'", branches[0])
		}
	})

	t.Run("returns multiple branches", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		cmd := exec.Command("git", "checkout", "-b", "feature")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		cmd = exec.Command("git", "checkout", "-b", "develop")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create develop branch: %v", err)
		}

		branches, err := ListLocalBranches(repo.Path)
		if err != nil {
			t.Fatalf("ListLocalBranches failed: %v", err)
		}

		if len(branches) != 3 {
			t.Errorf("expected 3 branches, got %d: %v", len(branches), branches)
		}

		expectedBranches := []string{"develop", "feature", "main"}
		for i, expected := range expectedBranches {
			if i >= len(branches) || branches[i] != expected {
				t.Errorf("expected branches %v, got %v", expectedBranches, branches)
				break
			}
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := ListLocalBranches(tempDir)
		if err == nil {
			t.Fatal("ListLocalBranches should fail for non-git directory")
		}
	})
}

func TestBranchExists(t *testing.T) {
	t.Run("returns true for existing branch", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		cmd := exec.Command("git", "checkout", "-b", "feature")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		exists, err := BranchExists(repo.Path, "feature")
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if !exists {
			t.Error("expected feature branch to exist")
		}
	})

	t.Run("returns false for non-existent branch", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		exists, err := BranchExists(repo.Path, "nonexistent")
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if exists {
			t.Error("expected nonexistent branch to not exist")
		}
	})

	t.Run("fails with empty repo path", func(t *testing.T) {
		_, err := BranchExists("", "main")
		if err == nil {
			t.Fatal("BranchExists should fail with empty repo path")
		}
		if err.Error() != "repository path and branch name cannot be empty" {
			t.Errorf("expected empty path error, got: %v", err)
		}
	})

	t.Run("fails with empty branch name", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		_, err := BranchExists(repo.Path, "")
		if err == nil {
			t.Fatal("BranchExists should fail with empty branch name")
		}
		if err.Error() != "repository path and branch name cannot be empty" {
			t.Errorf("expected empty branch name error, got: %v", err)
		}
	})

	t.Run("returns false for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		exists, err := BranchExists(tempDir, "main")
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if exists {
			t.Error("expected branch to not exist in non-git directory")
		}
	})

	t.Run("returns true for remote branch", func(t *testing.T) {
		originRepo := testgit.NewTestRepo(t)
		cmd := exec.Command("git", "checkout", "-b", "remote-feature")
		cmd.Dir = originRepo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create remote-feature branch: %v", err)
		}

		tempDir := t.TempDir()
		bareRepoPath := filepath.Join(tempDir, "bare")
		cloneCmd := exec.Command("git", "clone", "--bare", originRepo.Path, bareRepoPath) // nolint:gosec
		if err := cloneCmd.Run(); err != nil {
			t.Fatalf("failed to create bare clone: %v", err)
		}

		localRepoPath := filepath.Join(tempDir, "local")
		localCloneCmd := exec.Command("git", "clone", bareRepoPath, localRepoPath) // nolint:gosec
		if err := localCloneCmd.Run(); err != nil {
			t.Fatalf("failed to clone locally: %v", err)
		}

		exists, err := BranchExists(localRepoPath, "remote-feature")
		if err != nil {
			t.Fatalf("BranchExists failed: %v", err)
		}
		if !exists {
			t.Error("expected remote-feature branch to exist via remote reference")
		}
	})
}

func TestIsInsideGitRepo_ValidRepo(t *testing.T) {
	repo := testgit.NewTestRepo(t)

	if !IsInsideGitRepo(repo.Path) {
		t.Error("Expected IsInsideGitRepo to return true for valid git repository")
	}
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

	t.Run("fails for detached HEAD", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create git directory: %v", err)
		}

		headFile := filepath.Join(gitDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("abc1234567890\n"), fs.FileGit); err != nil {
			t.Fatalf("failed to create HEAD file: %v", err)
		}

		_, err := GetWorktreeInfo(tempDir)
		if err == nil {
			t.Fatal("expected error for detached HEAD")
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
		if err.Error() != "bare repository path cannot be empty" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty worktree path", func(t *testing.T) {
		err := CreateWorktreeWithNewBranch("/some/repo", "", "branch", true)
		if err == nil {
			t.Fatal("expected error for empty worktree path")
		}
		if err.Error() != "worktree path cannot be empty" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty branch name", func(t *testing.T) {
		err := CreateWorktreeWithNewBranch("/some/repo", "/tmp/wt", "", true)
		if err == nil {
			t.Fatal("expected error for empty branch name")
		}
		if err.Error() != "branch name cannot be empty" {
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
		if err.Error() != "base reference cannot be empty" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("fails with empty bare repo path", func(t *testing.T) {
		err := CreateWorktreeWithNewBranchFrom("", "/wt", "branch", "main", true)
		if err == nil {
			t.Fatal("expected error for empty bare repo path")
		}
	})

	t.Run("fails with empty worktree path", func(t *testing.T) {
		err := CreateWorktreeWithNewBranchFrom("/repo", "", "branch", "main", true)
		if err == nil {
			t.Fatal("expected error for empty worktree path")
		}
	})

	t.Run("fails with empty branch name", func(t *testing.T) {
		err := CreateWorktreeWithNewBranchFrom("/repo", "/wt", "", "main", true)
		if err == nil {
			t.Fatal("expected error for empty branch name")
		}
	})
}

func TestCreateWorktreeDetached(t *testing.T) {
	t.Run("fails with empty bare repo path", func(t *testing.T) {
		err := CreateWorktreeDetached("", "/wt", "v1.0.0", true)
		if err == nil {
			t.Fatal("expected error for empty bare repo path")
		}
	})

	t.Run("fails with empty worktree path", func(t *testing.T) {
		err := CreateWorktreeDetached("/repo", "", "v1.0.0", true)
		if err == nil {
			t.Fatal("expected error for empty worktree path")
		}
	})

	t.Run("fails with empty ref", func(t *testing.T) {
		err := CreateWorktreeDetached("/repo", "/wt", "", true)
		if err == nil {
			t.Fatal("expected error for empty ref")
		}
		if err.Error() != "ref cannot be empty" {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestRefExists(t *testing.T) {
	t.Run("returns error for non-existent ref", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		err := RefExists(repo.Path, "nonexistent-tag-12345")
		if err == nil {
			t.Error("expected error for non-existent ref")
		}
	})

	t.Run("returns nil for existing branch", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		err := RefExists(repo.Path, "main")
		if err != nil {
			t.Errorf("expected nil for existing branch, got: %v", err)
		}
	})
}

func TestGetSyncStatus(t *testing.T) {
	t.Parallel()

	t.Run("returns no upstream when none configured", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		status := GetSyncStatus(repo.Path)

		if !status.NoUpstream {
			t.Error("expected NoUpstream to be true")
		}
		if status.Upstream != "" {
			t.Errorf("expected empty upstream, got %s", status.Upstream)
		}
		if status.Ahead != 0 || status.Behind != 0 {
			t.Errorf("expected 0 ahead/behind, got %d/%d", status.Ahead, status.Behind)
		}
		if status.Gone {
			t.Error("expected Gone to be false")
		}
	})

	t.Run("detects commits ahead of upstream", func(t *testing.T) {
		t.Parallel()
		// Create bare repo to act as remote
		remoteDir := t.TempDir()
		remoteRepo := filepath.Join(remoteDir, "remote.git")
		if err := os.MkdirAll(remoteRepo, fs.DirGit); err != nil { // nolint:gosec // Test uses controlled temp directory
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = remoteRepo
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create local repo and push
		repo := testgit.NewTestRepo(t)
		cmd = exec.Command("git", "remote", "add", "origin", remoteRepo) // nolint:gosec // Test uses controlled temp directory
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create local commit
		testFile := filepath.Join(repo.Path, "new.txt")
		if err := os.WriteFile(testFile, []byte("new"), fs.FileStrict); err != nil { // nolint:gosec // Test uses controlled temp directory
			t.Fatal(err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = repo.Path
		_ = cmd.Run()
		cmd = exec.Command("git", "commit", "-m", "local commit")
		cmd.Dir = repo.Path
		_ = cmd.Run()

		status := GetSyncStatus(repo.Path)

		if status.NoUpstream {
			t.Error("expected NoUpstream to be false")
		}
		if status.Upstream != "origin/main" {
			t.Errorf("expected upstream origin/main, got %s", status.Upstream)
		}
		if status.Ahead != 1 {
			t.Errorf("expected 1 commit ahead, got %d", status.Ahead)
		}
		if status.Behind != 0 {
			t.Errorf("expected 0 commits behind, got %d", status.Behind)
		}
		if status.Gone {
			t.Error("expected Gone to be false")
		}
	})

	t.Run("detects commits behind upstream", func(t *testing.T) {
		t.Parallel()
		// Create bare repo to act as remote
		remoteDir := t.TempDir()
		remoteRepo := filepath.Join(remoteDir, "remote.git")
		if err := os.MkdirAll(remoteRepo, fs.DirGit); err != nil { // nolint:gosec // Test uses controlled temp directory
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = remoteRepo
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create local repo and push
		repo := testgit.NewTestRepo(t)
		cmd = exec.Command("git", "remote", "add", "origin", remoteRepo) // nolint:gosec // Test uses controlled temp directory
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Create commit on remote (via a separate clone)
		tempClone := t.TempDir()
		cmd = exec.Command("git", "clone", remoteRepo, tempClone) // nolint:gosec // Test uses controlled temp directory
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "config", "user.email", "test@test.com")
		cmd.Dir = tempClone
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = tempClone
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "commit.gpgsign", "false")
		cmd.Dir = tempClone
		_ = cmd.Run()
		testFile := filepath.Join(tempClone, "remote.txt")
		if err := os.WriteFile(testFile, []byte("remote"), fs.FileStrict); err != nil { // nolint:gosec // Test uses controlled temp directory
			t.Fatal(err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempClone
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files in temp clone: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "remote commit")
		cmd.Dir = tempClone
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit in temp clone: %v", err)
		}
		cmd = exec.Command("git", "push")
		cmd.Dir = tempClone
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to push from temp clone: %v", err)
		}

		// Fetch in original repo
		cmd = exec.Command("git", "fetch")
		cmd.Dir = repo.Path
		_ = cmd.Run()

		status := GetSyncStatus(repo.Path)

		if status.Ahead != 0 {
			t.Errorf("expected 0 commits ahead, got %d", status.Ahead)
		}
		if status.Behind != 1 {
			t.Errorf("expected 1 commit behind, got %d", status.Behind)
		}
	})
}

func TestFetchPrune(t *testing.T) {
	t.Run("fetches and prunes stale remote refs", func(t *testing.T) {
		t.Parallel()

		remoteDir := t.TempDir()
		remoteRepo := filepath.Join(remoteDir, "remote.git")
		if err := os.MkdirAll(remoteRepo, fs.DirGit); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "init", "--bare")
		cmd.Dir = remoteRepo
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		repo := testgit.NewTestRepo(t)
		cmd = exec.Command("git", "remote", "add", "origin", remoteRepo) // nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "push", "-u", "origin", "main")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		cmd = exec.Command("git", "checkout", "-b", "feature-to-delete")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "push", "-u", "origin", "feature-to-delete")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		cmd = exec.Command("git", "checkout", "main")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		cmd = exec.Command("git", "branch", "-r")
		cmd.Dir = repo.Path
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(out.String(), "origin/feature-to-delete") {
			t.Fatal("expected origin/feature-to-delete to exist before prune")
		}

		cmd = exec.Command("git", "branch", "-D", "feature-to-delete")
		cmd.Dir = remoteRepo
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		if err := FetchPrune(repo.Path); err != nil {
			t.Fatalf("FetchPrune failed: %v", err)
		}

		cmd = exec.Command("git", "branch", "-r")
		cmd.Dir = repo.Path
		out.Reset()
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "origin/feature-to-delete") {
			t.Error("expected origin/feature-to-delete to be pruned, but it still exists")
		}
	})

	t.Run("returns error for non-git directory", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		err := FetchPrune(tempDir)
		if err == nil {
			t.Fatal("expected error for non-git directory")
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

func TestGetLastCommitTime(t *testing.T) {
	t.Run("returns timestamp for repo with commits", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		timestamp := GetLastCommitTime(repo.Path)
		if timestamp == 0 {
			t.Error("expected non-zero timestamp for repo with commits")
		}

		now := time.Now().Unix()
		if timestamp > now {
			t.Errorf("timestamp %d is in the future (now: %d)", timestamp, now)
		}
		// Should be within last hour (test just created the repo)
		if now-timestamp > 3600 {
			t.Errorf("timestamp %d is too old (more than 1 hour ago)", timestamp)
		}
	})

	t.Run("returns 0 for empty repo", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, "empty.bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		timestamp := GetLastCommitTime(bareDir)
		if timestamp != 0 {
			t.Errorf("expected 0 for empty repo, got %d", timestamp)
		}
	})

	t.Run("returns 0 for non-git directory", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		timestamp := GetLastCommitTime(tempDir)
		if timestamp != 0 {
			t.Errorf("expected 0 for non-git directory, got %d", timestamp)
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

func TestGetOngoingOperation(t *testing.T) {
	t.Run("returns empty for clean repo", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		op, err := GetOngoingOperation(repo.Path)
		if err != nil {
			t.Fatalf("GetOngoingOperation failed: %v", err)
		}
		if op != "" {
			t.Errorf("expected empty operation, got %q", op)
		}
	})

	t.Run("detects merge operation", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
		if err := os.WriteFile(mergeHead, []byte("commit-hash"), fs.FileGit); err != nil {
			t.Fatal(err)
		}

		op, err := GetOngoingOperation(tempDir)
		if err != nil {
			t.Fatalf("GetOngoingOperation failed: %v", err)
		}
		if op != "merging" {
			t.Errorf("expected 'merging', got %q", op)
		}
	})

	t.Run("detects rebase operation", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		rebaseDir := filepath.Join(gitDir, "rebase-merge")
		if err := os.Mkdir(rebaseDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		op, err := GetOngoingOperation(tempDir)
		if err != nil {
			t.Fatalf("GetOngoingOperation failed: %v", err)
		}
		if op != "rebasing" {
			t.Errorf("expected 'rebasing', got %q", op)
		}
	})

	t.Run("detects cherry-pick operation", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")

		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		cherryHead := filepath.Join(gitDir, "CHERRY_PICK_HEAD")
		if err := os.WriteFile(cherryHead, []byte("commit-hash"), fs.FileGit); err != nil {
			t.Fatal(err)
		}

		op, err := GetOngoingOperation(tempDir)
		if err != nil {
			t.Fatalf("GetOngoingOperation failed: %v", err)
		}
		if op != "cherry-picking" {
			t.Errorf("expected 'cherry-picking', got %q", op)
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		_, err := GetOngoingOperation(tempDir)
		if err == nil {
			t.Fatal("expected error for non-git directory")
		}
	})
}

func TestGetConflictCount(t *testing.T) {
	t.Run("returns 0 for repo with no conflicts", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		count, err := GetConflictCount(repo.Path)
		if err != nil {
			t.Fatalf("GetConflictCount failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 conflicts, got %d", count)
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		_, err := GetConflictCount(tempDir)
		if err == nil {
			t.Fatal("expected error for non-git directory")
		}
	})
}

func TestGetStashCount(t *testing.T) {
	t.Run("returns 0 for repo with no stashes", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		count, err := GetStashCount(repo.Path)
		if err != nil {
			t.Fatalf("GetStashCount failed: %v", err)
		}
		if count != 0 {
			t.Errorf("expected 0 stashes, got %d", count)
		}
	})

	t.Run("returns correct count for repo with stashes", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		// Create a file and stash it
		testFile := filepath.Join(repo.Path, "stash-me.txt")
		if err := os.WriteFile(testFile, []byte("stash content"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}
		cmd := exec.Command("git", "add", ".")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("git add failed: %v", err)
		}
		cmd = exec.Command("git", "stash", "push", "-m", "test stash 1")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("git stash failed: %v", err)
		}

		// Create another stash
		if err := os.WriteFile(testFile, []byte("stash content 2"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("git add failed: %v", err)
		}
		cmd = exec.Command("git", "stash", "push", "-m", "test stash 2")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("git stash failed: %v", err)
		}

		count, err := GetStashCount(repo.Path)
		if err != nil {
			t.Fatalf("GetStashCount failed: %v", err)
		}
		if count != 2 {
			t.Errorf("expected 2 stashes, got %d", count)
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		_, err := GetStashCount(tempDir)
		if err == nil {
			t.Fatal("expected error for non-git directory")
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

func TestGetGitDir(t *testing.T) {
	t.Run("returns .git directory for regular repos", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		gitDir, err := GetGitDir(repo.Path)
		if err != nil {
			t.Fatalf("GetGitDir failed: %v", err)
		}

		expected := filepath.Join(repo.Path, ".git")
		if gitDir != expected {
			t.Errorf("expected %s, got %s", expected, gitDir)
		}
	})

	t.Run("resolves gitdir from .git file in worktrees", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		// Create fake git dir
		fakeGitDir := filepath.Join(tempDir, "fake-git-dir")
		if err := os.Mkdir(fakeGitDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create fake git dir: %v", err)
		}

		// Create worktree directory with .git file
		worktreeDir := filepath.Join(tempDir, "worktree")
		if err := os.Mkdir(worktreeDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create worktree dir: %v", err)
		}

		gitFile := filepath.Join(worktreeDir, ".git")
		content := "gitdir: " + fakeGitDir
		if err := os.WriteFile(gitFile, []byte(content), fs.FileStrict); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		gitDir, err := GetGitDir(worktreeDir)
		if err != nil {
			t.Fatalf("GetGitDir failed: %v", err)
		}
		if gitDir != fakeGitDir {
			t.Errorf("expected %s, got %s", fakeGitDir, gitDir)
		}
	})

	t.Run("handles relative paths in .git file", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		// Create fake git dir as sibling
		fakeGitDir := filepath.Join(tempDir, ".bare", "worktrees", "feature")
		if err := os.MkdirAll(fakeGitDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create fake git dir: %v", err)
		}

		// Create worktree directory with .git file using relative path
		worktreeDir := filepath.Join(tempDir, "feature")
		if err := os.Mkdir(worktreeDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create worktree dir: %v", err)
		}

		gitFile := filepath.Join(worktreeDir, ".git")
		content := "gitdir: ../.bare/worktrees/feature"
		if err := os.WriteFile(gitFile, []byte(content), fs.FileStrict); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		gitDir, err := GetGitDir(worktreeDir)
		if err != nil {
			t.Fatalf("GetGitDir failed: %v", err)
		}
		if gitDir != fakeGitDir {
			t.Errorf("expected %s, got %s", fakeGitDir, gitDir)
		}
	})

	t.Run("returns error for non-git directories", func(t *testing.T) {
		t.Parallel()
		tempDir := t.TempDir()

		_, err := GetGitDir(tempDir)
		if err == nil {
			t.Fatal("expected error for non-git directory")
		}
	})
}

func TestAddRemote(t *testing.T) {
	t.Run("adds new remote", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		err := AddRemote(repo.Path, "fork", "https://github.com/fork/repo.git")
		if err != nil {
			t.Fatalf("AddRemote failed: %v", err)
		}

		// Verify remote was added
		cmd := exec.Command("git", "remote", "get-url", "fork")
		cmd.Dir = repo.Path
		var out bytes.Buffer
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			t.Fatalf("remote was not added: %v", err)
		}

		if strings.TrimSpace(out.String()) != "https://github.com/fork/repo.git" {
			t.Errorf("unexpected remote URL: %s", out.String())
		}
	})

	t.Run("fails with empty path", func(t *testing.T) {
		err := AddRemote("", "fork", "https://github.com/fork/repo.git")
		if err == nil {
			t.Fatal("expected error for empty path")
		}
	})

	t.Run("fails with empty name", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		err := AddRemote(repo.Path, "", "https://github.com/fork/repo.git")
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})
}

func TestRemoteExists(t *testing.T) {
	t.Run("returns true for existing remote", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Add a remote first
		cmd := exec.Command("git", "remote", "add", "upstream", "https://github.com/upstream/repo.git")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		exists, err := RemoteExists(repo.Path, "upstream")
		if err != nil {
			t.Fatalf("RemoteExists failed: %v", err)
		}
		if !exists {
			t.Error("expected remote to exist")
		}
	})

	t.Run("returns false for non-existing remote", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		exists, err := RemoteExists(repo.Path, "nonexistent")
		if err != nil {
			t.Fatalf("RemoteExists failed: %v", err)
		}
		if exists {
			t.Error("expected remote to not exist")
		}
	})
}

func TestFetchBranch(t *testing.T) {
	t.Run("fetches branch from remote", func(t *testing.T) {
		// Create an "upstream" repo to fetch from
		upstream := testgit.NewTestRepo(t)

		// Create a branch in upstream
		cmd := exec.Command("git", "checkout", "-b", "feature-branch")
		cmd.Dir = upstream.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}

		// Create another file and commit
		testFile := filepath.Join(upstream.Path, "feature.txt")
		if err := os.WriteFile(testFile, []byte("feature"), fs.FileGit); err != nil {
			t.Fatalf("failed to create file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = upstream.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "feature commit")
		cmd.Dir = upstream.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Create a downstream repo
		downstream := testgit.NewTestRepo(t)

		// Add upstream as remote
		cmd = exec.Command("git", "remote", "add", "upstream", upstream.Path) //nolint:gosec // Test helper
		cmd.Dir = downstream.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		// Fetch the branch
		err := FetchBranch(downstream.Path, "upstream", "feature-branch")
		if err != nil {
			t.Fatalf("FetchBranch failed: %v", err)
		}

		// Verify the branch was fetched
		cmd = exec.Command("git", "rev-parse", "--verify", "upstream/feature-branch")
		cmd.Dir = downstream.Path
		if err := cmd.Run(); err != nil {
			t.Error("branch was not fetched")
		}
	})

	t.Run("fails for non-existing branch", func(t *testing.T) {
		upstream := testgit.NewTestRepo(t)
		downstream := testgit.NewTestRepo(t)

		// Add upstream as remote
		cmd := exec.Command("git", "remote", "add", "upstream", upstream.Path) //nolint:gosec // Test helper
		cmd.Dir = downstream.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		err := FetchBranch(downstream.Path, "upstream", "nonexistent-branch")
		if err == nil {
			t.Error("expected error for non-existing branch")
		}
	})
}

func TestDeleteBranch(t *testing.T) {
	t.Run("deletes existing branch", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		// Create a worktree to have something to work with
		worktreePath := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", testDefaultBranch) //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		// Configure git for commits
		cmd = exec.Command("git", "config", "user.email", "test@test.com") //nolint:gosec
		cmd.Dir = worktreePath
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.name", "Test") //nolint:gosec
		cmd.Dir = worktreePath
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "commit.gpgsign", "false") //nolint:gosec
		cmd.Dir = worktreePath
		_ = cmd.Run()

		// Create an initial commit (needed before creating additional branches)
		testFile := filepath.Join(worktreePath, "init.txt")
		if err := os.WriteFile(testFile, []byte("init"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "add", "init.txt") //nolint:gosec
		cmd.Dir = worktreePath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "initial commit") //nolint:gosec
		cmd.Dir = worktreePath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Create a feature branch
		cmd = exec.Command("git", "branch", "feature") //nolint:gosec
		cmd.Dir = worktreePath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature branch: %v", err)
		}

		// Verify branch exists
		exists, err := BranchExists(bareDir, "feature")
		if err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatal("feature branch should exist before deletion")
		}

		// Delete the branch
		if err := DeleteBranch(bareDir, "feature", false); err != nil {
			t.Fatalf("DeleteBranch failed: %v", err)
		}

		// Verify branch no longer exists
		exists, err = BranchExists(bareDir, "feature")
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Error("feature branch should not exist after deletion")
		}
	})

	t.Run("returns error for non-existent branch", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		// Create a worktree to initialize the repo
		worktreePath := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", testDefaultBranch) //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		err := DeleteBranch(bareDir, "nonexistent", false)
		if err == nil {
			t.Error("expected error for non-existent branch")
		}
	})

	t.Run("fails on unmerged branch without force", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		// Create main worktree
		mainPath := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", mainPath, "-b", testDefaultBranch) //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create main worktree: %v", err)
		}

		// Create feature worktree with new branch
		featurePath := filepath.Join(tempDir, "feature")
		cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature worktree: %v", err)
		}

		// Configure git for commits
		cmd = exec.Command("git", "config", "user.email", "test@test.com") //nolint:gosec
		cmd.Dir = featurePath
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.name", "Test") //nolint:gosec
		cmd.Dir = featurePath
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "commit.gpgsign", "false") //nolint:gosec
		cmd.Dir = featurePath
		_ = cmd.Run()

		// Add a commit to feature branch (making it unmerged)
		testFile := filepath.Join(featurePath, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "add", "test.txt") //nolint:gosec
		cmd.Dir = featurePath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "test commit") //nolint:gosec
		cmd.Dir = featurePath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Remove the worktree first (required before branch deletion)
		if err := RemoveWorktree(bareDir, featurePath, true); err != nil {
			t.Fatalf("failed to remove worktree: %v", err)
		}

		// Try to delete unmerged branch without force - should fail
		err := DeleteBranch(bareDir, "feature", false)
		if err == nil {
			t.Error("expected error when deleting unmerged branch without force")
		}
	})

	t.Run("force deletes unmerged branch", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		// Create main worktree
		mainPath := filepath.Join(tempDir, "main")
		cmd := exec.Command("git", "worktree", "add", mainPath, "-b", testDefaultBranch) //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create main worktree: %v", err)
		}

		// Create feature worktree with new branch
		featurePath := filepath.Join(tempDir, "feature")
		cmd = exec.Command("git", "worktree", "add", "-b", "feature", featurePath) //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create feature worktree: %v", err)
		}

		// Configure git for commits
		cmd = exec.Command("git", "config", "user.email", "test@test.com") //nolint:gosec
		cmd.Dir = featurePath
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "user.name", "Test") //nolint:gosec
		cmd.Dir = featurePath
		_ = cmd.Run()
		cmd = exec.Command("git", "config", "commit.gpgsign", "false") //nolint:gosec
		cmd.Dir = featurePath
		_ = cmd.Run()

		// Add a commit to feature branch (making it unmerged)
		testFile := filepath.Join(featurePath, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "add", "test.txt") //nolint:gosec
		cmd.Dir = featurePath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add file: %v", err)
		}
		cmd = exec.Command("git", "commit", "-m", "test commit") //nolint:gosec
		cmd.Dir = featurePath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit: %v", err)
		}

		// Remove the worktree first
		if err := RemoveWorktree(bareDir, featurePath, true); err != nil {
			t.Fatalf("failed to remove worktree: %v", err)
		}

		// Force delete unmerged branch - should succeed
		if err := DeleteBranch(bareDir, "feature", true); err != nil {
			t.Fatalf("DeleteBranch with force failed: %v", err)
		}

		// Verify branch no longer exists
		exists, err := BranchExists(bareDir, "feature")
		if err != nil {
			t.Fatal(err)
		}
		if exists {
			t.Error("feature branch should not exist after forced deletion")
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

func TestGetDefaultBranch(t *testing.T) {
	t.Run("returns main for bare repo with main branch", func(t *testing.T) {
		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, "test.bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create bare directory: %v", err)
		}

		// Initialize bare repo with main branch
		cmd := exec.Command("git", "init", "--bare", "-b", "main") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init bare repo: %v", err)
		}

		branch, err := GetDefaultBranch(bareDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if branch != "main" {
			t.Errorf("expected 'main', got %q", branch)
		}
	})

	t.Run("returns master for bare repo with master branch", func(t *testing.T) {
		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, "test.bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create bare directory: %v", err)
		}

		// Initialize bare repo with master branch
		cmd := exec.Command("git", "init", "--bare", "-b", "master") //nolint:gosec
		cmd.Dir = bareDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init bare repo: %v", err)
		}

		branch, err := GetDefaultBranch(bareDir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if branch != "master" {
			t.Errorf("expected 'master', got %q", branch)
		}
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		_, err := GetDefaultBranch("")
		if err == nil {
			t.Error("expected error for empty path")
		}
	})
}

func TestIsBranchMerged(t *testing.T) {
	t.Run("returns true for branch merged via regular merge", func(t *testing.T) {
		repo := testgit.NewTestRepo(t, testDefaultBranch)

		// Create feature branch with a commit
		repo.CreateBranch("feature")
		repo.Checkout("feature")
		repo.WriteFile("feature.txt", "content")
		repo.Add("feature.txt")
		repo.Commit("Add feature")

		// Merge into main
		repo.Checkout(testDefaultBranch)
		repo.Merge("feature")

		merged, err := IsBranchMerged(repo.Path, "feature", testDefaultBranch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !merged {
			t.Error("expected branch to be detected as merged")
		}
	})

	t.Run("returns false for unmerged branch", func(t *testing.T) {
		repo := testgit.NewTestRepo(t, testDefaultBranch)

		// Create feature branch with a commit
		repo.CreateBranch("feature")
		repo.Checkout("feature")
		repo.WriteFile("feature.txt", "content")
		repo.Add("feature.txt")
		repo.Commit("Add feature")

		// Don't merge - go back to main
		repo.Checkout(testDefaultBranch)

		merged, err := IsBranchMerged(repo.Path, "feature", testDefaultBranch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if merged {
			t.Error("expected branch to NOT be detected as merged")
		}
	})

	t.Run("returns true for branch merged via squash merge", func(t *testing.T) {
		repo := testgit.NewTestRepo(t, testDefaultBranch)

		// Create feature branch with commits
		repo.CreateBranch("feature")
		repo.Checkout("feature")
		repo.WriteFile("feature.txt", "content")
		repo.Add("feature.txt")
		repo.Commit("Add feature")

		// Squash merge into main
		repo.Checkout(testDefaultBranch)
		repo.SquashMerge("feature")

		merged, err := IsBranchMerged(repo.Path, "feature", testDefaultBranch)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !merged {
			t.Error("expected squash-merged branch to be detected as merged")
		}
	})
}
