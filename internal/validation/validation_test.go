package validation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

func TestDirectoryExists(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentDir := filepath.Join(tempDir, "nonexistent")

	if !DirectoryExists(tempDir) {
		t.Error("DirectoryExists should return true for existing directory")
	}

	if DirectoryExists(nonExistentDir) {
		t.Error("DirectoryExists should return false for non-existent directory")
	}
}

func TestIsEmptyDir_EmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	empty, err := IsEmptyDir(tempDir)
	if err != nil {
		t.Fatalf("IsEmptyDir should not error on empty directory: %v", err)
	}
	if !empty {
		t.Error("IsEmptyDir should return true for empty directory")
	}
}

func TestIsEmptyDir_NonEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("content"), fs.FileStrict); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	empty, err := IsEmptyDir(tempDir)
	if err != nil {
		t.Fatalf("IsEmptyDir should not error on non-empty directory: %v", err)
	}
	if empty {
		t.Error("IsEmptyDir should return false for non-empty directory")
	}
}

func TestIsEmptyDir_NonExistentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	nonExistentDir := filepath.Join(tempDir, "nonexistent")

	_, err := IsEmptyDir(nonExistentDir)
	if err == nil {
		t.Error("IsEmptyDir should return error for non-existent directory")
	}
}

func TestIsGitRepository(t *testing.T) {
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")

	if IsGitRepository(tempDir) {
		t.Error("IsGitRepository should return false for non-git directory")
	}

	if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}

	if !IsGitRepository(tempDir) {
		t.Error("IsGitRepository should return true for git repository")
	}

	if err := os.Remove(gitDir); err != nil {
		t.Fatalf("failed to remove .git directory: %v", err)
	}

	gitFile := filepath.Join(tempDir, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: /some/path"), fs.FileGit); err != nil {
		t.Fatalf("failed to create .git file: %v", err)
	}

	if !IsGitRepository(tempDir) {
		t.Error("IsGitRepository should return true for worktree (.git file)")
	}
}

func TestIsGitRepositoryNonExistent(t *testing.T) {
	nonExistentPath := "/nonexistent/path"

	if IsGitRepository(nonExistentPath) {
		t.Error("IsGitRepository should return false for nonexistent path")
	}
}

func TestIsGroveWorkspaceAdditional(t *testing.T) {
	tempDir := t.TempDir()

	if IsGroveWorkspace(tempDir) {
		t.Error("IsGroveWorkspace should return false for non-grove directory")
	}

	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create .bare directory: %v", err)
	}

	if !IsGroveWorkspace(tempDir) {
		t.Error("IsGroveWorkspace should return true for grove workspace with .bare directory")
	}

	if err := os.Remove(bareDir); err != nil {
		t.Fatalf("failed to remove .bare directory: %v", err)
	}

	gitFile := filepath.Join(tempDir, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: .bare"), fs.FileGit); err != nil {
		t.Fatalf("failed to create .git file: %v", err)
	}

	if !IsGroveWorkspace(tempDir) {
		t.Error("IsGroveWorkspace should return true for grove workspace with .git file")
	}
}

func TestIsGroveWorkspaceNonExistent(t *testing.T) {
	nonExistentPath := "/nonexistent/path"

	if IsGroveWorkspace(nonExistentPath) {
		t.Error("IsGroveWorkspace should return false for nonexistent path")
	}
}

func TestIsGroveWorkspaceNested(t *testing.T) {
	tempDir := t.TempDir()

	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create .bare directory: %v", err)
	}

	nestedDir := filepath.Join(tempDir, "nested", "deeper")
	if err := os.MkdirAll(nestedDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	if !IsGroveWorkspace(nestedDir) {
		t.Error("IsGroveWorkspace should return true for directory inside grove workspace")
	}
}
