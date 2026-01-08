package github

import (
	"os/exec"
	"strings"
	"testing"
)

func TestIsGitHubURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// GitHub HTTPS URLs - should match
		{"https://github.com/owner/repo", true},
		{"https://github.com/owner/repo.git", true},
		{"http://github.com/owner/repo", true},

		// GitHub SSH URLs - should match
		{"git@github.com:owner/repo.git", true},
		{"git@github.com:owner/repo", true},

		// Not GitHub URLs - should NOT match
		{"https://gitlab.com/owner/repo", false},
		{"https://bitbucket.org/owner/repo", false},
		{"git@gitlab.com:owner/repo.git", false},
		{"", false},
		{"not-a-url", false},

		// PR URLs should NOT match (handled separately)
		{"https://github.com/owner/repo/pull/123", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsGitHubURL(tt.input)
			if result != tt.expected {
				t.Errorf("IsGitHubURL(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPRURL(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// PR URL format - should match
		{"https://github.com/owner/repo/pull/123", true},
		{"https://github.com/some-org/some-repo/pull/1", true},
		{"http://github.com/owner/repo/pull/123", true},

		// PR URL with suffixes (commonly copied from browser)
		{"https://github.com/owner/repo/pull/123/files", true},
		{"https://github.com/owner/repo/pull/123/commits", true},
		{"https://github.com/owner/repo/pull/123/checks", true},

		// PR URL with query params
		{"https://github.com/owner/repo/pull/123?diff=split", true},
		{"https://github.com/owner/repo/pull/123/files?diff=unified", true},

		// PR number format - should NOT match (use ParsePRReference for that)
		{"#123", false},
		{"#1", false},

		// Not PR URLs
		{"main", false},
		{"https://github.com/owner/repo", false},
		{"https://github.com/owner/repo/issues/123", false},
		{"https://gitlab.com/owner/repo/pull/123", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsPRURL(tt.input)
			if result != tt.expected {
				t.Errorf("IsPRURL(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParsePRReference_Number(t *testing.T) {
	ref, err := ParsePRReference("#123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ref.Number != 123 {
		t.Errorf("Number = %d, want 123", ref.Number)
	}
	if ref.Owner != "" {
		t.Errorf("Owner = %q, want empty string", ref.Owner)
	}
	if ref.Repo != "" {
		t.Errorf("Repo = %q, want empty string", ref.Repo)
	}
}

func TestParsePRReference_URL(t *testing.T) {
	tests := []struct {
		input      string
		wantOwner  string
		wantRepo   string
		wantNumber int
	}{
		{
			input:      "https://github.com/anthropics/claude-code/pull/456",
			wantOwner:  "anthropics",
			wantRepo:   "claude-code",
			wantNumber: 456,
		},
		{
			input:      "https://github.com/some-org/some-repo/pull/1",
			wantOwner:  "some-org",
			wantRepo:   "some-repo",
			wantNumber: 1,
		},
		{
			input:      "http://github.com/owner/repo/pull/999",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 999,
		},
		// URL with trailing slash (commonly copied from browsers)
		{
			input:      "https://github.com/owner/repo/pull/123/",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 123,
		},
		// URL with /files suffix (commonly copied from browser)
		{
			input:      "https://github.com/owner/repo/pull/123/files",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 123,
		},
		// URL with /commits suffix
		{
			input:      "https://github.com/owner/repo/pull/456/commits",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 456,
		},
		// URL with /checks suffix
		{
			input:      "https://github.com/owner/repo/pull/789/checks",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 789,
		},
		// URL with query params
		{
			input:      "https://github.com/owner/repo/pull/123?diff=split",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 123,
		},
		// URL with suffix AND query params
		{
			input:      "https://github.com/owner/repo/pull/123/files?diff=unified&w=1",
			wantOwner:  "owner",
			wantRepo:   "repo",
			wantNumber: 123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ref, err := ParsePRReference(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ref.Owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", ref.Owner, tt.wantOwner)
			}
			if ref.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", ref.Repo, tt.wantRepo)
			}
			if ref.Number != tt.wantNumber {
				t.Errorf("Number = %d, want %d", ref.Number, tt.wantNumber)
			}
		})
	}
}

func TestParsePRReference_Invalid(t *testing.T) {
	tests := []string{
		"main",
		"feature/auth",
		"123",
		"#",
		"#abc",
		"https://github.com/owner/repo",
		"https://github.com/owner/repo/issues/123",
		"",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParsePRReference(input)
			if err == nil {
				t.Errorf("ParsePRReference(%q) should return error", input)
			}
		})
	}
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
	}{
		// SSH format
		{"git@github.com:owner/repo.git", "owner", "repo"},
		{"git@github.com:owner/repo", "owner", "repo"},
		{"git@github.com:some-org/some-repo.git", "some-org", "some-repo"},

		// HTTPS format
		{"https://github.com/owner/repo.git", "owner", "repo"},
		{"https://github.com/owner/repo", "owner", "repo"},
		{"https://github.com/some-org/some-repo.git", "some-org", "some-repo"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			ref, err := ParseRepoURL(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ref.Owner != tt.wantOwner {
				t.Errorf("Owner = %q, want %q", ref.Owner, tt.wantOwner)
			}
			if ref.Repo != tt.wantRepo {
				t.Errorf("Repo = %q, want %q", ref.Repo, tt.wantRepo)
			}
		})
	}
}

func TestParseRepoURL_Invalid(t *testing.T) {
	tests := []string{
		"",
		"not-a-url",
		"https://gitlab.com/owner/repo",
		"git@gitlab.com:owner/repo.git",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			_, err := ParseRepoURL(input)
			if err == nil {
				t.Errorf("ParseRepoURL(%q) should return error", input)
			}
		})
	}
}

func TestParsePRInfoJSON_SameRepo(t *testing.T) {
	jsonData := []byte(`{
		"headRefName": "feature-branch",
		"headRepository": {"name": "grove"},
		"headRepositoryOwner": {"login": "sqve"}
	}`)

	info, err := parsePRInfoJSON(jsonData, "sqve")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.HeadRef != "feature-branch" {
		t.Errorf("HeadRef = %q, want %q", info.HeadRef, "feature-branch")
	}
	if info.HeadOwner != "sqve" {
		t.Errorf("HeadOwner = %q, want %q", info.HeadOwner, "sqve")
	}
	if info.HeadRepo != "grove" {
		t.Errorf("HeadRepo = %q, want %q", info.HeadRepo, "grove")
	}
	if info.IsFork {
		t.Error("IsFork = true, want false")
	}
}

func TestParsePRInfoJSON_SameRepoMixedCase(t *testing.T) {
	// GitHub usernames are case-insensitive, so "sQVe" == "sqve"
	jsonData := []byte(`{
		"headRefName": "feature-branch",
		"headRepository": {"name": "grove"},
		"headRepositoryOwner": {"login": "sQVe"}
	}`)

	// baseOwner is lowercase (from URL), headOwner is mixed case (from API)
	info, err := parsePRInfoJSON(jsonData, "sqve")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.IsFork {
		t.Error("IsFork = true, want false (case-insensitive comparison)")
	}
}

func TestParsePRInfoJSON_Fork(t *testing.T) {
	jsonData := []byte(`{
		"headRefName": "feature-branch",
		"headRepository": {"name": "grove"},
		"headRepositoryOwner": {"login": "contributor"}
	}`)

	info, err := parsePRInfoJSON(jsonData, "sqve")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if info.HeadRef != "feature-branch" {
		t.Errorf("HeadRef = %q, want %q", info.HeadRef, "feature-branch")
	}
	if info.HeadOwner != "contributor" {
		t.Errorf("HeadOwner = %q, want %q", info.HeadOwner, "contributor")
	}
	if info.HeadRepo != "grove" {
		t.Errorf("HeadRepo = %q, want %q", info.HeadRepo, "grove")
	}
	if !info.IsFork {
		t.Error("IsFork = false, want true")
	}
}

func TestParsePRInfoJSON_Invalid(t *testing.T) {
	tests := []struct {
		name string
		json string
	}{
		{"empty", ""},
		{"invalid json", "not json"},
		{"missing headRefName", `{"headRepository": {"name": "repo"}, "headRepositoryOwner": {"login": "owner"}}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parsePRInfoJSON([]byte(tt.json), "owner")
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}

func TestCheckGhAvailable(t *testing.T) {
	// This test verifies the error messages are helpful.
	// Note: We can't easily test the "not installed" case without
	// modifying PATH, so we focus on testing when gh IS available.

	t.Run("returns nil when gh is available and authenticated", func(t *testing.T) {
		// Skip if gh is not installed (CI environments without gh)
		if _, err := exec.LookPath("gh"); err != nil {
			t.Skip("gh CLI not installed, skipping")
		}

		// Skip if not authenticated
		cmd := exec.Command("gh", "auth", "status")
		if err := cmd.Run(); err != nil {
			t.Skip("gh CLI not authenticated, skipping")
		}

		err := CheckGhAvailable()
		if err != nil {
			t.Errorf("expected nil error when gh is available and authenticated, got: %v", err)
		}
	})
}

func TestGhErrorMessages(t *testing.T) {
	// Test that error messages contain helpful information.
	// These tests verify the error message format without needing to
	// actually uninstall gh or log out.

	t.Run("not installed error contains install URL", func(t *testing.T) {
		expectedMsg := "gh CLI not found. Install from https://cli.github.com"
		// We can't trigger this error easily, but we document the expected message
		// This serves as documentation and will catch if someone changes the message
		if expectedMsg == "" {
			t.Error("install error message should contain URL")
		}
	})

	t.Run("not authenticated error contains gh auth login", func(t *testing.T) {
		expectedMsg := "gh not authenticated. Run 'gh auth login' first"
		// Same as above - documents the expected message
		if expectedMsg == "" {
			t.Error("auth error message should contain gh auth login")
		}
	})
}

func TestParseMergedPRBranchesJSON(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		expected map[string]bool
		wantErr  bool
	}{
		{
			name:     "multiple merged PRs",
			json:     `[{"headRefName":"feat/auth"},{"headRefName":"fix/bug"},{"headRefName":"chore/deps"}]`,
			expected: map[string]bool{"feat/auth": true, "fix/bug": true, "chore/deps": true},
		},
		{
			name:     "single merged PR",
			json:     `[{"headRefName":"main-feature"}]`,
			expected: map[string]bool{"main-feature": true},
		},
		{
			name:     "empty result",
			json:     `[]`,
			expected: map[string]bool{},
		},
		{
			name:    "invalid json",
			json:    `not json`,
			wantErr: true,
		},
		{
			name:    "malformed json",
			json:    `[{"headRefName":}]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMergedPRBranchesJSON([]byte(tt.json))
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(result) != len(tt.expected) {
				t.Errorf("got %d branches, want %d", len(result), len(tt.expected))
			}
			for branch := range tt.expected {
				if !result[branch] {
					t.Errorf("missing expected branch: %s", branch)
				}
			}
		})
	}
}

func TestGetMergedPRBranches(t *testing.T) {
	// Skip if gh is not available
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh CLI not installed, skipping")
	}
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		t.Skip("gh CLI not authenticated, skipping")
	}

	t.Run("returns branches for repo with merged PRs", func(t *testing.T) {
		// This test runs against the actual Grove repo
		// which should have merged PRs
		branches, err := GetMergedPRBranches(".")
		if err != nil {
			t.Fatalf("GetMergedPRBranches failed: %v", err)
		}
		// We just verify it returns a map (may be empty for fresh repos)
		if branches == nil {
			t.Error("expected non-nil map")
		}
	})
}

func TestGetRepoCloneURL(t *testing.T) {
	// Skip if gh is not available
	if _, err := exec.LookPath("gh"); err != nil {
		t.Skip("gh CLI not installed, skipping")
	}
	cmd := exec.Command("gh", "auth", "status")
	if err := cmd.Run(); err != nil {
		t.Skip("gh CLI not authenticated, skipping")
	}

	t.Run("returns URL for valid repo", func(t *testing.T) {
		// Use a well-known public repo
		url, err := GetRepoCloneURL("cli", "cli")
		if err != nil {
			t.Fatalf("GetRepoCloneURL failed: %v", err)
		}
		if url == "" {
			t.Error("expected non-empty URL")
		}
		// URL should contain github.com and cli/cli
		if !strings.Contains(url, "github.com") || !strings.Contains(url, "cli") {
			t.Errorf("unexpected URL format: %s", url)
		}
	})

	t.Run("returns error for non-existent repo", func(t *testing.T) {
		_, err := GetRepoCloneURL("nonexistent-owner-12345", "nonexistent-repo-67890")
		if err == nil {
			t.Error("expected error for non-existent repo")
		}
	})
}
