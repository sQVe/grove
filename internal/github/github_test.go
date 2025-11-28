package github

import (
	"testing"
)

func TestIsPRReference(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		// PR number format
		{"#123", true},
		{"#1", true},
		{"#999999", true},

		// PR URL format
		{"https://github.com/owner/repo/pull/123", true},
		{"https://github.com/some-org/some-repo/pull/1", true},
		{"http://github.com/owner/repo/pull/123", true},

		// Not PR references
		{"main", false},
		{"feature/auth", false},
		{"123", false},
		{"#", false},
		{"#abc", false},
		{"https://github.com/owner/repo", false},
		{"https://github.com/owner/repo/issues/123", false},
		{"https://gitlab.com/owner/repo/pull/123", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := IsPRReference(tt.input)
			if result != tt.expected {
				t.Errorf("IsPRReference(%q) = %v, want %v", tt.input, result, tt.expected)
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

	info, err := parsePRInfoJSON(jsonData, "sqve", "grove")
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

func TestParsePRInfoJSON_Fork(t *testing.T) {
	jsonData := []byte(`{
		"headRefName": "feature-branch",
		"headRepository": {"name": "grove"},
		"headRepositoryOwner": {"login": "contributor"}
	}`)

	info, err := parsePRInfoJSON(jsonData, "sqve", "grove")
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
			_, err := parsePRInfoJSON([]byte(tt.json), "owner", "repo")
			if err == nil {
				t.Error("expected error, got nil")
			}
		})
	}
}
