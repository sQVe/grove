package git

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestDetectDefaultBranch(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test in short mode")
	}

	tests := []struct {
		name     string
		setup    func() *testutils.MockGitExecutor
		expected string
	}{
		{
			name: "local remote HEAD available",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
				return mock
			},
			expected: "main",
		},
		{
			name: "local remote HEAD with master",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/master")
				return mock
			},
			expected: "master",
		},
		{
			name: "local remote HEAD fails, use current branch",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("symbolic-ref refs/remotes/origin/HEAD", "no such ref")
				mock.SetSuccessResponse("branch --show-current", "feature/auth")
				return mock
			},
			expected: "feature/auth",
		},
		{
			name: "local methods fail, use remote symref",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("symbolic-ref refs/remotes/origin/HEAD", "no such ref")
				mock.SetErrorResponseWithMessage("branch --show-current", "not a git repository")
				mock.SetSuccessResponse("ls-remote --symref origin HEAD", "ref: refs/heads/develop\tHEAD\n1234567890abcdef\tHEAD")
				return mock
			},
			expected: "develop",
		},
		{
			name: "remote symref fails, use remote show",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("symbolic-ref refs/remotes/origin/HEAD", "no such ref")
				mock.SetErrorResponseWithMessage("branch --show-current", "not a git repository")
				mock.SetErrorResponseWithMessage("ls-remote --symref origin HEAD", "network error")
				mock.SetSuccessResponse("remote show origin", "* remote origin\n  Fetch URL: https://github.com/user/repo.git\n  Push  URL: https://github.com/user/repo.git\n  HEAD branch: master\n  Remote branches:\n    master tracked\n    develop tracked")
				return mock
			},
			expected: "master",
		},
		{
			name: "network fails, use common branch pattern",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("symbolic-ref refs/remotes/origin/HEAD", "no such ref")
				mock.SetErrorResponseWithMessage("branch --show-current", "not a git repository")
				mock.SetErrorResponseWithMessage("ls-remote --symref origin HEAD", "network error")
				mock.SetErrorResponseWithMessage("remote show origin", "network error")
				mock.SetSuccessResponse("branch -r", "  origin/develop\n  origin/feature/auth\n  origin/main\n  origin/staging")
				return mock
			},
			expected: "main",
		},
		{
			name: "only master available in pattern matching",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("symbolic-ref refs/remotes/origin/HEAD", "no such ref")
				mock.SetErrorResponseWithMessage("branch --show-current", "not a git repository")
				mock.SetErrorResponseWithMessage("ls-remote --symref origin HEAD", "network error")
				mock.SetErrorResponseWithMessage("remote show origin", "network error")
				mock.SetSuccessResponse("branch -r", "  origin/feature/auth\n  origin/master\n  origin/staging")
				return mock
			},
			expected: "master",
		},
		{
			name: "no common patterns, use first remote branch",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("symbolic-ref refs/remotes/origin/HEAD", "no such ref")
				mock.SetErrorResponseWithMessage("branch --show-current", "not a git repository")
				mock.SetErrorResponseWithMessage("ls-remote --symref origin HEAD", "network error")
				mock.SetErrorResponseWithMessage("remote show origin", "network error")
				mock.SetSuccessResponse("branch -r", "  origin/feature/auth\n  origin/production\n  origin/staging")
				return mock
			},
			expected: "feature/auth",
		},
		{
			name: "all methods fail, use hard-coded default",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("symbolic-ref refs/remotes/origin/HEAD", "no such ref")
				mock.SetErrorResponseWithMessage("branch --show-current", "not a git repository")
				mock.SetErrorResponseWithMessage("ls-remote --symref origin HEAD", "network error")
				mock.SetErrorResponseWithMessage("remote show origin", "network error")
				mock.SetErrorResponseWithMessage("branch -r", "no remote branches")
				return mock
			},
			expected: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectDefaultBranch(tt.setup(), "origin")
			assert.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckLocalRemoteHead(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *testutils.MockGitExecutor
		expected string
	}{
		{
			name: "valid remote HEAD",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/main")
				return mock
			},
			expected: "main",
		},
		{
			name: "valid remote HEAD with master",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/master")
				return mock
			},
			expected: "master",
		},
		{
			name: "command fails",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("symbolic-ref refs/remotes/origin/HEAD", "no such ref")
				return mock
			},
			expected: "",
		},
		{
			name: "invalid output format",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("symbolic-ref refs/remotes/origin/HEAD", "invalid-format")
				return mock
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.DefaultConfig())
			result := checkLocalRemoteHead(tt.setup(), log, "origin")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckCurrentBranch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *testutils.MockGitExecutor
		expected string
	}{
		{
			name: "valid current branch",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch --show-current", "feature/auth")
				return mock
			},
			expected: "feature/auth",
		},
		{
			name: "main branch",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch --show-current", "main")
				return mock
			},
			expected: "main",
		},
		{
			name: "command fails",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("branch --show-current", "not a git repository")
				return mock
			},
			expected: "",
		},
		{
			name: "empty output",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch --show-current", "")
				return mock
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.DefaultConfig())
			result := checkCurrentBranch(tt.setup(), log)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckRemoteSymref(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *testutils.MockGitExecutor
		expected string
	}{
		{
			name: "valid symref output",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("ls-remote --symref origin HEAD", "ref: refs/heads/main\tHEAD\n1234567890abcdef\tHEAD")
				return mock
			},
			expected: "main",
		},
		{
			name: "master branch",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("ls-remote --symref origin HEAD", "ref: refs/heads/master\tHEAD\n1234567890abcdef\tHEAD")
				return mock
			},
			expected: "master",
		},
		{
			name: "develop branch",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("ls-remote --symref origin HEAD", "ref: refs/heads/develop\tHEAD\n1234567890abcdef\tHEAD")
				return mock
			},
			expected: "develop",
		},
		{
			name: "command fails",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("ls-remote --symref origin HEAD", "network error")
				return mock
			},
			expected: "",
		},
		{
			name: "invalid output format",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("ls-remote --symref origin HEAD", "invalid-format")
				return mock
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			mock := tt.setup()
			log := logger.New(logger.DefaultConfig())
			result := checkRemoteSymref(mock, log, ctx, "origin")

			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCheckRemoteShow(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *testutils.MockGitExecutor
		expected string
	}{
		{
			name: "valid remote show output",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("remote show origin", "* remote origin\n  Fetch URL: https://github.com/user/repo.git\n  Push  URL: https://github.com/user/repo.git\n  HEAD branch: main\n  Remote branches:\n    main tracked\n    develop tracked")
				return mock
			},
			expected: "main",
		},
		{
			name: "master branch",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("remote show origin", "* remote origin\n  HEAD branch: master\n  Remote branches:\n    master tracked")
				return mock
			},
			expected: "master",
		},
		{
			name: "command fails",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("remote show origin", "network error")
				return mock
			},
			expected: "",
		},
		{
			name: "no HEAD branch in output",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("remote show origin", "* remote origin\n  Remote branches:\n    main tracked")
				return mock
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
			defer cancel()

			log := logger.New(logger.DefaultConfig())
			result := checkRemoteShow(tt.setup(), log, ctx, "origin")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFindCommonBranchPattern(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *testutils.MockGitExecutor
		expected string
	}{
		{
			name: "main branch available",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/develop\n  origin/feature/auth\n  origin/main\n  origin/staging")
				return mock
			},
			expected: "main",
		},
		{
			name: "master branch available (no main)",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/develop\n  origin/feature/auth\n  origin/master\n  origin/staging")
				return mock
			},
			expected: "master",
		},
		{
			name: "develop branch available (no main/master)",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/develop\n  origin/feature/auth\n  origin/staging")
				return mock
			},
			expected: "develop",
		},
		{
			name: "trunk branch available",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/feature/auth\n  origin/staging\n  origin/trunk")
				return mock
			},
			expected: "trunk",
		},
		{
			name: "main takes precedence over master",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/develop\n  origin/main\n  origin/master\n  origin/staging")
				return mock
			},
			expected: "main",
		},
		{
			name: "no common patterns",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/feature/auth\n  origin/production\n  origin/staging")
				return mock
			},
			expected: "",
		},
		{
			name: "command fails",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("branch -r", "no remote branches")
				return mock
			},
			expected: "",
		},
		{
			name: "HEAD branch filtered out",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/HEAD -> origin/main\n  origin/develop\n  origin/main\n  origin/staging")
				return mock
			},
			expected: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.DefaultConfig())
			result := findCommonBranchPattern(tt.setup(), log, "origin")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetFirstRemoteBranch(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *testutils.MockGitExecutor
		expected string
	}{
		{
			name: "multiple remote branches",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/develop\n  origin/feature/auth\n  origin/main\n  origin/staging")
				return mock
			},
			expected: "develop",
		},
		{
			name: "single remote branch",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/custom-branch")
				return mock
			},
			expected: "custom-branch",
		},
		{
			name: "HEAD branch filtered out",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/HEAD -> origin/main\n  origin/develop\n  origin/main")
				return mock
			},
			expected: "develop",
		},
		{
			name: "command fails",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetErrorResponseWithMessage("branch -r", "no remote branches")
				return mock
			},
			expected: "",
		},
		{
			name: "no remote branches",
			setup: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "")
				return mock
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.DefaultConfig())
			result := getFirstRemoteBranch(tt.setup(), log, "origin")
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsValidBranchName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Valid branch names
		{"simple name", "main", true},
		{"with slash", "feature/auth", true},
		{"with numbers", "v1.2.3", true},
		{"with underscore", "feature_branch", true},
		{"with dots", "release.1.0", true},
		{"hierarchical", "releases/v1.0/hotfix", true},
		{"complex valid", "feature/user-auth_system.v2", true},

		// Invalid: empty or whitespace
		{"empty string", "", false},
		{"only whitespace", "   ", false},
		{"leading whitespace", " main", false},
		{"trailing whitespace", "main ", false},

		// Invalid: single character @
		{"single @", "@", false},

		// Invalid: begins with dash
		{"starts with dash", "-feature", false},
		{"starts with dash complex", "-feature/auth", false},

		// Invalid: begins or ends with /
		{"starts with slash", "/feature", false},
		{"ends with slash", "feature/", false},
		{"both slashes", "/feature/", false},

		// Invalid: ends with .
		{"ends with dot", "feature.", false},
		{"ends with dot complex", "feature/auth.", false},

		// Invalid: consecutive dots
		{"consecutive dots", "feature..auth", false},
		{"consecutive dots start", "..main", false},
		{"consecutive dots end", "main..", false},

		// Invalid: @{ sequence
		{"contains @{", "feature@{", false},
		{"contains @{ middle", "fea@{ture", false},

		// Invalid: multiple consecutive slashes
		{"double slash", "feature//auth", false},
		{"triple slash", "feature///auth", false},

		// Invalid: forbidden characters
		{"contains space", "feature auth", false},
		{"contains tilde", "feature~auth", false},
		{"contains caret", "feature^auth", false},
		{"contains colon", "feature:auth", false},
		{"contains question", "feature?auth", false},
		{"contains asterisk", "feature*auth", false},
		{"contains bracket", "feature[auth", false},
		{"contains backslash", "feature\\auth", false},
		{"contains DEL", "feature\x7Fauth", false},
		{"contains control char", "feature\x01auth", false},

		// Invalid: component rules
		{"component starts with dot", "feature/.auth", false},
		{"component starts with dot start", ".feature/auth", false},
		{"component ends with .lock", "feature/auth.lock", false},
		{"component ends with .lock start", "auth.lock/feature", false},

		// Edge cases
		{"just dot", ".", false},
		{"just slash", "/", false},
		{"just dots", "..", false},
		{"valid with valid chars", "feature-auth_test.v1", true},
		{"valid single char", "a", true},
		{"valid single char num", "1", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log := logger.New(logger.DefaultConfig())
			result := isValidBranchName(log, tt.input)
			assert.Equal(t, tt.expected, result, "Branch name: %q", tt.input)
		})
	}
}

func TestBranchValidationIntegration(t *testing.T) {
	// Test that all detection functions now validate branch names
	tests := []struct {
		name        string
		setupMock   func() *testutils.MockGitExecutor
		expectEmpty bool
		description string
	}{
		{
			name: "checkLocalRemoteHead validates branch name",
			setupMock: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("symbolic-ref refs/remotes/origin/HEAD", "refs/remotes/origin/-invalid")
				return mock
			},
			expectEmpty: true,
			description: "Should reject branch name starting with dash",
		},
		{
			name: "checkCurrentBranch validates branch name",
			setupMock: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch --show-current", "invalid~branch")
				return mock
			},
			expectEmpty: true,
			description: "Should reject branch name with tilde",
		},
		{
			name: "checkRemoteSymref validates branch name",
			setupMock: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("ls-remote --symref origin HEAD", "ref: refs/heads/bad..branch\tHEAD")
				return mock
			},
			expectEmpty: true,
			description: "Should reject branch name with consecutive dots",
		},
		{
			name: "checkRemoteShow validates branch name",
			setupMock: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("remote show origin", "* remote origin\n  HEAD branch: invalid*branch\n")
				return mock
			},
			expectEmpty: true,
			description: "Should reject branch name with asterisk",
		},
		{
			name: "findCommonBranchPattern validates branch name",
			setupMock: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/main\n  origin/invalid@{branch")
				return mock
			},
			expectEmpty: false, // Should find valid 'main' branch
			description: "Should find valid main branch and skip invalid one",
		},
		{
			name: "getFirstRemoteBranch validates branch name",
			setupMock: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetSuccessResponse("branch -r", "  origin/-invalid\n  origin/valid-branch")
				return mock
			},
			expectEmpty: false, // Should find valid 'valid-branch' and skip invalid one
			description: "Should skip invalid branch and find valid one",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := tt.setupMock()
			ctx := context.Background()
			log := logger.New(logger.DefaultConfig())

			var result string
			switch {
			case strings.Contains(tt.name, "checkLocalRemoteHead"):
				result = checkLocalRemoteHead(mock, log, "origin")
			case strings.Contains(tt.name, "checkCurrentBranch"):
				result = checkCurrentBranch(mock, log)
			case strings.Contains(tt.name, "checkRemoteSymref"):
				result = checkRemoteSymref(mock, log, ctx, "origin")
			case strings.Contains(tt.name, "checkRemoteShow"):
				result = checkRemoteShow(mock, log, ctx, "origin")
			case strings.Contains(tt.name, "findCommonBranchPattern"):
				result = findCommonBranchPattern(mock, log, "origin")
			case strings.Contains(tt.name, "getFirstRemoteBranch"):
				result = getFirstRemoteBranch(mock, log, "origin")
			}

			if tt.expectEmpty {
				assert.Empty(t, result, tt.description)
			} else {
				assert.NotEmpty(t, result, tt.description)
				assert.True(t, isValidBranchName(log, result), "Returned branch name should be valid: %q", result)
			}
		})
	}
}
