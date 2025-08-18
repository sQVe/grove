package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/fs"
)

func TestGroveCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go", "--help")
	if err := cmd.Run(); err != nil {
		t.Fatalf("grove command should exist and show help: %v", err)
	}
}

func TestGroveCommandNoArgs(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("grove command with no args should show help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Grove is a CLI tool that makes Git worktrees as simple") {
		t.Errorf("help should contain description, got: %s", outputStr)
	}
	if !strings.Contains(outputStr, "init") {
		t.Errorf("help should mention init command, got: %s", outputStr)
	}
}

func TestGroveInitCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go", "init", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("grove init command should exist and show help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "init") {
		t.Errorf("init command help should mention 'init', got: %s", outputStr)
	}
}

func TestGroveInitExecution(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go", "init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("grove init command should execute successfully: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "Initialized Grove") {
		t.Errorf("init should output initialization message, got: %s", outputStr)
	}
}

func TestGroveInitNewCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "main.go", "init", "new", "--help")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("grove init new command should exist and show help: %v", err)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "new") {
		t.Errorf("init new command help should mention 'new', got: %s", outputStr)
	}
}

func TestGroveInitNewCurrentDirectory(t *testing.T) {
	tempDir := t.TempDir()
	projectDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	cmd := exec.Command("go", "run", "main.go", "init", "new", tempDir) //nolint:gosec // Test with controlled temp directory
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("grove init new should initialize directory: %v, output: %s", err, output)
	}
	if _, err := os.Stat(filepath.Join(tempDir, ".bare")); os.IsNotExist(err) {
		t.Error(".bare directory should be created")
	}
	if _, err := os.Stat(filepath.Join(tempDir, ".git")); os.IsNotExist(err) {
		t.Error(".git file should be created")
	}
}

func TestGroveInitNewWithDirectoryName(t *testing.T) {
	tempDir := t.TempDir()
	projectDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	projectName := "myproject"
	projectPath := filepath.Join(tempDir, projectName)

	cmd := exec.Command("go", "run", "main.go", "init", "new", projectPath) //nolint:gosec // Test with controlled temp directory
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("grove init new %s should create and initialize directory: %v, output: %s", projectName, err, output)
	}
	if _, err := os.Stat(filepath.Join(projectPath, ".bare")); os.IsNotExist(err) {
		t.Errorf(".bare directory should be created in %s", projectPath)
	}
	if _, err := os.Stat(filepath.Join(projectPath, ".git")); os.IsNotExist(err) {
		t.Errorf(".git file should be created in %s", projectPath)
	}
}

func TestGroveInitNewDirectoryNotEmpty(t *testing.T) {
	tempDir := t.TempDir()
	projectDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	testFile := filepath.Join(tempDir, "existing.txt")
	if err := os.WriteFile(testFile, []byte("content"), fs.FileStrict); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	cmd := exec.Command("go", "run", "main.go", "init", "new", tempDir) //nolint:gosec // Test with controlled temp directory
	cmd.Dir = projectDir
	output, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("grove init new should fail when directory is not empty, got output: %s", output)
	}

	outputStr := string(output)
	if !strings.Contains(outputStr, "not empty") {
		t.Errorf("error message should mention directory not empty, got: %s", outputStr)
	}
}
