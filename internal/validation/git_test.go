package validation

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sqve/grove/internal/errors"
)

func TestValidateGitBranchName_Valid(t *testing.T) {
	validNames := []string{
		"main",
		"develop",
		"feature/auth",
		"feature/user-management",
		"bugfix/login-issue",
		"release/v1.2.3",
		"hotfix/critical-bug",
		"feat/new-feature",
		"fix/bug-123",
		"chore/update-deps",
		"docs/readme-update",
		"test/add-unit-tests",
		"refactor/cleanup-code",
		"improvement/performance",
		"epic/big-feature",
		"task/small-change",
		"branch_with_underscores",
		"branch-with-dashes",
		"BranchWithUpperCase",
		"123-numeric-start",
		"feature/sub/nested/deep",
	}

	for _, name := range validNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateGitBranchName(name)
			assert.NoError(t, err, "Branch name '%s' should be valid", name)
		})
	}
}

func TestValidateGitBranchName_Spaces(t *testing.T) {
	invalidNames := []string{
		"branch with spaces",
		"feature/ auth",
		"main ",
		" main",
		"feature /auth",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateGitBranchName(name)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.True(t, errors.As(err, &groveErr))
			assert.Equal(t, errors.ErrCodeInvalidBranchName, groveErr.Code)
			assert.Contains(t, err.Error(), "cannot contain spaces")
		})
	}
}

func TestValidateGitBranchName_StartWithDash(t *testing.T) {
	invalidNames := []string{
		"-main",
		"-feature/auth",
		"-",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateGitBranchName(name)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.True(t, errors.As(err, &groveErr))
			assert.Equal(t, errors.ErrCodeInvalidBranchName, groveErr.Code)
			assert.Contains(t, err.Error(), "cannot start with a dash")
		})
	}
}

func TestValidateGitBranchName_InvalidCharacters(t *testing.T) {
	invalidChars := []string{"~", "^", ":", "?", "*", "[", "]", "\\"}

	for _, char := range invalidChars {
		name := "feature" + char + "branch"
		t.Run(name, func(t *testing.T) {
			err := ValidateGitBranchName(name)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.True(t, errors.As(err, &groveErr))
			assert.Equal(t, errors.ErrCodeInvalidBranchName, groveErr.Code)
			assert.Contains(t, err.Error(), "contains invalid characters")
		})
	}
}

func TestValidateGitBranchName_ConsecutiveDots(t *testing.T) {
	invalidNames := []string{
		"feature..branch",
		"main..develop",
		"..start",
		"end..",
		"middle..dot..sequence",
		"feature...branch", // Three dots
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateGitBranchName(name)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.True(t, errors.As(err, &groveErr))
			assert.Equal(t, errors.ErrCodeInvalidBranchName, groveErr.Code)
			assert.Contains(t, err.Error(), "cannot contain consecutive dots")
		})
	}
}

func TestValidateGitBranchName_StartEndWithDotsSlashes(t *testing.T) {
	invalidNames := []string{
		".main",
		"main.",
		"/feature",
		"feature/",
		".feature/branch",
		"feature/branch.",
		"/feature/branch/",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateGitBranchName(name)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.True(t, errors.As(err, &groveErr))
			assert.Equal(t, errors.ErrCodeInvalidBranchName, groveErr.Code)
			assert.Contains(t, err.Error(), "cannot start or end with dots or slashes")
		})
	}
}

func TestValidateGitBranchName_ControlCharacters(t *testing.T) {
	// Test various control characters
	controlChars := []byte{
		0x00, // NULL
		0x01, // Start of Heading
		0x08, // Backspace
		0x0A, // Line Feed
		0x0D, // Carriage Return
		0x1F, // Unit Separator
		0x7F, // DEL
	}

	for _, char := range controlChars {
		name := "feature" + string(char) + "branch"
		t.Run(string(char), func(t *testing.T) {
			err := ValidateGitBranchName(name)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.True(t, errors.As(err, &groveErr))
			assert.Equal(t, errors.ErrCodeInvalidBranchName, groveErr.Code)
			assert.Contains(t, err.Error(), "cannot contain control characters")
		})
	}
}

func TestValidateGitBranchName_ReservedNames(t *testing.T) {
	reservedNames := []string{
		"HEAD",
		"@",
	}

	for _, name := range reservedNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateGitBranchName(name)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.True(t, errors.As(err, &groveErr))
			assert.Equal(t, errors.ErrCodeInvalidBranchName, groveErr.Code)
			assert.Contains(t, err.Error(), "cannot be 'HEAD' or '@'")
		})
	}
}

func TestValidateGitBranchName_LockSuffix(t *testing.T) {
	invalidNames := []string{
		"main.lock",
		"feature/auth.lock",
		"develop.lock",
		"some-branch.lock",
	}

	for _, name := range invalidNames {
		t.Run(name, func(t *testing.T) {
			err := ValidateGitBranchName(name)
			require.Error(t, err)

			var groveErr *errors.GroveError
			require.True(t, errors.As(err, &groveErr))
			assert.Equal(t, errors.ErrCodeInvalidBranchName, groveErr.Code)
			assert.Contains(t, err.Error(), "cannot end with '.lock'")
		})
	}
}

func TestIsURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"http URL", "http://example.com/repo.git", true},
		{"https URL", "https://github.com/user/repo.git", true},
		{"ssh git URL", "git@github.com:user/repo.git", true},
		{"not a URL", "main", false},
		{"empty string", "", false},
		{"local path", "/path/to/repo", false},
		{"relative path", "./repo", false},
		{"ftp URL", "ftp://example.com/repo", true}, // Current implementation accepts any :// protocol
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
