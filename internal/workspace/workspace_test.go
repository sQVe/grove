package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

const errInsideGitRepo = "cannot initialize grove inside existing git repository"

func TestInitialize(t *testing.T) {
	tempDir := t.TempDir()

	if err := Initialize(tempDir); err != nil {
		t.Fatalf("Initialize should succeed on empty directory: %v", err)
	}

	bareDir := filepath.Join(tempDir, ".bare")
	if _, err := os.Stat(bareDir); os.IsNotExist(err) {
		t.Error(".bare directory should be created")
	}

	gitFile := filepath.Join(tempDir, ".git")
	if _, err := os.Stat(gitFile); os.IsNotExist(err) {
		t.Error(".git file should be created")
	}

	content, err := os.ReadFile(gitFile) // nolint:gosec // Reading controlled test file
	if err != nil {
		t.Fatalf("failed to read .git file: %v", err)
	}
	expected := groveGitContent
	if string(content) != expected {
		t.Errorf(".git file should contain '%s', got '%s'", expected, string(content))
	}

	if _, err := os.Stat(filepath.Join(bareDir, "HEAD")); os.IsNotExist(err) {
		t.Error("HEAD file should exist in bare repository")
	}
}

func TestInitializeNonEmptyDirectory(t *testing.T) {
	tempDir := t.TempDir()

	testFile := filepath.Join(tempDir, "existing.txt")
	if err := os.WriteFile(testFile, []byte("content"), fs.FileStrict); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	err := Initialize(tempDir)
	if err == nil {
		t.Fatal("Initialize should fail on non-empty directory")
	}

	if !os.IsExist(err) && err.Error() != "directory "+tempDir+" is not empty" {
		t.Errorf("expected 'directory not empty' error, got: %v", err)
	}
}

func TestInitializeCleanupOnGitFailure(t *testing.T) {
	tempDir := t.TempDir()

	t.Setenv("PATH", "")

	err := Initialize(tempDir)
	if err == nil {
		t.Fatal("Initialize should fail when git is not available")
	}

	bareDir := filepath.Join(tempDir, ".bare")
	if _, err := os.Stat(bareDir); !os.IsNotExist(err) {
		t.Error(".bare directory should be cleaned up on git init failure")
	}

	gitFile := filepath.Join(tempDir, ".git")
	if _, err := os.Stat(gitFile); !os.IsNotExist(err) {
		t.Error(".git file should not exist when git init fails")
	}
}

func TestInitializeCleanupOnGitFileFailure(t *testing.T) {
	tempDir := t.TempDir()

	if err := os.Chmod(tempDir, 0o555); err != nil { // nolint:gosec // Test needs read-only permissions
		t.Fatalf("failed to make directory read-only: %v", err)
	}
	defer func() { _ = os.Chmod(tempDir, fs.DirGit) }()

	err := Initialize(tempDir)
	if err == nil {
		t.Fatal("Initialize should fail when .git file cannot be created")
	}

	_ = os.Chmod(tempDir, fs.DirGit)

	bareDir := filepath.Join(tempDir, ".bare")
	if _, err := os.Stat(bareDir); !os.IsNotExist(err) {
		t.Error(".bare directory should be cleaned up on .git file creation failure")
	}
}

func TestInitializeNoCleanupOnExistingDirectory(t *testing.T) {
	tempDir := t.TempDir()

	existingDir := filepath.Join(tempDir, "existing")
	if err := os.Mkdir(existingDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create existing directory: %v", err)
	}

	existingFile := filepath.Join(existingDir, "important.txt")
	if err := os.WriteFile(existingFile, []byte("important data"), fs.FileStrict); err != nil {
		t.Fatalf("failed to create existing file: %v", err)
	}

	t.Setenv("PATH", "") // Make git unavailable to force failure
	err := Initialize(existingDir)
	if err == nil {
		t.Fatal("Initialize should fail on non-empty directory")
	}

	if _, err := os.Stat(existingDir); os.IsNotExist(err) {
		t.Error("existing directory should not be removed on failure")
	}

	if _, err := os.Stat(existingFile); os.IsNotExist(err) {
		t.Error("existing file should not be removed on failure")
	}
}

func TestInitializeDetectExistingGitDirectory(t *testing.T) {
	tempDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to initialize git repository: %v", err)
	}

	err := Initialize(tempDir)
	if err == nil {
		t.Fatal("Initialize should fail when git repository already exists")
	}

	if !os.IsExist(err) && err.Error() != errInsideGitRepo {
		t.Errorf("expected 'inside existing git repository' error, got: %v", err)
	}
}

func TestInitializeDetectExistingGitFile(t *testing.T) {
	tempDir := t.TempDir()

	mainRepo := filepath.Join(tempDir, "main")
	if err := os.Mkdir(mainRepo, fs.DirGit); err != nil {
		t.Fatalf("failed to create main repo directory: %v", err)
	}

	cmd := exec.Command("git", "init")
	cmd.Dir = mainRepo
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to initialize main git repository: %v", err)
	}

	worktreeDir := filepath.Join(tempDir, "worktree")
	if err := os.Mkdir(worktreeDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create worktree directory: %v", err)
	}

	cmd = exec.Command("git", "worktree", "add", "../worktree")
	cmd.Dir = mainRepo
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to create git worktree: %v", err)
	}

	err := Initialize(worktreeDir)
	if err == nil {
		t.Fatal("Initialize should fail when git worktree already exists")
	}

	if !os.IsExist(err) && err.Error() != errInsideGitRepo {
		t.Errorf("expected 'inside existing git repository' error, got: %v", err)
	}
}

func TestInitializeInsideGitRepository(t *testing.T) {
	tempDir := t.TempDir()

	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to initialize git repository: %v", err)
	}

	subDir := filepath.Join(tempDir, "subproject")
	if err := os.Mkdir(subDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create subdirectory: %v", err)
	}

	err := Initialize(subDir)
	if err == nil {
		t.Fatal("Initialize should fail when inside an existing git repository")
	}

	if !os.IsExist(err) && err.Error() != errInsideGitRepo {
		t.Errorf("expected 'inside existing git repository' error, got: %v", err)
	}
}

func TestIsInsideGroveWorkspace(t *testing.T) {
	tempDir := t.TempDir()

	if IsInsideGroveWorkspace(tempDir) {
		t.Error("empty directory should not be inside grove workspace")
	}
}

func TestIsInsideGroveWorkspaceWithBareDir(t *testing.T) {
	tempDir := t.TempDir()

	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create .bare directory: %v", err)
	}

	if !IsInsideGroveWorkspace(tempDir) {
		t.Error("directory with .bare should be inside grove workspace")
	}
}

func TestIsInsideGroveWorkspaceWithGitFile(t *testing.T) {
	tempDir := t.TempDir()

	gitFile := filepath.Join(tempDir, ".git")
	if err := os.WriteFile(gitFile, []byte(groveGitContent), fs.FileGit); err != nil {
		t.Fatalf("failed to create .git file: %v", err)
	}

	if !IsInsideGroveWorkspace(tempDir) {
		t.Error("directory with grove .git file should be inside grove workspace")
	}
}

func TestIsInsideGroveWorkspaceNested(t *testing.T) {
	tempDir := t.TempDir()

	bareDir := filepath.Join(tempDir, ".bare")
	if err := os.Mkdir(bareDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create .bare directory: %v", err)
	}

	subDir := filepath.Join(tempDir, "subdir", "nested")
	if err := os.MkdirAll(subDir, fs.DirGit); err != nil {
		t.Fatalf("failed to create nested directory: %v", err)
	}

	if !IsInsideGroveWorkspace(subDir) {
		t.Error("nested directory should be inside grove workspace")
	}
}

func TestIsInsideGroveWorkspaceInvalidPath(t *testing.T) {
	if IsInsideGroveWorkspace("/nonexistent/path") {
		t.Error("nonexistent path should not be inside grove workspace")
	}
}
