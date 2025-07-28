//go:build !integration
// +build !integration

package create

import (
	"errors"
	"testing"

	groveErrors "github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBranchResolverImpl_ResolveBranch_ExistingLocalBranch(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("branch -a --list", "  feature-branch\n  main\n  remotes/origin/feature-branch")

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveBranch("feature-branch", "", false)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "feature-branch", result.Name)
	assert.True(t, result.Exists)
}

func TestBranchResolverImpl_ResolveBranch_ExistingRemoteBranch(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("branch -a --list", "  main\n  remotes/origin/main\n  remotes/origin/feature-branch")
	mockExecutor.SetSuccessResponse("branch -r --list */feature-branch", "origin/feature-branch")

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveBranch("feature-branch", "", false)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "feature-branch", result.Name)
	assert.False(t, result.Exists) // Remote branches have Exists=false in actual implementation.
	assert.True(t, result.IsRemote)
	assert.Equal(t, "origin", result.RemoteName)
}

func TestBranchResolverImpl_ResolveBranch_NonexistentBranchWithoutCreate(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("branch -a --list", "  main\n  remotes/origin/main")
	mockExecutor.SetSuccessResponse("branch -r --list */nonexistent", "")

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveBranch("nonexistent", "", false)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitOperation, groveErr.Code)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestBranchResolverImpl_ResolveBranch_NonexistentBranchWithCreate(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("branch -a --list", "  main\n  remotes/origin/main")
	mockExecutor.SetSuccessResponse("branch -r --list */nonexistent", "")
	mockExecutor.SetSuccessResponse("checkout -b nonexistent main", "")

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveBranch("nonexistent", "main", true)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "nonexistent", result.Name)
	assert.True(t, result.Exists)
	assert.False(t, result.IsRemote)
}

func TestBranchResolverImpl_ResolveBranch_CreateBranchFailure(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("branch -a --list", "  main\n  remotes/origin/main")
	mockExecutor.SetSuccessResponse("branch -r --list */nonexistent", "")
	mockExecutor.SetErrorResponse("checkout -b nonexistent main", errors.New("failed to create branch"))

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveBranch("nonexistent", "main", true)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitOperation, groveErr.Code)
}

func TestBranchResolverImpl_ResolveURL_GitHubPR(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	resolver := NewBranchResolver(mockExecutor)

	url := "https://github.com/owner/repo/pull/123"
	result, err := resolver.ResolveURL(url)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "github", result.Platform)
	assert.Equal(t, "123", result.PRNumber)
	assert.Equal(t, "https://github.com/owner/repo.git", result.RepoURL)
	// PR URLs don't have a predetermined BranchName - it's resolved later.
	assert.Empty(t, result.BranchName)
}

func TestBranchResolverImpl_ResolveURL_GitLabMR(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	resolver := NewBranchResolver(mockExecutor)

	url := "https://gitlab.com/owner/repo/-/merge_requests/456"
	result, err := resolver.ResolveURL(url)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "gitlab", result.Platform)
	assert.Equal(t, "456", result.PRNumber)
	assert.Equal(t, "https://gitlab.com/owner/repo.git", result.RepoURL)
	// MR URLs don't have a predetermined BranchName - it's resolved later.
	assert.Empty(t, result.BranchName)
}

func TestBranchResolverImpl_ResolveURL_GitHubBranch(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	resolver := NewBranchResolver(mockExecutor)

	url := "https://github.com/owner/repo/tree/feature-branch"
	result, err := resolver.ResolveURL(url)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "github", result.Platform)
	assert.Equal(t, "feature-branch", result.BranchName)
}

func TestBranchResolverImpl_ResolveURL_BitbucketPR(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	resolver := NewBranchResolver(mockExecutor)

	url := "https://bitbucket.org/owner/repo/pull-requests/789"
	result, err := resolver.ResolveURL(url)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "bitbucket", result.Platform)
	assert.Equal(t, "789", result.PRNumber)
	assert.Equal(t, "https://bitbucket.org/owner/repo.git", result.RepoURL)
	// PR URLs don't have a predetermined BranchName - it's resolved later.
	assert.Empty(t, result.BranchName)
}

func TestBranchResolverImpl_ResolveURL_UnsupportedURL(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	resolver := NewBranchResolver(mockExecutor)

	url := "https://example.com/invalid/url"
	result, err := resolver.ResolveURL(url)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeUnsupportedURL, groveErr.Code)
}

func TestBranchResolverImpl_ResolveURL_InvalidURL(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	resolver := NewBranchResolver(mockExecutor)

	url := "not-a-url"
	result, err := resolver.ResolveURL(url)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeUnsupportedURL, groveErr.Code)
}

func TestBranchResolverImpl_ResolveRemoteBranch_OriginBranch(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("remote", "origin\nupstream")
	mockExecutor.SetSuccessResponse("fetch origin", "")
	mockExecutor.SetSuccessResponse("branch -r --list origin/feature-branch", "origin/feature-branch")

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveRemoteBranch("origin/feature-branch")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "feature-branch", result.Name)
	assert.False(t, result.Exists) // Remote branches have Exists=false in actual implementation.
	assert.True(t, result.IsRemote)
	assert.Equal(t, "origin", result.RemoteName)
	assert.Equal(t, "origin/feature-branch", result.TrackingBranch)
}

func TestBranchResolverImpl_ResolveRemoteBranch_UpstreamBranch(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("remote", "origin\nupstream")
	mockExecutor.SetSuccessResponse("fetch upstream", "")
	mockExecutor.SetSuccessResponse("branch -r --list upstream/hotfix", "upstream/hotfix")

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveRemoteBranch("upstream/hotfix")

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "hotfix", result.Name)
	assert.False(t, result.Exists) // Remote branches have Exists=false in actual implementation.
	assert.True(t, result.IsRemote)
	assert.Equal(t, "upstream", result.RemoteName)
	assert.Equal(t, "upstream/hotfix", result.TrackingBranch)
}

func TestBranchResolverImpl_ResolveRemoteBranch_InvalidFormat(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveRemoteBranch("invalid-format")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitOperation, groveErr.Code)
	assert.Contains(t, err.Error(), "invalid remote branch format")
}

func TestBranchResolverImpl_ResolveRemoteBranch_RemoteNotFound(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("remote", "origin")

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveRemoteBranch("nonexistent/branch")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitOperation, groveErr.Code)
	assert.Contains(t, err.Error(), "remote 'nonexistent' not found")
}

func TestBranchResolverImpl_ResolveRemoteBranch_BranchNotFound(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("remote", "origin")
	mockExecutor.SetSuccessResponse("fetch origin", "")
	mockExecutor.SetSuccessResponse("branch -r --list origin/nonexistent", "")

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveRemoteBranch("origin/nonexistent")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitOperation, groveErr.Code)
	assert.Contains(t, err.Error(), "branch 'nonexistent' not found on remote 'origin'")
}

func TestBranchResolverImpl_ResolveRemoteBranch_FetchFailure(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()
	mockExecutor.SetSuccessResponse("remote", "origin")
	mockExecutor.SetErrorResponse("fetch origin", errors.New("fetch failed"))

	resolver := NewBranchResolver(mockExecutor)

	result, err := resolver.ResolveRemoteBranch("origin/feature-branch")

	require.Error(t, err)
	assert.Nil(t, result)
	assert.IsType(t, &groveErrors.GroveError{}, err)
	groveErr := err.(*groveErrors.GroveError)
	assert.Equal(t, groveErrors.ErrCodeGitOperation, groveErr.Code)
}

// Note: Internal method tests removed as they test unexported functions
// The public interface tests above provide sufficient coverage.
