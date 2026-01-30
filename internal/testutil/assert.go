package testutil

import (
	"os"
	"slices"
	"strings"
	"testing"
)

// AssertErrorContains fails if err is nil or doesn't contain substring.
func AssertErrorContains(t *testing.T, err error, substring string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error containing %q, got nil", substring)
	}
	if !strings.Contains(err.Error(), substring) {
		t.Fatalf("expected error containing %q, got: %v", substring, err)
	}
}

// AssertPathExists fails if path doesn't exist. Works for files and directories.
func AssertPathExists(t *testing.T, path string) {
	t.Helper()
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("expected path %s to exist", path)
		}
		t.Fatalf("failed to stat path %s: %v", path, err)
	}
}

// AssertFileContent fails if file content doesn't match expected.
func AssertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	content, err := os.ReadFile(path) // nolint:gosec // Test helper with controlled input
	if err != nil {
		t.Fatalf("failed to read file %s: %v", path, err)
	}
	if string(content) != expected {
		t.Fatalf("file %s: expected %q, got %q", path, expected, string(content))
	}
}

// AssertContains fails if slice doesn't contain value.
func AssertContains(t *testing.T, slice []string, value string) {
	t.Helper()
	if !slices.Contains(slice, value) {
		t.Fatalf("expected slice to contain %q, got: %v", value, slice)
	}
}
