package completion

import (
	"strings"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/utils"
)

type URLSuggestion struct {
	URL         string
	Description string
}

type PlatformInfo struct {
	HTTPSPrefix string
	SSHPrefix   string
	Description string
}

func URLCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("url_completion")

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

func URLAndDirectoryCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("url_directory_completion")

	if looksLikeURL(toComplete) {
		return URLCompletion(ctx, cmd, args, toComplete)
	}

	log.Debug("providing directory completion", "input", toComplete)
	return nil, cobra.ShellCompDirectiveDefault
}

func getURLSuggestions(toComplete string) ([]string, error) {
	suggestions := make([]string, 0)

	platformSuggestions := getPlatformURLSuggestions(toComplete)

	for _, suggestion := range platformSuggestions {
		suggestions = append(suggestions, suggestion.URL)
	}

	return suggestions, nil
}

func getPlatformURLSuggestions(toComplete string) []URLSuggestion {
	suggestions := make([]URLSuggestion, 0)

	// Avoid suggesting URLs for plain text that doesn't contain domain indicators.
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

	for name, platform := range platforms {
		if strings.HasPrefix(toComplete, "https://"+name) ||
			strings.HasPrefix(toComplete, platform.HTTPSPrefix) ||
			strings.Contains(toComplete, name+".com") {

			if strings.HasPrefix(toComplete, platform.HTTPSPrefix) {
				suggestions = append(suggestions, URLSuggestion{
					URL:         toComplete,
					Description: platform.Description,
				})
			} else if strings.HasPrefix(toComplete, "https://") {
				suggestions = append(suggestions, URLSuggestion{
					URL:         platform.HTTPSPrefix,
					Description: platform.Description,
				})
			}
		}
	}

	return suggestions
}

func GetURLSuggestionsWithDescriptions(toComplete string) []URLSuggestion {
	return getPlatformURLSuggestions(toComplete)
}

func looksLikeURL(input string) bool {
	return strings.HasPrefix(input, "https://") ||
		strings.HasPrefix(input, "http://") ||
		strings.HasPrefix(input, "git@") ||
		strings.HasPrefix(input, "ssh://") ||
		utils.IsGitURL(input)
}

func CompleteGitURL(toComplete string) []string {
	log := logger.WithComponent("git_url_completion")

	if utils.IsGitURL(toComplete) {
		log.Debug("input is already valid git URL", "url", toComplete)
		return []string{toComplete}
	}

	if info, err := utils.ParseGitPlatformURL(toComplete); err == nil {
		log.Debug("parsed platform URL", "platform", info.Platform, "repo", info.RepoURL)
		return []string{info.RepoURL}
	}

	suggestions, _ := getURLSuggestions(toComplete)
	log.Debug("providing URL suggestions", "count", len(suggestions))
	return suggestions
}

func CompleteGitURLWithDescriptions(toComplete string) []URLSuggestion {
	log := logger.WithComponent("git_url_completion")

	if utils.IsGitURL(toComplete) {
		log.Debug("input is already valid git URL", "url", toComplete)
		return []URLSuggestion{{URL: toComplete, Description: "Valid Git URL"}}
	}

	if info, err := utils.ParseGitPlatformURL(toComplete); err == nil {
		log.Debug("parsed platform URL", "platform", info.Platform, "repo", info.RepoURL)
		return []URLSuggestion{{URL: info.RepoURL, Description: "Parsed " + info.Platform + " repository URL"}}
	}

	suggestions := GetURLSuggestionsWithDescriptions(toComplete)
	log.Debug("providing URL suggestions with descriptions", "count", len(suggestions))
	return suggestions
}

func ValidateURLCompletion(url string) bool {
	return utils.IsGitURL(url) || looksLikeURL(url)
}

func GetPlatformFromURL(url string) string {
	if info, err := utils.ParseGitPlatformURL(url); err == nil {
		return info.Platform
	}
	return "unknown"
}

func SuggestBranchesForURL(url string) []string {
	if info, err := utils.ParseGitPlatformURL(url); err == nil && info.BranchName != "" {
		return []string{info.BranchName}
	}

	return []string{"main", "master", "develop", "development"}
}
