package main

import (
	"os/exec"
	"strings"
	"testing"
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
