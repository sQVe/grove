package git

import (
	"os"
	"os/exec"
	"testing"

	testgit "github.com/sqve/grove/internal/testutil/git"
)

func TestGetConfigs(t *testing.T) {
	t.Run("only matches keys starting with exact prefix", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		// Change to repo directory so GetConfigs finds the local config
		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		if err := os.Chdir(repo.Path); err != nil {
			t.Fatalf("failed to change to repo directory: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		// Set up config keys: one with exact prefix, one that would match unescaped regex
		cmd := exec.Command("git", "config", "grove.plain", "true")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set grove.plain: %v", err)
		}

		// "groveX.foo" should NOT match "grove." prefix, but would if dot is unescaped
		cmd = exec.Command("git", "config", "groveX.foo", "bar")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to set groveX.foo: %v", err)
		}

		// GetConfigs with "grove." should only return grove.* keys, not groveX.*
		configs, err := GetConfigs("grove.", false)
		if err != nil {
			t.Fatalf("GetConfigs failed: %v", err)
		}

		// Should have grove.plain but NOT groveX.foo
		if _, ok := configs["grove.plain"]; !ok {
			t.Error("expected grove.plain to be in results")
		}
		// git normalizes keys to lowercase, so check for grovex.foo
		if _, ok := configs["grovex.foo"]; ok {
			t.Error("grovex.foo should NOT match prefix 'grove.' - regex dot was not escaped")
		}
	})

	t.Run("returns empty map when no keys match", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		oldWd, err := os.Getwd()
		if err != nil {
			t.Fatalf("failed to get working directory: %v", err)
		}
		if err := os.Chdir(repo.Path); err != nil {
			t.Fatalf("failed to change to repo directory: %v", err)
		}
		defer func() { _ = os.Chdir(oldWd) }()

		configs, err := GetConfigs("nonexistent.", false)
		if err != nil {
			t.Fatalf("GetConfigs failed: %v", err)
		}
		if len(configs) != 0 {
			t.Errorf("expected empty map, got %d entries", len(configs))
		}
	})
}
