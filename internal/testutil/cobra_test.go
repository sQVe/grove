package testutil

import (
	"testing"

	"github.com/spf13/cobra"
)

func TestPtr(t *testing.T) {
	t.Run("returns pointer to string", func(t *testing.T) {
		p := Ptr("test")
		if *p != "test" {
			t.Errorf("expected 'test', got %q", *p)
		}
	})

	t.Run("returns pointer to empty string", func(t *testing.T) {
		p := Ptr("")
		if *p != "" {
			t.Errorf("expected empty string, got %q", *p)
		}
	})
}

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

		AssertFlagExists(t, cmd, "name", Ptr("default"), "string", "")
	})

	t.Run("skips default check when defValue is nil", func(t *testing.T) {
		cmd := makeCmd()

		AssertFlagExists(t, cmd, "name", nil, "string", "")
	})

	t.Run("skips type check when valueType is empty", func(t *testing.T) {
		cmd := makeCmd()

		AssertFlagExists(t, cmd, "name", Ptr("default"), "", "")
	})

	t.Run("checks empty string default value", func(t *testing.T) {
		cmd := &cobra.Command{Use: "test"}
		cmd.Flags().String("empty", "", "flag with empty default")

		AssertFlagExists(t, cmd, "empty", Ptr(""), "string", "")
	})

	t.Run("works with bool flag", func(t *testing.T) {
		cmd := makeCmd()

		AssertFlagExists(t, cmd, "verbose", Ptr("false"), "bool", "")
	})

	t.Run("works with int flag", func(t *testing.T) {
		cmd := makeCmd()

		AssertFlagExists(t, cmd, "count", Ptr("42"), "int", "")
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
