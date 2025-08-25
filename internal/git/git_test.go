package git

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/fs"
	testgit "github.com/sqve/grove/internal/testutil/git"
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
	testRepo := testgit.NewTestRepo(t)

	cmd := exec.Command("git", "checkout", "-b", "feature")
	cmd.Dir = testRepo.Path
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create feature branch: %v", err)
	}

	origCacheDir := os.Getenv("TEST_CACHE_DIR")
	_ = os.Setenv("TEST_CACHE_DIR", tempDir)
	defer func() {
		if origCacheDir == "" {
			_ = os.Unsetenv("TEST_CACHE_DIR")
		} else {
			_ = os.Setenv("TEST_CACHE_DIR", origCacheDir)
		}
	}()

	branches1, err := ListRemoteBranches("file://" + testRepo.Path)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}
	if len(branches1) == 0 {
		t.Fatal("expected branches from repository")
	}

	branches2, err := ListRemoteBranches("file://" + testRepo.Path)
	if err != nil {
		t.Fatalf("cached call failed: %v", err)
	}
	if len(branches1) != len(branches2) {
		t.Fatalf("cache inconsistency: first=%d, cached=%d", len(branches1), len(branches2))
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
}

func TestIsInsideGitRepo_ValidRepo(t *testing.T) {
	repo := testgit.NewTestRepo(t)

	if !IsInsideGitRepo(repo.Path) {
		t.Error("Expected IsInsideGitRepo to return true for valid git repository")
	}
}
