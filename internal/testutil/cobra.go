package testutil

import (
	"testing"

	"github.com/spf13/cobra"
)

// Ptr returns a pointer to the given string. Useful for AssertFlagExists.
func Ptr(s string) *string { return &s }

// AssertFlagExists fails if flag doesn't exist or has wrong properties.
// Pass nil for defValue to skip default check, or pointer to string to check (including empty).
// Pass empty string for valueType to skip type check.
// Pass empty string for shorthand to skip shorthand check.
func AssertFlagExists(t *testing.T, cmd *cobra.Command, name string, defValue *string, valueType, shorthand string) {
	t.Helper()
	flag := cmd.Flags().Lookup(name)
	if flag == nil {
		t.Fatalf("expected flag --%s to exist", name)
	}
	if defValue != nil && flag.DefValue != *defValue {
		t.Fatalf("flag --%s: expected default %q, got %q", name, *defValue, flag.DefValue)
	}
	if valueType != "" && flag.Value.Type() != valueType {
		t.Fatalf("flag --%s: expected type %q, got %q", name, valueType, flag.Value.Type())
	}
	if shorthand != "" && flag.Shorthand != shorthand {
		t.Fatalf("flag --%s: expected shorthand %q, got %q", name, shorthand, flag.Shorthand)
	}
}
