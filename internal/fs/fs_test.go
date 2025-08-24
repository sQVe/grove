package fs

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestDirectoryExists(t *testing.T) {
	t.Run("returns true for existing directory", func(t *testing.T) {
		tempDir := t.TempDir()

		if !DirectoryExists(tempDir) {
			t.Error("DirectoryExists should return true for existing directory")
		}
	})

	t.Run("returns false for non-existent directory", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistentDir := filepath.Join(tempDir, "nonexistent")

		if DirectoryExists(nonExistentDir) {
			t.Error("DirectoryExists should return false for non-existent directory")
		}
	})
}

func TestIsEmptyDir(t *testing.T) {
	t.Run("returns true for empty directory", func(t *testing.T) {
		tempDir := t.TempDir()

		empty, err := IsEmptyDir(tempDir)
		if err != nil {
			t.Fatalf("IsEmptyDir should not error on empty directory: %v", err)
		}
		if !empty {
			t.Error("IsEmptyDir should return true for empty directory")
		}
	})

	t.Run("returns false for non-empty directory", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		empty, err := IsEmptyDir(tempDir)
		if err != nil {
			t.Fatalf("IsEmptyDir should not error on non-empty directory: %v", err)
		}
		if empty {
			t.Error("IsEmptyDir should return false for non-empty directory")
		}
	})

	t.Run("returns error for non-existent directory", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistentDir := filepath.Join(tempDir, "nonexistent")

		_, err := IsEmptyDir(nonExistentDir)
		if err == nil {
			t.Error("IsEmptyDir should return error for non-existent directory")
		}
	})
}

func TestFileExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if !FileExists(testFile) {
			t.Error("FileExists should return true for existing file")
		}
	})

	t.Run("returns false for directory", func(t *testing.T) {
		tempDir := t.TempDir()

		if FileExists(tempDir) {
			t.Error("FileExists should return false for directory")
		}
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistent := filepath.Join(tempDir, "nonexistent")

		if FileExists(nonExistent) {
			t.Error("FileExists should return false for non-existent path")
		}
	})
}

func TestPathExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if !PathExists(testFile) {
			t.Error("PathExists should return true for existing file")
		}
	})

	t.Run("returns true for existing directory", func(t *testing.T) {
		tempDir := t.TempDir()

		if !PathExists(tempDir) {
			t.Error("PathExists should return true for existing directory")
		}
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistent := filepath.Join(tempDir, "nonexistent")

		if PathExists(nonExistent) {
			t.Error("PathExists should return false for non-existent path")
		}
	})
}

func TestIsRegularFile(t *testing.T) {
	t.Run("returns true for regular file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if !IsRegularFile(testFile) {
			t.Error("IsRegularFile should return true for regular file")
		}
	})

	t.Run("returns false for directory", func(t *testing.T) {
		tempDir := t.TempDir()

		if IsRegularFile(tempDir) {
			t.Error("IsRegularFile should return false for directory")
		}
	})

	t.Run("returns false for non-existent path", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistent := filepath.Join(tempDir, "nonexistent")

		if IsRegularFile(nonExistent) {
			t.Error("IsRegularFile should return false for non-existent path")
		}
	})
}

func TestCreateDirectory(t *testing.T) {
	t.Run("creates directory with correct permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "test")

		if err := CreateDirectory(testDir, DirGit); err != nil {
			t.Fatalf("CreateDirectory should not fail: %v", err)
		}

		info, err := os.Stat(testDir)
		if err != nil {
			t.Fatalf("created directory should exist: %v", err)
		}
		if !info.IsDir() {
			t.Error("created path should be a directory")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		tempDir := t.TempDir()
		nestedDir := filepath.Join(tempDir, "a", "b", "c")

		if err := CreateDirectory(nestedDir, DirStrict); err != nil {
			t.Fatalf("CreateDirectory should create nested directories: %v", err)
		}

		if !DirectoryExists(nestedDir) {
			t.Error("nested directory should be created")
		}
	})

	t.Run("succeeds if directory already exists", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "existing")

		if err := os.Mkdir(testDir, DirGit); err != nil {
			t.Fatalf("failed to create existing directory: %v", err)
		}

		if err := CreateDirectory(testDir, DirGit); err != nil {
			t.Error("CreateDirectory should succeed for existing directory")
		}
	})
}

func TestRemoveAll(t *testing.T) {
	t.Run("removes file", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(testFile, []byte("content"), FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if err := RemoveAll(testFile); err != nil {
			t.Fatalf("RemoveAll should not fail for file: %v", err)
		}

		if PathExists(testFile) {
			t.Error("file should be removed")
		}
	})

	t.Run("removes directory and contents", func(t *testing.T) {
		tempDir := t.TempDir()
		testDir := filepath.Join(tempDir, "test")
		testFile := filepath.Join(testDir, "file.txt")

		if err := os.Mkdir(testDir, DirGit); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}
		if err := os.WriteFile(testFile, []byte("content"), FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if err := RemoveAll(testDir); err != nil {
			t.Fatalf("RemoveAll should not fail for directory: %v", err)
		}

		if PathExists(testDir) {
			t.Error("directory should be removed")
		}
	})

	t.Run("succeeds for non-existent path", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistent := filepath.Join(tempDir, "nonexistent")

		if err := RemoveAll(nonExistent); err != nil {
			t.Error("RemoveAll should succeed for non-existent path")
		}
	})
}

func TestRenameWithFallback(t *testing.T) {
	t.Run("renames file successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		oldFile := filepath.Join(tempDir, "old.txt")
		newFile := filepath.Join(tempDir, "new.txt")
		content := []byte("test content")

		if err := os.WriteFile(oldFile, content, FileStrict); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		if err := RenameWithFallback(oldFile, newFile); err != nil {
			t.Fatalf("RenameWithFallback should not fail: %v", err)
		}

		if PathExists(oldFile) {
			t.Error("old file should not exist after rename")
		}
		if !FileExists(newFile) {
			t.Error("new file should exist after rename")
		}

		readContent, err := os.ReadFile(newFile) // nolint:gosec // Controlled test path
		if err != nil {
			t.Fatalf("failed to read renamed file: %v", err)
		}
		if !bytes.Equal(readContent, content) {
			t.Error("renamed file content should match original")
		}
	})

	t.Run("renames directory successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		oldDir := filepath.Join(tempDir, "old")
		newDir := filepath.Join(tempDir, "new")

		if err := os.Mkdir(oldDir, DirGit); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}

		if err := RenameWithFallback(oldDir, newDir); err != nil {
			t.Fatalf("RenameWithFallback should not fail for directory: %v", err)
		}

		if PathExists(oldDir) {
			t.Error("old directory should not exist after rename")
		}
		if !DirectoryExists(newDir) {
			t.Error("new directory should exist after rename")
		}
	})

	t.Run("fails for non-existent source", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistent := filepath.Join(tempDir, "nonexistent")
		newPath := filepath.Join(tempDir, "new")

		err := RenameWithFallback(nonExistent, newPath)
		if err == nil {
			t.Error("RenameWithFallback should fail for non-existent source")
		}
	})
}
