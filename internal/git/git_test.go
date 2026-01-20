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

	// Error message format varies by OS, but should indicate git not found
	errMsg := err.Error()
	if !strings.Contains(errMsg, "git") || !strings.Contains(errMsg, "executable file not found") {
		t.Errorf("expected git executable not found error, got: %v", err)
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

func TestClone(t *testing.T) {
	t.Run("returns error for non-existent repo in quiet mode", func(t *testing.T) {
		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, "test.bare")

		// quiet=true suppresses git's progress output but errors must still be captured
		err := Clone("file:///nonexistent/repo.git", bareDir, true, false)
		if err == nil {
			t.Fatal("expected error for non-existent repo")
		}

		if err.Error() == "" {
			t.Error("error message should not be empty even in quiet mode")
		}
	})

	t.Run("returns error for non-existent repo in verbose mode", func(t *testing.T) {
		tempDir := t.TempDir()
		bareDir := filepath.Join(tempDir, "test.bare")

		// quiet=false allows git's progress output; verify errors still work
		err := Clone("file:///nonexistent/repo.git", bareDir, false, false)
		if err == nil {
			t.Fatal("expected error for non-existent repo")
		}

		if err.Error() == "" {
			t.Error("error message should not be empty in verbose mode")
		}
	})
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

func TestIsInsideGitRepo_ValidRepo(t *testing.T) {
	repo := testgit.NewTestRepo(t)

	if !IsInsideGitRepo(repo.Path) {
		t.Error("Expected IsInsideGitRepo to return true for git repository")
	}
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

func TestRemoveRemote(t *testing.T) {
	t.Run("removes existing remote", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Add a remote first
		if err := AddRemote(repo.Path, "fork-remote", "https://github.com/fork/repo.git"); err != nil {
			t.Fatalf("AddRemote failed: %v", err)
		}

		// Verify it exists
		exists, _ := RemoteExists(repo.Path, "fork-remote")
		if !exists {
			t.Fatal("remote should exist before removal")
		}

		// Remove it
		if err := RemoveRemote(repo.Path, "fork-remote"); err != nil {
			t.Fatalf("RemoveRemote failed: %v", err)
		}

		// Verify it's gone
		exists, err := RemoteExists(repo.Path, "fork-remote")
		if err != nil {
			t.Fatalf("RemoteExists after removal failed: %v", err)
		}
		if exists {
			t.Error("remote should not exist after removal")
		}
	})

	t.Run("fails with empty path", func(t *testing.T) {
		err := RemoveRemote("", "fork-remote")
		if err == nil {
			t.Fatal("expected error for empty path")
		}
	})

	t.Run("fails with empty name", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		err := RemoveRemote(repo.Path, "")
		if err == nil {
			t.Fatal("expected error for empty name")
		}
	})

	t.Run("fails for non-existing remote", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		err := RemoveRemote(repo.Path, "nonexistent-remote")
		if err == nil {
			t.Fatal("expected error for non-existing remote")
		}
	})
}

// TestForkRemoteCleanup tests the pattern used for fork PR cleanup:
// add remote, then remove it if something fails.
func TestForkRemoteCleanup(t *testing.T) {
	t.Run("add and cleanup remote simulates fork PR flow", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		remoteName := "pr-123-contributor"
		remoteURL := "https://github.com/contributor/repo.git"

		// Simulate fork PR: add remote
		if err := AddRemote(repo.Path, remoteName, remoteURL); err != nil {
			t.Fatalf("failed to add fork remote: %v", err)
		}

		// Verify remote was added
		exists, _ := RemoteExists(repo.Path, remoteName)
		if !exists {
			t.Fatal("fork remote should exist")
		}

		// Simulate failure: cleanup by removing remote
		if err := RemoveRemote(repo.Path, remoteName); err != nil {
			t.Fatalf("failed to cleanup fork remote: %v", err)
		}

		// Verify cleanup worked
		exists, _ = RemoteExists(repo.Path, remoteName)
		if exists {
			t.Error("fork remote should be removed after cleanup")
		}
	})

	t.Run("cleanup is idempotent when remote already removed", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Try to remove a remote that doesn't exist
		// This simulates the case where cleanup is called but remote was never added
		err := RemoveRemote(repo.Path, "never-existed")
		// This should fail, but in the actual code we ignore the error
		// The important thing is it doesn't panic
		if err == nil {
			t.Log("RemoveRemote didn't return error for non-existent remote")
		}
	})

	t.Run("remote reuse when already exists", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		remoteName := "pr-456-user"
		remoteURL := "https://github.com/user/repo.git"

		// Add remote first time (simulating first PR checkout)
		if err := AddRemote(repo.Path, remoteName, remoteURL); err != nil {
			t.Fatalf("first AddRemote failed: %v", err)
		}

		// Check if remote exists (simulating second PR checkout)
		exists, err := RemoteExists(repo.Path, remoteName)
		if err != nil {
			t.Fatalf("RemoteExists failed: %v", err)
		}
		if !exists {
			t.Fatal("remote should exist")
		}

		// In the actual code, if remote exists, we skip adding it
		// This test verifies the "remote already exists" detection works
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

func TestWrapGitTooOldError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		result := WrapGitTooOldError(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("wraps relative-paths error", func(t *testing.T) {
		err := errors.New("exit status 129: error: unknown option `relative-paths'")
		result := WrapGitTooOldError(err)
		if !errors.Is(result, ErrGitTooOld) {
			t.Error("expected error to be wrapped with ErrGitTooOld")
		}
	})

	t.Run("passes through unrelated errors", func(t *testing.T) {
		err := errors.New("some other error")
		result := WrapGitTooOldError(err)
		if errors.Is(result, ErrGitTooOld) {
			t.Error("expected error NOT to be wrapped with ErrGitTooOld")
		}
		if !errors.Is(result, err) {
			t.Error("expected original error to be returned unchanged")
		}
	})
}

func TestIsGitTooOld(t *testing.T) {
	t.Run("returns true for ErrGitTooOld", func(t *testing.T) {
		err := WrapGitTooOldError(errors.New("error: unknown option `relative-paths'"))
		if !IsGitTooOld(err) {
			t.Error("expected IsGitTooOld to return true")
		}
	})

	t.Run("returns false for unrelated errors", func(t *testing.T) {
		err := errors.New("some other error")
		if IsGitTooOld(err) {
			t.Error("expected IsGitTooOld to return false")
		}
	})

	t.Run("returns false for nil", func(t *testing.T) {
		if IsGitTooOld(nil) {
			t.Error("expected IsGitTooOld to return false for nil")
		}
	})
}

func TestHintGitTooOld(t *testing.T) {
	t.Run("returns nil for nil error", func(t *testing.T) {
		result := HintGitTooOld(nil)
		if result != nil {
			t.Errorf("expected nil, got %v", result)
		}
	})

	t.Run("returns same error for ErrGitTooOld", func(t *testing.T) {
		err := WrapGitTooOldError(errors.New("error: unknown option `relative-paths'"))
		result := HintGitTooOld(err)
		if !errors.Is(result, err) {
			t.Error("expected same error to be returned")
		}
		if !errors.Is(result, ErrGitTooOld) {
			t.Error("expected error to still contain ErrGitTooOld")
		}
	})

	t.Run("returns same error for unrelated errors", func(t *testing.T) {
		err := errors.New("some other error")
		result := HintGitTooOld(err)
		if !errors.Is(result, err) {
			t.Error("expected same error to be returned")
		}
	})
}

func TestIsRemoteReachable(t *testing.T) {
	t.Parallel()

	t.Run("returns true for reachable local file remote", func(t *testing.T) {
		t.Parallel()
		// Create a bare repo to act as a remote
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

		// Create a local repo with that remote
		localRepo := testgit.NewTestRepo(t)
		cmd = exec.Command("git", "remote", "add", "origin", remoteRepo) //nolint:gosec
		cmd.Dir = localRepo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		if !IsRemoteReachable(localRepo.Path, "origin") {
			t.Error("expected reachable remote to return true")
		}
	})

	t.Run("returns false for unreachable remote", func(t *testing.T) {
		t.Parallel()
		localRepo := testgit.NewTestRepo(t)

		// Add a remote with a non-existent path
		cmd := exec.Command("git", "remote", "add", "origin", "/nonexistent/path/repo.git")
		cmd.Dir = localRepo.Path
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}

		if IsRemoteReachable(localRepo.Path, "origin") {
			t.Error("expected unreachable remote to return false")
		}
	})

	t.Run("returns false for non-existent remote name", func(t *testing.T) {
		t.Parallel()
		localRepo := testgit.NewTestRepo(t)

		if IsRemoteReachable(localRepo.Path, "nonexistent") {
			t.Error("expected non-existent remote to return false")
		}
	})

	t.Run("returns false for empty path", func(t *testing.T) {
		t.Parallel()
		if IsRemoteReachable("", "origin") {
			t.Error("expected empty path to return false")
		}
	})

	t.Run("returns false for empty remote name", func(t *testing.T) {
		t.Parallel()
		localRepo := testgit.NewTestRepo(t)

		if IsRemoteReachable(localRepo.Path, "") {
			t.Error("expected empty remote name to return false")
		}
	})
}

func TestListRemotes(t *testing.T) {
	t.Parallel()

	t.Run("returns empty slice for repo with no remotes", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		remotes, err := ListRemotes(repo.Path)
		if err != nil {
			t.Fatalf("ListRemotes failed: %v", err)
		}
		if len(remotes) != 0 {
			t.Errorf("expected no remotes, got %v", remotes)
		}
	})

	t.Run("returns single remote", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		remotes, err := ListRemotes(repo.Path)
		if err != nil {
			t.Fatalf("ListRemotes failed: %v", err)
		}
		if len(remotes) != 1 {
			t.Fatalf("expected 1 remote, got %d", len(remotes))
		}
		if remotes[0] != "origin" {
			t.Errorf("expected remote 'origin', got %q", remotes[0])
		}
	})

	t.Run("returns multiple remotes", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		cmd := exec.Command("git", "remote", "add", "origin", "https://github.com/test/repo.git")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add origin: %v", err)
		}

		cmd = exec.Command("git", "remote", "add", "upstream", "https://github.com/upstream/repo.git")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to add upstream: %v", err)
		}

		remotes, err := ListRemotes(repo.Path)
		if err != nil {
			t.Fatalf("ListRemotes failed: %v", err)
		}
		if len(remotes) != 2 {
			t.Fatalf("expected 2 remotes, got %d", len(remotes))
		}

		// Check both remotes exist (order may vary)
		hasOrigin := false
		hasUpstream := false
		for _, r := range remotes {
			if r == "origin" {
				hasOrigin = true
			}
			if r == "upstream" {
				hasUpstream = true
			}
		}
		if !hasOrigin || !hasUpstream {
			t.Errorf("expected origin and upstream, got %v", remotes)
		}
	})

	t.Run("returns error for empty path", func(t *testing.T) {
		_, err := ListRemotes("")
		if err == nil {
			t.Error("expected error for empty path")
		}
	})

	t.Run("returns error for non-git directory", func(t *testing.T) {
		tempDir := t.TempDir()
		_, err := ListRemotes(tempDir)
		if err == nil {
			t.Error("expected error for non-git directory")
		}
	})
}

func TestConfigureFetchRefspec(t *testing.T) {
	t.Run("configures fetch refspec for remote", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		if err := AddRemote(repo.Path, "origin", "https://example.com/repo.git"); err != nil {
			t.Fatalf("failed to add remote: %v", err)
		}

		err := ConfigureFetchRefspec(repo.Path, "origin")
		if err != nil {
			t.Fatalf("ConfigureFetchRefspec failed: %v", err)
		}

		cmd, cancel := GitCommand("git", "config", "--get", "remote.origin.fetch")
		defer cancel()
		cmd.Dir = repo.Path
		output, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get config: %v", err)
		}

		expected := "+refs/heads/*:refs/remotes/origin/*"
		if strings.TrimSpace(string(output)) != expected {
			t.Errorf("expected refspec %q, got %q", expected, strings.TrimSpace(string(output)))
		}
	})

	t.Run("returns error for empty repo path", func(t *testing.T) {
		err := ConfigureFetchRefspec("", "origin")
		if err == nil {
			t.Error("expected error for empty repo path")
		}
	})

	t.Run("returns error for empty remote", func(t *testing.T) {
		err := ConfigureFetchRefspec("/tmp/repo", "")
		if err == nil {
			t.Error("expected error for empty remote")
		}
	})
}
