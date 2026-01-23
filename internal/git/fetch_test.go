package git

import (
	"os/exec"
	"testing"

	testgit "github.com/sqve/grove/internal/testutil/git"
)

func TestDetectRefChanges(t *testing.T) {
	tests := []struct {
		name   string
		before map[string]string
		after  map[string]string
		want   []RefChange
	}{
		{
			name:   "new ref",
			before: map[string]string{},
			after:  map[string]string{"refs/remotes/origin/main": "abc123"},
			want: []RefChange{
				{RefName: "refs/remotes/origin/main", OldHash: "", NewHash: "abc123", Type: New},
			},
		},
		{
			name:   "updated ref",
			before: map[string]string{"refs/remotes/origin/main": "abc123"},
			after:  map[string]string{"refs/remotes/origin/main": "def456"},
			want: []RefChange{
				{RefName: "refs/remotes/origin/main", OldHash: "abc123", NewHash: "def456", Type: Updated},
			},
		},
		{
			name:   "pruned ref",
			before: map[string]string{"refs/remotes/origin/main": "abc123"},
			after:  map[string]string{},
			want: []RefChange{
				{RefName: "refs/remotes/origin/main", OldHash: "abc123", NewHash: "", Type: Pruned},
			},
		},
		{
			name:   "unchanged ref",
			before: map[string]string{"refs/remotes/origin/main": "abc123"},
			after:  map[string]string{"refs/remotes/origin/main": "abc123"},
			want:   nil,
		},
		{
			name: "mixed changes",
			before: map[string]string{
				"refs/remotes/origin/main":    "aaa111",
				"refs/remotes/origin/old":     "bbb222",
				"refs/remotes/origin/feature": "ccc333",
			},
			after: map[string]string{
				"refs/remotes/origin/main":    "aaa111",
				"refs/remotes/origin/feature": "ddd444",
				"refs/remotes/origin/new":     "eee555",
			},
			want: []RefChange{
				{RefName: "refs/remotes/origin/feature", OldHash: "ccc333", NewHash: "ddd444", Type: Updated},
				{RefName: "refs/remotes/origin/new", OldHash: "", NewHash: "eee555", Type: New},
				{RefName: "refs/remotes/origin/old", OldHash: "bbb222", NewHash: "", Type: Pruned},
			},
		},
		{
			name:   "empty before and after",
			before: map[string]string{},
			after:  map[string]string{},
			want:   nil,
		},
		{
			name:   "multiple new refs",
			before: map[string]string{},
			after: map[string]string{
				"refs/remotes/origin/main":    "aaa111",
				"refs/remotes/origin/feature": "bbb222",
			},
			want: []RefChange{
				{RefName: "refs/remotes/origin/feature", OldHash: "", NewHash: "bbb222", Type: New},
				{RefName: "refs/remotes/origin/main", OldHash: "", NewHash: "aaa111", Type: New},
			},
		},
		{
			name: "all refs pruned",
			before: map[string]string{
				"refs/remotes/origin/main":    "aaa111",
				"refs/remotes/origin/feature": "bbb222",
			},
			after: map[string]string{},
			want: []RefChange{
				{RefName: "refs/remotes/origin/feature", OldHash: "bbb222", NewHash: "", Type: Pruned},
				{RefName: "refs/remotes/origin/main", OldHash: "aaa111", NewHash: "", Type: Pruned},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectRefChanges(tt.before, tt.after)

			if len(got) != len(tt.want) {
				t.Fatalf("DetectRefChanges() returned %d changes, want %d\ngot: %+v\nwant: %+v", len(got), len(tt.want), got, tt.want)
			}

			gotMap := make(map[string]RefChange)
			for _, change := range got {
				gotMap[change.RefName] = change
			}

			for _, want := range tt.want {
				gotChange, ok := gotMap[want.RefName]
				if !ok {
					t.Errorf("DetectRefChanges() missing change for ref %q", want.RefName)
					continue
				}
				if gotChange.OldHash != want.OldHash {
					t.Errorf("DetectRefChanges() ref %q OldHash = %q, want %q", want.RefName, gotChange.OldHash, want.OldHash)
				}
				if gotChange.NewHash != want.NewHash {
					t.Errorf("DetectRefChanges() ref %q NewHash = %q, want %q", want.RefName, gotChange.NewHash, want.NewHash)
				}
				if gotChange.Type != want.Type {
					t.Errorf("DetectRefChanges() ref %q Type = %v, want %v", want.RefName, gotChange.Type, want.Type)
				}
			}
		})
	}
}

func TestGetRemoteRefs(t *testing.T) {
	t.Run("valid remote returns refs", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		bareRepo := testgit.NewTestRepo(t)
		cmd := exec.Command("git", "config", "--bool", "core.bare", "true")
		cmd.Dir = bareRepo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to configure bare repo: %v", err)
		}

		repo.AddRemote("origin", bareRepo.Path)
		cmd = exec.Command("git", "fetch", "origin")
		cmd.Dir = repo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to fetch: %v", err)
		}

		refs, err := GetRemoteRefs(repo.Path, "origin")
		if err != nil {
			t.Fatalf("GetRemoteRefs() error = %v", err)
		}

		if len(refs) == 0 {
			t.Error("GetRemoteRefs() returned no refs, expected at least one")
		}

		found := false
		for refName := range refs {
			if refName == "refs/remotes/origin/main" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("GetRemoteRefs() missing refs/remotes/origin/main, got: %v", refs)
		}
	})

	t.Run("missing remote returns empty map", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		refs, err := GetRemoteRefs(repo.Path, "nonexistent")
		if err != nil {
			t.Fatalf("GetRemoteRefs() error = %v, want nil", err)
		}
		if len(refs) != 0 {
			t.Errorf("GetRemoteRefs() = %v, want empty map", refs)
		}
	})
}

func TestFetchRemote(t *testing.T) {
	t.Run("successful fetch", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		bareRepo := testgit.NewTestRepo(t)
		cmd := exec.Command("git", "config", "--bool", "core.bare", "true")
		cmd.Dir = bareRepo.Path
		if err := cmd.Run(); err != nil {
			t.Fatalf("failed to configure bare repo: %v", err)
		}

		repo.AddRemote("origin", bareRepo.Path)

		err := FetchRemote(repo.Path, "origin")
		if err != nil {
			t.Errorf("FetchRemote() error = %v, want nil", err)
		}
	})

	t.Run("non-existent remote fails", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		err := FetchRemote(repo.Path, "nonexistent")
		if err == nil {
			t.Error("FetchRemote() error = nil, want error for non-existent remote")
		}
	})

	t.Run("empty repo path fails", func(t *testing.T) {
		err := FetchRemote("", "origin")
		if err == nil {
			t.Error("FetchRemote() error = nil, want error for empty repo path")
		}
	})

	t.Run("empty remote name fails", func(t *testing.T) {
		repo := testgit.NewTestRepo(t)

		err := FetchRemote(repo.Path, "")
		if err == nil {
			t.Error("FetchRemote() error = nil, want error for empty remote name")
		}
	})
}

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		ct   ChangeType
		want string
	}{
		{New, "new"},
		{Updated, "updated"},
		{Pruned, "pruned"},
		{ChangeType(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.ct.String(); got != tt.want {
				t.Errorf("ChangeType.String() = %q, want %q", got, tt.want)
			}
		})
	}
}
