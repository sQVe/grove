package commands

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/github"
)

func TestNewCloneCmd(t *testing.T) {
	cmd := NewCloneCmd()

	if cmd.Use != "clone <url|PR-URL> [directory]" {
		t.Errorf("expected Use 'clone <url|PR-URL> [directory]', got '%s'", cmd.Use)
	}

	if cmd.Flags().Lookup("branches") == nil {
		t.Error("expected --branches flag")
	}
	verboseFlag := cmd.Flags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatal("expected --verbose flag")
	}
	if verboseFlag.Shorthand != "v" {
		t.Errorf("expected verbose shorthand 'v', got %q", verboseFlag.Shorthand)
	}
	if cmd.Flags().Lookup("shallow") == nil {
		t.Error("expected --shallow flag")
	}
}

func TestNewCloneCmd_PreRunE(t *testing.T) {
	t.Run("rejects branches flag without URL", func(t *testing.T) {
		cmd := NewCloneCmd()
		_ = cmd.Flags().Set("branches", "main,develop")

		err := cmd.PreRunE(cmd, []string{})
		if err == nil {
			t.Error("expected error when --branches used without URL")
		}
	})

	t.Run("accepts branches flag with URL", func(t *testing.T) {
		cmd := NewCloneCmd()
		_ = cmd.Flags().Set("branches", "main,develop")

		err := cmd.PreRunE(cmd, []string{"https://github.com/owner/repo"})
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestNewCloneCmd_RunE_Validation(t *testing.T) {
	// These tests simulate edge cases where --branches is explicitly provided
	// but with empty/invalid values. We must manually set Changed=true because
	// flag.Set("", value) doesn't mark the flag as changed when value is empty.

	t.Run("rejects empty branches value", func(t *testing.T) {
		cmd := NewCloneCmd()
		_ = cmd.Flags().Set("branches", "")
		cmd.Flags().Lookup("branches").Changed = true

		err := cmd.RunE(cmd, []string{"https://github.com/owner/repo"})
		if err == nil {
			t.Fatal("expected error for empty branches value")
		}
		if err.Error() != "no branches specified" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("rejects quoted empty branches value", func(t *testing.T) {
		cmd := NewCloneCmd()
		_ = cmd.Flags().Set("branches", `""`)
		cmd.Flags().Lookup("branches").Changed = true

		err := cmd.RunE(cmd, []string{"https://github.com/owner/repo"})
		if err == nil {
			t.Fatal("expected error for quoted empty branches value")
		}
		if err.Error() != "no branches specified" {
			t.Errorf("unexpected error message: %v", err)
		}
	})
}

func TestNewCloneCmd_ValidArgsFunction(t *testing.T) {
	cmd := NewCloneCmd()

	t.Run("returns no file completion for first arg (URL)", func(t *testing.T) {
		completions, directive := cmd.ValidArgsFunction(cmd, []string{}, "")
		if completions != nil {
			t.Errorf("expected nil completions for first arg, got %v", completions)
		}
		if directive != cobra.ShellCompDirectiveNoFileComp {
			t.Errorf("expected ShellCompDirectiveNoFileComp, got %v", directive)
		}
	})

	t.Run("returns directory filtering for second arg", func(t *testing.T) {
		completions, directive := cmd.ValidArgsFunction(cmd, []string{"url"}, "")
		if completions != nil {
			t.Errorf("expected nil completions for second arg, got %v", completions)
		}
		if directive != cobra.ShellCompDirectiveFilterDirs {
			t.Errorf("expected ShellCompDirectiveFilterDirs, got %v", directive)
		}
	})
}

func TestGitHubURLClassification(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectGitHub   bool
		expectPR       bool
		expectFallback bool
	}{
		{"GitHub HTTPS URL", "https://github.com/owner/repo", true, false, false},
		{"GitHub SSH URL", "git@github.com:owner/repo.git", true, false, false},
		{"PR URL", "https://github.com/owner/repo/pull/123", false, true, false},
		{"GitLab URL", "https://gitlab.com/owner/repo", false, false, true},
		{"Self-hosted URL", "https://git.company.com/owner/repo", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isPR := github.IsPRURL(tt.url)
			isGitHub := github.IsGitHubURL(tt.url)
			isFallback := !isPR && !isGitHub

			if isPR != tt.expectPR {
				t.Errorf("IsPRURL(%q) = %v, want %v", tt.url, isPR, tt.expectPR)
			}
			if isGitHub != tt.expectGitHub {
				t.Errorf("IsGitHubURL(%q) = %v, want %v", tt.url, isGitHub, tt.expectGitHub)
			}
			if isFallback != tt.expectFallback {
				t.Errorf("fallback for %q = %v, want %v", tt.url, isFallback, tt.expectFallback)
			}
		})
	}
}
