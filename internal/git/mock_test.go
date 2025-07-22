//go:build !integration
// +build !integration

package git

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sqve/grove/internal/testutils"
)

// TestMockGitExecutor tests the mock executor itself.
func TestMockGitExecutor(t *testing.T) {
	mock := testutils.NewMockGitExecutor()

	// Test default behavior
	_, err := mock.Execute("unknown", "command")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unhandled git command")

	// Test configured response
	mock.SetSuccessResponse("test", "output")
	output, err := mock.Execute("test")
	require.NoError(t, err)
	assert.Equal(t, "output", output)

	// Test error response
	mock.SetErrorResponseWithMessage("fail", "test error")
	_, err = mock.Execute("fail")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "test error")

	// Test command tracking
	assert.Equal(t, 3, mock.CallCount)
	assert.True(t, mock.HasCommand("test"))
	assert.False(t, mock.HasCommand("nonexistent"))
}

func TestCloneBareWithExecutor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tests := []struct {
		name        string
		repoURL     string
		targetDir   string
		mockOutput  string
		mockError   error
		expectError bool
	}{
		{
			name:        "successful clone",
			repoURL:     "https://github.com/user/repo.git",
			targetDir:   "/tmp/test",
			mockOutput:  "",
			mockError:   nil,
			expectError: false,
		},
		{
			name:        "clone failure",
			repoURL:     "https://invalid.com/repo.git",
			targetDir:   "/tmp/test",
			mockOutput:  "",
			mockError:   fmt.Errorf("clone failed"),
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := testutils.NewMockGitExecutor()
			mock.SetResponse("clone --bare", tt.mockOutput, tt.mockError)

			err := CloneBareWithExecutor(mock, tt.repoURL, tt.targetDir)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify the correct command was called
			assert.True(t, mock.HasCommand("clone", "--bare", tt.repoURL, tt.targetDir))
		})
	}
}

func TestConfigureRemoteTrackingWithExecutor(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tests := []struct {
		name         string
		configError  error
		fetchError   error
		expectError  bool
		expectConfig bool
		expectFetch  bool
	}{
		{
			name:         "successful configuration",
			configError:  nil,
			fetchError:   nil,
			expectError:  false,
			expectConfig: true,
			expectFetch:  true,
		},
		{
			name:         "config command fails",
			configError:  fmt.Errorf("config failed"),
			fetchError:   nil,
			expectError:  true,
			expectConfig: true,
			expectFetch:  false,
		},
		{
			name:         "fetch command fails",
			configError:  nil,
			fetchError:   fmt.Errorf("fetch failed"),
			expectError:  true,
			expectConfig: true,
			expectFetch:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := testutils.NewMockGitExecutor()
			mock.SetResponse("config", "", tt.configError)
			mock.SetResponse("fetch", "", tt.fetchError)

			err := ConfigureRemoteTrackingWithExecutor(mock, "origin")

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Verify expected commands were called
			if tt.expectConfig {
				assert.True(t, mock.HasCommand("config", "remote.origin.fetch", "+refs/heads/*:refs/remotes/origin/*"))
			}
			if tt.expectFetch {
				assert.True(t, mock.HasCommand("fetch"))
			}
		})
	}
}

func TestSetupUpstreamBranchesWithExecutor(t *testing.T) {
	tests := getSetupUpstreamTestCases()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runSetupUpstreamTest(t, tt)
		})
	}
}

type setupUpstreamTestCase struct {
	name           string
	branchOutput   string
	branchError    error
	upstreamErrors map[string]error
	expectError    bool
}

func getSetupUpstreamTestCases() []setupUpstreamTestCase {
	return []setupUpstreamTestCase{
		{
			name:         "successful setup with branches",
			branchOutput: "main\nfeature\ndevelop",
			branchError:  nil,
			upstreamErrors: map[string]error{
				"main":    nil,
				"feature": nil,
				"develop": nil,
			},
			expectError: false,
		},
		{
			name:         "for-each-ref fails",
			branchOutput: "",
			branchError:  fmt.Errorf("refs failed"),
			expectError:  true,
		},
		{
			name:         "no branches",
			branchOutput: "",
			branchError:  nil,
			expectError:  false,
		},
		{
			name:         "some upstream failures (should not error)",
			branchOutput: "main\nfeature",
			branchError:  nil,
			upstreamErrors: map[string]error{
				"main":    nil,
				"feature": fmt.Errorf("no remote branch"),
			},
			expectError: false,
		},
	}
}

func runSetupUpstreamTest(t *testing.T, tt setupUpstreamTestCase) {
	t.Helper()
	mock := testutils.NewMockGitExecutor()
	mock.SetResponse("for-each-ref", tt.branchOutput, tt.branchError)

	// Set up upstream responses
	for branch, err := range tt.upstreamErrors {
		pattern := fmt.Sprintf("branch --set-upstream-to=origin/%s %s", branch, branch)
		mock.SetResponse(pattern, "", err)
	}

	err := SetupUpstreamBranchesWithExecutor(mock, "origin")

	if tt.expectError {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
	}

	// Verify for-each-ref was called
	assert.True(t, mock.HasCommand("for-each-ref", "--format=%(refname:short)", "refs/heads"))
}
