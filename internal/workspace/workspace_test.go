package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

const errInsideGitRepo = "cannot initialize grove inside existing git repository"

func TestInitialize(t *testing.T) {
	t.Run("succeeds on empty directory", func(t *testing.T) {
		tempDir := t.TempDir()

		if err := Initialize(tempDir); err != nil {
			t.Fatalf("Initialize should succeed on empty directory: %v", err)
		}

		bareDir := filepath.Join(tempDir, ".bare")
		if _, err := os.Stat(bareDir); os.IsNotExist(err) {
			t.Error(".bare directory should be created")
		}

		gitFile := filepath.Join(tempDir, ".git")
		if _, err := os.Stat(gitFile); os.IsNotExist(err) {
			t.Error(".git file should be created")
		}

		content, err := os.ReadFile(gitFile) // nolint:gosec // Reading controlled test file
		if err != nil {
			t.Fatalf("failed to read .git file: %v", err)
		}
		expected := groveGitContent
		if string(content) != expected {
			t.Errorf(".git file should contain '%s', got '%s'", expected, string(content))
		}

		if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); os.IsNotExist(err) {
			t.Error("HEAD file should exist in bare repository")
		}
	})

	t.Run("fails on non-empty directory", func(t *testing.T) {
		tempDir := t.TempDir()

		testFile := filepath.Join(tempDir, "existing.txt")
		if err := os.WriteFile(testFile, []byte("content"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		err := Initialize(tempDir)
		if err == nil {
			t.Fatal("Initialize should fail on non-empty directory")
		}

		if !os.IsExist(err) && err.Error() != "directory "+tempDir+" is not empty" {
			t.Errorf("expected 'directory not empty' error, got: %v", err)
		}
	})

	t.Run("cleans up on git failure", func(t *testing.T) {
		tempDir := t.TempDir()

		t.Setenv("PATH", "")

		err := Initialize(tempDir)
		if err == nil {
			t.Fatal("Initialize should fail when git is not available")
		}

		bareDir := filepath.Join(tempDir, ".bare")
		if _, err := os.Stat(bareDir); !os.IsNotExist(err) {
			t.Error(".bare directory should be cleaned up on git init failure")
		}

		gitFile := filepath.Join(tempDir, ".git")
		if _, err := os.Stat(gitFile); !os.IsNotExist(err) {
			t.Error(".git file should not exist when git init fails")
		}
	})

	t.Run("cleans up on git file failure", func(t *testing.T) {
		tempDir := t.TempDir()

		if err := os.Chmod(tempDir, 0o555); err != nil { // nolint:gosec // Test needs read-only permissions
			t.Fatalf("failed to make directory read-only: %v", err)
		}
		defer func() { _ = os.Chmod(tempDir, fs.DirGit) }()

		err := Initialize(tempDir)
		if err == nil {
			t.Fatal("Initialize should fail when .git file cannot be created")
		}

		_ = os.Chmod(tempDir, fs.DirGit)

		bareDir := filepath.Join(tempDir, ".bare")
		if _, err := os.Stat(bareDir); !os.IsNotExist(err) {
			t.Error(".bare directory should be cleaned up on .git file creation failure")
		}
	})

	t.Run("no cleanup on existing directory", func(t *testing.T) {
		tempDir := t.TempDir()

		existingDir := filepath.Join(tempDir, "existing")
		if err := os.Mkdir(existingDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create existing directory: %v", err)
		}

		existingFile := filepath.Join(existingDir, "important.txt")
		if err := os.WriteFile(existingFile, []byte("important data"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create existing file: %v", err)
		}

		t.Setenv("PATH", "") // Make git unavailable to force failure
		err := Initialize(existingDir)
		if err == nil {
			t.Fatal("Initialize should fail on non-empty directory")
		}

		if _, err := os.Stat(existingDir); os.IsNotExist(err) {
			t.Error("existing directory should not be removed on failure")
		}

		if _, err := os.Stat(existingFile); os.IsNotExist(err) {
			t.Error("existing file should not be removed on failure")
		}
	})

	t.Run("detects existing git directory", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		err := Initialize(tempDir)
		if err == nil {
			t.Fatal("Initialize should fail when git repository already exists")
		}

		if !os.IsExist(err) && err.Error() != errInsideGitRepo {
			t.Errorf("expected 'inside existing git repository' error, got: %v", err)
		}
	})

	t.Run("detects existing git file", func(t *testing.T) {
		tempDir := t.TempDir()

		mainRepo := filepath.Join(tempDir, "main")
		if err := os.Mkdir(mainRepo, fs.DirGit); err != nil {
			t.Fatalf("failed to create main repo directory: %v", err)
		}

		cmd := exec.Command("git", "init")
		cmd.Dir = mainRepo
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize main git repository: %v", err)
		}

		worktreeDir := filepath.Join(tempDir, "worktree")
		if err := os.Mkdir(worktreeDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create worktree directory: %v", err)
		}

		cmd = exec.Command("git", "worktree", "add", "../worktree")
		cmd.Dir = mainRepo
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create git worktree: %v", err)
		}

		err := Initialize(worktreeDir)
		if err == nil {
			t.Fatal("Initialize should fail when git worktree already exists")
		}

		if !os.IsExist(err) && err.Error() != errInsideGitRepo {
			t.Errorf("expected 'inside existing git repository' error, got: %v", err)
		}
	})

	t.Run("fails inside git repository", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		subDir := filepath.Join(tempDir, "subproject")
		if err := os.Mkdir(subDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		err := Initialize(subDir)
		if err == nil {
			t.Fatal("Initialize should fail when inside an existing git repository")
		}

		if !os.IsExist(err) && err.Error() != errInsideGitRepo {
			t.Errorf("expected 'inside existing git repository' error, got: %v", err)
		}
	})
}

func TestIsInsideGroveWorkspace(t *testing.T) {
	t.Run("returns false for empty directory", func(t *testing.T) {
		tempDir := t.TempDir()

		if IsInsideGroveWorkspace(tempDir) {
			t.Error("empty directory should not be inside grove workspace")
		}
	})

	t.Run("returns true with bare directory", func(t *testing.T) {
		tempDir := t.TempDir()

		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create .bare directory: %v", err)
		}

		if !IsInsideGroveWorkspace(tempDir) {
			t.Error("directory with .bare should be inside grove workspace")
		}
	})

	t.Run("returns true with git file", func(t *testing.T) {
		tempDir := t.TempDir()

		gitFile := filepath.Join(tempDir, ".git")
		if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		if !IsInsideGroveWorkspace(tempDir) {
			t.Error("directory with grove .git file should be inside grove workspace")
		}
	})

	t.Run("returns true for nested directory", func(t *testing.T) {
		tempDir := t.TempDir()

		bareDir := filepath.Join(tempDir, ".bare")
		if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create .bare directory: %v", err)
		}

		subDir := filepath.Join(tempDir, "subdir", "nested")
		if err := os.MkdirAll(subDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create nested directory: %v", err)
		}

		if !IsInsideGroveWorkspace(subDir) {
			t.Error("nested directory should be inside grove workspace")
		}
	})

	t.Run("returns false for invalid path", func(t *testing.T) {
		if IsInsideGroveWorkspace("/nonexistent/path") {
			t.Error("nonexistent path should not be inside grove workspace")
		}
	})
}

func TestCloneAndInitialize(t *testing.T) {
	t.Run("fails with branches for non-existent repo", func(t *testing.T) {
		tempDir := t.TempDir()

		err := CloneAndInitialize("file:///test/repo.git", tempDir, "main,develop", false)
		if err == nil {
			t.Fatal("Expected error for non-existent repo")
		}
	})

	t.Run("fails with empty branches for non-existent repo", func(t *testing.T) {
		tempDir := t.TempDir()

		err := CloneAndInitialize("file:///test/repo.git", tempDir, "", false)
		if err == nil {
			t.Fatal("Expected error for non-existent repo")
		}
	})

	t.Run("fails with invalid branches for non-existent repo", func(t *testing.T) {
		tempDir := t.TempDir()

		err := CloneAndInitialize("file:///test/repo.git", tempDir, "nonexistent", false)
		if err == nil {
			t.Fatal("Expected error for invalid branch")
		}
	})

	t.Run("fails in quiet mode for non-existent repo", func(t *testing.T) {
		tempDir := t.TempDir()

		err := CloneAndInitialize("file:///test/repo.git", tempDir, "main", false)
		if err == nil {
			t.Fatal("Expected error for non-existent repo")
		}
	})

	t.Run("fails in verbose mode for non-existent repo", func(t *testing.T) {
		tempDir := t.TempDir()

		err := CloneAndInitialize("file:///test/repo.git", tempDir, "main", true)
		if err == nil {
			t.Fatal("Expected error for non-existent repo")
		}
	})
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		branch   string
		expected string
	}{
		{"feat/user-auth", "feat-user-auth"},
		{"chore/better-error", "chore-better-error"},
		{"test<pipe>quote\"", "test-pipe-quote-"},
		{"main", "main"},
		{"feat|special", "feat-special"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			if got := sanitizeBranchName(tt.branch); got != tt.expected {
				t.Errorf("sanitizeBranchName(%q) = %q, want %q", tt.branch, got, tt.expected)
			}
		})
	}
}

func TestConvert(t *testing.T) {
	t.Run("fails when not git repository", func(t *testing.T) {
		tempDir := t.TempDir()

		err := Convert(tempDir)
		if err == nil {
			t.Fatal("Convert should fail when not a git repository")
		}

		expected := "not a git repository"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("already grove workspace", func(t *testing.T) {
		t.Skip("Complex test - covered by integration tests")
	})

	t.Run("fails on detached head", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		gitDir := filepath.Join(tempDir, ".git")
		headFile := filepath.Join(gitDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("abc1234567890abcdef1234567890abcdef123456\n"), fs.FileGit); err != nil {
			t.Fatalf("failed to create detached HEAD: %v", err)
		}

		err := Convert(tempDir)
		if err == nil {
			t.Fatal("Convert should fail when repository is in detached HEAD state")
		}

		expected := "cannot convert: repository is in detached HEAD state"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("fails on ongoing merge", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		gitDir := filepath.Join(tempDir, ".git")
		mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
		if err := os.WriteFile(mergeHead, []byte("commit-hash"), fs.FileGit); err != nil {
			t.Fatalf("failed to create MERGE_HEAD: %v", err)
		}

		err := Convert(tempDir)
		if err == nil {
			t.Fatal("Convert should fail when merge is in progress")
		}

		expected := "cannot convert: repository has ongoing merge/rebase/cherry-pick"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("fails on ongoing rebase", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		gitDir := filepath.Join(tempDir, ".git")
		rebaseDir := filepath.Join(gitDir, "rebase-merge")
		if err := os.Mkdir(rebaseDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create rebase-merge: %v", err)
		}

		err := Convert(tempDir)
		if err == nil {
			t.Fatal("Convert should fail when rebase is in progress")
		}

		expected := "cannot convert: repository has ongoing merge/rebase/cherry-pick"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("fails when already a worktree", func(t *testing.T) {
		tempDir := t.TempDir()

		mainRepo := filepath.Join(tempDir, "main")
		if err := os.Mkdir(mainRepo, fs.DirGit); err != nil {
			t.Fatalf("failed to create main repo directory: %v", err)
		}

		cmd := exec.Command("git", "init")
		cmd.Dir = mainRepo
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize main git repository: %v", err)
		}

		worktreeDir := filepath.Join(tempDir, "worktree")
		if err := os.Mkdir(worktreeDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create worktree directory: %v", err)
		}

		cmd = exec.Command("git", "worktree", "add", "../worktree")
		cmd.Dir = mainRepo
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create git worktree: %v", err)
		}

		err := Convert(worktreeDir)
		if err == nil {
			t.Fatal("Convert should fail when repository is already a worktree")
		}

		expected := "cannot convert: repository is already a worktree"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("fails with uncommitted changes", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("initial content"), fs.FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		err := Convert(tempDir)
		if err == nil {
			t.Fatal("Convert should fail when repository has uncommitted changes")
		}

		expected := "cannot convert: repository has uncommitted changes"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("fails with existing worktrees", func(t *testing.T) {
		tempDir := t.TempDir()

		cmd := exec.Command("git", "init")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to initialize git repository: %v", err)
		}

		worktreeDir := filepath.Join(tempDir, "worktree")
		if err := os.Mkdir(worktreeDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create worktree directory: %v", err)
		}

		cmd = exec.Command("git", "worktree", "add", "worktree")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to create git worktree: %v", err)
		}

		err := Convert(tempDir)
		if err == nil {
			t.Fatal("Convert should fail when repository already has worktrees")
		}

		expected := "cannot convert: repository already has worktrees"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})

	t.Run("fails with lock files", func(t *testing.T) {
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

		err := Convert(tempDir)
		if err == nil {
			t.Fatal("Convert should fail when repository has active lock files")
		}

		expected := "cannot convert: repository has active lock files"
		if err.Error() != expected {
			t.Errorf("expected '%s', got '%s'", expected, err.Error())
		}
	})
}
