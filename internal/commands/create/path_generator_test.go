package create

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathGenerator_GeneratePath_Success(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tests := []struct {
		name       string
		branchName string
		basePath   string
		want       func(string) bool
	}{
		{
			name:       "simple branch name",
			branchName: "feature/user-auth",
			basePath:   helper.CreateTempDir("test-base"),
			want: func(result string) bool {
				return strings.Contains(result, "feature-user-auth")
			},
		},
		{
			name:       "branch with special characters",
			branchName: "hotfix/bug-123_fix",
			basePath:   helper.CreateTempDir("test-base2"),
			want: func(result string) bool {
				return strings.Contains(result, "hotfix-bug-123_fix")
			},
		},
		{
			name:       "custom base path",
			branchName: "main",
			basePath:   helper.CreateTempDir("custom"),
			want: func(result string) bool {
				return strings.Contains(result, "custom") && strings.HasSuffix(result, "main")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := NewPathGenerator()
			result, err := pg.GeneratePath(tt.branchName, tt.basePath)

			require.NoError(t, err)
			require.True(t, tt.want(result), "unexpected result: %s", result)
			require.True(t, filepath.IsAbs(result), "result should be absolute path")

			// Verify directory was created
			info, err := os.Stat(result)
			require.NoError(t, err)
			assert.True(t, info.IsDir())
		})
	}
}

func TestPathGenerator_GeneratePath_Errors(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tests := []struct {
		name          string
		branchName    string
		basePath      string
		expectedError string
		expectedCode  string
	}{
		{
			name:          "empty branch name",
			branchName:    "",
			basePath:      helper.GetTempPath("test"),
			expectedError: "branch name cannot be empty",
			expectedCode:  errors.ErrCodeGitOperation,
		},
		{
			name:          "null byte in base path",
			branchName:    "test-branch",
			basePath:      "/tmp/test\x00dir",
			expectedError: "failed to resolve path collisions",
			expectedCode:  errors.ErrCodeFileSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := NewPathGenerator()
			result, err := pg.GeneratePath(tt.branchName, tt.basePath)

			require.Error(t, err)
			assert.Empty(t, result)

			var groveErr *errors.GroveError
			require.ErrorAs(t, err, &groveErr)
			assert.Equal(t, tt.expectedCode, groveErr.Code)
			assert.Contains(t, groveErr.Message, tt.expectedError)
		})
	}
}

func TestPathGenerator_ResolveUserPath_Success(t *testing.T) {
	tests := []struct {
		name     string
		userPath string
		want     func(string) bool
	}{
		{
			name:     "absolute path unchanged",
			userPath: "/absolute/path/test",
			want: func(result string) bool {
				return result == "/absolute/path/test"
			},
		},
		{
			name:     "relative path resolved",
			userPath: "relative/path",
			want: func(result string) bool {
				return filepath.IsAbs(result) && strings.HasSuffix(result, "relative/path")
			},
		},
		{
			name:     "current directory reference",
			userPath: "./current",
			want: func(result string) bool {
				return filepath.IsAbs(result) && strings.HasSuffix(result, "current")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := NewPathGenerator()
			result, err := pg.ResolveUserPath(tt.userPath)

			require.NoError(t, err)
			require.True(t, tt.want(result), "unexpected result: %s", result)
		})
	}
}

func TestPathGenerator_ResolveUserPath_Errors(t *testing.T) {
	tests := []struct {
		name     string
		userPath string
	}{
		{
			name:     "empty path",
			userPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := NewPathGenerator()
			result, err := pg.ResolveUserPath(tt.userPath)

			require.Error(t, err)
			assert.Empty(t, result)
			assert.Contains(t, err.Error(), "user path cannot be empty")
		})
	}
}

func TestPathGenerator_CollisionResolution(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	baseDir := helper.CreateTempDir("collision-test")

	// Pre-create directories to force collisions
	require.NoError(t, os.Mkdir(filepath.Join(baseDir, "test-branch"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(baseDir, "test-branch-1"), 0o755))
	require.NoError(t, os.Mkdir(filepath.Join(baseDir, "test-branch-2"), 0o755))

	pg := NewPathGenerator()
	result, err := pg.GeneratePath("test-branch", baseDir)

	require.NoError(t, err)
	assert.True(t, strings.HasSuffix(result, "test-branch-3"))

	// Verify the directory was actually created
	info, err := os.Stat(result)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestPathGenerator_AtomicOperations(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()
	baseDir := helper.CreateTempDir("atomic-test")

	pg := NewPathGenerator()

	// Test that the same path generation is atomic and consistent
	result1, err1 := pg.GeneratePath("atomic-branch", baseDir)
	require.NoError(t, err1)

	result2, err2 := pg.GeneratePath("atomic-branch", baseDir)
	require.NoError(t, err2)

	// Should get different paths due to collision resolution
	assert.NotEqual(t, result1, result2)
	assert.True(t, strings.HasSuffix(result1, "atomic-branch"))
	assert.True(t, strings.HasSuffix(result2, "atomic-branch-1"))

	// Both directories should exist
	info1, err := os.Stat(result1)
	require.NoError(t, err)
	assert.True(t, info1.IsDir())

	info2, err := os.Stat(result2)
	require.NoError(t, err)
	assert.True(t, info2.IsDir())
}

func TestPathGenerator_SecurityValidation(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tests := []struct {
		name          string
		branchName    string
		basePath      string
		expectedError string
		expectedCode  string
	}{
		{
			name:          "path traversal in branch name",
			branchName:    "../../../etc/passwd",
			basePath:      helper.GetTempPath("secure"),
			expectedError: "generated path is invalid",
			expectedCode:  errors.ErrCodeFileSystem,
		},
		{
			name:          "null byte injection in branch name",
			branchName:    "test\x00branch",
			basePath:      helper.GetTempPath("secure"),
			expectedError: "failed to resolve path collisions",
			expectedCode:  errors.ErrCodeFileSystem,
		},
		{
			name:          "extremely long path",
			branchName:    strings.Repeat("a", 5000), // Exceeds MaxPathLength
			basePath:      helper.GetTempPath("secure"),
			expectedError: "failed to resolve path collisions",
			expectedCode:  errors.ErrCodeFileSystem,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := NewPathGenerator()
			result, err := pg.GeneratePath(tt.branchName, tt.basePath)

			require.Error(t, err)
			assert.Empty(t, result)

			var groveErr *errors.GroveError
			require.ErrorAs(t, err, &groveErr)
			assert.Equal(t, tt.expectedCode, groveErr.Code)
			assert.Contains(t, groveErr.Message, tt.expectedError)
		})
	}
}

func TestPathGenerator_PathValidation(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tests := []struct {
		name          string
		setupFunc     func() string
		branchName    string
		expectedError string
	}{
		{
			name: "read-only parent directory",
			setupFunc: func() string {
				dir := helper.CreateTempDir("readonly-parent")
				require.NoError(t, os.Chmod(dir, 0o444)) // Read-only
				return dir
			},
			branchName:    "test-branch",
			expectedError: "permission denied",
		},
		{
			name: "parent is a file not directory",
			setupFunc: func() string {
				file := helper.CreateTempFile("not-a-dir", "content")
				return file
			},
			branchName:    "test-branch",
			expectedError: "not a directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePath := tt.setupFunc()

			pg := NewPathGenerator()
			result, err := pg.GeneratePath(tt.branchName, basePath)

			require.Error(t, err)
			assert.Empty(t, result)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestPathGenerator_Configuration(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem().WithCleanEnvironment()

	helper.Run(func() {
		// Reset viper for clean test
		viper.Reset()

		// Test default configuration
		config := DefaultPathGeneratorConfig()
		assert.Equal(t, 999, config.MaxCollisionAttempts)
		assert.Equal(t, 4096, config.MaxPathLength)
		assert.Equal(t, []int{1, 2, 3, 4, 5}, config.CommonCollisionNumbers)
	})
}

func TestPathGenerator_TildePathResolution(t *testing.T) {
	tests := []struct {
		name     string
		userPath string
		expected func(string) bool
	}{
		{
			name:     "tilde path in user path",
			userPath: "~/test-path",
			expected: func(result string) bool {
				return !strings.HasPrefix(result, "~") && strings.HasSuffix(result, "test-path")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := NewPathGenerator()
			result, err := pg.ResolveUserPath(tt.userPath)
			require.NoError(t, err)
			assert.True(t, tt.expected(result), "unexpected result: %s", result)
		})
	}
}

func TestPathGenerator_MaxCollisionAttempts(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem().WithCleanEnvironment()

	helper.Run(func() {
		viper.Reset()
		viper.Set("path_generator.max_collision_attempts", 3)

		baseDir := helper.CreateTempDir("max-collision-test")

		// Pre-create directories to force collisions beyond the limit
		for i := 0; i <= 5; i++ {
			suffix := ""
			if i > 0 {
				suffix = fmt.Sprintf("-%d", i)
			}
			require.NoError(t, os.Mkdir(filepath.Join(baseDir, "test-branch"+suffix), 0o755))
		}

		pg := NewPathGenerator()
		result, err := pg.GeneratePath("test-branch", baseDir)

		require.Error(t, err)
		assert.Empty(t, result)
		assert.Contains(t, err.Error(), "unable to find unique path after")
	})
}

func TestPathGenerator_EdgeCases(t *testing.T) {
	helper := testutils.NewUnitTestHelper(t).WithCleanFilesystem()

	tests := []struct {
		name       string
		branchName string
		setup      func() string
		wantErr    bool
	}{
		{
			name:       "branch with only special chars",
			branchName: "///---///",
			setup: func() string {
				return helper.GetTempPath("edge-case")
			},
			wantErr: false,
		},
		{
			name:       "very long branch name",
			branchName: strings.Repeat("long-branch-name-", 20),
			setup: func() string {
				return helper.GetTempPath("edge-case")
			},
			wantErr: true, // This should fail due to filesystem limits
		},
		{
			name:       "unicode branch name",
			branchName: "feature/测试分支-ñoño",
			setup: func() string {
				return helper.GetTempPath("edge-case")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePath := tt.setup()
			pg := NewPathGenerator()

			result, err := pg.GeneratePath(tt.branchName, basePath)

			if tt.wantErr {
				require.Error(t, err)
				assert.Empty(t, result)
			} else {
				require.NoError(t, err)
				assert.NotEmpty(t, result)
				assert.True(t, filepath.IsAbs(result))

				// Verify directory was created
				info, err := os.Stat(result)
				require.NoError(t, err)
				assert.True(t, info.IsDir())
			}
		})
	}
}
