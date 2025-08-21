package main

import (
	"os/exec"
	"strings"
	"testing"
)

func TestGroveCommand(t *testing.T) {
	t.Run("shows help with --help flag", func(t *testing.T) {
		cmd := exec.Command("go", "run", "main.go", "--help")
		if err := cmd.Run(); err != nil {
			t.Fatalf("grove command should exist and show help: %v", err)
		}
	})

	t.Run("shows help with no arguments", func(t *testing.T) {
		cmd := exec.Command("go", "run", "main.go")
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("grove command with no args should show help: %v", err)
		}

		outputStr := string(output)
		if !strings.Contains(outputStr, "Grove is a tool that makes Git worktrees as simple") {
			t.Errorf("help should contain description, got: %s", outputStr)
		}
		if !strings.Contains(outputStr, "init") {
			t.Errorf("help should mention init command, got: %s", outputStr)
		}
	})
}
