package testutils

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func AssertDirectoryEmpty(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var visibleFiles []string

	for _, entry := range entries {
		if entry.Name()[0] != '.' {
			visibleFiles = append(visibleFiles, entry.Name())
		}
	}

	assert.Empty(t, visibleFiles, "Directory should be empty of visible files")
}

func AssertDirectoryNotEmpty(t *testing.T, dir string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var visibleFiles []string

	for _, entry := range entries {
		if entry.Name()[0] != '.' {
			visibleFiles = append(visibleFiles, entry.Name())
		}
	}

	assert.NotEmpty(t, visibleFiles, "Directory should contain visible files")
}

func AssertFileContent(t *testing.T, filePath, expectedContent string) {
	t.Helper()

	content, err := os.ReadFile(filePath)
	require.NoError(t, err)
	assert.Equal(t, expectedContent, string(content))
}

func AssertErrorContains(t *testing.T, err error, expectedMessage string) {
	t.Helper()
	require.Error(t, err)
	assert.Contains(t, err.Error(), expectedMessage)
}
