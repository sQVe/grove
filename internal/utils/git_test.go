package utils

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockGitExecutor for testing utils functions.
type MockGitExecutor struct {
	Commands  [][]string
	Responses map[string]MockResponse
	CallCount int
}

type MockResponse struct {
	Output string
	Error  error
}

func NewMockGitExecutor() *MockGitExecutor {
	return &MockGitExecutor{
		Commands:  [][]string{},
		Responses: make(map[string]MockResponse),
		CallCount: 0,
	}
}

func (m *MockGitExecutor) Execute(args ...string) (string, error) {
	m.CallCount++
	m.Commands = append(m.Commands, args)

	cmdKey := fmt.Sprintf("%v", args)
	if response, exists := m.Responses[cmdKey]; exists {
		return response.Output, response.Error
	}

	// Default response for unmatched commands
	return "", fmt.Errorf("mock: unhandled git command: %v", args)
}

func (m *MockGitExecutor) SetResponse(args []string, output string, err error) {
	key := fmt.Sprintf("%v", args)
	m.Responses[key] = MockResponse{
		Output: output,
		Error:  err,
	}
}

func TestIsGitRepository(t *testing.T) {
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
			mock := NewMockGitExecutor()
			mock.SetResponse([]string{"rev-parse", "--git-dir"}, tt.gitOutput, tt.gitError)

			isRepo, err := IsGitRepository(mock)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectRepo, isRepo)

			// Verify correct git command was called
			assert.Equal(t, 1, mock.CallCount)
			assert.Equal(t, []string{"rev-parse", "--git-dir"}, mock.Commands[0])
		})
	}
}

func TestGetRepositoryRoot(t *testing.T) {
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
			mock := NewMockGitExecutor()
			mock.SetResponse([]string{"rev-parse", "--show-toplevel"}, tt.gitOutput, tt.gitError)

			root, err := GetRepositoryRoot(mock)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.expectRoot, root)

			// Verify correct git command was called
			assert.Equal(t, 1, mock.CallCount)
			assert.Equal(t, []string{"rev-parse", "--show-toplevel"}, mock.Commands[0])
		})
	}
}

func TestValidateRepository(t *testing.T) {
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
			mock := NewMockGitExecutor()
			mock.SetResponse([]string{"rev-parse", "--git-dir"}, tt.isRepoOutput, tt.isRepoError)
			mock.SetResponse([]string{"rev-parse", "HEAD"}, tt.headOutput, tt.headError)

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

func TestIsGitURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
		desc     string
	}{
		// Valid URLs
		{"https://github.com/user/repo.git", true, "HTTPS with .git extension"},
		{"https://github.com/user/repo", true, "HTTPS GitHub URL without .git"},
		{"https://gitlab.com/user/repo", true, "HTTPS GitLab URL"},
		{"http://example.com/repo.git", true, "HTTP with .git extension"},
		{"git@github.com:user/repo.git", true, "SSH GitHub URL"},
		{"ssh://git@github.com/user/repo.git", true, "SSH with full protocol"},
		{"git://example.com/repo.git", true, "Git protocol"},

		// Invalid URLs
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
