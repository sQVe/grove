package commands

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/testutil"
	testgit "github.com/sqve/grove/internal/testutil/git"
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

func TestRunSwitch_TargetMatching(t *testing.T) {
	tests := []struct {
		name      string
		branches  []string
		target    string
		wantPath  string
		wantError string
		notFound  bool
	}{
		{
			name:     "exact name wins",
			branches: []string{"main", "auth-fix", "feat/auth"},
			target:   "auth", wantPath: "feat/auth",
		},
		{
			name:     "exact branch wins",
			branches: []string{"main", "feat/auth-extra", "feat/auth"},
			target:   "feat/auth", wantPath: "feat/auth",
		},
		{
			name:     "unique name and branch substring counts once",
			branches: []string{"main", "feat-auth"},
			target:   "auth", wantPath: "feat-auth",
		},
		{
			name:     "unique branch substring",
			branches: []string{"main", "feature/login"},
			target:   "feature/", wantPath: "feature/login",
		},
		{
			name:     "ambiguous substring lists candidates in worktree order",
			branches: []string{"main", "feat-auth", "auth-fix"},
			target:   "auth", wantError: `ambiguous target "auth": auth-fix, feat-auth`,
		},
		{
			name:     "no match",
			branches: []string{"main", "feat-auth"},
			target:   "missing", wantError: "worktree not found: missing", notFound: true,
		},
		{
			name:     "empty after trimming",
			branches: []string{"main"},
			target:   "   ", wantError: "worktree not found: ", notFound: true,
		},
		{
			name:     "case sensitive",
			branches: []string{"main", "feat-auth"},
			target:   "AUTH", wantError: "worktree not found: AUTH", notFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			groveWorkspace := testgit.NewGroveWorkspace(t, tt.branches...)
			t.Chdir(groveWorkspace.Dir)

			oldStdout := os.Stdout
			reader, writer, err := os.Pipe()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = reader.Close() }()
			os.Stdout = writer

			err = runSwitch(tt.target)

			_ = writer.Close()
			os.Stdout = oldStdout
			var output bytes.Buffer
			_, _ = io.Copy(&output, reader)

			if tt.wantError != "" {
				if err == nil || err.Error() != tt.wantError {
					t.Errorf("runSwitch() error = %v, want %q", err, tt.wantError)
				}
				if errors.Is(err, ErrWorktreeNotFound) != tt.notFound {
					t.Errorf("runSwitch() error = %v, want not-found = %v", err, tt.notFound)
				}
				if output.Len() != 0 {
					t.Errorf("unexpected output on error: %q", output.String())
				}
				return
			}

			if err != nil {
				t.Fatalf("runSwitch() failed: %v", err)
			}
			want := filepath.Join(groveWorkspace.Dir, tt.wantPath) + "\n"
			if output.String() != want {
				t.Errorf("runSwitch() output = %q, want %q", output.String(), want)
			}
		})
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

func TestResolveBySubstring(t *testing.T) {
	t.Parallel()

	detached := &git.WorktreeInfo{Path: "/ws/tmp", Branch: "3fa61bc", Detached: true}
	alpha := &git.WorktreeInfo{Path: "/ws/alpha", Branch: "alpha"}
	authFix := &git.WorktreeInfo{Path: "/ws/auth-fix", Branch: "auth-fix"}
	featAuth := &git.WorktreeInfo{Path: "/ws/feat-auth", Branch: "feat/auth"}

	tests := []struct {
		name      string
		infos     []*git.WorktreeInfo
		target    string
		wantPath  string
		wantError string
	}{
		{
			name:  "unique name substring resolves",
			infos: []*git.WorktreeInfo{alpha, authFix}, target: "alph", wantPath: "/ws/alpha",
		},
		{
			name:  "unique branch substring resolves",
			infos: []*git.WorktreeInfo{alpha, featAuth}, target: "feat/", wantPath: "/ws/feat-auth",
		},
		{
			name:  "several matches list candidates in worktree order",
			infos: []*git.WorktreeInfo{authFix, featAuth}, target: "auth",
			wantError: `ambiguous target "auth": auth-fix, feat-auth`,
		},
		{
			name:  "detached placeholder branch never matches",
			infos: []*git.WorktreeInfo{alpha, detached}, target: "3fa",
			wantError: "worktree not found: 3fa",
		},
		{
			name:  "detached worktree still matches by directory name",
			infos: []*git.WorktreeInfo{alpha, detached}, target: "tm", wantPath: "/ws/tmp",
		},
		{
			name:  "detached placeholder does not make a name match ambiguous",
			infos: []*git.WorktreeInfo{alpha, detached}, target: "a", wantPath: "/ws/alpha",
		},
		{
			name:  "no match",
			infos: []*git.WorktreeInfo{alpha}, target: "missing",
			wantError: "worktree not found: missing",
		},
		{
			name:  "empty target",
			infos: []*git.WorktreeInfo{alpha}, target: "",
			wantError: "worktree not found: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			match, err := resolveBySubstring(tt.infos, tt.target)

			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("resolveBySubstring(%q) = %v, want error %q", tt.target, match, tt.wantError)
				}
				if err.Error() != tt.wantError {
					t.Errorf("resolveBySubstring(%q) error = %q, want %q", tt.target, err.Error(), tt.wantError)
				}
				return
			}

			if err != nil {
				t.Fatalf("resolveBySubstring(%q) unexpected error: %v", tt.target, err)
			}
			if match.Path != tt.wantPath {
				t.Errorf("resolveBySubstring(%q) path = %q, want %q", tt.target, match.Path, tt.wantPath)
			}
		})
	}
}
