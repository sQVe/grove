package commands

import (
	"testing"
)

func TestNewListCmd(t *testing.T) {
	cmd := NewListCmd()

	if cmd.Use != "list" {
		t.Errorf("expected Use 'list', got '%s'", cmd.Use)
	}

	// Check flags exist
	if cmd.Flags().Lookup("fast") == nil {
		t.Error("expected --fast flag")
	}
	if cmd.Flags().Lookup("json") == nil {
		t.Error("expected --json flag")
	}
	if cmd.Flags().Lookup("verbose") == nil {
		t.Error("expected --verbose flag")
	}
}
