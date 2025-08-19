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

	// Check that git bare repository was created
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

	// Make git unavailable by setting empty PATH
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

	// Test with invalid URL to verify error handling works in quiet mode
	err := Clone("file:///nonexistent/repo.git", bareDir, true)
	if err == nil {
		t.Fatal("Expected error for non-existent repo")
	}

	// Error should be captured even in quiet mode
	if err.Error() == "" {
		t.Error("Error message should not be empty in quiet mode")
	}
}

func TestCloneVerboseMode(t *testing.T) {
	tempDir := t.TempDir()
	bareDir := filepath.Join(tempDir, "test.bare")

	// Test with invalid URL to verify error handling works in verbose mode
	err := Clone("file:///nonexistent/repo.git", bareDir, false)
	if err == nil {
		t.Fatal("Expected error for non-existent repo")
	}

	// Error should be captured in verbose mode too
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

	// First call should create cache file (will fail with our test URL, but that's expected)
	_, err := ListRemoteBranches(testURL)
	if err == nil {
		t.Fatal("Expected error for non-existent repo on first call")
	}
}
