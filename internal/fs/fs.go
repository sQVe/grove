package fs

import (
	"errors"
	"io"
	"os"
)

const (
	// Strict permissions (gosec-compliant defaults)
	DirStrict  = 0o750 // rwxr-x--- - gosec-compliant directory
	FileStrict = 0o600 // rw------- - gosec-compliant file

	// Git-compatible permissions (required for git operations)
	DirGit   = 0o755 // rwxr-xr-x - git-compatible directory
	FileExec = 0o755 // rwxr-xr-x - executable file
	FileGit  = 0o644 // rw-r--r-- - git-compatible file
)

// DirectoryExists checks if a directory exists
func DirectoryExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsEmptyDir checks if a directory is empty
func IsEmptyDir(path string) (bool, error) {
	f, err := os.Open(path) // nolint:gosec // Controlled path for workspace validation
	if err != nil {
		return false, err
	}
	defer func() {
		_ = f.Close()
	}()

	_, err = f.Readdirnames(1)
	if err == nil {
		return false, nil
	}

	if errors.Is(err, io.EOF) {
		return true, nil
	}

	return false, err
}

// FileExists checks if path exists and is a file (not a directory)
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// PathExists checks if any path exists (file or directory)
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// IsRegularFile checks if path exists and is a regular file
func IsRegularFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// CreateDirectory creates a directory with the given permissions, including parent directories
func CreateDirectory(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

// RemoveAll removes a path and any children it contains
func RemoveAll(path string) error {
	return os.RemoveAll(path)
}

// RenameWithFallback renames oldpath to newpath
func RenameWithFallback(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
