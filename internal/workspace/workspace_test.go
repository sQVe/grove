package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

func TestInitialize(t *testing.T) {
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
	expected := "gitdir: .bare"
	if string(content) != expected {
		t.Errorf(".git file should contain '%s', got '%s'", expected, string(content))
	}

	if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); os.IsNotExist(err) {
		t.Error("HEAD file should exist in bare repository")
	}
}

func TestInitializeNonEmptyDirectory(t *testing.T) {
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
}

func TestInitializeCleanupOnGitFailure(t *testing.T) {
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
}

func TestInitializeCleanupOnGitFileFailure(t *testing.T) {
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
}

func TestInitializeNoCleanupOnExistingDirectory(t *testing.T) {
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
}

func TestInitializeDetectExistingGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	gitDir := filepath.Join(tempDir, ".git")
	if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create .git directory: %v", err)
	}

	err := Initialize(tempDir)
	if err == nil {
		t.Fatal("Initialize should fail when .git directory already exists")
	}

	if !os.IsExist(err) && err.Error() != "directory "+tempDir+" is already a git repository" {
		t.Errorf("expected 'already a git repository' error, got: %v", err)
	}
}

func TestInitializeDetectExistingGitFile(t *testing.T) {
	tempDir := t.TempDir()

	gitFile := filepath.Join(tempDir, ".git")
	if err := os.WriteFile(gitFile, []byte("gitdir: ../main/.git"), fs.FileGit); err != nil {
		t.Fatalf("failed to create .git file: %v", err)
	}

	err := Initialize(tempDir)
	if err == nil {
		t.Fatal("Initialize should fail when .git file already exists")
	}

	if !os.IsExist(err) && err.Error() != "directory "+tempDir+" is already a git repository" {
		t.Errorf("expected 'already a git repository' error, got: %v", err)
	}
}
