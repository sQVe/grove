package git

import (
	"os"
	"os/exec"
	"testing"

	testgit "github.com/sqve/grove/internal/testutil/git"
)

func TestBranchConfigs(t *testing.T) {
	repo := testgit.NewTestRepo(t)
	bareDir := repo.TempDir + "/bare.git"
	repo.RunOutput("clone", "--bare", repo.Path, bareDir)
	t.Chdir(repo.Path)

	t.Run("returns an empty map when no keys match", func(t *testing.T) {
		configs, err := GetBranchConfigs(bareDir, "grovePr")

		if err != nil || len(configs) != 0 {
			t.Fatalf("empty configs = %v, %v", configs, err)
		}
	})

	t.Run("overwrites values and preserves branch names", func(t *testing.T) {
		for _, branch := range []string{"Feature/topic.v2", "main"} {
			if err := SetBranchConfig(bareDir, branch, "grovePr", "41"); err != nil {
				t.Fatal(err)
			}
			if err := SetBranchConfig(bareDir, branch, "grovePr", "42"); err != nil {
				t.Fatal(err)
			}
		}

		configs, err := GetBranchConfigs(bareDir, "grovePr")
		if err != nil {
			t.Fatal(err)
		}

		if len(configs) != 2 || configs["Feature/topic.v2"] != "42" || configs["main"] != "42" {
			t.Fatalf("configs = %v", configs)
		}
	})

	t.Run("excludes other keys", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)
		repo.RunOutput("config", "branch.main.grovePrExtra", "99")

		configs, err := GetBranchConfigs(repo.Path, "grovePr")

		if err != nil || len(configs) != 0 {
			t.Fatalf("configs = %v, %v", configs, err)
		}
	})

	t.Run("reads and writes the requested repository regardless of cwd", func(t *testing.T) {
		repo.RunOutput("config", "branch.main.grovePr", "100")
		if err := SetBranchConfig(bareDir, "main", "grovePr", "42"); err != nil {
			t.Fatal(err)
		}

		configs, err := GetBranchConfigs(bareDir, "grovePr")
		if err != nil {
			t.Fatal(err)
		}

		if configs["main"] != "42" {
			t.Fatalf("configs = %v", configs)
		}
		if value := repo.RunOutput("config", "branch.main.grovePr"); value != "100\n" {
			t.Fatalf("cwd config = %q, want 100", value)
		}
	})

	t.Run("reports read failures", func(t *testing.T) {
		_, err := GetBranchConfigs(bareDir+"/missing", "grovePr")

		if err == nil {
			t.Fatal("expected read failure")
		}
	})

	t.Run("reports write failures", func(t *testing.T) {
		err := SetBranchConfig(bareDir+"/missing", "main", "grovePr", "42")

		if err == nil {
			t.Fatal("expected write failure")
		}
	})
}

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
