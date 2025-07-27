//go:build !integration
// +build !integration

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsGitAvailable(t *testing.T) {
	assert.True(t, IsGitAvailable(), "git should be available in PATH")
}

func TestIsGitAvailableWithModifiedPATH(t *testing.T) {
	originalPath := os.Getenv("PATH")
	defer func() {
		_ = os.Setenv("PATH", originalPath)
	}()

	t.Run("git not in PATH", func(t *testing.T) {
		err := os.Setenv("PATH", "")
		require.NoError(t, err)

		assert.False(t, IsGitAvailable(), "git should not be available with empty PATH")
	})

	t.Run("git available in custom path", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "git-test-*")
		require.NoError(t, err)
		defer func() { _ = os.RemoveAll(tempDir) }()

		gitPath := filepath.Join(tempDir, "git")
		err = os.WriteFile(gitPath, []byte("#!/bin/sh\necho 'mock git'\n"), 0o600)
		require.NoError(t, err)

		// Make the script executable (owner and group only).
		err = os.Chmod(gitPath, 0o700)
		require.NoError(t, err)

		err = os.Setenv("PATH", tempDir)
		require.NoError(t, err)

		assert.True(t, IsGitAvailable(), "git should be available in custom PATH")
	})
}
