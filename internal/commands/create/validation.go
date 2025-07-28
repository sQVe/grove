package create

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sqve/grove/internal/errors"
)

const (
	// testFilePrefix is used for permission testing to avoid conflicts with user files.
	testFilePrefix = ".grove_write_test"
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

	// supportedHostsMap provides O(1) lookup for supported Git platforms.
	supportedHostsMap = map[string]bool{
		"github.com":    true,
		"gitlab.com":    true,
		"bitbucket.org": true,
		"dev.azure.com": true,
		"codeberg.org":  true,
		"gitea.io":      true,
	}

	// supportedHosts provides the list for error messages.
	supportedHosts = []string{
		"github.com", "gitlab.com", "bitbucket.org",
		"dev.azure.com", "codeberg.org", "gitea.io",
	}
)

func validateBranchNameInput(input string) error {
	if isURL(input) || isRemoteBranch(input) {
		return nil
	}

	if strings.TrimSpace(input) == "" {
		return errors.ErrInvalidBranchName(input, "branch name cannot be empty")
	}

	if err := validateGitBranchName(input); err != nil {
		return err
	}

	return nil
}

func validateGitBranchName(name string) error {
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

func ValidateRepositoryState(repoPath string) error {
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); os.IsNotExist(err) {
		return errors.ErrRepoNotFound(repoPath).
			WithContext("suggestion", "Run 'grove init' to initialize a Grove repository")
	}
	return nil
}

func ValidateURL(input string) error {
	if !isURL(input) {
		return nil // Not a URL, validation handled elsewhere
	}

	parsed, err := url.Parse(input)
	if err != nil {
		return errors.ErrURLParsing(input, err)
	}

	if parsed.Host == "" {
		return errors.ErrInvalidURL(input, "missing host")
	}

	// Check if the host is supported using O(1) map lookup.
	hostSupported := false
	for host := range supportedHostsMap {
		if strings.Contains(parsed.Host, host) {
			hostSupported = true
			break
		}
	}

	if !hostSupported {
		return errors.ErrUnsupportedURL(input).
			WithContext("supported_platforms", strings.Join(supportedHosts, ", "))
	}

	return nil
}

func ValidatePath(path string) error {
	if path == "" {
		return nil // Empty path is valid, will use default
	}

	cleanPath := filepath.Clean(path)
	if strings.Contains(cleanPath, "..") {
		return errors.ErrPathTraversal(path)
	}

	// Prevent accidental overwrites of existing directories.
	if _, err := os.Stat(cleanPath); err == nil {
		return errors.ErrPathExists(cleanPath).
			WithContext("suggestion", "Use --force to overwrite or choose a different path")
	}

	// Ensure we can create the worktree directory by testing parent permissions.
	parentDir := filepath.Dir(cleanPath)
	if parentDir != "." {
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			// Parent doesn't exist, that's okay, we'll create it
			return nil
		}
		// Test directory creation permissions with a unique test file.
		testFileName := fmt.Sprintf("%s_%d", testFilePrefix, time.Now().UnixNano())
		testFile := filepath.Join(parentDir, testFileName)
		
		// Use deferred cleanup to ensure test file is always removed.
		var cleanup func()
		cleanup = func() {
			if err := os.Remove(testFile); err != nil {
				// Log but don't fail validation due to cleanup issues.
				// This prevents the race condition from causing validation failures.
			}
		}
		
		if f, err := os.Create(testFile); err != nil {
			return errors.ErrDirectoryAccess(parentDir, err)
		} else {
			f.Close()
			defer cleanup()
		}
	}

	return nil
}

func ValidateFilePatterns(patterns []string) error {
	for _, pattern := range patterns {
		if strings.TrimSpace(pattern) == "" {
			return errors.ErrInvalidPattern(pattern, "pattern cannot be empty")
		}

		// Path traversal attempts pose security risks.
		if strings.Contains(pattern, "..") {
			return errors.ErrInvalidPattern(pattern, "path traversal not allowed")
		}

		// Malformed patterns will cause runtime errors.
		if _, err := filepath.Match(pattern, "test"); err != nil {
			return errors.ErrInvalidPattern(pattern, fmt.Sprintf("invalid glob pattern: %v", err))
		}
	}
	return nil
}

func ValidateRemoteBranch(remoteBranch string) error {
	if !isRemoteBranch(remoteBranch) {
		return nil // Not a remote branch reference
	}

	parts := strings.SplitN(remoteBranch, "/", 2)
	if len(parts) != 2 {
		return errors.ErrInvalidBranchName(remoteBranch, "invalid remote branch format, expected: remote/branch")
	}

	remoteName := parts[0]
	branchName := parts[1]

	if strings.TrimSpace(remoteName) == "" {
		return errors.ErrRemoteNotFound(remoteName)
	}
	if err := validateGitBranchName(branchName); err != nil {
		return err
	}

	return nil
}

func EnhanceErrorWithContext(err error, input string) error {
	if err == nil {
		return nil
	}

	groveErr, ok := err.(*errors.GroveError)
	if !ok {
		return err
	}

	switch groveErr.Code {
	case errors.ErrCodeBranchNotFound:
		return groveErr.
			WithContext("suggestion", "Check available branches with 'git branch -a' or use --create to create new branch").
			WithContext("examples", []string{
				"grove create --create " + input,
				"grove create --base main " + input,
			})

	case errors.ErrCodeInvalidBranchName:
		return groveErr.
			WithContext("suggestion", "Use alphanumeric characters, hyphens, and slashes only").
			WithContext("examples", []string{
				"grove create feature/user-auth",
				"grove create hotfix-123",
				"grove create feature_branch",
			})

	case errors.ErrCodeUnsupportedURL:
		return groveErr.
			WithContext("suggestion", "Use URLs from supported Git platforms").
			WithContext("examples", []string{
				"grove create https://github.com/owner/repo/pull/123",
				"grove create https://gitlab.com/owner/repo/-/merge_requests/456",
				"grove create https://bitbucket.org/owner/repo/pull-requests/789",
			})

	case errors.ErrCodeURLParsing:
		return groveErr.
			WithContext("suggestion", "Check URL format and ensure it's a valid Git platform URL").
			WithContext("examples", []string{
				"https://github.com/owner/repo/tree/branch-name",
				"https://gitlab.com/owner/repo/-/tree/branch-name",
			})

	case errors.ErrCodePathExists:
		return groveErr.
			WithContext("suggestion", "Use --force to overwrite or choose a different path").
			WithContext("examples", []string{
				"grove create " + input + " --force",
				"grove create " + input + " ./alternative-path",
			})

	case errors.ErrCodeRemoteNotFound:
		return groveErr.
			WithContext("suggestion", "Check available remotes with 'git remote -v'").
			WithContext("examples", []string{
				"git remote add upstream https://github.com/original/repo.git",
				"grove create origin/" + strings.Split(input, "/")[1],
			})

	case errors.ErrCodeInvalidPattern:
		return groveErr.
			WithContext("suggestion", "Use valid glob patterns for file copying").
			WithContext("examples", []string{
				"grove create branch --copy '.env*'",
				"grove create branch --copy '.vscode/,*.json'",
				"grove create branch --copy-env",
			})

	case errors.ErrCodeRepoNotFound:
		return groveErr.
			WithContext("suggestion", "Initialize Grove repository first").
			WithContext("examples", []string{
				"grove init",
				"cd /path/to/git/repo && grove init",
			})

	case errors.ErrCodePathTraversal:
		return groveErr.
			WithContext("suggestion", "Use absolute paths or paths without '..' components").
			WithContext("examples", []string{
				"grove create branch ./worktree-path",
				"grove create branch /absolute/path/to/worktree",
			})
	}

	return groveErr
}
