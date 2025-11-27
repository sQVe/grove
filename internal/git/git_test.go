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
		testFile := filepath.Join(tempClone, "remote.txt")
		if err := os.WriteFile(testFile, []byte("remote"), fs.FileStrict); err != nil { // nolint:gosec // Test uses controlled temp directory
			t.Fatal(err)
		}
		cmd = exec.Command("git", "add", ".")
		cmd.Dir = tempClone
		_ = cmd.Run()
		cmd = exec.Command("git", "commit", "-m", "remote commit")
		cmd.Dir = tempClone
		_ = cmd.Run()
		cmd = exec.Command("git", "push")
		cmd.Dir = tempClone
		_ = cmd.Run()

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

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		worktreesDir := filepath.Join(bareDir, "worktrees", "feature")
		if err := os.MkdirAll(worktreesDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		if IsWorktreeLocked(bareDir, "feature") {
			t.Error("expected unlocked worktree to return false")
		}
	})

	t.Run("returns true for locked worktree", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatal(err)
		}

		worktreesDir := filepath.Join(bareDir, "worktrees", "feature")
		if err := os.MkdirAll(worktreesDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		lockFile := filepath.Join(worktreesDir, "locked")
		if err := os.WriteFile(lockFile, []byte("locked by test"), fs.FileStrict); err != nil {
			t.Fatal(err)
		}

		if !IsWorktreeLocked(bareDir, "feature") {
			t.Error("expected locked worktree to return true")
		}
	})

	t.Run("returns false for non-existent worktree", func(t *testing.T) {
		t.Parallel()

		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.MkdirAll(bareDir, fs.DirStrict); err != nil {
			t.Fatal(err)
		}

		if IsWorktreeLocked(bareDir, "nonexistent") {
			t.Error("expected non-existent worktree to return false")
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
