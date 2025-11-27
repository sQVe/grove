package commands

import (
	"testing"
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
