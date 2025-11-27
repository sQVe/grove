package commands

import (
	"errors"
	"os"
	"testing"

	"github.com/sqve/grove/internal/workspace"
)

func TestNewCreateCmd(t *testing.T) {
	cmd := NewCreateCmd()
	if cmd.Use != "create <branch>" {
		t.Errorf("expected Use to be 'create <branch>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestRunCreate_NotInWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := t.TempDir()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runCreate("feature-test")
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestSanitizeBranchName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"feature/auth", "feature-auth"},
		{"simple", "simple"},
		{"feature/auth/login", "feature-auth-login"},
		{"has<brackets>", "has-brackets-"},
		{"has|pipe", "has-pipe"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := sanitizeBranchName(tc.input)
			if result != tc.expected {
				t.Errorf("sanitizeBranchName(%q) = %q, want %q", tc.input, result, tc.expected)
			}
		})
	}
}
