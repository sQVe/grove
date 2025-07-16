// Package testutils provides utilities for testing Grove functionality.
package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertDirectoryEmpty asserts that the directory is empty (no non-hidden files).
func AssertDirectoryEmpty(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	// Filter out hidden files
	var visibleFiles []string

	for _, entry := range entries {
		if entry.Name()[0] != '.' {
			visibleFiles = append(visibleFiles, entry.Name())
		}
	}

	assert.Empty(t, visibleFiles, "Directory should be empty of visible files")
}

// AssertDirectoryNotEmpty asserts that the directory contains visible files.
func AssertDirectoryNotEmpty(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	// Filter out hidden files
	var visibleFiles []string

	for _, entry := range entries {
		if entry.Name()[0] != '.' {
			visibleFiles = append(visibleFiles, entry.Name())
		}
	}

	assert.NotEmpty(t, visibleFiles, "Directory should contain visible files")
}

// AssertFileContent asserts that a file contains the expected content.
func AssertFileContent(t *testing.T, filePath, expectedContent string) {
	t.Helper()

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
}

// AssertErrorContains asserts that an error is not nil and contains the expected message.
func AssertErrorContains(t *testing.T, err error, expectedMessage string) {
	t.Helper()
	require.Error(t, err)
	assert.Contains(t, err.Error(), expectedMessage)
}
