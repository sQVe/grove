package commands

import (
	"os"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

func TestInitCmd_ValidGitRepository(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change to temp dir: %v", err)
	}
	if err := os.Mkdir(fs.GitDir, fs.DirGit); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	cmd := NewInitCmd()
	if cmd == nil {
		t.Fatal("NewInitCmd() returned nil")
	}
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command should succeed in git repository: %v", err)
	}
}

func TestInitCmd_NonGitRepository(t *testing.T) {
	cmd := NewInitCmd()
	if cmd == nil {
		t.Fatal("NewInitCmd() returned nil")
	}

	// Currently init command does nothing, so it should succeed
	// This test will need updating when git validation is implemented
	if err := cmd.Execute(); err != nil {
		t.Fatalf("init command failed: %v", err)
	}
}
