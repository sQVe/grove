package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetBranchCompletions(t *testing.T) {
	tests := []struct {
		name           string
		toComplete     string
		remoteBranches []string
		want           []string
	}{
		{
			name:           "first branch partial match",
			toComplete:     "fea",
			remoteBranches: []string{"main", "feature", "feature-2", "develop"},
			want:           []string{"feature", "feature-2"},
		},
		{
			name:           "second branch partial match",
			toComplete:     "main,fea",
			remoteBranches: []string{"main", "feature", "feature-2", "develop"},
			want:           []string{"main,feature", "main,feature-2"},
		},
		{
			name:           "trailing comma shows all non-selected",
			toComplete:     "main,",
			remoteBranches: []string{"main", "feature", "develop"},
			want:           []string{"main,feature", "main,develop"},
		},
		{
			name:           "no duplicates allowed",
			toComplete:     "main,mai",
			remoteBranches: []string{"main", "maintenance"},
			want:           []string{"main,maintenance"},
		},
		{
			name:           "whitespace handling",
			toComplete:     "main, fea",
			remoteBranches: []string{"main", "feature", "fix"},
			want:           []string{"main,feature"},
		},
		{
			name:           "empty input lists all",
			toComplete:     "",
			remoteBranches: []string{"main", "feature"},
			want:           []string{"main", "feature"},
		},
		{
			name:           "double comma handling",
			toComplete:     "main,,fea",
			remoteBranches: []string{"main", "feature"},
			want:           []string{"main,feature"},
		},
		{
			name:           "no matches",
			toComplete:     "main,xyz",
			remoteBranches: []string{"main", "feature"},
			want:           []string{},
		},
		{
			name:           "multiple selected branches",
			toComplete:     "main,feature,dev",
			remoteBranches: []string{"main", "feature", "develop", "development"},
			want:           []string{"main,feature,develop", "main,feature,development"},
		},
		{
			name:           "leading space on last part",
			toComplete:     "main, ",
			remoteBranches: []string{"main", "feature", "develop"},
			want:           []string{"main,feature", "main,develop"},
		},
		{
			name:           "single character completion",
			toComplete:     "main,f",
			remoteBranches: []string{"main", "feature", "fix", "foo"},
			want:           []string{"main,feature", "main,fix", "main,foo"},
		},
		{
			name:           "exact match should still complete",
			toComplete:     "main,feature",
			remoteBranches: []string{"main", "feature", "feature-branch"},
			want:           []string{"main,feature", "main,feature-branch"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBranchCompletions(tt.toComplete, tt.remoteBranches)

			if len(got) != len(tt.want) {
				t.Errorf("got %d completions, want %d\nGot: %v\nWant: %v", len(got), len(tt.want), got, tt.want)
				return
			}

			for i, gotCompletion := range got {
				if gotCompletion != tt.want[i] {
					t.Errorf("completion[%d] = %q, want %q", i, gotCompletion, tt.want[i])
				}
			}
		})
	}
}

func TestResolveTargetDirectory(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get current directory: %v", err)
	}

	relativePathAbs, err := filepath.Abs("relative/path")
	if err != nil {
		t.Fatalf("failed to get absolute path: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		argIndex int
		want     string
		wantErr  bool
	}{
		{
			name:     "no args returns current directory",
			args:     []string{},
			argIndex: 0,
			want:     cwd,
			wantErr:  false,
		},
		{
			name:     "arg index beyond length returns current directory",
			args:     []string{"test"},
			argIndex: 1,
			want:     cwd,
			wantErr:  false,
		},
		{
			name:     "relative path gets absolute",
			args:     []string{"relative/path"},
			argIndex: 0,
			want:     relativePathAbs,
			wantErr:  false,
		},
		{
			name:     "absolute path returns as-is",
			args:     []string{"/absolute/path"},
			argIndex: 0,
			want:     "/absolute/path",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveTargetDirectory(tt.args, tt.argIndex)

			if (err != nil) != tt.wantErr {
				t.Errorf("resolveTargetDirectory() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got != tt.want {
				t.Errorf("resolveTargetDirectory() = %v, want %v", got, tt.want)
			}
		})
	}
}
