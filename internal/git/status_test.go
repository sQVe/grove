package git

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/sqve/grove/internal/fs"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

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

	t.Run("detects gone upstream after branch deleted on remote", func(t *testing.T) {
		t.Parallel()
		// Create bare repo to act as remote
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

		// Create local repo and push main
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

		// Create and push a feature branch
		cmd = exec.Command("git", "checkout", "-b", "feature-gone")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		cmd = exec.Command("git", "push", "-u", "origin", "feature-gone")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Delete the branch on the remote
		cmd = exec.Command("git", "branch", "-D", "feature-gone")
		cmd.Dir = remoteRepo
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		// Fetch to update remote tracking refs (with prune)
		cmd = exec.Command("git", "fetch", "--prune")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		status := GetSyncStatus(repo.Path)

		if !status.Gone {
			t.Error("expected Gone to be true when upstream branch was deleted")
		}
		if status.NoUpstream {
			t.Error("expected NoUpstream to be false (branch still tracks deleted remote)")
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
