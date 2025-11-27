package commands

import (
	"testing"
)

func TestNewSwitchCmd(t *testing.T) {
	cmd := NewSwitchCmd()
	if cmd.Use != "switch <branch>" {
		t.Errorf("expected Use to be 'switch <branch>', got %q", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("expected Short description to be set")
	}
}
