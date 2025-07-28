package validation

import (
	"strings"

	"github.com/sqve/grove/internal/errors"
)

type ErrorEnhancement struct {
	Suggestion string
	Examples   []string
}

var errorEnhancements = map[string]ErrorEnhancement{
	errors.ErrCodeBranchNotFound: {
		Suggestion: "Check available branches with 'git branch -a' or use --create to create new branch",
		Examples: []string{
			"grove create --create %s",
			"grove create --base main %s",
		},
	},
	errors.ErrCodeInvalidBranchName: {
		Suggestion: "Use alphanumeric characters, hyphens, and slashes only",
		Examples: []string{
			"grove create feature/user-auth",
			"grove create hotfix-123",
			"grove create feature_branch",
		},
	},
	errors.ErrCodeUnsupportedURL: {
		Suggestion: "Use URLs from supported Git platforms",
		Examples: []string{
			"grove create https://github.com/owner/repo/pull/123",
			"grove create https://gitlab.com/owner/repo/-/merge_requests/456",
			"grove create https://bitbucket.org/owner/repo/pull-requests/789",
		},
	},
	errors.ErrCodeURLParsing: {
		Suggestion: "Check URL format and ensure it's a valid Git platform URL",
		Examples: []string{
			"https://github.com/owner/repo/tree/branch-name",
			"https://gitlab.com/owner/repo/-/tree/branch-name",
		},
	},
	errors.ErrCodePathExists: {
		Suggestion: "Use --force to overwrite or choose a different path",
		Examples: []string{
			"grove create %s --force",
			"grove create %s ./alternative-path",
		},
	},
	errors.ErrCodeRemoteNotFound: {
		Suggestion: "Check available remotes with 'git remote -v'",
		Examples: []string{
			"git remote add upstream https://github.com/original/repo.git",
			"grove create origin/%s",
		},
	},
	errors.ErrCodeInvalidPattern: {
		Suggestion: "Use valid glob patterns for file copying",
		Examples: []string{
			"grove create branch --copy '.env*'",
			"grove create branch --copy '.vscode/,*.json'",
			"grove create branch --copy-env",
		},
	},
	errors.ErrCodeRepoNotFound: {
		Suggestion: "Initialize Grove repository first",
		Examples: []string{
			"grove init",
			"cd /path/to/git/repo && grove init",
		},
	},
	errors.ErrCodePathTraversal: {
		Suggestion: "Use absolute paths or paths without '..' components",
		Examples: []string{
			"grove create branch ./worktree-path",
			"grove create branch /absolute/path/to/worktree",
		},
	},
}

func EnhanceErrorWithContext(err error, input string) error {
	if err == nil {
		return nil
	}

	groveErr, ok := err.(*errors.GroveError)
	if !ok {
		return err
	}

	enhancement, exists := errorEnhancements[groveErr.Code]
	if !exists {
		return groveErr
	}

	enhancedErr := groveErr.WithContext("suggestion", enhancement.Suggestion)

	// Apply input-specific formatting to examples.
	examples := make([]string, len(enhancement.Examples))
	for i, example := range enhancement.Examples {
		// Handle special cases for remote branch examples.
		if groveErr.Code == errors.ErrCodeRemoteNotFound && strings.Contains(input, "/") {
			branchPart := strings.Split(input, "/")[1]
			examples[i] = strings.ReplaceAll(example, "%s", branchPart)
		} else {
			examples[i] = strings.ReplaceAll(example, "%s", input)
		}
	}

	return enhancedErr.WithContext("examples", examples)
}
