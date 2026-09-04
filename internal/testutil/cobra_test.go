package testutil

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestAssertFlagExists(t *testing.T) {
	makeCmd := func() *cobra.Command {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().StringP("name", "n", "default", "name flag")
		cmd.Flags().BoolP("verbose", "v", false, "verbose flag")
		cmd.Flags().IntP("count", "c", 42, "count flag")
		return cmd
	}

	t.Run("passes when flag exists with correct default and type", func(t *testing.T) {
		cmd := makeCmd()
		defaultValue := "default"

		AssertFlagExists(t, cmd, "name", &defaultValue, "string", "")
	})

	t.Run("skips default check when defValue is nil", func(t *testing.T) {
		cmd := makeCmd()

		AssertFlagExists(t, cmd, "name", nil, "string", "")
	})

	t.Run("skips type check when valueType is empty", func(t *testing.T) {
		cmd := makeCmd()
		defaultValue := "default"

		AssertFlagExists(t, cmd, "name", &defaultValue, "", "")
	})

	t.Run("checks empty string default value", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("empty", "", "flag with empty default")
		defaultValue := ""

		AssertFlagExists(t, cmd, "empty", &defaultValue, "string", "")
	})

	t.Run("works with bool flag", func(t *testing.T) {
		cmd := makeCmd()
		defaultValue := "false"

		AssertFlagExists(t, cmd, "verbose", &defaultValue, "bool", "")
	})

	t.Run("works with int flag", func(t *testing.T) {
		cmd := makeCmd()
		defaultValue := "42"

		AssertFlagExists(t, cmd, "count", &defaultValue, "int", "")
	})

	t.Run("skips all optional checks", func(t *testing.T) {
		cmd := makeCmd()

		AssertFlagExists(t, cmd, "name", nil, "", "")
	})

	t.Run("checks shorthand", func(t *testing.T) {
		cmd := makeCmd()

		AssertFlagExists(t, cmd, "name", nil, "", "n")
	})

	t.Run("skips shorthand check when empty", func(t *testing.T) {
		cmd := makeCmd()

		AssertFlagExists(t, cmd, "name", nil, "", "")
	})
}
