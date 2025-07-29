//go:build !integration
// +build !integration

package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sqve/grove/internal/testutils"
)

func TestGitRepository_IsGitRepository(t *testing.T) {
	tests := []struct {
		name        string
		gitOutput   string
		gitError    error
		expectRepo  bool
		expectError bool
	}{
		{
			name:        "valid git repository",
			gitOutput:   ".git",
			gitError:    nil,
			expectRepo:  true,
			expectError: false,
		},
		{
			name:        "not a git repository - exit 128",
			gitOutput:   "",
			gitError:    fmt.Errorf("git rev-parse --git-dir failed (exit 128): fatal: not a git repository"),
			expectRepo:  false,
			expectError: false,
		},
		{
			name:        "git command error - other exit code",
			gitOutput:   "",
			gitError:    fmt.Errorf("git rev-parse --git-dir failed (exit 1): permission denied"),
			expectRepo:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := testutils.NewMockGitExecutor()
			mock.SetResponseSlice([]string{"rev-parse", "--git-dir"}, tt.gitOutput, tt.gitError)

			isRepo, err := IsGitRepository(mock)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectRepo, isRepo)

			assert.Equal(t, 1, mock.CallCount)
			assert.Equal(t, []string{"rev-parse", "--git-dir"}, mock.Commands[0])
		})
	}
}

func TestGitRepository_GetRepositoryRoot(t *testing.T) {
	tests := []struct {
		name        string
		gitOutput   string
		gitError    error
		expectRoot  string
		expectError bool
	}{
		{
			name:        "successful get repository root",
			gitOutput:   "/home/user/project",
			gitError:    nil,
			expectRoot:  "/home/user/project",
			expectError: false,
		},
		{
			name:        "git command fails",
			gitOutput:   "",
			gitError:    fmt.Errorf("not a git repository"),
			expectRoot:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := testutils.NewMockGitExecutor()
			mock.SetResponseSlice([]string{"rev-parse", "--show-toplevel"}, tt.gitOutput, tt.gitError)

			root, err := GetRepositoryRoot(mock)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectRoot, root)

			assert.Equal(t, 1, mock.CallCount)
			assert.Equal(t, []string{"rev-parse", "--show-toplevel"}, mock.Commands[0])
		})
	}
}

func TestGitRepository_ValidateRepository(t *testing.T) {
	tests := []struct {
		name           string
		isRepoOutput   string
		isRepoError    error
		headOutput     string
		headError      error
		expectError    bool
		expectedErrMsg string
	}{
		{
			name:         "valid repository with commits",
			isRepoOutput: ".git",
			isRepoError:  nil,
			headOutput:   "abc123",
			headError:    nil,
			expectError:  false,
		},
		{
			name:           "not a git repository",
			isRepoOutput:   "",
			isRepoError:    fmt.Errorf("git rev-parse --git-dir failed (exit 128): not a repo"),
			expectError:    true,
			expectedErrMsg: "not in a git repository",
		},
		{
			name:           "repository with no commits",
			isRepoOutput:   ".git",
			isRepoError:    nil,
			headOutput:     "",
			headError:      fmt.Errorf("git rev-parse HEAD failed: bad revision 'HEAD'"),
			expectError:    true,
			expectedErrMsg: "repository has no commits",
		},
		{
			name:           "other git error on HEAD",
			isRepoOutput:   ".git",
			isRepoError:    nil,
			headOutput:     "",
			headError:      fmt.Errorf("permission denied"),
			expectError:    true,
			expectedErrMsg: "failed to validate repository",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := testutils.NewMockGitExecutor()
			mock.SetResponseSlice([]string{"rev-parse", "--git-dir"}, tt.isRepoOutput, tt.isRepoError)
			mock.SetResponseSlice([]string{"rev-parse", "HEAD"}, tt.headOutput, tt.headError)

			err := ValidateRepository(mock)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGitURL_IsGitURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
		desc     string
	}{
		// Valid URLs.
		{"https://github.com/user/repo.git", true, "HTTPS with .git extension"},
		{"https://github.com/user/repo", true, "HTTPS GitHub URL without .git"},
		{"https://gitlab.com/user/repo", true, "HTTPS GitLab URL"},
		{"http://example.com/repo.git", true, "HTTP with .git extension"},
		{"git@github.com:user/repo.git", true, "SSH GitHub URL"},
		{"ssh://git@github.com/user/repo.git", true, "SSH with full protocol"},
		{"git://example.com/repo.git", true, "Git protocol"},

		// Invalid URLs.
		{"not-a-url", false, "Plain text"},
		{"https://example.com", false, "HTTPS without repo pattern"},
		{"file.txt", false, "Local filename"},
		{"/path/to/directory", false, "Local directory path"},
		{"", false, "Empty string"},
		{"ftp://example.com/repo.git", false, "Non-git protocol"},
		{"https://github.com", false, "GitHub domain only"},
		{"mailto:user@example.com", false, "Email URL"},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := IsGitURL(tt.url)
			assert.Equal(t, tt.expected, result, "URL: %s", tt.url)
		})
	}
}

func TestGitURL_ParseGitPlatformURL(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedInfo   *GitURLInfo
		expectError    bool
		expectedErrMsg string
	}{
		// GitHub URLs.
		{
			name: "GitHub repository",
			url:  "https://github.com/owner/repo",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://github.com/owner/repo.git",
				Platform: "github",
			},
		},
		{
			name: "GitHub repository with .git",
			url:  "https://github.com/owner/repo.git",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://github.com/owner/repo.git",
				Platform: "github",
			},
		},
		{
			name: "GitHub branch",
			url:  "https://github.com/owner/repo/tree/feature-branch",
			expectedInfo: &GitURLInfo{
				RepoURL:    "https://github.com/owner/repo.git",
				BranchName: "feature-branch",
				Platform:   "github",
			},
		},
		{
			name: "GitHub pull request",
			url:  "https://github.com/owner/repo/pull/123",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://github.com/owner/repo.git",
				PRNumber: "123",
				Platform: "github",
			},
		},
		// GitLab URLs.
		{
			name: "GitLab repository",
			url:  "https://gitlab.com/owner/repo",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://gitlab.com/owner/repo.git",
				Platform: "gitlab",
			},
		},
		{
			name: "GitLab branch",
			url:  "https://gitlab.com/owner/repo/-/tree/develop",
			expectedInfo: &GitURLInfo{
				RepoURL:    "https://gitlab.com/owner/repo.git",
				BranchName: "develop",
				Platform:   "gitlab",
			},
		},
		{
			name: "GitLab merge request",
			url:  "https://gitlab.com/owner/repo/-/merge_requests/456",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://gitlab.com/owner/repo.git",
				PRNumber: "456",
				Platform: "gitlab",
			},
		},
		// Bitbucket URLs.
		{
			name: "Bitbucket repository",
			url:  "https://bitbucket.org/owner/repo",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://bitbucket.org/owner/repo.git",
				Platform: "bitbucket",
			},
		},
		{
			name: "Bitbucket branch",
			url:  "https://bitbucket.org/owner/repo/src/main/",
			expectedInfo: &GitURLInfo{
				RepoURL:    "https://bitbucket.org/owner/repo.git",
				BranchName: "main",
				Platform:   "bitbucket",
			},
		},
		{
			name: "Bitbucket pull request",
			url:  "https://bitbucket.org/owner/repo/pull-requests/789",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://bitbucket.org/owner/repo.git",
				PRNumber: "789",
				Platform: "bitbucket",
			},
		},
		// Azure DevOps URLs.
		{
			name: "Azure DevOps repository",
			url:  "https://dev.azure.com/org/project/_git/repo",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://dev.azure.com/org/project/_git/repo",
				Platform: "azure-devops",
			},
		},
		{
			name: "Azure DevOps branch",
			url:  "https://dev.azure.com/org/project/_git/repo?version=GBfeature-branch",
			expectedInfo: &GitURLInfo{
				RepoURL:    "https://dev.azure.com/org/project/_git/repo",
				BranchName: "feature-branch",
				Platform:   "azure-devops",
			},
		},
		{
			name: "Azure DevOps pull request",
			url:  "https://dev.azure.com/org/project/_git/repo/pullrequest/101",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://dev.azure.com/org/project/_git/repo",
				PRNumber: "101",
				Platform: "azure-devops",
			},
		},
		// Codeberg URLs.
		{
			name: "Codeberg repository",
			url:  "https://codeberg.org/owner/repo",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://codeberg.org/owner/repo.git",
				Platform: "codeberg",
			},
		},
		{
			name: "Codeberg branch",
			url:  "https://codeberg.org/owner/repo/src/branch/main",
			expectedInfo: &GitURLInfo{
				RepoURL:    "https://codeberg.org/owner/repo.git",
				BranchName: "main",
				Platform:   "codeberg",
			},
		},
		{
			name: "Codeberg pull request",
			url:  "https://codeberg.org/owner/repo/pulls/42",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://codeberg.org/owner/repo.git",
				PRNumber: "42",
				Platform: "codeberg",
			},
		},
		// Standard Git URLs (fallback).
		{
			name: "Standard Git HTTPS URL",
			url:  "https://example.com/repo.git",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://example.com/repo.git",
				Platform: "git",
			},
		},
		{
			name: "SSH Git URL",
			url:  "git@example.com:user/repo.git",
			expectedInfo: &GitURLInfo{
				RepoURL:  "git@example.com:user/repo.git",
				Platform: "git",
			},
		},
		// Error cases.
		{
			name:           "Empty URL",
			url:            "",
			expectError:    true,
			expectedErrMsg: "empty URL provided",
		},
		{
			name:           "Invalid URL format",
			url:            "not-a-url",
			expectError:    true,
			expectedErrMsg: "URL format not recognized",
		},
		{
			name:           "Unsupported domain",
			url:            "https://unknown.com/owner/repo",
			expectError:    true,
			expectedErrMsg: "URL format not recognized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info, err := ParseGitPlatformURL(tt.url)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedErrMsg)
				assert.Nil(t, info)
			} else {
				require.NoError(t, err)
				require.NotNil(t, info)
				assert.Equal(t, tt.expectedInfo.RepoURL, info.RepoURL)
				assert.Equal(t, tt.expectedInfo.BranchName, info.BranchName)
				assert.Equal(t, tt.expectedInfo.PRNumber, info.PRNumber)
				assert.Equal(t, tt.expectedInfo.Platform, info.Platform)
			}
		})
	}
}

func TestGitURL_ParseGitHubURL(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		expectedInfo *GitURLInfo
	}{
		{
			name: "GitHub repo with trailing slash",
			url:  "https://github.com/owner/repo/",
			expectedInfo: &GitURLInfo{
				RepoURL:  "https://github.com/owner/repo.git",
				Platform: "github",
			},
		},
		{
			name: "GitHub branch with complex name",
			url:  "https://github.com/owner/repo/tree/feature/complex-branch-name",
			expectedInfo: &GitURLInfo{
				RepoURL:    "https://github.com/owner/repo.git",
				BranchName: "feature/complex-branch-name",
				Platform:   "github",
			},
		},
		{
			name:         "Non-GitHub URL",
			url:          "https://example.com/owner/repo",
			expectedInfo: nil,
		},
		{
			name:         "Malformed GitHub URL",
			url:          "https://github.com/owner",
			expectedInfo: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := parseGitHubURL(tt.url)
			if tt.expectedInfo == nil {
				assert.Nil(t, info)
			} else {
				require.NotNil(t, info)
				assert.Equal(t, tt.expectedInfo.RepoURL, info.RepoURL)
				assert.Equal(t, tt.expectedInfo.BranchName, info.BranchName)
				assert.Equal(t, tt.expectedInfo.PRNumber, info.PRNumber)
				assert.Equal(t, tt.expectedInfo.Platform, info.Platform)
			}
		})
	}
}

func TestGitURL_DetermineGiteaPlatform(t *testing.T) {
	tests := []struct {
		host     string
		expected string
	}{
		{"codeberg.org", "codeberg"},
		{"gitea.com", "gitea"},
		{"my-gitea.example.com", "gitea"},
		{"example.com", "gitea"},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := determineGiteaPlatform(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGitURL_IsKnownGiteaInstance(t *testing.T) {
	tests := []struct {
		host     string
		expected bool
	}{
		{"codeberg.org", true},
		{"gitea.com", true},
		{"try.gitea.io", true},
		{"my-gitea.example.com", true},
		{"gitea-hosting.net", true},
		{"github.com", false},
		{"gitlab.com", false},
		{"example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			result := isKnownGiteaInstance(tt.host)
			assert.Equal(t, tt.expected, result)
		})
	}
}
