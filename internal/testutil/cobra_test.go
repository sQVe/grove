package testutil

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAssertFlagExists(t *testing.T) {
	// Helper to create a command with test flags
	makeCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("name", "default", "name flag")
		cmd.Flags().Bool("verbose", false, "verbose flag")
		cmd.Flags().Int("count", 42, "count flag")
		return cmd
	}

	t.Run("passes when flag exists with correct default and type", func(t *testing.T) {
		cmd := makeCmd()
		defValue := "default"

		AssertFlagExists(t, cmd, "name", &defValue, "string")
	})

	t.Run("skips default check when defValue is nil", func(t *testing.T) {
		cmd := makeCmd()

		// This should pass even though we don't check the default
		AssertFlagExists(t, cmd, "name", nil, "string")
	})

	t.Run("skips type check when valueType is empty", func(t *testing.T) {
		cmd := makeCmd()
		defValue := "default"

		// This should pass even though we don't check the type
		AssertFlagExists(t, cmd, "name", &defValue, "")
	})

	t.Run("checks empty string default value", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("empty", "", "flag with empty default")
		defValue := ""

		AssertFlagExists(t, cmd, "empty", &defValue, "string")
	})

	t.Run("works with bool flag", func(t *testing.T) {
		cmd := makeCmd()
		defValue := "false"

		AssertFlagExists(t, cmd, "verbose", &defValue, "bool")
	})

	t.Run("works with int flag", func(t *testing.T) {
		cmd := makeCmd()
		defValue := "42"

		AssertFlagExists(t, cmd, "count", &defValue, "int")
	})

	t.Run("skips both checks", func(t *testing.T) {
		cmd := makeCmd()

		// Only checks that the flag exists
		AssertFlagExists(t, cmd, "name", nil, "")
	})
}
