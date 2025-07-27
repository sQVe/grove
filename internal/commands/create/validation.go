package create

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

func validateBranchNameInput(input string) error {
	if isURL(input) || isRemoteBranch(input) {
		return nil
	}

	if strings.TrimSpace(input) == "" {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name cannot be empty", nil)
	}

	if err := validateGitBranchName(input); err != nil {
		return err
	}

	return nil
}

func validateGitBranchName(name string) error {
	if strings.Contains(name, " ") {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name cannot contain spaces", nil)
	}

	if strings.HasPrefix(name, "-") {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name cannot start with a dash", nil)
	}

	if invalidCharsRegex.MatchString(name) {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name contains invalid characters (~^:?*[]\\)", nil)
	}

	if consecutiveDotsRegex.MatchString(name) {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name cannot contain consecutive dots (..)", nil)
	}

	if invalidStartEndRegex.MatchString(name) {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name cannot start or end with dots or slashes", nil)
	}

	if controlCharsRegex.MatchString(name) {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name cannot contain control characters", nil)
	}

	if name == "HEAD" || name == "@" {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name cannot be 'HEAD' or '@'", nil)
	}

	if strings.HasSuffix(name, ".lock") {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name cannot end with '.lock'", nil)
	}

	return nil
}

func isURL(input string) bool {
	return strings.HasPrefix(input, "http://") ||
		strings.HasPrefix(input, "https://") ||
		strings.HasPrefix(input, "git@") ||
		strings.Contains(input, "://")
}

func isRemoteBranch(input string) bool {
	return strings.Contains(input, "/") && !isURL(input)
}

func ValidateCreateArgs(args []string) error {
	if len(args) == 0 {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name, URL, or remote branch is required", nil)
	}
	if len(args) > 2 {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "too many arguments, expected: grove create [branch-name|url] [path]", nil)
	}

	// Catch invalid branch names early to provide clear error messages.
	branchName := args[0]
	if err := validateBranchNameInput(branchName); err != nil {
		return err
	}

	return nil
}

func ValidateFlags(noCopy, copyEnv bool, copyPatterns string) error {
	if noCopy && (copyEnv || copyPatterns != "") {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "--no-copy cannot be used with --copy-env or --copy flags", nil)
	}
	return nil
}
