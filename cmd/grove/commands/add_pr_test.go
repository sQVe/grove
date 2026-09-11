package commands

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/fs"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/github"
	"github.com/sqve/grove/internal/testutil"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

func TestRunAddFromPRRecordsNumberOnRefresh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell gh stub is not executable on Windows")
	}

	remote := testgit.NewTestRepo(t)
	remote.CreateBranch("Feature/topic.v2")
	remotePath := filepath.Join(remote.TempDir, "remote.git")
	remote.RunOutput("clone", "--bare", remote.Path, remotePath)
	bareDir := filepath.Join(remote.TempDir, "workspace.git")
	remote.RunOutput("clone", "--bare", remotePath, bareDir)
	worktreePath := filepath.Join(remote.TempDir, "existing")
	remote.RunOutput("-C", bareDir, "worktree", "add", worktreePath, "Feature/topic.v2")

	ghPath := filepath.Join(t.TempDir(), "gh")
	ghScript := `#!/bin/sh
if [ "$1" = auth ]; then exit 0; fi
printf '%s\n' '{"headRefName":"Feature/topic.v2","headRepository":{"name":"repo"},"headRepositoryOwner":{"login":"owner"}}'
`
	testutil.WriteFileMode(t, ghPath, ghScript, fs.FileExec)

	t.Setenv("PATH", filepath.Dir(ghPath)+string(os.PathListSeparator)+os.Getenv("PATH"))

	err := runAddFromPR("https://github.com/owner/repo/pull/42", false, false, "", bareDir, remote.TempDir, worktreePath, false, func() {})
	if err != nil {
		t.Fatal(err)
	}

	configs, err := git.GetBranchConfigs(bareDir, "grovePr")
	if err != nil {
		t.Fatal(err)
	}
	if configs["Feature/topic.v2"] != "42" {
		t.Fatalf("PR number = %q, want 42", configs["Feature/topic.v2"])
	}

	err = runAddFromPR("https://github.com/owner/repo/pull/42", false, true, "", bareDir, remote.TempDir, worktreePath, false, func() {})
	if err == nil || !strings.Contains(err.Error(), "worktree already exists") {
		t.Fatalf("expected --herdr to reject an existing PR worktree, got %v", err)
	}
}

func TestCheckoutPRRecordsNumber(t *testing.T) {
	for _, existingWorkspace := range []bool{true, false} {
		name := "clone"
		if existingWorkspace {
			name = "add"
		}
		t.Run(name, func(t *testing.T) {
			remote := testgit.NewTestRepo(t)
			remote.CreateBranch("Feature/topic.v2")
			remotePath := filepath.Join(remote.TempDir, "remote.git")
			remote.RunOutput("clone", "--bare", remote.Path, remotePath)
			bareDir := filepath.Join(remote.TempDir, "workspace.git")
			remote.RunOutput("clone", "--bare", remotePath, bareDir)
			worktreePath := filepath.Join(remote.TempDir, "pr-42")
			reference, err := github.ParsePRReference("https://github.com/owner/repo/pull/42")
			if err != nil {
				t.Fatal(err)
			}

			err = checkoutPR(bareDir, worktreePath, reference, &github.PRInfo{HeadRef: "Feature/topic.v2"}, true, false, existingWorkspace)
			if err != nil {
				t.Fatal(err)
			}

			number := strings.TrimSpace(remote.RunOutput("-C", bareDir, "config", "--get", "branch.Feature/topic.v2.grovePr"))
			if number != "42" {
				t.Fatalf("PR number = %q, want 42", number)
			}
		})
	}
}
