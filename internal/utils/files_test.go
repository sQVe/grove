//go:build !integration
// +build !integration

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsHidden(t *testing.T) {
	tests := []struct {
		filename string
		expected bool
		desc     string
	}{
		{".git", true, "Git directory"},
		{".gitignore", true, "Git ignore file"},
		{".DS_Store", true, "macOS system file"},
		{".env", true, "Environment file"},
		{".hidden", true, "Generic hidden file"},
		{"file.txt", false, "Regular text file"},
		{"README.md", false, "Regular markdown file"},
		{"normal", false, "File without extension"},
		{"", false, "Empty filename"},
		{".", true, "Current directory (single dot)"},
		{"..", true, "Parent directory (double dot)"},
		{"a.txt", false, "File starting with letter"},
		{"123", false, "Numeric filename"},
		{".config", true, "Config directory/file"},
		{"config", false, "Config without dot"},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			result := IsHidden(test.filename)
			assert.Equal(t, test.expected, result, "filename: %s", test.filename)
		})
	}
}
