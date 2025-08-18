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
