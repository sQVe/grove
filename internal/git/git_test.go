package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

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

	err := Clone("file:///nonexistent/repo.git", bareDir, true)
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

	err := Clone("file:///nonexistent/repo.git", bareDir, false)
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

func TestListRemoteBranchesFromURL(t *testing.T) {
	_, err := ListRemoteBranches("file:///nonexistent/repo.git")
	if err == nil {
		t.Fatal("Expected error for non-existent repo")
	}
}

func TestListRemoteBranchesCaching(t *testing.T) {
	tempDir := t.TempDir()
	testURL := "file:///nonexistent/repo.git"

	origCacheDir := os.Getenv("TEST_CACHE_DIR")
	_ = os.Setenv("TEST_CACHE_DIR", tempDir)
	defer func() {
		if origCacheDir == "" {
			_ = os.Unsetenv("TEST_CACHE_DIR")
		} else {
			_ = os.Setenv("TEST_CACHE_DIR", origCacheDir)
		}
	}()

	_, err := ListRemoteBranches(testURL)
	if err == nil {
		t.Fatal("Expected error for non-existent repo on first call")
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

		if branch != "main" {
			t.Errorf("expected branch 'main', got '%s'", branch)
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

func TestHasUncommittedChanges(t *testing.T) {
	t.Run("returns false for clean repo", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		hasChanges, err := HasUncommittedChanges(tempDir)
		if err != nil {
			t.Fatalf("HasUncommittedChanges failed: %v", err)
		}
		if hasChanges {
			t.Error("expected clean repo to have no uncommitted changes")
		}
	})

	t.Run("returns true for untracked files", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		testFile := filepath.Join(tempDir, "untracked.txt")
		if err := os.WriteFile(testFile, []byte("content"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		hasChanges, err := HasUncommittedChanges(tempDir)
		if err != nil {
			t.Fatalf("HasUncommittedChanges failed: %v", err)
		}
		if !hasChanges {
			t.Error("expected untracked files to be detected as uncommitted changes")
		}
	})

	t.Run("returns true for modified files", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		testFile := filepath.Join(tempDir, "tracked.txt")
		if err := os.WriteFile(testFile, []byte("initial"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.name: %v", err)
		}

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.email: %v", err)
		}

		cmd = exec.Command("git", "config", "commit.gpgsign", "false")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to disable GPG signing: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit files: %v", err)
		}

		if err := os.WriteFile(testFile, []byte("modified"), fs.FileStrict); err != nil {
			t.Fatalf("failed to modify test file: %v", err)
		}

		hasChanges, err := HasUncommittedChanges(tempDir)
		if err != nil {
			t.Fatalf("HasUncommittedChanges failed: %v", err)
		}
		if !hasChanges {
			t.Error("expected modified files to be detected as uncommitted changes")
		}
	})

	t.Run("returns true for staged files", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		testFile := filepath.Join(tempDir, "staged.txt")
		if err := os.WriteFile(testFile, []byte("content"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to stage files: %v", err)
		}

		hasChanges, err := HasUncommittedChanges(tempDir)
		if err != nil {
			t.Fatalf("HasUncommittedChanges failed: %v", err)
		}
		if !hasChanges {
			t.Error("expected staged files to be detected as uncommitted changes")
		}
	})

	t.Run("fails for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := HasUncommittedChanges(tempDir)
		if err == nil {
			t.Fatal("HasUncommittedChanges should fail for non-git directory")
		}
	})
}

func TestListWorktrees(t *testing.T) {
	t.Run("returns empty slice when no worktrees", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.name: %v", err)
		}

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.email: %v", err)
		}

		cmd = exec.Command("git", "config", "commit.gpgsign", "false")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to disable GPG signing: %v", err)
		}

		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit files: %v", err)
		}

		worktrees, err := ListWorktrees(tempDir)
		if err != nil {
			t.Fatalf("ListWorktrees failed: %v", err)
		}

		if len(worktrees) != 0 {
			t.Errorf("expected empty slice, got %v", worktrees)
		}
	})

	t.Run("returns worktree paths when worktrees exist", func(t *testing.T) {
		tempDir := t.TempDir()
		worktreeDir := filepath.Join(tempDir, "branch-worktree")

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.name: %v", err)
		}

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.email: %v", err)
		}

		cmd = exec.Command("git", "config", "commit.gpgsign", "false")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to disable GPG signing: %v", err)
		}

		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit files: %v", err)
		}

		cmd = exec.Command("git", "worktree", "add", worktreeDir, "-b", "feature") // nolint:gosec // Test uses controlled temp directory
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		worktrees, err := ListWorktrees(tempDir)
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
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.name: %v", err)
		}

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.email: %v", err)
		}

		cmd = exec.Command("git", "config", "commit.gpgsign", "false")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to disable GPG signing: %v", err)
		}

		testFile := filepath.Join(tempDir, "conflict.txt")
		if err := os.WriteFile(testFile, []byte("initial"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit files: %v", err)
		}

		cmd = exec.Command("git", "checkout", "-b", "branch1")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create branch1: %v", err)
		}

		if err := os.WriteFile(testFile, []byte("branch1"), fs.FileStrict); err != nil {
			t.Fatalf("failed to modify test file on branch1: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files on branch1: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "branch1 changes")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit branch1 changes: %v", err)
		}

		cmd = exec.Command("git", "checkout", "main")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout main: %v", err)
		}

		if err := os.WriteFile(testFile, []byte("main"), fs.FileStrict); err != nil {
			t.Fatalf("failed to modify test file on main: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files on main: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "main changes")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit main changes: %v", err)
		}

		cmd = exec.Command("git", "merge", "branch1")
		cmd.Dir = tempDir
		_ = cmd.Run() // Expected to fail with conflict

		hasConflicts, err := HasUnresolvedConflicts(tempDir)
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
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.name: %v", err)
		}

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set git user.email: %v", err)
		}

		cmd = exec.Command("git", "config", "commit.gpgsign", "false")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to disable GPG signing: %v", err)
		}

		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("test"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "initial")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit files: %v", err)
		}

		submoduleDir := filepath.Join(tempDir, "submodule")
		if err := os.Mkdir(submoduleDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create submodule directory: %v", err)
		}

		cmd = exec.Command("git", "init")
		cmd.Dir = submoduleDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize submodule git repository: %v", err)
		}

		cmd = exec.Command("git", "config", "user.name", "Test")
		cmd.Dir = submoduleDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set submodule git user.name: %v", err)
		}

		cmd = exec.Command("git", "config", "user.email", "test@example.com")
		cmd.Dir = submoduleDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set submodule git user.email: %v", err)
		}

		cmd = exec.Command("git", "config", "commit.gpgsign", "false")
		cmd.Dir = submoduleDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to disable submodule GPG signing: %v", err)
		}

		subFile := filepath.Join(submoduleDir, "sub.txt")
		if err := os.WriteFile(subFile, []byte("sub"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create submodule file: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = submoduleDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add submodule files: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "submodule initial")
		cmd.Dir = submoduleDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit submodule files: %v", err)
		}

		cmd = exec.Command("git", "submodule", "add", "./submodule", "submodule")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add submodule: %v", err)
		}

		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to stage submodule changes: %v", err)
		}

		cmd = exec.Command("git", "commit", "-m", "add submodule")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to commit submodule: %v", err)
		}

		hasSubmodules, err := HasSubmodules(tempDir)
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
