package completion

import (
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/sqve/grove/internal/git"
	"github.com/sqve/grove/internal/logger"
	"github.com/sqve/grove/internal/utils"
)

// CompletionTimeout is the maximum time to wait for completion operations.
const CompletionTimeout = 2 * time.Second

// CompletionContext provides context for completion operations.
type CompletionContext struct {
	Executor git.GitExecutor
	Timeout  time.Duration
}

func NewCompletionContext(executor git.GitExecutor) *CompletionContext {
	return &CompletionContext{
		Executor: executor,
		Timeout:  CompletionTimeout,
	}
}

func (c *CompletionContext) WithTimeout(fn func() ([]string, error)) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), c.Timeout)
	defer cancel()

	resultChan := make(chan []string, 1)
	errorChan := make(chan error, 1)

	go func() {
		result, err := fn()
		if err != nil {
			errorChan <- err
		} else {
			resultChan <- result
		}
	}()

	select {
	case result := <-resultChan:
		return result, nil
	case err := <-errorChan:
		return nil, err
	case <-ctx.Done():
		return nil, fmt.Errorf("completion operation timed out")
	}
}

func (c *CompletionContext) IsInGroveRepo() bool {
	log := logger.WithComponent("completion")

	if isGroveRepo, exists := GetCachedRepositoryState(); exists {
		log.Debug("using cached repository state", "is_grove_repo", isGroveRepo)
		return isGroveRepo
	}

	isRepo, err := utils.IsGitRepository(c.Executor)
	if err != nil || !isRepo {
		log.Debug("not in git repository for completion", "error", err)
		SetCachedRepositoryState(false)
		return false
	}

	// Any git repository is considered valid for completion.
	SetCachedRepositoryState(true)
	return true
}

func (c *CompletionContext) IsOnline() bool {
	log := logger.WithComponent("network_check")

	if isOnline, exists := GetCachedNetworkState(); exists {
		log.Debug("using cached network state", "is_online", isOnline)
		return isOnline
	}

	// Quick DNS check to determine network connectivity without blocking.
	conn, err := net.DialTimeout("tcp", "8.8.8.8:53", 500*time.Millisecond)
	if err != nil {
		log.Debug("network connectivity check failed", "error", err)
		SetCachedNetworkState(false)
		return false
	}
	defer func() { _ = conn.Close() }()

	log.Debug("network connectivity confirmed")
	SetCachedNetworkState(true)
	return true
}

func (c *CompletionContext) IsNetworkOperationAllowed() bool {
	// Conservative network operations prevent shell blocking during completion.
	return c.IsOnline()
}

func FilterCompletions(completions []string, toComplete string) []string {
	if toComplete == "" {
		return completions
	}

	var filtered []string
	for _, completion := range completions {
		if strings.HasPrefix(completion, toComplete) {
			filtered = append(filtered, completion)
		}
	}

	return filtered
}

func CreateCompletionCommands(rootCmd *cobra.Command) {
	completionCmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate completion script",
		Long: `Generate completion script for Grove CLI.

To enable completion, run the appropriate command for your shell:

Bash:
  grove completion bash > /etc/bash_completion.d/grove
  # or
  grove completion bash > ~/.bash_completion.d/grove

Zsh:
  grove completion zsh > "${fpath[1]}/_grove"
  # or add to ~/.zshrc:
  echo 'autoload -U compinit; compinit' >> ~/.zshrc

Fish:
  grove completion fish > ~/.config/fish/completions/grove.fish

PowerShell:
  grove completion powershell > grove.ps1
  # then source it in your profile`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := args[0]
			switch shell {
			case "bash":
				return rootCmd.GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return rootCmd.GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return rootCmd.GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return rootCmd.GenPowerShellCompletion(cmd.OutOrStdout())
			default:
				return fmt.Errorf("unsupported shell: %s", shell)
			}
		},
	}

	rootCmd.AddCommand(completionCmd)
}

func RegisterCompletionFunctions(rootCmd *cobra.Command, executor git.GitExecutor) {
	ctx := NewCompletionContext(executor)

	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "init" {
			registerInitCompletions(cmd, ctx)
		}
	}
}

func registerInitCompletions(cmd *cobra.Command, ctx *CompletionContext) {
	_ = cmd.RegisterFlagCompletionFunc("branches", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return BranchListCompletion(ctx, cmd, args, toComplete)
	})

	// Register completion for positional arguments (URLs and directories).
	cmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) == 0 {
			return URLAndDirectoryCompletion(ctx, cmd, args, toComplete)
		}
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

func BranchListCompletion(ctx *CompletionContext, cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	log := logger.WithComponent("branch_list_completion")

	if !ctx.IsInGroveRepo() {
		log.Debug("not in grove repository, skipping branch list completion")
		return nil, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
	}

	var currentInput, lastBranch string

	if lastCommaIndex := strings.LastIndex(toComplete, ","); lastCommaIndex != -1 {
		currentInput = toComplete[:lastCommaIndex+1]
		lastBranch = strings.TrimSpace(toComplete[lastCommaIndex+1:])
	} else {
		currentInput = ""
		lastBranch = toComplete
	}

	completions, err := ctx.WithTimeout(func() ([]string, error) {
		return CompleteBranchList(ctx, toComplete, lastBranch)
	})
	if err != nil {
		log.Debug("failed to get branch list completions", "error", err)
		return nil, cobra.ShellCompDirectiveError
	}

	// Preserve comma-separated structure by prepending existing input.
	var result []string
	for _, completion := range completions {
		result = append(result, currentInput+completion)
	}

	log.Debug("branch list completion results", "total", len(result), "input", toComplete, "current_input", currentInput, "last_branch", lastBranch)
	return result, cobra.ShellCompDirectiveNoSpace | cobra.ShellCompDirectiveNoFileComp
}

func SafeExecuteWithFallback(fn func() ([]string, cobra.ShellCompDirective), fallback []string) (result []string, directive cobra.ShellCompDirective) {
	defer func() {
		if r := recover(); r != nil {
			logger.WithComponent("completion").Debug("completion function panicked", "error", r)
			result = fallback
			directive = cobra.ShellCompDirectiveError
		}
	}()

	result, directive = fn()

	if len(result) == 0 {
		return fallback, directive
	}
	return result, directive
}
