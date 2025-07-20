package git

import (
	"errors"
	"testing"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestIsNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("timeout"),
			expected: true,
		},
		{
			name:     "connection error",
			err:      errors.New("connection refused"),
			expected: true,
		},
		{
			name:     "network unreachable",
			err:      errors.New("network is unreachable"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("file not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isNetworkError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "authentication failed",
			err:      errors.New("authentication failed"),
			expected: true,
		},
		{
			name:     "permission denied",
			err:      errors.New("permission denied"),
			expected: true,
		},
		{
			name:     "unauthorized",
			err:      errors.New("unauthorized"),
			expected: true,
		},
		{
			name:     "other error",
			err:      errors.New("file not found"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAuthError(tt.err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidatePathsErrors(t *testing.T) {
	tests := []struct {
		name      string
		mainDir   string
		bareDir   string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid paths",
			mainDir:   "/valid/path",
			bareDir:   "/valid/bare",
			expectErr: false,
		},
		{
			name:      "path traversal attempt in main",
			mainDir:   "../../../etc/passwd",
			bareDir:   "/valid/bare",
			expectErr: true,
			errMsg:    "directory traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePaths(tt.mainDir, tt.bareDir)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCreateGitFileError(t *testing.T) {
	testDir := testutils.NewTestDirectory(t, "grove-git-file-test")
	defer testDir.Cleanup()

	tests := []struct {
		name        string
		expectError bool
	}{
		{
			name:        "create valid git file",
			expectError: false,
		},
		{
			name:        "create with invalid directories",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mainDir := testDir.Path
			bareDir := testDir.Path + "/.bare"

			if tt.expectError {
				// Use invalid directories to trigger error
				bareDir = "../invalid"
			}

			err := CreateGitFile(mainDir, bareDir)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestErrorHelpers(t *testing.T) {
	t.Run("network errors", func(t *testing.T) {
		assert.True(t, isNetworkError(errors.New("timeout")))
		assert.False(t, isNetworkError(errors.New("other error")))
	})

	t.Run("auth errors", func(t *testing.T) {
		assert.True(t, isAuthError(errors.New("authentication failed")))
		assert.False(t, isAuthError(errors.New("other error")))
	})
}
