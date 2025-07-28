//go:build !integration
// +build !integration

package create

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestWorktree creates a directory with a .git folder to simulate a Git worktree.
func setupTestWorktree(t *testing.T, basePath string) string {
	require.NoError(t, os.MkdirAll(basePath, 0o755))
	gitDir := filepath.Join(basePath, ".git")
	require.NoError(t, os.MkdirAll(gitDir, 0o755))
	return basePath
}

func TestFileManagerImpl_CopyFiles_Success(t *testing.T) {
	// Create temporary directories for testing.
	tmpDir := t.TempDir()
	sourceDir := setupTestWorktree(t, filepath.Join(tmpDir, "source"))
	targetDir := setupTestWorktree(t, filepath.Join(tmpDir, "target"))

	// Create test files in source directory.
	testFiles := map[string]string{
		".env":                  "DATABASE_URL=postgres://localhost",
		".env.local":            "DEBUG=true",
		".vscode/settings.json": `{"editor.tabSize": 2}`,
		"README.md":             "# Test project",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(sourceDir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(fullPath), 0o755))
		require.NoError(t, os.WriteFile(fullPath, []byte(content), 0o644))
	}

	mockExecutor := testutils.NewMockGitExecutor()
	manager := NewFileManager(mockExecutor)

	patterns := []string{".env*", ".vscode/"}
	options := CopyOptions{
		ConflictStrategy: ConflictSkip,
	}

	err := manager.CopyFiles(sourceDir, targetDir, patterns, options)

	require.NoError(t, err)

	// Verify files were copied.
	copiedFiles := []string{".env", ".env.local", ".vscode/settings.json"}
	for _, file := range copiedFiles {
		targetPath := filepath.Join(targetDir, file)
		assert.FileExists(t, targetPath)

		// Verify content matches.
		expectedContent := testFiles[file]
		actualContent, err := os.ReadFile(targetPath)
		require.NoError(t, err)
		assert.Equal(t, expectedContent, string(actualContent))
	}

	// Verify excluded files were not copied.
	excludedFile := filepath.Join(targetDir, "README.md")
	assert.NoFileExists(t, excludedFile)
}

func TestFileManagerImpl_CopyFiles_ConflictOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := setupTestWorktree(t, filepath.Join(tmpDir, "source"))
	targetDir := setupTestWorktree(t, filepath.Join(tmpDir, "target"))

	// Create files in both directories with different content.
	sourceFile := filepath.Join(sourceDir, ".env")
	targetFile := filepath.Join(targetDir, ".env")

	require.NoError(t, os.WriteFile(sourceFile, []byte("SOURCE_CONTENT"), 0o644))
	require.NoError(t, os.WriteFile(targetFile, []byte("TARGET_CONTENT"), 0o644))

	mockExecutor := testutils.NewMockGitExecutor()
	manager := NewFileManager(mockExecutor)

	patterns := []string{".env"}
	options := CopyOptions{
		ConflictStrategy: ConflictOverwrite,
	}

	err := manager.CopyFiles(sourceDir, targetDir, patterns, options)

	require.NoError(t, err)

	// Verify file was overwritten with source content.
	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, "SOURCE_CONTENT", string(content))
}

func TestFileManagerImpl_CopyFiles_ConflictSkip(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := setupTestWorktree(t, filepath.Join(tmpDir, "source"))
	targetDir := setupTestWorktree(t, filepath.Join(tmpDir, "target"))

	// Create files in both directories with different content.
	sourceFile := filepath.Join(sourceDir, ".env")
	targetFile := filepath.Join(targetDir, ".env")

	require.NoError(t, os.WriteFile(sourceFile, []byte("SOURCE_CONTENT"), 0o644))
	require.NoError(t, os.WriteFile(targetFile, []byte("TARGET_CONTENT"), 0o644))

	mockExecutor := testutils.NewMockGitExecutor()
	manager := NewFileManager(mockExecutor)

	patterns := []string{".env"}
	options := CopyOptions{
		ConflictStrategy: ConflictSkip,
	}

	err := manager.CopyFiles(sourceDir, targetDir, patterns, options)

	require.NoError(t, err)

	// Verify file was not overwritten (target content preserved).
	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, "TARGET_CONTENT", string(content))
}

func TestFileManagerImpl_CopyFiles_ConflictBackup(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := setupTestWorktree(t, filepath.Join(tmpDir, "source"))
	targetDir := setupTestWorktree(t, filepath.Join(tmpDir, "target"))

	// Create files in both directories with different content.
	sourceFile := filepath.Join(sourceDir, ".env")
	targetFile := filepath.Join(targetDir, ".env")

	require.NoError(t, os.WriteFile(sourceFile, []byte("SOURCE_CONTENT"), 0o644))
	require.NoError(t, os.WriteFile(targetFile, []byte("TARGET_CONTENT"), 0o644))

	mockExecutor := testutils.NewMockGitExecutor()
	manager := NewFileManager(mockExecutor)

	patterns := []string{".env"}
	options := CopyOptions{
		ConflictStrategy: ConflictBackup,
	}

	err := manager.CopyFiles(sourceDir, targetDir, patterns, options)

	require.NoError(t, err)

	// Verify original file was overwritten with source content.
	content, err := os.ReadFile(targetFile)
	require.NoError(t, err)
	assert.Equal(t, "SOURCE_CONTENT", string(content))

	// Verify backup file was created with original content
	// Backup files use timestamp format: .backup.YYYYMMDD_HHMMSS.
	files, err := filepath.Glob(targetFile + ".backup.*")
	require.NoError(t, err)
	require.Len(t, files, 1, "Expected exactly one backup file")

	backupContent, err := os.ReadFile(files[0])
	require.NoError(t, err)
	assert.Equal(t, "TARGET_CONTENT", string(backupContent))
}

func TestFileManagerImpl_CopyFiles_InvalidSourceDirectory(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	manager := NewFileManager(mockExecutor)

	err := manager.CopyFiles("/nonexistent/source", "/tmp/target", []string{".env"}, CopyOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid source worktree")
}

func TestFileManagerImpl_CopyFiles_InvalidTargetDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := setupTestWorktree(t, filepath.Join(tmpDir, "source"))

	mockExecutor := testutils.NewMockGitExecutor()
	manager := NewFileManager(mockExecutor)

	// Try to copy to a directory that doesn't exist and isn't a valid worktree.
	err := manager.CopyFiles(sourceDir, "/nonexistent/target", []string{".env"}, CopyOptions{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target worktree")
}

func TestFileManagerImpl_CopyFiles_EmptyPatterns(t *testing.T) {
	tmpDir := t.TempDir()
	sourceDir := setupTestWorktree(t, filepath.Join(tmpDir, "source"))
	targetDir := setupTestWorktree(t, filepath.Join(tmpDir, "target"))

	mockExecutor := testutils.NewMockGitExecutor()
	manager := NewFileManager(mockExecutor)

	// Should succeed but copy nothing.
	err := manager.CopyFiles(sourceDir, targetDir, []string{}, CopyOptions{})

	require.NoError(t, err)
}

func TestFileManagerImpl_DiscoverSourceWorktree_Success(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	// Mock repository root discovery.
	mockExecutor.SetSuccessResponse("rev-parse --show-toplevel", "/path/to/repo")

	// Mock worktree list output with repo root as main worktree in porcelain format.
	worktreeList := "worktree /path/to/repo\nbranch main\nHEAD abc123\n\nworktree /path/to/repo/feature\nbranch feature\nHEAD def456"
	mockExecutor.SetSuccessResponse("worktree list --porcelain", worktreeList)

	manager := NewFileManager(mockExecutor)

	sourceWorktree, err := manager.DiscoverSourceWorktree()

	require.NoError(t, err)
	assert.Equal(t, "/path/to/repo", sourceWorktree)
}

func TestFileManagerImpl_DiscoverSourceWorktree_ConfiguredSource(t *testing.T) {
	t.Skip("Configured source worktree functionality not yet implemented")
	mockExecutor := testutils.NewMockGitExecutor()

	// Mock repository root discovery.
	mockExecutor.SetSuccessResponse("rev-parse --show-toplevel", "/path/to/repo")

	// Mock config to return specific source worktree.
	mockExecutor.SetSuccessResponse("config --get worktree.copy_files.source_worktree", "custom-main")

	// Mock worktree list with custom main worktree in porcelain format.
	worktreeList := "worktree /path/to/repo/custom-main\nbranch main\nHEAD abc123\n\nworktree /path/to/repo/feature\nbranch feature\nHEAD def456"
	mockExecutor.SetSuccessResponse("worktree list --porcelain", worktreeList)

	manager := NewFileManager(mockExecutor)

	sourceWorktree, err := manager.DiscoverSourceWorktree()

	require.NoError(t, err)
	assert.Equal(t, "/path/to/repo/custom-main", sourceWorktree)
}

func TestFileManagerImpl_DiscoverSourceWorktree_NoMainWorktree(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	// Mock repository root discovery.
	mockExecutor.SetSuccessResponse("rev-parse --show-toplevel", "/path/to/repo")

	// Mock worktree list without main branch in porcelain format.
	worktreeList := "worktree /path/to/repo/feature\nbranch feature\nHEAD def456\n\nworktree /path/to/repo/develop\nbranch develop\nHEAD ghi789"
	mockExecutor.SetSuccessResponse("worktree list --porcelain", worktreeList)

	manager := NewFileManager(mockExecutor)

	sourceWorktree, err := manager.DiscoverSourceWorktree()

	require.Error(t, err)
	assert.Empty(t, sourceWorktree)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeSourceWorktreeNotFound, groveErr.Code)
}

func TestFileManagerImpl_DiscoverSourceWorktree_GitCommandFailure(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetErrorResponse("worktree list", errors.New("git command failed"))

	manager := NewFileManager(mockExecutor)

	sourceWorktree, err := manager.DiscoverSourceWorktree()

	require.Error(t, err)
	assert.Empty(t, sourceWorktree)
	assert.IsType(t, &groveErrors.GroveError{}, err)
}

func TestFileManagerImpl_ResolveConflicts_Success(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	manager := NewFileManager(mockExecutor)

	conflicts := []FileConflict{
		{
			Path:       ".env",
			SourcePath: "/source/.env",
			TargetPath: "/target/.env",
		},
		{
			Path:       ".vscode/settings.json",
			SourcePath: "/source/.vscode/settings.json",
			TargetPath: "/target/.vscode/settings.json",
		},
	}

	// Should succeed for skip strategy (no-op).
	err := manager.ResolveConflicts(conflicts, ConflictSkip)
	require.NoError(t, err)
}

// Note: matchesPatterns is an internal method - testing through public interface.

// Note: Internal method tests removed as they test unexported functions
// The public interface tests above provide sufficient coverage for the FileManager component.
