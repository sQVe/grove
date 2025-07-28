package validation

import (
	"regexp"
	"strings"

	"github.com/sqve/grove/internal/errors"
)

var (
	// Git rejects branches with special characters that conflict with ref syntax.
	invalidCharsRegex = regexp.MustCompile(`[~^:?*\[\]\\]`)
	// Consecutive dots create ambiguous ref resolution in Git.
	consecutiveDotsRegex = regexp.MustCompile(`\.\.`)
	// Leading/trailing dots and slashes break Git's hierarchical ref structure.
	invalidStartEndRegex = regexp.MustCompile(`^[./]|[./]$`)
	// Control characters would break terminal output and Git commands.
	controlCharsRegex = regexp.MustCompile(`[\x00-\x1f\x7f]`)
)

func ValidateGitBranchName(name string) error {
	if strings.Contains(name, " ") {
		return errors.ErrInvalidBranchName(name, "cannot contain spaces")
	}

	if strings.HasPrefix(name, "-") {
		return errors.ErrInvalidBranchName(name, "cannot start with a dash")
	}

	if invalidCharsRegex.MatchString(name) {
		return errors.ErrInvalidBranchName(name, "contains invalid characters (~^:?*[]\\)")
	}

	if consecutiveDotsRegex.MatchString(name) {
		return errors.ErrInvalidBranchName(name, "cannot contain consecutive dots (..)")
	}

	if invalidStartEndRegex.MatchString(name) {
		return errors.ErrInvalidBranchName(name, "cannot start or end with dots or slashes")
	}

	if controlCharsRegex.MatchString(name) {
		return errors.ErrInvalidBranchName(name, "cannot contain control characters")
	}

	if name == "HEAD" || name == "@" {
		return errors.ErrInvalidBranchName(name, "cannot be 'HEAD' or '@'")
	}

	if strings.HasSuffix(name, ".lock") {
		return errors.ErrInvalidBranchName(name, "cannot end with '.lock'")
	}

	return nil
}

func IsURL(input string) bool {
	return strings.HasPrefix(input, "http://") ||
		strings.HasPrefix(input, "https://") ||
		strings.HasPrefix(input, "git@") ||
		strings.Contains(input, "://")
}

// IsRemoteBranch checks if the input appears to be a remote branch reference.
func IsRemoteBranch(input string) bool {
	return strings.Contains(input, "/") && !IsURL(input)
}
