package git

import (
	"bytes"
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
