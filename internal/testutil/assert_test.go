package testutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAssertErrorContains(t *testing.T) {
	t.Run("passes when error contains substring", func(t *testing.T) {
		err := errors.New("file not found: test.txt")

		// Should not panic or fail - this is a passing assertion
		AssertErrorContains(t, err, "not found")
	})

	t.Run("passes with exact match", func(t *testing.T) {
		err := errors.New("exact error message")

		AssertErrorContains(t, err, "exact error message")
	})

	t.Run("passes with partial substring", func(t *testing.T) {
		err := errors.New("connection refused: server unavailable")

		AssertErrorContains(t, err, "refused")
	})
}

func TestAssertFileExists(t *testing.T) {
	t.Run("passes when file exists", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "exists.txt")
		if err := os.WriteFile(path, []byte("content"), 0o600); err != nil {
			t.Fatal(err)
		}

		AssertFileExists(t, path)
	})

	t.Run("passes when directory exists", func(t *testing.T) {
		tempDir := t.TempDir()

		AssertFileExists(t, tempDir)
	})
}

func TestAssertFileContent(t *testing.T) {
	t.Run("passes when content matches", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "test.txt")
		if err := os.WriteFile(path, []byte("expected content"), 0o600); err != nil {
			t.Fatal(err)
		}

		AssertFileContent(t, path, "expected content")
	})

	t.Run("passes with empty content", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "empty.txt")
		if err := os.WriteFile(path, []byte(""), 0o600); err != nil {
			t.Fatal(err)
		}

		AssertFileContent(t, path, "")
	})

	t.Run("passes with multiline content", func(t *testing.T) {
		tempDir := t.TempDir()
		path := filepath.Join(tempDir, "multiline.txt")
		content := "line1\nline2\nline3"
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}

		AssertFileContent(t, path, content)
	})
}

func TestAssertContains(t *testing.T) {
	t.Run("passes when slice contains value", func(t *testing.T) {
		slice := []string{"apple", "banana", "cherry"}

		AssertContains(t, slice, "banana")
	})

	t.Run("passes with first element", func(t *testing.T) {
		slice := []string{"first", "second", "third"}

		AssertContains(t, slice, "first")
	})

	t.Run("passes with last element", func(t *testing.T) {
		slice := []string{"first", "second", "third"}

		AssertContains(t, slice, "third")
	})

	t.Run("passes with single element slice", func(t *testing.T) {
		slice := []string{"only"}

		AssertContains(t, slice, "only")
	})
}
