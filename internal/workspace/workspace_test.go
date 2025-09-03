package workspace

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/config"
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

func TestPreserveIgnoredFilesFromList_NoIgnoredFiles(t *testing.T) {
	tempDir := t.TempDir()
	branches := []string{"main", "develop"}

	count, patterns, err := preserveIgnoredFilesFromList(tempDir, branches, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected preserved file count 0, got %d", count)
	}
	if patterns != nil {
		t.Errorf("expected nil matched patterns, got %v", patterns)
	}
}

func TestPreserveIgnoredFilesFromList_ValidPreserve(t *testing.T) {
	tempDir := t.TempDir()
	branches := []string{"main", "develop"}

	// Create worktree directories
	for _, branch := range branches {
		branchDir := filepath.Join(tempDir, sanitizeBranchName(branch))
		if err := os.MkdirAll(branchDir, fs.DirGit); err != nil {
			t.Fatalf("failed to create branch directory %s: %v", branchDir, err)
		}
	}

	// Create a file that matches default ".env" preserve pattern
	envPath := filepath.Join(tempDir, ".env")
	content := []byte("preserve test content")
	if err := os.WriteFile(envPath, content, fs.FileStrict); err != nil {
		t.Fatalf("failed to create file %s: %v", envPath, err)
	}

	// Create a non-matching file
	nonMatchPath := filepath.Join(tempDir, "ignored.txt")
	if err := os.WriteFile(nonMatchPath, []byte("should not be preserved"), fs.FileStrict); err != nil {
		t.Fatalf("failed to create file %s: %v", nonMatchPath, err)
	}

	ignoredFiles := []string{".env", "ignored.txt"}
	count, matched, err := preserveIgnoredFilesFromList(tempDir, branches, ignoredFiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the ".env" file should be preserved
	if count != 1 {
		t.Errorf("expected preserved file count 1, got %d", count)
	}
	found := false
	for _, pat := range matched {
		if pat == ".env" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected matched patterns to include '.env', got %v", matched)
	}

	// Verify the preserved file exists in each worktree directory
	for _, branch := range branches {
		branchDir := filepath.Join(tempDir, sanitizeBranchName(branch))
		preservedFile := filepath.Join(branchDir, ".env")
		if _, err := os.Stat(preservedFile); err != nil {
			t.Errorf("expected preserved file %s in branch %q, error: %v", preservedFile, branch, err)
		} else {
			c, err := os.ReadFile(preservedFile) // nolint:gosec // Test file - controlled path
			if err != nil {
				t.Errorf("failed to read file %s: %v", preservedFile, err)
			}
			if !bytes.Equal(c, content) {
				t.Errorf("content mismatch in preserved file %s: got %q, want %q", preservedFile, string(c), string(content))
			}
		}
	}
}

func TestPreserveIgnoredFilesFromList_CustomPattern(t *testing.T) {
	tempDir := t.TempDir()
	branches := []string{"feature"}

	// Create worktree directory
	branchDir := filepath.Join(tempDir, sanitizeBranchName("feature"))
	if err := os.MkdirAll(branchDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create branch directory %s: %v", branchDir, err)
	}

	// Save original config
	originalPatterns := config.Global.PreservePatterns
	defer func() {
		config.Global.PreservePatterns = originalPatterns
	}()

	// Set custom preserve patterns
	config.Global.PreservePatterns = []string{"*.custom"}

	// Create a file that matches the custom pattern
	customFileName := "data.custom"
	customFilePath := filepath.Join(tempDir, customFileName)
	content := []byte("custom content")
	if err := os.WriteFile(customFilePath, content, fs.FileStrict); err != nil {
		t.Fatalf("failed to create file %s: %v", customFilePath, err)
	}

	ignoredFiles := []string{customFileName}
	count, matched, err := preserveIgnoredFilesFromList(tempDir, branches, ignoredFiles)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if count != 1 {
		t.Errorf("expected preserved file count 1, got %d", count)
	}
	found := false
	for _, pat := range matched {
		if pat == "*.custom" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected matched patterns to include '*.custom', got %v", matched)
	}

	// Verify the file is copied to the worktree directory
	preservedFile := filepath.Join(branchDir, customFileName)
	if _, err := os.Stat(preservedFile); err != nil {
		t.Errorf("expected preserved file %s in branch 'feature', error: %v", preservedFile, err)
	}
}

func TestPreserveIgnoredFilesFromList_MissingSource(t *testing.T) {
	tempDir := t.TempDir()
	branches := []string{"main"}

	// Create worktree directory
	branchDir := filepath.Join(tempDir, sanitizeBranchName("main"))
	if err := os.MkdirAll(branchDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create branch directory %s: %v", branchDir, err)
	}

	// Do not create the file ".env" even though it matches default preserve pattern
	ignoredFiles := []string{".env"}
	_, _, err := preserveIgnoredFilesFromList(tempDir, branches, ignoredFiles)
	if err == nil {
		t.Error("expected error due to missing source file, got nil")
	}
	if !strings.Contains(err.Error(), "failed to preserve file") {
		t.Errorf("expected error message about failing to preserve file, got: %v", err)
	}
}
