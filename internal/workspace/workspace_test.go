package workspace

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

func TestIsInsideGroveWorkspace(t *testing.T) {
	t.Run("returns false for non-grove directory", func(t *testing.T) {
		tempDir := t.TempDir()
		result := IsInsideGroveWorkspace(tempDir)
		if result {
			t.Error("expected false for non-grove directory")
		}
	})

	t.Run("returns true for grove workspace root", func(t *testing.T) {
		tempDir := t.TempDir()
		gitFile := filepath.Join(tempDir, ".git")
		if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		result := IsInsideGroveWorkspace(tempDir)
		if !result {
			t.Error("expected true for grove workspace root")
		}
	})

	t.Run("returns true for subdirectory of grove workspace", func(t *testing.T) {
		tempDir := t.TempDir()
		gitFile := filepath.Join(tempDir, ".git")
		if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
			t.Fatalf("failed to create .git file: %v", err)
		}

		subDir := filepath.Join(tempDir, "subdir", "nested")
		if err := os.MkdirAll(subDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create subdirectory: %v", err)
		}

		result := IsInsideGroveWorkspace(subDir)
		if !result {
			t.Error("expected true for subdirectory of grove workspace")
		}
	})

	t.Run("returns false for regular git repository", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")
		if err := os.Mkdir(gitDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create .git directory: %v", err)
		}

		result := IsInsideGroveWorkspace(tempDir)
		if result {
			t.Error("expected false for regular git repository")
		}
	})
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		branch   string
		expected string
	}{
		{"feature/add-button", "feature-add-button"},
		{"feat/user-auth", "feat-user-auth"},
		{"bug/fix-123", "bug-fix-123"},
		{"release/v1.0.0", "release-v1.0.0"},
		{"hotfix/urgent-patch", "hotfix-urgent-patch"},
		{"no-slash", "no-slash"},
		{"multiple//slashes///here", "multiple--slashes---here"},
		{"trailing/slash/", "trailing-slash-"},
		{"/leading/slash", "-leading-slash"},
		{"branch<name>with|chars", "branch-name-with-chars"},
		{`branch"with"quotes`, "branch-with-quotes"},
	}

	for _, tt := range tests {
		t.Run(tt.branch, func(t *testing.T) {
			result := sanitizeBranchName(tt.branch)
			if result != tt.expected {
				t.Errorf("expected '%s', got '%s'", tt.expected, result)
			}
		})
	}
}
