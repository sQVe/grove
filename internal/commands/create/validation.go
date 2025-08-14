package create

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/sqve/grove/internal/errors"
	"github.com/sqve/grove/internal/validation"
)

func validateBranchNameInput(input string) error {
	if validation.IsURL(input) || validation.IsRemoteBranch(input) {
		return nil
	}

	if strings.TrimSpace(input) == "" {
		return errors.ErrInvalidBranchName(input, "branch name cannot be empty")
	}

	return validation.ValidateGitBranchName(input)
}

func ValidateCreateArgs(args []string) error {
	if len(args) == 0 {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "branch name, URL, or remote branch is required", nil)
	}
	if len(args) > 2 {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "too many arguments, expected: grove create [branch-name|url] [path]", nil)
	}

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
	if copyEnv && copyPatterns != "" {
		return errors.NewGroveError(errors.ErrCodeConfigInvalid, "cannot use both --copy-env and --copy", nil)
	}
	return nil
}

func ValidateRepositoryState(repoPath string) error {
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return errors.ErrRepoNotFound(repoPath).
			WithContext("suggestion", "Run 'grove init' to initialize a Grove repository")
	}
	return nil
}

func ValidateURL(input string) error {
	return validation.ValidateURL(input)
}

func ValidatePath(path string) error {
	return validation.ValidatePath(path)
}

func ValidateFilePatterns(patterns []string) error {
	return validation.ValidateFilePatterns(patterns)
}

func ValidateRemoteBranch(remoteBranch string) error {
	if !validation.IsRemoteBranch(remoteBranch) {
		return nil // Not a remote branch reference.
	}

	// Optimize string operations by trimming once.
	trimmedBranch := strings.TrimSpace(remoteBranch)
	parts := strings.SplitN(trimmedBranch, "/", 2)
	if len(parts) != 2 {
		return errors.ErrInvalidBranchName(remoteBranch, "invalid remote branch format, expected: remote/branch")
	}

	remoteName := strings.TrimSpace(parts[0])
	branchName := strings.TrimSpace(parts[1])

	if remoteName == "" {
		return errors.ErrRemoteNotFound(remoteName)
	}

	return validation.ValidateGitBranchName(branchName)
}

func EnhanceErrorWithContext(err error, input string) error {
	return validation.EnhanceErrorWithContext(err, input)
}
