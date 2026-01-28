package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewTestRepo(t *testing.T) {
	t.Run("creates repo with default main branch", func(t *testing.T) {
		repo := NewTestRepo(t)

		// Verify repo was created
		if _, err := os.Stat(repo.Path); err != nil {
			t.Fatalf("repo path should exist: %v", err)
		}

		// Verify it's a git repo
		gitDir := filepath.Join(repo.Path, ".git")
		if _, err := os.Stat(gitDir); err != nil {
			t.Fatalf(".git should exist: %v", err)
		}

		// Verify default branch is main
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get branch: %v", err)
		}
		branch := strings.TrimSpace(string(out))
		if branch != "main" {
			t.Errorf("expected branch 'main', got %q", branch)
		}
	})

	t.Run("creates repo with custom branch", func(t *testing.T) {
		repo := NewTestRepo(t, "develop")

		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get branch: %v", err)
		}
		branch := strings.TrimSpace(string(out))
		if branch != "develop" {
			t.Errorf("expected branch 'develop', got %q", branch)
		}
	})

	t.Run("uses main when empty string provided", func(t *testing.T) {
		repo := NewTestRepo(t, "")

		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get branch: %v", err)
		}
		branch := strings.TrimSpace(string(out))
		if branch != "main" {
			t.Errorf("expected branch 'main' for empty string, got %q", branch)
		}
	})

	t.Run("has initial commit", func(t *testing.T) {
		repo := NewTestRepo(t)

		cmd := exec.Command("git", "log", "--oneline")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get log: %v", err)
		}
		if !strings.Contains(string(out), "initial") {
			t.Error("expected initial commit")
		}
	})

	t.Run("has gpg signing disabled", func(t *testing.T) {
		repo := NewTestRepo(t)

		cmd := exec.Command("git", "config", "commit.gpgsign")
		cmd.Dir = repo.Path
		out, _ := cmd.Output()
		if strings.TrimSpace(string(out)) != "false" {
			t.Error("expected gpgsign to be disabled")
		}
	})
}

func TestTestRepo_Run(t *testing.T) {
	t.Run("returns output and nil error on success", func(t *testing.T) {
		repo := NewTestRepo(t)

		out, err := repo.Run("status")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if out == "" {
			t.Error("expected output")
		}
	})

	t.Run("returns output and error on failure", func(t *testing.T) {
		repo := NewTestRepo(t)

		out, err := repo.Run("checkout", "nonexistent-branch")

		if err == nil {
			t.Error("expected error for nonexistent branch")
		}
		if out == "" {
			t.Error("expected error output")
		}
	})
}

func TestTestRepo_RunOutput(t *testing.T) {
	t.Run("returns output on success", func(t *testing.T) {
		repo := NewTestRepo(t)

		out := repo.RunOutput("rev-parse", "--abbrev-ref", "HEAD")

		if strings.TrimSpace(out) != "main" {
			t.Errorf("expected 'main', got %q", strings.TrimSpace(out))
		}
	})
}

func TestTestRepo_MustFail(t *testing.T) {
	t.Run("passes when command fails", func(t *testing.T) {
		repo := NewTestRepo(t)

		// This should not panic or fail the test
		repo.MustFail("checkout", "definitely-nonexistent-branch")
	})
}

func TestTestRepo_AssertBranchExists(t *testing.T) {
	t.Run("passes when branch exists", func(t *testing.T) {
		repo := NewTestRepo(t)

		// Should not fail
		repo.AssertBranchExists("main")
	})
}

func TestTestRepo_AssertOnBranch(t *testing.T) {
	t.Run("passes when on expected branch", func(t *testing.T) {
		repo := NewTestRepo(t)

		// Should not fail
		repo.AssertOnBranch("main")
	})
}

func TestTestRepo_AssertClean(t *testing.T) {
	t.Run("passes when working tree is clean", func(t *testing.T) {
		repo := NewTestRepo(t)

		// Should not fail
		repo.AssertClean()
	})
}

func TestTestRepo_CreateBranch(t *testing.T) {
	t.Run("creates new branch", func(t *testing.T) {
		repo := NewTestRepo(t)

		repo.CreateBranch("feature")

		// Verify branch exists
		cmd := exec.Command("git", "branch", "--list", "feature")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to list branches: %v", err)
		}
		if !strings.Contains(string(out), "feature") {
			t.Error("expected feature branch to exist")
		}
	})
}

func TestTestRepo_Checkout(t *testing.T) {
	t.Run("switches to branch", func(t *testing.T) {
		repo := NewTestRepo(t)
		repo.CreateBranch("feature")

		repo.Checkout("feature")

		repo.AssertOnBranch("feature")
	})
}

func TestTestRepo_WriteFile(t *testing.T) {
	t.Run("writes file in repo", func(t *testing.T) {
		repo := NewTestRepo(t)

		repo.WriteFile("test.txt", "content")

		content, err := os.ReadFile(filepath.Join(repo.Path, "test.txt"))
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "content" {
			t.Errorf("expected 'content', got %q", string(content))
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		repo := NewTestRepo(t)

		repo.WriteFile("nested/deep/file.txt", "nested content")

		nestedPath := filepath.Join(repo.Path, "nested", "deep", "file.txt")
		content, err := os.ReadFile(nestedPath) // nolint:gosec // Test-controlled path
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}
		if string(content) != "nested content" {
			t.Errorf("expected 'nested content', got %q", string(content))
		}
	})
}

func TestTestRepo_Add(t *testing.T) {
	t.Run("stages file", func(t *testing.T) {
		repo := NewTestRepo(t)
		repo.WriteFile("new.txt", "new content")

		repo.Add("new.txt")

		cmd := exec.Command("git", "diff", "--cached", "--name-only")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get staged files: %v", err)
		}
		if !strings.Contains(string(out), "new.txt") {
			t.Error("expected new.txt to be staged")
		}
	})
}

func TestTestRepo_Commit(t *testing.T) {
	t.Run("creates commit with message", func(t *testing.T) {
		repo := NewTestRepo(t)
		repo.WriteFile("new.txt", "content")
		repo.Add("new.txt")

		repo.Commit("test commit message")

		cmd := exec.Command("git", "log", "-1", "--format=%s")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get commit message: %v", err)
		}
		if strings.TrimSpace(string(out)) != "test commit message" {
			t.Errorf("expected 'test commit message', got %q", strings.TrimSpace(string(out)))
		}
	})
}

func TestNewBareTestRepo(t *testing.T) {
	t.Run("creates bare repository", func(t *testing.T) {
		repo := NewBareTestRepo(t)

		// Verify it's a bare repo
		cmd := exec.Command("git", "rev-parse", "--is-bare-repository")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to check bare status: %v", err)
		}
		if strings.TrimSpace(string(out)) != "true" {
			t.Error("expected bare repository")
		}
	})
}

func TestBareTestRepo_Run(t *testing.T) {
	t.Run("returns output and error", func(t *testing.T) {
		repo := NewBareTestRepo(t)

		out, err := repo.Run("rev-parse", "--is-bare-repository")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if strings.TrimSpace(out) != "true" {
			t.Errorf("expected 'true', got %q", out)
		}
	})
}

func TestNewGroveWorkspace(t *testing.T) {
	t.Run("creates workspace with default main branch", func(t *testing.T) {
		ws := NewGroveWorkspace(t)

		// Verify bare dir exists
		if _, err := os.Stat(ws.BareDir); err != nil {
			t.Fatalf("bare dir should exist: %v", err)
		}

		// Verify main worktree exists
		mainPath, ok := ws.Worktrees["main"]
		if !ok {
			t.Fatal("expected main worktree")
		}
		if _, err := os.Stat(mainPath); err != nil {
			t.Fatalf("main worktree should exist: %v", err)
		}
	})

	t.Run("creates workspace with custom branches", func(t *testing.T) {
		ws := NewGroveWorkspace(t, "develop", "feature")

		if len(ws.Worktrees) != 2 {
			t.Errorf("expected 2 worktrees, got %d", len(ws.Worktrees))
		}
		if _, ok := ws.Worktrees["develop"]; !ok {
			t.Error("expected develop worktree")
		}
		if _, ok := ws.Worktrees["feature"]; !ok {
			t.Error("expected feature worktree")
		}
	})
}

func TestGroveWorkspace_CreateWorktree(t *testing.T) {
	t.Run("creates additional worktree", func(t *testing.T) {
		ws := NewGroveWorkspace(t, "main")

		path := ws.CreateWorktree("feature")

		if _, err := os.Stat(path); err != nil {
			t.Fatalf("worktree should exist: %v", err)
		}

		// Verify branch was created
		cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
		cmd.Dir = path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get branch: %v", err)
		}
		if strings.TrimSpace(string(out)) != "feature" {
			t.Errorf("expected branch 'feature', got %q", strings.TrimSpace(string(out)))
		}
	})
}

func TestGroveWorkspace_WorktreePath(t *testing.T) {
	t.Run("returns path for existing worktree", func(t *testing.T) {
		ws := NewGroveWorkspace(t, "main")

		path := ws.WorktreePath("main")

		if path == "" {
			t.Error("expected non-empty path")
		}
		if _, err := os.Stat(path); err != nil {
			t.Errorf("path should exist: %v", err)
		}
	})
}

func TestGroveWorkspace_Run(t *testing.T) {
	t.Run("runs git command in bare repo", func(t *testing.T) {
		ws := NewGroveWorkspace(t, "main")

		out, err := ws.Run("rev-parse", "--is-bare-repository")
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if strings.TrimSpace(out) != "true" {
			t.Errorf("expected 'true', got %q", out)
		}
	})
}

func TestCleanupWorktree(t *testing.T) {
	t.Run("registers cleanup function", func(t *testing.T) {
		// This test verifies the function can be called without error
		// The actual cleanup happens in t.Cleanup which we can't easily verify
		repo := NewBareTestRepo(t)

		// Create a worktree manually
		worktreePath := filepath.Join(repo.Dir, "wt-test")
		cmd := exec.Command("git", "worktree", "add", worktreePath, "-b", "test-branch") // nolint:gosec // Test-controlled args
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create worktree: %v", err)
		}

		// Register cleanup - this should not error
		CleanupWorktree(t, repo.Path, worktreePath)
	})
}
