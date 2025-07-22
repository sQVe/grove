package git

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorktreeInfo_Struct(t *testing.T) {
	now := time.Now()

	info := WorktreeInfo{
		Path:         "/repo/test",
		Branch:       "main",
		Head:         "abc123",
		IsCurrent:    true,
		LastActivity: now,
		Status: WorktreeStatus{
			Modified:  2,
			Staged:    1,
			Untracked: 3,
			IsClean:   false,
		},
		Remote: RemoteStatus{
			HasRemote: true,
			Ahead:     1,
			Behind:    2,
			IsMerged:  false,
		},
	}

	assert.Equal(t, "/repo/test", info.Path)
	assert.Equal(t, "main", info.Branch)
	assert.Equal(t, "abc123", info.Head)
	assert.True(t, info.IsCurrent)
	assert.Equal(t, now, info.LastActivity)
	assert.Equal(t, 2, info.Status.Modified)
	assert.Equal(t, 1, info.Status.Staged)
	assert.Equal(t, 3, info.Status.Untracked)
	assert.False(t, info.Status.IsClean)
	assert.True(t, info.Remote.HasRemote)
	assert.Equal(t, 1, info.Remote.Ahead)
	assert.Equal(t, 2, info.Remote.Behind)
	assert.False(t, info.Remote.IsMerged)
}

func TestParseWorktreePorcelain(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []WorktreeInfo
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []WorktreeInfo{},
		},
		{
			name: "single worktree",
			input: `worktree /repo/main
HEAD abc123def456
branch refs/heads/main

`,
			expected: []WorktreeInfo{
				{
					Path:   "/repo/main",
					Branch: "refs/heads/main",
					Head:   "abc123def456",
				},
			},
		},
		{
			name: "multiple worktrees",
			input: `worktree /repo/main
HEAD abc123def456
branch refs/heads/main

worktree /repo/feature
HEAD def456abc123
branch refs/heads/feature/test

`,
			expected: []WorktreeInfo{
				{
					Path:   "/repo/main",
					Branch: "refs/heads/main",
					Head:   "abc123def456",
				},
				{
					Path:   "/repo/feature",
					Branch: "refs/heads/feature/test",
					Head:   "def456abc123",
				},
			},
		},
		{
			name: "detached HEAD",
			input: `worktree /repo/detached
HEAD abc123def456

`,
			expected: []WorktreeInfo{
				{
					Path:   "/repo/detached",
					Branch: "",
					Head:   "abc123def456",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseWorktreePorcelain(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCurrentWorktreePath(t *testing.T) {
	tests := []struct {
		name          string
		mockResponse  string
		mockError     error
		expectedPath  string
		expectedError bool
	}{
		{
			name:         "successful response",
			mockResponse: "/repo/current/worktree\n",
			expectedPath: "/repo/current/worktree",
		},
		{
			name:         "git error",
			mockError:    fmt.Errorf("not a git repository"),
			expectedPath: "", // Function returns empty string on error, not an error
		},
		{
			name:         "response with whitespace",
			mockResponse: "  /repo/current/worktree  \n\n",
			expectedPath: "/repo/current/worktree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			// Get current working directory to set up the proper mock command
			cwd, err := os.Getwd()
			require.NoError(t, err)
			expectedCmd := fmt.Sprintf("-C %s rev-parse --show-toplevel", cwd)

			if tt.mockError != nil {
				mockExecutor.SetErrorResponse(expectedCmd, tt.mockError)
			} else {
				mockExecutor.SetResponse(expectedCmd, tt.mockResponse, nil)
			}

			path, err := getCurrentWorktreePath(mockExecutor)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, path)
			}
		})
	}
}

func TestGetWorktreeStatus(t *testing.T) {
	tests := []struct {
		name           string
		mockResponse   string
		mockError      error
		expectedStatus WorktreeStatus
		expectedError  bool
		setupDir       bool
	}{
		{
			name:         "clean worktree",
			mockResponse: "",
			expectedStatus: WorktreeStatus{
				Modified:  0,
				Staged:    0,
				Untracked: 0,
				IsClean:   true,
			},
			setupDir: true,
		},
		{
			name: "mixed status",
			mockResponse: ` M file1.txt
A  file2.txt
 D file3.txt
?? untracked.txt
?? another_untracked.txt`,
			expectedStatus: WorktreeStatus{
				Modified:  2, // file1.txt (modified), file3.txt (deleted)
				Staged:    1, // file2.txt (added)
				Untracked: 2, // untracked.txt, another_untracked.txt
				IsClean:   false,
			},
			setupDir: true,
		},
		{
			name: "only staged changes",
			mockResponse: `A  new_file.txt
M  modified_file.txt`,
			expectedStatus: WorktreeStatus{
				Modified:  0,
				Staged:    2,
				Untracked: 0,
				IsClean:   false,
			},
			setupDir: true,
		},
		{
			name:          "git status error",
			mockError:     fmt.Errorf("git status failed"),
			expectedError: true,
			setupDir:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testDir string
			if tt.setupDir {
				// Create temporary directory for test
				var err error
				testDir, err = os.MkdirTemp("", "grove-test-*")
				require.NoError(t, err)
				defer func() { _ = os.RemoveAll(testDir) }()
			}

			mockExecutor := testutils.NewMockGitExecutor()
			expectedCmd := fmt.Sprintf("-C %s status --porcelain", testDir)
			if tt.mockError != nil {
				mockExecutor.SetErrorResponse(expectedCmd, tt.mockError)
			} else {
				mockExecutor.SetResponse(expectedCmd, tt.mockResponse, nil)
			}

			status, err := getWorktreeStatus(mockExecutor, testDir)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedStatus, status)
			}
		})
	}
}

func TestGetRemoteStatus(t *testing.T) {
	tests := []struct {
		name             string
		branchName       string
		upstreamResponse string
		upstreamError    error
		countResponse    string
		countError       error
		expectedRemote   RemoteStatus
		setupDir         bool
	}{
		{
			name:           "no branch name",
			branchName:     "",
			expectedRemote: RemoteStatus{},
		},
		{
			name:          "no upstream",
			branchName:    "main",
			upstreamError: fmt.Errorf("no upstream"),
			expectedRemote: RemoteStatus{
				HasRemote: false,
			},
			setupDir: true,
		},
		{
			name:             "has upstream, up to date",
			branchName:       "main",
			upstreamResponse: "origin/main",
			countResponse:    "0\t0",
			expectedRemote: RemoteStatus{
				HasRemote: true,
				Ahead:     0,
				Behind:    0,
				IsMerged:  false,
			},
			setupDir: true,
		},
		{
			name:             "ahead of upstream",
			branchName:       "feature",
			upstreamResponse: "origin/main",
			countResponse:    "3\t0",
			expectedRemote: RemoteStatus{
				HasRemote: true,
				Ahead:     3,
				Behind:    0,
				IsMerged:  false,
			},
			setupDir: true,
		},
		{
			name:             "behind upstream",
			branchName:       "feature",
			upstreamResponse: "origin/main",
			countResponse:    "0\t2",
			expectedRemote: RemoteStatus{
				HasRemote: true,
				Ahead:     0,
				Behind:    2,
				IsMerged:  false,
			},
			setupDir: true,
		},
		{
			name:             "diverged from upstream",
			branchName:       "feature",
			upstreamResponse: "origin/main",
			countResponse:    "2\t1",
			expectedRemote: RemoteStatus{
				HasRemote: true,
				Ahead:     2,
				Behind:    1,
				IsMerged:  false,
			},
			setupDir: true,
		},
		{
			name:             "upstream exists but count fails",
			branchName:       "feature",
			upstreamResponse: "origin/main",
			countError:       fmt.Errorf("count failed"),
			expectedRemote: RemoteStatus{
				HasRemote: true,
				Ahead:     0,
				Behind:    0,
				IsMerged:  false,
			},
			setupDir: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testDir string
			if tt.setupDir {
				var err error
				testDir, err = os.MkdirTemp("", "grove-test-*")
				require.NoError(t, err)
				defer func() { _ = os.RemoveAll(testDir) }()
			}

			mockExecutor := testutils.NewMockGitExecutor()

			// Set up mock responses with -C flag that the implementation uses
			upstreamCmd := fmt.Sprintf("-C %s rev-parse --abbrev-ref %s@{upstream}", testDir, tt.branchName)
			if tt.upstreamError != nil {
				mockExecutor.SetErrorResponse(upstreamCmd, tt.upstreamError)
			} else {
				mockExecutor.SetResponse(upstreamCmd, tt.upstreamResponse, nil)
			}

			if tt.upstreamResponse != "" {
				countCmd := fmt.Sprintf("-C %s rev-list --count --left-right %s...%s", testDir, tt.branchName, tt.upstreamResponse)
				if tt.countError != nil {
					mockExecutor.SetErrorResponse(countCmd, tt.countError)
				} else {
					mockExecutor.SetResponse(countCmd, tt.countResponse, nil)
				}
			}

			remote, err := getRemoteStatus(mockExecutor, testDir, tt.branchName)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedRemote, remote)
		})
	}
}

func TestListWorktrees_Integration(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	// Mock the porcelain output
	porcelainOutput := `worktree /repo/main
HEAD abc123def456
branch refs/heads/main

worktree /repo/feature
HEAD def456abc123
branch refs/heads/feature/test

`
	mockExecutor.SetResponse("worktree list --porcelain", porcelainOutput, nil)

	// Mock current worktree detection - this needs to match current working directory
	cwd, _ := os.Getwd()
	mockExecutor.SetResponse("-C "+cwd+" rev-parse --show-toplevel", "/repo/main", nil)

	// Mock git status for each worktree
	mockExecutor.SetResponse("-C /repo/main status --porcelain", "", nil)
	mockExecutor.SetResponse("-C /repo/feature status --porcelain", "", nil)

	// Mock upstream checking with -C flag (no upstream for simplicity)
	mockExecutor.SetErrorResponse("-C /repo/main rev-parse --abbrev-ref main@{upstream}", fmt.Errorf("no upstream"))
	mockExecutor.SetErrorResponse("-C /repo/feature rev-parse --abbrev-ref feature/test@{upstream}", fmt.Errorf("no upstream"))

	// Create temporary directories to simulate worktrees
	tempDir, err := os.MkdirTemp("", "grove-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	mainDir := filepath.Join(tempDir, "main")
	featureDir := filepath.Join(tempDir, "feature")

	err = os.MkdirAll(mainDir, 0o755)
	require.NoError(t, err)
	err = os.MkdirAll(featureDir, 0o755)
	require.NoError(t, err)

	// Create some files with different timestamps
	mainFile := filepath.Join(mainDir, "test.txt")
	featureFile := filepath.Join(featureDir, "test.txt")

	err = os.WriteFile(mainFile, []byte("content"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(featureFile, []byte("content"), 0o644)
	require.NoError(t, err)

	// Test the enhanced ListWorktrees function
	// Note: This test is limited because we can't easily mock the file system operations
	// In a real scenario, you might want to add dependency injection for file operations
	worktrees, err := ListWorktrees(mockExecutor)

	assert.NoError(t, err)
	assert.Len(t, worktrees, 2)

	// Verify basic parsing worked
	assert.Equal(t, "/repo/main", worktrees[0].Path)
	assert.Equal(t, "refs/heads/main", worktrees[0].Branch)
	assert.Equal(t, "abc123def456", worktrees[0].Head)

	assert.Equal(t, "/repo/feature", worktrees[1].Path)
	assert.Equal(t, "refs/heads/feature/test", worktrees[1].Branch)
	assert.Equal(t, "def456abc123", worktrees[1].Head)

	// Verify current worktree detection
	assert.True(t, worktrees[0].IsCurrent)
	assert.False(t, worktrees[1].IsCurrent)
}

func TestListWorktreesPaths_BackwardCompatibility(t *testing.T) {
	mockExecutor := testutils.NewMockGitExecutor()

	porcelainOutput := `worktree /repo/main
HEAD abc123
branch refs/heads/main

worktree /repo/feature
HEAD def456
branch refs/heads/feature

`
	mockExecutor.SetResponse("worktree list --porcelain", porcelainOutput, nil)
	mockExecutor.SetResponse("rev-parse --show-toplevel", "/repo/main", nil)
	mockExecutor.SetResponse("status --porcelain", "", nil)
	mockExecutor.SetErrorResponse("rev-parse --abbrev-ref main@{upstream}", fmt.Errorf("no upstream"))
	mockExecutor.SetErrorResponse("rev-parse --abbrev-ref feature@{upstream}", fmt.Errorf("no upstream"))

	paths, err := ListWorktreesPaths(mockExecutor)

	assert.NoError(t, err)
	assert.Len(t, paths, 2)
	assert.Equal(t, "/repo/main", paths[0])
	assert.Equal(t, "/repo/feature", paths[1])
}

func TestGetLastActivity_MockFileSystem(t *testing.T) {
	// Create a temporary directory structure for testing
	tempDir, err := os.MkdirTemp("", "grove-test-*")
	require.NoError(t, err)
	defer func() { _ = os.RemoveAll(tempDir) }()

	// Create some test files with different timestamps
	testFile1 := filepath.Join(tempDir, "file1.txt")
	testFile2 := filepath.Join(tempDir, "file2.txt")
	gitDir := filepath.Join(tempDir, ".git")
	hiddenFile := filepath.Join(tempDir, ".hidden")

	// Create files
	err = os.WriteFile(testFile1, []byte("content1"), 0o644)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	err = os.WriteFile(testFile2, []byte("content2"), 0o644)
	require.NoError(t, err)

	err = os.MkdirAll(gitDir, 0o755)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(gitDir, "config"), []byte("git config"), 0o644)
	require.NoError(t, err)

	err = os.WriteFile(hiddenFile, []byte("hidden"), 0o644)
	require.NoError(t, err)

	// Test the function
	lastActivity, err := getLastActivity(tempDir)
	require.NoError(t, err)

	// Should be the timestamp of file2.txt (most recent)
	file2Info, err := os.Stat(testFile2)
	require.NoError(t, err)

	// Allow for small differences due to file system precision
	timeDiff := lastActivity.Sub(file2Info.ModTime())
	assert.True(t, timeDiff >= 0 && timeDiff < time.Second)
}

func TestSplitLines_EnhancedCases(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: nil,
		},
		{
			name:     "single line no newline",
			input:    "line1",
			expected: []string{"line1"},
		},
		{
			name:     "single line with newline",
			input:    "line1\n",
			expected: []string{"line1"},
		},
		{
			name:     "multiple lines unix",
			input:    "line1\nline2\nline3\n",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "multiple lines windows",
			input:    "line1\r\nline2\r\nline3\r\n",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "mixed line endings",
			input:    "line1\nline2\r\nline3\n",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			name:     "empty lines",
			input:    "line1\n\nline3\n",
			expected: []string{"line1", "", "line3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitLines(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
