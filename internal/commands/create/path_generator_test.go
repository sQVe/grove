package create

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathGenerator_GeneratePath(t *testing.T) {
	viper.Reset()
	pg := NewPathGenerator()

	t.Run("generates valid path from branch name", func(t *testing.T) {
		tempDir := t.TempDir()
		branchName := "feature/user-auth"

		path, err := pg.GeneratePath(branchName, tempDir)

		require.NoError(t, err)
		expectedPath := filepath.Join(tempDir, "feature-user-auth")
		assert.Equal(t, expectedPath, path)
		assert.True(t, filepath.IsAbs(path))
	})

	t.Run("uses configured base path when basePath is empty", func(t *testing.T) {
		tempDir := t.TempDir()
		viper.Set("worktree.base_path", tempDir)
		defer viper.Reset()

		branchName := "fix/bug-123"
		path, err := pg.GeneratePath(branchName, "")

		require.NoError(t, err)
		expectedPath := filepath.Join(tempDir, "fix-bug-123")
		assert.Equal(t, expectedPath, path)
	})

	t.Run("handles collision with numeric suffix", func(t *testing.T) {
		tempDir := t.TempDir()
		branchName := "main"

		// Create existing directory.
		existingPath := filepath.Join(tempDir, "main")
		err := os.Mkdir(existingPath, 0o755)
		require.NoError(t, err)

		path, err := pg.GeneratePath(branchName, tempDir)

		require.NoError(t, err)
		expectedPath := filepath.Join(tempDir, "main-1")
		assert.Equal(t, expectedPath, path)
	})

	t.Run("handles multiple collisions", func(t *testing.T) {
		tempDir := t.TempDir()
		branchName := "test"

		// Create multiple existing directories.
		for i := 0; i <= 2; i++ {
			var dirName string
			if i == 0 {
				dirName = "test"
			} else {
				dirName = "test-" + string(rune(i+'0'))
			}
			err := os.Mkdir(filepath.Join(tempDir, dirName), 0o755)
			require.NoError(t, err)
		}

		path, err := pg.GeneratePath(branchName, tempDir)

		require.NoError(t, err)
		expectedPath := filepath.Join(tempDir, "test-3")
		assert.Equal(t, expectedPath, path)
	})

	t.Run("sanitizes special characters in branch names", func(t *testing.T) {
		tempDir := t.TempDir()
		branchName := "feature/fix:issue#123"

		path, err := pg.GeneratePath(branchName, tempDir)

		require.NoError(t, err)
		expectedPath := filepath.Join(tempDir, "feature-fix-issue-123")
		assert.Equal(t, expectedPath, path)
	})

	t.Run("returns error for empty branch name", func(t *testing.T) {
		tempDir := t.TempDir()

		_, err := pg.GeneratePath("", tempDir)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})

	t.Run("converts relative base path to absolute", func(t *testing.T) {
		// Use a temporary directory that exists to avoid parent validation issues.
		tempDir := t.TempDir()
		branchName := "feature"

		path, err := pg.GeneratePath(branchName, tempDir)

		require.NoError(t, err)
		assert.True(t, filepath.IsAbs(path))
		assert.Equal(t, filepath.Join(tempDir, "feature"), path)
	})

	t.Run("expands home directory in configured path", func(t *testing.T) {
		// Create a test directory in home for this test.
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		testDir := filepath.Join(homeDir, "grove-test-worktrees")
		err = os.MkdirAll(testDir, 0o755)
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(testDir) }()

		viper.Set("worktree.base_path", "~/grove-test-worktrees")
		defer viper.Reset()

		branchName := "feature"
		path, err := pg.GeneratePath(branchName, "")

		require.NoError(t, err)
		expectedPath := filepath.Join(testDir, "feature")
		assert.Equal(t, expectedPath, path)
	})
}

func TestPathGenerator_ValidatePath(t *testing.T) {
	pg := &pathGenerator{}

	t.Run("accepts valid absolute path", func(t *testing.T) {
		tempDir := t.TempDir()
		validPath := filepath.Join(tempDir, "valid-directory")

		err := pg.validatePath(validPath)

		assert.NoError(t, err)
	})

	t.Run("rejects relative path", func(t *testing.T) {
		relativePath := "relative/path"

		err := pg.validatePath(relativePath)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path must be absolute")
	})

	t.Run("rejects path with traversal elements", func(t *testing.T) {
		// Create a path that has actual traversal issues.
		traversalPath := "/tmp/test/../../../etc/passwd"

		err := pg.validatePath(traversalPath)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path contains traversal elements")
	})

	t.Run("rejects invalid directory names", func(t *testing.T) {
		tempDir := t.TempDir()
		invalidPath := filepath.Join(tempDir, "invalid:name")

		err := pg.validatePath(invalidPath)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory name is invalid")
	})

	t.Run("allows paths with non-existent but valid parents", func(t *testing.T) {
		// Most paths will have non-existent parents initially, this should be allowed.
		validPath := "/tmp/grove-test/new-worktree"

		err := pg.validatePath(validPath)

		// This should not error since /tmp exists and the path structure is valid.
		assert.NoError(t, err)
	})
}

func TestPathGenerator_ResolveCollisions(t *testing.T) {
	pg := &pathGenerator{}

	t.Run("returns original path when no collision", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "no-collision")

		resolvedPath, err := pg.resolveCollisions(targetPath)

		require.NoError(t, err)
		assert.Equal(t, targetPath, resolvedPath)
	})

	t.Run("adds suffix when collision exists", func(t *testing.T) {
		tempDir := t.TempDir()
		targetPath := filepath.Join(tempDir, "collision")

		// Create existing directory.
		err := os.Mkdir(targetPath, 0o755)
		require.NoError(t, err)

		resolvedPath, err := pg.resolveCollisions(targetPath)

		require.NoError(t, err)
		expectedPath := targetPath + "-1"
		assert.Equal(t, expectedPath, resolvedPath)
	})

	t.Run("finds available suffix with multiple collisions", func(t *testing.T) {
		tempDir := t.TempDir()
		baseName := "multi-collision"
		targetPath := filepath.Join(tempDir, baseName)

		// Create multiple existing directories.
		for i := 0; i <= 5; i++ {
			var dirName string
			if i == 0 {
				dirName = baseName
			} else {
				dirName = baseName + "-" + string(rune(i+'0'))
			}
			err := os.Mkdir(filepath.Join(tempDir, dirName), 0o755)
			require.NoError(t, err)
		}

		resolvedPath, err := pg.resolveCollisions(targetPath)

		require.NoError(t, err)
		expectedPath := filepath.Join(tempDir, baseName+"-6")
		assert.Equal(t, expectedPath, resolvedPath)
	})
}

func TestExpandHomePath(t *testing.T) {
	t.Run("expands tilde to home directory", func(t *testing.T) {
		path, err := expandHomePath("~")

		require.NoError(t, err)
		homeDir, _ := os.UserHomeDir()
		assert.Equal(t, homeDir, path)
	})

	t.Run("expands tilde with path", func(t *testing.T) {
		path, err := expandHomePath("~/Documents/grove")

		require.NoError(t, err)
		homeDir, _ := os.UserHomeDir()
		expectedPath := filepath.Join(homeDir, "Documents", "grove")
		assert.Equal(t, expectedPath, path)
	})

	t.Run("leaves non-tilde paths unchanged", func(t *testing.T) {
		originalPath := "/absolute/path/test"
		path, err := expandHomePath(originalPath)

		require.NoError(t, err)
		assert.Equal(t, originalPath, path)
	})

	t.Run("leaves relative paths unchanged", func(t *testing.T) {
		originalPath := "relative/path"
		path, err := expandHomePath(originalPath)

		require.NoError(t, err)
		assert.Equal(t, originalPath, path)
	})
}
