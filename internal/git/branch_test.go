package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/fs"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

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

func TestIsUnbornHead(t *testing.T) {
	t.Parallel()

	t.Run("returns error for empty path", func(t *testing.T) {
		t.Parallel()
		_, err := IsUnbornHead("")
		if err == nil {
			t.Error("expected error for empty path")
		}
	})

	t.Run("returns false for repo with commits", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		unborn, err := IsUnbornHead(repo.Path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if unborn {
			t.Error("expected unborn to be false for repo with commits")
		}
	})

	t.Run("returns true for repo without commits", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		repoPath := filepath.Join(dir, "empty-repo")

		if err := os.MkdirAll(repoPath, fs.DirGit); err != nil {
			t.Fatal(err)
		}

		cmd := exec.Command("git", "init", "-b", "main")
		cmd.Dir = repoPath
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to init repo: %v", err)
		}

		unborn, err := IsUnbornHead(repoPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !unborn {
			t.Error("expected unborn to be true for repo without commits")
		}
	})

	t.Run("returns false for detached HEAD", func(t *testing.T) {
		t.Parallel()
		repo := testgit.NewTestRepo(t)

		// Get current commit hash and checkout to detached HEAD
		cmd := exec.Command("git", "rev-parse", "HEAD")
		cmd.Dir = repo.Path
		out, err := cmd.Output()
		if err != nil {
			t.Fatalf("failed to get HEAD: %v", err)
		}
		hash := strings.TrimSpace(string(out))

		cmd = exec.Command("git", "checkout", hash) // nolint:gosec
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to checkout: %v", err)
		}

		unborn, err := IsUnbornHead(repo.Path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if unborn {
			t.Error("expected unborn to be false for detached HEAD")
		}
	})
}
