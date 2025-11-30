package commands

import "testing"

func TestNewCloneCmd(t *testing.T) {
	cmd := NewCloneCmd()

	if cmd.Use != "clone <url|PR-URL> [directory]" {
		t.Errorf("expected Use 'clone <url|PR-URL> [directory]', got '%s'", cmd.Use)
	}

	if cmd.Flags().Lookup("branches") == nil {
		t.Error("expected --branches flag")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag")
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
	t.Run("rejects empty branches value", func(t *testing.T) {
		cmd := NewCloneCmd()
		_ = cmd.Flags().Set("branches", "")
		// Mark the flag as changed to trigger validation
		cmd.Flags().Lookup("branches").Changed = true

		err := cmd.RunE(cmd, []string{"https://github.com/owner/repo"})
		if err == nil {
			t.Error("expected error for empty branches value")
		}
	})
}
