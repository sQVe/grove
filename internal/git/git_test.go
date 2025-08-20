package git

import (
	"os"
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
}

func TestGetCurrentBranchDetachedHead(t *testing.T) {
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
}

func TestGetCurrentBranchNoGitRepo(t *testing.T) {
	tempDir := t.TempDir()

	_, err := GetCurrentBranch(tempDir)
	if err == nil {
		t.Fatal("GetCurrentBranch should fail for non-git directory")
	}
}

func TestIsDetachedHead(t *testing.T) {
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

	// Test normal branch
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), fs.FileGit); err != nil {
		t.Fatalf("failed to update HEAD file: %v", err)
	}

	isDetached, err = IsDetachedHead(tempDir)
	if err != nil {
		t.Fatalf("IsDetachedHead failed: %v", err)
	}
	if isDetached {
		t.Error("expected branch HEAD not to be detected as detached")
	}
}

func TestIsDetachedHeadNoGitRepo(t *testing.T) {
	tempDir := t.TempDir()

	_, err := IsDetachedHead(tempDir)
	if err == nil {
		t.Fatal("IsDetachedHead should fail for non-git directory")
	}
}

func TestHasOngoingOperation(t *testing.T) {
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

	mergeHead := filepath.Join(gitDir, "MERGE_HEAD")
	if err := os.WriteFile(mergeHead, []byte("commit-hash"), fs.FileGit); err != nil {
		t.Fatalf("failed to create MERGE_HEAD: %v", err)
	}

	hasOperation, err = HasOngoingOperation(tempDir)
	if err != nil {
		t.Fatalf("HasOngoingOperation failed: %v", err)
	}
	if !hasOperation {
		t.Error("expected merge operation to be detected")
	}

	if err := os.Remove(mergeHead); err != nil {
		t.Fatalf("failed to remove MERGE_HEAD: %v", err)
	}

	rebaseDir := filepath.Join(gitDir, "rebase-merge")
	if err := os.Mkdir(rebaseDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create rebase-merge: %v", err)
	}

	hasOperation, err = HasOngoingOperation(tempDir)
	if err != nil {
		t.Fatalf("HasOngoingOperation failed: %v", err)
	}
	if !hasOperation {
		t.Error("expected rebase operation to be detected")
	}
}

func TestHasOngoingOperationNoGitRepo(t *testing.T) {
	tempDir := t.TempDir()

	_, err := HasOngoingOperation(tempDir)
	if err == nil {
		t.Fatal("HasOngoingOperation should fail for non-git directory")
	}
}
