package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/testutil"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

func TestHasNoBranches(t *testing.T) {
	t.Parallel()

	t.Run("returns true for a repository without branches", func(t *testing.T) {
		t.Parallel()
		bareDir := filepath.Join(testutil.TempDir(t), "empty.bare")
		if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create bare directory: %v", err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatalf("InitBare() error = %v", err)
		}

		empty, err := hasNoBranches(bareDir)
		if err != nil {
			t.Fatalf("hasNoBranches() error = %v", err)
		}
		if !empty {
			t.Fatal("hasNoBranches() = false, want true")
		}
	})

	t.Run("returns false when HEAD dangles over an existing branch", func(t *testing.T) {
		t.Parallel()
		w := testgit.NewGroveWorkspace(t)
		w.RunOutput("branch", "dev")
		w.RunOutput("worktree", "remove", "--force", filepath.Join(w.Dir, "main"))
		w.RunOutput("branch", "-D", "main")

		empty, err := hasNoBranches(w.BareDir)
		if err != nil {
			t.Fatalf("hasNoBranches() error = %v", err)
		}
		if empty {
			t.Fatal("hasNoBranches() = true, want false")
		}
	})
}

func TestResolveWorktreeBase(t *testing.T) {
	t.Parallel()

	t.Run("returns the remote-tracking ref when it exists", func(t *testing.T) {
		t.Parallel()
		w := testgit.NewGroveWorkspace(t)
		origin := testgit.NewTestRepo(t)
		w.RunOutput("remote", "add", "origin", origin.Path)
		w.RunOutput("fetch", "origin", "+refs/heads/main:refs/remotes/origin/main")

		base, err := ResolveWorktreeBase(w.BareDir, "", false)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != "origin/main" {
			t.Fatalf("base = %q, want origin/main", base)
		}
	})

	t.Run("falls back to the local default branch without a remote-tracking ref", func(t *testing.T) {
		t.Parallel()
		w := testgit.NewGroveWorkspace(t)

		base, err := ResolveWorktreeBase(w.BareDir, "", false)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != "main" {
			t.Fatalf("base = %q, want main", base)
		}
	})

	t.Run("returns HEAD for a repository without commits", func(t *testing.T) {
		t.Parallel()
		bareDir := filepath.Join(testutil.TempDir(t), "empty.bare")
		if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create bare directory: %v", err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatalf("InitBare() error = %v", err)
		}

		base, err := ResolveWorktreeBase(bareDir, "", true)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != headRef {
			t.Fatalf("base = %q, want %s", base, headRef)
		}
	})

	t.Run("resolves an explicit base to its origin counterpart", func(t *testing.T) {
		t.Parallel()
		w := testgit.NewGroveWorkspace(t)
		origin := testgit.NewTestRepo(t)
		w.RunOutput("remote", "add", "origin", origin.Path)
		w.RunOutput("fetch", "origin", "+refs/heads/main:refs/remotes/origin/main")

		base, err := ResolveWorktreeBase(w.BareDir, "main", false)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != "origin/main" {
			t.Fatalf("base = %q, want origin/main", base)
		}
	})

	t.Run("keeps a local-only base literal", func(t *testing.T) {
		t.Parallel()
		w := testgit.NewGroveWorkspace(t)
		w.RunOutput("branch", "local-only")

		base, err := ResolveWorktreeBase(w.BareDir, "local-only", true)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != "local-only" {
			t.Fatalf("base = %q, want local-only", base)
		}
	})

	t.Run("returns HEAD when no default branch resolves", func(t *testing.T) {
		t.Parallel()
		w := testgit.NewGroveWorkspace(t)
		w.RunOutput("branch", "dev")
		w.RunOutput("worktree", "remove", "--force", filepath.Join(w.Dir, "main"))
		w.RunOutput("branch", "-D", "main")

		base, err := ResolveWorktreeBase(w.BareDir, "", false)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != headRef {
			t.Fatalf("base = %q, want %s", base, headRef)
		}
	})

	t.Run("returns an error for a base that does not exist", func(t *testing.T) {
		t.Parallel()
		w := testgit.NewGroveWorkspace(t)

		if _, err := ResolveWorktreeBase(w.BareDir, "missing", true); err == nil {
			t.Fatal("expected an error for a missing base branch")
		}
	})
}
