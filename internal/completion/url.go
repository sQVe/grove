package completion

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/utils"
)

// URLSuggestion represents a URL completion suggestion with description.
type URLSuggestion struct {
	URL         string
	Description string
}

// PlatformInfo holds information about Git hosting platforms.
type PlatformInfo struct {
	HTTPSPrefix string
	SSHPrefix   string
	Description string
}

// URLCompletion provides completion for Git URLs.
func URLCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("url_completion")

	// Get URL suggestions based on current input
	urls, err := ctx.WithTimeout(func() ([]string, error) {
		return getURLSuggestions(toComplete)
	})
	if err != nil {
		log.Debug("failed to get URL suggestions", "error", err)
		return nil, cobra.ShellCompDirectiveError
	}

	log.Debug("URL completion results", "suggestions", len(urls), "input", toComplete)
	return urls, cobra.ShellCompDirectiveNoFileComp
}

// URLAndDirectoryCompletion provides completion for URLs and directories.
func URLAndDirectoryCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("url_directory_completion")

	// If input looks like a URL, provide URL completion
	if looksLikeURL(toComplete) {
		return URLCompletion(ctx, cmd, args, toComplete)
	}

	// Otherwise, provide directory completion
	log.Debug("providing directory completion", "input", toComplete)
	return nil, cobra.ShellCompDirectiveDefault
}

// getURLSuggestions generates URL completion suggestions.
func getURLSuggestions(toComplete string) ([]string, error) {
	suggestions := make([]string, 0)

	// Get platform suggestions with descriptions
	platformSuggestions := getPlatformURLSuggestions(toComplete)

	// Extract URLs for backward compatibility
	for _, suggestion := range platformSuggestions {
		suggestions = append(suggestions, suggestion.URL)
	}

	return suggestions, nil
}

// getPlatformURLSuggestions provides HTTPS URL suggestions for various platforms.
func getPlatformURLSuggestions(toComplete string) []URLSuggestion {
	suggestions := make([]URLSuggestion, 0)

	// Only provide suggestions for URL-like inputs, not plain text
	if !looksLikeURL(toComplete) && !strings.Contains(toComplete, ".") {
		return suggestions
	}

	platforms := map[string]PlatformInfo{
		"github": {
			HTTPSPrefix: "https://github.com/",
			SSHPrefix:   "git@github.com:",
			Description: "GitHub repository",
		},
		"gitlab": {
			HTTPSPrefix: "https://gitlab.com/",
			SSHPrefix:   "git@gitlab.com:",
			Description: "GitLab repository",
		},
		"bitbucket": {
			HTTPSPrefix: "https://bitbucket.org/",
			SSHPrefix:   "git@bitbucket.org:",
			Description: "Bitbucket repository",
		},
	}

	// Provide HTTPS suggestions for partial URL inputs
	for name, platform := range platforms {
		if strings.HasPrefix(toComplete, "https://"+name) || 
		   strings.HasPrefix(toComplete, platform.HTTPSPrefix) ||
		   strings.Contains(toComplete, name+".com") {
			
			// If input already has the platform prefix, suggest completion
			if strings.HasPrefix(toComplete, platform.HTTPSPrefix) {
				suggestions = append(suggestions, URLSuggestion{
					URL:         toComplete,
					Description: platform.Description,
				})
			} else if strings.HasPrefix(toComplete, "https://") {
				// Complete the platform URL
				suggestions = append(suggestions, URLSuggestion{
					URL:         platform.HTTPSPrefix,
					Description: platform.Description,
				})
			}
		}
	}

	return suggestions
}

// GetURLSuggestionsWithDescriptions returns URL suggestions with descriptions for enhanced user experience.
func GetURLSuggestionsWithDescriptions(toComplete string) []URLSuggestion {
	return getPlatformURLSuggestions(toComplete)
}

// looksLikeURL determines if the input looks like a URL.
func looksLikeURL(input string) bool {
	return strings.HasPrefix(input, "https://") ||
		strings.HasPrefix(input, "http://") ||
		strings.HasPrefix(input, "git@") ||
		strings.HasPrefix(input, "ssh://") ||
		utils.IsGitURL(input)
}

// CompleteGitURL provides intelligent completion for Git URLs.
func CompleteGitURL(toComplete string) []string {
	log := logger.WithComponent("git_url_completion")

	// If it's already a valid Git URL, return it as-is
	if utils.IsGitURL(toComplete) {
		log.Debug("input is already valid git URL", "url", toComplete)
		return []string{toComplete}
	}

	// Try to parse as platform URL and suggest completions
	if info, err := utils.ParseGitPlatformURL(toComplete); err == nil {
		log.Debug("parsed platform URL", "platform", info.Platform, "repo", info.RepoURL)
		return []string{info.RepoURL}
	}

	// Provide platform-specific suggestions
	suggestions, _ := getURLSuggestions(toComplete)
	log.Debug("providing URL suggestions", "count", len(suggestions))
	return suggestions
}

// CompleteGitURLWithDescriptions provides intelligent completion for Git URLs with descriptions.
func CompleteGitURLWithDescriptions(toComplete string) []URLSuggestion {
	log := logger.WithComponent("git_url_completion")

	// If it's already a valid Git URL, return it as-is
	if utils.IsGitURL(toComplete) {
		log.Debug("input is already valid git URL", "url", toComplete)
		return []URLSuggestion{{URL: toComplete, Description: "Valid Git URL"}}
	}

	// Try to parse as platform URL and suggest completions
	if info, err := utils.ParseGitPlatformURL(toComplete); err == nil {
		log.Debug("parsed platform URL", "platform", info.Platform, "repo", info.RepoURL)
		return []URLSuggestion{{URL: info.RepoURL, Description: "Parsed " + info.Platform + " repository URL"}}
	}

	// Provide platform-specific suggestions with descriptions
	suggestions := GetURLSuggestionsWithDescriptions(toComplete)
	log.Debug("providing URL suggestions with descriptions", "count", len(suggestions))
	return suggestions
}

// ValidateURLCompletion validates that a URL completion is appropriate.
func ValidateURLCompletion(url string) bool {
	// Basic validation - check if it looks like a valid repository URL
	return utils.IsGitURL(url) || looksLikeURL(url)
}

// GetPlatformFromURL extracts the platform name from a URL.
func GetPlatformFromURL(url string) string {
	if info, err := utils.ParseGitPlatformURL(url); err == nil {
		return info.Platform
	}
	return "unknown"
}

// SuggestBranchesForURL suggests branch names based on URL context.
func SuggestBranchesForURL(url string) []string {
	// Extract branch information from URL if available
	if info, err := utils.ParseGitPlatformURL(url); err == nil && info.BranchName != "" {
		return []string{info.BranchName}
	}

	// Default branch suggestions
	return []string{"main", "master", "develop", "development"}
}
