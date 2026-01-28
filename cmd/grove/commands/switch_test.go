package commands

import (
	"bytes"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/testutil"
	"github.com/sqve/grove/internal/workspace"
)

func TestNewSwitchCmd(t *testing.T) {
	cmd := NewSwitchCmd()
	if cmd.Use != "switch <worktree>" {
		t.Errorf("expected Use to be 'switch <worktree>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}

func TestNewSwitchCmd_RequiresOneArg(t *testing.T) {
	cmd := NewSwitchCmd()

	// Check that Args is set to expect exactly 1 argument
	err := cmd.Args(cmd, []string{})
	if err == nil {
		t.Error("expected error when no arguments provided")
	}

	err = cmd.Args(cmd, []string{"branch1", "branch2"})
	if err == nil {
		t.Error("expected error when too many arguments provided")
	}

	err = cmd.Args(cmd, []string{"branch1"})
	if err != nil {
		t.Errorf("unexpected error with single argument: %v", err)
	}
}

func TestRunSwitch_NotInWorkspace(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	tmpDir := testutil.TempDir(t)
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	err = runSwitch("main")
	if !errors.Is(err, workspace.ErrNotInWorkspace) {
		t.Errorf("expected ErrNotInWorkspace, got %v", err)
	}
}

func TestNewSwitchCmd_HasShellInitSubcommand(t *testing.T) {
	cmd := NewSwitchCmd()
	subCmd, _, err := cmd.Find([]string{"shell-init"})
	if err != nil {
		t.Fatalf("expected shell-init subcommand, got error: %v", err)
	}
	if subCmd.Name() != "shell-init" {
		t.Errorf("expected subcommand name 'shell-init', got %q", subCmd.Name())
	}
}

func TestNewSwitchCmd_ValidArgsFunction(t *testing.T) {
	cmd := NewSwitchCmd()

	// ValidArgsFunction should be set
	if cmd.ValidArgsFunction == nil {
		t.Error("expected ValidArgsFunction to be set")
	}

	// When already has an arg, should return no file completion
	_, directive := cmd.ValidArgsFunction(cmd, []string{"existing"}, "")
	if directive != cobra.ShellCompDirectiveNoFileComp {
		t.Errorf("expected ShellCompDirectiveNoFileComp when args present, got %v", directive)
	}
}

func TestDetectShell(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		psModule string
		expected string
	}{
		{"bash returns sh", "/bin/bash", "", "sh"},
		{"zsh returns sh", "/bin/zsh", "", "sh"},
		{"fish returns fish", "/usr/bin/fish", "", "fish"},
		{"sh returns sh", "/bin/sh", "", "sh"},
		{"dash returns sh", "/bin/dash", "", "sh"},
		{"ash returns sh", "/bin/ash", "", "sh"},
		{"empty with PSModulePath returns powershell", "", "C:\\Program Files\\PowerShell", "powershell"},
		{"empty returns sh", "", "", "sh"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("SHELL", tt.shell)
			t.Setenv("PSModulePath", tt.psModule)
			result := detectShell()
			if result != tt.expected {
				t.Errorf("detectShell() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPrintShellIntegration(t *testing.T) {
	tests := []struct {
		shell       string
		wantErr     bool
		wantContain string // expected content in output
	}{
		{"bash", false, "grove()"},
		{"zsh", false, "grove()"},
		{"sh", false, "grove()"},
		{"dash", false, "grove()"},
		{"fish", false, "function grove"},
		{"powershell", false, "function grove"},
		{"pwsh", false, "function grove"},
		{"tcsh", true, ""}, // unsupported
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			err := printShellIntegration(tt.shell)

			_ = w.Close()
			os.Stdout = oldStdout

			var buf bytes.Buffer
			_, _ = io.Copy(&buf, r)
			output := buf.String()

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for shell %q", tt.shell)
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error for shell %q: %v", tt.shell, err)
				return
			}

			if !strings.Contains(output, tt.wantContain) {
				t.Errorf("printShellIntegration(%q) output = %q, want to contain %q", tt.shell, output, tt.wantContain)
			}
		})
	}
}
