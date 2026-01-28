package testutil

import (
	"testing"

	"github.com/spf13/cobra"
)

// AssertFlagExists fails if flag doesn't exist or has wrong properties.
// Pass nil for defValue to skip default check, or pointer to string to check (including empty).
// Pass empty string for valueType to skip type check.
func AssertFlagExists(t *testing.T, cmd *cobra.Command, name string, defValue *string, valueType string) {
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
}
