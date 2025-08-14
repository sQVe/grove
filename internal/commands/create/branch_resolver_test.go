package create

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
)

const (
	testLocalBranchOutput  = "* main\n  another-branch"
	testRemoteOriginBranch = "origin/feature-branch"
	testRemoteOutput       = "origin\nupstream"
)

func TestBranchResolver_ResolveBranch_LocalBranchExists(t *testing.T) {
	// Use the centralized mock creation for consistency
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	branchOutput := "* main\n  feature-branch\n  another-branch"
	mockGit.On("Run", "", "branch", "-a", "--list").Return([]byte(branchOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveBranch("feature-branch", "main", true)

	require.NoError(t, err)
	assert.NotNil(t, branchInfo)
	assert.Equal(t, "feature-branch", branchInfo.Name)
	assert.True(t, branchInfo.Exists)
	assert.False(t, branchInfo.IsRemote)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveBranch_RemoteBranchExists(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	mockGit.On("Run", "", "branch", "-a", "--list").Return([]byte(testLocalBranchOutput), []byte(""), nil)

	remoteBranchOutput := testRemoteOriginBranch
	mockGit.On("Run", "", "branch", "-r", "--list", "*/feature-branch").Return([]byte(remoteBranchOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveBranch("feature-branch", "main", true)

	require.NoError(t, err)
	assert.NotNil(t, branchInfo)
	assert.Equal(t, "feature-branch", branchInfo.Name)
	assert.False(t, branchInfo.Exists)
	assert.True(t, branchInfo.IsRemote)
	assert.Equal(t, "origin/feature-branch", branchInfo.TrackingBranch)
	assert.Equal(t, "origin", branchInfo.RemoteName)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveBranch_BranchWillBeCreated(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	mockGit.On("Run", "", "branch", "-a", "--list").Return([]byte(testLocalBranchOutput), []byte(""), nil)

	remoteBranchOutput := ""
	mockGit.On("Run", "", "branch", "-r", "--list", "*/new-feature").Return([]byte(remoteBranchOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveBranch("new-feature", "main", true)

	require.NoError(t, err)
	assert.NotNil(t, branchInfo)
	assert.Equal(t, "new-feature", branchInfo.Name)
	assert.False(t, branchInfo.Exists)
	assert.False(t, branchInfo.IsRemote)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveBranch_BranchNotFoundCreateDisabled(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	mockGit.On("Run", "", "branch", "-a", "--list").Return([]byte(testLocalBranchOutput), []byte(""), nil)

	remoteBranchOutput := ""
	mockGit.On("Run", "", "branch", "-r", "--list", "*/nonexistent").Return([]byte(remoteBranchOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveBranch("nonexistent", "main", false)

	assert.Nil(t, branchInfo)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeGitOperation, groveErr.Code)
	assert.Contains(t, groveErr.Message, "branch 'nonexistent' does not exist")
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveBranch_InvalidBranchName(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	testCases := []struct {
		name       string
		branchName string
		wantErr    string
	}{
		{
			name:       "empty branch name",
			branchName: "",
			wantErr:    "branch name cannot be empty",
		},
		{
			name:       "branch name with whitespace",
			branchName: "feature branch",
			wantErr:    "branch name contains whitespace characters",
		},
		{
			name:       "branch name with invalid characters",
			branchName: "feature~branch",
			wantErr:    "branch name contains invalid characters",
		},
		{
			name:       "branch name starting with dash",
			branchName: "-feature",
			wantErr:    "branch name cannot start with '-' or end with '/'",
		},
		{
			name:       "branch name ending with slash",
			branchName: "feature/",
			wantErr:    "branch name cannot start with '-' or end with '/'",
		},
		{
			name:       "branch name with double dots",
			branchName: "feature..branch",
			wantErr:    "branch name contains invalid sequences",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			branchInfo, err := resolver.ResolveBranch(tc.branchName, "main", true)

			assert.Nil(t, branchInfo)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.ErrorAs(t, err, &groveErr)
			assert.Equal(t, errors.ErrCodeGitOperation, groveErr.Code)
			assert.Contains(t, groveErr.Cause.Error(), tc.wantErr)
		})
	}
}

func TestBranchResolver_ResolveBranch_GitCommandFailure(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	gitError := fmt.Errorf("fatal: not a git repository")
	mockGit.On("Run", "", "branch", "-a", "--list").Return([]byte(""), []byte("fatal: not a git repository"), gitError)

	branchInfo, err := resolver.ResolveBranch("feature-branch", "main", true)

	assert.Nil(t, branchInfo)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeGitOperation, groveErr.Code)
	assert.Equal(t, "failed to check local branch existence", groveErr.Message)
	assert.Equal(t, gitError, groveErr.Cause)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveURL_ValidGitHubURL(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	url := "https://github.com/owner/repo/tree/feature-branch"
	urlBranchInfo, err := resolver.ResolveURL(url)

	require.NoError(t, err)
	assert.NotNil(t, urlBranchInfo)
	assert.Equal(t, "feature-branch", urlBranchInfo.BranchName)
	assert.Equal(t, "https://github.com/owner/repo.git", urlBranchInfo.RepoURL)
	assert.Equal(t, "github", urlBranchInfo.Platform)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveURL_InvalidURL(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	testCases := []struct {
		name    string
		url     string
		wantErr string
	}{
		{
			name:    "empty URL",
			url:     "",
			wantErr: "URL cannot be empty",
		},
		{
			name:    "invalid URL format",
			url:     "not-a-url",
			wantErr: "URL must be a valid HTTP/HTTPS URL",
		},
		{
			name:    "ftp URL",
			url:     "ftp://example.com/repo",
			wantErr: "URL must be a valid HTTP/HTTPS URL",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			urlBranchInfo, err := resolver.ResolveURL(tc.url)

			assert.Nil(t, urlBranchInfo)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.ErrorAs(t, err, &groveErr)
			assert.Equal(t, errors.ErrCodeUnsupportedURL, groveErr.Code)
			assert.Contains(t, groveErr.Cause.Error(), tc.wantErr)
		})
	}
}

func TestBranchResolver_ResolveRemoteBranch_Success(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	remoteOutput := testRemoteOutput
	mockGit.On("Run", "", "remote").Return([]byte(remoteOutput), []byte(""), nil)

	// Mock fetch remote (may fail, but that's okay)
	mockGit.On("Run", "", "fetch", "origin").Return([]byte(""), []byte(""), nil)

	// Mock remote branch exists check
	remoteBranchOutput := testRemoteOriginBranch
	mockGit.On("Run", "", "branch", "-r", "--list", "origin/feature-branch").Return([]byte(remoteBranchOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveRemoteBranch("origin/feature-branch")

	require.NoError(t, err)
	assert.NotNil(t, branchInfo)
	assert.Equal(t, "feature-branch", branchInfo.Name)
	assert.False(t, branchInfo.Exists)
	assert.True(t, branchInfo.IsRemote)
	assert.Equal(t, "origin/feature-branch", branchInfo.TrackingBranch)
	assert.Equal(t, "origin", branchInfo.RemoteName)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveRemoteBranch_InvalidFormat(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	branchInfo, err := resolver.ResolveRemoteBranch("invalid-format")

	assert.Nil(t, branchInfo)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeGitOperation, groveErr.Code)
	assert.Contains(t, groveErr.Message, "invalid remote branch format")
	assert.Contains(t, groveErr.Context["expected"], "remote/branch")
}

func TestBranchResolver_ResolveRemoteBranch_RemoteNotFound(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	remoteOutput := "origin"
	mockGit.On("Run", "", "remote").Return([]byte(remoteOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveRemoteBranch("upstream/feature-branch")

	assert.Nil(t, branchInfo)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeGitOperation, groveErr.Code)
	assert.Contains(t, groveErr.Message, "remote 'upstream' not found")
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveRemoteBranch_BranchNotFound(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	remoteOutput := testRemoteOutput
	mockGit.On("Run", "", "remote").Return([]byte(remoteOutput), []byte(""), nil)

	// Mock fetch remote (successful)
	mockGit.On("Run", "", "fetch", "origin").Return([]byte(""), []byte(""), nil)

	remoteBranchOutput := ""
	mockGit.On("Run", "", "branch", "-r", "--list", "origin/nonexistent").Return([]byte(remoteBranchOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveRemoteBranch("origin/nonexistent")

	assert.Nil(t, branchInfo)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeGitOperation, groveErr.Code)
	assert.Contains(t, groveErr.Message, "branch 'nonexistent' not found on remote 'origin'")
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveRemoteBranch_FetchFailure(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	remoteOutput := "origin"
	mockGit.On("Run", "", "remote").Return([]byte(remoteOutput), []byte(""), nil)

	// Mock fetch remote failure (should not stop the process)
	fetchError := fmt.Errorf("network error")
	mockGit.On("Run", "", "fetch", "origin").Return([]byte(""), []byte("network error"), fetchError)

	remoteBranchOutput := testRemoteOriginBranch
	mockGit.On("Run", "", "branch", "-r", "--list", "origin/feature-branch").Return([]byte(remoteBranchOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveRemoteBranch("origin/feature-branch")

	require.NoError(t, err)
	assert.NotNil(t, branchInfo)
	assert.Equal(t, "feature-branch", branchInfo.Name)
	assert.True(t, branchInfo.IsRemote)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_RemoteExists_True(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	remoteOutput := "origin\nupstream\nfork"
	mockGit.On("Run", "", "remote").Return([]byte(remoteOutput), []byte(""), nil)

	exists := resolver.RemoteExists("upstream")

	assert.True(t, exists)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_RemoteExists_False(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	remoteOutput := testRemoteOutput
	mockGit.On("Run", "", "remote").Return([]byte(remoteOutput), []byte(""), nil)

	exists := resolver.RemoteExists("nonexistent")

	assert.False(t, exists)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_RemoteExists_GitError(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	gitError := fmt.Errorf("fatal: not a git repository")
	mockGit.On("Run", "", "remote").Return([]byte(""), []byte("fatal: not a git repository"), gitError)

	exists := resolver.RemoteExists("origin")

	assert.False(t, exists)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveBranch_RemoteTrackingWithMultipleBranches(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	localBranchOutput := "* main\n  develop"
	mockGit.On("Run", "", "branch", "-a", "--list").Return([]byte(localBranchOutput), []byte(""), nil)

	remoteBranchOutput := "  origin/feature-123\n  origin/another-feature"
	mockGit.On("Run", "", "branch", "-r", "--list", "*/feature-123").Return([]byte(remoteBranchOutput), []byte(""), nil)

	branchInfo, err := resolver.ResolveBranch("feature-123", "main", true)

	require.NoError(t, err)
	assert.NotNil(t, branchInfo)
	assert.Equal(t, "feature-123", branchInfo.Name)
	assert.False(t, branchInfo.Exists)
	assert.True(t, branchInfo.IsRemote)
	assert.Equal(t, "origin/feature-123", branchInfo.TrackingBranch)
	assert.Equal(t, "origin", branchInfo.RemoteName)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveURL_GitLabURL(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	url := "https://gitlab.com/owner/repo/-/tree/feature-branch"
	urlBranchInfo, err := resolver.ResolveURL(url)

	require.NoError(t, err)
	assert.NotNil(t, urlBranchInfo)
	assert.Equal(t, "feature-branch", urlBranchInfo.BranchName)
	assert.Equal(t, "https://gitlab.com/owner/repo.git", urlBranchInfo.RepoURL)
	assert.Equal(t, "gitlab", urlBranchInfo.Platform)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveURL_BitbucketURL(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	// Bitbucket uses /src/{branch} format where branch can contain slashes
	url := "https://bitbucket.org/owner/repo/src/feature-branch"
	urlBranchInfo, err := resolver.ResolveURL(url)

	require.NoError(t, err)
	assert.NotNil(t, urlBranchInfo)
	assert.Equal(t, "feature-branch", urlBranchInfo.BranchName)
	assert.Equal(t, "https://bitbucket.org/owner/repo.git", urlBranchInfo.RepoURL)
	assert.Equal(t, "bitbucket", urlBranchInfo.Platform)
	mockGit.AssertExpectations(t)
}

func TestBranchResolver_ResolveURL_UnsupportedPlatform(t *testing.T) {
	mockGit := testutils.CreateMockGitCommander()
	resolver := NewBranchResolver(mockGit)

	// Try a non-git URL format that doesn't match known platforms
	url := "https://custom-git.example.com/owner/repo/tree/feature-branch"
	urlBranchInfo, err := resolver.ResolveURL(url)

	assert.Nil(t, urlBranchInfo)
	require.Error(t, err)

	var groveErr *errors.GroveError
	require.ErrorAs(t, err, &groveErr)
	assert.Equal(t, errors.ErrCodeUnsupportedURL, groveErr.Code)
	assert.Contains(t, groveErr.Message, "failed to parse URL")
	mockGit.AssertExpectations(t)
}
