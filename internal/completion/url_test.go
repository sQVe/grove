//go:build !integration
// +build !integration

package completion

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/testutils"
	"github.com/stretchr/testify/assert"
)

func TestURLCompletion(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		wantLength int
		wantError  bool
	}{
		{
			name:       "empty input",
			toComplete: "",
			wantLength: 0,
			wantError:  false,
		},
		{
			name:       "github prefix",
			toComplete: "github",
			wantLength: 0, // getURLSuggestions returns empty for simple text
			wantError:  false,
		},
		{
			name:       "https prefix",
			toComplete: "https://",
			wantLength: 0,
			wantError:  false,
		},
		{
			name:       "git protocol",
			toComplete: "git@",
			wantLength: 0,
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			ctx := NewCompletionContext(mockExecutor)
			cmd := &cobra.Command{}

			results, directive := URLCompletion(ctx, cmd, []string{}, tt.toComplete)

			if tt.wantError {
				assert.Equal(t, cobra.ShellCompDirectiveError, directive)
			} else {
				assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
				assert.Len(t, results, tt.wantLength)
			}
		})
	}
}

func TestURLAndDirectoryCompletion(t *testing.T) {
	tests := []struct {
		name       string
		toComplete string
		wantDir    cobra.ShellCompDirective
	}{
		{
			name:       "URL-like input",
			toComplete: "https://github.com",
			wantDir:    cobra.ShellCompDirectiveNoFileComp,
		},
		{
			name:       "directory-like input",
			toComplete: "./local",
			wantDir:    cobra.ShellCompDirectiveDefault, // allows file completion
		},
		{
			name:       "simple path",
			toComplete: "my-repo",
			wantDir:    cobra.ShellCompDirectiveDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExecutor := testutils.NewMockGitExecutor()
			ctx := NewCompletionContext(mockExecutor)
			cmd := &cobra.Command{}

			_, directive := URLAndDirectoryCompletion(ctx, cmd, []string{}, tt.toComplete)
			assert.Equal(t, tt.wantDir, directive)
		})
	}
}

func TestGetURLSuggestions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "github input",
			input:    "github",
			expected: []string{},
		},
		{
			name:     "https input",
			input:    "https://",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, _ := getURLSuggestions(tt.input)
			assert.Equal(t, tt.expected, results)
		})
	}
}

func TestGetPlatformURLSuggestions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []URLSuggestion
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []URLSuggestion{},
		},
		{
			name:     "github input",
			input:    "github",
			expected: []URLSuggestion{},		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := getPlatformURLSuggestions(tt.input)
			assert.Equal(t, tt.expected, results)
		})
	}
}

func TestGetURLSuggestionsWithDescriptions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []URLSuggestion
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []URLSuggestion{},
		},
		{
			name:     "simple text",
			input:    "test",
			expected: []URLSuggestion{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := GetURLSuggestionsWithDescriptions(tt.input)
			assert.Equal(t, tt.expected, results)
		})
	}
}

func TestLooksLikeURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "https URL",
			input:    "https://github.com",
			expected: true,
		},
		{
			name:     "http URL",
			input:    "http://example.com",
			expected: true,
		},
		{
			name:     "git SSH",
			input:    "git@github.com",
			expected: true,
		},
		{
			name:     "simple path",
			input:    "./local-repo",
			expected: false,
		},
		{
			name:     "simple name",
			input:    "my-project",
			expected: false,
		},
		{
			name:     "git protocol",
			input:    "git://example.com",
			expected: true,
		},
		{
			name:     "ssh protocol",
			input:    "ssh://git@example.com",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := looksLikeURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCompleteGitURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []string{},
		},
		{
			name:     "github partial",
			input:    "github.com",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := CompleteGitURL(tt.input)
			assert.Equal(t, tt.expected, results)
		})
	}
}

func TestCompleteGitURLWithDescriptions(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []URLSuggestion
	}{
		{
			name:     "empty input",
			input:    "",
			expected: []URLSuggestion{},
		},
		{
			name:     "partial URL",
			input:    "github.com",
			expected: []URLSuggestion{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := CompleteGitURLWithDescriptions(tt.input)
			assert.Equal(t, tt.expected, results)
		})
	}
}

func TestValidateURLCompletion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "valid https URL",
			input:    "https://github.com/user/repo.git",
			expected: true,
		},
		{
			name:     "valid ssh URL",
			input:    "git@github.com:user/repo.git",
			expected: true,
		},
		{
			name:     "invalid URL",
			input:    "not-a-url",
			expected: false,
		},
		{
			name:     "empty input",
			input:    "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateURLCompletion(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPlatformFromURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "GitHub URL",
			input:    "https://github.com/user/repo",
			expected: "github",
		},
		{
			name:     "GitLab URL",
			input:    "https://gitlab.com/user/repo",
			expected: "gitlab",
		},
		{
			name:     "unknown platform",
			input:    "https://example.com/repo",
			expected: "unknown",
		},
		{
			name:     "invalid URL",
			input:    "not-a-url",
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetPlatformFromURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSuggestBranchesForURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "GitHub URL",
			input:    "https://github.com/user/repo",
			expected: []string{"main", "master", "develop", "development"},
		},
		{
			name:     "GitLab URL",
			input:    "https://gitlab.com/user/repo",
			expected: []string{"main", "master", "develop", "development"},
		},
		{
			name:     "unknown platform",
			input:    "https://example.com/repo",
			expected: []string{"main", "master", "develop", "development"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results := SuggestBranchesForURL(tt.input)
			assert.Equal(t, tt.expected, results)
		})
	}
}
