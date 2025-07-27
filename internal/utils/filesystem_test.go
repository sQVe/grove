//go:build !integration
// +build !integration

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithDirectoryChange(t *testing.T) {
	originalDir, err := os.Getwd()
	require.NoError(t, err)

	t.Run("successful directory change and restore", func(t *testing.T) {
		tempDir := t.TempDir()

		executed := false
		err := WithDirectoryChange(tempDir, func() error {
			executed = true
			currentDir, err := os.Getwd()
			assert.NoError(t, err)
			assert.Equal(t, tempDir, currentDir)
			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed)

		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalDir, currentDir)
	})

	t.Run("function returns error", func(t *testing.T) {
		tempDir := t.TempDir()
		expectedErr := fmt.Errorf("operation failed")

		executed := false
		err := WithDirectoryChange(tempDir, func() error {
			executed = true
			return expectedErr
		})

		require.Error(t, err)
		assert.Equal(t, expectedErr, err)
		assert.True(t, executed)

		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalDir, currentDir)
	})

	t.Run("target directory does not exist", func(t *testing.T) {
		nonExistentDir := "/this/directory/does/not/exist"

		executed := false
		err := WithDirectoryChange(nonExistentDir, func() error {
			executed = true
			return nil
		})

		require.Error(t, err)
		assert.False(t, executed)
		assert.Contains(t, err.Error(), "failed to change to directory")

		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalDir, currentDir)
	})

	t.Run("target directory is not a directory", func(t *testing.T) {
		tempFile := filepath.Join(t.TempDir(), "file.txt")
		err := os.WriteFile(tempFile, []byte("test"), 0o644)
		require.NoError(t, err)

		executed := false
		err = WithDirectoryChange(tempFile, func() error {
			executed = true
			return nil
		})

		require.Error(t, err)
		assert.False(t, executed)
		assert.Contains(t, err.Error(), "failed to change to directory")
	})

	t.Run("empty target directory", func(t *testing.T) {
		executed := false
		err := WithDirectoryChange("", func() error {
			executed = true
			return nil
		})

		require.Error(t, err)
		assert.False(t, executed)
		assert.Contains(t, err.Error(), "failed to change to directory")
	})

	t.Run("relative path target", func(t *testing.T) {
		tempDir := t.TempDir()

		err := os.Chdir(tempDir)
		require.NoError(t, err)
		defer func() {
			_ = os.Chdir(originalDir)
		}()

		subdir := "subdir"
		err = os.Mkdir(subdir, 0o755)
		require.NoError(t, err)

		executed := false
		err = WithDirectoryChange(subdir, func() error {
			executed = true
			currentDir, err := os.Getwd()
			assert.NoError(t, err)
			assert.Equal(t, filepath.Join(tempDir, subdir), currentDir)
			return nil
		})

		require.NoError(t, err)
		assert.True(t, executed)

		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, tempDir, currentDir)
	})

	t.Run("nested directory changes", func(t *testing.T) {
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()

		executed := false
		err := WithDirectoryChange(tempDir1, func() error {
			return WithDirectoryChange(tempDir2, func() error {
				executed = true
				currentDir, err := os.Getwd()
				assert.NoError(t, err)
				assert.Equal(t, tempDir2, currentDir)
				return nil
			})
		})

		require.NoError(t, err)
		assert.True(t, executed)

		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalDir, currentDir)
	})

	t.Run("function panics - directory still restored", func(t *testing.T) {
		tempDir := t.TempDir()

		defer func() {
			r := recover()
			assert.Equal(t, "test panic", r)

			// Verify we're back to original directory even after panic.
			currentDir, err := os.Getwd()
			require.NoError(t, err)
			assert.Equal(t, originalDir, currentDir)
		}()

		err := WithDirectoryChange(tempDir, func() error {
			currentDir, err := os.Getwd()
			assert.NoError(t, err)
			assert.Equal(t, tempDir, currentDir)

			panic("test panic")
		})

		// This should not be reached due to panic.
		t.Fail()
		_ = err
	})

	t.Run("permission denied on target directory", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("Test requires non-root user")
		}

		tempDir := t.TempDir()
		restrictedDir := filepath.Join(tempDir, "restricted")
		err := os.Mkdir(restrictedDir, 0o000) // No permissions
		require.NoError(t, err)

		defer func() {
			// Restore permissions for cleanup.
			_ = os.Chmod(restrictedDir, 0o755)
		}()

		executed := false
		err = WithDirectoryChange(restrictedDir, func() error {
			executed = true
			return nil
		})

		require.Error(t, err)
		assert.False(t, executed)
		assert.Contains(t, err.Error(), "failed to change to directory")

		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalDir, currentDir)
	})

	t.Run("concurrent directory changes", func(t *testing.T) {
		tempDir1 := t.TempDir()
		tempDir2 := t.TempDir()

		done1 := make(chan bool)
		done2 := make(chan bool)

		go func() {
			defer close(done1)
			err := WithDirectoryChange(tempDir1, func() error {
				currentDir, err := os.Getwd()
				assert.NoError(t, err)
				assert.Equal(t, tempDir1, currentDir)
				return nil
			})
			assert.NoError(t, err)
		}()

		go func() {
			defer close(done2)
			err := WithDirectoryChange(tempDir2, func() error {
				currentDir, err := os.Getwd()
				assert.NoError(t, err)
				assert.Equal(t, tempDir2, currentDir)
				return nil
			})
			assert.NoError(t, err)
		}()

		<-done1
		<-done2

		currentDir, err := os.Getwd()
		require.NoError(t, err)
		assert.Equal(t, originalDir, currentDir)
	})
}
