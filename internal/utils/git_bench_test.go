//go:build !integration
// +build !integration

package utils

import (
	"testing"

	"github.com/sqve/grove/internal/testutils"
)

// BenchmarkGitRepository_IsGitRepository benchmarks git repository detection.
func BenchmarkGitRepository_IsGitRepository(b *testing.B) {
	mockExec := testutils.NewMockGitExecutor()
	mockExec.SetResponseSlice([]string{"rev-parse", "--git-dir"}, ".git", nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := IsGitRepository(mockExec)
		if err != nil {
			b.Fatalf("IsGitRepository failed: %v", err)
		}
	}
}

// BenchmarkGitRepository_GetRepositoryRoot benchmarks repository root detection.
func BenchmarkGitRepository_GetRepositoryRoot(b *testing.B) {
	mockExec := testutils.NewMockGitExecutor()
	mockExec.SetResponseSlice([]string{"rev-parse", "--show-toplevel"}, "/path/to/repo", nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := GetRepositoryRoot(mockExec)
		if err != nil {
			b.Fatalf("GetRepositoryRoot failed: %v", err)
		}
	}
}

// BenchmarkGitRepository_ValidateRepository benchmarks repository validation.
func BenchmarkGitRepository_ValidateRepository(b *testing.B) {
	mockExec := testutils.NewMockGitExecutor()
	mockExec.SetResponseSlice([]string{"rev-parse", "--git-dir"}, ".git", nil)
	mockExec.SetResponseSlice([]string{"rev-parse", "--show-toplevel"}, "/path/to/repo", nil)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := ValidateRepository(mockExec)
		if err != nil {
			b.Fatalf("ValidateRepository failed: %v", err)
		}
	}
}

// BenchmarkGitURL_IsGitURL benchmarks Git URL detection.
func BenchmarkGitURL_IsGitURL(b *testing.B) {
	urls := []string{
		"https://github.com/user/repo.git",
		"git@github.com:user/repo.git",
		"https://gitlab.com/user/repo",
		"ssh://git@example.com/repo.git",
		"not-a-git-url",
		"https://example.com/not-git",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		url := urls[i%len(urls)]
		IsGitURL(url)
	}
}

// BenchmarkGitURL_ParseGitPlatformURL benchmarks Git platform URL parsing.
func BenchmarkGitURL_ParseGitPlatformURL(b *testing.B) {
	urls := []string{
		"https://github.com/user/repo.git",
		"git@github.com:user/repo.git",
		"https://gitlab.com/group/subgroup/repo",
		"https://gitea.com/user/repo",
		"ssh://git@bitbucket.org/user/repo.git",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		url := urls[i%len(urls)]
		_, err := ParseGitPlatformURL(url)
		if err != nil {
			// Some URLs may fail parsing, which is expected
			continue
		}
	}
}

// BenchmarkGitURL_ParseGitHubURL benchmarks GitHub URL parsing specifically.
func BenchmarkGitURL_ParseGitHubURL(b *testing.B) {
	urls := []string{
		"https://github.com/user/repo.git",
		"git@github.com:user/repo.git",
		"https://github.com/organization/complex-repo-name",
		"git@github.com:org/repo-with-dashes.git",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		url := urls[i%len(urls)]
		result := parseGitHubURL(url)
		if result == nil {
			b.Fatalf("parseGitHubURL failed for %s", url)
		}
	}
}

// BenchmarkGitURL_DetermineGiteaPlatform benchmarks Gitea platform detection.
func BenchmarkGitURL_DetermineGiteaPlatform(b *testing.B) {
	hosts := []string{
		"gitea.com",
		"codeberg.org",
		"gitea.io",
		"git.sr.ht",
		"unknown-host.com",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		host := hosts[i%len(hosts)]
		determineGiteaPlatform(host)
	}
}

// BenchmarkGitURL_IsKnownGiteaInstance benchmarks known Gitea instance detection.
func BenchmarkGitURL_IsKnownGiteaInstance(b *testing.B) {
	hosts := []string{
		"gitea.com",
		"codeberg.org",
		"try.gitea.io",
		"git.sr.ht",
		"github.com",
		"gitlab.com",
		"unknown-host.com",
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		host := hosts[i%len(hosts)]
		isKnownGiteaInstance(host)
	}
}
