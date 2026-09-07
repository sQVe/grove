//go:build integration

package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	gitutil "github.com/sqve/grove/internal/testutil/git"
)

func TestAdd(t *testing.T) {
	t.Parallel()
	t.Run("returns a worktree when the base fetch times out", func(t *testing.T) {
		t.Parallel()
		origin := gitutil.NewTestRepo(t)
		w := gitutil.NewGroveWorkspace(t)
		w.RunOutput("remote", "add", "origin", "file://"+origin.Path)
		w.RunOutput("fetch", "origin", "refs/heads/main:refs/remotes/origin/main")
		w.RunOutput("config", "grove.timeout", "30s")
		w.RunOutput("config", "grove.fetchBase", "true")
		// Stall the real upload-pack process, leaving git fetch waiting for a response.
		w.RunOutput("config", "remote.origin.uploadpack", "sleep 10; git-upload-pack")
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, "grove", "add", "bounded-fetch")
		cmd.Dir = w.BareDir
		cmd.WaitDelay = time.Second
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("add failed: %v\n%s", err, out)
		}
		if strings.Count(string(out), "basing on origin/main from ") != 1 || !strings.Contains(string(out), "ago - fetch failed") {
			t.Fatalf("expected one warning with base and age, got %s", out)
		}
		if _, err := os.Stat(filepath.Join(w.Dir, "bounded-fetch", ".git")); err != nil {
			t.Fatal(err)
		}
		if got, want := w.RunOutput("rev-parse", "bounded-fetch"), w.RunOutput("rev-parse", "origin/main"); got != want {
			t.Fatalf("worktree commit = %s, want %s", got, want)
		}
	})
}
