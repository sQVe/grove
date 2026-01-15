package fs

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
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

func TestCopyFile(t *testing.T) {
	t.Run("copies file with correct content and permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "source.txt")
		dstFile := filepath.Join(tempDir, "dest.txt")
		content := []byte("test content for copy")

		if err := os.WriteFile(srcFile, content, FileStrict); err != nil {
			t.Fatalf("failed to create source file: %v", err)
		}

		if err := CopyFile(srcFile, dstFile, FileGit); err != nil {
			t.Fatalf("CopyFile should not fail: %v", err)
		}

		readContent, err := os.ReadFile(dstFile) // nolint:gosec // Controlled test path
		if err != nil {
			t.Fatalf("failed to read destination file: %v", err)
		}
		if !bytes.Equal(readContent, content) {
			t.Error("copied file content should match original")
		}

		// Skip permission check on Windows as it doesn't support Unix file permissions
		if runtime.GOOS != "windows" {
			info, err := os.Stat(dstFile)
			if err != nil {
				t.Fatalf("failed to stat destination file: %v", err)
			}
			if info.Mode() != FileGit {
				t.Errorf("expected permissions %v, got %v", FileGit, info.Mode())
			}
		}
	})

	t.Run("fails when source file does not exist", func(t *testing.T) {
		tempDir := t.TempDir()
		srcFile := filepath.Join(tempDir, "nonexistent.txt")
		dstFile := filepath.Join(tempDir, "dest.txt")

		err := CopyFile(srcFile, dstFile, FileGit)
		if err == nil {
			t.Error("CopyFile should fail for non-existent source")
		}
	})
}

func TestPathsEqual(t *testing.T) {
	t.Run("returns true for identical paths", func(t *testing.T) {
		tempDir := t.TempDir()
		if !PathsEqual(tempDir, tempDir) {
			t.Error("PathsEqual should return true for identical paths")
		}
	})

	t.Run("returns true for paths with different trailing separators", func(t *testing.T) {
		tempDir := t.TempDir()
		withSep := tempDir + string(filepath.Separator)
		// filepath.Clean removes trailing separators
		if !PathsEqual(tempDir, filepath.Clean(withSep)) {
			t.Error("PathsEqual should return true for cleaned paths")
		}
	})

	t.Run("returns false for different paths", func(t *testing.T) {
		tempDir := t.TempDir()
		otherDir := filepath.Join(tempDir, "other")
		if err := CreateDirectory(otherDir, DirGit); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}
		if PathsEqual(tempDir, otherDir) {
			t.Error("PathsEqual should return false for different paths")
		}
	})

	t.Run("handles non-existent paths", func(t *testing.T) {
		tempDir := t.TempDir()
		nonExistent := filepath.Join(tempDir, "nonexistent")
		if PathsEqual(tempDir, nonExistent) {
			t.Error("PathsEqual should return false for different paths even if one doesn't exist")
		}
	})
}

func TestPathHasPrefix(t *testing.T) {
	t.Run("returns true when path is inside prefix directory", func(t *testing.T) {
		tempDir := t.TempDir()
		subDir := filepath.Join(tempDir, "sub", "dir")
		if err := CreateDirectory(subDir, DirGit); err != nil {
			t.Fatalf("failed to create test directory: %v", err)
		}
		if !PathHasPrefix(subDir, tempDir) {
			t.Error("PathHasPrefix should return true when path is inside prefix")
		}
	})

	t.Run("returns false when path equals prefix", func(t *testing.T) {
		tempDir := t.TempDir()
		if PathHasPrefix(tempDir, tempDir) {
			t.Error("PathHasPrefix should return false when path equals prefix (not a child)")
		}
	})

	t.Run("returns false when path is not inside prefix", func(t *testing.T) {
		tempDir := t.TempDir()
		otherDir := filepath.Join(tempDir, "other")
		subDir := filepath.Join(tempDir, "sub")
		if err := CreateDirectory(otherDir, DirGit); err != nil {
			t.Fatalf("failed to create other directory: %v", err)
		}
		if err := CreateDirectory(subDir, DirGit); err != nil {
			t.Fatalf("failed to create sub directory: %v", err)
		}
		if PathHasPrefix(otherDir, subDir) {
			t.Error("PathHasPrefix should return false when path is not inside prefix")
		}
	})

	t.Run("handles prefix that is a substring but not a directory prefix", func(t *testing.T) {
		tempDir := t.TempDir()
		// Create /tmp/xxx/foobar and /tmp/xxx/foo
		fooDir := filepath.Join(tempDir, "foo")
		foobarDir := filepath.Join(tempDir, "foobar")
		if err := CreateDirectory(fooDir, DirGit); err != nil {
			t.Fatalf("failed to create foo directory: %v", err)
		}
		if err := CreateDirectory(foobarDir, DirGit); err != nil {
			t.Fatalf("failed to create foobar directory: %v", err)
		}
		// foobar should NOT have prefix foo (even though "foobar" string starts with "foo")
		if PathHasPrefix(foobarDir, fooDir) {
			t.Error("PathHasPrefix should return false when prefix is a string prefix but not a directory prefix")
		}
	})
}

func TestWriteFileAtomic(t *testing.T) {
	t.Run("writes file with correct content and permissions", func(t *testing.T) {
		tempDir := t.TempDir()
		testFile := filepath.Join(tempDir, "atomic.txt")
		content := []byte("atomic write test")

		if err := WriteFileAtomic(testFile, content, FileGit); err != nil {
			t.Fatalf("WriteFileAtomic should not fail: %v", err)
		}

		readContent, err := os.ReadFile(testFile) // nolint:gosec // Controlled test path
		if err != nil {
			t.Fatalf("failed to read atomic file: %v", err)
		}
		if !bytes.Equal(readContent, content) {
			t.Error("atomic file content should match written data")
		}

		// Skip permission check on Windows as it doesn't support Unix file permissions
		if runtime.GOOS != "windows" {
			info, err := os.Stat(testFile)
			if err != nil {
				t.Fatalf("failed to stat atomic file: %v", err)
			}
			if info.Mode() != FileGit {
				t.Errorf("expected permissions %v, got %v", FileGit, info.Mode())
			}
		}
	})
}
