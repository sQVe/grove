package git

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/testutil"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

func TestHasNoBranches(t *testing.T) {
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

	t.Run("returns false when only remote-tracking refs exist", func(t *testing.T) {
		t.Parallel()
		w := testgit.NewGroveWorkspace(t)
		origin := testgit.NewTestRepo(t)
		w.RunOutput("remote", "add", "origin", origin.Path)
		w.RunOutput("fetch", "origin", "+refs/heads/main:refs/remotes/origin/main")
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
		bareDir := filepath.Join(testutil.TempDir(t), "empty.bare")
		if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create bare directory: %v", err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatalf("InitBare() error = %v", err)
		}

		var buf bytes.Buffer
		logger.SetOutput(&buf)
		defer logger.SetOutput(nil)
		logger.Init(true, false)

		base, err := ResolveWorktreeBase(bareDir, "", true)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != headRef {
			t.Fatalf("base = %q, want %s", base, headRef)
		}
		if buf.String() != "" {
			t.Fatalf("warning = %q, want none for an empty repository", buf.String())
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

	t.Run("fetches a qualified base whose tracking ref was pruned", func(t *testing.T) {
		w := testgit.NewGroveWorkspace(t)
		origin := testgit.NewTestRepo(t)
		origin.CreateBranch("develop")
		w.RunOutput("remote", "add", "origin", origin.Path)

		base, err := ResolveWorktreeBase(w.BareDir, "origin/develop", true)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != "origin/develop" {
			t.Fatalf("base = %q, want origin/develop", base)
		}
	})

	for _, staleLocal := range []bool{false, true} {
		name := "fetches an unqualified base whose tracking ref was pruned"
		if staleLocal {
			name = "prefers fetched unqualified base over stale local branch"
		}
		t.Run(name, func(t *testing.T) {
			w := testgit.NewGroveWorkspace(t)
			origin := testgit.NewTestRepo(t)
			origin.CreateBranch("develop")
			w.RunOutput("remote", "add", "origin", origin.Path)
			w.RunOutput("fetch", "origin", "+refs/heads/develop:refs/remotes/origin/develop")
			before := w.RunOutput("rev-parse", "origin/develop")
			if staleLocal {
				w.RunOutput("branch", "develop", "origin/develop")
			}
			w.RunOutput("update-ref", "-d", "refs/remotes/origin/develop")
			origin.RunOutput("checkout", "develop")
			origin.RunOutput("commit", "--allow-empty", "-m", "upstream")

			base, err := ResolveWorktreeBase(w.BareDir, "develop", true)
			if err != nil {
				t.Fatalf("ResolveWorktreeBase() error = %v", err)
			}
			if base != "origin/develop" {
				t.Fatalf("base = %q, want origin/develop", base)
			}
			if got := w.RunOutput("rev-parse", base); got == before {
				t.Fatal("base still points at the stale commit")
			}
			if staleLocal && w.RunOutput("rev-parse", "develop") != before {
				t.Fatal("fetch changed the local branch")
			}
		})
	}

	t.Run("returns an error for an unqualified base origin does not have", func(t *testing.T) {
		w := testgit.NewGroveWorkspace(t)
		origin := testgit.NewTestRepo(t)
		w.RunOutput("remote", "add", "origin", origin.Path)

		if _, err := ResolveWorktreeBase(w.BareDir, "nope", true); err == nil || err.Error() != `base branch "nope" does not exist` {
			t.Fatalf("error = %v, want missing-branch error", err)
		}
	})

	t.Run("returns an error for a qualified base origin does not have", func(t *testing.T) {
		w := testgit.NewGroveWorkspace(t)
		origin := testgit.NewTestRepo(t)
		w.RunOutput("remote", "add", "origin", origin.Path)

		if _, err := ResolveWorktreeBase(w.BareDir, "origin/nope", true); err == nil {
			t.Fatal("expected an error for a base that origin does not have")
		}
	})

	t.Run("keeps a local-only base literal without a fetch failure warning", func(t *testing.T) {
		w := testgit.NewGroveWorkspace(t)
		origin := testgit.NewTestRepo(t)
		w.RunOutput("remote", "add", "origin", origin.Path)
		w.RunOutput("branch", "local-only")

		var buf bytes.Buffer
		logger.SetOutput(&buf)
		defer logger.SetOutput(nil)
		logger.Init(true, false)

		base, err := ResolveWorktreeBase(w.BareDir, "local-only", true)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != "local-only" {
			t.Fatalf("base = %q, want local-only", base)
		}
		if strings.Contains(buf.String(), "fetch failed") {
			t.Fatalf("unexpected fetch failure warning: %s", buf.String())
		}
	})

	t.Run("checks the remote-tracking ref only after fetching", func(t *testing.T) {
		w := testgit.NewGroveWorkspace(t)
		origin := testgit.NewTestRepo(t)
		w.RunOutput("remote", "add", "origin", origin.Path)
		trace := filepath.Join(testutil.TempDir(t), "git-trace")
		t.Setenv("GIT_TRACE", trace)

		base, err := ResolveWorktreeBase(w.BareDir, "main", true)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != "origin/main" {
			t.Fatalf("base = %q, want origin/main", base)
		}
		output, err := os.ReadFile(trace) //nolint:gosec // Trace path is inside the test's temporary directory.
		if err != nil {
			t.Fatal(err)
		}
		if count := strings.Count(string(output), "git show-ref --verify --quiet refs/remotes/origin/main"); count != 1 {
			t.Fatalf("remote-tracking ref checks = %d, want 1\n%s", count, output)
		}
	})

	t.Run("returns an error when HEAD points at a deleted branch", func(t *testing.T) {
		w := testgit.NewGroveWorkspace(t)
		w.RunOutput("branch", "dev")
		w.RunOutput("worktree", "remove", "--force", filepath.Join(w.Dir, "main"))
		w.RunOutput("branch", "-D", "main")

		_, err := ResolveWorktreeBase(w.BareDir, "", false)
		if err == nil {
			t.Fatal("expected an error when HEAD points at a deleted branch")
		}
		if !strings.Contains(err.Error(), "--base") {
			t.Fatalf("error = %v, want it to suggest --base", err)
		}
	})

	t.Run("warns and returns HEAD when a detached HEAD has no default branch", func(t *testing.T) {
		w := testgit.NewGroveWorkspace(t)
		head := strings.TrimSpace(w.RunOutput("rev-parse", "HEAD"))
		w.RunOutput("worktree", "remove", "--force", filepath.Join(w.Dir, "main"))
		w.RunOutput("update-ref", "--no-deref", "HEAD", head)
		w.RunOutput("update-ref", "-d", "refs/heads/main")

		var buf bytes.Buffer
		logger.SetOutput(&buf)
		defer logger.SetOutput(nil)
		logger.Init(true, false)

		base, err := ResolveWorktreeBase(w.BareDir, "", false)
		if err != nil {
			t.Fatalf("ResolveWorktreeBase() error = %v", err)
		}
		if base != headRef {
			t.Fatalf("base = %q, want %s", base, headRef)
		}
		if !strings.Contains(buf.String(), "default branch unavailable") {
			t.Fatalf("warning = %q, want it to name the unavailable default branch", buf.String())
		}
	})

	t.Run("returns HEAD for an explicit HEAD base in an empty repository", func(t *testing.T) {
		bareDir := filepath.Join(testutil.TempDir(t), "empty.bare")
		if err := os.Mkdir(bareDir, fs.DirStrict); err != nil {
			t.Fatalf("failed to create bare directory: %v", err)
		}
		if err := InitBare(bareDir); err != nil {
			t.Fatalf("InitBare() error = %v", err)
		}

		base, err := ResolveWorktreeBase(bareDir, headRef, false)
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
