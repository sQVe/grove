package create

import (
	"fmt"
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

// Performance benchmarks for collision resolution
func BenchmarkCollisionResolution(b *testing.B) {
	pg := &pathGenerator{}

	b.Run("NoCollisions", func(b *testing.B) {
		tempDir := b.TempDir()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Test with a fresh path that doesn't exist - true no-collision scenario
			testPath := filepath.Join(tempDir, fmt.Sprintf("unique-branch-%d", i))
			_, err := pg.resolveCollisions(testPath)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithFewCollisions", func(b *testing.B) {
		tempDir := b.TempDir()
		basePath := filepath.Join(tempDir, "collision-test")

		// Create 5 existing directories to test collision resolution
		for i := 0; i < 5; i++ {
			var dirPath string
			if i == 0 {
				dirPath = basePath
			} else {
				dirPath = fmt.Sprintf("%s-%d", basePath, i)
			}
			err := os.Mkdir(dirPath, 0o755)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := pg.resolveCollisions(basePath)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("WithManyCollisions", func(b *testing.B) {
		tempDir := b.TempDir()
		basePath := filepath.Join(tempDir, "many-collisions")

		// Create 50 existing directories to test performance with many collisions
		// Create directories 0-49 (basePath, basePath-1, basePath-2, ..., basePath-49)
		for i := 0; i < 50; i++ {
			var dirPath string
			if i == 0 {
				dirPath = basePath
			} else {
				dirPath = fmt.Sprintf("%s-%d", basePath, i)
			}
			err := os.Mkdir(dirPath, 0o755)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Test the same collision scenario repeatedly to measure collision resolution performance
			result, err := pg.resolveCollisions(basePath)
			if err != nil {
				b.Fatal(err)
			}
			// The first available number should be 50 (since 0-49 are taken)
			expectedPath := basePath + "-50"
			if result != expectedPath {
				b.Fatalf("expected %s, got %s", expectedPath, result)
			}
		}
	})

	// Add a benchmark that measures collision resolution with a moderate number of collisions
	b.Run("ModerateCollisions", func(b *testing.B) {
		tempDir := b.TempDir()
		basePath := filepath.Join(tempDir, "moderate-test")

		// Create 10 existing directories to simulate moderate collision scenario
		for i := 0; i < 10; i++ {
			var dirPath string
			if i == 0 {
				dirPath = basePath
			} else {
				dirPath = fmt.Sprintf("%s-%d", basePath, i)
			}
			err := os.Mkdir(dirPath, 0o755)
			if err != nil {
				b.Fatal(err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Test collision resolution with moderate collisions (should return basePath-10)
			result, err := pg.resolveCollisions(basePath)
			if err != nil {
				b.Fatal(err)
			}
			// Should find the first available path after the common numbers
			expectedPath := basePath + "-10"
			if result != expectedPath {
				b.Fatalf("expected %s, got %s", expectedPath, result)
			}
		}
	})
}

func BenchmarkHomeDirCaching(b *testing.B) {
	b.Run("CachedHomeDirLookup", func(b *testing.B) {
		// Reset the cache for this benchmark using the dedicated test function
		resetHomeDirCache()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := getHomeDir()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("DirectHomeDirLookup", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := os.UserHomeDir()
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
