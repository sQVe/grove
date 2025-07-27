//go:build !integration
// +build !integration

package completion

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/testutils"
)

func TestCompletionContext_WithTimeout(t *testing.T) {
	tests := []struct {
		name        string
		timeout     time.Duration
		fn          func() ([]string, error)
		expected    []string
		expectError bool
	}{
		{
			name:    "successful completion",
			timeout: 100 * time.Millisecond,
			fn: func() ([]string, error) {
				return []string{"main", "develop"}, nil
			},
			expected:    []string{"main", "develop"},
			expectError: false,
		},
		{
			name:    "completion with error",
			timeout: 100 * time.Millisecond,
			fn: func() ([]string, error) {
				return nil, errors.New("test error")
			},
			expected:    nil,
			expectError: true,
		},
		{
			name:    "timeout",
			timeout: 10 * time.Millisecond,
			fn: func() ([]string, error) {
				time.Sleep(50 * time.Millisecond)
				return []string{"main"}, nil
			},
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &CompletionContext{
				Executor: testutils.NewMockGitExecutor(),
				Timeout:  tt.timeout,
			}

			result, err := ctx.WithTimeout(tt.fn)

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

func TestCompletionContext_IsInGroveRepo(t *testing.T) {
	tests := []struct {
		name     string
		mockFunc func() *testutils.MockGitExecutor
		expected bool
	}{
		{
			name: "in git repository",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("rev-parse --git-dir", "", nil)
				return mock
			},
			expected: true,
		},
		{
			name: "not in git repository",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("rev-parse --git-dir", "", errors.New("exit 128"))
				return mock
			},
			expected: false,
		},
		{
			name: "git command error",
			mockFunc: func() *testutils.MockGitExecutor {
				mock := testutils.NewMockGitExecutor()
				mock.SetResponse("rev-parse --git-dir", "", errors.New("command not found"))
				return mock
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalCache.Clear()

			ctx := &CompletionContext{
				Executor: tt.mockFunc(),
				Timeout:  CompletionTimeout,
			}

			result := ctx.IsInGroveRepo()

			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestFilterCompletions(t *testing.T) {
	tests := []struct {
		name        string
		completions []string
		toComplete  string
		expected    []string
	}{
		{
			name:        "empty toComplete",
			completions: []string{"main", "develop", "feature/test"},
			toComplete:  "",
			expected:    []string{"main", "develop", "feature/test"},
		},
		{
			name:        "filter by prefix",
			completions: []string{"main", "develop", "feature/test"},
			toComplete:  "f",
			expected:    []string{"feature/test"},
		},
		{
			name:        "no matches",
			completions: []string{"main", "develop", "feature/test"},
			toComplete:  "x",
			expected:    []string{},
		},
		{
			name:        "empty completions",
			completions: []string{},
			toComplete:  "main",
			expected:    []string{},
		},
		{
			name:        "partial match",
			completions: []string{"main", "master", "maintenance"},
			toComplete:  "mai",
			expected:    []string{"main", "maintenance"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FilterCompletions(tt.completions, tt.toComplete)

			if !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCreateCompletionCommands(t *testing.T) {
	rootCmd := &cobra.Command{
		Use: "test",
	}

	CreateCompletionCommands(rootCmd)

	completionCmd := findCommand(rootCmd, "completion")
	if completionCmd == nil {
		t.Error("completion command not found")
		return
	}

	expectedArgs := []string{"bash", "zsh", "fish", "powershell"}
	if !equalSlices(completionCmd.ValidArgs, expectedArgs) {
		t.Errorf("expected valid args %v, got %v", expectedArgs, completionCmd.ValidArgs)
	}
}

func TestSafeExecuteWithFallback(t *testing.T) {
	tests := []struct {
		name              string
		fn                func() ([]string, cobra.ShellCompDirective)
		fallback          []string
		expectedResults   []string
		expectedDirective cobra.ShellCompDirective
	}{
		{
			name: "successful execution",
			fn: func() ([]string, cobra.ShellCompDirective) {
				return []string{"main", "develop"}, cobra.ShellCompDirectiveNoFileComp
			},
			fallback:          []string{"fallback"},
			expectedResults:   []string{"main", "develop"},
			expectedDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "empty result uses fallback",
			fn: func() ([]string, cobra.ShellCompDirective) {
				return []string{}, cobra.ShellCompDirectiveNoFileComp
			},
			fallback:          []string{"fallback"},
			expectedResults:   []string{"fallback"},
			expectedDirective: cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name: "panic recovery",
			fn: func() ([]string, cobra.ShellCompDirective) {
				panic("test panic")
			},
			fallback:          []string{"fallback"},
			expectedResults:   []string{"fallback"},
			expectedDirective: cobra.ShellCompDirectiveError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, directive := SafeExecuteWithFallback(tt.fn, tt.fallback)

			if !equalSlices(results, tt.expectedResults) {
				t.Errorf("expected results %v, got %v", tt.expectedResults, results)
			}
			if directive != tt.expectedDirective {
				t.Errorf("expected directive %v, got %v", tt.expectedDirective, directive)
			}
		})
	}
}

func TestNewCompletionContext(t *testing.T) {
	executor := testutils.NewMockGitExecutor()
	ctx := NewCompletionContext(executor)

	if ctx.Executor != executor {
		t.Error("executor not set correctly")
	}
	if ctx.Timeout != CompletionTimeout {
		t.Errorf("expected timeout %v, got %v", CompletionTimeout, ctx.Timeout)
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func findCommand(parent *cobra.Command, name string) *cobra.Command {
	for _, cmd := range parent.Commands() {
		if cmd.Name() == name {
			return cmd
		}
	}
	return nil
}

func TestCompletionContextWithTimeout_Cancel(t *testing.T) {
	ctx := &CompletionContext{
		Executor: testutils.NewMockGitExecutor(),
		Timeout:  100 * time.Millisecond,
	}

	start := time.Now()
	_, err := ctx.WithTimeout(func() ([]string, error) {
		time.Sleep(200 * time.Millisecond)
		return []string{"should not reach here"}, nil
	})

	duration := time.Since(start)

	if err == nil {
		t.Error("expected timeout error")
	}
	if duration > 150*time.Millisecond {
		t.Errorf("timeout took too long: %v", duration)
	}
}

func TestCompletionContextWithTimeout_Success(t *testing.T) {
	ctx := &CompletionContext{
		Executor: testutils.NewMockGitExecutor(),
		Timeout:  100 * time.Millisecond,
	}

	expected := []string{"main", "develop"}
	result, err := ctx.WithTimeout(func() ([]string, error) {
		return expected, nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !equalSlices(result, expected) {
		t.Errorf("expected %v, got %v", expected, result)
	}
}

func TestCompletionContext_NetworkAwareness(t *testing.T) {
	GlobalCache.Clear()

	ctx := &CompletionContext{
		Executor: testutils.NewMockGitExecutor(),
		Timeout:  100 * time.Millisecond,
	}

	// Network detection is unreliable in unit tests, so we test the caching mechanism instead.

	isOnline1 := ctx.IsOnline()

	isOnline2 := ctx.IsOnline()

	if isOnline1 != isOnline2 {
		t.Errorf("network state should be consistent, got %v then %v", isOnline1, isOnline2)
	}

	allowedOp := ctx.IsNetworkOperationAllowed()
	if allowedOp != isOnline1 {
		t.Errorf("network operation allowance should match online state, got allowance=%v, online=%v", allowedOp, isOnline1)
	}
}

func TestBranchListCompletion(t *testing.T) {
	GlobalCache.Clear()

	tests := []struct {
		name         string
		toComplete   string
		mockBranches []string
		expected     []string
		inRepo       bool
	}{
		{
			name:         "single branch completion",
			toComplete:   "mai",
			mockBranches: []string{"main", "master", "develop"},
			expected:     []string{"main"},
			inRepo:       true,
		},
		{
			name:         "comma-separated completion",
			toComplete:   "main,dev",
			mockBranches: []string{"main", "develop", "feature/test"},
			expected:     []string{"main,develop"},
			inRepo:       true,
		},
		{
			name:         "comma-separated with spaces",
			toComplete:   "main, fea",
			mockBranches: []string{"main", "develop", "feature/test"},
			expected:     []string{"main,feature/test"},
			inRepo:       true,
		},
		{
			name:         "not in repository",
			toComplete:   "main",
			mockBranches: []string{"main", "develop"},
			expected:     []string{},
			inRepo:       false,
		},
		{
			name:         "empty completion",
			toComplete:   "",
			mockBranches: []string{"main", "develop"},
			expected:     []string{"main", "develop"},
			inRepo:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			GlobalCache.Clear()

			mock := testutils.NewMockGitExecutor()

			if tt.inRepo {
				mock.SetResponse("rev-parse --git-dir", "", nil)
				mock.SetResponse("branch --format=%(refname:short)", strings.Join(tt.mockBranches, "\n"), nil)
				mock.SetResponse("branch -r --format=%(refname:short)", "", nil) // No remote branches for simplicity.
			} else {
				mock.SetResponse("rev-parse --git-dir", "", errors.New("exit 128"))
			}

			ctx := &CompletionContext{
				Executor: mock,
				Timeout:  100 * time.Millisecond,
			}

			result, directive := BranchListCompletion(ctx, nil, []string{}, tt.toComplete)

			expectedDirective := cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
			if directive != expectedDirective {
				t.Errorf("expected directive %v, got %v", expectedDirective, directive)
			}

			if !equalSlices(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
