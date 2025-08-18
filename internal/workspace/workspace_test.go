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

	// Check that .bare directory was created
	bareDir := filepath.Join(tempDir, ".bare")
	if _, err := os.Stat(bareDir); os.IsNotExist(err) {
		t.Error(".bare directory should be created")
	}

	// Check that .git file was created
	gitFile := filepath.Join(tempDir, ".git")
	if _, err := os.Stat(gitFile); os.IsNotExist(err) {
		t.Error(".git file should be created")
	}

	// Check .git file content
	content, err := os.ReadFile(gitFile) //nolint:gosec // Reading controlled test file
	if err != nil {
		t.Fatalf("failed to read .git file: %v", err)
	}
	expected := "gitdir: .bare"
	if string(content) != expected {
		t.Errorf(".git file should contain '%s', got '%s'", expected, string(content))
	}

	// Check that bare repository was initialized
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
