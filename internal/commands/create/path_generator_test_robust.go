package create

import (
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

// TestPathGenerator_ValidatePath_Robust demonstrates robust path validation testing
func TestPathGenerator_ValidatePath_Robust(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem().WithIsolatedPath()
	pg := &pathGenerator{config: DefaultPathGeneratorConfig()}

	t.Run("accepts valid absolute path", func(t *testing.T) {
		// Use helper to create a clean temp directory for testing
		validPath := helper.CreateTempDir("valid-directory")

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
		// Create a path that has actual traversal issues
		traversalPath := "/tmp/test/../../../etc/passwd"

		err := pg.validatePath(traversalPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path contains traversal elements")
	})

	t.Run("rejects invalid directory names", func(t *testing.T) {
		// Use helper's temp directory to ensure parent exists
		tempDir := helper.GetTempDir()
		invalidPath := filepath.Join(tempDir, "invalid:name")

		err := pg.validatePath(invalidPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "directory name is invalid")
	})

	t.Run("allows paths with non-existent but valid parents", func(t *testing.T) {
		// Use a unique test path that definitely won't exist but has valid parent
		validPath := helper.GetUniqueTestPath("new-worktree")

		err := pg.validatePath(validPath)

		// This should not error since /tmp exists and the path structure is valid
		assert.NoError(t, err)
	})
}

// TestPathGenerator_GeneratePath_Robust demonstrates robust path generation testing
func TestPathGenerator_GeneratePath_Robust(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem().WithIsolatedPath()
	pg := &pathGenerator{config: DefaultPathGeneratorConfig()}

	t.Run("generates valid path from branch name", func(t *testing.T) {
		branchName := "feature/awesome-feature"
		basePath := helper.GetTempDir()

		path, err := pg.GeneratePath(branchName, basePath)

		assert.NoError(t, err)
		assert.Contains(t, path, "feature-awesome-feature")
		assert.True(t, filepath.IsAbs(path))
	})

	t.Run("handles collision with numeric suffix", func(t *testing.T) {
		branchName := "test-branch"
		basePath := helper.GetTempDir()

		// Create a collision by making a directory with the same name
		expectedPath := filepath.Join(basePath, "test-branch")
		helper.CreateTempDir("test-branch")

		path, err := pg.GeneratePath(branchName, basePath)

		assert.NoError(t, err)
		assert.NotEqual(t, expectedPath, path)
		assert.Contains(t, path, "test-branch-")
	})

	t.Run("returns error for empty branch name", func(t *testing.T) {
		basePath := helper.GetTempDir()

		_, err := pg.GeneratePath("", basePath)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "branch name cannot be empty")
	})
}

// TestPathGenerator_ResolveCollisions_Robust demonstrates robust collision resolution testing
func TestPathGenerator_ResolveCollisions_Robust(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	pg := &pathGenerator{config: DefaultPathGeneratorConfig()}

	t.Run("returns original path when no collision", func(t *testing.T) {
		originalPath := helper.GetUniqueTestPath("no-collision")

		resolvedPath, err := pg.resolveCollisions(originalPath)

		assert.NoError(t, err)
		assert.Equal(t, originalPath, resolvedPath)
	})

	t.Run("adds suffix when collision exists", func(t *testing.T) {
		basePath := helper.GetTempDir()
		originalPath := filepath.Join(basePath, "collision-test")

		// Create the collision
		helper.CreateTempDir("collision-test")

		resolvedPath, err := pg.resolveCollisions(originalPath)

		assert.NoError(t, err)
		assert.NotEqual(t, originalPath, resolvedPath)
		assert.Contains(t, resolvedPath, "collision-test-")
	})

	t.Run("finds available suffix with multiple collisions", func(t *testing.T) {
		basePath := helper.GetTempDir()
		originalPath := filepath.Join(basePath, "multi-collision")

		// Create multiple collisions
		helper.CreateTempDir("multi-collision")
		helper.CreateTempDir("multi-collision-1")
		helper.CreateTempDir("multi-collision-2")

		resolvedPath, err := pg.resolveCollisions(originalPath)

		assert.NoError(t, err)
		assert.NotEqual(t, originalPath, resolvedPath)
		assert.Contains(t, resolvedPath, "multi-collision-3")
	})
}
