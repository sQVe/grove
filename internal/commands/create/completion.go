package create

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/sqve/grove/internal/completion"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
)

const (
	PatternEnv            = ".env"
	PatternEnvAll         = ".env*"
	PatternVSCode         = ".vscode/"
	PatternIDEA           = ".idea/"
	PatternLocalFiles     = "*.local.*"
	PatternLocalGitignore = ".gitignore.local"
	PatternDockerOverride = "docker-compose.override.yml"
)

var commonCopyPatterns = []string{
	PatternEnv,
	PatternEnvAll,
	PatternVSCode,
	PatternIDEA,
	PatternLocalFiles,
	PatternLocalGitignore,
	PatternDockerOverride,
}

func RegisterCreateCompletion(cmd *cobra.Command) {
	log := logger.WithComponent("create_completion")

	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			branches, directive := completion.BranchCompletion(completion.NewCompletionContext(git.DefaultExecutor), cmd, args, toComplete)

			if strings.Contains(toComplete, "/") && len(branches) == 0 {
				return []string{toComplete}, cobra.ShellCompDirectiveNoSpace
			}

			return branches, directive
		} else if len(args) == 1 {
			return nil, cobra.ShellCompDirectiveFilterDirs
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	if err := cmd.RegisterFlagCompletionFunc("base", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return completion.BranchCompletion(completion.NewCompletionContext(git.DefaultExecutor), cmd, args, toComplete)
	}); err != nil {
		log.Warn("Failed to register base flag completion", "error", err)
	}

	if err := cmd.RegisterFlagCompletionFunc("copy", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if strings.Contains(toComplete, ",") {
			parts := strings.Split(toComplete, ",")
			lastPart := strings.TrimSpace(parts[len(parts)-1])
			prefix := strings.Join(parts[:len(parts)-1], ",") + ","

			var suggestions []string
			for _, pattern := range commonCopyPatterns {
				if strings.HasPrefix(pattern, lastPart) {
					suggestions = append(suggestions, prefix+pattern)
				}
			}
			return suggestions, cobra.ShellCompDirectiveNoFileComp
		}

		var suggestions []string
		for _, pattern := range commonCopyPatterns {
			if strings.HasPrefix(pattern, toComplete) {
				suggestions = append(suggestions, pattern)
			}
		}
		return suggestions, cobra.ShellCompDirectiveNoFileComp
	}); err != nil {
		log.Warn("Failed to register copy flag completion", "error", err)
	}
}
