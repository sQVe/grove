package commands

import (
	"strings"
	"testing"

	"github.com/sqve/grove/internal/config"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		input   string
		major   int
		minor   int
		patch   int
		wantErr bool
	}{
		{"2.48.0", 2, 48, 0, false},
		{"2.48", 2, 48, 0, false},
		{"10.0.1", 10, 0, 1, false},
		{"1.2.3", 1, 2, 3, false},
		{"0.0.0", 0, 0, 0, false},
		{"invalid", 0, 0, 0, true},
		{"", 0, 0, 0, true},
		{"a.b.c", 0, 0, 0, true},
		{"1.2.3.4", 1, 2, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, patch, err := parseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseVersion(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if major != tt.major || minor != tt.minor || patch != tt.patch {
					t.Errorf("parseVersion(%q) = %d.%d.%d, want %d.%d.%d",
						tt.input, major, minor, patch, tt.major, tt.minor, tt.patch)
				}
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		current string
		minimum string
		want    int
	}{
		{"2.48.0", "2.48.0", 0},
		{"2.49.0", "2.48.0", 1},
		{"2.47.0", "2.48.0", -1},
		{"3.0.0", "2.48.0", 1},
		{"1.0.0", "2.48.0", -1},
		{"2.48.1", "2.48.0", 1},
		{"2.48.0", "2.48.1", -1},
	}

	for _, tt := range tests {
		t.Run(tt.current+"_vs_"+tt.minimum, func(t *testing.T) {
			got := compareVersions(tt.current, tt.minimum)
			if got != tt.want {
				t.Errorf("compareVersions(%q, %q) = %d, want %d", tt.current, tt.minimum, got, tt.want)
			}
		})
	}
}

func TestParseGitVersionOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{"standard", "git version 2.48.0", "2.48.0", false},
		{"windows", "git version 2.48.0.windows.1", "2.48.0", false},
		{"with_newline", "git version 2.47.1\n", "2.47.1", false},
		{"macos", "git version 2.39.3 (Apple Git-146)", "2.39.3", false},
		{"empty", "", "", true},
		{"no_version", "some other output", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGitVersionOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGitVersionOutput(%q) error = %v, wantErr %v", tt.output, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseGitVersionOutput(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestParseGhVersionOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		want    string
		wantErr bool
	}{
		{"standard", "gh version 2.40.0 (2024-01-01)\nhttps://github.com/cli/cli/releases/tag/v2.40.0", "2.40.0", false},
		{"simple", "gh version 2.0.0", "2.0.0", false},
		{"with_date", "gh version 2.62.0 (2024-12-03)", "2.62.0", false},
		{"empty", "", "", true},
		{"no_version", "some other output", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseGhVersionOutput(tt.output)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseGhVersionOutput(%q) error = %v, wantErr %v", tt.output, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("parseGhVersionOutput(%q) = %q, want %q", tt.output, got, tt.want)
			}
		})
	}
}

func TestCountSeverities(t *testing.T) {
	tests := []struct {
		name         string
		issues       []Issue
		wantErrors   int
		wantWarnings int
		wantInfos    int
	}{
		{
			name:   "empty",
			issues: []Issue{},
		},
		{
			name:       "single_error",
			issues:     []Issue{{Severity: SeverityError}},
			wantErrors: 1,
		},
		{
			name:         "single_warning",
			issues:       []Issue{{Severity: SeverityWarning}},
			wantWarnings: 1,
		},
		{
			name:      "single_info",
			issues:    []Issue{{Severity: SeverityInfo}},
			wantInfos: 1,
		},
		{
			name: "mixed",
			issues: []Issue{
				{Severity: SeverityError},
				{Severity: SeverityWarning},
				{Severity: SeverityInfo},
				{Severity: SeverityError},
			},
			wantErrors:   2,
			wantWarnings: 1,
			wantInfos:    1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errors, warnings, infos := countSeverities(tt.issues)
			if errors != tt.wantErrors {
				t.Errorf("errors = %d, want %d", errors, tt.wantErrors)
			}
			if warnings != tt.wantWarnings {
				t.Errorf("warnings = %d, want %d", warnings, tt.wantWarnings)
			}
			if infos != tt.wantInfos {
				t.Errorf("infos = %d, want %d", infos, tt.wantInfos)
			}
		})
	}
}

func TestCheckDepVersion(t *testing.T) {
	tests := []struct {
		name          string
		dep           depInfo
		installed     bool
		version       string
		wantIssue     bool
		wantSeverity  Severity
		wantMsgSubstr string
	}{
		{
			name:          "git_missing_required",
			dep:           depInfo{name: "git", minVersion: "2.48.0", missingSeverity: SeverityError, outdatedSeverity: SeverityError},
			installed:     false,
			version:       "",
			wantIssue:     true,
			wantSeverity:  SeverityError,
			wantMsgSubstr: "not installed",
		},
		{
			name:          "git_outdated",
			dep:           depInfo{name: "git", minVersion: "2.48.0", missingSeverity: SeverityError, outdatedSeverity: SeverityError},
			installed:     true,
			version:       "2.47.0",
			wantIssue:     true,
			wantSeverity:  SeverityError,
			wantMsgSubstr: "below minimum",
		},
		{
			name:      "git_ok",
			dep:       depInfo{name: "git", minVersion: "2.48.0", missingSeverity: SeverityError, outdatedSeverity: SeverityError},
			installed: true,
			version:   "2.48.0",
			wantIssue: false,
		},
		{
			name:      "git_newer",
			dep:       depInfo{name: "git", minVersion: "2.48.0", missingSeverity: SeverityError, outdatedSeverity: SeverityError},
			installed: true,
			version:   "2.49.0",
			wantIssue: false,
		},
		{
			name:          "gh_missing_optional",
			dep:           depInfo{name: "gh", minVersion: "2.0.0", missingSeverity: SeverityInfo, outdatedSeverity: SeverityInfo},
			installed:     false,
			version:       "",
			wantIssue:     true,
			wantSeverity:  SeverityInfo,
			wantMsgSubstr: "not installed",
		},
		{
			name:          "gh_outdated",
			dep:           depInfo{name: "gh", minVersion: "2.0.0", missingSeverity: SeverityInfo, outdatedSeverity: SeverityInfo},
			installed:     true,
			version:       "1.9.0",
			wantIssue:     true,
			wantSeverity:  SeverityInfo,
			wantMsgSubstr: "below minimum",
		},
		{
			name:      "gh_ok",
			dep:       depInfo{name: "gh", minVersion: "2.0.0", missingSeverity: SeverityInfo, outdatedSeverity: SeverityInfo},
			installed: true,
			version:   "2.40.0",
			wantIssue: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dep := tt.dep
			issue := checkDepVersion(&dep, tt.installed, tt.version)
			if tt.wantIssue {
				if issue == nil {
					t.Error("expected issue, got nil")
					return
				}
				if issue.Severity != tt.wantSeverity {
					t.Errorf("severity = %v, want %v", issue.Severity, tt.wantSeverity)
				}
				if issue.Category != CategoryDeps {
					t.Errorf("category = %v, want CategoryDeps", issue.Category)
				}
				if tt.wantMsgSubstr != "" && !strings.Contains(issue.Message, tt.wantMsgSubstr) {
					t.Errorf("message %q does not contain %q", issue.Message, tt.wantMsgSubstr)
				}
			} else if issue != nil {
				t.Errorf("expected no issue, got %+v", issue)
			}
		})
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "     0 B "},
		{"small bytes", 512, "   512 B "},
		{"just under 1KB", 1023, "  1023 B "},
		{"exactly 1KB", 1024, "   1.0 KB"},
		{"1.5KB", 1536, "   1.5 KB"},
		{"just under 1MB", 1048575, "1024.0 KB"},
		{"exactly 1MB", 1048576, "   1.0 MB"},
		{"10.5MB", 11010048, "  10.5 MB"},
		{"just under 1GB", 1073741823, "1024.0 MB"},
		{"exactly 1GB", 1073741824, "   1.0 GB"},
		{"2.5GB", 2684354560, "   2.5 GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatSize(tt.bytes)
			if got != tt.expected {
				t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.expected)
			}
		})
	}
}

func TestCategoryToString(t *testing.T) {
	tests := []struct {
		category Category
		expected string
	}{
		{CategoryDeps, "deps"},
		{CategoryGit, "git"},
		{CategoryConfig, "config"},
		{Category(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := categoryToString(tt.category)
			if got != tt.expected {
				t.Errorf("categoryToString(%v) = %q, want %q", tt.category, got, tt.expected)
			}
		})
	}
}

func TestSeverityToString(t *testing.T) {
	tests := []struct {
		severity Severity
		expected string
	}{
		{SeverityInfo, "info"},
		{SeverityWarning, "warning"},
		{SeverityError, "error"},
		{Severity(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := severityToString(tt.severity)
			if got != tt.expected {
				t.Errorf("severityToString(%v) = %q, want %q", tt.severity, got, tt.expected)
			}
		})
	}
}

func TestGetIssueSymbol(t *testing.T) {
	tests := []struct {
		name     string
		severity Severity
		plain    bool
		expected string
	}{
		{"error plain", SeverityError, true, "[x]"},
		{"warning plain", SeverityWarning, true, "[!]"},
		{"info plain", SeverityInfo, true, "[i]"},
		{"unknown plain", Severity(999), true, "[-]"},
		{"error styled", SeverityError, false, "✗"},
		{"warning styled", SeverityWarning, false, "⚠"},
		{"info styled", SeverityInfo, false, "→"},
		{"unknown styled", Severity(999), false, "•"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetPlain(tt.plain)
			defer config.SetPlain(false)

			got := getIssueSymbol(tt.severity)
			if got != tt.expected {
				t.Errorf("getIssueSymbol(%v) with plain=%v = %q, want %q",
					tt.severity, tt.plain, got, tt.expected)
			}
		})
	}
}

func TestFilterIssuesByCategory(t *testing.T) {
	issues := []Issue{
		{Category: CategoryDeps, Message: "deps1"},
		{Category: CategoryGit, Message: "git1"},
		{Category: CategoryDeps, Message: "deps2"},
		{Category: CategoryConfig, Message: "config1"},
		{Category: CategoryGit, Message: "git2"},
	}

	tests := []struct {
		name     string
		category Category
		wantLen  int
	}{
		{"filter deps", CategoryDeps, 2},
		{"filter git", CategoryGit, 2},
		{"filter config", CategoryConfig, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterIssuesByCategory(issues, tt.category)
			if len(got) != tt.wantLen {
				t.Errorf("filterIssuesByCategory() returned %d issues, want %d", len(got), tt.wantLen)
			}
			for _, issue := range got {
				if issue.Category != tt.category {
					t.Errorf("filtered issue has wrong category: got %v, want %v", issue.Category, tt.category)
				}
			}
		})
	}

	t.Run("empty input returns empty", func(t *testing.T) {
		got := filterIssuesByCategory([]Issue{}, CategoryDeps)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d items", len(got))
		}
	})

	t.Run("no matches returns empty", func(t *testing.T) {
		depsOnly := []Issue{{Category: CategoryDeps}}
		got := filterIssuesByCategory(depsOnly, CategoryGit)
		if len(got) != 0 {
			t.Errorf("expected empty slice, got %d items", len(got))
		}
	})
}
