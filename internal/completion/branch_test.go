//go:build !integration
// +build !integration

package completion

import (
	"errors"
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/testutils"
)

func TestBranchCompletion(t *testing.T) {
	tests := []struct {
		name              string
		mockFunc          func() *testutils.MockGitExecutor
		toComplete        string
		expectedResults   []string
		expectedDirective cobra.ShellCompDirective
		skipRepoCheck     bool
	}{
		{
			name: "successful branch completion",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("rev-parse --git-dir", "", nil)
				mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop\nfeature/test", nil)
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main\norigin/develop\norigin/feature/auth", nil)
				return mock
			},
			toComplete:        "",
			expectedResults:   []string{"main", "develop", "feature/auth", "feature/test"},
			expectedDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "filtered branch completion",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("rev-parse --git-dir", "", nil)
				mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop\nfeature/test", nil)
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main\norigin/develop\norigin/feature/auth", nil)
				return mock
			},
			toComplete:        "f",
			expectedResults:   []string{"feature/auth", "feature/test"},
			expectedDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "not in git repo",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("rev-parse --git-dir", "", errors.New("exit 128"))
				return mock
			},
			toComplete:        "",
			expectedResults:   nil,
			expectedDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "git command error",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("rev-parse --git-dir", "", nil)
				mock.SetResponse("branch --format=%(refname:short)", "", errors.New("git error"))
				mock.SetResponse("branch -r --format=%(refname:short)", "", errors.New("git error"))
				return mock
			},
			toComplete:        "",
			expectedResults:   nil,
			expectedDirective: cobra.ShellCompDirectiveError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear cache before each test
			GlobalCache.Clear()

			ctx := NewCompletionContext(tt.mockFunc())
			cmd := &cobra.Command{}
			args := []string{}

			results, directive := BranchCompletion(ctx, cmd, args, tt.toComplete)

			if !equalSlices(results, tt.expectedResults) {
				t.Errorf("expected results %v, got %v", tt.expectedResults, results)
			}
			if directive != tt.expectedDirective {
				t.Errorf("expected directive %v, got %v", tt.expectedDirective, directive)
			}
		})
	}
}

func TestGetBranchNames(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func() *testutils.MockGitExecutor
		expected    []string
		expectError bool
	}{
		{
			name: "successful branch retrieval",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop", nil)
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main\norigin/feature/test", nil)
				return mock
			},
			expected:    []string{"main", "develop", "feature/test"},
			expectError: false,
		},
		{
			name: "local branches only",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop", nil)
				mock.SetResponse("branch -r --format=%(refname:short)", "", errors.New("no remotes"))
				return mock
			},
			expected:    []string{"main", "develop"},
			expectError: false,
		},
		{
			name: "remote branches only",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "", errors.New("no local branches"))
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main\norigin/develop", nil)
				return mock
			},
			expected:    []string{"main", "develop"},
			expectError: false,
		},
		{
			name: "deduplication",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop", nil)
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main\norigin/develop", nil)
				return mock
			},
			expected:    []string{"main", "develop"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear cache before each test
			GlobalCache.Clear()

			ctx := NewCompletionContext(tt.mockFunc())
			result, err := getBranchNames(ctx)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetLocalBranches(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func() *testutils.MockGitExecutor
		expected    []string
		expectError bool
	}{
		{
			name: "successful local branch retrieval",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop\nfeature/test", nil)
				return mock
			},
			expected:    []string{"main", "develop", "feature/test"},
			expectError: false,
		},
		{
			name: "git command error",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "", errors.New("git error"))
				return mock
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "empty output",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "", nil)
				return mock
			},
			expected:    []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCompletionContext(tt.mockFunc())
			result, err := getLocalBranches(ctx)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetRemoteBranches(t *testing.T) {
	tests := []struct {
		name        string
		mockFunc    func() *testutils.MockGitExecutor
		expected    []string
		expectError bool
	}{
		{
			name: "successful remote branch retrieval",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main\norigin/develop\norigin/feature/test", nil)
				return mock
			},
			expected:    []string{"main", "develop", "feature/test"},
			expectError: false,
		},
		{
			name: "with symbolic refs",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main\norigin/develop\norigin/HEAD -> origin/main", nil)
				return mock
			},
			expected:    []string{"main", "develop"},
			expectError: false,
		},
		{
			name: "git command error",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch -r --format=%(refname:short)", "", errors.New("git error"))
				return mock
			},
			expected:    nil,
			expectError: true,
		},
		{
			name: "empty output",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch -r --format=%(refname:short)", "", nil)
				return mock
			},
			expected:    []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := NewCompletionContext(tt.mockFunc())
			result, err := getRemoteBranches(ctx)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestParseBranchList(t *testing.T) {
	tests := []struct {
		name      string
		branchStr string
		expected  []string
	}{
		{
			name:      "empty string",
			branchStr: "",
			expected:  nil,
		},
		{
			name:      "single branch",
			branchStr: "main",
			expected:  []string{"main"},
		},
		{
			name:      "multiple branches",
			branchStr: "main,develop,feature/test",
			expected:  []string{"main", "develop", "feature/test"},
		},
		{
			name:      "with spaces",
			branchStr: "main, develop , feature/test",
			expected:  []string{"main", "develop", "feature/test"},
		},
		{
			name:      "with empty parts",
			branchStr: "main,,develop,,",
			expected:  []string{"main", "develop"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseBranchList(tt.branchStr)

			if !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestGetLastBranchInList(t *testing.T) {
	tests := []struct {
		name      string
		branchStr string
		expected  string
	}{
		{
			name:      "empty string",
			branchStr: "",
			expected:  "",
		},
		{
			name:      "single branch",
			branchStr: "main",
			expected:  "main",
		},
		{
			name:      "multiple branches",
			branchStr: "main,develop,feature/test",
			expected:  "feature/test",
		},
		{
			name:      "with spaces",
			branchStr: "main, develop , feature/test",
			expected:  "feature/test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetLastBranchInList(tt.branchStr)

			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestCompleteBranchList(t *testing.T) {
	tests := []struct {
		name         string
		mockFunc     func() *testutils.MockGitExecutor
		currentInput string
		toComplete   string
		expected     []string
		expectError  bool
	}{
		{
			name: "filter already used branches",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop\nfeature/test", nil)
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main\norigin/develop", nil)
				return mock
			},
			currentInput: "main,develop",
			toComplete:   "",
			expected:     []string{"feature/test"},
			expectError:  false,
		},
		{
			name: "filter with prefix",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop\nfeature/test\nfeature/auth", nil)
				mock.SetResponse("branch -r --format=%(refname:short)", "origin/main", nil)
				return mock
			},
			currentInput: "main",
			toComplete:   "f",
			expected:     []string{"feature/auth", "feature/test"},
			expectError:  false,
		},
		{
			name: "git error",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("branch --format=%(refname:short)", "", errors.New("git error"))
				mock.SetResponse("branch -r --format=%(refname:short)", "", errors.New("git error"))
				return mock
			},
			currentInput: "",
			toComplete:   "",
			expected:     nil,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear cache before each test
			GlobalCache.Clear()

			ctx := NewCompletionContext(tt.mockFunc())
			result, err := CompleteBranchList(ctx, tt.currentInput, tt.toComplete)

			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBranchCompletionWithCache(t *testing.T) {
	// Clear cache before test
	GlobalCache.Clear()

	mock := testutils.NewMockGitExecutor()
	mock.SetResponse("rev-parse --git-dir", "", nil)
	mock.SetResponse("branch --format=%(refname:short)", "main\ndevelop", nil)
	mock.SetResponse("branch -r --format=%(refname:short)", "origin/main", nil)

	ctx := NewCompletionContext(mock)
	cmd := &cobra.Command{}
	args := []string{}

	// First call should fetch from git
	results1, _ := BranchCompletion(ctx, cmd, args, "")

	// Second call should use cache (mock won't be called again)
	results2, _ := BranchCompletion(ctx, cmd, args, "")

	if !equalSlices(results1, results2) {
		t.Errorf("cached results differ: %v vs %v", results1, results2)
	}

	// Verify the results are correct (main should come first due to priority)
	expected := []string{"main", "develop"}
	if !equalSlices(results1, expected) {
		t.Errorf("expected %v, got %v", expected, results1)
	}
}

func TestPrioritizeBranches(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty list",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "main first",
			input:    []string{"feature/test", "main", "develop"},
			expected: []string{"main", "develop", "feature/test"},
		},
		{
			name:     "master priority over develop",
			input:    []string{"feature/test", "master", "develop"},
			expected: []string{"master", "develop", "feature/test"},
		},
		{
			name:     "main priority over master",
			input:    []string{"feature/test", "master", "main", "develop"},
			expected: []string{"main", "master", "develop", "feature/test"},
		},
		{
			name:     "alphabetical for regular branches",
			input:    []string{"feature/zed", "feature/auth", "bugfix/test", "main"},
			expected: []string{"main", "bugfix/test", "feature/auth", "feature/zed"},
		},
		{
			name:     "development branch priority",
			input:    []string{"feature/test", "development", "hotfix/bug"},
			expected: []string{"development", "feature/test", "hotfix/bug"},
		},
		{
			name:     "no priority branches",
			input:    []string{"feature/c", "feature/a", "feature/b"},
			expected: []string{"feature/a", "feature/b", "feature/c"},
		},
		{
			name:     "all priority branches",
			input:    []string{"development", "develop", "master", "main"},
			expected: []string{"main", "master", "develop", "development"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prioritizeBranches(tt.input)

			if !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
