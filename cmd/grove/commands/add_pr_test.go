package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/sqve/grove/internal/github"
	testgit "github.com/sqve/grove/internal/testutil/git"
)

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
