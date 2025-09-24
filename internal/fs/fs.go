package fs

import (
	"errors"
	"fmt"
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

// RenameWithFallback renames oldpath to newpath, falling back to copy+delete for cross-filesystem moves
func RenameWithFallback(oldpath, newpath string) error {
	// Try direct rename first (fastest, preserves everything)
	if err := os.Rename(oldpath, newpath); err == nil {
		return nil
	}

	// Fallback to copy+delete for cross-filesystem moves
	srcInfo, err := os.Lstat(oldpath)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}

	// Handle different file types
	switch {
	case srcInfo.Mode()&os.ModeSymlink != 0:
		// Symlink - recreate the link
		linkTarget, err := os.Readlink(oldpath)
		if err != nil {
			return fmt.Errorf("failed to read symlink: %w", err)
		}
		if err := os.Symlink(linkTarget, newpath); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
	case srcInfo.IsDir():
		return fmt.Errorf("cross-filesystem directory moves not supported")
	default:
		// Regular file - copy with permission preservation
		if err := CopyFile(oldpath, newpath, srcInfo.Mode()); err != nil {
			return fmt.Errorf("failed to copy file: %w", err)
		}
	}

	// Remove source only after successful copy/symlink creation
	if err := os.Remove(oldpath); err != nil {
		// Cleanup the destination if we can't remove source
		_ = os.Remove(newpath)
		return fmt.Errorf("failed to remove source after copy: %w", err)
	}

	return nil
}

// CopyFile copies content from src to dst using streaming I/O and sets given permissions
func CopyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src) // nolint:gosec // Controlled path from git ignored files
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm) // nolint:gosec // Controlled path for worktree files
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// WriteFileAtomic writes data to a file atomically by writing to a temp file then renaming
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmpPath := fmt.Sprintf("%s.tmp.%d", path, os.Getpid())
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return nil
}
